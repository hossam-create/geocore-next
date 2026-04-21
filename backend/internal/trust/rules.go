package trust

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Rule-Based Fraud Engine ──────────────────────────────────────────────────
// Implements Layer 2 Trust & Safety blueprint rules.
// When risk_score > 0.7  → auto-create trust_flag (source="rule_engine")
// When risk_score > 0.9  → auto-suspend user + flag + notify admin
// Setting "feature.ai_fraud_detection"=true will later replace with ML.

// EvaluateRequest is the input for POST /api/internal/fraud/evaluate
type EvaluateRequest struct {
	EventType string            `json:"event_type" binding:"required"`
	UserID    uuid.UUID         `json:"user_id" binding:"required"`
	Metadata  map[string]string `json:"metadata"`
}

// EvaluateResponse is returned by the fraud evaluation endpoint.
type EvaluateResponse struct {
	RiskScore      float64  `json:"risk_score"`
	Flags          []string `json:"flags"`
	Recommendation string   `json:"recommendation"` // allow, review, suspend
}

// TrustFlag represents a flag raised by the rule engine.
type TrustFlag struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	FlagType   string     `gorm:"size:100;not null" json:"flag_type"`
	Severity   string     `gorm:"size:20;not null;default:'medium'" json:"severity"`
	Source     string     `gorm:"size:100;not null;default:'rule_engine'" json:"source"`
	RiskScore  float64    `gorm:"type:numeric(5,3);default:0" json:"risk_score"`
	Metadata   string     `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	Status     string     `gorm:"size:20;not null;default:'open'" json:"status"`
	ResolvedBy *uuid.UUID `gorm:"type:uuid" json:"resolved_by,omitempty"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	Notes      string     `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (TrustFlag) TableName() string { return "trust_flags" }

// ── Rule Definitions ─────────────────────────────────────────────────────────

type Rule struct {
	Name     string
	Evaluate func(ctx context.Context, db *gorm.DB, req EvaluateRequest) (risk float64, triggered bool)
}

// AllRules returns the complete set of fraud detection rules.
func AllRules() []Rule {
	return []Rule{
		{Name: "new_account_high_value", Evaluate: ruleNewAccountHighValue},
		{Name: "velocity_spike", Evaluate: ruleVelocitySpike},
		{Name: "bid_increment_abuse", Evaluate: ruleBidIncrementAbuse},
		{Name: "repeated_reporter", Evaluate: ruleRepeatedReporter},
		{Name: "payment_mismatch", Evaluate: rulePaymentMismatch},
		{Name: "mass_listing", Evaluate: ruleMassListing},
	}
}

// ── Rule 1: new_account_high_value ───────────────────────────────────────────
// Account < 7 days + listing > $500 → risk 0.6
func ruleNewAccountHighValue(_ context.Context, db *gorm.DB, req EvaluateRequest) (float64, bool) {
	if req.EventType != "listing_create" && req.EventType != "listing" {
		return 0, false
	}
	price := parseMetaFloat(req.Metadata, "price")
	if price <= 500 {
		return 0, false
	}
	var createdAt time.Time
	db.Table("users").Where("id = ?", req.UserID).Select("created_at").Scan(&createdAt)
	if createdAt.IsZero() || time.Since(createdAt) >= 7*24*time.Hour {
		return 0, false
	}
	return 0.6, true
}

// ── Rule 2: velocity_spike ───────────────────────────────────────────────────
// > 20 bids in 1 hour → risk 0.7
func ruleVelocitySpike(_ context.Context, db *gorm.DB, req EvaluateRequest) (float64, bool) {
	if req.EventType != "bid" && req.EventType != "bid_place" {
		return 0, false
	}
	var cnt int64
	db.Table("bids").
		Where("bidder_id = ? AND created_at > ?", req.UserID, time.Now().Add(-1*time.Hour)).
		Count(&cnt)
	if cnt <= 20 {
		return 0, false
	}
	return 0.7, true
}

// ── Rule 3: bid_increment_abuse ──────────────────────────────────────────────
// Always exactly min increment on last 10 bids → risk 0.4
func ruleBidIncrementAbuse(_ context.Context, db *gorm.DB, req EvaluateRequest) (float64, bool) {
	if req.EventType != "bid" && req.EventType != "bid_place" {
		return 0, false
	}
	type BidRow struct {
		Amount       float64
		MinIncrement float64
	}
	var bids []BidRow
	db.Table("bids").
		Select("bids.amount, auctions.min_increment").
		Joins("JOIN auctions ON auctions.id = bids.auction_id").
		Where("bids.bidder_id = ?", req.UserID).
		Order("bids.created_at DESC").
		Limit(10).
		Scan(&bids)
	if len(bids) < 5 {
		return 0, false
	}
	allMinIncrement := true
	for i, b := range bids {
		if i == 0 {
			continue // skip first (no previous to compare)
		}
		diff := b.Amount - bids[i-1].Amount
		if b.MinIncrement > 0 && diff != b.MinIncrement {
			allMinIncrement = false
			break
		}
	}
	if !allMinIncrement {
		return 0, false
	}
	return 0.4, true
}

// ── Rule 4: repeated_reporter ────────────────────────────────────────────────
// Reported by 3+ unique users this week → risk 0.5
func ruleRepeatedReporter(_ context.Context, db *gorm.DB, req EvaluateRequest) (float64, bool) {
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)
	var cnt int64
	db.Table("reports").
		Where("target_user_id = ? AND created_at > ?", req.UserID, weekAgo).
		Distinct("reporter_id").Count(&cnt)
	if cnt < 3 {
		return 0, false
	}
	return 0.5, true
}

// ── Rule 5: payment_mismatch ─────────────────────────────────────────────────
// Billing country ≠ IP country → risk 0.3
func rulePaymentMismatch(_ context.Context, _ *gorm.DB, req EvaluateRequest) (float64, bool) {
	if req.EventType != "payment" && req.EventType != "checkout" {
		return 0, false
	}
	billing := req.Metadata["billing_country"]
	ipCountry := req.Metadata["ip_country"]
	if billing == "" || ipCountry == "" {
		return 0, false
	}
	if billing == ipCountry {
		return 0, false
	}
	return 0.3, true
}

// ── Rule 6: mass_listing ─────────────────────────────────────────────────────
// > 50 listings in 24h → risk 0.8
func ruleMassListing(_ context.Context, db *gorm.DB, req EvaluateRequest) (float64, bool) {
	if req.EventType != "listing_create" && req.EventType != "listing" {
		return 0, false
	}
	var cnt int64
	db.Table("listings").
		Where("seller_id = ? AND created_at > ?", req.UserID, time.Now().Add(-24*time.Hour)).
		Count(&cnt)
	if cnt <= 50 {
		return 0, false
	}
	return 0.8, true
}

// ── Engine Entry Point ───────────────────────────────────────────────────────

// Evaluate runs all rules and returns an aggregated result.
// It auto-creates trust flags and suspends users when thresholds are exceeded.
func Evaluate(ctx context.Context, db *gorm.DB, req EvaluateRequest) EvaluateResponse {
	rules := AllRules()
	resp := EvaluateResponse{Recommendation: "allow"}

	for _, rule := range rules {
		risk, triggered := rule.Evaluate(ctx, db, req)
		if triggered {
			resp.Flags = append(resp.Flags, rule.Name)
			if risk > resp.RiskScore {
				resp.RiskScore = risk
			}
		}
	}

	// Determine recommendation
	switch {
	case resp.RiskScore > 0.9:
		resp.Recommendation = "suspend"
	case resp.RiskScore > 0.7:
		resp.Recommendation = "review"
	default:
		resp.Recommendation = "allow"
	}

	// Auto-actions
	if resp.RiskScore > 0.7 {
		meta, _ := json.Marshal(req.Metadata)
		severity := "high"
		if resp.RiskScore > 0.9 {
			severity = "critical"
		}
		flag := TrustFlag{
			UserID:    req.UserID,
			FlagType:  flagTypeFromRules(resp.Flags),
			Severity:  severity,
			Source:    "rule_engine",
			RiskScore: resp.RiskScore,
			Metadata:  string(meta),
			Status:    "open",
		}
		db.Create(&flag)
	}

	if resp.RiskScore > 0.9 {
		// Auto-suspend user
		db.Table("users").Where("id = ?", req.UserID).Updates(map[string]interface{}{
			"is_banned":  true,
			"ban_reason": fmt.Sprintf("Auto-suspended by fraud engine: %v (score: %.2f)", resp.Flags, resp.RiskScore),
		})
	}

	return resp
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func parseMetaFloat(meta map[string]string, key string) float64 {
	v, ok := meta[key]
	if !ok {
		return 0
	}
	var f float64
	fmt.Sscanf(v, "%f", &f)
	return f
}

func flagTypeFromRules(flags []string) string {
	if len(flags) == 0 {
		return "unknown"
	}
	return flags[0]
}
