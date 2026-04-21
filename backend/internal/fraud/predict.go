package fraud

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ── Sprint 24: Predictive Risk Layer ────────────────────────────────────────
//
// A fast (<10 ms) pre-action risk score built on top of data the platform
// already collects. Unlike the heavier Engine.Score() path, PredictRisk takes
// ONLY a user_id and returns a 0–100 score + the factor breakdown.
//
// Decision thresholds (per spec):
//   0–29  → allow
//   30–59 → soft friction
//   60–79 → limit actions
//   80+   → block

// PredictDecision is the categorical action tied to a predicted score.
type PredictDecision string

const (
	PredictAllow PredictDecision = "allow"
	PredictSoft  PredictDecision = "soft"
	PredictLimit PredictDecision = "limit"
	PredictBlock PredictDecision = "block"
)

// RiskFactors is the transparent breakdown of a predicted risk score.
type RiskFactors struct {
	Velocity        int `json:"velocity"`
	PaymentRisk     int `json:"payment_risk"`
	DisputePenalty  int `json:"dispute_penalty"`
	TrustInverse    int `json:"trust_inverse"`
	ReferralPenalty int `json:"referral_penalty"`
	Total           int `json:"total"`
}

// PredictResult is the output of PredictRisk.
type PredictResult struct {
	UserID     uuid.UUID       `json:"user_id"`
	Score      int             `json:"score"`
	Decision   PredictDecision `json:"decision"`
	Factors    RiskFactors     `json:"factors"`
	Cached     bool            `json:"cached"`
	DurationMs int64           `json:"duration_ms"`
	CreatedAt  time.Time       `json:"created_at"`
}

// Predictor is the pre-action risk predictor.
type Predictor struct {
	db  *gorm.DB
	rdb *redis.Client
}

// NewPredictor creates a new predictor. rdb may be nil (caching disabled).
func NewPredictor(db *gorm.DB, rdb *redis.Client) *Predictor {
	return &Predictor{db: db, rdb: rdb}
}

// PredictRisk computes a 0–100 risk score for the given user.
// Results are cached in Redis for 60 seconds to keep the hot path fast.
func (p *Predictor) PredictRisk(ctx context.Context, userID uuid.UUID) PredictResult {
	start := time.Now()

	if cached, ok := p.readCache(ctx, userID); ok {
		cached.Cached = true
		cached.DurationMs = time.Since(start).Milliseconds()
		return cached
	}

	f := RiskFactors{}
	f.Velocity = p.velocityScore(userID)
	f.PaymentRisk = p.paymentRiskScore(userID)
	f.DisputePenalty = p.disputePenalty(userID)
	f.TrustInverse = p.trustInverse(userID)
	f.ReferralPenalty = p.referralPenalty(userID)
	f.Total = clamp(f.Velocity+f.PaymentRisk+f.DisputePenalty+f.TrustInverse+f.ReferralPenalty, 0, 100)

	res := PredictResult{
		UserID:    userID,
		Score:     f.Total,
		Decision:  DecideFromScore(f.Total),
		Factors:   f,
		CreatedAt: time.Now().UTC(),
	}
	res.DurationMs = time.Since(start).Milliseconds()

	p.writeCache(ctx, userID, res)
	go p.saveSnapshot(res) // fire-and-forget audit trail

	return res
}

// DecideFromScore maps an integer score to a PredictDecision.
func DecideFromScore(score int) PredictDecision {
	switch {
	case score >= 80:
		return PredictBlock
	case score >= 60:
		return PredictLimit
	case score >= 30:
		return PredictSoft
	default:
		return PredictAllow
	}
}

// ── Individual factors ──────────────────────────────────────────────────────

// velocityScore: rapid actions in the last hour.
// Counts exchange requests + auction bids + withdraw requests.
// Caps at 25 points once the user hits ~40+ actions/hour.
func (p *Predictor) velocityScore(userID uuid.UUID) int {
	var c1, c2, c3 int64
	p.db.Raw(`SELECT COUNT(*) FROM exchange_requests WHERE requester_id = ? AND created_at > NOW() - INTERVAL '1 hour'`, userID).Scan(&c1)
	p.db.Raw(`SELECT COUNT(*) FROM live_auction_bids WHERE bidder_id = ? AND created_at > NOW() - INTERVAL '1 hour'`, userID).Scan(&c2)
	p.db.Raw(`SELECT COUNT(*) FROM withdraw_requests WHERE user_id = ? AND created_at > NOW() - INTERVAL '1 hour'`, userID).Scan(&c3)
	total := c1 + c2 + c3
	switch {
	case total >= 40:
		return 25
	case total >= 20:
		return 15
	case total >= 10:
		return 8
	case total >= 5:
		return 3
	default:
		return 0
	}
}

// paymentRiskScore: derived from the user's default / last-used payment method.
// Crypto / unverified methods score higher, verified bank / agent lower.
func (p *Predictor) paymentRiskScore(userID uuid.UUID) int {
	var methodType string
	p.db.Raw(`
		SELECT COALESCE(payment_method_type, '')
		FROM deposit_requests
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(&methodType)

	switch methodType {
	case "crypto", "p2p":
		return 15
	case "card":
		return 8
	case "bank_transfer":
		return 4
	case "agent", "wallet":
		return 2
	default:
		return 5 // unknown = slight penalty
	}
}

// disputePenalty: count of rejected / disputed settlements in the last 30 days.
func (p *Predictor) disputePenalty(userID uuid.UUID) int {
	var disputes int64
	p.db.Raw(`
		SELECT COUNT(*)
		FROM exchange_settlements s
		JOIN exchange_matches m ON m.id = s.match_id
		JOIN exchange_requests r ON r.id IN (m.buy_request_id, m.sell_request_id)
		WHERE r.requester_id = ?
		  AND s.proof_status = 'rejected'
		  AND s.updated_at > NOW() - INTERVAL '30 days'
	`, userID).Scan(&disputes)

	switch {
	case disputes >= 5:
		return 30
	case disputes >= 2:
		return 15
	case disputes >= 1:
		return 5
	default:
		return 0
	}
}

// trustInverse: derived from the Sprint 23 security profile — direct indicator
// of how "bad" the user has behaved recently. 0-100 score on that table
// becomes a 0-30 penalty here (scaled down so one factor cannot dominate).
func (p *Predictor) trustInverse(userID uuid.UUID) int {
	var secScore int
	var frozen bool
	p.db.Raw(`
		SELECT COALESCE(risk_score, 0), COALESCE(frozen, false)
		FROM user_security_profiles
		WHERE user_id = ?
	`, userID).Row().Scan(&secScore, &frozen)

	if frozen {
		return 30
	}
	// 0-100 on security profile → 0-30 on prediction.
	return clamp(secScore*30/100, 0, 30)
}

// referralPenalty: if the user came through a low-quality or unverified invite,
// or through an inviter who later got flagged, add risk.
func (p *Predictor) referralPenalty(userID uuid.UUID) int {
	// Did the user come through an invite?
	var inviterID uuid.UUID
	p.db.Raw(`
		SELECT COALESCE(i.inviter_id, '00000000-0000-0000-0000-000000000000')
		FROM invite_usages u
		JOIN invites i ON i.id = u.invite_id
		WHERE u.invitee_id = ?
		LIMIT 1
	`, userID).Scan(&inviterID)

	if inviterID == uuid.Nil {
		// No invite at all — small penalty for organic + less trusted signup paths.
		return 2
	}

	// Is the inviter themselves flagged / frozen?
	var inviterSec int
	var inviterFrozen bool
	p.db.Raw(`
		SELECT COALESCE(risk_score, 0), COALESCE(frozen, false)
		FROM user_security_profiles
		WHERE user_id = ?
	`, inviterID).Row().Scan(&inviterSec, &inviterFrozen)

	if inviterFrozen || inviterSec >= 70 {
		return 15
	}
	if inviterSec >= 40 {
		return 8
	}
	return 0
}

// ── Snapshot persistence + cache ────────────────────────────────────────────

const predictCachePrefix = "fraud:predict:"

func (p *Predictor) readCache(ctx context.Context, userID uuid.UUID) (PredictResult, bool) {
	if p.rdb == nil {
		return PredictResult{}, false
	}
	raw, err := p.rdb.Get(ctx, predictCachePrefix+userID.String()).Bytes()
	if err != nil || len(raw) == 0 {
		return PredictResult{}, false
	}
	var r PredictResult
	if err := json.Unmarshal(raw, &r); err != nil {
		return PredictResult{}, false
	}
	return r, true
}

func (p *Predictor) writeCache(ctx context.Context, userID uuid.UUID, r PredictResult) {
	if p.rdb == nil {
		return
	}
	buf, _ := json.Marshal(r)
	p.rdb.Set(ctx, predictCachePrefix+userID.String(), buf, 60*time.Second)
}

func (p *Predictor) saveSnapshot(r PredictResult) {
	if p.db == nil {
		return
	}
	snap := UserRiskSnapshot{
		UserID:    r.UserID,
		Score:     r.Score,
		Decision:  string(r.Decision),
		Factors:   snapshotFactors(r.Factors),
		CreatedAt: r.CreatedAt,
	}
	p.db.Create(&snap)
}

func snapshotFactors(f RiskFactors) map[string]any {
	return map[string]any{
		"velocity":         f.Velocity,
		"payment_risk":     f.PaymentRisk,
		"dispute_penalty":  f.DisputePenalty,
		"trust_inverse":    f.TrustInverse,
		"referral_penalty": f.ReferralPenalty,
		"total":            f.Total,
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
