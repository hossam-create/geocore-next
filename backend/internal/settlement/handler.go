package settlement

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/pkg/jobs"
	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Handler provides HTTP handlers for settlement and payout operations.
type Handler struct {
	db *gorm.DB
}

// NewHandler creates a new settlement handler.
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// CreateSettlement creates a settlement from a completed escrow/payment.
// POST /settlements
func (h *Handler) CreateSettlement(c *gin.Context) {
	var req struct {
		SellerID       string  `json:"seller_id" binding:"required"`
		BuyerID        string  `json:"buyer_id" binding:"required"`
		OrderID        *string `json:"order_id"`
		PaymentID      *string `json:"payment_id"`
		EscrowID       *string `json:"escrow_id"`
		Amount         float64 `json:"amount" binding:"required,gt=0"`
		Currency       string  `json:"currency"`
		PlatformFee    float64 `json:"platform_fee"`
		PaymentFee     float64 `json:"payment_fee"`
		IdempotencyKey string  `json:"idempotency_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	sellerID, _ := uuid.Parse(req.SellerID)
	buyerID, _ := uuid.Parse(req.BuyerID)
	if req.Currency == "" {
		req.Currency = "AED"
	}

	// Idempotency check
	if req.IdempotencyKey != "" {
		var existing Settlement
		if err := h.db.Where("idempotency_key = ?", req.IdempotencyKey).First(&existing).Error; err == nil {
			response.OK(c, gin.H{"settlement_id": existing.ID, "status": existing.Status})
			return
		}
	}

	netAmount := req.Amount - req.PlatformFee - req.PaymentFee
	if netAmount <= 0 {
		response.BadRequest(c, "Net amount must be positive after fees")
		return
	}

	var orderID, paymentID, escrowID *uuid.UUID
	if req.OrderID != nil {
		oid, _ := uuid.Parse(*req.OrderID)
		orderID = &oid
	}
	if req.PaymentID != nil {
		pid, _ := uuid.Parse(*req.PaymentID)
		paymentID = &pid
	}
	if req.EscrowID != nil {
		eid, _ := uuid.Parse(*req.EscrowID)
		escrowID = &eid
	}

	settlement := Settlement{
		SellerID:       sellerID,
		BuyerID:        buyerID,
		OrderID:        orderID,
		PaymentID:      paymentID,
		EscrowID:       escrowID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		PlatformFee:    req.PlatformFee,
		PaymentFee:     req.PaymentFee,
		NetAmount:      netAmount,
		Status:         SettlementStatusPending,
		IdempotencyKey: req.IdempotencyKey,
	}

	if err := h.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&settlement).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Enqueue background settlement processing job
	_ = jobs.EnqueueDefault(&jobs.Job{
		Type:      jobs.JobTypeSettlementProcess,
		Payload:   map[string]interface{}{"settlement_id": settlement.ID.String()},
		RequestID: c.GetString("request_id"),
	})

	metrics.IncWalletOp("settlement_create", "success")
	response.Created(c, gin.H{
		"settlement_id": settlement.ID,
		"net_amount":    netAmount,
		"status":        settlement.Status,
	})
}

// ListSettlements returns settlements for the authenticated seller.
// GET /settlements
func (h *Handler) ListSettlements(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	sellerID, _ := uuid.Parse(userID)

	var settlements []Settlement
	h.db.Where("seller_id = ?", sellerID).Order("created_at DESC").Limit(50).Find(&settlements)
	response.OK(c, settlements)
}

// ProcessSettlementJob is the background job handler for settlement processing.
func ProcessSettlementJob(db *gorm.DB) jobs.JobHandler {
	return func(ctx context.Context, job *jobs.Job) error {
		sid, ok := job.Payload["settlement_id"].(string)
		if !ok {
			return fmt.Errorf("missing settlement_id in payload")
		}

		settlementID, err := uuid.Parse(sid)
		if err != nil {
			return fmt.Errorf("invalid settlement_id: %w", err)
		}

		var settlement Settlement
		if err := db.Where("id = ?", settlementID).First(&settlement).Error; err != nil {
			return fmt.Errorf("settlement not found: %w", err)
		}

		if settlement.Status != SettlementStatusPending {
			return nil // already processed
		}

		// Mark as processing
		db.Model(&settlement).Update("status", SettlementStatusProcessing)

		// In a real implementation, this would:
		// 1. Transfer net_amount from escrow to seller wallet
		// 2. Credit platform_fee to platform wallet
		// 3. Record all double-entry transactions

		now := time.Now()
		db.Model(&settlement).Updates(map[string]interface{}{
			"status":       SettlementStatusCompleted,
			"processed_at": now,
		})

		slog.Info("Settlement processed",
			"settlement_id", settlement.ID,
			"seller_id", settlement.SellerID,
			"net_amount", settlement.NetAmount,
			"currency", settlement.Currency,
		)
		metrics.IncWalletOp("settlement_process", "success")
		return nil
	}
}
