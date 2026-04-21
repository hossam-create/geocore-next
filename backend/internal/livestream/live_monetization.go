package livestream

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/geocore-next/backend/internal/freeze"
	"github.com/geocore-next/backend/internal/reputation"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Sprint 12: Live Monetization Engine
//
//   - Live Boost System (3 packages)
//   - Premium Auction Mode (trust-score gated)
//   - Pay-to-Enter VIP (80/20 split)
//   - Seller Subscription plans
//   - Performance metrics dashboard
// ════════════════════════════════════════════════════════════════════════════

// ── Feature Flags ──────────────────────────────────────────────────────────

func IsLiveBoostEnabled() bool {
	val := os.Getenv("ENABLE_LIVE_BOOST")
	if val == "" {
		return true
	}
	return val != "false" && val != "0"
}

func IsPremiumAuctionsEnabled() bool {
	val := os.Getenv("ENABLE_PREMIUM_AUCTIONS")
	if val == "" {
		return true
	}
	return val != "false" && val != "0"
}

func IsEntryFeeEnabled() bool {
	val := os.Getenv("ENABLE_ENTRY_FEE")
	if val == "" {
		return true
	}
	return val != "false" && val != "0"
}

// ════════════════════════════════════════════════════════════════════════════
// Live Boost System
// ════════════════════════════════════════════════════════════════════════════

// BoostPackage represents a boost tier.
type BoostPackage struct {
	Tier        string `json:"tier"`
	PriceCents  int64  `json:"price_cents"`
	ScoreBoost  int    `json:"score_boost"`
	Description string `json:"description"`
}

// BoostPackages are the available boost tiers.
var BoostPackages = map[string]BoostPackage{
	"basic": {
		Tier: "basic", PriceCents: 5_000, ScoreBoost: 100, // 50 EGP
		Description: "+visibility score in feed",
	},
	"premium": {
		Tier: "premium", PriceCents: 15_000, ScoreBoost: 500, // 150 EGP
		Description: "Top placement in category feed",
	},
	"vip": {
		Tier: "vip", PriceCents: 30_000, ScoreBoost: 1000, // 300 EGP
		Description: "Homepage exposure + all premium benefits",
	},
}

// LiveBoost tracks a purchased boost.
type LiveBoost struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SessionID      uuid.UUID `gorm:"type:uuid;not null;index"                        json:"session_id"`
	SellerID       uuid.UUID `gorm:"type:uuid;not null;index"                        json:"seller_id"`
	Tier           string    `gorm:"size:20;not null"                                json:"tier"`
	PriceCents     int64     `gorm:"not null"                                        json:"price_cents"`
	ScoreBoost     int       `gorm:"not null"                                        json:"score_boost"`
	Status         string    `gorm:"size:20;not null;default:'active'"               json:"status"` // active, expired, refunded
	IdempotencyKey string    `gorm:"size:255;index"                                  json:"idempotency_key,omitempty"`
	ExpiresAt      time.Time `gorm:"not null"                                         json:"expires_at"`
	CreatedAt      time.Time `json:"created_at"`
}

func (LiveBoost) TableName() string { return "live_boosts" }

// BoostDiscountPercent returns the discount % for a given seller plan.
func BoostDiscountPercent(plan string) float64 {
	switch plan {
	case "pro":
		return 10.0 // 10% off
	case "elite":
		return 25.0 // 25% off
	default:
		return 0
	}
}

// ── POST /livestream/:id/boost — purchase a boost ───────────────────────

func (h *LiveAuctionHandler) PurchaseBoost(c *gin.Context) {
	if !IsLiveBoostEnabled() {
		response.BadRequest(c, "Live boost is not enabled")
		return
	}

	sellerID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))

	if freeze.IsUserFrozen(h.db, sellerID) {
		response.Forbidden(c)
		return
	}

	var req struct {
		Tier string `json:"tier" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	pkg, ok := BoostPackages[req.Tier]
	if !ok {
		response.BadRequest(c, "Invalid boost tier. Allowed: basic, premium, vip")
		return
	}

	// Verify seller owns the session
	var sess Session
	if err := h.db.Where("id = ? AND host_id = ?", sessionID, sellerID).First(&sess).Error; err != nil {
		response.Forbidden(c)
		return
	}

	// Apply subscription discount
	discountPct := BoostDiscountPercent(sess.SellerPlan)
	effectivePrice := pkg.PriceCents
	if discountPct > 0 {
		effectivePrice = pkg.PriceCents - int64(float64(pkg.PriceCents)*discountPct/100.0)
	}

	idemKey := c.GetHeader("X-Idempotency-Key")

	err := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		if !wallet.HasSufficientBalance(tx, sellerID, effectivePrice) {
			return fmt.Errorf("insufficient_balance")
		}
		// Hold funds (revenue) — we debit immediately to platform wallet is out-of-scope;
		// using ReserveFunds to block the balance so it can't be double-spent.
		if err := wallet.ReserveFunds(tx, sellerID, effectivePrice); err != nil {
			return fmt.Errorf("reserve_failed:%s", err.Error())
		}

		boost := LiveBoost{
			SessionID:      sessionID,
			SellerID:       sellerID,
			Tier:           pkg.Tier,
			PriceCents:     effectivePrice,
			ScoreBoost:     pkg.ScoreBoost,
			Status:         "active",
			IdempotencyKey: idemKey,
			ExpiresAt:      time.Now().Add(24 * time.Hour),
		}
		if err := tx.Create(&boost).Error; err != nil {
			_ = wallet.ReleaseReservedFunds(tx, sellerID, effectivePrice)
			return fmt.Errorf("boost_create_failed:%s", err.Error())
		}

		// Apply boost to session
		if err := tx.Model(&sess).Updates(map[string]interface{}{
			"boost_tier":  pkg.Tier,
			"boost_score": gorm.Expr("boost_score + ?", pkg.ScoreBoost),
		}).Error; err != nil {
			return err
		}
		// Sprint 13: Revenue Flywheel — boost also amplifies urgency/notifications
		if IsRevenueFlywheelEnabled() {
			if err := ApplyBoostEffects(tx, &sess, pkg.Tier); err != nil {
				return fmt.Errorf("boost_effects_failed:%s", err.Error())
			}
		}
		return nil
	})

	if err != nil {
		msg := err.Error()
		switch msg {
		case "insufficient_balance":
			response.BadRequest(c, "Insufficient balance for boost")
		default:
			response.BadRequest(c, msg)
		}
		return
	}

	freeze.LogAudit(h.db, "live_boost_purchased", sellerID, sessionID,
		fmt.Sprintf("tier=%s price_cents=%d score=%d discount=%.1f%%", pkg.Tier, effectivePrice, pkg.ScoreBoost, discountPct))
	slog.Info("live-monetization: boost purchased",
		"seller", sellerID, "session", sessionID, "tier", pkg.Tier, "price", effectivePrice)

	response.OK(c, gin.H{
		"message":          "Boost activated",
		"tier":             pkg.Tier,
		"price_cents":      effectivePrice,
		"discount_pct":     discountPct,
		"score_boost":      pkg.ScoreBoost,
		"expires_in_hours": 24,
	})
}

// ════════════════════════════════════════════════════════════════════════════
// Premium Auction Mode
// ════════════════════════════════════════════════════════════════════════════

const premiumTrustScoreRequired = 70

// ── POST /livestream/:id/premium — enable premium mode ──────────────────

func (h *LiveAuctionHandler) EnablePremiumMode(c *gin.Context) {
	if !IsPremiumAuctionsEnabled() {
		response.BadRequest(c, "Premium auctions are not enabled")
		return
	}

	sellerID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))

	// Verify seller owns the session
	var sess Session
	if err := h.db.Where("id = ? AND host_id = ?", sessionID, sellerID).First(&sess).Error; err != nil {
		response.Forbidden(c)
		return
	}

	// Trust gate
	trustScore := reputation.GetOverallScore(h.db, sellerID)
	if trustScore < premiumTrustScoreRequired {
		response.BadRequest(c, fmt.Sprintf("Premium mode requires trust score ≥%d (yours: %.1f)", premiumTrustScoreRequired, trustScore))
		return
	}

	if err := h.db.Model(&sess).Update("is_premium", true).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	freeze.LogAudit(h.db, "live_premium_enabled", sellerID, sessionID,
		fmt.Sprintf("trust_score=%.1f", trustScore))

	response.OK(c, gin.H{
		"message":     "Premium mode enabled",
		"trust_score": trustScore,
	})
}

// ════════════════════════════════════════════════════════════════════════════
// Pay-to-Enter VIP Auctions
// ════════════════════════════════════════════════════════════════════════════

// LivePaidEntry records a user's paid entry to a session.
type LivePaidEntry struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SessionID          uuid.UUID `gorm:"type:uuid;not null;index:idx_session_user,unique"  json:"session_id"`
	UserID             uuid.UUID `gorm:"type:uuid;not null;index:idx_session_user,unique"  json:"user_id"`
	AmountCents        int64     `gorm:"not null"                                        json:"amount_cents"`
	SellerShareCents   int64     `gorm:"not null"                                        json:"seller_share_cents"`
	PlatformShareCents int64     `gorm:"not null"                                        json:"platform_share_cents"`
	IdempotencyKey     string    `gorm:"size:255;index"                                  json:"idempotency_key,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

func (LivePaidEntry) TableName() string { return "live_paid_entries" }

// HasUserPaidEntry checks if a user has paid the entry fee for a session.
func HasUserPaidEntry(db *gorm.DB, userID, sessionID uuid.UUID) bool {
	var count int64
	db.Model(&LivePaidEntry{}).
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		Count(&count)
	return count > 0
}

// SellerShareFromEntry returns the seller's cut (80%) of an entry fee in cents.
func SellerShareFromEntry(amountCents int64) int64 {
	return int64(float64(amountCents) * 0.80)
}

// ── POST /livestream/:id/entry — pay the entry fee ──────────────────────

func (h *LiveAuctionHandler) PayEntryFee(c *gin.Context) {
	if !IsEntryFeeEnabled() {
		response.BadRequest(c, "Entry fee is not enabled")
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))

	if freeze.IsUserFrozen(h.db, userID) {
		response.Forbidden(c)
		return
	}

	var sess Session
	if err := h.db.Where("id = ?", sessionID).First(&sess).Error; err != nil {
		response.NotFound(c, "Session")
		return
	}
	if sess.EntryFeeCents <= 0 {
		response.BadRequest(c, "This session has no entry fee")
		return
	}

	// Already paid?
	if HasUserPaidEntry(h.db, userID, sessionID) {
		response.OK(c, gin.H{"message": "Entry already paid"})
		return
	}

	idemKey := c.GetHeader("X-Idempotency-Key")
	fee := sess.EntryFeeCents
	sellerShare := SellerShareFromEntry(fee)
	platformShare := fee - sellerShare

	err := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		if !wallet.HasSufficientBalance(tx, userID, fee) {
			return fmt.Errorf("insufficient_balance")
		}
		// Reserve then transfer to seller using escrow hold pipeline
		if err := wallet.ReserveFunds(tx, userID, fee); err != nil {
			return fmt.Errorf("reserve_failed:%s", err.Error())
		}
		// Convert user reserve → seller hold for the seller's share.
		// The platform share stays reserved/to-be-collected off-wallet via platform_fee tracking.
		if _, err := wallet.ConvertReserveToHold(tx, userID, sess.HostID, sellerShare, "live_entry_fee", sessionID.String()); err != nil {
			_ = wallet.ReleaseReservedFunds(tx, userID, fee)
			return fmt.Errorf("hold_failed:%s", err.Error())
		}
		// Release the remaining (platform share) from reserve — it has been accounted for in `platform_share_cents`.
		if err := wallet.ReleaseReservedFunds(tx, userID, platformShare); err != nil {
			slog.Warn("live-entry: failed to release platform share reserve", "error", err)
		}
		entry := LivePaidEntry{
			SessionID:          sessionID,
			UserID:             userID,
			AmountCents:        fee,
			SellerShareCents:   sellerShare,
			PlatformShareCents: platformShare,
			IdempotencyKey:     idemKey,
		}
		return tx.Create(&entry).Error
	})

	if err != nil {
		msg := err.Error()
		switch msg {
		case "insufficient_balance":
			response.BadRequest(c, "Insufficient balance for entry fee")
		default:
			response.BadRequest(c, msg)
		}
		return
	}

	freeze.LogAudit(h.db, "live_entry_fee_paid", userID, sessionID,
		fmt.Sprintf("fee_cents=%d seller_cents=%d platform_cents=%d", fee, sellerShare, platformShare))

	response.OK(c, gin.H{
		"message":        "Entry fee paid",
		"amount_cents":   fee,
		"seller_share":   sellerShare,
		"platform_share": platformShare,
	})
}

// ════════════════════════════════════════════════════════════════════════════
// Seller Subscription Plans
// ════════════════════════════════════════════════════════════════════════════

// SellerPlan enumerates live-commerce seller plans.
type SellerPlanTier string

const (
	PlanFree  SellerPlanTier = "free"
	PlanPro   SellerPlanTier = "pro"
	PlanElite SellerPlanTier = "elite"
)

// SellerPlanLimits defines session/boost limits per plan.
type SellerPlanLimits struct {
	MaxActiveSessions int     `json:"max_active_sessions"`
	BoostDiscountPct  float64 `json:"boost_discount_pct"`
	FeaturedPlacement bool    `json:"featured_placement"`
}

var SellerPlanTiers = map[SellerPlanTier]SellerPlanLimits{
	PlanFree:  {MaxActiveSessions: 2, BoostDiscountPct: 0, FeaturedPlacement: false},
	PlanPro:   {MaxActiveSessions: -1, BoostDiscountPct: 10, FeaturedPlacement: false}, // -1 = unlimited
	PlanElite: {MaxActiveSessions: -1, BoostDiscountPct: 25, FeaturedPlacement: true},
}

// GetSellerPlanLimits returns limits for the seller's plan on a given session.
func GetSellerPlanLimits(plan string) SellerPlanLimits {
	p := SellerPlanTier(plan)
	if limits, ok := SellerPlanTiers[p]; ok {
		return limits
	}
	return SellerPlanTiers[PlanFree]
}

// ════════════════════════════════════════════════════════════════════════════
// Performance Dashboard — GET /admin/live/metrics
// ════════════════════════════════════════════════════════════════════════════

// LivePerformanceMetrics is the response shape for /admin/live/metrics.
type LivePerformanceMetrics struct {
	SessionID            *uuid.UUID `json:"session_id,omitempty"`
	RevenueCents         int64      `json:"revenue_cents"`
	CommissionCents      int64      `json:"commission_cents"`
	BoostRevenueCents    int64      `json:"boost_revenue_cents"`
	EntryFeeRevenueCents int64      `json:"entry_fee_revenue_cents"`
	TotalBids            int64      `json:"total_bids"`
	BidsPerMinute        float64    `json:"bids_per_minute"`
	Viewers              int64      `json:"viewers"`
	ItemsListed          int64      `json:"items_listed"`
	ItemsSold            int64      `json:"items_sold"`
	ConversionRate       float64    `json:"conversion_rate"` // items_sold / items_listed
	GMVCents             int64      `json:"gmv_cents"`       // gross merchandise value
}

// AdminMetrics returns aggregated live performance metrics.
// Query param: session_id (optional)
func (h *LiveAuctionHandler) AdminMetrics(c *gin.Context) {
	sessionIDStr := c.Query("session_id")

	m := LivePerformanceMetrics{}
	var sessionFilter *uuid.UUID
	if sessionIDStr != "" {
		if uid, err := uuid.Parse(sessionIDStr); err == nil {
			sessionFilter = &uid
			m.SessionID = &uid
		}
	}

	// Commission totals (revenue)
	commQ := h.db.Table("live_commissions").
		Select("COALESCE(SUM(commission_cents), 0) AS c, COALESCE(SUM(final_price_cents), 0) AS gmv")
	if sessionFilter != nil {
		commQ = commQ.Where("session_id = ?", *sessionFilter)
	}
	var commRow struct{ C, Gmv int64 }
	commQ.Scan(&commRow)
	m.CommissionCents = commRow.C
	m.GMVCents = commRow.Gmv

	// Boost revenue
	boostQ := h.db.Table("live_boosts").
		Select("COALESCE(SUM(price_cents), 0) AS b").
		Where("status = ?", "active")
	if sessionFilter != nil {
		boostQ = boostQ.Where("session_id = ?", *sessionFilter)
	}
	var boostRow struct{ B int64 }
	boostQ.Scan(&boostRow)
	m.BoostRevenueCents = boostRow.B

	// Entry fee revenue
	entryQ := h.db.Table("live_paid_entries").
		Select("COALESCE(SUM(platform_share_cents), 0) AS p")
	if sessionFilter != nil {
		entryQ = entryQ.Where("session_id = ?", *sessionFilter)
	}
	var entryRow struct{ P int64 }
	entryQ.Scan(&entryRow)
	m.EntryFeeRevenueCents = entryRow.P

	m.RevenueCents = m.CommissionCents + m.BoostRevenueCents + m.EntryFeeRevenueCents

	// Bids + bids/min
	bidsQ := h.db.Table("live_bids").Select("COUNT(*) AS total")
	if sessionFilter != nil {
		bidsQ = bidsQ.Joins("JOIN live_items ON live_items.id = live_bids.item_id").
			Where("live_items.session_id = ?", *sessionFilter)
	}
	var bidRow struct{ Total int64 }
	bidsQ.Scan(&bidRow)
	m.TotalBids = bidRow.Total

	// Items listed + sold
	itemsQ := h.db.Table("live_items").
		Select("COUNT(*) AS listed, SUM(CASE WHEN status = 'sold' THEN 1 ELSE 0 END) AS sold")
	if sessionFilter != nil {
		itemsQ = itemsQ.Where("session_id = ?", *sessionFilter)
	}
	var itemsRow struct{ Listed, Sold int64 }
	itemsQ.Scan(&itemsRow)
	m.ItemsListed = itemsRow.Listed
	m.ItemsSold = itemsRow.Sold
	if m.ItemsListed > 0 {
		m.ConversionRate = float64(m.ItemsSold) / float64(m.ItemsListed)
	}

	// Bids per minute (based on session duration if single-session, else 60-min rolling)
	if sessionFilter != nil {
		var sess Session
		if err := h.db.Where("id = ?", *sessionFilter).First(&sess).Error; err == nil {
			start := sess.CreatedAt
			if sess.StartedAt != nil {
				start = *sess.StartedAt
			}
			end := time.Now()
			if sess.EndedAt != nil {
				end = *sess.EndedAt
			}
			minutes := end.Sub(start).Minutes()
			if minutes > 0 {
				m.BidsPerMinute = float64(m.TotalBids) / minutes
			}
			m.Viewers = int64(sess.ViewerCount)
		}
	}

	response.OK(c, m)
}
