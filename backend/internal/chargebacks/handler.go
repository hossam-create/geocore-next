package chargebacks

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/dispute"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// ListChargebacks returns all chargebacks with optional status filter.
// GET /admin/chargebacks?status=open&page=1&per_page=20
func (h *Handler) ListChargebacks(c *gin.Context) {
	page, perPage := 1, 20
	fmt.Sscan(c.DefaultQuery("page", "1"), &page)
	fmt.Sscan(c.DefaultQuery("per_page", "20"), &perPage)
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	q := h.db.Model(&Chargeback{})
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	q.Count(&total)

	var list []Chargeback
	q.Offset((page - 1) * perPage).Limit(perPage).Order("created_at DESC").Find(&list)

	response.OKMeta(c, list, response.Meta{
		Total: total, Page: page, PerPage: perPage,
		Pages: (total + int64(perPage) - 1) / int64(perPage),
	})
}

// GetChargeback returns a single chargeback by ID.
// GET /admin/chargebacks/:id
func (h *Handler) GetChargeback(c *gin.Context) {
	id := c.Param("id")
	var cb Chargeback
	if err := h.db.Where("id = ?", id).First(&cb).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chargeback not found"})
		return
	}
	response.OK(c, cb)
}

// SubmitEvidence submits evidence to Stripe for a chargeback.
// POST /admin/chargebacks/:id/evidence
func (h *Handler) SubmitEvidence(c *gin.Context) {
	id := c.Param("id")
	var cb Chargeback
	if err := h.db.Where("id = ?", id).First(&cb).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chargeback not found"})
		return
	}

	var req EvidenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// If there's a Stripe dispute ID, submit evidence to Stripe
	if cb.StripeDisputeID != "" && stripe.Key != "" {
		params := &stripe.DisputeParams{}
		if req.Description != "" {
			params.Evidence = &stripe.DisputeEvidenceParams{
				UncategorizedText: stripe.String(req.Description),
			}
		}
		if req.FileURL != "" {
			params.Evidence = &stripe.DisputeEvidenceParams{
				UncategorizedFile: stripe.String(req.FileURL),
			}
		}
		if _, err := dispute.Update(cb.StripeDisputeID, params); err != nil {
			slog.Error("failed to submit evidence to Stripe", "dispute_id", cb.StripeDisputeID, "error", err.Error())
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to submit evidence to Stripe", "detail": err.Error()})
			return
		}
	}

	// Update local status to under_review
	h.db.Model(&cb).Update("status", ChargebackStatusUnderReview)
	cb.Status = ChargebackStatusUnderReview

	slog.Info("chargeback evidence submitted",
		"chargeback_id", cb.ID.String(),
		"stripe_dispute_id", cb.StripeDisputeID,
		"evidence_type", req.EvidenceType,
	)
	response.OK(c, cb)
}

// HandleDisputeCreated upserts a Chargeback from a Stripe charge.dispute.created event.
func HandleDisputeCreated(db *gorm.DB, disputeID, paymentIntentID, reason, status string, amount float64, currency string, evidenceDueBy *time.Time) {
	// Find payment by Stripe PI ID
	var paymentID uuid.UUID
	var orderID *uuid.UUID
	row := db.Table("payments").Select("id").Where("stripe_payment_intent_id = ?", paymentIntentID).Row()
	if err := row.Scan(&paymentID); err != nil {
		slog.Warn("chargeback: payment not found for stripe PI", "stripe_pi", paymentIntentID)
		return
	}

	// Try to find order associated with this payment
	var oid *uuid.UUID
	db.Table("orders").Select("id").Where("payment_id = ?", paymentID).Row().Scan(&oid)
	orderID = oid

	cb := Chargeback{
		PaymentID:       paymentID,
		OrderID:         orderID,
		StripeDisputeID: disputeID,
		Amount:          amount,
		Currency:        currency,
		Reason:          reason,
		Status:          ChargebackStatus(status),
		EvidenceDueBy:   evidenceDueBy,
	}

	result := db.Where("stripe_dispute_id = ?", disputeID).FirstOrCreate(&cb)
	if result.Error != nil {
		slog.Error("chargeback: failed to upsert", "error", result.Error.Error())
		return
	}

	// Update fields on existing record
	db.Model(&cb).Updates(map[string]interface{}{
		"status":           ChargebackStatus(status),
		"reason":           reason,
		"amount":           amount,
		"evidence_due_by":  evidenceDueBy,
	})

	slog.Info("chargeback: created/updated from Stripe webhook",
		"dispute_id", disputeID,
		"status", status,
		"amount", amount,
	)
}

// HandleDisputeUpdated updates a Chargeback from a Stripe charge.dispute.updated event.
func HandleDisputeUpdated(db *gorm.DB, disputeID, status string, evidenceDueBy *time.Time) {
	var cb Chargeback
	if err := db.Where("stripe_dispute_id = ?", disputeID).First(&cb).Error; err != nil {
		slog.Warn("chargeback: dispute not found for update", "dispute_id", disputeID)
		return
	}
	updates := map[string]interface{}{"status": ChargebackStatus(status)}
	if evidenceDueBy != nil {
		updates["evidence_due_by"] = evidenceDueBy
	}
	db.Model(&cb).Updates(updates)

	slog.Info("chargeback: updated from Stripe webhook", "dispute_id", disputeID, "status", status)
}

// HandleDisputeClosed closes a Chargeback from a Stripe charge.dispute.closed event.
func HandleDisputeClosed(db *gorm.DB, disputeID, status string) {
	var cb Chargeback
	if err := db.Where("stripe_dispute_id = ?", disputeID).First(&cb).Error; err != nil {
		slog.Warn("chargeback: dispute not found for close", "dispute_id", disputeID)
		return
	}

	// Stripe sends "won" or "lost" in the status field on close
	cbStatus := ChargebackStatusLost
	if status == "won" {
		cbStatus = ChargebackStatusWon
	}
	db.Model(&cb).Update("status", cbStatus)

	slog.Info("chargeback: closed from Stripe webhook", "dispute_id", disputeID, "status", cbStatus)
}
