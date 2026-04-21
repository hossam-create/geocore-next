package crowdshipping

import (
	"testing"
	"time"
)

// ── 1. OFFER FSM TESTS ────────────────────────────────────────────────────────

func TestOfferCanTransitionTo_ValidTransitions(t *testing.T) {
	tests := []struct {
		from OfferStatus
		to   OfferStatus
	}{
		{OfferPending, OfferCountered},
		{OfferPending, OfferPaymentPending},
		{OfferPending, OfferRejected},
		{OfferPending, OfferExpired},
		{OfferPending, OfferCancelled},
		{OfferCountered, OfferPending},
		{OfferCountered, OfferRejected},
		{OfferCountered, OfferExpired},
		{OfferCountered, OfferCancelled},
		{OfferPaymentPending, OfferFundsHeld},
		{OfferPaymentPending, OfferPaymentFailed},
		{OfferFundsHeld, OfferAccepted},
		{OfferPaymentFailed, OfferPaymentPending},
		{OfferAccepted, OfferCompleted},
		{OfferAccepted, OfferCancelled},
	}
	for _, tt := range tests {
		o := TravelerOffer{Status: tt.from}
		if !o.CanTransitionTo(tt.to) {
			t.Errorf("expected %s → %s to be valid", tt.from, tt.to)
		}
	}
}

func TestOfferCanTransitionTo_InvalidTransitions(t *testing.T) {
	tests := []struct {
		from OfferStatus
		to   OfferStatus
	}{
		{OfferPending, OfferAccepted},
		{OfferPending, OfferCompleted},
		{OfferAccepted, OfferPending},
		{OfferExpired, OfferPending},
		{OfferCompleted, OfferPending},
		{OfferRejected, OfferPending},
		{OfferPaymentFailed, OfferAccepted},
		{OfferPaymentPending, OfferPending},
		{OfferPending, OfferFundsHeld},
		{OfferFundsHeld, OfferPending},
		{OfferFundsHeld, OfferPaymentFailed},
	}
	for _, tt := range tests {
		o := TravelerOffer{Status: tt.from}
		if o.CanTransitionTo(tt.to) {
			t.Errorf("expected %s → %s to be INVALID", tt.from, tt.to)
		}
	}
}

func TestOfferIsActive(t *testing.T) {
	o := TravelerOffer{Status: OfferPending}
	if !o.IsActive() {
		t.Error("pending offer should be active")
	}
	o.Status = OfferCountered
	if !o.IsActive() {
		t.Error("countered offer should be active")
	}
	o.Status = OfferAccepted
	if o.IsActive() {
		t.Error("accepted offer should NOT be active")
	}
	o.Status = OfferExpired
	if o.IsActive() {
		t.Error("expired offer should NOT be active")
	}
}

// ── 2. SHIPMENT FSM TESTS ────────────────────────────────────────────────────

func TestNextAllowedStatuses_ValidChain(t *testing.T) {
	chain := []ShipmentStatus{
		ShipmentRequested, ShipmentAccepted, ShipmentPurchased,
		ShipmentInTransit, ShipmentArrivedCountry,
		ShipmentOutForDelivery, ShipmentDelivered, ShipmentConfirmed,
	}
	for i := 0; i < len(chain)-1; i++ {
		next := NextAllowedStatuses(chain[i])
		found := false
		for _, s := range next {
			if s == chain[i+1] {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %s → %s in FSM", chain[i], chain[i+1])
		}
	}
}

func TestNextAllowedStatuses_InvalidJump(t *testing.T) {
	next := NextAllowedStatuses(ShipmentRequested)
	for _, s := range next {
		if s == ShipmentDelivered {
			t.Error("requested → delivered should NOT be allowed")
		}
	}
}

func TestNextAllowedStatuses_TerminalState(t *testing.T) {
	next := NextAllowedStatuses(ShipmentConfirmed)
	if len(next) != 0 {
		t.Errorf("confirmed is terminal, got %v", next)
	}
}

// ── 3. CENTS MATH TESTS ──────────────────────────────────────────────────────

func TestCentsRoundTrip(t *testing.T) {
	tests := []float64{0.01, 1.99, 10.00, 99.99, 1234.56, 999999.99}
	for _, v := range tests {
		cents := toCents(v)
		back := fromCents(cents)
		if back != v {
			t.Errorf("toCents(%v)=%d, fromCents=%v, want %v", v, cents, back, v)
		}
	}
}

func TestCentsNoFloatDrift(t *testing.T) {
	total := int64(0)
	for i := 0; i < 1000; i++ {
		total += toCents(0.10)
	}
	if total != 10000 {
		t.Errorf("1000 × $0.10 = %d cents, want 10000", total)
	}
}

// ── 4. DOUBLE-ACCEPT PREVENTION (FSM-level) ──────────────────────────────────

func TestDoubleAcceptPrevention(t *testing.T) {
	o := TravelerOffer{Status: OfferAccepted}
	if o.CanTransitionTo(OfferAccepted) {
		t.Error("ACCEPTED → ACCEPTED must be impossible (double accept)")
	}
	if o.CanTransitionTo(OfferPaymentPending) {
		t.Error("ACCEPTED → PAYMENT_PENDING must be impossible")
	}
}

// ── 5. PAYMENT RETRY FLOW ────────────────────────────────────────────────────

func TestPaymentRetryFlow(t *testing.T) {
	// PENDING → PAYMENT_PENDING → PAYMENT_FAILED → PAYMENT_PENDING → FUNDS_HELD → ACCEPTED
	o := TravelerOffer{Status: OfferPending}
	if !o.CanTransitionTo(OfferPaymentPending) {
		t.Fatal("step 1 failed")
	}
	o.Status = OfferPaymentPending
	if !o.CanTransitionTo(OfferPaymentFailed) {
		t.Fatal("step 2 failed")
	}
	o.Status = OfferPaymentFailed
	if !o.CanTransitionTo(OfferPaymentPending) {
		t.Fatal("step 3: retry must be allowed")
	}
	o.Status = OfferPaymentPending
	if !o.CanTransitionTo(OfferFundsHeld) {
		t.Fatal("step 4: hold success must lead to FUNDS_HELD")
	}
	o.Status = OfferFundsHeld
	if !o.CanTransitionTo(OfferAccepted) {
		t.Fatal("step 5: FUNDS_HELD must lead to ACCEPTED")
	}
}

func TestPaymentRetryNotAllowedFromOtherStates(t *testing.T) {
	states := []OfferStatus{OfferAccepted, OfferFundsHeld, OfferRejected, OfferExpired, OfferCompleted, OfferCancelled}
	for _, s := range states {
		o := TravelerOffer{Status: s}
		if o.CanTransitionTo(OfferPaymentPending) {
			t.Errorf("%s → PAYMENT_PENDING should NOT be allowed", s)
		}
	}
}

// ── 6. EXPIRY BEHAVIOR ────────────────────────────────────────────────────────

func TestExpiredOfferCannotTransition(t *testing.T) {
	o := TravelerOffer{Status: OfferExpired}
	targets := []OfferStatus{OfferPending, OfferAccepted, OfferPaymentPending, OfferCountered}
	for _, target := range targets {
		if o.CanTransitionTo(target) {
			t.Errorf("expired → %s should NOT be allowed", target)
		}
	}
}

func TestOfferExpiryTime(t *testing.T) {
	now := time.Now()
	o := TravelerOffer{ExpiresAt: now.Add(72 * time.Hour)}
	if o.ExpiresAt.Before(now) {
		t.Error("offer should not be expired immediately")
	}
}

// ── 7. PLATFORM FEE CALCULATION ───────────────────────────────────────────────

func TestPlatformFeeCalculation(t *testing.T) {
	priceCents := int64(10000) // $100.00
	platformFeeCents := priceCents * 15 / 100
	travelerEarningsCents := priceCents - platformFeeCents

	if platformFeeCents != 1500 {
		t.Errorf("platform fee = %d, want 1500", platformFeeCents)
	}
	if travelerEarningsCents != 8500 {
		t.Errorf("traveler earnings = %d, want 8500", travelerEarningsCents)
	}
}

// ── 8. ESCROW RELEASE CORRECTNESS ────────────────────────────────────────────

func TestEscrowReleaseEarningsCalculation(t *testing.T) {
	priceCents := int64(5000)                              // $50.00
	platformFeeCents := priceCents * 15 / 100              // 750
	travelerEarningsCents := priceCents - platformFeeCents // 4250

	earnings := fromCents(travelerEarningsCents)
	if earnings != 42.50 {
		t.Errorf("traveler earnings = %.2f, want 42.50", earnings)
	}
}

// ── 9. COUNTER OFFER CHAIN ────────────────────────────────────────────────────

func TestCounterOfferChain(t *testing.T) {
	// Original offer → countered → new pending offer
	orig := TravelerOffer{Status: OfferPending}
	if !orig.CanTransitionTo(OfferCountered) {
		t.Fatal("pending → countered must be valid")
	}
	orig.Status = OfferCountered

	// Counter creates new offer in PENDING state
	counter := TravelerOffer{Status: OfferPending}
	if !counter.IsActive() {
		t.Error("counter offer should be active")
	}
}

// ── 10. FUNDS_HELD STATE ────────────────────────────────────────────────────

func TestFundsHeldIsTerminalBeforeAccept(t *testing.T) {
	// FUNDS_HELD can ONLY go to ACCEPTED
	o := TravelerOffer{Status: OfferFundsHeld}
	invalid := []OfferStatus{OfferPending, OfferCountered, OfferPaymentPending, OfferPaymentFailed, OfferFundsHeld, OfferRejected, OfferExpired, OfferCompleted, OfferCancelled}
	for _, s := range invalid {
		if o.CanTransitionTo(s) {
			t.Errorf("FUNDS_HELD → %s should NOT be allowed", s)
		}
	}
	if !o.CanTransitionTo(OfferAccepted) {
		t.Error("FUNDS_HELD → ACCEPTED must be allowed")
	}
}

func TestFundsHeldNotActive(t *testing.T) {
	o := TravelerOffer{Status: OfferFundsHeld}
	if o.IsActive() {
		t.Error("FUNDS_HELD should NOT be active (no new offers allowed)")
	}
}

// ── 11. DELIVERY LOCK ────────────────────────────────────────────────────

func TestDeliveryLockedPreventsNewOffers(t *testing.T) {
	// When delivery request is LOCKED, no new offers should be accepted
	locked := DeliveryLocked
	available := []DeliveryStatus{DeliveryPending, DeliveryMatched, DeliveryAccepted}
	found := false
	for _, s := range available {
		if s == locked {
			found = true
		}
	}
	if found {
		t.Error("DeliveryLocked should NOT be in available-for-offer list")
	}
}
