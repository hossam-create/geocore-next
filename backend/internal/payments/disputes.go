package payments

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/reputation"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ════════════════════════════════════════════════════════════════════════════
// Payment Dispute Engine
// Handles disputes between users and agents: not credited, agent didn't release, double charge.
// ════════════════════════════════════════════════════════════════════════════

// PaymentDispute represents a dispute between a user and an agent.
type PaymentDispute struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID      uuid.UUID       `gorm:"type:uuid;not null;index" json:"user_id"`
	AgentID     uuid.UUID       `gorm:"type:uuid;not null;index" json:"agent_id"`
	DepositID   *uuid.UUID      `gorm:"type:uuid;index" json:"deposit_id,omitempty"`
	WithdrawID  *uuid.UUID      `gorm:"type:uuid;index" json:"withdraw_id,omitempty"`
	AmountCents int64           `gorm:"not null" json:"amount_cents"`
	Amount      decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"amount"`
	Reason      string          `gorm:"size:50;not null" json:"reason"` // not_credited, not_released, double_charge, other
	ProofImage  *string         `gorm:"size:500" json:"proof_image,omitempty"`
	Status      string          `gorm:"size:20;not null;default:'open'" json:"status"` // open, review, resolved, rejected
	Resolution  *string         `gorm:"size:50" json:"resolution,omitempty"`           // credit_issued, agent_penalized, rejected, auto_resolved
	ResolvedBy  *uuid.UUID      `gorm:"type:uuid" json:"resolved_by,omitempty"`
	ResolvedAt  *time.Time      `json:"resolved_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func (PaymentDispute) TableName() string { return "payment_disputes" }

// Payment dispute reasons
const (
	DisputeReasonNotCredited  = "not_credited"
	DisputeReasonNotReleased  = "not_released"
	DisputeReasonDoubleCharge = "double_charge"
	DisputeReasonOther        = "other"
)

// Payment dispute statuses
const (
	DisputeStatusOpen     = "open"
	DisputeStatusReview   = "review"
	DisputeStatusResolved = "resolved"
	DisputeStatusRejected = "rejected"
)

// ════════════════════════════════════════════════════════════════════════════
// Dispute Handlers
// ════════════════════════════════════════════════════════════════════════════

// OpenDispute — POST /api/v1/payments/disputes
func (h *Handler) OpenDispute(c *gin.Context) {
	var req struct {
		AgentID    uuid.UUID       `json:"agent_id" binding:"required"`
		DepositID  *uuid.UUID      `json:"deposit_id"`
		WithdrawID *uuid.UUID      `json:"withdraw_id"`
		Amount     decimal.Decimal `json:"amount" binding:"required"`
		Reason     string          `json:"reason" binding:"required"`
		ProofImage string          `json:"proof_image"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Validate reason
	validReasons := map[string]bool{
		DisputeReasonNotCredited:  true,
		DisputeReasonNotReleased:  true,
		DisputeReasonDoubleCharge: true,
		DisputeReasonOther:        true,
	}
	if !validReasons[req.Reason] {
		response.BadRequest(c, "invalid dispute reason")
		return
	}

	userID := getUserUUID(c)

	dispute := PaymentDispute{
		ID:         uuid.New(),
		UserID:     userID,
		AgentID:    req.AgentID,
		DepositID:  req.DepositID,
		WithdrawID: req.WithdrawID,
		Amount:     req.Amount,
		Reason:     req.Reason,
		Status:     DisputeStatusOpen,
	}
	if req.ProofImage != "" {
		proofImage := req.ProofImage
		dispute.ProofImage = &proofImage
	}

	if err := h.db.Create(&dispute).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Auto-escalate if proof provided
	if dispute.ProofImage != nil {
		h.autoResolveDispute(dispute.ID)
	}

	response.Created(c, dispute)
}

// ResolveDispute — PUT /api/v1/admin/payments/disputes/:id/resolve
func (h *Handler) ResolveDispute(c *gin.Context) {
	disputeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid dispute ID")
		return
	}

	var req struct {
		Resolution string `json:"resolution" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	adminIDStr, _ := c.Get("user_id")
	adminID, _ := uuid.Parse(adminIDStr.(string))

	err = locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var dispute PaymentDispute
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", disputeID).First(&dispute).Error; err != nil {
			return fmt.Errorf("dispute not found")
		}
		if dispute.Status != DisputeStatusOpen && dispute.Status != DisputeStatusReview {
			return fmt.Errorf("dispute cannot be resolved in status: %s", dispute.Status)
		}

		now := time.Now()
		resolution := req.Resolution
		dispute.Status = DisputeStatusResolved
		dispute.Resolution = &resolution
		dispute.ResolvedBy = &adminID
		dispute.ResolvedAt = &now

		if err := tx.Save(&dispute).Error; err != nil {
			return err
		}

		// Apply consequences based on resolution
		switch resolution {
		case "credit_issued":
			// Credit user wallet
			if err := creditWallet(tx, dispute.UserID, dispute.Amount, dispute.ID); err != nil {
				slog.Error("dispute: failed to credit wallet", "dispute_id", dispute.ID, "error", err)
			}
			// Record dispute on agent
			_ = RecordAgentDispute(tx, dispute.AgentID)

		case "agent_penalized":
			// Credit user + penalize agent
			_ = creditWallet(tx, dispute.UserID, dispute.Amount, dispute.ID)
			_ = RecordAgentDispute(tx, dispute.AgentID)
			// Apply reputation penalty
			var agent PaymentAgent
			if tx.Where("id = ?", dispute.AgentID).First(&agent).Error == nil {
				_ = reputation.ApplyPenalty(tx, nil, agent.UserID, "agent", reputation.PenaltyDisputeLost)
			}

		case "auto_resolved":
			// Auto-credited
			_ = creditWallet(tx, dispute.UserID, dispute.Amount, dispute.ID)
			_ = RecordAgentDispute(tx, dispute.AgentID)
		}

		return nil
	})

	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"message": "dispute resolved"})
}

// GetDispute — GET /api/v1/payments/disputes/:id
func (h *Handler) GetDispute(c *gin.Context) {
	disputeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid dispute ID")
		return
	}
	userID := getUserUUID(c)

	var dispute PaymentDispute
	if h.db.Where("id = ? AND user_id = ?", disputeID, userID).First(&dispute).Error != nil {
		response.NotFound(c, "Dispute")
		return
	}
	response.OK(c, dispute)
}

// ListDisputes — GET /api/v1/admin/payments/disputes
func (h *Handler) ListDisputes(c *gin.Context) {
	status := c.Query("status")
	var disputes []PaymentDispute
	q := h.db.Order("created_at DESC")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Find(&disputes)
	response.OK(c, disputes)
}

// GetUserDisputes — GET /api/v1/payments/disputes
func (h *Handler) GetUserDisputes(c *gin.Context) {
	userID := getUserUUID(c)
	var disputes []PaymentDispute
	h.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&disputes)
	response.OK(c, disputes)
}

// autoResolveDispute attempts to auto-resolve disputes with valid proof.
func (h *Handler) autoResolveDispute(disputeID uuid.UUID) {
	err := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var dispute PaymentDispute
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status = ?", disputeID, DisputeStatusOpen).First(&dispute).Error; err != nil {
			return err
		}

		// Auto-resolve if proof is valid and agent is flagged
		if dispute.ProofImage == nil {
			return fmt.Errorf("no proof image")
		}

		// Check if agent has low score or is flagged
		agentScore := GetAgentScore(tx, dispute.AgentID)
		if agentScore == nil || agentScore.Score < 60 {
			// Auto-resolve in favor of user
			now := time.Now()
			resolution := "auto_resolved"
			dispute.Status = DisputeStatusResolved
			dispute.Resolution = &resolution
			dispute.ResolvedAt = &now
			tx.Save(&dispute)

			// Credit user
			_ = creditWallet(tx, dispute.UserID, dispute.Amount, dispute.ID)
			_ = RecordAgentDispute(tx, dispute.AgentID)

			slog.Info("dispute: auto-resolved", "dispute_id", disputeID, "agent_id", dispute.AgentID)
			return nil
		}

		// Escalate to review
		dispute.Status = DisputeStatusReview
		tx.Save(&dispute)
		return nil
	})

	if err != nil {
		slog.Error("dispute: auto-resolve failed", "dispute_id", disputeID, "error", err)
	}
}
