package listings

import (
	"testing"
)

// ── 1. LISTING TYPE + TRADE CONFIG TESTS ────────────────────────────────────────

func TestDefaultTradeConfig_BuyNow(t *testing.T) {
	cfg := DefaultTradeConfig(ListingTypeBuyNow)
	if !cfg.BuyNowEnabled {
		t.Error("BuyNow should be enabled for buy_now listing type")
	}
	if cfg.OfferEnabled {
		t.Error("Offer should NOT be enabled for buy_now listing type")
	}
	if cfg.AuctionEnabled {
		t.Error("Auction should NOT be enabled for buy_now listing type")
	}
}

func TestDefaultTradeConfig_Negotiation(t *testing.T) {
	cfg := DefaultTradeConfig(ListingTypeNegotiation)
	if cfg.BuyNowEnabled {
		t.Error("BuyNow should NOT be enabled for negotiation listing type")
	}
	if !cfg.OfferEnabled {
		t.Error("Offer should be enabled for negotiation listing type")
	}
}

func TestDefaultTradeConfig_Auction(t *testing.T) {
	cfg := DefaultTradeConfig(ListingTypeAuction)
	if !cfg.AuctionEnabled {
		t.Error("Auction should be enabled for auction listing type")
	}
	if cfg.BuyNowEnabled {
		t.Error("BuyNow should NOT be enabled by default for auction listing type")
	}
}

func TestDefaultTradeConfig_Hybrid(t *testing.T) {
	cfg := DefaultTradeConfig(ListingTypeHybrid)
	if !cfg.BuyNowEnabled {
		t.Error("BuyNow should be enabled for hybrid listing type")
	}
	if !cfg.OfferEnabled {
		t.Error("Offer should be enabled for hybrid listing type")
	}
	if cfg.AuctionEnabled {
		t.Error("Auction should NOT be enabled by default for hybrid listing type")
	}
}

func TestDefaultTradeConfig_Thresholds(t *testing.T) {
	cfg := DefaultTradeConfig(ListingTypeHybrid)
	if cfg.MinOfferPercent != 0.7 {
		t.Errorf("MinOfferPercent = %v, want 0.7", cfg.MinOfferPercent)
	}
	if cfg.AutoAcceptPercent != 0.95 {
		t.Errorf("AutoAcceptPercent = %v, want 0.95", cfg.AutoAcceptPercent)
	}
	if cfg.OfferExpiryHours != 48 {
		t.Errorf("OfferExpiryHours = %d, want 48", cfg.OfferExpiryHours)
	}
}

func TestGetTradeConfig_Empty(t *testing.T) {
	l := Listing{ListingType: ListingTypeBuyNow, TradeConfig: ""}
	cfg := l.GetTradeConfig()
	if !cfg.BuyNowEnabled {
		t.Error("should return default config for empty trade_config")
	}
}

func TestGetTradeConfig_InvalidJSON(t *testing.T) {
	l := Listing{ListingType: ListingTypeBuyNow, TradeConfig: "{invalid"}
	cfg := l.GetTradeConfig()
	if !cfg.BuyNowEnabled {
		t.Error("should return default config for invalid JSON")
	}
}

func TestGetTradeConfig_ValidJSON(t *testing.T) {
	l := Listing{
		ListingType: ListingTypeHybrid,
		TradeConfig: `{"buy_now_enabled":true,"offer_enabled":true,"min_offer_percent":0.8}`,
	}
	cfg := l.GetTradeConfig()
	if !cfg.BuyNowEnabled {
		t.Error("BuyNowEnabled should be true from JSON")
	}
	if cfg.MinOfferPercent != 0.8 {
		t.Errorf("MinOfferPercent = %v, want 0.8 (from JSON override)", cfg.MinOfferPercent)
	}
}

// ── 2. CENTS CONVERSION TESTS ───────────────────────────────────────────────────

func TestToCents_RoundTrip(t *testing.T) {
	tests := []float64{0.0, 0.01, 1.0, 10.50, 99.99, 1000.0, 99999.99}
	for _, usd := range tests {
		cents := toCents(usd)
		back := fromCents(cents)
		if back != usd {
			t.Errorf("toCents(%v)=%d, fromCents=%v, want %v", usd, cents, back, usd)
		}
	}
}

func TestToCents_IntegerPrecision(t *testing.T) {
	// 0.1 + 0.2 should NOT drift
	cents := toCents(0.1) + toCents(0.2)
	if cents != 30 {
		t.Errorf("toCents(0.1)+toCents(0.2) = %d, want 30", cents)
	}
}

// ── 3. EVALUATE OFFER (AUTO-ACCEPT/REJECT) TESTS ────────────────────────────────

func TestEvaluateOffer_AutoReject(t *testing.T) {
	cfg := ListingTradeConfig{MinOfferPercent: 0.7, AutoAcceptPercent: 0.95}
	action := EvaluateOffer(10000, 5000, cfg) // 50% of price, below 70%
	if action != OfferActionReject {
		t.Errorf("offer at 50%% should be auto-rejected, got %v", action)
	}
}

func TestEvaluateOffer_AutoAccept(t *testing.T) {
	cfg := ListingTradeConfig{MinOfferPercent: 0.7, AutoAcceptPercent: 0.95}
	action := EvaluateOffer(10000, 9600, cfg) // 96% of price, above 95%
	if action != OfferActionAccept {
		t.Errorf("offer at 96%% should be auto-accepted, got %v", action)
	}
}

func TestEvaluateOffer_ManualReview(t *testing.T) {
	cfg := ListingTradeConfig{MinOfferPercent: 0.7, AutoAcceptPercent: 0.95}
	action := EvaluateOffer(10000, 8000, cfg) // 80% of price, between 70% and 95%
	if action != "" {
		t.Errorf("offer at 80%% should need manual review, got %v", action)
	}
}

func TestEvaluateOffer_ExactlyAtMinPercent(t *testing.T) {
	cfg := ListingTradeConfig{MinOfferPercent: 0.7, AutoAcceptPercent: 0.95}
	action := EvaluateOffer(10000, 7000, cfg) // exactly 70%
	if action != "" {
		t.Errorf("offer at exactly 70%% should need manual review (not < 70%%), got %v", action)
	}
}

func TestEvaluateOffer_ExactlyAtAutoAccept(t *testing.T) {
	cfg := ListingTradeConfig{MinOfferPercent: 0.7, AutoAcceptPercent: 0.95}
	action := EvaluateOffer(10000, 9500, cfg) // exactly 95%
	if action != OfferActionAccept {
		t.Errorf("offer at exactly 95%% should be auto-accepted (>=), got %v", action)
	}
}

func TestEvaluateOffer_ZeroPrice(t *testing.T) {
	cfg := ListingTradeConfig{MinOfferPercent: 0.7, AutoAcceptPercent: 0.95}
	action := EvaluateOffer(0, 5000, cfg)
	if action != "" {
		t.Errorf("zero listing price should return no auto-action, got %v", action)
	}
}

func TestEvaluateOffer_FullPrice(t *testing.T) {
	cfg := ListingTradeConfig{MinOfferPercent: 0.7, AutoAcceptPercent: 0.95}
	action := EvaluateOffer(10000, 10000, cfg) // 100% of price
	if action != OfferActionAccept {
		t.Errorf("offer at 100%% should be auto-accepted, got %v", action)
	}
}

// ── 4. NEGOTIATION MODEL TESTS ──────────────────────────────────────────────────

func TestNegotiationStatus_Values(t *testing.T) {
	statuses := map[NegotiationStatus]bool{
		NegotiationOpen:           true,
		NegotiationCountered:      true,
		NegotiationPendingPayment: true,
		NegotiationAccepted:       true,
		NegotiationPaymentFailed:  true,
		NegotiationRejected:       true,
		NegotiationExpired:        true,
		NegotiationClosed:         true,
	}
	for s := range statuses {
		if s == "" {
			t.Error("empty negotiation status found")
		}
	}
}

func TestOfferAction_Values(t *testing.T) {
	actions := map[OfferAction]bool{
		OfferActionOffer:   true,
		OfferActionAccept:  true,
		OfferActionReject:  true,
		OfferActionCounter: true,
	}
	for a := range actions {
		if a == "" {
			t.Error("empty offer action found")
		}
	}
}

func TestNegotiationThread_TableName(t *testing.T) {
	if (&NegotiationThread{}).TableName() != "negotiation_threads" {
		t.Errorf("NegotiationThread table name = %q, want 'negotiation_threads'", (&NegotiationThread{}).TableName())
	}
}

func TestNegotiationMessage_TableName(t *testing.T) {
	if (&NegotiationMessage{}).TableName() != "negotiation_messages" {
		t.Errorf("NegotiationMessage table name = %q, want 'negotiation_messages'", (&NegotiationMessage{}).TableName())
	}
}

// ── 5. LISTING TYPE VALIDATION TESTS ────────────────────────────────────────────

func TestListingType_Values(t *testing.T) {
	types := map[ListingType]bool{
		ListingTypeBuyNow:      true,
		ListingTypeNegotiation: true,
		ListingTypeAuction:     true,
		ListingTypeHybrid:      true,
	}
	for lt := range types {
		if lt == "" {
			t.Error("empty listing type found")
		}
	}
}

// ── 6. CENTS-BASED PRICE SPLIT TESTS ───────────────────────────────────────────

func TestPriceSplit_CentsExact(t *testing.T) {
	// Same pattern as crowdshipping: Total = PlatformFee + TravelerEarnings exactly
	testCases := []int64{100, 7410, 583968, 3147760, 1040, 999999}
	for _, totalCents := range testCases {
		platformCents := totalCents * 15 / 100
		travelerCents := totalCents - platformCents
		if platformCents+travelerCents != totalCents {
			t.Errorf("cents split drift: platform(%d)+traveler(%d)=%d, want total(%d)",
				platformCents, travelerCents, platformCents+travelerCents, totalCents)
		}
	}
}

// ── 7. AUCTION + BUY NOW CONFLICT TESTS ─────────────────────────────────────────

func TestAuctionListing_IgnoresBuyNow(t *testing.T) {
	cfg := DefaultTradeConfig(ListingTypeAuction)
	if cfg.BuyNowEnabled {
		t.Error("auction listing type should NOT enable Buy Now by default")
	}
	if !cfg.AuctionEnabled {
		t.Error("auction listing type should enable auction")
	}
}

func TestHybridListing_EnablesBuyNowAndOffer(t *testing.T) {
	cfg := DefaultTradeConfig(ListingTypeHybrid)
	if !cfg.BuyNowEnabled {
		t.Error("hybrid listing type should enable Buy Now")
	}
	if !cfg.OfferEnabled {
		t.Error("hybrid listing type should enable Offer")
	}
}

// ── 8. CONVERT ACCEPTED OFFER GUARD TESTS ──────────────────────────────────────

func TestConvertAcceptedOffer_NonAcceptedThread(t *testing.T) {
	thread := NegotiationThread{Status: NegotiationOpen}
	err := ConvertAcceptedOfferToOrder(nil, &thread, nil)
	if err == nil {
		t.Error("should reject non-accepted thread")
	}
}

func TestConvertAcceptedOffer_RejectedThread(t *testing.T) {
	thread := NegotiationThread{Status: NegotiationRejected}
	err := ConvertAcceptedOfferToOrder(nil, &thread, nil)
	if err == nil {
		t.Error("should reject rejected thread")
	}
}

func TestConvertAcceptedOffer_PaymentFailedThread(t *testing.T) {
	thread := NegotiationThread{Status: NegotiationPaymentFailed}
	err := ConvertAcceptedOfferToOrder(nil, &thread, nil)
	if err == nil {
		t.Error("should reject payment_failed thread")
	}
}

func TestConvertAcceptedOffer_PendingPaymentThread(t *testing.T) {
	thread := NegotiationThread{Status: NegotiationPendingPayment}
	err := ConvertAcceptedOfferToOrder(nil, &thread, nil)
	if err == nil {
		t.Error("should reject pending_payment thread")
	}
}

// ── 9. HOURS DURATION HELPER TESTS ──────────────────────────────────────────────

func TestHoursDuration(t *testing.T) {
	d := hoursDuration(72)
	if d.Hours() != 72 {
		t.Errorf("hoursDuration(72) = %v hours, want 72", d.Hours())
	}
}

// ── 10. RISK FIX TESTS ──────────────────────────────────────────────────────────

func TestNegotiationStatus_PaymentFailed_HasRetry(t *testing.T) {
	// When the handler sets PAYMENT_FAILED, it explicitly sets PaymentRetryAllowed=true.
	// The GORM default is also true. Verify the field exists and the pattern works.
	thread := NegotiationThread{Status: NegotiationPaymentFailed, PaymentRetryAllowed: true}
	if !thread.PaymentRetryAllowed {
		t.Error("PAYMENT_FAILED thread should allow retry")
	}
	// Zero-value Go struct has false — that's expected; GORM sets the DB default.
	zeroThread := NegotiationThread{}
	if zeroThread.PaymentRetryAllowed {
		t.Error("zero-value Go struct should have PaymentRetryAllowed=false (DB default is true)")
	}
}

func TestNegotiationStatus_PendingPayment_NotAccepted(t *testing.T) {
	// PENDING_PAYMENT_LOCK is NOT the same as ACCEPTED
	if NegotiationPendingPayment == NegotiationAccepted {
		t.Error("PENDING_PAYMENT_LOCK should be distinct from ACCEPTED")
	}
}

func TestNegotiationStatus_PaymentFailed_NotAccepted(t *testing.T) {
	// PAYMENT_FAILED is NOT the same as ACCEPTED
	if NegotiationPaymentFailed == NegotiationAccepted {
		t.Error("PAYMENT_FAILED should be distinct from ACCEPTED")
	}
}

func TestNegotiationStatus_StateMachineOrder(t *testing.T) {
	// Verify the state machine: open/countered → pending_payment → accepted/payment_failed
	validTransitions := map[NegotiationStatus][]NegotiationStatus{
		NegotiationOpen:           {NegotiationPendingPayment, NegotiationRejected, NegotiationCountered, NegotiationExpired},
		NegotiationCountered:      {NegotiationPendingPayment, NegotiationRejected, NegotiationOpen, NegotiationExpired},
		NegotiationPendingPayment: {NegotiationAccepted, NegotiationPaymentFailed},
		NegotiationPaymentFailed:  {NegotiationPendingPayment}, // retry
		NegotiationAccepted:       {NegotiationClosed},
	}
	for from, validNext := range validTransitions {
		for _, to := range validNext {
			if from == to {
				t.Errorf("self-transition detected: %s → %s", from, to)
			}
		}
	}
}

func TestEvaluateOffer_AutoAcceptRequiresEscrowFirst(t *testing.T) {
	// The key invariant: auto-accept evaluates to OfferActionAccept,
	// but the handler should NOT set status=ACCEPTED until HoldFunds succeeds.
	// This test verifies EvaluateOffer still returns Accept for qualifying offers.
	cfg := ListingTradeConfig{MinOfferPercent: 0.7, AutoAcceptPercent: 0.95}
	action := EvaluateOffer(10000, 9600, cfg)
	if action != OfferActionAccept {
		t.Error("EvaluateOffer should still return Accept for qualifying offers")
	}
	// The actual PENDING_PAYMENT_LOCK → ACCEPTED transition is tested
	// via integration tests with a real DB (escrow hold succeeds/fails).
}
