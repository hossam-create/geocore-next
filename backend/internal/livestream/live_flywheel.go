package livestream

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/geocore-next/backend/internal/freeze"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LiveStreamerEarning records unpaid earnings owed to a streamer from the
// creator-revenue split. Reconciliation to wallet is out-of-band (cron/admin).
type LiveStreamerEarning struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	StreamerID  uuid.UUID  `gorm:"type:uuid;not null;index"                        json:"streamer_id"`
	SessionID   uuid.UUID  `gorm:"type:uuid;not null;index"                        json:"session_id"`
	OrderID     uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"                  json:"order_id"`
	AmountCents int64      `gorm:"not null"                                        json:"amount_cents"`
	Status      string     `gorm:"size:20;not null;default:'pending';index"        json:"status"` // pending, paid
	CreatedAt   time.Time  `gorm:"not null"                                        json:"created_at"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
}

func (LiveStreamerEarning) TableName() string { return "live_streamer_earnings" }

// ════════════════════════════════════════════════════════════════════════════
// Sprint 13: Revenue Flywheel
//
// Connects Boost, Urgency, Whale Extraction, Priority Bidding, Smart Entry Fee,
// and Creator Revenue Split into a single reinforcing revenue system.
//
// Philosophy:
//
//	Boost → ↑Urgency → ↑Bids/sec → ↑Surge Fee → Whale attracted → ↑Commission
//	         ↑Notifications → ↑Viewers → ↑Dynamic Bonus → more Boost buyers…
// ════════════════════════════════════════════════════════════════════════════

// ── Feature Flags ──────────────────────────────────────────────────────────

func IsRevenueFlywheelEnabled() bool {
	val := os.Getenv("ENABLE_REVENUE_FLYWHEEL")
	if val == "" {
		return true
	}
	return val != "false" && val != "0"
}

func IsPriorityBidEnabled() bool {
	val := os.Getenv("ENABLE_PRIORITY_BID")
	if val == "" {
		return true
	}
	return val != "false" && val != "0"
}

func IsCreatorSplitEnabled() bool {
	val := os.Getenv("ENABLE_CREATOR_SPLIT")
	if val == "" {
		return true
	}
	return val != "false" && val != "0"
}

// ════════════════════════════════════════════════════════════════════════════
// 1. Boost → Conversion (not just visibility)
// ════════════════════════════════════════════════════════════════════════════

// BoostConversionEffects maps each boost tier to its urgency/notification impact.
var BoostConversionEffects = map[string]struct {
	UrgencyBonus    float64 // added to UrgencyMultiplier
	NotifyMoreUsers bool
	IsHot           bool
}{
	"basic":   {UrgencyBonus: 0.10, NotifyMoreUsers: false, IsHot: false},
	"premium": {UrgencyBonus: 0.20, NotifyMoreUsers: true, IsHot: true},
	"vip":     {UrgencyBonus: 0.35, NotifyMoreUsers: true, IsHot: true},
}

// ApplyBoostEffects applies conversion-level effects of a boost purchase to the
// session. Called from PurchaseBoost after the wallet hold succeeds.
//
// Updates in the same tx that's already active.
func ApplyBoostEffects(tx *gorm.DB, sess *Session, tier string) error {
	fx, ok := BoostConversionEffects[tier]
	if !ok {
		return nil // unknown tier → no-op
	}
	updates := map[string]interface{}{
		"urgency_multiplier": gorm.Expr("LEAST(urgency_multiplier + ?, 3.0)", fx.UrgencyBonus),
	}
	if fx.NotifyMoreUsers {
		updates["notify_more_users"] = true
	}
	if fx.IsHot {
		updates["is_hot"] = true
	}
	return tx.Model(sess).Updates(updates).Error
}

// ════════════════════════════════════════════════════════════════════════════
// 2. Last-10s Bid Surge Fee
// ════════════════════════════════════════════════════════════════════════════

const (
	surgeBidThreshold = 10  // ≥10 bids in last 10s triggers surge
	surgeBonusPct     = 5.0 // +5% commission
)

// ComputeSurgeBonus returns additional commission % when bid velocity crosses surge threshold.
func ComputeSurgeBonus(bidsLast10s int) float64 {
	if !IsRevenueFlywheelEnabled() {
		return 0
	}
	if bidsLast10s >= surgeBidThreshold {
		return surgeBonusPct
	}
	return 0
}

// ════════════════════════════════════════════════════════════════════════════
// 3. Whale Mode — extract luxury buyers
// ════════════════════════════════════════════════════════════════════════════

const (
	whaleThresholdCents = 50_000_00 // 50,000 EGP in cents (≈ $1,000)
	whaleBonusPct       = 3.0       // +3% on luxury items
)

// ComputeWhaleBonus returns additional commission % for high-value items.
func ComputeWhaleBonus(finalPriceCents int64) float64 {
	if !IsRevenueFlywheelEnabled() {
		return 0
	}
	if finalPriceCents >= whaleThresholdCents {
		return whaleBonusPct
	}
	return 0
}

// ════════════════════════════════════════════════════════════════════════════
// 4. Paid Priority Bidding
// ════════════════════════════════════════════════════════════════════════════

// PriorityBidFeeCents is the flat cost of priority bidding (10 EGP).
const PriorityBidFeeCents int64 = 1_000

// LivePriorityBid records a single priority-bid fee payment.
type LivePriorityBid struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SessionID      uuid.UUID `gorm:"type:uuid;not null;index"                        json:"session_id"`
	ItemID         uuid.UUID `gorm:"type:uuid;not null;index"                        json:"item_id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index"                        json:"user_id"`
	FeeCents       int64     `gorm:"not null"                                        json:"fee_cents"`
	BidAmountCents int64     `gorm:"not null"                                        json:"bid_amount_cents"`
	CreatedAt      time.Time `json:"created_at"`
}

func (LivePriorityBid) TableName() string { return "live_priority_bids" }

// HasPriorityBidForItem checks if the user has an active priority flag for this item.
// Used to decorate broadcast events (bid highlighted/animated).
func HasPriorityBidForItem(db *gorm.DB, userID, itemID uuid.UUID) bool {
	var count int64
	db.Model(&LivePriorityBid{}).
		Where("user_id = ? AND item_id = ?", userID, itemID).
		Count(&count)
	return count > 0
}

// ── POST /livestream/:id/items/:itemId/bid/priority ─────────────────────────

// PriorityBid charges a 10 EGP fee and then processes the bid with a "priority"
// flag (highlighted, appears-first, animation). The bid itself uses the normal
// PlaceBid pipeline — this handler just prepays the fee and tags the bid.
func (h *LiveAuctionHandler) PriorityBid(c *gin.Context) {
	if !IsPriorityBidEnabled() {
		response.BadRequest(c, "Priority bidding is not enabled")
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))
	itemID, _ := uuid.Parse(c.Param("itemId"))

	if freeze.IsUserFrozen(h.db, userID) {
		response.Forbidden(c)
		return
	}

	var req struct {
		AmountCents int64 `json:"amount_cents" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Charge the priority fee first (atomic)
	err := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		if !wallet.HasSufficientBalance(tx, userID, PriorityBidFeeCents+req.AmountCents) {
			return fmt.Errorf("insufficient_balance")
		}
		// Reserve the priority fee permanently — it's platform revenue
		if err := wallet.ReserveFunds(tx, userID, PriorityBidFeeCents); err != nil {
			return fmt.Errorf("priority_reserve_failed:%s", err.Error())
		}
		return tx.Create(&LivePriorityBid{
			SessionID:      sessionID,
			ItemID:         itemID,
			UserID:         userID,
			FeeCents:       PriorityBidFeeCents,
			BidAmountCents: req.AmountCents,
		}).Error
	})
	if err != nil {
		msg := err.Error()
		switch msg {
		case "insufficient_balance":
			response.BadRequest(c, "Insufficient balance for priority bid + fee")
		default:
			response.BadRequest(c, msg)
		}
		return
	}

	freeze.LogAudit(h.db, "live_priority_bid_fee", userID, itemID,
		fmt.Sprintf("fee_cents=%d bid_cents=%d", PriorityBidFeeCents, req.AmountCents))

	// Inject the amount into the context and call PlaceBid normally
	c.Set("quick_bid_amount_cents", req.AmountCents)
	c.Set("priority_bid", true)
	h.PlaceBid(c)
}

// ════════════════════════════════════════════════════════════════════════════
// 5. Smart Entry Fee Scaling
// ════════════════════════════════════════════════════════════════════════════

const (
	entryFeeRate     = 0.005  // 0.5% of price
	entryFeeMinCents = 1_000  // 10 EGP
	entryFeeMaxCents = 20_000 // 200 EGP
)

// ComputeScaledEntryFee returns a scaled entry fee in cents based on the
// item/session price. 0.5% with floor 10 EGP and ceiling 200 EGP.
func ComputeScaledEntryFee(estimatedPriceCents int64) int64 {
	fee := int64(float64(estimatedPriceCents) * entryFeeRate)
	if fee < entryFeeMinCents {
		fee = entryFeeMinCents
	}
	if fee > entryFeeMaxCents {
		fee = entryFeeMaxCents
	}
	return fee
}

// ════════════════════════════════════════════════════════════════════════════
// 6. Creator Revenue Split
// ════════════════════════════════════════════════════════════════════════════

const creatorSharePct = 0.30 // 30% of commission goes to streamer

// ComputeCreatorShare splits commission between platform and streamer.
// Returns (streamerShareCents, platformShareCents).
// If streamerID == nil or equals sellerID, no split.
func ComputeCreatorShare(commissionCents int64, streamerID *uuid.UUID, sellerID uuid.UUID) (int64, int64) {
	if !IsCreatorSplitEnabled() || streamerID == nil || *streamerID == sellerID {
		return 0, commissionCents
	}
	streamerShare := int64(float64(commissionCents) * creatorSharePct)
	if streamerShare < 0 {
		streamerShare = 0
	}
	platformShare := commissionCents - streamerShare
	return streamerShare, platformShare
}

// RecordStreamerEarning appends an unpaid ledger entry for the streamer's cut.
// The platform's share already stays with the platform (commission is deducted
// from buyer→seller escrow as usual); this row is a promise-to-pay that the
// platform reconciles via admin/cron.
//
// Fire-and-forget — errors logged but don't block settlement.
func RecordStreamerEarning(db *gorm.DB, streamerID, sessionID, orderID uuid.UUID, amountCents int64) {
	if amountCents <= 0 {
		return
	}
	row := LiveStreamerEarning{
		StreamerID:  streamerID,
		SessionID:   sessionID,
		OrderID:     orderID,
		AmountCents: amountCents,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}
	if err := db.Create(&row).Error; err != nil {
		slog.Error("live-flywheel: failed to record streamer earning", "error", err,
			"streamer_id", streamerID, "order_id", orderID, "cents", amountCents)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// Full Dynamic Bonus (combines all bonuses into one)
// ════════════════════════════════════════════════════════════════════════════

// FlywheelBonusInputs captures all signals that affect the dynamic commission bonus.
type FlywheelBonusInputs struct {
	ViewerCount       int
	BidsLast10s       int
	FinalPriceCents   int64
	UrgencyMultiplier float64      // from Session (boost-amplified, default 1.0)
	Urgency           UrgencyState // Sprint 11.5: current FOMO urgency state
}

// ComputeFlywheelBonus returns the total dynamic bonus % combining:
//   - Viewer bonus (+2% if viewers>100)
//   - Bid rate bonus (+3% if bids/10s≥6)
//   - Surge bonus (+5% if bids/10s≥10)
//   - Whale bonus (+3% if price≥50k EGP)
//   - Urgency fee bonus (Sprint 11.5: HOT +2%, VERY_HOT +4%)
//   - Urgency multiplier (applied to the sum, from Boost tier)
//
// Always capped via CalculateCommission's 20% hard cap downstream.
func ComputeFlywheelBonus(in FlywheelBonusInputs) float64 {
	base := ComputeDynamicBonus(in.ViewerCount, in.BidsLast10s) // viewer + bid-rate
	base += ComputeSurgeBonus(in.BidsLast10s)
	base += ComputeWhaleBonus(in.FinalPriceCents)
	base += UrgencyFeeBonus(in.Urgency) // Sprint 11.5: FOMO → pricing
	// Apply urgency multiplier (default 1.0 → no change)
	if in.UrgencyMultiplier > 0 {
		base *= in.UrgencyMultiplier
	}
	return base
}
