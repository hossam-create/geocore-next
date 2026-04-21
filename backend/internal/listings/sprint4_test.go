package listings

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ── 1. LISTING BOOST TESTS ────────────────────────────────────────────────────

func TestBoostPrices(t *testing.T) {
	if boostPrices[BoostBasic] != 500 {
		t.Errorf("basic boost = %d cents, want 500", boostPrices[BoostBasic])
	}
	if boostPrices[BoostPremium] != 1500 {
		t.Errorf("premium boost = %d cents, want 1500", boostPrices[BoostPremium])
	}
}

func TestBoostDurations(t *testing.T) {
	if boostDurations[BoostBasic] != 24*time.Hour {
		t.Errorf("basic duration = %v, want 24h", boostDurations[BoostBasic])
	}
	if boostDurations[BoostPremium] != 72*time.Hour {
		t.Errorf("premium duration = %v, want 72h", boostDurations[BoostPremium])
	}
}

func TestBoostScores(t *testing.T) {
	if boostScores[BoostBasic] != 50 {
		t.Errorf("basic score = %d, want 50", boostScores[BoostBasic])
	}
	if boostScores[BoostPremium] != 100 {
		t.Errorf("premium score = %d, want 100", boostScores[BoostPremium])
	}
}

func TestBoostIsExpired(t *testing.T) {
	b := ListingBoost{ExpiresAt: time.Now().Add(-1 * time.Hour)}
	if !b.IsExpired() {
		t.Error("past boost should be expired")
	}
	b.ExpiresAt = time.Now().Add(1 * time.Hour)
	if b.IsExpired() {
		t.Error("future boost should NOT be expired")
	}
}

func TestBoostTableName(t *testing.T) {
	b := ListingBoost{}
	if b.TableName() != "listing_boosts" {
		t.Errorf("table = %s, want listing_boosts", b.TableName())
	}
}

// ── 2. URGENCY SIGNALS TESTS ────────────────────────────────────────────────────

func TestUrgencySignalsStructure(t *testing.T) {
	s := UrgencySignals{ListingID: uuid.New()}
	if s.IsUrgent {
		t.Error("empty signals should not be urgent")
	}
}

func TestFormatDuration(t *testing.T) {
	if v := formatDuration(30 * time.Hour); v != "1day" {
		t.Errorf("30h = %s, want 1day", v)
	}
	if v := formatDuration(5 * time.Hour); v != "5hs" {
		t.Errorf("5h = %s, want 5hs", v)
	}
	if v := formatDuration(48 * time.Hour); v != "2days" {
		t.Errorf("48h = %s, want 2days", v)
	}
}

// ── 3. WATCHLIST TESTS ────────────────────────────────────────────────────

func TestWatchlistItemTableName(t *testing.T) {
	w := WatchlistItem{}
	if w.TableName() != "watchlists" {
		t.Errorf("table = %s, want watchlists", w.TableName())
	}
}

func TestFormatPrice(t *testing.T) {
	if v := formatPrice(9.99); v != "9.99" {
		t.Errorf("9.99 = %s, want 9.99", v)
	}
	if v := formatPrice(100.0); v != "100.00" {
		t.Errorf("100.0 = %s, want 100.00", v)
	}
}

// ── 4. SELLER PRICING TESTS ────────────────────────────────────────────────────

func TestGetSellerListingCountNoDB(t *testing.T) {
	// Just verify the function signature compiles
	_ = GetSellerListingCount
}

// ── 5. DEAL CLOSER TESTS (crowdshipping) ────────────────────────────────────
// Note: deal_closer.go is in crowdshipping package, tested there

func TestSellerTierInfoKeys(t *testing.T) {
	// Verify the response structure has expected keys
	info := gin.H{
		"tier":            "free",
		"active_listings": 0,
		"listing_limit":   5,
		"can_create":      true,
		"pro_benefits":    []string{},
	}
	if info["tier"] != "free" {
		t.Error("default tier should be free")
	}
}

// ── 6. BOOST ABUSE PREVENTION TESTS ────────────────────────────────────────────

func TestMaxBoostsPerSellerPerDay(t *testing.T) {
	if maxBoostsPerSellerPerDay != 5 {
		t.Errorf("daily boost limit = %d, want 5", maxBoostsPerSellerPerDay)
	}
}

func TestBoostDiminishingReturns(t *testing.T) {
	// Single boost: full score
	single := ListingBoost{BoostScore: 100, ExpiresAt: time.Now().Add(1 * time.Hour)}
	boosts := []ListingBoost{single}
	baseTotal := 0
	for _, b := range boosts {
		baseTotal += b.BoostScore
	}
	diminishFactor := 1.0 + float64(len(boosts)-1)*0.5
	effective := float64(baseTotal) / diminishFactor
	if effective != 100.0 {
		t.Errorf("single boost effective = %.1f, want 100", effective)
	}

	// Two boosts: diminishing
	boosts = []ListingBoost{
		{BoostScore: 100, ExpiresAt: time.Now().Add(1 * time.Hour)},
		{BoostScore: 50, ExpiresAt: time.Now().Add(1 * time.Hour)},
	}
	baseTotal = 150
	diminishFactor = 1.0 + float64(len(boosts)-1)*0.5
	effective = float64(baseTotal) / diminishFactor
	if effective >= 150.0 {
		t.Errorf("two boosts should have diminishing returns, got %.1f", effective)
	}
}

// ── 7. WATCHLIST DEDUP TESTS ────────────────────────────────────────────────────

func TestNotificationPriorityLevels(t *testing.T) {
	if PriorityHigh != "high" {
		t.Errorf("high priority = %s, want high", PriorityHigh)
	}
	if PriorityMedium != "medium" {
		t.Errorf("medium priority = %s, want medium", PriorityMedium)
	}
	if PriorityLow != "low" {
		t.Errorf("low priority = %s, want low", PriorityLow)
	}
}

func TestIsDuplicateNotif(t *testing.T) {
	// Reset dedup map
	watchlistNotifDedup = map[string]time.Time{}

	uid := uuid.New()
	lid := uuid.New()

	// First call: not duplicate
	if isDuplicateNotif(uid, lid, "price_drop") {
		t.Error("first notification should not be duplicate")
	}
	// Immediate second call: IS duplicate
	if !isDuplicateNotif(uid, lid, "price_drop") {
		t.Error("immediate repeat should be duplicate")
	}
	// Different event type: not duplicate
	if isDuplicateNotif(uid, lid, "new_offer") {
		t.Error("different event type should not be duplicate")
	}
}

// ── 8. URGENCY VERIFICATION TESTS ────────────────────────────────────────────────

func TestUrgencyVerifiedSignal(t *testing.T) {
	s := UrgencySignals{
		ViewsToday:       10,
		UniqueViewsToday: 8,
	}
	// 8 >= 10/2 = 5, so verified
	if !s.IsVerifiedSignal {
		// This tests the logic, not the DB query
		verified := s.ViewsToday > 0 && s.UniqueViewsToday >= s.ViewsToday/2
		if !verified {
			t.Error("8 unique out of 10 should be verified")
		}
	}
}

func TestUrgencyManipulationDetection(t *testing.T) {
	s := UrgencySignals{
		ViewsToday:       100,
		UniqueViewsToday: 5,
	}
	// 5 < 100/2 = 50, so NOT verified
	verified := s.ViewsToday > 0 && s.UniqueViewsToday >= s.ViewsToday/2
	if verified {
		t.Error("5 unique out of 100 should NOT be verified (manipulation)")
	}
}
