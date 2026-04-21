package listings

import (
	"fmt"

	"github.com/geocore-next/backend/internal/subscriptions"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetSellerListingCount returns the number of active listings for a seller.
func GetSellerListingCount(db *gorm.DB, userID uuid.UUID) int64 {
	var count int64
	db.Model(&Listing{}).Where("user_id=? AND status IN ?", userID, []string{"active", "pending"}).Count(&count)
	return count
}

// GetSellerTierInfo returns the seller's subscription tier and listing limits.
func GetSellerTierInfo(db *gorm.DB, userID uuid.UUID) gin.H {
	limit := subscriptions.GetUserPlanLimits(db, userID)
	count := GetSellerListingCount(db, userID)
	tier := "free"
	if limit <= 0 || limit > subscriptions.FreePlanListingLimit {
		tier = "pro"
	}
	return gin.H{
		"tier":            tier,
		"active_listings": count,
		"listing_limit":   limit,
		"can_create":      limit <= 0 || count < int64(limit),
		"pro_benefits": []string{
			"Unlimited listings",
			"Dashboard + analytics",
			"Priority in matching",
			"Boost discounts",
		},
	}
}

// ── HTTP Handlers ────────────────────────────────────────────────────────────

func (h *Handler) GetSellerTierInfoHandler(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	info := GetSellerTierInfo(h.dbRead, userID)
	response.OK(c, info)
}

// Ensure fmt is used
var _ = fmt.Sprintf
