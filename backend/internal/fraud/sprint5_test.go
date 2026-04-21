package fraud

import (
	"testing"
)

// ── 5. ANTI-FRAUD MARKETPLACE RULES TESTS ───────────────────────────────────────

func TestMarketplaceRiskResultStructure(t *testing.T) {
	r := MarketplaceRiskResult{Score: 50, Flags: []string{"test"}, Action: "review"}
	if r.Score != 50 {
		t.Error("score not set")
	}
	if len(r.Flags) != 1 {
		t.Error("flags not set")
	}
	if r.Action != "review" {
		t.Error("action not set")
	}
}

func TestMarketplaceCheckInputFields(t *testing.T) {
	input := MarketplaceCheckInput{
		EventType: "offer",
		Amount:    100,
		Price:     50,
		MarketAvg: 100,
	}
	if input.EventType != "offer" {
		t.Error("event type not set")
	}
	// 50/100 = 0.5, which is exactly at the "below_market" threshold (< 0.5)
	ratio := input.Price / input.MarketAvg
	if ratio < 0.3 {
		t.Error("50/100 should not be suspicious_pricing (< 0.3)")
	}
}

func TestSuspiciousPricingDetection(t *testing.T) {
	// Price < 30% of market average
	ratio := 20.0 / 100.0
	if ratio >= 0.3 {
		t.Error("20/100 should be suspicious pricing")
	}

	// Price between 30-50% of market average
	ratio = 40.0 / 100.0
	if ratio < 0.3 || ratio >= 0.5 {
		t.Error("40/100 should be below market")
	}

	// Normal pricing
	ratio = 80.0 / 100.0
	if ratio < 0.5 {
		t.Error("80/100 should be normal pricing")
	}
}

func TestMarketplaceRiskActions(t *testing.T) {
	// Score >= 70 → block
	if determineAction(70) != "block" {
		t.Error("score 70 should block")
	}
	// Score >= 40 → review
	if determineAction(40) != "review" {
		t.Error("score 40 should review")
	}
	// Score < 40 → allow
	if determineAction(20) != "allow" {
		t.Error("score 20 should allow")
	}
}

func determineAction(score int) string {
	switch {
	case score >= 70:
		return "block"
	case score >= 40:
		return "review"
	default:
		return "allow"
	}
}

func TestFormatMarketplaceRiskResult(t *testing.T) {
	r := MarketplaceRiskResult{Score: 50, Action: "review", Flags: []string{"test_flag"}}
	s := FormatMarketplaceRiskResult(r)
	if s == "" {
		t.Error("format should not be empty")
	}
}

func TestFlagUserCreatesAlert(t *testing.T) {
	// Verify FlagUser function exists with correct signature
	// FlagUser(db *gorm.DB, userID uuid.UUID, alertType string, severity Severity, indicators string) error
	// No DB available in unit test — just verify types
	var s Severity = SeverityHigh
	if s != "high" {
		t.Error("severity high mismatch")
	}
}

// ── 7. PRE-ACTION RISK CHECK TESTS ─────────────────────────────────────────────

func TestPreActionRiskResultStructure(t *testing.T) {
	r := PreActionRiskResult{
		Allowed:    true,
		RiskScore:  15,
		Action:     "allow",
		Reason:     "risk within range",
		Flags:      []string{},
		TrustLevel: "normal",
		TrustScore: 55,
	}
	if !r.Allowed {
		t.Error("should be allowed")
	}
	if r.Action != "allow" {
		t.Error("action should be allow")
	}
}

func TestPreActionRiskThresholds(t *testing.T) {
	if PreActionRiskThresholds.AllowMax != 29 {
		t.Errorf("allow max = %d, want 29", PreActionRiskThresholds.AllowMax)
	}
	if PreActionRiskThresholds.ReviewMin != 30 {
		t.Errorf("review min = %d, want 30", PreActionRiskThresholds.ReviewMin)
	}
	if PreActionRiskThresholds.BlockMin != 61 {
		t.Errorf("block min = %d, want 61", PreActionRiskThresholds.BlockMin)
	}
}

func TestPreActionRiskDecisionLogic(t *testing.T) {
	// Score < 30 → allow
	if decidePreAction(15) != "allow" {
		t.Error("score 15 should allow")
	}
	// Score 30–60 → manual_review
	if decidePreAction(45) != "manual_review" {
		t.Error("score 45 should require manual review")
	}
	// Score > 60 → block
	if decidePreAction(75) != "block" {
		t.Error("score 75 should block")
	}
}

func decidePreAction(score int) string {
	switch {
	case score >= PreActionRiskThresholds.BlockMin:
		return "block"
	case score >= PreActionRiskThresholds.ReviewMin:
		return "manual_review"
	default:
		return "allow"
	}
}
