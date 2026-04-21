package redteam

import (
	"context"
	"fmt"
	"time"

	"github.com/geocore-next/backend/internal/exchange"
	"github.com/geocore-next/backend/internal/fraud"
	"github.com/geocore-next/backend/internal/security"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Simulator is the in-process red-team engine. It exercises real defensive
// code paths (IDS, fraud.Predictor, exchange risk detectors) WITHOUT mutating
// user-visible data — synthetic IPs and ephemeral user IDs are used.
type Simulator struct {
	db   *gorm.DB
	rdb  *redis.Client
	ids  *security.IDS
	pred *fraud.Predictor
}

func NewSimulator(db *gorm.DB, rdb *redis.Client, ids *security.IDS) *Simulator {
	return &Simulator{
		db:   db,
		rdb:  rdb,
		ids:  ids,
		pred: fraud.NewPredictor(db, rdb),
	}
}

// ─── Scenario 1: Spam Attack ─────────────────────────────────────────────────
// Fires 200 synthetic "requests" from a single IP against the IDS.
// Success criteria: IDS auto-blocks before all attempts finish.
func (s *Simulator) SimulateSpamAttack(ctx context.Context) ScenarioResult {
	res := ScenarioResult{
		Scenario:     "spam",
		StartedAt:    time.Now().UTC(),
		FirstBlockAt: -1,
		Notes:        []string{},
	}
	start := time.Now()

	simIP := simIP("spam")
	const attempts = 200

	// Always reset the simulated IP so repeated runs start clean.
	if s.rdb != nil {
		s.rdb.Del(ctx, "ids:blocked:"+simIP)
	}

	for i := 1; i <= attempts; i++ {
		res.Attempts++
		if s.ids != nil {
			// Track first — this bumps the minute counter; at threshold IDS auto-blocks.
			s.ids.TrackRequest(ctx, simIP)
			if s.ids.IsBlocked(ctx, simIP) {
				res.Blocked++
				if res.FirstBlockAt == -1 {
					res.FirstBlockAt = i
					res.Notes = append(res.Notes,
						fmt.Sprintf("IDS auto-blocked simIP at attempt #%d", i))
					res.TriggeredAlerts = append(res.TriggeredAlerts, "ids.auto_block")
				}
			}
		}
	}

	res.DurationMs = time.Since(start).Milliseconds()
	res.SystemResponded = s.ids != nil
	res.DefenseTriggered = res.Blocked > 0
	res.Passed = res.DefenseTriggered && res.FirstBlockAt > 0 && res.FirstBlockAt <= 150
	res.Metrics = map[string]any{
		"sim_ip":                  simIP,
		"expected_block_by":       security.ReqSpikeLimit,
		"requests_before_block":   res.FirstBlockAt,
	}

	// Housekeeping: unblock the simulated IP so it doesn't linger.
	if s.rdb != nil {
		s.rdb.Del(ctx, "ids:blocked:"+simIP)
	}
	return res
}

// ─── Scenario 2: Referral Abuse ──────────────────────────────────────────────
// Plants a self-referral / low-quality-referrer row and calls PredictRisk.
// Success criteria: referral_penalty > 0 AND overall score non-zero.
func (s *Simulator) SimulateReferralAbuse(ctx context.Context) ScenarioResult {
	res := ScenarioResult{
		Scenario:     "referral",
		StartedAt:    time.Now().UTC(),
		FirstBlockAt: -1,
		Notes:        []string{},
	}
	start := time.Now()

	// Use a deterministic UUID so successive runs work against the same row
	// AND so it's trivially identifiable as synthetic.
	synthUser := simUserID("referral-victim")
	synthInviter := simUserID("referral-bad-inviter")

	// Plant a synthetic security profile for the inviter with a high score
	// — simulates an inviter who's already been flagged.
	s.db.Exec(`
		INSERT INTO user_security_profiles (user_id, risk_score, frozen, flags_count, updated_at, last_event_at)
		VALUES (?, 85, true, 3, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET risk_score = 85, frozen = true
	`, synthInviter)

	// Run prediction on the victim who was "invited" by the bad actor.
	// (we plant the invite row too so referral_penalty kicks in)
	var inviteID uuid.UUID
	s.db.Raw(`
		INSERT INTO invites (id, inviter_id, code, created_at)
		VALUES (uuid_generate_v4(), ?, ?, NOW())
		ON CONFLICT DO NOTHING
		RETURNING id
	`, synthInviter, "REDTEAM-"+time.Now().Format("150405")).Scan(&inviteID)
	if inviteID != uuid.Nil {
		s.db.Exec(`
			INSERT INTO invite_usages (id, invite_id, invitee_id, referral_status, created_at)
			VALUES (uuid_generate_v4(), ?, ?, 'pending', NOW())
			ON CONFLICT DO NOTHING
		`, inviteID, synthUser)
	}

	attempts := 3
	defenseHit := false
	var lastScore int
	for i := 1; i <= attempts; i++ {
		res.Attempts++
		pred := s.pred.PredictRisk(ctx, synthUser)
		lastScore = pred.Score
		res.Notes = append(res.Notes,
			fmt.Sprintf("attempt %d → score=%d decision=%s referral_penalty=%d",
				i, pred.Score, pred.Decision, pred.Factors.ReferralPenalty))
		if pred.Factors.ReferralPenalty > 0 {
			defenseHit = true
			if res.FirstBlockAt == -1 {
				res.FirstBlockAt = i
				res.TriggeredAlerts = append(res.TriggeredAlerts, "fraud.referral_penalty")
			}
			res.Blocked++
		}
	}

	// Clean up synthetic rows (best-effort).
	s.db.Exec(`DELETE FROM invite_usages WHERE invitee_id = ?`, synthUser)
	s.db.Exec(`DELETE FROM invites WHERE inviter_id = ?`, synthInviter)
	s.db.Exec(`DELETE FROM user_security_profiles WHERE user_id = ?`, synthInviter)
	s.db.Exec(`DELETE FROM user_risk_snapshots WHERE user_id = ?`, synthUser)

	res.DurationMs = time.Since(start).Milliseconds()
	res.SystemResponded = true
	res.DefenseTriggered = defenseHit
	res.Passed = defenseHit
	res.Metrics = map[string]any{
		"last_predicted_score": lastScore,
		"synth_user":           synthUser,
		"synth_inviter":        synthInviter,
	}
	return res
}

// ─── Scenario 3: Circular Trade ──────────────────────────────────────────────
// Plants two matched-reversed exchange requests A→B then B→A within 24h,
// then invokes the exchange circular-trade detector via CheckMatchRisk.
// Success: RiskCheckResult.Allowed == false OR RiskLevel elevated.
func (s *Simulator) SimulateCircularTrade(ctx context.Context) ScenarioResult {
	res := ScenarioResult{
		Scenario:     "exchange",
		StartedAt:    time.Now().UTC(),
		FirstBlockAt: -1,
		Notes:        []string{},
	}
	start := time.Now()

	userA := simUserID("circular-A")
	userB := simUserID("circular-B")

	// Forward pair request stubs — just enough for the detector to inspect.
	reqA := &exchange.ExchangeRequest{
		ID:     uuid.New(),
		UserID: userA,
	}
	reqB := &exchange.ExchangeRequest{
		ID:     uuid.New(),
		UserID: userB,
	}

	// Plant a historical A→B→A match cycle directly (detector reads this).
	// Uses raw SQL so we don't depend on full Request/Match relation builders.
	now := time.Now().UTC()
	reqOld1 := uuid.New()
	reqOld2 := uuid.New()
	matchOld := uuid.New()
	s.db.Exec(`
		INSERT INTO exchange_requests (id, user_id, status, created_at, updated_at)
		VALUES (?, ?, 'matched', ?, ?), (?, ?, 'matched', ?, ?)
		ON CONFLICT DO NOTHING
	`, reqOld1, userA, now.Add(-2*time.Hour), now, reqOld2, userB, now.Add(-2*time.Hour), now)
	s.db.Exec(`
		INSERT INTO exchange_matches (id, request_a_id, request_b_id, status, created_at, updated_at)
		VALUES (?, ?, ?, 'completed', ?, ?)
		ON CONFLICT DO NOTHING
	`, matchOld, reqOld1, reqOld2, now.Add(-2*time.Hour), now)

	// Now attempt the reverse direction — this is the red-team payload.
	const attempts = 3
	defenseHit := false
	for i := 1; i <= attempts; i++ {
		res.Attempts++
		check := exchange.CheckMatchRisk(s.db, reqA, reqB)
		res.Notes = append(res.Notes,
			fmt.Sprintf("attempt %d → allowed=%v level=%s flags=%v",
				i, check.Allowed, check.RiskLevel, check.Flags))
		if !check.Allowed || check.RiskLevel == exchange.RiskHigh {
			defenseHit = true
			if res.FirstBlockAt == -1 {
				res.FirstBlockAt = i
				res.TriggeredAlerts = append(res.TriggeredAlerts, "exchange.circular_trade")
			}
			res.Blocked++
		}
	}

	// Clean up.
	s.db.Exec(`DELETE FROM exchange_matches WHERE id = ?`, matchOld)
	s.db.Exec(`DELETE FROM exchange_requests WHERE id IN (?, ?)`, reqOld1, reqOld2)
	s.db.Exec(`DELETE FROM exchange_risk_flags WHERE user_id IN (?, ?)`, userA, userB)

	res.DurationMs = time.Since(start).Milliseconds()
	res.SystemResponded = true
	res.DefenseTriggered = defenseHit
	res.Passed = defenseHit
	return res
}

// ─── Scenario 4: Bid Flood ───────────────────────────────────────────────────
// Plants synthetic "rapid actions" for a test user then calls PredictRisk
// repeatedly until the decision transitions allow → limit/block.
// Success: fraud.Predictor decision progresses past ALLOW within N attempts.
func (s *Simulator) SimulateBidFlood(ctx context.Context) ScenarioResult {
	res := ScenarioResult{
		Scenario:     "bids",
		StartedAt:    time.Now().UTC(),
		FirstBlockAt: -1,
		Notes:        []string{},
	}
	start := time.Now()

	synthUser := simUserID("bid-flood")

	// Pump synthetic bid rows so velocity factor maxes out.
	// ~60 rows is comfortably above the 40-cap threshold (25 pts).
	for i := 0; i < 60; i++ {
		s.db.Exec(`
			INSERT INTO live_auction_bids (id, bidder_id, session_id, item_id, amount_cents, created_at)
			VALUES (uuid_generate_v4(), ?, uuid_generate_v4(), uuid_generate_v4(), 100, NOW())
			ON CONFLICT DO NOTHING
		`, synthUser)
	}

	// Also emit a fraud_flag security event so trust_inverse rises.
	uid := synthUser
	security.LogSecurityEvent(s.db, &uid, "127.0.0.1", "redteam", "fraud_flag", security.SevSecCritical,
		map[string]any{"source": "redteam.bid_flood"})

	// Small wait so async inserts land, then re-run predictor.
	time.Sleep(100 * time.Millisecond)

	// Invalidate predictor cache for this user so we see fresh scores.
	if s.rdb != nil {
		s.rdb.Del(ctx, "fraud:predict:"+synthUser.String())
	}

	const attempts = 5
	var decisions []string
	var lastScore int
	defenseHit := false
	for i := 1; i <= attempts; i++ {
		res.Attempts++
		pred := s.pred.PredictRisk(ctx, synthUser)
		decisions = append(decisions, string(pred.Decision))
		lastScore = pred.Score
		res.Notes = append(res.Notes,
			fmt.Sprintf("attempt %d → score=%d decision=%s velocity=%d",
				i, pred.Score, pred.Decision, pred.Factors.Velocity))
		if pred.Decision != fraud.PredictAllow {
			defenseHit = true
			if res.FirstBlockAt == -1 {
				res.FirstBlockAt = i
				res.TriggeredAlerts = append(res.TriggeredAlerts, "fraud."+string(pred.Decision))
			}
			res.Blocked++
		}
	}

	// Clean up.
	s.db.Exec(`DELETE FROM live_auction_bids WHERE bidder_id = ?`, synthUser)
	s.db.Exec(`DELETE FROM user_security_profiles WHERE user_id = ?`, synthUser)
	s.db.Exec(`DELETE FROM user_risk_snapshots WHERE user_id = ?`, synthUser)
	s.db.Exec(`DELETE FROM security_audit_log WHERE user_id = ?`, synthUser)
	if s.rdb != nil {
		s.rdb.Del(ctx, "fraud:predict:"+synthUser.String())
	}

	res.DurationMs = time.Since(start).Milliseconds()
	res.SystemResponded = true
	res.DefenseTriggered = defenseHit
	res.Passed = defenseHit
	res.Metrics = map[string]any{
		"decision_trace":       decisions,
		"last_predicted_score": lastScore,
		"synth_user":           synthUser,
	}
	return res
}

// ─── helpers ────────────────────────────────────────────────────────────────

// simIP returns a deterministic synthetic IP per scenario name.
// Using TEST-NET-1 (RFC 5737) ensures zero collision with real traffic.
func simIP(name string) string {
	switch name {
	case "spam":
		return "192.0.2.10"
	case "bids":
		return "192.0.2.11"
	case "referral":
		return "192.0.2.12"
	case "exchange":
		return "192.0.2.13"
	default:
		return "192.0.2.99"
	}
}

// simUserID derives a deterministic UUIDv5 from a seed string so every run
// targets the exact same synthetic row (makes cleanup idempotent).
func simUserID(seed string) uuid.UUID {
	// Namespace chosen as a known-stable UUIDv4 so we're not flipping IDs per boot.
	ns := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	return uuid.NewSHA1(ns, []byte("redteam:"+seed))
}
