package protection

import (
	"net/http"
	"time"

	"github.com/geocore-next/backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// ── Purchase Protection ──────────────────────────────────────────────────────────

type PurchaseProtectionRequest struct {
	BundleType string `json:"bundle_type" binding:"required"` // cancellation | delay | full
}

func (h *Handler) PurchaseProtection(c *gin.Context) {
	if !config.GetFlags().EnableTravelGuarantee {
		c.JSON(http.StatusForbidden, gin.H{"error": "Travel Guarantee is not available"})
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

	var req PurchaseProtectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bundle_type is required (cancellation, delay, full)"})
		return
	}

	// Anti-abuse check
	if ok, reason := CanPurchaseProtection(h.db, uid); !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "Protection unavailable: " + reason})
		return
	}

	protection, err := PurchaseProtection(h.db, orderID, uid, req.BundleType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, protection)
}

// ── Get Protection Pricing ──────────────────────────────────────────────────────

func (h *Handler) GetProtectionPricing(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	userID, _ := c.Get("userID")
	uid := userID.(uuid.UUID)

	type bundlePrice struct {
		Type            string  `json:"type"`
		Label           string  `json:"label"`
		Description     string  `json:"description"`
		PriceCents      int64   `json:"price_cents"`
		RiskFactor      float64 `json:"risk_factor"`
		UrgencyFactor   float64 `json:"urgency_factor"`
		HasCancellation bool    `json:"has_cancellation"`
		HasDelay        bool    `json:"has_delay"`
		HasFull         bool    `json:"has_full"`
	}

	prices := make([]bundlePrice, len(ProtectionBundles))
	for i, b := range ProtectionBundles {
		priceCents, riskFactor, urgencyFactor, pErr := CalculateProtectionPrice(h.db, orderID, uid, b.Type)
		if pErr != nil {
			priceCents = 0
			riskFactor = 0
			urgencyFactor = 0
		}
		prices[i] = bundlePrice{
			Type:            b.Type,
			Label:           b.Label,
			Description:     b.Description,
			PriceCents:      priceCents,
			RiskFactor:      riskFactor,
			UrgencyFactor:   urgencyFactor,
			HasCancellation: b.HasCancellation,
			HasDelay:        b.HasDelay,
			HasFull:         b.HasFull,
		}
	}

	// Get user's A/B variant
	variant := GetVariantForUser(h.db, uid, ExperimentProtectionDefault)

	c.JSON(http.StatusOK, gin.H{
		"order_id": orderID,
		"bundles":  prices,
		"variant":  variant,
		"enabled":  config.GetFlags().EnableTravelGuarantee,
	})
}

// ── Get Order Protection ─────────────────────────────────────────────────────────

func (h *Handler) GetOrderProtection(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var protection OrderProtection
	if err := h.db.Where("order_id = ?", orderID).First(&protection).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No protection found for this order"})
		return
	}

	c.JSON(http.StatusOK, protection)
}

// ── File Guarantee Claim ────────────────────────────────────────────────────────

type FileClaimReq struct {
	Type         ClaimType `json:"type" binding:"required"`
	EvidenceJSON string    `json:"evidence_json"`
}

func (h *Handler) FileClaim(c *gin.Context) {
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

	var req FileClaimReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type is required (no_show, delay, mismatch)"})
		return
	}

	// Anti-abuse check
	if ok, reason := CanFileClaim(h.db, uid); !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot file claim: " + reason})
		return
	}

	claim, err := FileClaim(h.db, orderID, uid, FileClaimRequest{
		Type:         req.Type,
		EvidenceJSON: req.EvidenceJSON,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, claim)
}

// ── Check Delay Status ──────────────────────────────────────────────────────────

func (h *Handler) CheckDelayStatus(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	status, err := CheckDelayStatus(h.db, orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// ── Get A/B Variant ──────────────────────────────────────────────────────────────

func (h *Handler) GetABVariant(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	uid := userID.(uuid.UUID)

	variant := GetVariantForUser(h.db, uid, ExperimentProtectionDefault)

	c.JSON(http.StatusOK, gin.H{
		"experiment": ExperimentProtectionDefault,
		"variant":    variant,
	})
}

// ── Admin: Protection Metrics ────────────────────────────────────────────────────

func (h *Handler) GetAdminMetrics(c *gin.Context) {
	metrics := AdminProtectionMetrics{}

	h.db.Table("order_protections").Count(&metrics.TotalProtected)

	var revenue struct{ Total int64 }
	h.db.Table("order_protections").
		Select("COALESCE(SUM(price_cents), 0) as total").Scan(&revenue)
	metrics.TotalRevenueCents = revenue.Total

	var payouts struct{ Total int64 }
	h.db.Table("guarantee_claims").
		Where("status IN ?", []string{"auto_approved", "approved"}).
		Select("COALESCE(SUM(refund_cents + compensation_cents), 0) as total").Scan(&payouts)
	metrics.TotalPayoutsCents = payouts.Total

	metrics.NetRevenueCents = metrics.TotalRevenueCents - metrics.TotalPayoutsCents

	var totalOrders int64
	h.db.Table("orders").Count(&totalOrders)
	if totalOrders > 0 {
		metrics.AttachRate = float64(metrics.TotalProtected) / float64(totalOrders)
	}

	var totalClaims int64
	h.db.Table("guarantee_claims").Count(&totalClaims)
	if metrics.TotalProtected > 0 {
		metrics.ClaimsRate = float64(totalClaims) / float64(metrics.TotalProtected)
	}

	var approvedClaims int64
	h.db.Table("guarantee_claims").Where("status IN ?", []string{"auto_approved", "approved"}).Count(&approvedClaims)
	if totalClaims > 0 {
		metrics.ApprovalRate = float64(approvedClaims) / float64(totalClaims)
	}

	// Abuse rate: users with abuse score > 30
	var flaggedUsers int64
	h.db.Table("guarantee_claims").
		Select("COUNT(DISTINCT user_id)").
		Where("status IN ?", []string{"auto_approved", "approved"}).
		Group("user_id").
		Having("COUNT(*) > 3").
		Scan(&flaggedUsers)
	if metrics.TotalProtected > 0 {
		metrics.AbuseRate = float64(flaggedUsers) / float64(metrics.TotalProtected)
	}

	// Top risky users
	metrics.TopRiskyUsers = GetTopRiskyUsers(h.db, 10)

	c.JSON(http.StatusOK, metrics)
}

// ── Admin: List Claims ────────────────────────────────────────────────────────────

func (h *Handler) ListClaims(c *gin.Context) {
	status := c.Query("status")
	var claims []GuaranteeClaim

	q := h.db.Order("created_at DESC")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Find(&claims)

	c.JSON(http.StatusOK, claims)
}

// ── Admin: Review Claim ──────────────────────────────────────────────────────────

type ReviewClaimReq struct {
	Decision          ClaimStatus `json:"decision" binding:"required"`
	RefundPercent     float64     `json:"refund_percent"`
	CompensationCents int64       `json:"compensation_cents"`
	TravelerPenalty   bool        `json:"traveler_penalty"`
}

func (h *Handler) ReviewClaim(c *gin.Context) {
	adminID, _ := c.Get("userID")
	uid := adminID.(uuid.UUID)

	claimID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid claim ID"})
		return
	}

	var req ReviewClaimReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claim, err := ReviewClaim(h.db, claimID, uid, ReviewClaimRequest{
		Decision:          req.Decision,
		RefundPercent:     req.RefundPercent,
		CompensationCents: req.CompensationCents,
		TravelerPenalty:   req.TravelerPenalty,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, claim)
}

// ── Admin: A/B Test Results ──────────────────────────────────────────────────────

func (h *Handler) GetABTestResults(c *gin.Context) {
	experiment := c.DefaultQuery("experiment", ExperimentProtectionDefault)

	results, err := GetABTestResults(h.db, experiment)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}

// ── Admin: Daily Metrics ──────────────────────────────────────────────────────────

func (h *Handler) GetDailyMetrics(c *gin.Context) {
	days := 30
	var metrics []ProtectionDailyMetrics
	h.db.Order("date DESC").Limit(days).Find(&metrics)

	c.JSON(http.StatusOK, metrics)
}

// ── Admin: Aggregate Metrics (trigger) ────────────────────────────────────────────

func (h *Handler) TriggerAggregation(c *gin.Context) {
	if err := AggregateDailyMetrics(h.db); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Metrics aggregated"})
}

// ── Admin: Scan Delayed Orders ────────────────────────────────────────────────────

func (h *Handler) TriggerDelayScan(c *gin.Context) {
	if err := ScanDelayedOrders(h.db); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Delay scan completed", "timestamp": time.Now()})
}
