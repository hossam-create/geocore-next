package pricing

import (
	"net/http"

	"github.com/geocore-next/backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PricingHandler struct {
	db *gorm.DB
}

func NewPricingHandler(db *gorm.DB) *PricingHandler {
	return &PricingHandler{db: db}
}

// ── Get Dynamic Price ────────────────────────────────────────────────────────────

func (h *PricingHandler) GetInsurancePrice(c *gin.Context) {
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

	// Build pricing context from features
	ctx, err := BuildPricingContext(h.db, uid, orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check cooldown — return cached price if within session
	cooldown := CheckPriceCooldown(h.db, uid)
	if cooldown != nil {
		c.JSON(http.StatusOK, gin.H{
			"price_cents": cooldown.PriceCents,
			"strategy":    "cooldown",
			"message":     "Price locked for this session",
		})
		return
	}

	// Calculate dynamic price
	result, err := CalculateDynamicPrice(h.db, ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"price_cents":       result.PriceCents,
		"base_price_cents":  result.BasePriceCents,
		"price_percent":     result.PricePercent,
		"adjustments":       result.Adjustments,
		"buy_probability":   result.BuyProbability,
		"confidence":        result.Confidence,
		"strategy":          result.Strategy,
		"anchor_price":      result.AnchorPrice,
		"recommended_label": "Recommended protection for you",
	})
}

// ── Get Pricing Variant ──────────────────────────────────────────────────────────

func (h *PricingHandler) GetPricingVariant(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	uid := userID.(uuid.UUID)

	variant := AssignPricingVariant(h.db, uid)

	c.JSON(http.StatusOK, gin.H{
		"experiment": PricingExperiment,
		"variant":    variant,
	})
}

// ── Record Purchase Outcome ──────────────────────────────────────────────────────

type RecordOutcomeReq struct {
	DidBuy     bool `json:"did_buy"`
	DidCancel  bool `json:"did_cancel"`
	ClaimFiled bool `json:"claim_filed"`
}

func (h *PricingHandler) RecordOutcome(c *gin.Context) {
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

	var req RecordOutcomeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	UpdatePricingOutcome(h.db, uid, orderID, req.DidBuy, req.DidCancel, req.ClaimFiled)

	c.JSON(http.StatusOK, gin.H{"message": "Outcome recorded"})
}

// ── Admin: Pricing Metrics ────────────────────────────────────────────────────────

func (h *PricingHandler) GetAdminMetrics(c *gin.Context) {
	metrics, err := GetPricingABResults(h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, metrics)
}

// ── Admin: Load Model ────────────────────────────────────────────────────────────

type LoadModelReq struct {
	ModelJSON string `json:"model_json" binding:"required"`
	Version   string `json:"version"`
}

func (h *PricingHandler) LoadModel(c *gin.Context) {
	var req LoadModelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := LoadModelFromJSON([]byte(req.ModelJSON)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model: " + err.Error()})
		return
	}

	// Persist model config
	cfg := PricingModelConfig{
		Version:             req.Version,
		Strategy:            "ai",
		ModelJSON:           req.ModelJSON,
		IsActive:            true,
		BasePricePercent:    DefaultBasePricePercent,
		MinPricePercent:     DefaultMinPricePercent,
		MaxPricePercent:     DefaultMaxPricePercent,
		StaticPricePercent:  DefaultStaticPricePercent,
		ConfidenceThreshold: 0.7,
	}
	h.db.Create(&cfg)

	c.JSON(http.StatusOK, gin.H{
		"message": "Model loaded successfully",
		"version": req.Version,
	})
}

// ── Admin: Update Model Config ────────────────────────────────────────────────────

type UpdateConfigReq struct {
	MinPricePercent     *float64 `json:"min_price_percent"`
	MaxPricePercent     *float64 `json:"max_price_percent"`
	BasePricePercent    *float64 `json:"base_price_percent"`
	StaticPricePercent  *float64 `json:"static_price_percent"`
	ConfidenceThreshold *float64 `json:"confidence_threshold"`
}

func (h *PricingHandler) UpdateConfig(c *gin.Context) {
	var req UpdateConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.MinPricePercent != nil {
		updates["min_price_percent"] = *req.MinPricePercent
	}
	if req.MaxPricePercent != nil {
		updates["max_price_percent"] = *req.MaxPricePercent
	}
	if req.BasePricePercent != nil {
		updates["base_price_percent"] = *req.BasePricePercent
	}
	if req.StaticPricePercent != nil {
		updates["static_price_percent"] = *req.StaticPricePercent
	}
	if req.ConfidenceThreshold != nil {
		updates["confidence_threshold"] = *req.ConfidenceThreshold
	}

	h.db.Model(&PricingModelConfig{}).
		Where("is_active = ?", true).
		Updates(updates)

	c.JSON(http.StatusOK, gin.H{"message": "Config updated"})
}

// ── Admin: Detect Gaming ──────────────────────────────────────────────────────────

func (h *PricingHandler) DetectGaming(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	isGaming, score := DetectPriceGaming(h.db, userID)
	c.JSON(http.StatusOK, gin.H{
		"user_id":      userID,
		"is_gaming":    isGaming,
		"gaming_score": score,
	})
}
