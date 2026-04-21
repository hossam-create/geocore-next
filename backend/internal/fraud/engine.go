package fraud

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"time"

	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/geocore-next/backend/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"gorm.io/gorm"
)

// ── Decision Types ──────────────────────────────────────────────────────────

type Decision string

const (
	DecisionAllow     Decision = "ALLOW"
	DecisionChallenge Decision = "CHALLENGE"
	DecisionBlock     Decision = "BLOCK"
)

// ── Scoring Request / Response ──────────────────────────────────────────────

// ScoreRequest is the input to the fraud scoring engine.
type ScoreRequest struct {
	UserID    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	IP        string  `json:"ip"`
	Country   string  `json:"country"`
	DeviceID  string  `json:"device_id"`
	EventType string  `json:"event_type"` // order.create, wallet.withdraw, escrow.release, payment.webhook
	RequestID string  `json:"request_id"`
	TraceID   string  `json:"trace_id"`
}

// ScoreResponse is the output of the fraud scoring engine.
type ScoreResponse struct {
	Decision   Decision     `json:"decision"`
	RiskScore  float64      `json:"risk_score"`
	RiskLevel  string       `json:"risk_level"`
	Signals    []RiskSignal `json:"signals"`
	ReviewURL  string       `json:"review_url,omitempty"`
	ScoringMs  int64        `json:"scoring_ms"`
	FeatureHit bool         `json:"feature_hit"`
}

// ── Engine ───────────────────────────────────────────────────────────────────

// Engine is the real-time fraud scoring engine.
// It uses Redis feature store for velocity data and rule-based scoring (v1).
// ML-ready abstraction: replace scoreWithRules with scoreWithML.
type Engine struct {
	store      *FeatureStore
	thresholds *ThresholdStore
	db         *gorm.DB
}

// NewEngine creates a fraud scoring engine with a Redis feature store.
func NewEngine(store *FeatureStore, opts ...EngineOption) *Engine {
	e := &Engine{store: store}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// EngineOption configures the fraud engine.
type EngineOption func(*Engine)

// WithDB sets the database connection for audit logging.
func WithDB(db *gorm.DB) EngineOption {
	return func(e *Engine) { e.db = db }
}

// WithThresholds sets the dynamic threshold store.
func WithThresholds(ts *ThresholdStore) EngineOption {
	return func(e *Engine) { e.thresholds = ts }
}

// Score evaluates a transaction against fraud rules.
// Target: <20ms overhead per transaction.
func (e *Engine) Score(ctx context.Context, req ScoreRequest) ScoreResponse {
	start := time.Now()
	ctx, span := tracing.StartSpan(ctx, "fraud.Score",
		attribute.String("user_id", req.UserID),
		attribute.String("event_type", req.EventType),
		attribute.Float64("amount", req.Amount),
	)
	defer span.End()

	// Fetch features from Redis (fast path)
	features, err := e.store.Get(ctx, req.UserID)
	featureHit := features != nil
	if err != nil {
		slog.Warn("fraud: feature store miss", "user_id", req.UserID, "error", err)
	}

	// Score using rule engine (v1) — ML-ready abstraction point
	result := e.scoreWithRules(ctx, req, features)

	// Update features in Redis
	e.updateFeatures(ctx, req, features)

	elapsed := time.Since(start).Milliseconds()
	result.ScoringMs = elapsed
	result.FeatureHit = featureHit

	// Record Prometheus metrics
	metrics.IncFraudDecision(string(result.Decision), req.EventType)
	metrics.ObserveFraudScore(result.RiskScore)

	slog.Info("fraud: scored transaction",
		"user_id", req.UserID,
		"event_type", req.EventType,
		"amount", req.Amount,
		"decision", string(result.Decision),
		"risk_score", result.RiskScore,
		"scoring_ms", elapsed,
		"trace_id", req.TraceID,
		"request_id", req.RequestID,
	)

	// Audit log to DB (async — don't block scoring)
	go e.auditLog(req, result)

	return result
}

// ── Rule-Based Scoring (v1) ─────────────────────────────────────────────────
// Replace this method with ML model inference when ready.

func (e *Engine) scoreWithRules(ctx context.Context, req ScoreRequest, f *UserFeatures) ScoreResponse {
	signals := []RiskSignal{}
	total := 0.0

	// ── Dynamic thresholds from Redis ─────────────────────────────────────
	velThreshold := DefaultVelocityThreshold
	wdThreshold := DefaultWithdrawThreshold
	geoLimit := DefaultGeoMismatchLimit
	driftLimit := DefaultWalletDriftLimit
	if e.thresholds != nil {
		velThreshold = e.thresholds.GetVelocityThreshold(ctx)
		wdThreshold = e.thresholds.GetWithdrawThreshold(ctx)
		geoLimit = e.thresholds.GetGeoMismatchLimit(ctx)
		driftLimit = e.thresholds.GetWalletDriftLimit(ctx)
	}

	// ── Velocity checks (from feature store) ──────────────────────────────
	if f != nil {
		// High transaction velocity in 1h
		if float64(f.TxCountLast1h) > velThreshold {
			s := math.Min(1.0, float64(f.TxCountLast1h)/(velThreshold*2))
			total += s * RiskWeights["velocity"] * 100
			signals = append(signals, RiskSignal{"high_velocity_1h", s, "Transaction velocity exceeds threshold"})
		}

		// High transaction velocity in 24h
		if f.TxCountLast24h > int(velThreshold*3) {
			s := math.Min(1.0, float64(f.TxCountLast24h)/(velThreshold*6))
			total += s * RiskWeights["velocity"] * 100
			signals = append(signals, RiskSignal{"high_velocity_24h", s, "High transaction velocity in 24h"})
		}

		// Multiple withdrawals in 24h
		if float64(f.WithdrawCount24h) > wdThreshold {
			s := math.Min(1.0, float64(f.WithdrawCount24h)/(wdThreshold*2))
			total += s * RiskWeights["behavior"] * 100
			signals = append(signals, RiskSignal{"high_withdrawals", s, "Withdrawal count exceeds threshold"})
		}

		// Failed login attempts
		if f.FailedLogins24h > 3 {
			s := math.Min(1.0, float64(f.FailedLogins24h)/6.0)
			total += s * RiskWeights["device"] * 100
			signals = append(signals, RiskSignal{"brute_force_attempt", s, "Multiple failed login attempts"})
		}

		// Geo mismatch
		if float64(f.GeoMismatchCount) > geoLimit {
			s := math.Min(1.0, float64(f.GeoMismatchCount)/(geoLimit*2.5))
			total += s * RiskWeights["location"] * 100
			signals = append(signals, RiskSignal{"geo_hop", s, "Login from multiple countries"})
		}

		// Wallet drift — large net balance change
		if f.WalletDrift24h > driftLimit {
			s := math.Min(1.0, f.WalletDrift24h/(driftLimit*5))
			total += s * RiskWeights["amount"] * 100
			signals = append(signals, RiskSignal{"wallet_drift", s, "Large net wallet balance change in 24h"})
		}

		// Account age + amount combo
		if f.AccountAgeHours < 24 && req.Amount > 500 {
			s := 0.6
			total += s * RiskWeights["behavior"] * 100
			signals = append(signals, RiskSignal{"new_account_high_value", s, "New account with high-value transaction"})
		}
	}

	// ── Static checks (no feature store needed) ───────────────────────────
	// High single amount
	if req.Amount > 5000 {
		s := math.Min(1.0, req.Amount/10000.0)
		total += s * RiskWeights["amount"] * 100
		signals = append(signals, RiskSignal{"high_amount", s, "High-value transaction"})
	}

	// Amount deviation from average
	if f != nil && f.AvgOrderValue > 0 && f.TotalOrders > 3 {
		deviation := req.Amount / f.AvgOrderValue
		if deviation > 5 {
			s := math.Min(1.0, deviation/10.0)
			total += s * RiskWeights["amount"] * 100
			signals = append(signals, RiskSignal{"amount_deviation", s, "Amount significantly higher than average"})
		}
	}

	score := math.Max(0, math.Min(100, total))
	level := riskLevel(score)

	// ── Dynamic decision thresholds ────────────────────────────────────────
	blockThreshold := DefaultBlockThreshold
	challengeThreshold := DefaultChallengeThreshold
	if e.thresholds != nil {
		blockThreshold = e.thresholds.GetBlockThreshold(ctx)
		challengeThreshold = e.thresholds.GetChallengeThreshold(ctx)
	}

	decision := DecisionAllow
	if score >= blockThreshold {
		decision = DecisionBlock
	} else if score >= challengeThreshold {
		decision = DecisionChallenge
	}

	return ScoreResponse{
		Decision:  decision,
		RiskScore: score,
		RiskLevel: level,
		Signals:   signals,
	}
}

// ── Feature Update ──────────────────────────────────────────────────────────

func (e *Engine) updateFeatures(ctx context.Context, req ScoreRequest, f *UserFeatures) {
	if e.store == nil {
		return
	}
	if f == nil {
		f = &UserFeatures{
			UserID:    req.UserID,
			UpdatedAt: time.Now().Format(time.RFC3339),
		}
	}

	// Update geo info
	if req.IP != "" || req.Country != "" {
		_ = e.store.UpdateGeoInfo(ctx, req.UserID, req.IP, req.Country)
	}

	// Update transaction velocity
	_ = e.store.IncrTxCount(ctx, req.UserID)

	// Update wallet drift for withdrawals
	if req.EventType == "wallet.withdraw" {
		_ = e.store.IncrWithdrawCount(ctx, req.UserID)
		_ = e.store.UpdateWalletDrift(ctx, req.UserID, -req.Amount)
	} else if req.EventType == "wallet.deposit" || req.EventType == "escrow.release" {
		_ = e.store.UpdateWalletDrift(ctx, req.UserID, req.Amount)
	}
}

// ── Audit Logging ────────────────────────────────────────────────────────────

// FraudAuditRecord is the DB model for fraud decision audit trail.
type FraudAuditRecord struct {
	ID         string  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID     string  `gorm:"type:uuid;not null"`
	EventType  string  `gorm:"size:50;not null"`
	Decision   string  `gorm:"size:20;not null"`
	RiskScore  float64 `gorm:"type:numeric(5,2)"`
	RiskLevel  string  `gorm:"size:20;not null"`
	Signals    string  `gorm:"type:jsonb"`
	RequestID  string  `gorm:"size:100"`
	TraceID    string  `gorm:"size:64"`
	ScoringMs  int     `gorm:"default:0"`
	FeatureHit bool    `gorm:"default:false"`
	CreatedAt  time.Time
}

func (FraudAuditRecord) TableName() string { return "fraud_decision_audit" }

func (e *Engine) auditLog(req ScoreRequest, result ScoreResponse) {
	if e.db == nil {
		return
	}
	signalsJSON, _ := json.Marshal(result.Signals)
	record := FraudAuditRecord{
		UserID:     req.UserID,
		EventType:  req.EventType,
		Decision:   string(result.Decision),
		RiskScore:  result.RiskScore,
		RiskLevel:  result.RiskLevel,
		Signals:    string(signalsJSON),
		RequestID:  req.RequestID,
		TraceID:    req.TraceID,
		ScoringMs:  int(result.ScoringMs),
		FeatureHit: result.FeatureHit,
		CreatedAt:  time.Now(),
	}
	if err := e.db.Create(&record).Error; err != nil {
		slog.Warn("fraud: audit log write failed", "error", err)
	}
}
