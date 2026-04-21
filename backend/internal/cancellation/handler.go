package cancellation

import (
	"net/http"
	"time"

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

// ── Cancel Order (Buyer → Traveler) ─────────────────────────────────────────────

type CancelOrderRequest struct {
	Reason string `json:"reason"`
}

// CancelOrderWithFee calculates and applies the smart cancellation fee,
// then updates the order status to cancelled.
func (h *Handler) CancelOrderWithFee(c *gin.Context) {
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

	var req CancelOrderRequest
	c.ShouldBindJSON(&req)

	// 1. Calculate cancellation fee
	result, err := CalculateCancellationFee(h.db, orderID, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 2. Load order for traveler ID
	var ord struct {
		BuyerID      uuid.UUID
		SellerID     uuid.UUID
		Status       string
		DeliveryType string
	}
	if err := h.db.Table("orders").
		Select("buyer_id, seller_id, status, delivery_type").
		Where("id = ?", orderID).First(&ord).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// 3. Validate status transition
	if ord.Status == "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order already cancelled"})
		return
	}
	if ord.Status == "completed" || ord.Status == "delivered" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot cancel completed order"})
		return
	}

	// 4. Update order status to cancelled
	now := time.Now()
	if err := h.db.Table("orders").Where("id = ?", orderID).
		Updates(map[string]interface{}{
			"status":           "cancelled",
			"cancelled_at":     now,
			"cancelled_reason": req.Reason,
			"updated_at":       now,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel order"})
		return
	}

	// 5. Apply fee distribution (ledger + wallet credits)
	if err := ApplyCancellationFee(h.db, orderID, uid, ord.SellerID, result, req.Reason); err != nil {
		// Order is cancelled but fee distribution failed — log but don't block
		_ = err
	}

	// 6. If crowdshipping, also cancel the delivery request
	if ord.DeliveryType == "CROWDSHIPPING" {
		h.db.Table("delivery_requests").
			Where("buyer_id = ? AND status IN ?", ord.BuyerID, []string{"pending", "matched", "accepted"}).
			Updates(map[string]interface{}{"status": "cancelled", "updated_at": now})
	}

	c.JSON(http.StatusOK, gin.H{
		"message":               "Order cancelled",
		"fee_cents":             result.FeeCents,
		"traveler_compensation": result.TravelerCompensation,
		"platform_fee":          result.PlatformFee,
		"fee_percent":           result.FeePercent,
		"tier":                  result.Tier,
		"token_used":            result.TokenUsed,
		"abuse_multiplier":      result.AbuseMultiplier,
		"insurance_applied":     result.InsuranceApplied,
		"original_fee_cents":    result.OriginalFeeCents,
	})
}

// ── Preview Cancellation Fee ─────────────────────────────────────────────────────

// PreviewCancellationFee shows the buyer what the fee would be BEFORE they confirm.
func (h *Handler) PreviewCancellationFee(c *gin.Context) {
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

	result, err := CalculateCancellationFee(h.db, orderID, uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Also return remaining free tokens
	remainingTokens := GetRemainingTokens(h.db, uid)

	c.JSON(http.StatusOK, gin.H{
		"fee_cents":             result.FeeCents,
		"traveler_compensation": result.TravelerCompensation,
		"platform_fee":          result.PlatformFee,
		"fee_percent":           result.FeePercent,
		"tier":                  result.Tier,
		"abuse_multiplier":      result.AbuseMultiplier,
		"seconds_since_accept":  result.SecondsSinceAccept,
		"remaining_free_tokens": remainingTokens,
	})
}

// ── Admin: Cancellation Stats ────────────────────────────────────────────────────

func (h *Handler) GetAdminStats(c *gin.Context) {
	var stats AdminCancellationStats

	h.db.Table("cancellation_ledger").Count(&stats.TotalCancellations)

	var feeSum struct {
		TotalFees    int64
		TravelerComp int64
		PlatformFees int64
		AvgPercent   float64
		TokensUsed   int64
	}
	h.db.Table("cancellation_ledger").
		Select("COALESCE(SUM(fee_cents),0) as total_fees, " +
			"COALESCE(SUM(traveler_compensation),0) as traveler_comp, " +
			"COALESCE(SUM(platform_fee),0) as platform_fees, " +
			"COALESCE(AVG(fee_percent),0) as avg_percent, " +
			"COUNT(CASE WHEN token_used THEN 1 END) as tokens_used").
		Scan(&feeSum)

	stats.TotalFeesCollected = feeSum.TotalFees
	stats.TravelerCompensated = feeSum.TravelerComp
	stats.PlatformFees = feeSum.PlatformFees
	stats.AvgFeePercent = feeSum.AvgPercent
	stats.TokensUsed = feeSum.TokensUsed

	// High risk users (cancel_rate > 30%)
	h.db.Table("user_cancellation_stats").
		Where("cancel_rate > ?", CancelRateThreshold).
		Count(&stats.HighRiskUsers)

	c.JSON(http.StatusOK, stats)
}

// ── Admin: List Policies ─────────────────────────────────────────────────────────

func (h *Handler) ListPolicies(c *gin.Context) {
	var policies []CancellationPolicy
	h.db.Where("is_active = ?", true).Find(&policies)
	c.JSON(http.StatusOK, policies)
}

// ── Admin: Update Policy ─────────────────────────────────────────────────────────

type UpdatePolicyRequest struct {
	GraceSeconds  *int     `json:"grace_seconds"`
	Tier1Seconds  *int     `json:"tier1_seconds"`
	Tier2Seconds  *int     `json:"tier2_seconds"`
	FeeGracePct   *float64 `json:"fee_grace_pct"`
	FeeTier1Pct   *float64 `json:"fee_tier1_pct"`
	FeeTier2Pct   *float64 `json:"fee_tier2_pct"`
	FeeMaxPct     *float64 `json:"fee_max_pct"`
	TravelerSplit *float64 `json:"traveler_split"`
}

func (h *Handler) UpdatePolicy(c *gin.Context) {
	policyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid policy ID"})
		return
	}

	var req UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var policy CancellationPolicy
	if err := h.db.First(&policy, "id = ?", policyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
		return
	}

	updates := map[string]interface{}{}
	if req.GraceSeconds != nil {
		updates["grace_seconds"] = *req.GraceSeconds
	}
	if req.Tier1Seconds != nil {
		updates["tier1_seconds"] = *req.Tier1Seconds
	}
	if req.Tier2Seconds != nil {
		updates["tier2_seconds"] = *req.Tier2Seconds
	}
	if req.FeeGracePct != nil {
		updates["fee_grace_pct"] = *req.FeeGracePct
	}
	if req.FeeTier1Pct != nil {
		updates["fee_tier1_pct"] = *req.FeeTier1Pct
	}
	if req.FeeTier2Pct != nil {
		updates["fee_tier2_pct"] = *req.FeeTier2Pct
	}
	if req.FeeMaxPct != nil {
		updates["fee_max_pct"] = *req.FeeMaxPct
	}
	if req.TravelerSplit != nil {
		updates["traveler_split"] = *req.TravelerSplit
	}

	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		h.db.Model(&policy).Updates(updates)
	}

	h.db.First(&policy, "id = ?", policyID)
	c.JSON(http.StatusOK, policy)
}

// ── Admin: List High-Risk Users ──────────────────────────────────────────────────

func (h *Handler) ListHighRiskUsers(c *gin.Context) {
	var stats []UserCancellationStats
	h.db.Where("cancel_rate > ?", CancelRateThreshold).
		Order("cancel_rate DESC").
		Find(&stats)
	c.JSON(http.StatusOK, stats)
}
