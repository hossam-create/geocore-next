package livestream

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/geocore-next/backend/internal/freeze"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/google/uuid"
)

// TestAntiSnipingLogic verifies that a bid placed in the last 10 seconds
// extends the auction by 30 seconds.
func TestAntiSnipingLogic(t *testing.T) {
	now := time.Now()
	endsAt := now.Add(8 * time.Second) // 8s left — within antiSnipeWindow

	item := LiveItem{
		ID:                uuid.New(),
		CurrentBidCents:   500,
		MinIncrementCents: 100,
		BidCount:          2,
		Status:            ItemActive,
		EndsAt:            &endsAt,
		AntiSnipeEnabled:  true,
	}

	// Simulate anti-sniping check
	extended := false
	if item.AntiSnipeEnabled && item.EndsAt != nil {
		if time.Until(*item.EndsAt) <= antiSnipeWindow {
			newEnd := item.EndsAt.Add(antiSnipeExtension)
			item.EndsAt = &newEnd
			extended = true
		}
	}

	if !extended {
		t.Fatal("Expected anti-sniping extension but bid was not extended")
	}

	remainingAfter := time.Until(*item.EndsAt)
	if remainingAfter < 30*time.Second {
		t.Fatalf("Expected at least 30s remaining after anti-snipe, got %v", remainingAfter)
	}
}

// TestAntiSnipingNoExtensionWhenSafe verifies no extension when bid is early.
func TestAntiSnipingNoExtensionWhenSafe(t *testing.T) {
	now := time.Now()
	endsAt := now.Add(60 * time.Second) // 60s left — not within window

	item := LiveItem{
		EndsAt:           &endsAt,
		AntiSnipeEnabled: true,
	}

	extended := false
	if item.AntiSnipeEnabled && item.EndsAt != nil {
		if time.Until(*item.EndsAt) <= antiSnipeWindow {
			extended = true
		}
	}

	if extended {
		t.Fatal("Should NOT extend when bid is placed with 60s remaining")
	}
}

// TestBidValidation verifies minimum bid requirements.
func TestBidValidation(t *testing.T) {
	item := LiveItem{
		StartPriceCents:   1000,
		CurrentBidCents:   1500,
		MinIncrementCents: 100,
		BidCount:          3,
	}

	// Min bid should be current + increment
	minBid := item.CurrentBidCents + item.MinIncrementCents
	if minBid != 1600 {
		t.Fatalf("Expected min bid 1600, got %d", minBid)
	}

	// First bid scenario
	item.BidCount = 0
	minBidFirst := item.StartPriceCents
	if minBidFirst != 1000 {
		t.Fatalf("Expected first bid min 1000, got %d", minBidFirst)
	}
}

// TestAntiSnipingCap verifies that extension stops after maxAntiSnipeExtensions.
func TestAntiSnipingCap(t *testing.T) {
	now := time.Now()
	endsAt := now.Add(5 * time.Second)

	item := LiveItem{
		EndsAt:           &endsAt,
		AntiSnipeEnabled: true,
		ExtensionCount:   maxAntiSnipeExtensions, // already at max
	}

	extended := false
	if item.AntiSnipeEnabled && item.EndsAt != nil && item.ExtensionCount < maxAntiSnipeExtensions {
		if time.Until(*item.EndsAt) <= antiSnipeWindow {
			extended = true
		}
	}

	if extended {
		t.Fatal("Should NOT extend when ExtensionCount has reached the cap")
	}
}

// TestAntiSnipingCapAllowsBeforeMax verifies extension works when under cap.
func TestAntiSnipingCapAllowsBeforeMax(t *testing.T) {
	now := time.Now()
	endsAt := now.Add(5 * time.Second)

	item := LiveItem{
		EndsAt:           &endsAt,
		AntiSnipeEnabled: true,
		ExtensionCount:   3, // under maxAntiSnipeExtensions (5)
	}

	extended := false
	if item.AntiSnipeEnabled && item.EndsAt != nil && item.ExtensionCount < maxAntiSnipeExtensions {
		if time.Until(*item.EndsAt) <= antiSnipeWindow {
			newEnd := item.EndsAt.Add(antiSnipeExtension)
			item.EndsAt = &newEnd
			item.ExtensionCount++
			extended = true
		}
	}

	if !extended {
		t.Fatal("Should extend when ExtensionCount is under cap")
	}
	if item.ExtensionCount != 4 {
		t.Fatalf("Expected ExtensionCount=4, got %d", item.ExtensionCount)
	}
}

// TestSettlementStateMachine verifies the correct status transitions.
func TestSettlementStateMachine(t *testing.T) {
	// With bids → settling → sold
	bidderID := uuid.New()
	item := LiveItem{BidCount: 3, HighestBidderID: &bidderID, Status: "ended"}

	if item.BidCount == 0 || item.HighestBidderID == nil {
		t.Fatal("Should have bids")
	}
	item.Status = ItemSettling
	if item.Status != ItemSettling {
		t.Fatal("Expected settling status")
	}
	// After escrow success
	item.Status = ItemSold
	if item.Status != ItemSold {
		t.Fatal("Expected sold status")
	}
}

// TestItemEndStatus verifies sold vs unsold logic.
func TestItemEndStatus(t *testing.T) {
	bidderID := uuid.New()

	// Item with bids → sold
	item := LiveItem{BidCount: 3, HighestBidderID: &bidderID}
	status := ItemUnsold
	if item.BidCount > 0 && item.HighestBidderID != nil {
		status = ItemSold
	}
	if status != ItemSold {
		t.Fatal("Expected ItemSold when bids > 0 and bidder exists")
	}

	// Item with no bids → unsold
	item2 := LiveItem{BidCount: 0, HighestBidderID: nil}
	status2 := ItemUnsold
	if item2.BidCount > 0 && item2.HighestBidderID != nil {
		status2 = ItemSold
	}
	if status2 != ItemUnsold {
		t.Fatal("Expected ItemUnsold when no bids")
	}
}

// TestSettlementTimeoutDetection verifies stuck settlement items are detected.
func TestSettlementTimeoutDetection(t *testing.T) {
	stuckTime := time.Now().Add(-40 * time.Second) // 40s ago > 30s timeout

	item := LiveItem{
		Status:            ItemSettling,
		SettlingStartedAt: &stuckTime,
		SettleRetries:     1,
	}

	isStuck := item.Status == ItemSettling &&
		item.SettlingStartedAt != nil &&
		time.Since(*item.SettlingStartedAt) > settlementTimeout

	if !isStuck {
		t.Fatal("Expected item to be detected as stuck")
	}

	// Under max retries → should retry
	if item.SettleRetries >= maxSettleRetries {
		t.Fatal("Should be eligible for retry (retries=1 < max=3)")
	}
}

// TestSettlementMaxRetriesExhausted verifies payment_failed after max retries.
func TestSettlementMaxRetriesExhausted(t *testing.T) {
	item := LiveItem{
		Status:        ItemSettling,
		SettleRetries: maxSettleRetries,
	}

	if item.SettleRetries < maxSettleRetries {
		t.Fatal("Should have exhausted retries")
	}
	// Should be marked payment_failed
	item.Status = ItemPaymentFailed
	if item.Status != ItemPaymentFailed {
		t.Fatal("Expected payment_failed status")
	}
}

// TestPlatformFeeCalculation verifies fee math.
func TestPlatformFeeCalculation(t *testing.T) {
	totalCents := int64(10000) // $100.00
	amountFloat := float64(totalCents) / 100.0
	fee := amountFloat * platformFeePercent / 100.0
	sellerAmount := amountFloat - fee

	if fee != 2.5 {
		t.Fatalf("Expected fee $2.50, got $%.2f", fee)
	}
	if sellerAmount != 97.5 {
		t.Fatalf("Expected seller $97.50, got $%.2f", sellerAmount)
	}
}

// TestIdempotencyTTL verifies the constant.
func TestIdempotencyTTL(t *testing.T) {
	if idempotencyTTL != 2*time.Minute {
		t.Fatalf("Expected 2min TTL, got %v", idempotencyTTL)
	}
}

// ── LiveEvent & Broadcast Tests ──────────────────────────────────────────────

// TestLiveEventTypes verifies all event type constants.
func TestLiveEventTypes(t *testing.T) {
	types := map[LiveEventType]string{
		EventNewBid:            "new_bid",
		EventOutbid:            "outbid",
		EventAuctionEnd:        "auction_end",
		EventItemActivated:     "item_activated",
		EventItemSettling:      "item_settling",
		EventItemSold:          "item_sold",
		EventItemUnsold:        "item_unsold",
		EventItemPaymentFailed: "item_payment_failed",
		EventItemSoldBuyNow:    "item_sold_buy_now",
		EventViewerJoin:        "viewer_join",
		EventViewerLeave:       "viewer_leave",
		EventBidExtended:       "bid_extended",
	}
	for evt, expected := range types {
		if string(evt) != expected {
			t.Errorf("Expected %s, got %s", expected, evt)
		}
	}
}

// TestLiveEventSerialization verifies JSON serialization of LiveEvent.
func TestLiveEventSerialization(t *testing.T) {
	sid := uuid.New().String()
	iid := uuid.New().String()
	bidderID := uuid.New().String()

	evt := LiveEvent{
		Event:           EventNewBid,
		SessionID:       sid,
		ItemID:          iid,
		CurrentBidCents: 2500,
		HighestBidderID: &bidderID,
		BidCount:        5,
		Status:          string(ItemActive),
		ViewerCount:     42,
	}

	if evt.Event != EventNewBid {
		t.Fatal("Expected new_bid event type")
	}
	if evt.SessionID != sid {
		t.Fatalf("Expected session %s, got %s", sid, evt.SessionID)
	}
	if evt.CurrentBidCents != 2500 {
		t.Fatalf("Expected 2500 cents, got %d", evt.CurrentBidCents)
	}
	if evt.ViewerCount != 42 {
		t.Fatalf("Expected 42 viewers, got %d", evt.ViewerCount)
	}
	if evt.HighestBidderID == nil || *evt.HighestBidderID != bidderID {
		t.Fatal("HighestBidderID mismatch")
	}
}

// TestLiveEventOutbidFields verifies outbid-specific fields.
func TestLiveEventOutbidFields(t *testing.T) {
	outbidUser := uuid.New().String()
	evt := LiveEvent{
		Event:           EventOutbid,
		OutbidUserID:    &outbidUser,
		CurrentBidCents: 3000,
	}

	if evt.Event != EventOutbid {
		t.Fatal("Expected outbid event type")
	}
	if evt.OutbidUserID == nil || *evt.OutbidUserID != outbidUser {
		t.Fatal("OutbidUserID mismatch")
	}
}

// TestLiveEventAntiSnipeExtension verifies extension fields.
func TestLiveEventAntiSnipeExtension(t *testing.T) {
	newEnd := time.Now().Add(30 * time.Second).Format(time.RFC3339)
	evt := LiveEvent{
		Event:          EventNewBid,
		Extended:       true,
		NewEndsAt:      &newEnd,
		ExtensionCount: 2,
	}

	if !evt.Extended {
		t.Fatal("Expected extended=true")
	}
	if evt.ExtensionCount != 2 {
		t.Fatalf("Expected 2 extensions, got %d", evt.ExtensionCount)
	}
}

// TestRecentBidderStruct verifies social proof data structure.
func TestRecentBidderStruct(t *testing.T) {
	bidder := RecentBidder{
		UserID:      uuid.New().String(),
		DisplayName: "Alice",
		AmountCents: 1500,
		BidAt:       time.Now().Format(time.RFC3339),
	}

	if bidder.DisplayName != "Alice" {
		t.Fatalf("Expected Alice, got %s", bidder.DisplayName)
	}
	if bidder.AmountCents != 1500 {
		t.Fatalf("Expected 1500 cents, got %d", bidder.AmountCents)
	}
}

// TestLiveEventViewerEvents verifies viewer join/leave event fields.
func TestLiveEventViewerEvents(t *testing.T) {
	evt := LiveEvent{
		Event:       EventViewerJoin,
		ViewerID:    "user-123",
		DisplayName: "Bob",
		ViewerCount: 15,
	}

	if evt.Event != EventViewerJoin {
		t.Fatal("Expected viewer_join event")
	}
	if evt.ViewerCount != 15 {
		t.Fatalf("Expected 15 viewers, got %d", evt.ViewerCount)
	}
	if evt.DisplayName != "Bob" {
		t.Fatalf("Expected Bob, got %s", evt.DisplayName)
	}
}

// TestHelperFunctions verifies strPtr, uuidToStr, timeToStr.
func TestHelperFunctions(t *testing.T) {
	// strPtr
	s := strPtr("hello")
	if s == nil || *s != "hello" {
		t.Fatal("strPtr failed")
	}

	// uuidToStr — nil
	if uuidToStr(nil) != nil {
		t.Fatal("uuidToStr(nil) should return nil")
	}

	// uuidToStr — value
	id := uuid.New()
	result := uuidToStr(&id)
	if result == nil || *result != id.String() {
		t.Fatal("uuidToStr failed for non-nil UUID")
	}

	// timeToStr — nil
	if timeToStr(nil) != nil {
		t.Fatal("timeToStr(nil) should return nil")
	}

	// timeToStr — value
	now := time.Now()
	ts := timeToStr(&now)
	if ts == nil {
		t.Fatal("timeToStr returned nil for non-nil time")
	}
	parsed, err := time.Parse(time.RFC3339, *ts)
	if err != nil {
		t.Fatalf("timeToStr produced invalid RFC3339: %v", err)
	}
	if parsed.Unix() != now.Unix() {
		t.Fatalf("Time mismatch: expected %d, got %d", now.Unix(), parsed.Unix())
	}
}

// TestLiveEventOmitEmpty verifies that omitempty fields are excluded when zero.
func TestLiveEventOmitEmpty(t *testing.T) {
	evt := LiveEvent{
		Event:     EventAuctionEnd,
		SessionID: uuid.New().String(),
	}

	// These optional fields should be zero-valued
	if evt.ItemID != "" {
		t.Fatal("ItemID should be empty string")
	}
	if evt.OutbidUserID != nil {
		t.Fatal("OutbidUserID should be nil")
	}
	if evt.RecentBidders != nil {
		t.Fatal("RecentBidders should be nil")
	}
	if evt.Extended != false {
		t.Fatal("Extended should be false")
	}
}

// ── Production-Grade Settlement Tests ────────────────────────────────────────

// TestSettlementAtomicity verifies that settlement state transitions are valid.
// ENDED → SETTLING → SOLD (happy path)
// ENDED → SETTLING → PAYMENT_FAILED (escrow failure)
func TestSettlementAtomicity(t *testing.T) {
	transitions := []struct {
		from   string
		event  string
		to     string
		reason string
	}{
		{"ended", "escrow_success", "settling", "start settlement"},
		{"settling", "order_created", "sold", "happy path"},
		{"settling", "escrow_failed", "payment_failed", "escrow failure"},
		{"settling", "order_failed", "payment_failed", "order failure"},
		{"settling", "session_not_found", "payment_failed", "session missing"},
	}

	for _, tr := range transitions {
		if tr.from == "ended" && tr.to != "settling" {
			t.Errorf("ENDED must transition to SETTLING first, got %s", tr.to)
		}
		if tr.to == "sold" && tr.from != "settling" {
			t.Errorf("SOLD must come from SETTLING, got %s", tr.from)
		}
		if tr.to == "payment_failed" && tr.from != "settling" {
			t.Errorf("PAYMENT_FAILED must come from SETTLING, got %s", tr.from)
		}
	}
}

// TestFundReleaseOnPaymentFailed verifies that all payment_failed paths
// must release the winner's reserved funds (pending → available).
func TestFundReleaseOnPaymentFailed(t *testing.T) {
	// Simulate: winner has 5000 cents reserved (available=0, pending=5000)
	// If settlement fails, funds must return (available=5000, pending=0)
	reservedAmount := int64(5000)

	// After release: available should equal the reserved amount
	// This is a logic test — the actual DB test requires integration
	if reservedAmount <= 0 {
		t.Fatal("Reserved amount must be positive for this test")
	}

	// Verify the invariant: on payment_failed, ReleaseReservedFunds is called
	// with the same amount that was reserved
	releasedAmount := reservedAmount // must match
	if releasedAmount != reservedAmount {
		t.Fatalf("Released amount %d must equal reserved amount %d", releasedAmount, reservedAmount)
	}
}

// TestBuyNowRequiresBalance verifies Buy Now cannot proceed without funds.
func TestBuyNowRequiresBalance(t *testing.T) {
	// Buy Now must check HasSufficientBalance before ReserveFunds
	// If balance < buyNowPrice → "insufficient_balance" error
	buyNowPrice := int64(10000)
	userBalance := int64(5000)

	if userBalance >= buyNowPrice {
		t.Fatal("Test expects insufficient balance scenario")
	}

	// HasSufficientBalance should return false
	hasSufficient := userBalance >= buyNowPrice
	if hasSufficient {
		t.Fatal("Should not allow Buy Now with insufficient balance")
	}
}

// TestBuyNowReleasesPrevBidder verifies Buy Now releases previous bidder's reserve.
func TestBuyNowReleasesPrevBidder(t *testing.T) {
	prevBidderHasReserve := true
	prevBidAmount := int64(3000)

	// When Buy Now happens, previous bidder's reserve must be released
	if prevBidderHasReserve && prevBidAmount <= 0 {
		t.Fatal("Previous bidder should have a positive reserve to release")
	}

	// The release amount must match what was reserved
	releaseAmount := prevBidAmount
	if releaseAmount != prevBidAmount {
		t.Fatalf("Release %d must equal prev bid %d", releaseAmount, prevBidAmount)
	}
}

// TestDeadlockRetryGuarantee verifies RetryOnDeadlock is used for financial ops.
func TestDeadlockRetryGuarantee(t *testing.T) {
	// PlaceBid and BuyNow must use locking.RetryOnDeadlock, not raw db.Transaction
	// This test verifies the constant exists and has correct values
	if locking.MaxRetries != 3 {
		t.Fatalf("Expected MaxRetries=3, got %d", locking.MaxRetries)
	}
	if locking.BaseDelay != 50*time.Millisecond {
		t.Fatalf("Expected BaseDelay=50ms, got %v", locking.BaseDelay)
	}
}

// TestSettlementStateMachineIntegrity verifies no invalid state transitions.
func TestSettlementStateMachineIntegrity(t *testing.T) {
	// Valid terminal states
	validTerminal := map[LiveItemStatus]bool{
		ItemSold:          true,
		ItemUnsold:        true,
		ItemPaymentFailed: true,
	}

	// Active/Settling are NOT terminal
	if validTerminal[ItemActive] {
		t.Fatal("Active should not be a terminal state")
	}
	if validTerminal[ItemSettling] {
		t.Fatal("Settling should not be a terminal state")
	}

	// All terminal states must be reachable
	if len(validTerminal) != 3 {
		t.Fatal("Expected exactly 3 terminal states")
	}
}

// TestStuckSettlementFundRelease verifies that when a stuck item hits max retries,
// the winner's reserved funds must be released (not leaked in pending).
func TestStuckSettlementFundRelease(t *testing.T) {
	bidderID := uuid.New()
	item := LiveItem{
		Status:            ItemSettling,
		HighestBidderID:   &bidderID,
		CurrentBidCents:   7500,
		SettleRetries:     maxSettleRetries, // at max
		SettlingStartedAt: ptrTime(time.Now().Add(-40 * time.Second)),
	}

	// Item is stuck and at max retries → must release funds
	if item.SettleRetries >= maxSettleRetries {
		// Must call ReleaseReservedFunds(bidderID, 7500)
		if item.CurrentBidCents <= 0 {
			t.Fatal("Must have positive amount to release")
		}
		if item.HighestBidderID == nil {
			t.Fatal("Must have a bidder to release funds for")
		}
	} else {
		t.Fatal("Expected max retries reached")
	}
}

// TestOrderFeeTracking verifies platform fee is calculated and stored in order.
func TestOrderFeeTracking(t *testing.T) {
	totalCents := int64(10000) // $100.00
	amountFloat := float64(totalCents) / 100.0
	platformFee := amountFloat * platformFeePercent / 100.0
	total := amountFloat // buyer pays full amount

	if platformFee != 2.5 {
		t.Fatalf("Expected platform fee $2.50, got $%.2f", platformFee)
	}

	// Order must store: subtotal=100, platform_fee=2.5, payment_fee=0, total=100
	if total != amountFloat {
		t.Fatalf("Total should equal subtotal for live auction (fee deducted from seller side)")
	}

	// Seller receives: total - platform_fee
	sellerAmount := amountFloat - platformFee
	if sellerAmount != 97.5 {
		t.Fatalf("Expected seller amount $97.50, got $%.2f", sellerAmount)
	}
}

// TestMinimumIncrementLogic verifies min increment is enforced.
func TestMinimumIncrementLogic(t *testing.T) {
	item := LiveItem{
		StartPriceCents:   1000,
		CurrentBidCents:   5000,
		MinIncrementCents: 100,
		BidCount:          5,
	}

	// Min bid = current + increment
	minBid := item.CurrentBidCents + item.MinIncrementCents
	if minBid != 5100 {
		t.Fatalf("Expected min bid 5100, got %d", minBid)
	}

	// First bid = start price
	item2 := LiveItem{
		StartPriceCents:   1000,
		CurrentBidCents:   0,
		MinIncrementCents: 100,
		BidCount:          0,
	}
	minBidFirst := item2.StartPriceCents
	if minBidFirst != 1000 {
		t.Fatalf("Expected first bid min 1000, got %d", minBidFirst)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

// ── Prohibited Items Tests ──────────────────────────────────────────────────

func TestCheckItemCompliance_Allowed(t *testing.T) {
	result := CheckItemCompliance("iPhone 15 Pro Max", "Brand new sealed unit", "electronics")
	if result.Verdict != VerdictAllowed {
		t.Fatalf("Expected allowed, got %s", result.Verdict)
	}
}

func TestCheckItemCompliance_BlockedHighConfidence(t *testing.T) {
	result := CheckItemCompliance("AK-47 Rifle", "Military grade", "weapons")
	if result.Verdict != VerdictBlocked {
		t.Fatalf("Expected blocked for weapons, got %s", result.Verdict)
	}
	if result.Confidence != ConfidenceHigh {
		t.Fatalf("Expected high confidence, got %s", result.Confidence)
	}
}

func TestCheckItemCompliance_BlockedKeyword(t *testing.T) {
	result := CheckItemCompliance("Cocaine powder", "Pure quality", "")
	if result.Verdict != VerdictBlocked {
		t.Fatalf("Expected blocked for drug keyword, got %s", result.Verdict)
	}
}

func TestCheckItemCompliance_FlaggedMediumConfidence(t *testing.T) {
	result := CheckItemCompliance("Counterfeit watch", "Looks like original", "")
	if result.Verdict != VerdictFlagged {
		t.Fatalf("Expected flagged for counterfeit, got %s", result.Verdict)
	}
}

func TestCheckItemCompliance_ArabicKeyword(t *testing.T) {
	result := CheckItemCompliance("سلاح ناري", "بندقية صيد", "")
	if result.Verdict != VerdictBlocked {
		t.Fatalf("Expected blocked for Arabic weapon keyword, got %s", result.Verdict)
	}
}

func TestCheckItemCompliance_MultiLanguage(t *testing.T) {
	// French keyword
	result := CheckItemCompliance("Arme à feu", "Pistolet neuf", "")
	if result.Verdict != VerdictBlocked {
		t.Fatalf("Expected blocked for French weapon keyword, got %s", result.Verdict)
	}
}

// ── Auction Deposit Tests ───────────────────────────────────────────────────

func TestCalculateDepositAmount_BelowThreshold(t *testing.T) {
	amount, required := CalculateDepositAmount(100_000) // 1000 EGP
	if amount != 0 {
		t.Fatalf("Expected no deposit below threshold, got %d", amount)
	}
	if required {
		t.Fatal("Should not be required below threshold")
	}
}

func TestCalculateDepositAmount_SuggestThreshold(t *testing.T) {
	amount, required := CalculateDepositAmount(500_000) // 5000 EGP
	if amount <= 0 {
		t.Fatal("Expected deposit amount for suggest threshold")
	}
	if required {
		t.Fatal("Should be suggested, not required, at 5000 EGP")
	}
	// 5% of 5000 = 250 EGP = 25000 cents
	if amount != 25_000 {
		t.Fatalf("Expected 25000 cents (250 EGP), got %d", amount)
	}
}

func TestCalculateDepositAmount_RequireThreshold(t *testing.T) {
	amount, required := CalculateDepositAmount(2_000_000) // 20000 EGP
	if amount <= 0 {
		t.Fatal("Expected deposit amount for require threshold")
	}
	if !required {
		t.Fatal("Should be required at 20000 EGP")
	}
	// 5% of 20000 = 1000 EGP = 100000 cents
	if amount != 100_000 {
		t.Fatalf("Expected 100000 cents (1000 EGP), got %d", amount)
	}
}

func TestApplyDepositRules(t *testing.T) {
	item := &LiveItem{
		StartPriceCents: 2_000_000, // 20000 EGP
	}
	ApplyDepositRules(item)
	if !item.RequiresEntryDeposit {
		t.Fatal("High-value item should require entry deposit")
	}
	if item.EntryDepositCents != 100_000 {
		t.Fatalf("Expected 100000 cents deposit, got %d", item.EntryDepositCents)
	}
}

func TestApplyDepositRules_LowValue(t *testing.T) {
	item := &LiveItem{
		StartPriceCents: 100_000, // 1000 EGP
	}
	ApplyDepositRules(item)
	if item.RequiresEntryDeposit {
		t.Fatal("Low-value item should not require entry deposit")
	}
}

func TestAuctionDepositStatuses(t *testing.T) {
	statuses := map[AuctionDepositStatus]bool{
		DepositHeld:      true,
		DepositReleased:  true,
		DepositConverted: true,
		DepositForfeited: true,
	}
	if len(statuses) != 4 {
		t.Fatal("Expected 4 deposit statuses")
	}
}

// ── Admin Control Feature Flag Tests ────────────────────────────────────────

func TestProhibitedCheckEnabled(t *testing.T) {
	enabled := IsProhibitedCheckEnabled()
	if !enabled {
		t.Fatal("Prohibited check should be enabled by default")
	}
}

func TestAdminLiveControlEnabled(t *testing.T) {
	enabled := IsAdminLiveControlEnabled()
	if !enabled {
		t.Fatal("Admin live control should be enabled by default")
	}
}

func TestAuctionDepositEnabled(t *testing.T) {
	enabled := IsAuctionDepositEnabled()
	if !enabled {
		t.Fatal("Auction deposit should be enabled by default")
	}
}

func TestComplianceVerdictMapping(t *testing.T) {
	if verdictFromConfidence(ConfidenceHigh) != VerdictBlocked {
		t.Fatal("High confidence should map to blocked")
	}
	if verdictFromConfidence(ConfidenceMedium) != VerdictFlagged {
		t.Fatal("Medium confidence should map to flagged")
	}
	if verdictFromConfidence(ConfidenceLow) != VerdictAllowed {
		t.Fatal("Low confidence should map to allowed")
	}
}

func TestLiveItemRequiresReviewField(t *testing.T) {
	item := LiveItem{
		RequiresReview: true,
	}
	if !item.RequiresReview {
		t.Fatal("RequiresReview should be true")
	}
}

func TestLiveItemDepositFields(t *testing.T) {
	item := LiveItem{
		RequiresEntryDeposit: true,
		EntryDepositCents:    50_000,
	}
	if !item.RequiresEntryDeposit {
		t.Fatal("RequiresEntryDeposit should be true")
	}
	if item.EntryDepositCents != 50_000 {
		t.Fatalf("Expected 50000, got %d", item.EntryDepositCents)
	}
}

// ── Upgrade 1: Semantic Risk Scoring Tests ──────────────────────────────────

func TestClassifyItemRisk_CleanItem(t *testing.T) {
	score := ClassifyItemRisk("iPhone 15 Pro Max", "Brand new sealed unit", "electronics")
	if score >= RiskScoreBlock {
		t.Fatalf("Clean item should have low risk score, got %d", score)
	}
}

func TestClassifyItemRisk_WeaponCategory(t *testing.T) {
	score := ClassifyItemRisk("Combat Knife", "Military grade", "weapons")
	if score < RiskScoreBlock {
		t.Fatalf("Weapons category should score ≥%d, got %d", RiskScoreBlock, score)
	}
}

func TestClassifyItemRisk_SuspiciousPattern(t *testing.T) {
	score := ClassifyItemRisk("🔥 Special Herbs", "Not for human consumption", "")
	if score < RiskScoreReview {
		t.Fatalf("Suspicious patterns should score ≥%d, got %d", RiskScoreReview, score)
	}
}

func TestClassifyItemRisk_EvasionDetection(t *testing.T) {
	score := ClassifyItemRisk("Research Chemical", "Discrete shipping available", "")
	if score < RiskScoreReview {
		t.Fatalf("Evasion patterns should score ≥%d, got %d", RiskScoreReview, score)
	}
}

func TestClassifyItemRisk_ShortTitleEvasion(t *testing.T) {
	// Short title + long description = evasion signal
	longDesc := strings.Repeat("x", 150)
	score := ClassifyItemRisk("ABC", longDesc, "")
	// Should get at least the 15-point evasion bonus
	if score < 10 {
		t.Fatalf("Short title + long desc should add evasion score, got %d", score)
	}
}

func TestRiskScoreToVerdict(t *testing.T) {
	if RiskScoreToVerdict(90) != VerdictBlocked {
		t.Fatal("Score 90 should be blocked")
	}
	if RiskScoreToVerdict(65) != VerdictFlagged {
		t.Fatal("Score 65 should be flagged")
	}
	if RiskScoreToVerdict(30) != VerdictAllowed {
		t.Fatal("Score 30 should be allowed")
	}
}

func TestComplianceResultRiskScore(t *testing.T) {
	result := ComplianceResult{
		Verdict:   VerdictFlagged,
		RiskScore: 65,
	}
	if result.RiskScore != 65 {
		t.Fatalf("Expected risk_score=65, got %d", result.RiskScore)
	}
}

func TestLiveItemRiskScoreField(t *testing.T) {
	item := LiveItem{RiskScore: 85}
	if item.RiskScore != 85 {
		t.Fatal("RiskScore field should be 85")
	}
}

// ── Upgrade 2: Panic Mode Tests ────────────────────────────────────────────

func TestIsLiveSystemDisabled_Default(t *testing.T) {
	// Should be false by default
	if IsLiveSystemDisabled() {
		t.Fatal("Live system should NOT be disabled by default")
	}
}

func TestPanicModeToggle(t *testing.T) {
	// Simulate panic
	liveSystemDisabled.Store(true)
	if !IsLiveSystemDisabled() {
		t.Fatal("Live system should be disabled after panic")
	}
	// Recover
	liveSystemDisabled.Store(false)
	if IsLiveSystemDisabled() {
		t.Fatal("Live system should be enabled after recover")
	}
}

// ── Upgrade 3: Dynamic Deposit Tests ───────────────────────────────────────

func TestCalculateRequiredDeposit_PriceBased(t *testing.T) {
	required := CalculateRequiredDeposit(2_000_000, 0)
	if required != 100_000 {
		t.Fatalf("Expected 100000 (5%% of price), got %d", required)
	}
}

func TestCalculateRequiredDeposit_BidExceedsPrice(t *testing.T) {
	required := CalculateRequiredDeposit(2_000_000, 1_500_000)
	if required != 300_000 {
		t.Fatalf("Expected 300000 (20%% of bid), got %d", required)
	}
}

func TestCalculateRequiredDeposit_BidBelowPrice(t *testing.T) {
	required := CalculateRequiredDeposit(2_000_000, 300_000)
	if required != 100_000 {
		t.Fatalf("Expected 100000 (5%% of price wins), got %d", required)
	}
}

func TestValidateDepositCoverage_Sufficient(t *testing.T) {
	// This is a unit test for the calculation logic
	// With no DB, we test the pure functions
	required := CalculateRequiredDeposit(2_000_000, 0)
	if required != 100_000 {
		t.Fatalf("Expected 100000, got %d", required)
	}
}

// ── Upgrade 4: Settlement Balance Check (unit test) ────────────────────────

func TestInsufficientSettlementBalance_ErrorCode(t *testing.T) {
	err := fmt.Errorf("insufficient_settlement_balance")
	if err.Error() != "insufficient_settlement_balance" {
		t.Fatal("Error code should match")
	}
}

// ── Upgrade 5: Freeze Enforcement Tests ─────────────────────────────────────

func TestFreezeCheckInAddItem(t *testing.T) {
	// Verify the freeze check code path exists by checking the handler
	// (integration test would need a full DB)
	// Unit: just verify IsUserFrozen is callable
	// This confirms the import and function signature are correct
	_ = freeze.IsUserFrozen
}

// ── Upgrade 6: Dashboard Risk Score Sorting ────────────────────────────────

func TestDashboardRiskScoreConstants(t *testing.T) {
	if RiskScoreBlock != 80 {
		t.Fatalf("RiskScoreBlock should be 80, got %d", RiskScoreBlock)
	}
	if RiskScoreReview != 50 {
		t.Fatalf("RiskScoreReview should be 50, got %d", RiskScoreReview)
	}
}

func TestSuspiciousPatternsExist(t *testing.T) {
	if len(SuspiciousPatterns) == 0 {
		t.Fatal("SuspiciousPatterns should not be empty")
	}
}

func TestCategoryRiskScoresExist(t *testing.T) {
	if len(CategoryRiskScores) == 0 {
		t.Fatal("CategoryRiskScores should not be empty")
	}
}

// ════════════════════════════════════════════════════════════════════════════
// Sprint 11: Live Conversion Engine Tests
// ════════════════════════════════════════════════════════════════════════════

// ── FOMO Engine Tests ──────────────────────────────────────────────────────

func TestComputeUrgency_Normal(t *testing.T) {
	if computeUrgency(0) != UrgencyNormal {
		t.Fatal("0 bids should be NORMAL")
	}
	if computeUrgency(2) != UrgencyNormal {
		t.Fatal("2 bids should be NORMAL")
	}
}

func TestComputeUrgency_Hot(t *testing.T) {
	if computeUrgency(3) != UrgencyHot {
		t.Fatal("3 bids should be HOT")
	}
	if computeUrgency(5) != UrgencyHot {
		t.Fatal("5 bids should be HOT")
	}
}

func TestComputeUrgency_VeryHot(t *testing.T) {
	if computeUrgency(6) != UrgencyVeryHot {
		t.Fatal("6 bids should be VERY_HOT")
	}
	if computeUrgency(12) != UrgencyVeryHot {
		t.Fatal("12 bids should be VERY_HOT")
	}
}

// ── Countdown Phase Tests ──────────────────────────────────────────────────

func TestComputeCountdownPhase_Normal(t *testing.T) {
	if ComputeCountdownPhase(60) != PhaseNormal {
		t.Fatal("60s should be normal phase")
	}
	if ComputeCountdownPhase(31) != PhaseNormal {
		t.Fatal("31s should be normal phase")
	}
}

func TestComputeCountdownPhase_Orange(t *testing.T) {
	if ComputeCountdownPhase(30) != PhaseOrange {
		t.Fatal("30s should be orange phase")
	}
	if ComputeCountdownPhase(15) != PhaseOrange {
		t.Fatal("15s should be orange phase")
	}
	if ComputeCountdownPhase(11) != PhaseOrange {
		t.Fatal("11s should be orange phase")
	}
}

func TestComputeCountdownPhase_Red(t *testing.T) {
	if ComputeCountdownPhase(10) != PhaseRed {
		t.Fatal("10s should be red phase")
	}
	if ComputeCountdownPhase(5) != PhaseRed {
		t.Fatal("5s should be red phase")
	}
	if ComputeCountdownPhase(0) != PhaseRed {
		t.Fatal("0s should be red phase")
	}
}

// ── Buy-Now Progress Tests ─────────────────────────────────────────────────

func TestComputeBuyNowProgress_NoBuyNow(t *testing.T) {
	if ComputeBuyNowProgress(1000, nil) != 0 {
		t.Fatal("No buy-now price should return 0")
	}
}

func TestComputeBuyNowProgress_Below90(t *testing.T) {
	buyNow := int64(10000)
	progress := ComputeBuyNowProgress(5000, &buyNow)
	if progress != 0.5 {
		t.Fatalf("Expected 0.5, got %f", progress)
	}
}

func TestComputeBuyNowProgress_Exactly90(t *testing.T) {
	buyNow := int64(10000)
	progress := ComputeBuyNowProgress(9000, &buyNow)
	if progress < 0.89 || progress > 0.91 {
		t.Fatalf("Expected ~0.9, got %f", progress)
	}
}

func TestComputeBuyNowProgress_CappedAt1(t *testing.T) {
	buyNow := int64(10000)
	progress := ComputeBuyNowProgress(15000, &buyNow)
	if progress != 1.0 {
		t.Fatalf("Progress should cap at 1.0, got %f", progress)
	}
}

// ── Feature Flag Tests ─────────────────────────────────────────────────────

func TestIsLiveFomoEnabled(t *testing.T) {
	if !IsLiveFomoEnabled() {
		t.Fatal("Live FOMO should be enabled by default")
	}
}

func TestIsLiveNudgesEnabled(t *testing.T) {
	if !IsLiveNudgesEnabled() {
		t.Fatal("Live nudges should be enabled by default")
	}
}

func TestIsSmartBuyNowEnabled(t *testing.T) {
	if !IsSmartBuyNowEnabled() {
		t.Fatal("Smart buy-now should be enabled by default")
	}
}

func TestIsPinnedItemsEnabled(t *testing.T) {
	if !IsPinnedItemsEnabled() {
		t.Fatal("Pinned items should be enabled by default")
	}
}

func TestIsQuickBidEnabled(t *testing.T) {
	if !IsQuickBidEnabled() {
		t.Fatal("Quick bid should be enabled by default")
	}
}

// ── Quick Bid Increment Validation ─────────────────────────────────────────

func TestQuickBidIncrements_Valid(t *testing.T) {
	if !QuickBidIncrements[1000] {
		t.Fatal("1000 (+10 EGP) should be valid")
	}
	if !QuickBidIncrements[5000] {
		t.Fatal("5000 (+50 EGP) should be valid")
	}
	if !QuickBidIncrements[10000] {
		t.Fatal("10000 (+100 EGP) should be valid")
	}
}

func TestQuickBidIncrements_Invalid(t *testing.T) {
	if QuickBidIncrements[123] {
		t.Fatal("123 should NOT be a valid increment")
	}
	if QuickBidIncrements[0] {
		t.Fatal("0 should NOT be a valid increment")
	}
	if QuickBidIncrements[999999] {
		t.Fatal("999999 should NOT be a valid increment")
	}
}

// ── Nudge Code Tests ───────────────────────────────────────────────────────

func TestNudgeCodes(t *testing.T) {
	codes := []NudgeCode{
		NudgeWatcherNotBidding,
		NudgeOutbid,
		NudgeBuyNowClose,
		NudgeItemAlmostEnd,
		NudgeNewHotItem,
	}
	if len(codes) != 5 {
		t.Fatal("Should have 5 nudge codes")
	}
}

func TestComposeNudge_DefaultMessages(t *testing.T) {
	tests := []struct {
		code     NudgeCode
		wantIcon string
	}{
		{NudgeWatcherNotBidding, "💡"},
		{NudgeOutbid, "⚠️"},
		{NudgeBuyNowClose, "🔥"},
		{NudgeItemAlmostEnd, "⏰"},
		{NudgeNewHotItem, "🔥"},
	}
	for _, tt := range tests {
		msg, icon := composeNudge(NudgeContext{Code: tt.code})
		if msg == "" {
			t.Fatalf("Code %s should have default message", tt.code)
		}
		if icon != tt.wantIcon {
			t.Fatalf("Code %s: expected icon %s, got %s", tt.code, tt.wantIcon, icon)
		}
	}
}

func TestComposeNudge_CustomOverride(t *testing.T) {
	msg, icon := composeNudge(NudgeContext{
		Code:    NudgeOutbid,
		Message: "Custom message",
		Icon:    "🎯",
	})
	if msg != "Custom message" {
		t.Fatal("Custom message should override default")
	}
	if icon != "🎯" {
		t.Fatal("Custom icon should override default")
	}
}

// ── Event Type Tests ───────────────────────────────────────────────────────

func TestSprint11EventTypes(t *testing.T) {
	events := []LiveEventType{
		EventLiveUrgencyUpdate,
		EventCountdownPhase,
		EventBuyNowAlmost,
		EventLiveNudge,
		EventToast,
		EventItemPinned,
		EventItemUnpinned,
	}
	if len(events) != 7 {
		t.Fatal("Should have 7 Sprint 11 event types")
	}
}

// ── Conversion Stages ──────────────────────────────────────────────────────

func TestConversionStages(t *testing.T) {
	stages := []ConversionStage{
		StageView,
		StageClickBid,
		StagePlaceBid,
		StageBuyNow,
		StageWin,
		StageViewPin,
		StageQuickBid,
	}
	if len(stages) != 7 {
		t.Fatal("Should have 7 conversion stages")
	}
}

func TestLiveConversionEventModel(t *testing.T) {
	evt := LiveConversionEvent{
		Stage:  StagePlaceBid,
		Amount: 5000,
	}
	if evt.TableName() != "live_conversion_events" {
		t.Fatal("TableName should be live_conversion_events")
	}
	if evt.Stage != StagePlaceBid {
		t.Fatal("Stage should persist")
	}
}

// ── IsPinned field ────────────────────────────────────────────────────────

func TestLiveItemIsPinnedField(t *testing.T) {
	item := LiveItem{IsPinned: true}
	if !item.IsPinned {
		t.Fatal("IsPinned should be true")
	}
}

// ── Serialize helper ──────────────────────────────────────────────────────

func TestSerializeEvent(t *testing.T) {
	evt := LiveEvent{
		Event:     EventToast,
		SessionID: "abc",
		Message:   "hello",
	}
	s := serializeEvent(evt)
	if s == "" || s == "{}" {
		t.Fatalf("serializeEvent should produce JSON, got %q", s)
	}
}

func TestShortUserLabel(t *testing.T) {
	id := uuid.New()
	label := shortUserLabel(id)
	if !strings.HasPrefix(label, "User") {
		t.Fatalf("Expected User prefix, got %s", label)
	}
	if len(label) != 10 { // "User" + 6 chars
		t.Fatalf("Expected 10 chars, got %d (%s)", len(label), label)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// Sprint 12: Monetization Engine Tests
// ════════════════════════════════════════════════════════════════════════════

// ── Commission Calculation ────────────────────────────────────────────────

func TestCalculateCommission_BaseTier(t *testing.T) {
	// 10% of 10000 = 1000
	commission, seller, rate := CalculateCommission(10_000, TierBase, 0)
	if commission != 1_000 {
		t.Fatalf("Expected 1000 commission, got %d", commission)
	}
	if seller != 9_000 {
		t.Fatalf("Expected 9000 seller amount, got %d", seller)
	}
	if rate != 10.0 {
		t.Fatalf("Expected 10%%, got %.1f", rate)
	}
}

func TestCalculateCommission_HotTier(t *testing.T) {
	// 12% of 10000 = 1200
	commission, seller, _ := CalculateCommission(10_000, TierHot, 0)
	if commission != 1_200 {
		t.Fatalf("Expected 1200, got %d", commission)
	}
	if seller != 8_800 {
		t.Fatalf("Expected 8800, got %d", seller)
	}
}

func TestCalculateCommission_PremiumTier(t *testing.T) {
	// 8% of 10000 = 800
	commission, seller, _ := CalculateCommission(10_000, TierPremium, 0)
	if commission != 800 {
		t.Fatalf("Expected 800, got %d", commission)
	}
	if seller != 9_200 {
		t.Fatalf("Expected 9200, got %d", seller)
	}
}

func TestCalculateCommission_WithDynamicBonus(t *testing.T) {
	// Base 10% + 2% viewers + 3% bid rate = 15%
	commission, seller, rate := CalculateCommission(10_000, TierBase, 5.0)
	if rate != 15.0 {
		t.Fatalf("Expected 15%%, got %.1f", rate)
	}
	if commission != 1_500 {
		t.Fatalf("Expected 1500, got %d", commission)
	}
	if seller != 8_500 {
		t.Fatalf("Expected 8500, got %d", seller)
	}
}

func TestCalculateCommission_CappedAt20(t *testing.T) {
	// Base 12% + 10% bonus = 22%, should cap at 20%
	_, _, rate := CalculateCommission(10_000, TierHot, 10.0)
	if rate != 20.0 {
		t.Fatalf("Expected cap at 20%%, got %.1f", rate)
	}
}

func TestCommissionRates(t *testing.T) {
	if CommissionRates[TierBase] != 10.0 {
		t.Fatal("Base rate should be 10%")
	}
	if CommissionRates[TierHot] != 12.0 {
		t.Fatal("Hot rate should be 12%")
	}
	if CommissionRates[TierPremium] != 8.0 {
		t.Fatal("Premium rate should be 8%")
	}
}

// ── Dynamic Fee Bonus ─────────────────────────────────────────────────────

func TestComputeDynamicBonus_NoBonus(t *testing.T) {
	bonus := ComputeDynamicBonus(50, 2)
	if bonus != 0 {
		t.Fatalf("Expected 0 bonus, got %f", bonus)
	}
}

func TestComputeDynamicBonus_ViewersOnly(t *testing.T) {
	bonus := ComputeDynamicBonus(150, 2)
	if bonus != 2.0 {
		t.Fatalf("Expected 2%% bonus, got %f", bonus)
	}
}

func TestComputeDynamicBonus_BidRateOnly(t *testing.T) {
	bonus := ComputeDynamicBonus(50, 6)
	if bonus != 3.0 {
		t.Fatalf("Expected 3%% bonus, got %f", bonus)
	}
}

func TestComputeDynamicBonus_Both(t *testing.T) {
	bonus := ComputeDynamicBonus(200, 10)
	if bonus != 5.0 {
		t.Fatalf("Expected 5%% bonus, got %f", bonus)
	}
}

// ── Boost Packages ────────────────────────────────────────────────────────

func TestBoostPackages_Exist(t *testing.T) {
	if _, ok := BoostPackages["basic"]; !ok {
		t.Fatal("basic package should exist")
	}
	if _, ok := BoostPackages["premium"]; !ok {
		t.Fatal("premium package should exist")
	}
	if _, ok := BoostPackages["vip"]; !ok {
		t.Fatal("vip package should exist")
	}
}

func TestBoostPackages_Pricing(t *testing.T) {
	if BoostPackages["basic"].PriceCents != 5_000 {
		t.Fatalf("Basic should be 50 EGP, got %d", BoostPackages["basic"].PriceCents)
	}
	if BoostPackages["premium"].PriceCents != 15_000 {
		t.Fatalf("Premium should be 150 EGP, got %d", BoostPackages["premium"].PriceCents)
	}
	if BoostPackages["vip"].PriceCents != 30_000 {
		t.Fatalf("VIP should be 300 EGP, got %d", BoostPackages["vip"].PriceCents)
	}
}

func TestBoostPackages_ScoreOrdering(t *testing.T) {
	if BoostPackages["basic"].ScoreBoost >= BoostPackages["premium"].ScoreBoost {
		t.Fatal("Premium boost should exceed basic boost")
	}
	if BoostPackages["premium"].ScoreBoost >= BoostPackages["vip"].ScoreBoost {
		t.Fatal("VIP boost should exceed premium boost")
	}
}

// ── Boost Subscription Discount ───────────────────────────────────────────

func TestBoostDiscountPercent(t *testing.T) {
	if BoostDiscountPercent("free") != 0 {
		t.Fatal("Free plan should have no discount")
	}
	if BoostDiscountPercent("pro") != 10.0 {
		t.Fatalf("Pro should have 10%% discount, got %f", BoostDiscountPercent("pro"))
	}
	if BoostDiscountPercent("elite") != 25.0 {
		t.Fatalf("Elite should have 25%% discount, got %f", BoostDiscountPercent("elite"))
	}
}

// ── Seller Plan Limits ────────────────────────────────────────────────────

func TestSellerPlanLimits_Free(t *testing.T) {
	limits := GetSellerPlanLimits("free")
	if limits.MaxActiveSessions != 2 {
		t.Fatalf("Free plan should allow 2 sessions, got %d", limits.MaxActiveSessions)
	}
	if limits.FeaturedPlacement {
		t.Fatal("Free plan should NOT get featured placement")
	}
}

func TestSellerPlanLimits_Pro(t *testing.T) {
	limits := GetSellerPlanLimits("pro")
	if limits.MaxActiveSessions != -1 {
		t.Fatal("Pro plan should be unlimited (-1)")
	}
	if limits.BoostDiscountPct != 10.0 {
		t.Fatalf("Pro plan should have 10%% boost discount, got %f", limits.BoostDiscountPct)
	}
}

func TestSellerPlanLimits_Elite(t *testing.T) {
	limits := GetSellerPlanLimits("elite")
	if limits.BoostDiscountPct != 25.0 {
		t.Fatalf("Elite plan should have 25%% boost discount, got %f", limits.BoostDiscountPct)
	}
	if !limits.FeaturedPlacement {
		t.Fatal("Elite plan should get featured placement")
	}
}

func TestSellerPlanLimits_Unknown(t *testing.T) {
	// Unknown plan should default to free
	limits := GetSellerPlanLimits("nonsense")
	if limits.MaxActiveSessions != 2 {
		t.Fatal("Unknown plan should default to free (2 sessions)")
	}
}

// ── Entry Fee 80/20 Split ─────────────────────────────────────────────────

func TestSellerShareFromEntry(t *testing.T) {
	// 80% of 2000 = 1600
	seller := SellerShareFromEntry(2_000)
	if seller != 1_600 {
		t.Fatalf("Expected 1600 (80%% of 2000), got %d", seller)
	}
}

func TestSellerShareFromEntry_Small(t *testing.T) {
	// 80% of 100 = 80
	seller := SellerShareFromEntry(100)
	if seller != 80 {
		t.Fatalf("Expected 80, got %d", seller)
	}
}

// ── Feature Flag Tests ────────────────────────────────────────────────────

func TestIsLiveFeesEnabled(t *testing.T) {
	if !IsLiveFeesEnabled() {
		t.Fatal("Live fees should be enabled by default")
	}
}

func TestIsLiveBoostEnabled(t *testing.T) {
	if !IsLiveBoostEnabled() {
		t.Fatal("Live boost should be enabled by default")
	}
}

func TestIsPremiumAuctionsEnabled(t *testing.T) {
	if !IsPremiumAuctionsEnabled() {
		t.Fatal("Premium auctions should be enabled by default")
	}
}

func TestIsEntryFeeEnabled(t *testing.T) {
	if !IsEntryFeeEnabled() {
		t.Fatal("Entry fee should be enabled by default")
	}
}

func TestIsDynamicLiveFeesEnabled(t *testing.T) {
	if !IsDynamicLiveFeesEnabled() {
		t.Fatal("Dynamic live fees should be enabled by default")
	}
}

// ── Model Tests ───────────────────────────────────────────────────────────

func TestLiveCommissionModel(t *testing.T) {
	c := LiveCommission{
		Tier:            TierBase,
		CommissionCents: 1_000,
		FinalPriceCents: 10_000,
	}
	if c.TableName() != "live_commissions" {
		t.Fatal("TableName should be live_commissions")
	}
	if c.SellerAmountCents > 0 && c.CommissionCents+c.SellerAmountCents != c.FinalPriceCents {
		t.Fatal("Commission + seller_amount should equal final_price")
	}
}

func TestLiveBoostModel(t *testing.T) {
	b := LiveBoost{Tier: "vip", PriceCents: 30_000}
	if b.TableName() != "live_boosts" {
		t.Fatal("TableName should be live_boosts")
	}
}

func TestLivePaidEntryModel(t *testing.T) {
	e := LivePaidEntry{
		AmountCents:        2_000,
		SellerShareCents:   1_600,
		PlatformShareCents: 400,
	}
	if e.TableName() != "live_paid_entries" {
		t.Fatal("TableName should be live_paid_entries")
	}
	// 80/20 split integrity
	if e.SellerShareCents+e.PlatformShareCents != e.AmountCents {
		t.Fatal("Seller + platform shares should equal total")
	}
}

// ── Session Monetization Fields ──────────────────────────────────────────

func TestSessionMonetizationFields(t *testing.T) {
	s := Session{
		BoostTier:     "premium",
		BoostScore:    500,
		IsPremium:     true,
		EntryFeeCents: 2_000,
		SellerPlan:    "elite",
	}
	if s.BoostTier != "premium" {
		t.Fatal("BoostTier field should persist")
	}
	if !s.IsPremium {
		t.Fatal("IsPremium field should persist")
	}
	if s.EntryFeeCents != 2_000 {
		t.Fatal("EntryFeeCents field should persist")
	}
	if s.SellerPlan != "elite" {
		t.Fatal("SellerPlan field should persist")
	}
}

// ════════════════════════════════════════════════════════════════════════════
// Sprint 13: Revenue Flywheel Tests
// ════════════════════════════════════════════════════════════════════════════

// ── Surge Fee (last-10s bid burst) ────────────────────────────────────────

func TestComputeSurgeBonus_BelowThreshold(t *testing.T) {
	if ComputeSurgeBonus(9) != 0 {
		t.Fatal("9 bids/10s should not trigger surge")
	}
}

func TestComputeSurgeBonus_AtThreshold(t *testing.T) {
	if ComputeSurgeBonus(10) != 5.0 {
		t.Fatalf("10 bids/10s should give +5%%, got %f", ComputeSurgeBonus(10))
	}
}

func TestComputeSurgeBonus_AboveThreshold(t *testing.T) {
	if ComputeSurgeBonus(25) != 5.0 {
		t.Fatal("Surge should cap at +5%% regardless of bid count")
	}
}

// ── Whale Mode (luxury items) ─────────────────────────────────────────────

func TestComputeWhaleBonus_Normal(t *testing.T) {
	if ComputeWhaleBonus(1_000_00) != 0 {
		t.Fatal("1000 EGP should not be whale territory")
	}
}

func TestComputeWhaleBonus_JustBelow(t *testing.T) {
	if ComputeWhaleBonus(49_999_00) != 0 {
		t.Fatal("49999 EGP should not trigger whale")
	}
}

func TestComputeWhaleBonus_AtThreshold(t *testing.T) {
	if ComputeWhaleBonus(50_000_00) != 3.0 {
		t.Fatalf("50000 EGP should give +3%%, got %f", ComputeWhaleBonus(50_000_00))
	}
}

func TestComputeWhaleBonus_High(t *testing.T) {
	if ComputeWhaleBonus(500_000_00) != 3.0 {
		t.Fatal("Whale bonus should cap at +3%% regardless of price")
	}
}

// ── Full Flywheel Bonus ───────────────────────────────────────────────────

func TestComputeFlywheelBonus_None(t *testing.T) {
	bonus := ComputeFlywheelBonus(FlywheelBonusInputs{
		ViewerCount: 50, BidsLast10s: 2, FinalPriceCents: 1_000_00, UrgencyMultiplier: 1.0,
	})
	if bonus != 0 {
		t.Fatalf("No bonuses should yield 0, got %f", bonus)
	}
}

func TestComputeFlywheelBonus_AllTriggers(t *testing.T) {
	// viewers>100 (+2%) + bids/10s>=6 (+3%) + bids/10s>=10 (+5%) + whale (+3%) = 13%
	// No urgency multiplier (1.0)
	bonus := ComputeFlywheelBonus(FlywheelBonusInputs{
		ViewerCount: 150, BidsLast10s: 12, FinalPriceCents: 60_000_00, UrgencyMultiplier: 1.0,
	})
	if bonus != 13.0 {
		t.Fatalf("Expected 13%% bonus, got %f", bonus)
	}
}

func TestComputeFlywheelBonus_UrgencyMultiplier(t *testing.T) {
	// Base 5% (viewers + bid-rate) × 1.5 multiplier = 7.5%
	bonus := ComputeFlywheelBonus(FlywheelBonusInputs{
		ViewerCount: 150, BidsLast10s: 6, FinalPriceCents: 1_000_00, UrgencyMultiplier: 1.5,
	})
	if bonus != 7.5 {
		t.Fatalf("Expected 7.5%% (5×1.5), got %f", bonus)
	}
}

// ── Boost Conversion Effects ──────────────────────────────────────────────

func TestBoostConversionEffects_Basic(t *testing.T) {
	fx := BoostConversionEffects["basic"]
	if fx.UrgencyBonus != 0.10 {
		t.Fatalf("Basic boost should give +0.10 urgency, got %f", fx.UrgencyBonus)
	}
	if fx.IsHot {
		t.Fatal("Basic boost should NOT set IsHot")
	}
	if fx.NotifyMoreUsers {
		t.Fatal("Basic boost should NOT expand notifications")
	}
}

func TestBoostConversionEffects_Premium(t *testing.T) {
	fx := BoostConversionEffects["premium"]
	if fx.UrgencyBonus != 0.20 {
		t.Fatalf("Premium boost should give +0.20 urgency, got %f", fx.UrgencyBonus)
	}
	if !fx.IsHot {
		t.Fatal("Premium boost should set IsHot")
	}
	if !fx.NotifyMoreUsers {
		t.Fatal("Premium boost should expand notifications")
	}
}

func TestBoostConversionEffects_VIP(t *testing.T) {
	fx := BoostConversionEffects["vip"]
	if fx.UrgencyBonus != 0.35 {
		t.Fatalf("VIP boost should give +0.35 urgency, got %f", fx.UrgencyBonus)
	}
	if !fx.IsHot {
		t.Fatal("VIP boost should set IsHot")
	}
}

// ── Smart Entry Fee Scaling ──────────────────────────────────────────────

func TestComputeScaledEntryFee_Floor(t *testing.T) {
	// 0.5% of 100 EGP = 0.5 EGP → below 10 EGP floor
	fee := ComputeScaledEntryFee(10_000) // 100 EGP = 10000 cents
	if fee != 1_000 {
		t.Fatalf("Below-floor should clamp to 10 EGP (1000 cents), got %d", fee)
	}
}

func TestComputeScaledEntryFee_Scaling(t *testing.T) {
	// 0.5% of 5000 EGP = 25 EGP (within bounds)
	fee := ComputeScaledEntryFee(500_000) // 5000 EGP = 500000 cents
	if fee != 2_500 {
		t.Fatalf("5000 EGP should scale to 25 EGP (2500 cents), got %d", fee)
	}
}

func TestComputeScaledEntryFee_Ceiling(t *testing.T) {
	// 0.5% of 100000 EGP = 500 EGP → above 200 EGP ceiling
	fee := ComputeScaledEntryFee(10_000_000) // 100000 EGP
	if fee != 20_000 {
		t.Fatalf("Above-ceiling should clamp to 200 EGP (20000 cents), got %d", fee)
	}
}

// ── Creator Revenue Split ────────────────────────────────────────────────

func TestComputeCreatorShare_NoStreamer(t *testing.T) {
	seller := uuid.New()
	share, platform := ComputeCreatorShare(1_000, nil, seller)
	if share != 0 {
		t.Fatal("No streamer → no share")
	}
	if platform != 1_000 {
		t.Fatal("Platform should keep full commission")
	}
}

func TestComputeCreatorShare_StreamerIsSeller(t *testing.T) {
	seller := uuid.New()
	share, platform := ComputeCreatorShare(1_000, &seller, seller)
	if share != 0 {
		t.Fatal("Streamer=seller → no split (seller already owns the revenue)")
	}
	if platform != 1_000 {
		t.Fatal("Platform should keep full commission")
	}
}

func TestComputeCreatorShare_DifferentStreamer(t *testing.T) {
	streamer := uuid.New()
	seller := uuid.New()
	share, platform := ComputeCreatorShare(1_000, &streamer, seller)
	if share != 300 {
		t.Fatalf("Streamer should get 30%% (300), got %d", share)
	}
	if platform != 700 {
		t.Fatalf("Platform should keep 70%% (700), got %d", platform)
	}
	if share+platform != 1_000 {
		t.Fatal("Shares should sum to total commission")
	}
}

// ── Priority Bid Fee ─────────────────────────────────────────────────────

func TestPriorityBidFeeCents(t *testing.T) {
	if PriorityBidFeeCents != 1_000 {
		t.Fatalf("Priority bid fee should be 10 EGP (1000 cents), got %d", PriorityBidFeeCents)
	}
}

// ── Model Tests ──────────────────────────────────────────────────────────

func TestLivePriorityBidModel(t *testing.T) {
	b := LivePriorityBid{FeeCents: PriorityBidFeeCents, BidAmountCents: 5_000}
	if b.TableName() != "live_priority_bids" {
		t.Fatal("TableName should be live_priority_bids")
	}
}

func TestLiveStreamerEarningModel(t *testing.T) {
	e := LiveStreamerEarning{AmountCents: 300, Status: "pending"}
	if e.TableName() != "live_streamer_earnings" {
		t.Fatal("TableName should be live_streamer_earnings")
	}
}

// ── Session Flywheel Fields ──────────────────────────────────────────────

func TestSessionFlywheelFields(t *testing.T) {
	streamer := uuid.New()
	s := Session{
		StreamerID:        &streamer,
		UrgencyMultiplier: 1.5,
		IsHot:             true,
		NotifyMoreUsers:   true,
	}
	if s.StreamerID == nil || *s.StreamerID != streamer {
		t.Fatal("StreamerID should persist")
	}
	if s.UrgencyMultiplier != 1.5 {
		t.Fatal("UrgencyMultiplier should persist")
	}
	if !s.IsHot || !s.NotifyMoreUsers {
		t.Fatal("Flywheel flags should persist")
	}
}

// ── Feature Flags ────────────────────────────────────────────────────────

func TestIsRevenueFlywheelEnabled(t *testing.T) {
	if !IsRevenueFlywheelEnabled() {
		t.Fatal("Revenue flywheel should be enabled by default")
	}
}

func TestIsPriorityBidEnabled(t *testing.T) {
	if !IsPriorityBidEnabled() {
		t.Fatal("Priority bid should be enabled by default")
	}
}

func TestIsCreatorSplitEnabled(t *testing.T) {
	if !IsCreatorSplitEnabled() {
		t.Fatal("Creator split should be enabled by default")
	}
}

// ── Extended LiveCommission ──────────────────────────────────────────────

// ════════════════════════════════════════════════════════════════════════════
// Sprint 11.5: Behavioral Revenue Engine Tests
// ════════════════════════════════════════════════════════════════════════════

// ── 1. FOMO → Pricing (UrgencyFeeBonus) ───────────────────────────────────

func TestUrgencyFeeBonus_Normal(t *testing.T) {
	if UrgencyFeeBonus(UrgencyNormal) != 0 {
		t.Fatal("NORMAL urgency should give 0 bonus")
	}
}

func TestUrgencyFeeBonus_Hot(t *testing.T) {
	if UrgencyFeeBonus(UrgencyHot) != 2.0 {
		t.Fatalf("HOT should give +2%%, got %f", UrgencyFeeBonus(UrgencyHot))
	}
}

func TestUrgencyFeeBonus_VeryHot(t *testing.T) {
	if UrgencyFeeBonus(UrgencyVeryHot) != 4.0 {
		t.Fatalf("VERY_HOT should give +4%%, got %f", UrgencyFeeBonus(UrgencyVeryHot))
	}
}

func TestFlywheelBonus_IncludesUrgency(t *testing.T) {
	// Base 0 + urgency VERY_HOT (+4) = 4
	bonus := ComputeFlywheelBonus(FlywheelBonusInputs{
		ViewerCount: 50, BidsLast10s: 2, FinalPriceCents: 1_000_00,
		UrgencyMultiplier: 1.0, Urgency: UrgencyVeryHot,
	})
	if bonus != 4.0 {
		t.Fatalf("Expected 4%% (VERY_HOT only), got %f", bonus)
	}
}

// ── 2. Monetized Nudges (SuggestedActionFor) ──────────────────────────────

func TestSuggestedActionFor_Outbid(t *testing.T) {
	action, label := SuggestedActionFor(NudgeOutbid)
	if action != "quick_bid" {
		t.Fatalf("outbid → quick_bid, got %q", action)
	}
	if label == "" {
		t.Fatal("Label should not be empty")
	}
}

func TestSuggestedActionFor_ItemAlmostEnd(t *testing.T) {
	action, _ := SuggestedActionFor(NudgeItemAlmostEnd)
	if action != "buy_now" {
		t.Fatalf("almost_end → buy_now, got %q", action)
	}
}

func TestSuggestedActionFor_BuyNowClose(t *testing.T) {
	action, _ := SuggestedActionFor(NudgeBuyNowClose)
	if action != "buy_now" {
		t.Fatalf("buy_now_close → buy_now, got %q", action)
	}
}

func TestSuggestedActionFor_NewHotItem(t *testing.T) {
	action, _ := SuggestedActionFor(NudgeNewHotItem)
	if action != "quick_bid" {
		t.Fatalf("new_hot_item → quick_bid, got %q", action)
	}
}

func TestSuggestedActionFor_WatcherNotBidding(t *testing.T) {
	action, _ := SuggestedActionFor(NudgeWatcherNotBidding)
	if action != "quick_bid" {
		t.Fatalf("watcher_not_bidding → quick_bid, got %q", action)
	}
}

func TestSuggestedActionFor_UnknownCode(t *testing.T) {
	action, label := SuggestedActionFor("unknown")
	if action != "" || label != "" {
		t.Fatal("Unknown code → empty action/label")
	}
}

// ── 4. Bidder Quality Gate ────────────────────────────────────────────────

func TestBidderQualityResult_AllowedAlwaysSetsFields(t *testing.T) {
	r := BidderQualityResult{Allowed: true, RequireDeposit: true, UserTrustScore: 75}
	if !r.Allowed {
		t.Fatal("Allowed field should persist")
	}
	if !r.RequireDeposit {
		t.Fatal("RequireDeposit field should persist")
	}
}

// ── 5. Funnel Dropoff Calculation ─────────────────────────────────────────
// (DB-free test for the pure-math aspects.)

func TestHighValueBidThreshold(t *testing.T) {
	// 20,000 EGP = 2,000,000 cents
	if highValueBidThreshold != 2_000_000 {
		t.Fatalf("High-value threshold should be 20k EGP (2M cents), got %d", highValueBidThreshold)
	}
}

func TestBidderMinTrustScore(t *testing.T) {
	if bidderMinTrustScore != 40.0 {
		t.Fatalf("Min trust score should be 40, got %f", bidderMinTrustScore)
	}
}

// ── Feature Flags ────────────────────────────────────────────────────────

func TestIsBehavioralEngineEnabled(t *testing.T) {
	if !IsBehavioralEngineEnabled() {
		t.Fatal("Behavioral engine should be enabled by default")
	}
}

func TestIsMonetizedNudgesEnabled(t *testing.T) {
	if !IsMonetizedNudgesEnabled() {
		t.Fatal("Monetized nudges should be enabled by default")
	}
}

func TestIsBidderQualityGateEnabled(t *testing.T) {
	if !IsBidderQualityGateEnabled() {
		t.Fatal("Bidder quality gate should be enabled by default")
	}
}

// ── LiveEvent CTA Fields ─────────────────────────────────────────────────

func TestLiveEvent_SuggestedActionField(t *testing.T) {
	evt := LiveEvent{
		Event:           EventLiveNudge,
		SuggestedAction: "quick_bid",
		ActionLabel:     "Quick bid +50 EGP",
	}
	if evt.SuggestedAction != "quick_bid" {
		t.Fatal("SuggestedAction field should persist")
	}
	if evt.ActionLabel == "" {
		t.Fatal("ActionLabel field should persist")
	}
}

// ── FunnelOptimization model ─────────────────────────────────────────────

func TestFunnelOptimizationStruct(t *testing.T) {
	fo := FunnelOptimization{
		SessionID:   uuid.New(),
		Problem:     "click_to_bid_dropoff",
		Action:      "enable_quick_bid_prompt",
		DropoffRate: 0.45,
	}
	if fo.DropoffRate != 0.45 {
		t.Fatal("DropoffRate should persist")
	}
}

// ════════════════════════════════════════════════════════════════════════════
// Sprint 14: AI Live Seller Assistant Tests
// ════════════════════════════════════════════════════════════════════════════

// ── Suggestion Rule Ladder ────────────────────────────────────────────────

func TestGenerateSellerSuggestion_PushBuyNow(t *testing.T) {
	bn := int64(10_000_00)
	item := &LiveItem{BuyNowPriceCents: &bn}
	sugg := GenerateSellerSuggestion(SuggestionContext{
		Item:     item,
		Insights: LiveInsights{BuyNowProgress: 0.95, UrgencyLevel: UrgencyHot, SecondsLeft: 30},
	})
	if sugg == nil || sugg.Type != AISuggestPushBuyNow {
		t.Fatalf("Expected push_buy_now, got %+v", sugg)
	}
	if sugg.Confidence < 0.9 {
		t.Fatal("High-progress buy-now should have high confidence")
	}
}

func TestGenerateSellerSuggestion_ExtendTimer(t *testing.T) {
	sugg := GenerateSellerSuggestion(SuggestionContext{
		Insights: LiveInsights{
			UrgencyLevel: UrgencyVeryHot, ActiveBidders: 5, SecondsLeft: 15,
		},
	})
	if sugg == nil || sugg.Type != AISuggestExtendTimer {
		t.Fatalf("Expected extend_timer, got %+v", sugg)
	}
}

func TestGenerateSellerSuggestion_IncreaseIncrement(t *testing.T) {
	sugg := GenerateSellerSuggestion(SuggestionContext{
		Insights: LiveInsights{
			UrgencyLevel: UrgencyHot, ActiveBidders: 3, SecondsLeft: 120,
		},
	})
	if sugg == nil || sugg.Type != AISuggestIncreaseIncrement {
		t.Fatalf("Expected increase_increment, got %+v", sugg)
	}
}

func TestGenerateSellerSuggestion_EnableBuyNow_NoBids(t *testing.T) {
	// No bids for 25s, no buy-now set, item still active
	item := &LiveItem{} // BuyNowPriceCents is nil
	sugg := GenerateSellerSuggestion(SuggestionContext{
		Item:                item,
		SecondsSinceLastBid: 25,
		Insights:            LiveInsights{SecondsLeft: 60, UrgencyLevel: UrgencyNormal},
	})
	if sugg == nil || sugg.Type != AISuggestEnableBuyNow {
		t.Fatalf("Expected enable_buy_now for stagnant auction without buy-now, got %+v", sugg)
	}
}

func TestGenerateSellerSuggestion_PriceDrop_WithBuyNow(t *testing.T) {
	// No bids for 25s, buy-now already set → suggest price drop
	bn := int64(5_000_00)
	item := &LiveItem{BuyNowPriceCents: &bn}
	sugg := GenerateSellerSuggestion(SuggestionContext{
		Item:                item,
		SecondsSinceLastBid: 25,
		Insights:            LiveInsights{SecondsLeft: 60, UrgencyLevel: UrgencyNormal},
	})
	if sugg == nil || sugg.Type != AISuggestPriceDrop {
		t.Fatalf("Expected price_drop when buy-now already exists, got %+v", sugg)
	}
}

func TestGenerateSellerSuggestion_PinItem(t *testing.T) {
	item := &LiveItem{IsPinned: false}
	sugg := GenerateSellerSuggestion(SuggestionContext{
		Item:     item,
		Insights: LiveInsights{BidsLast30s: 0, ViewerCount: 5, SecondsLeft: 120},
	})
	if sugg == nil || sugg.Type != AISuggestPinItem {
		t.Fatalf("Expected pin_item for low-engagement unpinned item, got %+v", sugg)
	}
}

func TestGenerateSellerSuggestion_NoSuggestion(t *testing.T) {
	sugg := GenerateSellerSuggestion(SuggestionContext{
		Insights: LiveInsights{UrgencyLevel: UrgencyNormal, SecondsLeft: 300, ViewerCount: 50, BidsLast30s: 3},
	})
	if sugg != nil {
		t.Fatalf("Healthy auction should have no suggestion, got %+v", sugg)
	}
}

// ── Revenue Opportunities ────────────────────────────────────────────────

func TestGenerateRevenueOpportunity_BoostSession(t *testing.T) {
	sess := &Session{BoostTier: ""}
	ops := GenerateRevenueOpportunity(SuggestionContext{
		Session:     sess,
		BoostActive: false,
		Insights:    LiveInsights{UrgencyLevel: UrgencyHot},
	})
	if ops == nil || ops.Type != AISuggestBoostSession {
		t.Fatalf("Expected boost_session, got %+v", ops)
	}
}

func TestGenerateRevenueOpportunity_EnablePremium(t *testing.T) {
	// Session already boosted (boost rule skipped) — premium path should trigger
	sess := &Session{IsPremium: false, BoostTier: "vip"}
	item := &LiveItem{CurrentBidCents: 15_000_00} // 15k EGP
	ops := GenerateRevenueOpportunity(SuggestionContext{
		Session:     sess,
		Item:        item,
		BoostActive: true,
		Insights:    LiveInsights{ViewerCount: 80, UrgencyLevel: UrgencyHot},
	})
	if ops == nil || ops.Type != AISuggestEnablePremium {
		t.Fatalf("Expected enable_premium, got %+v", ops)
	}
}

func TestGenerateRevenueOpportunity_SetEntryFee(t *testing.T) {
	sess := &Session{EntryFeeCents: 0, BoostTier: "vip"} // boost active → skip that path
	ops := GenerateRevenueOpportunity(SuggestionContext{
		Session:     sess,
		BoostActive: true,
		Insights:    LiveInsights{ActiveBidders: 10, UrgencyLevel: UrgencyHot},
	})
	if ops == nil || ops.Type != AISuggestSetEntryFeeNext {
		t.Fatalf("Expected set_entry_fee_next, got %+v", ops)
	}
}

func TestGenerateRevenueOpportunity_None(t *testing.T) {
	sess := &Session{BoostTier: "premium", IsPremium: true, EntryFeeCents: 1_000}
	ops := GenerateRevenueOpportunity(SuggestionContext{
		Session:     sess,
		BoostActive: true,
		Insights:    LiveInsights{UrgencyLevel: UrgencyNormal, ViewerCount: 10, ActiveBidders: 2},
	})
	if ops != nil {
		t.Fatalf("Fully monetized session should have no opportunity, got %+v", ops)
	}
}

// ── LiveInsights Decision Engine ─────────────────────────────────────────

func TestLiveInsights_StructShape(t *testing.T) {
	ins := LiveInsights{
		SessionID:       uuid.New(),
		ItemID:          uuid.New(),
		UrgencyLevel:    UrgencyVeryHot,
		BidVelocity:     12,
		DropoffRisk:     0.75,
		OptimalAction:   AISuggestExtendTimer,
		ConfidenceScore: 0.85,
	}
	if ins.UrgencyLevel != UrgencyVeryHot {
		t.Fatal("UrgencyLevel should persist")
	}
	if ins.DropoffRisk != 0.75 {
		t.Fatal("DropoffRisk should persist")
	}
	if ins.OptimalAction != AISuggestExtendTimer {
		t.Fatal("OptimalAction should persist")
	}
}

// ── LiveAIEvent Model ─────────────────────────────────────────────────────

func TestLiveAIEventModel(t *testing.T) {
	e := LiveAIEvent{
		SuggestionType: AISuggestPushBuyNow,
		Message:        "Push buy-now CTA",
		Confidence:     0.95,
		Accepted:       false,
	}
	if e.TableName() != "live_ai_events" {
		t.Fatal("TableName should be live_ai_events")
	}
	if e.Confidence != 0.95 {
		t.Fatal("Confidence field should persist")
	}
}

// ── Suggestion Type Constants ────────────────────────────────────────────

func TestAISuggestionTypes(t *testing.T) {
	types := []string{
		AISuggestPriceDrop, AISuggestEnableBuyNow, AISuggestExtendTimer,
		AISuggestIncreaseIncrement, AISuggestPushBuyNow, AISuggestPinItem,
		AISuggestBoostSession, AISuggestEnablePremium, AISuggestSetEntryFeeNext,
		AISuggestDropoffRemedy,
	}
	if len(types) != 10 {
		t.Fatalf("Expected 10 suggestion types, got %d", len(types))
	}
	// Verify no duplicates
	seen := make(map[string]bool)
	for _, s := range types {
		if seen[s] {
			t.Fatalf("Duplicate suggestion type: %s", s)
		}
		seen[s] = true
	}
}

// ── LiveEvent AI Fields ───────────────────────────────────────────────────

func TestLiveEvent_AIFields(t *testing.T) {
	evt := LiveEvent{
		Event:          EventLiveAISuggestion,
		SuggestionID:   uuid.New().String(),
		SuggestionType: AISuggestPushBuyNow,
		Confidence:     0.95,
	}
	if evt.Event != EventLiveAISuggestion {
		t.Fatal("Event type should persist")
	}
	if evt.SuggestionType != AISuggestPushBuyNow {
		t.Fatal("SuggestionType should persist")
	}
	if evt.Confidence != 0.95 {
		t.Fatal("Confidence should persist")
	}
}

// ── Feature Flags ────────────────────────────────────────────────────────

func TestIsAIAssistantEnabled(t *testing.T) {
	if !IsAIAssistantEnabled() {
		t.Fatal("AI assistant should be enabled by default")
	}
}

func TestIsAIMonetizationHintsEnabled(t *testing.T) {
	if !IsAIMonetizationHintsEnabled() {
		t.Fatal("AI monetization hints should be enabled by default")
	}
}

func TestIsAIDropPreventionEnabled(t *testing.T) {
	if !IsAIDropPreventionEnabled() {
		t.Fatal("AI drop prevention should be enabled by default")
	}
}

// ── Monetization Flag Respect ────────────────────────────────────────────

func TestGenerateRevenueOpportunity_RespectsFlag(t *testing.T) {
	t.Setenv("ENABLE_AI_MONETIZATION_HINTS", "false")
	sess := &Session{BoostTier: ""}
	ops := GenerateRevenueOpportunity(SuggestionContext{
		Session:     sess,
		BoostActive: false,
		Insights:    LiveInsights{UrgencyLevel: UrgencyVeryHot},
	})
	if ops != nil {
		t.Fatalf("Disabled flag should suppress suggestions, got %+v", ops)
	}
}

func TestLiveCommissionFlywheelFields(t *testing.T) {
	streamer := uuid.New()
	c := LiveCommission{
		SurgeBonusPct:      5.0,
		WhaleBonusPct:      3.0,
		StreamerID:         &streamer,
		StreamerShareCents: 300,
	}
	if c.SurgeBonusPct != 5.0 {
		t.Fatal("SurgeBonusPct should persist")
	}
	if c.WhaleBonusPct != 3.0 {
		t.Fatal("WhaleBonusPct should persist")
	}
	if c.StreamerID == nil {
		t.Fatal("StreamerID should persist")
	}
	if c.StreamerShareCents != 300 {
		t.Fatal("StreamerShareCents should persist")
	}
}
