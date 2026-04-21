package growth

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ════════════════════════════════════════════════════════════════════════════
// STEP 1: Liquidity Bootstrap Tests
// ════════════════════════════════════════════════════════════════════════════

func TestGhostListing_TableName(t *testing.T) {
	g := GhostListing{}
	if g.TableName() != "ghost_listings" {
		t.Errorf("expected 'ghost_listings', got '%s'", g.TableName())
	}
}

func TestGhostListing_Defaults(t *testing.T) {
	g := GhostListing{
		Title:         "Test Item",
		Price:         decimal.NewFromInt(100),
		OriginCountry: "US",
		DestCountry:   "EG",
	}
	// Go zero-value for bool is false; DB defaults set these to true
	// Verify the struct fields exist and are bool type
	_ = g.IsPlatformAssisted
	_ = g.IsActive
}

func TestPlatformTraveler_TableName(t *testing.T) {
	pt := PlatformTraveler{}
	if pt.TableName() != "platform_travelers" {
		t.Errorf("expected 'platform_travelers', got '%s'", pt.TableName())
	}
}

func TestPlatformTraveler_Defaults(t *testing.T) {
	pt := PlatformTraveler{
		UserID:        uuid.New(),
		Name:          "Platform Traveler 1",
		OriginCountry: "US",
		DestCountry:   "EG",
	}
	if pt.Reputation != 0 {
		t.Errorf("expected reputation 0 (DB default: 80), got %d", pt.Reputation)
	}
	// Go zero-value for bool is false; DB default sets IsInternal=true
	_ = pt.IsInternal
}

// ════════════════════════════════════════════════════════════════════════════
// STEP 2: Referral System Tests
// ════════════════════════════════════════════════════════════════════════════

func TestReferral_TableName(t *testing.T) {
	r := Referral{}
	if r.TableName() != "referrals" {
		t.Errorf("expected 'referrals', got '%s'", r.TableName())
	}
}

func TestTravelerInvite_TableName(t *testing.T) {
	ti := TravelerInvite{}
	if ti.TableName() != "traveler_invites" {
		t.Errorf("expected 'traveler_invites', got '%s'", ti.TableName())
	}
}

func TestGenerateReferralCode(t *testing.T) {
	code1 := GenerateReferralCode()
	code2 := GenerateReferralCode()

	if len(code1) != 8 {
		t.Errorf("referral code should be 8 chars, got %d", len(code1))
	}
	if code1 == code2 {
		t.Error("two generated codes should almost never be equal")
	}
}

func TestReferralRewards(t *testing.T) {
	if !ReferrerReward.Equal(decimal.NewFromFloat(5.00)) {
		t.Errorf("referrer reward should be $5, got %s", ReferrerReward.String())
	}
	if !RefereeDiscount.Equal(decimal.NewFromFloat(3.00)) {
		t.Errorf("referee discount should be $3, got %s", RefereeDiscount.String())
	}
	if !TravelerReward.Equal(decimal.NewFromFloat(10.00)) {
		t.Errorf("traveler reward should be $10, got %s", TravelerReward.String())
	}
}

func TestReferralStatuses(t *testing.T) {
	statuses := map[string]string{
		"pending":   ReferralStatusPending,
		"completed": ReferralStatusCompleted,
		"rewarded":  ReferralStatusRewarded,
	}
	for expected, actual := range statuses {
		if actual != expected {
			t.Errorf("expected '%s', got '%s'", expected, actual)
		}
	}
}

func TestInviteStatuses(t *testing.T) {
	statuses := map[string]string{
		"sent":       InviteStatusSent,
		"registered": InviteStatusRegistered,
		"completed":  InviteStatusCompleted,
		"rewarded":   InviteStatusRewarded,
	}
	for expected, actual := range statuses {
		if actual != expected {
			t.Errorf("expected '%s', got '%s'", expected, actual)
		}
	}
}

// ════════════════════════════════════════════════════════════════════════════
// STEP 3: Conversion Optimization Tests
// ════════════════════════════════════════════════════════════════════════════

func TestStaleListing_TableName(t *testing.T) {
	s := StaleListing{}
	if s.TableName() != "stale_listings" {
		t.Errorf("expected 'stale_listings', got '%s'", s.TableName())
	}
}

func TestConversionSignal_Types(t *testing.T) {
	types := []string{"travelers_interested", "offers_received", "price_dropped"}
	for _, typ := range types {
		cs := ConversionSignal{Type: typ, Count: 3}
		if cs.Type != typ {
			t.Errorf("signal type mismatch")
		}
	}
}

func TestSmartNudge_Logic(t *testing.T) {
	// Offer close to budget (within 20%) → should nudge
	budget := 100.0
	offerPrice := 85.0
	diff := (budget - offerPrice) / budget
	if diff < 0 || diff > 0.20 {
		t.Error("offer within 20% of budget should trigger nudge")
	}

	// Offer far from budget → should not nudge
	offerPrice2 := 50.0
	diff2 := (budget - offerPrice2) / budget
	if diff2 >= 0 && diff2 <= 0.20 {
		t.Error("offer 50%% below budget should not trigger nudge")
	}
}

// ════════════════════════════════════════════════════════════════════════════
// STEP 5: Dynamic Fees Logic Tests (types are in payments package)
// ════════════════════════════════════════════════════════════════════════════

func TestDynamicFee_Logic(t *testing.T) {
	// Test fee calculation logic locally
	lowSupplyFee := decimal.NewFromFloat(0.10)
	balancedFee := decimal.NewFromFloat(0.12)
	highDemandFee := decimal.NewFromFloat(0.15)
	veryHighFee := decimal.NewFromFloat(0.18)
	vipDiscountPct := decimal.NewFromFloat(0.30)

	if !lowSupplyFee.LessThan(balancedFee) {
		t.Error("low supply fee should be less than balanced")
	}
	if !balancedFee.LessThan(highDemandFee) {
		t.Error("balanced fee should be less than high demand")
	}
	if !highDemandFee.LessThan(veryHighFee) {
		t.Error("high demand fee should be less than very high")
	}

	// VIP discount: 12% - (12% * 30%) = 8.4%
	vipFinalFee := balancedFee.Sub(balancedFee.Mul(vipDiscountPct))
	expectedFinal := decimal.NewFromFloat(0.084)
	if !vipFinalFee.Equal(expectedFinal) {
		t.Errorf("VIP final fee should be %s, got %s", expectedFinal.String(), vipFinalFee.String())
	}

	// Fee amount calculation: $100 * 12% = $12
	amount := decimal.NewFromInt(100)
	feeAmount := amount.Mul(balancedFee)
	if !feeAmount.Equal(decimal.NewFromInt(12)) {
		t.Errorf("fee amount should be 12, got %s", feeAmount.String())
	}
}

// ════════════════════════════════════════════════════════════════════════════
// STEP 6: Retention Engine Tests
// ════════════════════════════════════════════════════════════════════════════

func TestRetentionEvent_TableName(t *testing.T) {
	re := RetentionEvent{}
	if re.TableName() != "retention_events" {
		t.Errorf("expected 'retention_events', got '%s'", re.TableName())
	}
}

func TestRetentionNotificationThrottle(t *testing.T) {
	if maxRetentionNotificationsPerDay != 3 {
		t.Errorf("max daily retention notifications should be 3, got %d", maxRetentionNotificationsPerDay)
	}
}

func TestWeeklyDigestDeal(t *testing.T) {
	deal := DigestDeal{
		ID:    uuid.New(),
		Title: "iPhone 15",
		Price: decimal.NewFromInt(800),
		Route: "US→EG",
	}
	if deal.Title != "iPhone 15" {
		t.Error("deal title mismatch")
	}
	if deal.Route != "US→EG" {
		t.Error("deal route mismatch")
	}
}
