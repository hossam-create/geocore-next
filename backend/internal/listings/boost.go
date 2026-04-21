package listings

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/reputation"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BoostType string

const (
	BoostBasic   BoostType = "basic"
	BoostPremium BoostType = "premium"
)

var boostPrices = map[BoostType]int64{
	BoostBasic:   500,  // $5.00
	BoostPremium: 1500, // $15.00
}

var boostDurations = map[BoostType]time.Duration{
	BoostBasic:   24 * time.Hour,
	BoostPremium: 72 * time.Hour,
}

var boostScores = map[BoostType]int{
	BoostBasic:   50,
	BoostPremium: 100,
}

const maxBoostsPerSellerPerDay = 5

type ListingBoost struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ListingID  uuid.UUID `gorm:"type:uuid;not null;index" json:"listing_id"`
	SellerID   uuid.UUID `gorm:"type:uuid;not null;index" json:"seller_id"`
	BoostType  BoostType `gorm:"size:20;not null" json:"boost_type"`
	BoostScore int       `gorm:"not null" json:"boost_score"`
	ExpiresAt  time.Time `gorm:"not null;index" json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

func (ListingBoost) TableName() string { return "listing_boosts" }

func (b *ListingBoost) IsExpired() bool { return time.Now().After(b.ExpiresAt) }

// ApplyBoost creates a paid boost for a listing.
// Holds funds from seller wallet, creates boost record, updates listing boost_score.
func ApplyBoost(db *gorm.DB, listingID, sellerID uuid.UUID, plan BoostType) (*ListingBoost, error) {
	price, ok := boostPrices[plan]
	if !ok {
		return nil, fmt.Errorf("invalid boost plan: %s", plan)
	}
	duration := boostDurations[plan]
	score := boostScores[plan]

	// Trust gate: low trust users cannot boost
	if err := reputation.CheckTrustGate(db, sellerID, "boost"); err != nil {
		return nil, err
	}

	var boost ListingBoost
	err := locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		var listing Listing
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id=? AND user_id=? AND status=?", listingID, sellerID, "active").
			First(&listing).Error; err != nil {
			return fmt.Errorf("listing not found or not yours")
		}

		// Anti-abuse: no duplicate active boost on same listing
		var activeCnt int64
		tx.Model(&ListingBoost{}).Where("listing_id=? AND expires_at>?", listingID, time.Now()).Count(&activeCnt)
		if activeCnt > 0 {
			return fmt.Errorf("listing already has an active boost")
		}

		// Anti-abuse: daily seller limit
		var sellerDailyCnt int64
		tx.Model(&ListingBoost{}).Where("seller_id=? AND created_at>?", sellerID, time.Now().Truncate(24*time.Hour)).Count(&sellerDailyCnt)
		if sellerDailyCnt >= int64(maxBoostsPerSellerPerDay) {
			return fmt.Errorf("daily boost limit reached (%d)", maxBoostsPerSellerPerDay)
		}

		// Hold funds from seller
		priceFloat := float64(price) / 100.0
		_, holdErr := wallet.HoldFunds(tx, sellerID, uuid.Nil, priceFloat, "USD", "listing_boost", listingID.String())
		if holdErr != nil {
			return fmt.Errorf("payment failed: %w", holdErr)
		}

		boost = ListingBoost{
			ListingID:  listingID,
			SellerID:   sellerID,
			BoostType:  plan,
			BoostScore: score,
			ExpiresAt:  time.Now().Add(duration),
		}
		if err := tx.Create(&boost).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	slog.Info("listings: boost applied", "listing_id", listingID, "plan", plan, "score", score)
	return &boost, nil
}

// GetActiveBoostScore returns the effective boost score for a listing.
// Uses diminishing returns: score = baseScore / (1 + activeBoosts)
// This prevents stacking abuse.
func GetActiveBoostScore(db *gorm.DB, listingID uuid.UUID) int {
	var boosts []ListingBoost
	db.Where("listing_id=? AND expires_at>?", listingID, time.Now()).Find(&boosts)
	if len(boosts) == 0 {
		return 0
	}
	// Sum base scores then apply diminishing returns
	baseTotal := 0
	for _, b := range boosts {
		baseTotal += b.BoostScore
	}
	// Diminishing: divide by (1 + count-1*0.5) so stacking has reduced effect
	diminishFactor := 1.0 + float64(len(boosts)-1)*0.5
	effective := float64(baseTotal) / diminishFactor
	return int(effective + 0.5)
}

// ── HTTP Handlers ────────────────────────────────────────────────────────────

type applyBoostReq struct {
	Plan string `json:"plan" binding:"required"`
}

func (h *Handler) ApplyBoost(c *gin.Context) {
	listingID, _ := uuid.Parse(c.Param("id"))
	sellerID, _ := uuid.Parse(c.GetString("user_id"))

	var req applyBoostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "plan is required (basic/premium)")
		return
	}

	boost, err := ApplyBoost(h.dbWrite, listingID, sellerID, BoostType(req.Plan))
	if err != nil {
		slog.Error("listings: boost failed", "error", err)
		c.JSON(402, gin.H{"success": false, "error": err.Error()})
		return
	}
	response.OK(c, boost)
}

func (h *Handler) GetBoostInfo(c *gin.Context) {
	listingID, _ := uuid.Parse(c.Param("id"))
	score := GetActiveBoostScore(h.dbRead, listingID)

	var boosts []ListingBoost
	h.dbRead.Where("listing_id=? AND expires_at>?", listingID, time.Now()).Find(&boosts)

	response.OK(c, gin.H{
		"listing_id":    listingID,
		"boost_score":   score,
		"active_boosts": boosts,
		"plans": gin.H{
			"basic":   gin.H{"price_cents": boostPrices[BoostBasic], "duration": "24h", "score": boostScores[BoostBasic]},
			"premium": gin.H{"price_cents": boostPrices[BoostPremium], "duration": "72h", "score": boostScores[BoostPremium]},
		},
	})
}
