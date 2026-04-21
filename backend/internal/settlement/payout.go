package settlement

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ════════════════════════════════════════════════════════════════════════════
// Payout handler — admin-triggered seller payouts with audit logging
// ════════════════════════════════════════════════════════════════════════════

// CreatePayoutReq is the request body for creating a payout.
type CreatePayoutReq struct {
	SellerID    string  `json:"seller_id" binding:"required"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Currency    string  `json:"currency"`
	Destination string  `json:"destination" binding:"required"`
	Method      string  `json:"method" binding:"required"` // bank_transfer, wallet, paymob
}

// ApprovePayoutReq is the request body for approving a payout.
type ApprovePayoutReq struct {
	Notes string `json:"notes"`
}

// CreatePayout creates a new payout request (admin only).
// POST /admin/payouts
func (h *Handler) CreatePayout(c *gin.Context) {
	var req CreatePayoutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	sellerID, _ := uuid.Parse(req.SellerID)
	if req.Currency == "" {
		req.Currency = "AED"
	}

	adminID, _ := uuid.Parse(c.MustGet("user_id").(string))

	payout := Payout{
		SellerID:    sellerID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Destination: req.Destination,
		Method:      req.Method,
		Status:      PayoutStatusPending,
		AuditLog:    fmt.Sprintf("Created by admin %s at %s", adminID, time.Now().Format(time.RFC3339)),
	}

	if err := h.db.Create(&payout).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	metrics.IncWalletOp("payout_create", "success")
	slog.Info("Payout created",
		"payout_id", payout.ID,
		"seller_id", sellerID,
		"amount", req.Amount,
		"method", req.Method,
		"admin_id", adminID,
	)
	response.Created(c, payout)
}

// ApprovePayout approves a pending payout (admin only).
// POST /admin/payouts/:id/approve
func (h *Handler) ApprovePayout(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid payout ID")
		return
	}

	var payout Payout
	if err := h.db.Where("id = ?", id).First(&payout).Error; err != nil {
		response.NotFound(c, "Payout")
		return
	}

	if payout.Status != PayoutStatusPending {
		response.BadRequest(c, "Payout is not in pending status")
		return
	}

	adminID, _ := uuid.Parse(c.MustGet("user_id").(string))
	now := time.Now()

	payout.Status = PayoutStatusApproved
	payout.ApprovedBy = &adminID
	payout.ApprovedAt = &now
	payout.AuditLog += fmt.Sprintf("\nApproved by admin %s at %s", adminID, now.Format(time.RFC3339))

	h.db.Save(&payout)

	metrics.IncWalletOp("payout_approve", "success")
	slog.Info("Payout approved", "payout_id", id, "admin_id", adminID)
	response.OK(c, payout)
}

// ProcessPayout executes an approved payout (admin only).
// POST /admin/payouts/:id/process
func (h *Handler) ProcessPayout(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid payout ID")
		return
	}

	var payout Payout
	if err := h.db.Where("id = ?", id).First(&payout).Error; err != nil {
		response.NotFound(c, "Payout")
		return
	}

	if payout.Status != PayoutStatusApproved {
		response.BadRequest(c, "Payout must be approved before processing")
		return
	}

	adminID, _ := uuid.Parse(c.MustGet("user_id").(string))
	now := time.Now()

	// In a real implementation, this would:
	// 1. Call PayMob disbursement API or bank transfer API
	// 2. Debit seller wallet
	// 3. Record financial audit trail

	payout.Status = PayoutStatusProcessing
	payout.AuditLog += fmt.Sprintf("\nProcessing started by admin %s at %s", adminID, now.Format(time.RFC3339))
	h.db.Save(&payout)

	// Simulate completion (in production, webhook would finalize)
	payout.Status = PayoutStatusCompleted
	payout.ProcessedAt = &now
	payout.ReferenceID = fmt.Sprintf("PAYOUT-%s", id.String()[:8])
	payout.AuditLog += fmt.Sprintf("\nCompleted at %s, ref: %s", now.Format(time.RFC3339), payout.ReferenceID)
	h.db.Save(&payout)

	metrics.IncWalletOp("payout_process", "success")
	slog.Info("Payout processed", "payout_id", id, "amount", payout.Amount, "admin_id", adminID)
	response.OK(c, payout)
}

// ListPayouts returns all payouts (admin only).
// GET /admin/payouts
func (h *Handler) ListPayouts(c *gin.Context) {
	var payouts []Payout
	h.db.Order("created_at DESC").Limit(100).Find(&payouts)
	response.OK(c, payouts)
}

// GetPayout returns a single payout (admin only).
// GET /admin/payouts/:id
func (h *Handler) GetPayout(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid payout ID")
		return
	}

	var payout Payout
	if err := h.db.Where("id = ?", id).First(&payout).Error; err != nil {
		response.NotFound(c, "Payout")
		return
	}
	response.OK(c, payout)
}
