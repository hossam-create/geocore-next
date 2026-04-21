package pricing

import (
	"net/http"

	"github.com/geocore-next/backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Cross-System RL Handler ──────────────────────────────────────────────────────

type CrossHandler struct {
	db *gorm.DB
}

func NewCrossHandler(db *gorm.DB) *CrossHandler {
	return &CrossHandler{db: db}
}

// ── Select (GET /cross-rl/select/:id) ──────────────────────────────────────────────

func (h *CrossHandler) Select(c *gin.Context) {
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

	ctx, err := BuildPricingContext(h.db, uid, orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	action, err := CrossSelect(h.db, ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"price_cents":     action.PriceCents,
		"price_percent":   action.PricePercent,
		"boost_score":     action.BoostScore,
		"rec_ids":         action.RecIDs,
		"rec_strategy":    action.RecStrategy,
		"source":          action.Source,
		"source_pricing":  action.SourcePricing,
		"source_ranking":  action.SourceRanking,
		"source_recs":     action.SourceRecs,
		"confidence":      action.Confidence,
		"is_exploration":  action.IsExploration,
		"is_shadow":       action.IsShadow,
		"ux_variant":      action.UXVariant,
		"anchor_price":    action.AnchorPrice,
	})
}

// ── Feedback (POST /cross-rl/feedback) ──────────────────────────────────────────────

func (h *CrossHandler) Feedback(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	uid := userID.(uuid.UUID)

	var req CrossFeedback
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	if err := CrossRecordFeedback(h.db, uid, orderID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Feedback recorded — all Q-tables + hybrid updated"})
}

// ── Admin: Dashboard ──────────────────────────────────────────────────────────────

func (h *CrossHandler) GetDashboard(c *gin.Context) {
	dashboard := GetCrossDashboard(h.db)
	c.JSON(http.StatusOK, dashboard)
}

// ── Admin: Update Config ──────────────────────────────────────────────────────────

type UpdateCrossConfigReq struct {
	WeightGMV              *float64 `json:"weight_gmv"`
	WeightCTR              *float64 `json:"weight_ctr"`
	WeightClaimCost        *float64 `json:"weight_claim_cost"`
	WeightChurn            *float64 `json:"weight_churn"`
	LearningRate           *float64 `json:"learning_rate"`
	DiscountFactor         *float64 `json:"discount_factor"`
	Epsilon                *float64 `json:"epsilon"`
	ConfidenceThreshold    *float64 `json:"confidence_threshold"`
	MinPricePercent        *float64 `json:"min_price_percent"`
	MaxPricePercent        *float64 `json:"max_price_percent"`
	MaxBoostWithHighPrice  *int     `json:"max_boost_with_high_price"`
	HighPriceThreshold     *float64 `json:"high_price_threshold"`
	ConversionDropThreshold *float64 `json:"conversion_drop_threshold"`
	SessionCooldownMinutes *int     `json:"session_cooldown_minutes"`
	MaxSessionSteps        *int     `json:"max_session_steps"`
	AnomalyDetectionEnabled *bool   `json:"anomaly_detection_enabled"`
	RolloutPercent         *int     `json:"rollout_percent"`
	FallbackPricePercent   *float64 `json:"fallback_price_percent"`
	FallbackBoostScore     *int     `json:"fallback_boost_score"`
	FallbackRecStrategy    *string  `json:"fallback_rec_strategy"`
}

func (h *CrossHandler) UpdateConfig(c *gin.Context) {
	var req UpdateCrossConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.WeightGMV != nil {
		updates["weight_gmv"] = *req.WeightGMV
	}
	if req.WeightCTR != nil {
		updates["weight_ctr"] = *req.WeightCTR
	}
	if req.WeightClaimCost != nil {
		updates["weight_claim_cost"] = *req.WeightClaimCost
	}
	if req.WeightChurn != nil {
		updates["weight_churn"] = *req.WeightChurn
	}
	if req.LearningRate != nil {
		updates["learning_rate"] = *req.LearningRate
	}
	if req.DiscountFactor != nil {
		updates["discount_factor"] = *req.DiscountFactor
	}
	if req.Epsilon != nil {
		updates["epsilon"] = *req.Epsilon
	}
	if req.ConfidenceThreshold != nil {
		updates["confidence_threshold"] = *req.ConfidenceThreshold
	}
	if req.MinPricePercent != nil {
		updates["min_price_percent"] = *req.MinPricePercent
	}
	if req.MaxPricePercent != nil {
		updates["max_price_percent"] = *req.MaxPricePercent
	}
	if req.MaxBoostWithHighPrice != nil {
		updates["max_boost_with_high_price"] = *req.MaxBoostWithHighPrice
	}
	if req.HighPriceThreshold != nil {
		updates["high_price_threshold"] = *req.HighPriceThreshold
	}
	if req.ConversionDropThreshold != nil {
		updates["conversion_drop_threshold"] = *req.ConversionDropThreshold
	}
	if req.SessionCooldownMinutes != nil {
		updates["session_cooldown_minutes"] = *req.SessionCooldownMinutes
	}
	if req.MaxSessionSteps != nil {
		updates["max_session_steps"] = *req.MaxSessionSteps
	}
	if req.AnomalyDetectionEnabled != nil {
		updates["anomaly_detection_enabled"] = *req.AnomalyDetectionEnabled
	}
	if req.RolloutPercent != nil {
		updates["rollout_percent"] = *req.RolloutPercent
	}
	if req.FallbackPricePercent != nil {
		updates["fallback_price_percent"] = *req.FallbackPricePercent
	}
	if req.FallbackBoostScore != nil {
		updates["fallback_boost_score"] = *req.FallbackBoostScore
	}
	if req.FallbackRecStrategy != nil {
		updates["fallback_rec_strategy"] = *req.FallbackRecStrategy
	}

	h.db.Model(&CrossConfig{}).Where("is_active = ?", true).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"message": "Cross-system config updated"})
}

// ── Admin: Emergency ──────────────────────────────────────────────────────────────

func (h *CrossHandler) ActivateEmergency(c *gin.Context) {
	ActivateCrossEmergency(h.db)
	c.JSON(http.StatusOK, gin.H{"message": "Cross-system emergency activated — all AI disabled"})
}

func (h *CrossHandler) DeactivateEmergency(c *gin.Context) {
	DeactivateCrossEmergency(h.db)
	c.JSON(http.StatusOK, gin.H{"message": "Cross-system emergency deactivated"})
}

// ── Admin: Rollout ──────────────────────────────────────────────────────────────────

func (h *CrossHandler) AdvanceRollout(c *gin.Context) {
	newPercent := AdvanceCrossRollout(h.db)
	c.JSON(http.StatusOK, gin.H{"message": "Rollout advanced", "new_percent": newPercent})
}

func (h *CrossHandler) RollbackRollout(c *gin.Context) {
	newPercent := RollbackCrossRollout(h.db)
	c.JSON(http.StatusOK, gin.H{"message": "Rollout rolled back", "new_percent": newPercent})
}

// ── Admin: Save Q-Tables ──────────────────────────────────────────────────────────

func (h *CrossHandler) SaveQTables(c *gin.Context) {
	if err := SaveCrossQTables(h.db); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Cross-system Q-tables saved"})
}

// ── Admin: Conversion Check ────────────────────────────────────────────────────────

func (h *CrossHandler) CheckConversion(c *gin.Context) {
	dropped, rate := CheckCrossConversion(h.db)
	c.JSON(http.StatusOK, gin.H{
		"conversion_dropped": dropped,
		"current_rate":       rate,
	})
}
