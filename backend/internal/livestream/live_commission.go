package livestream

import (
	"log/slog"
	"os"
	"time"

	"github.com/geocore-next/backend/internal/reputation"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Sprint 12: Live Monetization Engine — Commission
//
// Tiered commission rates with dynamic bonuses.
// Deducted at settlement, tracked per-transaction in `live_commissions` table.
//
// Feature-flagged via ENABLE_LIVE_FEES (default: true).
// ════════════════════════════════════════════════════════════════════════════

// ── Commission Tiers ──────────────────────────────────────────────────────

// CommissionTier enumerates commission tiers.
type CommissionTier string

const (
	TierBase    CommissionTier = "base"    // 10% — default
	TierHot     CommissionTier = "hot"     // 12% — trending/hot item
	TierPremium CommissionTier = "premium" // 8%  — premium/trusted seller
)

// CommissionRates defines % per tier (as percentage, not fraction).
var CommissionRates = map[CommissionTier]float64{
	TierBase:    10.0,
	TierHot:     12.0,
	TierPremium: 8.0,
}

// Dynamic fee bonuses (Feature 7)
const (
	dynamicViewerBonusThreshold = 100 // >100 viewers → +2%
	dynamicBidRateBonus         = 3.0 // high bid rate → +3%
	dynamicViewerBonus          = 2.0 // % added when viewer threshold exceeded
	dynamicBidRateHighThreshold = 6   // ≥6 bids in last 10s = high
)

// LiveCommission records a single commission charge (audit trail).
type LiveCommission struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SessionID         uuid.UUID      `gorm:"type:uuid;not null;index"                        json:"session_id"`
	ItemID            uuid.UUID      `gorm:"type:uuid;not null;index"                        json:"item_id"`
	OrderID           uuid.UUID      `gorm:"type:uuid;not null;index"                        json:"order_id"`
	SellerID          uuid.UUID      `gorm:"type:uuid;not null;index"                        json:"seller_id"`
	BuyerID           uuid.UUID      `gorm:"type:uuid;not null;index"                        json:"buyer_id"`
	FinalPriceCents   int64          `gorm:"not null"                                        json:"final_price_cents"`
	Tier              CommissionTier `gorm:"size:20;not null"                                json:"tier"`
	BaseRatePercent   float64        `gorm:"type:numeric(5,2);not null"                      json:"base_rate_percent"`
	DynamicBonusPct   float64        `gorm:"type:numeric(5,2);not null;default:0"            json:"dynamic_bonus_pct"`
	EffectiveRatePct  float64        `gorm:"type:numeric(5,2);not null"                      json:"effective_rate_pct"`
	CommissionCents   int64          `gorm:"not null"                                        json:"commission_cents"`
	SellerAmountCents int64          `gorm:"not null"                                        json:"seller_amount_cents"`

	// ── Sprint 13: Revenue Flywheel breakdown ──────────────────────────────
	SurgeBonusPct      float64    `gorm:"type:numeric(5,2);not null;default:0"            json:"surge_bonus_pct"`       // last-10s surge bonus
	WhaleBonusPct      float64    `gorm:"type:numeric(5,2);not null;default:0"            json:"whale_bonus_pct"`       // high-value item bonus
	StreamerID         *uuid.UUID `gorm:"type:uuid;index"                                 json:"streamer_id,omitempty"` // creator split (if streamer≠seller)
	StreamerShareCents int64      `gorm:"not null;default:0"                              json:"streamer_share_cents"`

	CreatedAt time.Time `gorm:"not null;index" json:"created_at"`
}

func (LiveCommission) TableName() string { return "live_commissions" }

// IsLiveFeesEnabled returns true unless disabled.
func IsLiveFeesEnabled() bool {
	val := os.Getenv("ENABLE_LIVE_FEES")
	if val == "" {
		return true
	}
	return val != "false" && val != "0"
}

// IsDynamicLiveFeesEnabled returns true unless disabled.
func IsDynamicLiveFeesEnabled() bool {
	val := os.Getenv("ENABLE_DYNAMIC_LIVE_FEES")
	if val == "" {
		return true
	}
	return val != "false" && val != "0"
}

// ── Commission Calculation ─────────────────────────────────────────────────

// DetermineTier picks the appropriate commission tier for an item/seller.
//   - Premium seller (trust score ≥80) → TierPremium (8%)
//   - Hot item (urgency VERY_HOT OR high bid velocity) → TierHot (12%)
//   - Otherwise → TierBase (10%)
func DetermineTier(db *gorm.DB, sellerID uuid.UUID, isHot bool) CommissionTier {
	// Premium seller check
	sellerScore := reputation.GetOverallScore(db, sellerID)
	if sellerScore >= 80 {
		return TierPremium
	}
	if isHot {
		return TierHot
	}
	return TierBase
}

// ComputeDynamicBonus returns the extra % to add to the commission rate.
// Based on viewer count + bid rate.
func ComputeDynamicBonus(viewerCount, bidsLast10s int) float64 {
	if !IsDynamicLiveFeesEnabled() {
		return 0
	}
	bonus := 0.0
	if viewerCount > dynamicViewerBonusThreshold {
		bonus += dynamicViewerBonus
	}
	if bidsLast10s >= dynamicBidRateHighThreshold {
		bonus += dynamicBidRateBonus
	}
	return bonus
}

// CalculateCommission computes commission cents + seller-receives cents.
// Returns (commissionCents, sellerAmountCents, effectiveRatePct).
func CalculateCommission(finalPriceCents int64, tier CommissionTier, dynamicBonusPct float64) (int64, int64, float64) {
	baseRate, ok := CommissionRates[tier]
	if !ok {
		baseRate = CommissionRates[TierBase]
	}
	effectiveRate := baseRate + dynamicBonusPct
	if effectiveRate > 20 {
		effectiveRate = 20 // hard cap at 20%
	}
	commission := int64(float64(finalPriceCents) * effectiveRate / 100.0)
	if commission < 0 {
		commission = 0
	}
	sellerAmount := finalPriceCents - commission
	if sellerAmount < 0 {
		sellerAmount = 0
	}
	return commission, sellerAmount, effectiveRate
}

// RecordCommission persists a commission row.
// Fire-and-forget — errors logged but don't block settlement.
func RecordCommission(db *gorm.DB, row LiveCommission) {
	if !IsLiveFeesEnabled() {
		return
	}
	row.CreatedAt = time.Now()
	if err := db.Create(&row).Error; err != nil {
		slog.Error("live-commission: failed to persist", "error", err,
			"item_id", row.ItemID, "order_id", row.OrderID)
	}
}

// ── Hotness Signal ─────────────────────────────────────────────────────────

// isItemHot returns true if the item is currently in a HOT / VERY_HOT urgency state.
// Uses the FOMO engine's Redis-backed bid velocity counter.
func (h *LiveAuctionHandler) isItemHot(itemID uuid.UUID) bool {
	if h.rdb == nil {
		return false
	}
	bids10 := countBidsInWindow(h.rdb, itemID, fomoWindow10s)
	return computeUrgency(bids10) != UrgencyNormal
}
