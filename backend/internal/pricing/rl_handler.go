package pricing

import (
	"net/http"

	"github.com/geocore-next/backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── RL Handler ──────────────────────────────────────────────────────────────────

type RLHandler struct {
	db *gorm.DB
}

func NewRLHandler(db *gorm.DB) *RLHandler {
	return &RLHandler{db: db}
}

// ── Select Price (POST /rl/select) ────────────────────────────────────────────────

func (h *RLHandler) SelectPrice(c *gin.Context) {
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

	// Select action via RL engine
	result, err := RLSelectAction(h.db, ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"price_cents":       result.PriceCents,
		"price_percent":     result.PricePercent,
		"action":            result.Action.Label,
		"ux_variant":        result.UXVariant,
		"state_key":         result.StateKey,
		"confidence":        result.Confidence,
		"is_exploration":    result.IsExploration,
		"is_shadow":         result.IsShadow,
		"session_step":      result.SessionStep,
		"anchor_price":      result.AnchorPrice,
		"kill_switch":       result.KillSwitchOn,
		"recommended_label": result.RecommendedLabel,
	})
}

// ── Record Feedback (POST /rl/feedback) ──────────────────────────────────────────

type RLFeedbackReq struct {
	OrderID        string  `json:"order_id" binding:"required"`
	DidBuy         bool    `json:"did_buy"`
	DidClaim       bool    `json:"did_claim"`
	DidChurn       bool    `json:"did_churn"`
	ClaimCostCents float64 `json:"claim_cost_cents"`
}

func (h *RLHandler) RecordFeedback(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	uid := userID.(uuid.UUID)

	var req RLFeedbackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	if err := RLRecordFeedback(h.db, uid, orderID, req.DidBuy, req.DidClaim, req.DidChurn, req.ClaimCostCents); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Feedback recorded — Q-table updated"})
}

// ── Admin: RL Dashboard ──────────────────────────────────────────────────────────

func (h *RLHandler) GetDashboard(c *gin.Context) {
	dashboard := GetRLDashboard(h.db)
	c.JSON(http.StatusOK, dashboard)
}

// ── Admin: Update RL Config ──────────────────────────────────────────────────────

type UpdateRLConfigReq struct {
	Algorithm                *string  `json:"algorithm"`
	LearningRate             *float64 `json:"learning_rate"`
	DiscountFactor           *float64 `json:"discount_factor"`
	Epsilon                  *float64 `json:"epsilon"`
	EpsilonDecay             *float64 `json:"epsilon_decay"`
	MinEpsilon               *float64 `json:"min_epsilon"`
	ChurnPenalty             *float64 `json:"churn_penalty"`
	MinPricePercent          *float64 `json:"min_price_percent"`
	MaxPricePercent          *float64 `json:"max_price_percent"`
	ConversionDropThreshold  *float64 `json:"conversion_drop_threshold"`
	SessionCooldownMinutes   *int     `json:"session_cooldown_minutes"`
	MaxSessionSteps          *int     `json:"max_session_steps"`
	RolloutPhase             *string  `json:"rollout_phase"`
	FallbackPricePercent     *float64 `json:"fallback_price_percent"`
}

func (h *RLHandler) UpdateConfig(c *gin.Context) {
	var req UpdateRLConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Algorithm != nil {
		updates["algorithm"] = *req.Algorithm
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
	if req.EpsilonDecay != nil {
		updates["epsilon_decay"] = *req.EpsilonDecay
	}
	if req.MinEpsilon != nil {
		updates["min_epsilon"] = *req.MinEpsilon
	}
	if req.ChurnPenalty != nil {
		updates["churn_penalty"] = *req.ChurnPenalty
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
	if req.MaxSessionSteps != nil {
		updates["max_session_steps"] = *req.MaxSessionSteps
	}
	if req.RolloutPhase != nil {
		updates["rollout_phase"] = *req.RolloutPhase
	}
	if req.FallbackPricePercent != nil {
		updates["fallback_price_percent"] = *req.FallbackPricePercent
	}

	h.db.Model(&RLConfig{}).Where("is_active = ?", true).Updates(updates)

	c.JSON(http.StatusOK, gin.H{"message": "RL config updated"})
}

// ── Admin: Kill Switch ────────────────────────────────────────────────────────────

func (h *RLHandler) ActivateKillSwitch(c *gin.Context) {
	ActivateRLKillSwitch(h.db, "admin_manual")
	c.JSON(http.StatusOK, gin.H{"message": "RL kill switch activated — falling back to bandit/static"})
}

func (h *RLHandler) DeactivateKillSwitch(c *gin.Context) {
	DeactivateRLKillSwitch(h.db)
	c.JSON(http.StatusOK, gin.H{"message": "RL kill switch deactivated — RL pricing resumed"})
}

// ── Admin: Rollout Control ────────────────────────────────────────────────────────

func (h *RLHandler) AdvanceRollout(c *gin.Context) {
	next := AdvanceRolloutPhase(h.db)
	c.JSON(http.StatusOK, gin.H{"message": "Rollout advanced", "new_phase": next})
}

func (h *RLHandler) RollbackRollout(c *gin.Context) {
	prev := RollbackRolloutPhase(h.db)
	c.JSON(http.StatusOK, gin.H{"message": "Rollout rolled back", "new_phase": prev})
}

// ── Admin: Save Q-Table ──────────────────────────────────────────────────────────

func (h *RLHandler) SaveQTable(c *gin.Context) {
	if err := SaveQTable(h.db); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Q-table saved"})
}

// ── Admin: Q-Table Stats ──────────────────────────────────────────────────────────

func (h *RLHandler) GetQTableStats(c *gin.Context) {
	stats := QTableStats()
	c.JSON(http.StatusOK, stats)
}

// ── Admin: Conversion Check ────────────────────────────────────────────────────────

func (h *RLHandler) CheckConversion(c *gin.Context) {
	dropped, rate := CheckRLConversionDrop(h.db)
	c.JSON(http.StatusOK, gin.H{
		"conversion_dropped": dropped,
		"current_rate":       rate,
	})
}
