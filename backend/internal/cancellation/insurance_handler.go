package cancellation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ── Purchase Insurance ──────────────────────────────────────────────────────────

type PurchaseInsuranceRequest struct {
	CoverageType CoverageType `json:"coverage_type" binding:"required"`
}

func (h *Handler) PurchaseInsurance(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	uid := userID.(uuid.UUID)

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var req PurchaseInsuranceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "coverage_type is required (basic, plus, premium)"})
		return
	}

	result, err := PurchaseInsurance(h.db, orderID, uid, req.CoverageType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ── Get Insurance for Order ────────────────────────────────────────────────────

func (h *Handler) GetOrderInsurance(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var insurance OrderInsurance
	if err := h.db.Where("order_id = ?", orderID).First(&insurance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No insurance found for this order"})
		return
	}

	c.JSON(http.StatusOK, insurance)
}

// ── Get Insurance Pricing ──────────────────────────────────────────────────────

func (h *Handler) GetInsurancePricing(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	// Load order total for price calculation
	var ord struct {
		Total float64
	}
	if err := h.db.Table("orders").Select("total").Where("id = ?", orderID).First(&ord).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	type tierPrice struct {
		CoverageType  CoverageType `json:"coverage_type"`
		Label         string       `json:"label"`
		Description   string       `json:"description"`
		PriceCents    int64        `json:"price_cents"`
		MaxFeeCovered float64      `json:"max_fee_covered_pct"`
	}

	totalCents := int64(ord.Total * 100)
	prices := make([]tierPrice, len(InsuranceTiers))
	for i, t := range InsuranceTiers {
		priceCents := int64(float64(totalCents) * t.PricePercent / 100.0)
		if priceCents < 50 {
			priceCents = 50
		}
		prices[i] = tierPrice{
			CoverageType:  t.CoverageType,
			Label:         t.Label,
			Description:   t.Description,
			PriceCents:    priceCents,
			MaxFeeCovered: t.MaxFeeCoveredPct,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"order_id":    orderID,
		"total_cents": totalCents,
		"tiers":       prices,
		"enabled":     IsInsuranceEnabled(),
	})
}

// ── Admin: Insurance Stats ──────────────────────────────────────────────────────

func (h *Handler) GetAdminInsuranceStats(c *gin.Context) {
	stats := AdminInsuranceStats{
		ByCoverageType: make(map[string]int64),
	}

	h.db.Table("order_insurances").Count(&stats.TotalPurchased)

	var revenue struct {
		Total int64
	}
	h.db.Table("order_insurances").
		Select("COALESCE(SUM(price_cents), 0) as total").
		Scan(&revenue)
	stats.TotalRevenueCents = revenue.Total

	h.db.Table("order_insurances").Where("is_used = ?", true).Count(&stats.TotalUsed)
	h.db.Table("order_insurances").Where("first_order_free = ?", true).Count(&stats.FirstOrderFreeCount)

	if stats.TotalPurchased > 0 {
		stats.UsageRate = float64(stats.TotalUsed) / float64(stats.TotalPurchased)
	}

	var avgPrice struct {
		Avg int64
	}
	h.db.Table("order_insurances").
		Select("COALESCE(AVG(price_cents), 0) as avg").
		Scan(&avgPrice)
	stats.AvgPriceCents = avgPrice.Avg

	// By coverage type
	type typeCount struct {
		CoverageType string
		Count        int64
	}
	var counts []typeCount
	h.db.Table("order_insurances").
		Select("coverage_type, COUNT(*) as count").
		Group("coverage_type").
		Scan(&counts)
	for _, tc := range counts {
		stats.ByCoverageType[tc.CoverageType] = tc.Count
	}

	c.JSON(http.StatusOK, stats)
}

// ── Admin: Disable Insurance for User ───────────────────────────────────────────

func (h *Handler) DisableInsuranceForUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Deactivate all active insurances for the user
	h.db.Model(&OrderInsurance{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Updates(map[string]interface{}{"is_active": false})

	c.JSON(http.StatusOK, gin.H{"message": "Insurance disabled for user"})
}
