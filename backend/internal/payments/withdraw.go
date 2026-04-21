package payments

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/freeze"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ════════════════════════════════════════════════════════════════════════════
// WithdrawRequest Model
// ════════════════════════════════════════════════════════════════════════════

type WithdrawRequest struct {
	ID               uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID           uuid.UUID       `gorm:"type:uuid;not null;index:idx_wr_user" json:"user_id"`
	AgentID          *uuid.UUID      `gorm:"type:uuid;index:idx_wr_agent" json:"agent_id,omitempty"`
	Amount           decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"amount"`
	Currency         string          `gorm:"size:3;not null" json:"currency"`
	USDAmount        decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"usd_amount"`
	FXRate           decimal.Decimal `gorm:"type:decimal(10,6);not null" json:"fx_rate"`
	RecipientDetails string          `gorm:"type:jsonb;not null" json:"recipient_details"`
	Status           string          `gorm:"size:20;not null;default:'pending'" json:"status"`
	AssignedAt       *time.Time      `json:"assigned_at,omitempty"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	FailureReason    *string         `gorm:"type:text" json:"failure_reason,omitempty"`
	IdempotencyKey   *string         `gorm:"size:255;uniqueIndex" json:"idempotency_key,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

func (WithdrawRequest) TableName() string { return "withdraw_requests" }

// ════════════════════════════════════════════════════════════════════════════
// Withdraw Handlers
// ════════════════════════════════════════════════════════════════════════════

// RequestWithdraw — POST /api/v1/payments/withdraw/request
func (h *Handler) RequestWithdraw(c *gin.Context) {
	var req struct {
		Amount           decimal.Decimal        `json:"amount" binding:"required"`
		Currency         string                 `json:"currency" binding:"required"`
		RecipientDetails map[string]interface{} `json:"recipient_details" binding:"required"`
		IdempotencyKey   string                 `json:"idempotency_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	userID := getUserUUID(c)

	// Sprint 8.5: Block frozen users from withdrawing
	if freeze.IsUserFrozen(h.db, userID) {
		response.Forbidden(c)
		return
	}

	// Fraud gate
	gateResult := CheckPaymentTrustGate(h.db, h.rdb, userID, "withdraw", req.Amount)
	if !gateResult.Allowed {
		response.Forbidden(c)
		return
	}

	// Fast withdraw after deposit check
	if err := checkFastWithdrawAbuse(h.db, h.rdb, userID); err != nil {
		response.RateLimited(c, err.Error())
		return
	}

	fxRate := h.fx.GetRate("USD", req.Currency)
	usdAmount := req.Amount.Mul(fxRate)

	// VIP daily/monthly limit check
	if err := checkUserLimits(h.db, userID, usdAmount, "withdraw"); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	recipientJSON, _ := json.Marshal(req.RecipientDetails)

	err := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		// Debit user wallet (move to pending)
		if err := debitWallet(tx, userID, usdAmount); err != nil {
			return err
		}

		// Find best available agent
		agent := findBestAgent(tx, req.Currency, usdAmount)
		var agentID *uuid.UUID
		var assignedAt *time.Time
		status := "pending"
		if agent != nil {
			agentID = &agent.ID
			now := time.Now()
			assignedAt = &now
			status = "assigned"
		}

		idempotencyKey := req.IdempotencyKey
		withdraw := WithdrawRequest{
			ID:               uuid.New(),
			UserID:           userID,
			AgentID:          agentID,
			Amount:           req.Amount,
			Currency:         req.Currency,
			USDAmount:        usdAmount,
			FXRate:           fxRate,
			RecipientDetails: string(recipientJSON),
			Status:           status,
			AssignedAt:       assignedAt,
			IdempotencyKey:   &idempotencyKey,
		}
		return tx.Create(&withdraw).Error
	})

	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Created(c, gin.H{"message": "withdraw request submitted"})
}

// AgentCompleteWithdraw — POST /api/v1/agent/withdraw/:id/complete
func (h *Handler) AgentCompleteWithdraw(c *gin.Context) {
	withdrawID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid withdraw ID")
		return
	}
	agentUserID := getUserUUID(c)

	err = locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var withdraw WithdrawRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", withdrawID).First(&withdraw).Error; err != nil {
			return fmt.Errorf("withdraw not found")
		}

		agent := getAgentByUserID(tx, agentUserID)
		if agent == nil {
			return fmt.Errorf("unauthorized: not a payment agent")
		}
		if withdraw.AgentID != nil && *withdraw.AgentID != agent.ID {
			return fmt.Errorf("unauthorized: not the assigned agent")
		}
		if withdraw.Status != "assigned" && withdraw.Status != "processing" {
			return fmt.Errorf("invalid withdraw status: %s", withdraw.Status)
		}

		// 1. Complete the debit (remove from pending)
		if err := completeDebit(tx, withdraw.UserID, withdraw.USDAmount); err != nil {
			return fmt.Errorf("wallet debit failed: %w", err)
		}

		// 2. Update agent balance (decrease)
		balanceBefore := agent.CurrentBalance
		if err := tx.Model(&PaymentAgent{}).Where("id = ?", agent.ID).
			Update("current_balance", gorm.Expr("current_balance - ?", withdraw.USDAmount)).Error; err != nil {
			return err
		}

		// 3. Log liquidity event
		balanceAfter := balanceBefore.Sub(withdraw.USDAmount)
		tx.Create(&AgentLiquidityLog{
			AgentID:       agent.ID,
			EventType:     "withdraw_completed",
			Amount:        withdraw.USDAmount,
			BalanceBefore: balanceBefore,
			BalanceAfter:  balanceAfter,
			ReferenceID:   &withdrawID,
		})

		// 4. Update withdraw status
		now := time.Now()
		return tx.Model(&withdraw).Updates(map[string]interface{}{
			"status":       "completed",
			"completed_at": &now,
		}).Error
	})

	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	slog.Info("payments: agent completed withdraw", "withdraw_id", withdrawID, "agent_user_id", agentUserID)
	response.OK(c, gin.H{"message": "withdraw completed"})
}

// CancelWithdraw — DELETE /api/v1/payments/withdraw/:id/cancel
func (h *Handler) CancelWithdraw(c *gin.Context) {
	withdrawID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid withdraw ID")
		return
	}
	userID := getUserUUID(c)

	err = locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var withdraw WithdrawRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", withdrawID, userID).First(&withdraw).Error; err != nil {
			return fmt.Errorf("withdraw not found")
		}
		if withdraw.Status != "pending" && withdraw.Status != "assigned" {
			return fmt.Errorf("cannot cancel withdraw in status: %s", withdraw.Status)
		}

		// Release funds back to available balance
		if err := releasePending(tx, withdraw.UserID, withdraw.USDAmount); err != nil {
			return fmt.Errorf("wallet release failed: %w", err)
		}

		return tx.Model(&withdraw).Update("status", "cancelled").Error
	})

	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"message": "withdraw cancelled, funds released"})
}

// GetWithdrawHistory — GET /api/v1/payments/withdraw/history
func (h *Handler) GetWithdrawHistory(c *gin.Context) {
	userID := getUserUUID(c)
	var withdraws []WithdrawRequest
	h.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(50).Find(&withdraws)
	response.OK(c, withdraws)
}
