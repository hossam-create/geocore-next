package pricing

import (
	"net/http"

	"github.com/geocore-next/backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Hybrid Handler ──────────────────────────────────────────────────────────────

type HybridHandler struct {
	db *gorm.DB
}

func NewHybridHandler(db *gorm.DB) *HybridHandler {
	return &HybridHandler{db: db}
}

// ── Select Price (GET /pricing/hybrid/:id) ────────────────────────────────────────

func (h *HybridHandler) SelectPrice(c *gin.Context) {
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

	// Hybrid decision
	decision, err := HybridSelect(h.db, ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"price_cents":        decision.PriceCents,
		"price_percent":      decision.PricePercent,
		"anchor_price":       decision.AnchorPrice,
		"source":             decision.Source,
		"confidence":         decision.Confidence,
		"is_exploration":     decision.IsExploration,
		"is_shadow":          decision.IsShadow,
		"ux_variant":         decision.UXVariant,
		"recommended_label":  decision.RecommendedLabel,
		"session_step":       decision.SessionStep,
		"guardrails_applied": decision.GuardrailsApplied,
		"clamped":            decision.Clamped,
		"rl_output":          decision.RLOutput,
		"bandit_output":      decision.BanditOutput,
		"rules_output":       decision.RulesOutput,
	})
}

// ── Record Feedback (POST /pricing/hybrid/feedback) ──────────────────────────────

func (h *HybridHandler) RecordFeedback(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	uid := userID.(uuid.UUID)

	var req HybridFeedback
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	if err := ProcessHybridFeedback(h.db, uid, orderID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Feedback recorded — all engines updated"})
}

// ── Admin: Dashboard ──────────────────────────────────────────────────────────────

func (h *HybridHandler) GetDashboard(c *gin.Context) {
	dashboard := GetHybridDashboard(h.db)
	c.JSON(http.StatusOK, dashboard)
}

// ── Admin: Update Config ──────────────────────────────────────────────────────────

type UpdateHybridConfigReq struct {
	RLConfidenceThreshold   *float64 `json:"rl_confidence_threshold"`
	BlendWeightRL           *float64 `json:"blend_weight_rl"`
	EnableSoftBlend         *bool    `json:"enable_soft_blend"`
	MinPricePercent         *float64 `json:"min_price_percent"`
	MaxPricePercent         *float64 `json:"max_price_percent"`
	EmergencyPricePercent   *float64 `json:"emergency_price_percent"`
	ConversionDropThreshold *float64 `json:"conversion_drop_threshold"`
	SessionCooldownMinutes  *int     `json:"session_cooldown_minutes"`
	MaxSessionSteps         *int     `json:"max_session_steps"`
	AnomalyDetectionEnabled *bool    `json:"anomaly_detection_enabled"`
	RolloutPercent          *int     `json:"rollout_percent"`
}

func (h *HybridHandler) UpdateConfig(c *gin.Context) {
	var req UpdateHybridConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.RLConfidenceThreshold != nil {
		updates["rl_confidence_threshold"] = *req.RLConfidenceThreshold
	}
	if req.BlendWeightRL != nil {
		updates["blend_weight_rl"] = *req.BlendWeightRL
	}
	if req.EnableSoftBlend != nil {
		updates["enable_soft_blend"] = *req.EnableSoftBlend
	}
	if req.MinPricePercent != nil {
		updates["min_price_percent"] = *req.MinPricePercent
	}
	if req.MaxPricePercent != nil {
		updates["max_price_percent"] = *req.MaxPricePercent
	}
	if req.EmergencyPricePercent != nil {
		updates["emergency_price_percent"] = *req.EmergencyPricePercent
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

	h.db.Model(&HybridConfig{}).Where("is_active = ?", true).Updates(updates)

	c.JSON(http.StatusOK, gin.H{"message": "Hybrid config updated"})
}

// ── Admin: Emergency Mode ──────────────────────────────────────────────────────────

func (h *HybridHandler) ActivateEmergency(c *gin.Context) {
	ActivateEmergencyMode(h.db)
	c.JSON(http.StatusOK, gin.H{"message": "Emergency mode activated — all AI disabled, static pricing only"})
}

func (h *HybridHandler) DeactivateEmergency(c *gin.Context) {
	DeactivateEmergencyMode(h.db)
	c.JSON(http.StatusOK, gin.H{"message": "Emergency mode deactivated — hybrid pricing resumed"})
}

// ── Admin: Advance Rollout ────────────────────────────────────────────────────────

func (h *HybridHandler) AdvanceRollout(c *gin.Context) {
	config := loadHybridConfig(h.db)
	newPercent := config.RolloutPercent
	switch newPercent {
	case 5:
		newPercent = 25
	case 25:
		newPercent = 50
	case 50:
		newPercent = 100
	default:
		if newPercent < 100 {
			newPercent = 100
		}
	}

	h.db.Model(&HybridConfig{}).Where("id = ?", config.ID).
		Update("rollout_percent", newPercent)

	c.JSON(http.StatusOK, gin.H{"message": "Rollout advanced", "new_percent": newPercent})
}

func (h *HybridHandler) RollbackRollout(c *gin.Context) {
	config := loadHybridConfig(h.db)
	newPercent := config.RolloutPercent
	switch newPercent {
	case 100:
		newPercent = 50
	case 50:
		newPercent = 25
	case 25:
		newPercent = 5
	default:
		newPercent = 5
	}

	h.db.Model(&HybridConfig{}).Where("id = ?", config.ID).
		Update("rollout_percent", newPercent)

	c.JSON(http.StatusOK, gin.H{"message": "Rollout rolled back", "new_percent": newPercent})
}

// ── Admin: Conversion Check ──────────────────────────────────────────────────────

func (h *HybridHandler) CheckConversion(c *gin.Context) {
	// Check both RL and Bandit conversion
	rlDropped, rlRate := CheckRLConversionDrop(h.db)
	banditDropped, banditRate := CheckConversionDrop(h.db)

	c.JSON(http.StatusOK, gin.H{
		"rl_conversion_dropped":    rlDropped,
		"rl_current_rate":         rlRate,
		"bandit_conversion_dropped": banditDropped,
		"bandit_current_rate":     banditRate,
	})
}
