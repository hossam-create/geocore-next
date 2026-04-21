package pricing

import (
	"net/http"

	"github.com/geocore-next/backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Bandit Handler ──────────────────────────────────────────────────────────────

type BanditHandler struct {
	db *gorm.DB
}

func NewBanditHandler(db *gorm.DB) *BanditHandler {
	return &BanditHandler{db: db}
}

// ── Select Price (POST /pricing/insurance/bandit) ─────────────────────────────────

func (h *BanditHandler) SelectInsurancePrice(c *gin.Context) {
	if !config.GetFlags().EnableDynamicPricing {
		c.JSON(http.StatusForbidden, gin.H{"error": "Dynamic pricing is not available"})
		return
	}

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

	// Build pricing context
	ctx, err := BuildPricingContext(h.db, uid, orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Select price via bandit engine
	result, err := SelectPrice(h.db, ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"arm_id":        result.ArmID,
		"segment":       result.Segment,
		"price_cents":   result.PriceCents,
		"price_percent": result.PricePercent,
		"algorithm":     result.Algorithm,
		"sample_value":  result.SampleValue,
		"confidence":    result.Confidence,
		"is_exploration": result.IsExploration,
		"anchor_price":  result.AnchorPrice,
		"kill_switch":   result.KillSwitchOn,
		"recommended_label": "Recommended protection for you",
	})
}

// ── Record Feedback (POST /pricing/insurance/bandit/feedback) ─────────────────────

type BanditFeedbackReq struct {
	OrderID        string  `json:"order_id" binding:"required"`
	DidBuy         bool    `json:"did_buy"`
	ClaimCostCents float64 `json:"claim_cost_cents"`
}

func (h *BanditHandler) RecordFeedback(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	uid := userID.(uuid.UUID)

	var req BanditFeedbackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	if err := RecordBanditOutcome(h.db, uid, orderID, req.DidBuy, req.ClaimCostCents); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Feedback recorded"})
}

// ── Admin: Bandit Dashboard ──────────────────────────────────────────────────────

func (h *BanditHandler) GetDashboard(c *gin.Context) {
	dashboard := GetBanditDashboard(h.db)
	c.JSON(http.StatusOK, dashboard)
}

// ── Admin: Update Bandit Config ──────────────────────────────────────────────────

type UpdateBanditConfigReq struct {
	Algorithm                *string  `json:"algorithm"`
	Epsilon                  *float64 `json:"epsilon"`
	MinPricePercent          *float64 `json:"min_price_percent"`
	MaxPricePercent          *float64 `json:"max_price_percent"`
	ConversionDropThreshold  *float64 `json:"conversion_drop_threshold"`
	SessionCooldownMinutes   *int     `json:"session_cooldown_minutes"`
	MinImpressionsBeforeExploit *int   `json:"min_impressions_before_exploit"`
	FallbackPricePercent     *float64 `json:"fallback_price_percent"`
}

func (h *BanditHandler) UpdateConfig(c *gin.Context) {
	var req UpdateBanditConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Algorithm != nil {
		updates["algorithm"] = *req.Algorithm
	}
	if req.Epsilon != nil {
		updates["epsilon"] = *req.Epsilon
	}
	if req.MinPricePercent != nil {
		updates["min_price_percent"] = *req.MinPricePercent
	}
	if req.MaxPricePercent != nil {
		updates["max_price_percent"] = *req.MaxPricePercent
	}
	if req.ConversionDropThreshold != nil {
		updates["conversion_drop_threshold"] = *req.ConversionDropThreshold
	}
	if req.SessionCooldownMinutes != nil {
		updates["session_cooldown_minutes"] = *req.SessionCooldownMinutes
	}
	if req.MinImpressionsBeforeExploit != nil {
		updates["min_impressions_before_exploit"] = *req.MinImpressionsBeforeExploit
	}
	if req.FallbackPricePercent != nil {
		updates["fallback_price_percent"] = *req.FallbackPricePercent
	}

	h.db.Model(&BanditConfig{}).Where("is_active = ?", true).Updates(updates)

	c.JSON(http.StatusOK, gin.H{"message": "Bandit config updated"})
}

// ── Admin: Kill Switch ────────────────────────────────────────────────────────────

func (h *BanditHandler) ActivateKillSwitch(c *gin.Context) {
	ActivateBanditKillSwitch(h.db, "admin_manual")
	c.JSON(http.StatusOK, gin.H{"message": "Kill switch activated — using static pricing"})
}

func (h *BanditHandler) DeactivateKillSwitch(c *gin.Context) {
	DeactivateBanditKillSwitch(h.db)
	c.JSON(http.StatusOK, gin.H{"message": "Kill switch deactivated — bandit pricing resumed"})
}

// ── Admin: Reset Segment ──────────────────────────────────────────────────────────

type ResetSegmentReq struct {
	Segment string `json:"segment" binding:"required"`
}

func (h *BanditHandler) ResetSegment(c *gin.Context) {
	var req ResetSegmentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ResetSegmentArms(h.db, req.Segment)
	c.JSON(http.StatusOK, gin.H{"message": "Segment arms reset", "segment": req.Segment})
}

// ── Admin: Check Conversion ──────────────────────────────────────────────────────

func (h *BanditHandler) CheckConversion(c *gin.Context) {
	dropped, rate := CheckConversionDrop(h.db)
	c.JSON(http.StatusOK, gin.H{
		"conversion_dropped": dropped,
		"current_rate":       rate,
	})
}
