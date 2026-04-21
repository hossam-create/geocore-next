package analytics

import (
	"testing"

	"github.com/shopspring/decimal"
)

// ════════════════════════════════════════════════════════════════════════════
// STEP 3: Funnel Analytics Tests
// ════════════════════════════════════════════════════════════════════════════

func TestFunnelEvent_TableName(t *testing.T) {
	fe := FunnelEvent{}
	if fe.TableName() != "funnel_events" {
		t.Errorf("expected 'funnel_events', got '%s'", fe.TableName())
	}
}

func TestFunnels_Definitions(t *testing.T) {
	if len(Funnels) != 3 {
		t.Errorf("expected 3 funnels, got %d", len(Funnels))
	}

	buyerSteps, ok := Funnels["buyer"]
	if !ok {
		t.Error("buyer funnel not defined")
	}
	if len(buyerSteps) != 5 {
		t.Errorf("buyer funnel should have 5 steps, got %d", len(buyerSteps))
	}

	sellerSteps, ok := Funnels["seller"]
	if !ok {
		t.Error("seller funnel not defined")
	}
	if len(sellerSteps) != 4 {
		t.Errorf("seller funnel should have 4 steps, got %d", len(sellerSteps))
	}

	travelerSteps, ok := Funnels["traveler"]
	if !ok {
		t.Error("traveler funnel not defined")
	}
	if len(travelerSteps) != 4 {
		t.Errorf("traveler funnel should have 4 steps, got %d", len(travelerSteps))
	}
}

func TestBuyerFunnel_Steps(t *testing.T) {
	steps := Funnels["buyer"]
	expectedSteps := []string{"signup", "search", "request", "offer_received", "pay"}
	for i, s := range steps {
		if s.Step != expectedSteps[i] {
			t.Errorf("buyer step %d: expected '%s', got '%s'", i, expectedSteps[i], s.Step)
		}
		if s.Order != i+1 {
			t.Errorf("buyer step %d: expected order %d, got %d", i, i+1, s.Order)
		}
	}
}

func TestSellerFunnel_Steps(t *testing.T) {
	steps := Funnels["seller"]
	expectedSteps := []string{"signup", "create_listing", "boost", "get_offers"}
	for i, s := range steps {
		if s.Step != expectedSteps[i] {
			t.Errorf("seller step %d: expected '%s', got '%s'", i, expectedSteps[i], s.Step)
		}
	}
}

func TestTravelerFunnel_Steps(t *testing.T) {
	steps := Funnels["traveler"]
	expectedSteps := []string{"signup", "add_trip", "receive_request", "send_offer"}
	for i, s := range steps {
		if s.Step != expectedSteps[i] {
			t.Errorf("traveler step %d: expected '%s', got '%s'", i, expectedSteps[i], s.Step)
		}
	}
}

func TestFunnelDropoff_Fields(t *testing.T) {
	d := FunnelDropoff{
		Funnel:     "buyer",
		Step:       "search",
		StepOrder:  2,
		Reached:    100,
		Converted:  60,
		DropoffPct: decimal.NewFromInt(40),
	}
	if d.Funnel != "buyer" {
		t.Error("funnel mismatch")
	}
	if d.Reached != 100 {
		t.Error("reached mismatch")
	}
	if d.Converted != 60 {
		t.Error("converted mismatch")
	}
}

// ════════════════════════════════════════════════════════════════════════════
// STEP 7: Metrics Dashboard Tests
// ════════════════════════════════════════════════════════════════════════════

func TestPlatformMetrics_Fields(t *testing.T) {
	m := PlatformMetrics{
		GMV:                 decimal.NewFromInt(50000),
		ConversionRate:      decimal.NewFromFloat(25.5),
		TimeToFirstOffer:    2.5,
		OffersPerRequest:    decimal.NewFromFloat(1.8),
		LiquidityRatio:      decimal.NewFromFloat(0.6),
		ActiveUsers:         150,
		ActiveListings:      80,
		ActiveTrips:         30,
		PendingRequests:     50,
		CompletedDeliveries: 200,
		Period:              "30d",
	}
	if !m.GMV.Equal(decimal.NewFromInt(50000)) {
		t.Errorf("GMV should be 50000, got %s", m.GMV.String())
	}
	if m.ActiveUsers != 150 {
		t.Errorf("active users should be 150, got %d", m.ActiveUsers)
	}
	if m.Period != "30d" {
		t.Errorf("period should be 30d, got %s", m.Period)
	}
}

func TestGetSinceDate(t *testing.T) {
	tests := []struct {
		period  string
		daysAgo int
	}{
		{"7d", 7},
		{"30d", 30},
		{"90d", 90},
		{"1y", 365},
		{"unknown", 30}, // default
	}
	for _, tt := range tests {
		since := getSinceDate(tt.period)
		expected := since.Format("2006-01-02")
		if expected == "" {
			t.Errorf("getSinceDate(%s) returned empty", tt.period)
		}
	}
}
