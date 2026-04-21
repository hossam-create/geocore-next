package livestream

import (
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/freeze"
	"github.com/geocore-next/backend/internal/reputation"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Sprint 16: Creator Economy Engine
//
// Enables influencers (creators) to sell products on behalf of sellers via
// live streams and earn commissions.
//
// Layers on top of existing:
//   - Session.StreamerID + ComputeCreatorShare (Sprint 13 flywheel)
//   - LiveStreamerEarning ledger (pending/paid)
//   - GrowthReward ledger (Sprint 15 viral)
//   - Reputation system (trust scores)
//   - Wallet + Escrow (safe settlement)
//
// All features are additive, feature-flagged, idempotent, and wallet-safe.
// ════════════════════════════════════════════════════════════════════════════

// ── Feature Flags ──────────────────────────────────────────────────────────

func IsCreatorsEnabled() bool        { return envBoolDefault("ENABLE_CREATORS", true) }
func IsCreatorMatchingEnabled() bool { return envBoolDefault("ENABLE_CREATOR_MATCHING", true) }
func IsCreatorBonusesEnabled() bool  { return envBoolDefault("ENABLE_CREATOR_BONUSES", true) }

// ── Constants ──────────────────────────────────────────────────────────────

const (
	creatorMinTrustScore     = 50.0 // minimum trust to be active creator
	creatorDefaultCommission = 10.0 // default 10% commission rate
	creatorMaxCommission     = 30.0 // hard cap on commission %
	creatorMaxDealsPerSeller = 50   // anti-abuse: max active deals per seller

	// Milestone bonuses (cents)
	bonusGMV50k  int64 = 50_000  // 500 EGP at 50k EGP GMV
	bonusGMV100k int64 = 150_000 // 1,500 EGP at 100k EGP GMV
	bonusSales10 int64 = 20_000  // 200 EGP at 10 sales
	bonusSales50 int64 = 100_000 // 1,000 EGP at 50 sales

	// Matching weights
	matchWeightNiche      = 0.35
	matchWeightAudience   = 0.25
	matchWeightConversion = 0.25
	matchWeightTrust      = 0.15
)

// ════════════════════════════════════════════════════════════════════════════
// Models
// ════════════════════════════════════════════════════════════════════════════

// CreatorStatus enumerates creator lifecycle states.
type CreatorStatus string

const (
	CreatorActive    CreatorStatus = "active"
	CreatorPending   CreatorStatus = "pending"
	CreatorSuspended CreatorStatus = "suspended"
)

// Creator represents an influencer who can sell on behalf of sellers.
type Creator struct {
	ID                 uuid.UUID     `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID             uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex"                  json:"user_id"`
	DisplayName        string        `gorm:"size:100;not null"                               json:"display_name"`
	Niche              string        `gorm:"size:50;not null;index"                          json:"niche"` // fashion, tech, home, beauty, food, general
	FollowersCount     int           `gorm:"not null;default:0"                              json:"followers_count"`
	TrustScore         float64       `gorm:"type:numeric(5,2);not null;default:0"            json:"trust_score"`
	Status             CreatorStatus `gorm:"size:20;not null;default:'pending';index"       json:"status"`
	CommissionRate     float64       `gorm:"type:numeric(5,2);not null;default:10"           json:"commission_rate"` // default %
	TotalGMVCents      int64         `gorm:"not null;default:0"                              json:"total_gmv_cents"`
	TotalSales         int           `gorm:"not null;default:0"                              json:"total_sales"`
	TotalEarningsCents int64         `gorm:"not null;default:0"                              json:"total_earnings_cents"`
	CreatedAt          time.Time     `gorm:"not null;index"                                  json:"created_at"`
	UpdatedAt          time.Time     `gorm:"not null"                                        json:"updated_at"`
}

func (Creator) TableName() string { return "live_creators" }

// DealStatus enumerates creator-seller deal states.
type DealStatus string

const (
	DealPending  DealStatus = "pending"
	DealActive   DealStatus = "active"
	DealRejected DealStatus = "rejected"
	DealExpired  DealStatus = "expired"
)

// CreatorDeal represents a partnership between a seller and a creator.
type CreatorDeal struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SellerID       uuid.UUID  `gorm:"type:uuid;not null;index:idx_deal_seller"        json:"seller_id"`
	CreatorID      uuid.UUID  `gorm:"type:uuid;not null;index:idx_deal_creator"       json:"creator_id"`
	CommissionRate float64    `gorm:"type:numeric(5,2);not null"                      json:"commission_rate"` // overrides creator default
	Status         DealStatus `gorm:"size:20;not null;default:'pending';index"        json:"status"`
	ExpiresAt      *time.Time `                                                        json:"expires_at,omitempty"`
	CreatedAt      time.Time  `gorm:"not null"                                        json:"created_at"`
	UpdatedAt      time.Time  `gorm:"not null"                                        json:"updated_at"`
}

func (CreatorDeal) TableName() string { return "live_creator_deals" }

// CreatorEarning records a single earning event for a creator.
// Unique on (creator_id, order_id) — idempotent.
type CreatorEarning struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	CreatorID       uuid.UUID  `gorm:"type:uuid;not null;index"                        json:"creator_id"`
	SellerID        uuid.UUID  `gorm:"type:uuid;not null;index"                        json:"seller_id"`
	SessionID       uuid.UUID  `gorm:"type:uuid;not null;index"                        json:"session_id"`
	OrderID         uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"                  json:"order_id"`
	ItemID          uuid.UUID  `gorm:"type:uuid;not null;index"                        json:"item_id"`
	PriceCents      int64      `gorm:"not null"                                        json:"price_cents"`
	CommissionPct   float64    `gorm:"type:numeric(5,2);not null"                      json:"commission_pct"`
	CommissionCents int64      `gorm:"not null"                                        json:"commission_cents"`
	Status          string     `gorm:"size:20;not null;default:'pending';index"        json:"status"` // pending, paid, voided
	CreatedAt       time.Time  `gorm:"not null;index"                                  json:"created_at"`
	PaidAt          *time.Time `                                                        json:"paid_at,omitempty"`
}

func (CreatorEarning) TableName() string { return "live_creator_earnings" }

// CreatorMilestone records a milestone bonus awarded to a creator.
// Unique on (creator_id, milestone_type, milestone_value) — idempotent.
type CreatorMilestone struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	CreatorID      uuid.UUID `gorm:"type:uuid;not null;index"                        json:"creator_id"`
	MilestoneType  string    `gorm:"size:30;not null"                                json:"milestone_type"` // gmv, sales
	MilestoneValue int64     `gorm:"not null"                                        json:"milestone_value"`
	BonusCents     int64     `gorm:"not null"                                        json:"bonus_cents"`
	GrantedAt      time.Time `gorm:"not null"                                        json:"granted_at"`
}

func (CreatorMilestone) TableName() string { return "live_creator_milestones" }

// ════════════════════════════════════════════════════════════════════════════
// 1. Creator Onboarding
// ════════════════════════════════════════════════════════════════════════════

// ApplyCreator creates a creator profile for a user (pending until approved).
func ApplyCreator(db *gorm.DB, userID uuid.UUID, displayName, niche string, followersCount int) (*Creator, error) {
	if !IsCreatorsEnabled() {
		return nil, fmt.Errorf("creator system disabled")
	}
	if displayName == "" || niche == "" {
		return nil, fmt.Errorf("display_name and niche are required")
	}
	// Check if already a creator
	var existing Creator
	if err := db.Where("user_id = ?", userID).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("already a creator")
	}
	trustScore := reputation.GetOverallScore(db, userID)
	status := CreatorPending
	if trustScore >= creatorMinTrustScore {
		status = CreatorActive // auto-approve if trust is high enough
	}
	c := &Creator{
		UserID:         userID,
		DisplayName:    displayName,
		Niche:          niche,
		FollowersCount: followersCount,
		TrustScore:     trustScore,
		Status:         status,
		CommissionRate: creatorDefaultCommission,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := db.Create(c).Error; err != nil {
		return nil, err
	}
	freeze.LogAudit(db, "creator_applied", userID, c.ID,
		fmt.Sprintf("niche=%s followers=%d trust=%.1f status=%s", niche, followersCount, trustScore, status))
	return c, nil
}

// GetCreator retrieves a creator by ID.
func GetCreator(db *gorm.DB, id uuid.UUID) (*Creator, error) {
	var c Creator
	if err := db.Where("id = ?", id).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// GetCreatorByUser retrieves a creator by user ID.
func GetCreatorByUser(db *gorm.DB, userID uuid.UUID) (*Creator, error) {
	var c Creator
	if err := db.Where("user_id = ?", userID).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// GetTopCreators returns creators sorted by GMV (descending).
func GetTopCreators(db *gorm.DB, limit int) ([]Creator, error) {
	if limit <= 0 {
		limit = 20
	}
	var creators []Creator
	err := db.Where("status = ?", CreatorActive).
		Order("total_gmv_cents DESC").
		Limit(limit).
		Find(&creators).Error
	return creators, err
}

// RefreshCreatorTrust re-computes a creator's trust score from the reputation system.
func RefreshCreatorTrust(db *gorm.DB, creatorID uuid.UUID) error {
	var c Creator
	if err := db.Where("id = ?", creatorID).First(&c).Error; err != nil {
		return err
	}
	score := reputation.GetOverallScore(db, c.UserID)
	updates := map[string]interface{}{
		"trust_score": score,
		"updated_at":  time.Now(),
	}
	// Auto-suspend if below threshold
	if score < creatorMinTrustScore && c.Status == CreatorActive {
		updates["status"] = CreatorSuspended
		slog.Warn("creator: suspended due to low trust", "creator_id", creatorID, "score", score)
	}
	// Auto-reactivate if trust recovered
	if score >= creatorMinTrustScore && c.Status == CreatorSuspended {
		updates["status"] = CreatorActive
		slog.Info("creator: reactivated", "creator_id", creatorID, "score", score)
	}
	return db.Model(&c).Updates(updates).Error
}

// ════════════════════════════════════════════════════════════════════════════
// Helper: resolve streamer user ID → creator ID
// ════════════════════════════════════════════════════════════════════════════

// resolveCreatorID looks up the Creator ID from a streamer's user ID.
// Returns nil if the user is not a registered creator.
func resolveCreatorID(db *gorm.DB, streamerUserID uuid.UUID) *uuid.UUID {
	var c Creator
	if err := db.Where("user_id = ?", streamerUserID).Select("id").First(&c).Error; err != nil {
		return nil
	}
	return &c.ID
}

// ════════════════════════════════════════════════════════════════════════════
// 2. Seller ↔ Creator Connection (Deals)
// ════════════════════════════════════════════════════════════════════════════

// InviteCreator creates a pending deal from a seller to a creator.
func InviteCreator(db *gorm.DB, sellerID, creatorID uuid.UUID, commissionRate float64) (*CreatorDeal, error) {
	if !IsCreatorsEnabled() {
		return nil, fmt.Errorf("creator system disabled")
	}
	// Validate creator exists and is active
	var c Creator
	if err := db.Where("id = ? AND status = ?", creatorID, CreatorActive).First(&c).Error; err != nil {
		return nil, fmt.Errorf("creator not found or not active")
	}
	// Anti-abuse: cap deals per seller
	var dealCount int64
	db.Model(&CreatorDeal{}).Where("seller_id = ? AND status IN ?", sellerID, []DealStatus{DealPending, DealActive}).Count(&dealCount)
	if dealCount >= int64(creatorMaxDealsPerSeller) {
		return nil, fmt.Errorf("max active deals reached for this seller")
	}
	// Validate commission rate
	if commissionRate <= 0 {
		commissionRate = c.CommissionRate // use creator default
	}
	if commissionRate > creatorMaxCommission {
		return nil, fmt.Errorf("commission rate cannot exceed %.0f%%", creatorMaxCommission)
	}
	// Check for existing deal (same pair)
	var existing CreatorDeal
	if err := db.Where("seller_id = ? AND creator_id = ? AND status IN ?", sellerID, creatorID,
		[]DealStatus{DealPending, DealActive}).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("deal already exists (status=%s)", existing.Status)
	}
	deal := &CreatorDeal{
		SellerID:       sellerID,
		CreatorID:      creatorID,
		CommissionRate: commissionRate,
		Status:         DealPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := db.Create(deal).Error; err != nil {
		return nil, err
	}
	freeze.LogAudit(db, "creator_deal_invited", sellerID, deal.ID,
		fmt.Sprintf("creator=%s commission=%.1f%%", creatorID, commissionRate))
	return deal, nil
}

// AcceptDeal transitions a pending deal to active (creator accepts).
func AcceptDeal(db *gorm.DB, dealID, creatorID uuid.UUID) (*CreatorDeal, error) {
	if !IsCreatorsEnabled() {
		return nil, fmt.Errorf("creator system disabled")
	}
	var deal CreatorDeal
	if err := db.Where("id = ? AND creator_id = ? AND status = ?", dealID, creatorID, DealPending).
		First(&deal).Error; err != nil {
		return nil, fmt.Errorf("pending deal not found")
	}
	now := time.Now()
	if err := db.Model(&deal).Updates(map[string]interface{}{
		"status":     DealActive,
		"updated_at": now,
	}).Error; err != nil {
		return nil, err
	}
	deal.Status = DealActive
	freeze.LogAudit(db, "creator_deal_accepted", creatorID, dealID,
		fmt.Sprintf("seller=%s commission=%.1f%%", deal.SellerID, deal.CommissionRate))
	return &deal, nil
}

// RejectDeal transitions a pending deal to rejected (creator declines).
func RejectDeal(db *gorm.DB, dealID, creatorID uuid.UUID) error {
	var deal CreatorDeal
	if err := db.Where("id = ? AND creator_id = ? AND status = ?", dealID, creatorID, DealPending).
		First(&deal).Error; err != nil {
		return fmt.Errorf("pending deal not found")
	}
	now := time.Now()
	return db.Model(&deal).Updates(map[string]interface{}{
		"status":     DealRejected,
		"updated_at": now,
	}).Error
}

// GetActiveDeal returns the active deal between a seller and creator (if any).
func GetActiveDeal(db *gorm.DB, sellerID, creatorID uuid.UUID) (*CreatorDeal, error) {
	var deal CreatorDeal
	err := db.Where("seller_id = ? AND creator_id = ? AND status = ?", sellerID, creatorID, DealActive).
		First(&deal).Error
	if err != nil {
		return nil, err
	}
	return &deal, nil
}

// GetCreatorDeals returns all deals for a creator.
func GetCreatorDeals(db *gorm.DB, creatorID uuid.UUID) ([]CreatorDeal, error) {
	var deals []CreatorDeal
	err := db.Where("creator_id = ?", creatorID).Order("created_at DESC").Find(&deals).Error
	return deals, err
}

// GetSellerDeals returns all deals for a seller.
func GetSellerDeals(db *gorm.DB, sellerID uuid.UUID) ([]CreatorDeal, error) {
	var deals []CreatorDeal
	err := db.Where("seller_id = ?", sellerID).Order("created_at DESC").Find(&deals).Error
	return deals, err
}

// CanCreatorSellItem checks if a creator has an active deal with the item's seller.
func CanCreatorSellItem(db *gorm.DB, creatorUserID, sellerID uuid.UUID) bool {
	if !IsCreatorsEnabled() {
		return false
	}
	c, err := GetCreatorByUser(db, creatorUserID)
	if err != nil || c.Status != CreatorActive {
		return false
	}
	_, err = GetActiveDeal(db, sellerID, c.ID)
	return err == nil
}

// ════════════════════════════════════════════════════════════════════════════
// 3. Revenue Split Engine
// ════════════════════════════════════════════════════════════════════════════

// RevenueSplitResult holds the breakdown of a settlement split.
type RevenueSplitResult struct {
	PlatformFeeCents  int64   `json:"platform_fee_cents"`
	CreatorCommCents  int64   `json:"creator_commission_cents"`
	SellerAmountCents int64   `json:"seller_amount_cents"`
	CreatorCommPct    float64 `json:"creator_commission_pct"`
}

// SplitRevenue divides the final price into platform fee, creator commission,
// and seller amount. Uses the deal's commission rate if a deal exists,
// otherwise falls back to the creator's default rate.
//
// Example: 1000 EGP item, 10% platform fee, 10% creator commission
//
//	platform: 100 EGP (10%)
//	creator:  100 EGP (10%)
//	seller:   800 EGP
func SplitRevenue(db *gorm.DB, finalPriceCents int64, sellerID uuid.UUID, creatorID *uuid.UUID) RevenueSplitResult {
	result := RevenueSplitResult{}

	if !IsCreatorsEnabled() || creatorID == nil {
		// No creator → seller gets everything minus platform fee
		platformFee := int64(float64(finalPriceCents) * 0.10) // 10% platform
		result = RevenueSplitResult{
			PlatformFeeCents:  platformFee,
			CreatorCommCents:  0,
			SellerAmountCents: finalPriceCents - platformFee,
		}
		return result
	}

	// Find commission rate from active deal
	commRate := creatorDefaultCommission
	if db != nil {
		deal, err := GetActiveDeal(db, sellerID, *creatorID)
		if err == nil && deal.CommissionRate > 0 {
			commRate = deal.CommissionRate
		}
	}

	// Platform fee: 10% of final price
	platformFee := int64(float64(finalPriceCents) * 0.10)

	// Creator commission: commRate% of (final price - platform fee)
	netAfterPlatform := finalPriceCents - platformFee
	creatorComm := int64(float64(netAfterPlatform) * commRate / 100.0)

	// Seller gets the rest
	sellerAmount := finalPriceCents - platformFee - creatorComm
	if sellerAmount < 0 {
		sellerAmount = 0
	}

	result = RevenueSplitResult{
		PlatformFeeCents:  platformFee,
		CreatorCommCents:  creatorComm,
		SellerAmountCents: sellerAmount,
		CreatorCommPct:    commRate,
	}
	return result
}

// RecordCreatorEarning persists a creator earning entry (idempotent on order_id).
func RecordCreatorEarning(db *gorm.DB, creatorID, sellerID, sessionID, orderID, itemID uuid.UUID,
	priceCents int64, commPct float64, commCents int64) error {
	if commCents <= 0 {
		return nil
	}
	row := CreatorEarning{
		CreatorID:       creatorID,
		SellerID:        sellerID,
		SessionID:       sessionID,
		OrderID:         orderID,
		ItemID:          itemID,
		PriceCents:      priceCents,
		CommissionPct:   commPct,
		CommissionCents: commCents,
		Status:          "pending",
		CreatedAt:       time.Now(),
	}
	if err := db.Create(&row).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE") {
			return nil // idempotent
		}
		return err
	}
	// Update creator aggregate stats (non-blocking)
	go updateCreatorStats(db, creatorID, priceCents, commCents)
	return nil
}

// updateCreatorStats increments GMV, sales, and earnings counters.
func updateCreatorStats(db *gorm.DB, creatorID uuid.UUID, gmvCents, earningCents int64) {
	err := db.Model(&Creator{}).Where("id = ?", creatorID).Updates(map[string]interface{}{
		"total_gmv_cents":      gorm.Expr("total_gmv_cents + ?", gmvCents),
		"total_sales":          gorm.Expr("total_sales + 1"),
		"total_earnings_cents": gorm.Expr("total_earnings_cents + ?", earningCents),
		"updated_at":           time.Now(),
	}).Error
	if err != nil {
		slog.Error("creator: failed to update stats", "creator_id", creatorID, "error", err)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// 4. Creator Live Selling Gate
// ════════════════════════════════════════════════════════════════════════════

// ValidateCreatorSession checks that a creator can host a session selling
// items from a given seller. Returns the deal if valid, error otherwise.
func ValidateCreatorSession(db *gorm.DB, creatorUserID, sellerID uuid.UUID) (*CreatorDeal, error) {
	if !IsCreatorsEnabled() {
		return nil, fmt.Errorf("creator system disabled")
	}
	c, err := GetCreatorByUser(db, creatorUserID)
	if err != nil {
		return nil, fmt.Errorf("not a creator")
	}
	if c.Status != CreatorActive {
		return nil, fmt.Errorf("creator not active (status=%s)", c.Status)
	}
	if c.TrustScore < creatorMinTrustScore {
		return nil, fmt.Errorf("creator trust score too low (%.1f < %.1f)", c.TrustScore, creatorMinTrustScore)
	}
	deal, err := GetActiveDeal(db, sellerID, c.ID)
	if err != nil {
		return nil, fmt.Errorf("no active deal with seller — cannot sell their items")
	}
	// Check deal expiry
	if deal.ExpiresAt != nil && deal.ExpiresAt.Before(time.Now()) {
		db.Model(deal).Update("status", DealExpired)
		return nil, fmt.Errorf("deal expired")
	}
	return deal, nil
}

// ════════════════════════════════════════════════════════════════════════════
// 5. Creator Performance Analytics
// ════════════════════════════════════════════════════════════════════════════

// CreatorAnalytics holds performance metrics for a creator.
type CreatorAnalytics struct {
	CreatorID            uuid.UUID `json:"creator_id"`
	DisplayName          string    `json:"display_name"`
	Niche                string    `json:"niche"`
	TrustScore           float64   `json:"trust_score"`
	TotalGMVCents        int64     `json:"total_gmv_cents"`
	TotalSales           int       `json:"total_sales"`
	TotalEarningsCents   int64     `json:"total_earnings_cents"`
	ConversionRate       float64   `json:"conversion_rate"` // sales / sessions hosted
	AvgBidsPerSession    float64   `json:"avg_bids_per_session"`
	ActiveDeals          int       `json:"active_deals"`
	PendingEarningsCents int64     `json:"pending_earnings_cents"`
	PaidEarningsCents    int64     `json:"paid_earnings_cents"`
}

// GetCreatorAnalytics computes full analytics for a creator.
func GetCreatorAnalytics(db *gorm.DB, creatorID uuid.UUID) (*CreatorAnalytics, error) {
	var c Creator
	if err := db.Where("id = ?", creatorID).First(&c).Error; err != nil {
		return nil, err
	}
	a := &CreatorAnalytics{
		CreatorID:          c.ID,
		DisplayName:        c.DisplayName,
		Niche:              c.Niche,
		TrustScore:         c.TrustScore,
		TotalGMVCents:      c.TotalGMVCents,
		TotalSales:         c.TotalSales,
		TotalEarningsCents: c.TotalEarningsCents,
	}

	// Active deals count
	var activeDeals int64
	db.Model(&CreatorDeal{}).Where("creator_id = ? AND status = ?", creatorID, DealActive).Count(&activeDeals)
	a.ActiveDeals = int(activeDeals)

	// Earnings breakdown
	var pending, paid int64
	db.Model(&CreatorEarning{}).Where("creator_id = ? AND status = ?", creatorID, "pending").
		Select("COALESCE(SUM(commission_cents), 0)").Scan(&pending)
	db.Model(&CreatorEarning{}).Where("creator_id = ? AND status = ?", creatorID, "paid").
		Select("COALESCE(SUM(commission_cents), 0)").Scan(&paid)
	a.PendingEarningsCents = pending
	a.PaidEarningsCents = paid

	// Conversion rate: sales / sessions where creator was streamer
	var sessionsHosted int64
	db.Table("livestream_sessions").Where("streamer_id = ?", c.UserID).Count(&sessionsHosted)
	if sessionsHosted > 0 {
		a.ConversionRate = float64(c.TotalSales) / float64(sessionsHosted)
	}

	// Avg bids per session
	var totalBids int64
	db.Table("live_bids lb").
		Joins("JOIN live_items li ON li.id = lb.item_id").
		Joins("JOIN livestream_sessions s ON s.id = li.session_id").
		Where("s.streamer_id = ?", c.UserID).
		Count(&totalBids)
	if sessionsHosted > 0 {
		a.AvgBidsPerSession = float64(totalBids) / float64(sessionsHosted)
	}

	return a, nil
}

// ════════════════════════════════════════════════════════════════════════════
// 6. Smart Creator Matching
// ════════════════════════════════════════════════════════════════════════════

// CreatorMatchScore holds a creator and their match score for an item.
type CreatorMatchScore struct {
	Creator         Creator `json:"creator"`
	Score           float64 `json:"score"`
	NicheMatch      float64 `json:"niche_match"`
	AudienceMatch   float64 `json:"audience_match"`
	ConversionMatch float64 `json:"conversion_match"`
	TrustMatch      float64 `json:"trust_match"`
}

// FindBestCreatorsForItem ranks creators for a given item based on niche,
// audience size, past conversion, and trust score.
func FindBestCreatorsForItem(db *gorm.DB, itemID uuid.UUID, limit int) ([]CreatorMatchScore, error) {
	if !IsCreatorMatchingEnabled() {
		return nil, fmt.Errorf("creator matching disabled")
	}
	if limit <= 0 {
		limit = 10
	}

	// Get item's category from its listing (if linked)
	var itemCategory string
	var item LiveItem
	if err := db.Where("id = ?", itemID).First(&item).Error; err == nil && item.ListingID != nil {
		var cat string
		db.Table("listings").Where("id = ?", item.ListingID).Select("category").Scan(&cat)
		if cat != "" {
			itemCategory = cat
		}
	}

	// Get all active creators
	var creators []Creator
	if err := db.Where("status = ?", CreatorActive).Find(&creators).Error; err != nil {
		return nil, err
	}

	scores := make([]CreatorMatchScore, 0, len(creators))
	for _, c := range creators {
		// Niche match: 1.0 if exact, 0.5 if "general", 0.0 otherwise
		nicheScore := 0.0
		if itemCategory != "" && strings.EqualFold(c.Niche, itemCategory) {
			nicheScore = 1.0
		} else if strings.EqualFold(c.Niche, "general") {
			nicheScore = 0.5
		}

		// Audience match: log-scaled followers (capped at 1.0)
		audienceScore := math.Log1p(float64(c.FollowersCount)) / math.Log1p(100000) // 100k followers = ~1.0
		if audienceScore > 1.0 {
			audienceScore = 1.0
		}

		// Conversion match: based on total sales / max(1, sessions)
		conversionScore := 0.0
		var sessionsHosted int64
		db.Table("livestream_sessions").Where("streamer_id = ?", c.UserID).Count(&sessionsHosted)
		if sessionsHosted > 0 {
			conversionScore = math.Min(float64(c.TotalSales)/float64(sessionsHosted), 1.0)
		}

		// Trust match: normalized to 0-1
		trustScore := math.Min(c.TrustScore/100.0, 1.0)

		// Weighted composite
		total := matchWeightNiche*nicheScore +
			matchWeightAudience*audienceScore +
			matchWeightConversion*conversionScore +
			matchWeightTrust*trustScore

		scores = append(scores, CreatorMatchScore{
			Creator:         c,
			Score:           total,
			NicheMatch:      nicheScore,
			AudienceMatch:   audienceScore,
			ConversionMatch: conversionScore,
			TrustMatch:      trustScore,
		})
	}

	// Sort descending by score
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	if len(scores) > limit {
		scores = scores[:limit]
	}
	return scores, nil
}

// ════════════════════════════════════════════════════════════════════════════
// 7. Creator Incentives & Milestones
// ════════════════════════════════════════════════════════════════════════════

// CheckCreatorMilestones evaluates and grants milestone bonuses.
// Idempotent: unique on (creator_id, milestone_type, milestone_value).
func CheckCreatorMilestones(db *gorm.DB, creatorID uuid.UUID) {
	if !IsCreatorBonusesEnabled() {
		return
	}
	var c Creator
	if err := db.Where("id = ?", creatorID).First(&c).Error; err != nil {
		return
	}

	milestones := []struct {
		mtype string
		value int64
		bonus int64
		check int64
	}{
		{"gmv", 50_000_00, bonusGMV50k, c.TotalGMVCents},   // 50k EGP GMV
		{"gmv", 100_000_00, bonusGMV100k, c.TotalGMVCents}, // 100k EGP GMV
		{"sales", 10, bonusSales10, int64(c.TotalSales)},   // 10 sales
		{"sales", 50, bonusSales50, int64(c.TotalSales)},   // 50 sales
	}

	for _, m := range milestones {
		if m.check < m.value {
			continue
		}
		// Check if already granted (idempotent)
		var count int64
		db.Model(&CreatorMilestone{}).Where("creator_id = ? AND milestone_type = ? AND milestone_value = ?",
			creatorID, m.mtype, m.value).Count(&count)
		if count > 0 {
			continue
		}
		row := CreatorMilestone{
			CreatorID:      creatorID,
			MilestoneType:  m.mtype,
			MilestoneValue: m.value,
			BonusCents:     m.bonus,
			GrantedAt:      time.Now(),
		}
		if err := db.Create(&row).Error; err != nil {
			slog.Error("creator: failed to grant milestone", "creator_id", creatorID, "error", err)
			continue
		}
		// Grant bonus as GrowthReward (reuses Sprint 15 ledger)
		_, _ = GrantReward(db, c.UserID, "creator_milestone", m.mtype,
			fmt.Sprintf("%s:%d:%s", creatorID, m.value, m.mtype), m.bonus)
		slog.Info("creator: milestone granted", "creator_id", creatorID,
			"milestone", fmt.Sprintf("%s=%d", m.mtype, m.value), "bonus_cents", m.bonus)
		freeze.LogAudit(db, "creator_milestone_granted", c.UserID, row.ID,
			fmt.Sprintf("type=%s value=%d bonus=%d", m.mtype, m.value, m.bonus))
	}
}

// ════════════════════════════════════════════════════════════════════════════
// 8. Trust & Fraud Rules
// ════════════════════════════════════════════════════════════════════════════

// ReduceCreatorTrust lowers a creator's trust score (e.g. on disputes).
func ReduceCreatorTrust(db *gorm.DB, creatorID uuid.UUID, penalty float64) error {
	var c Creator
	if err := db.Where("id = ?", creatorID).First(&c).Error; err != nil {
		return err
	}
	newScore := c.TrustScore - penalty
	if newScore < 0 {
		newScore = 0
	}
	updates := map[string]interface{}{
		"trust_score": newScore,
		"updated_at":  time.Now(),
	}
	if newScore < creatorMinTrustScore && c.Status == CreatorActive {
		updates["status"] = CreatorSuspended
		slog.Warn("creator: suspended due to trust penalty", "creator_id", creatorID, "new_score", newScore)
	}
	return db.Model(&c).Updates(updates).Error
}

// ════════════════════════════════════════════════════════════════════════════
// 9. Viral Loop Integration
// ════════════════════════════════════════════════════════════════════════════

// GetCreatorReferralCode returns a live invite code for a creator's session.
// Creators earn extra when their referred users bid/win.
func GetCreatorReferralCode(db *gorm.DB, creatorUserID, sessionID uuid.UUID) (string, error) {
	if !IsCreatorsEnabled() || !IsLiveInvitesEnabled() {
		return "", fmt.Errorf("disabled")
	}
	c, err := GetCreatorByUser(db, creatorUserID)
	if err != nil {
		return "", fmt.Errorf("not a creator")
	}
	inv, err := CreateLiveInvite(db, c.UserID, sessionID)
	if err != nil {
		return "", err
	}
	return inv.InviteCode, nil
}

// ════════════════════════════════════════════════════════════════════════════
// 10. Admin: Creator Payout Summary
// ════════════════════════════════════════════════════════════════════════════

// CreatorPayoutSummary is the admin payload for pending creator payouts.
type CreatorPayoutSummary struct {
	CreatorID            uuid.UUID `json:"creator_id"`
	DisplayName          string    `json:"display_name"`
	PendingEarningsCents int64     `json:"pending_earnings_cents"`
	PendingCount         int       `json:"pending_count"`
}

// GetPendingCreatorPayouts returns all creators with pending earnings.
func GetPendingCreatorPayouts(db *gorm.DB) ([]CreatorPayoutSummary, error) {
	var results []CreatorPayoutSummary
	err := db.Table("live_creator_earnings ce").
		Select("ce.creator_id, c.display_name, SUM(ce.commission_cents) as pending_earnings_cents, COUNT(*) as pending_count").
		Joins("JOIN live_creators c ON c.id = ce.creator_id").
		Where("ce.status = ?", "pending").
		Group("ce.creator_id, c.display_name").
		Order("pending_earnings_cents DESC").
		Scan(&results).Error
	return results, err
}
