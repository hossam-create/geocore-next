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
// DepositRequest Model
// ════════════════════════════════════════════════════════════════════════════

type DepositRequest struct {
	ID                 uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID             uuid.UUID       `gorm:"type:uuid;not null;index:idx_dr_user" json:"user_id"`
	AgentID            uuid.UUID       `gorm:"type:uuid;not null;index:idx_dr_agent" json:"agent_id"`
	Amount             decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"amount"`
	Currency           string          `gorm:"size:3;not null" json:"currency"`
	USDAmount          decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"usd_amount"`
	FXRate             decimal.Decimal `gorm:"type:decimal(10,6);not null" json:"fx_rate"`
	Status             string          `gorm:"size:20;not null;default:'pending'" json:"status"`
	ProofURL           *string         `gorm:"size:500" json:"proof_url,omitempty"`
	ProofUploadedAt    *time.Time      `json:"proof_uploaded_at,omitempty"`
	ConfirmedByAgentAt *time.Time      `json:"confirmed_by_agent_at,omitempty"`
	RejectionReason    *string         `gorm:"type:text" json:"rejection_reason,omitempty"`
	IdempotencyKey     *string         `gorm:"size:255;uniqueIndex" json:"idempotency_key,omitempty"`
	ExpiresAt          time.Time       `json:"expires_at"`
	CreatedAt          time.Time       `json:"created_at"`
}

func (DepositRequest) TableName() string { return "deposit_requests" }

// ════════════════════════════════════════════════════════════════════════════
// Deposit Handlers
// ════════════════════════════════════════════════════════════════════════════

// InitiateDeposit — POST /api/v1/payments/deposit/initiate
func (h *Handler) InitiateDeposit(c *gin.Context) {
	var req struct {
		AgentID        uuid.UUID       `json:"agent_id" binding:"required"`
		Amount         decimal.Decimal `json:"amount" binding:"required"`
		Currency       string          `json:"currency" binding:"required"`
		IdempotencyKey string          `json:"idempotency_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	userID := getUserUUID(c)

	// Sprint 8.5: Block frozen users from initiating deposits
	if freeze.IsUserFrozen(h.db, userID) {
		response.Forbidden(c)
		return
	}

	// Idempotency check
	if req.IdempotencyKey != "" {
		var existing DepositRequest
		if h.db.Where("idempotency_key = ? AND user_id = ?", req.IdempotencyKey, userID).
			First(&existing).Error == nil {
			response.OK(c, existing)
			return
		}
	}

	// Fraud trust gate
	gateResult := CheckPaymentTrustGate(h.db, h.rdb, userID, "deposit", req.Amount)
	if !gateResult.Allowed {
		response.Forbidden(c)
		return
	}

	// FX conversion
	fxRate := h.fx.GetRate(req.Currency, "USD")
	usdAmount := req.Amount.Mul(fxRate)

	// VIP daily/monthly limit check
	if err := checkUserLimits(h.db, userID, usdAmount, "deposit"); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Verify agent and capacity
	var agent PaymentAgent
	if err := h.db.Where("id = ? AND status = ?", req.AgentID, "active").First(&agent).Error; err != nil {
		response.BadRequest(c, "agent not found or not active")
		return
	}
	if agent.CurrentBalance.Add(usdAmount).GreaterThan(agent.BalanceLimit) {
		response.BadRequest(c, "agent capacity exceeded")
		return
	}

	idempotencyKey := req.IdempotencyKey
	deposit := DepositRequest{
		ID:             uuid.New(),
		UserID:         userID,
		AgentID:        req.AgentID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		USDAmount:      usdAmount,
		FXRate:         fxRate,
		IdempotencyKey: &idempotencyKey,
		ExpiresAt:      time.Now().Add(30 * time.Minute),
	}
	if err := h.db.Create(&deposit).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Build payment instructions
	var agentMethods []PaymentMethod
	json.Unmarshal([]byte(agent.PaymentMethods), &agentMethods)

	response.Created(c, gin.H{
		"deposit_id":      deposit.ID,
		"amount":          req.Amount.String(),
		"currency":        req.Currency,
		"usd_amount":      usdAmount.String(),
		"fx_rate":         fxRate.String(),
		"payment_methods": agentMethods,
		"instructions":    buildPaymentInstructions(agentMethods, req.Amount, req.Currency),
		"expires_at":      deposit.ExpiresAt,
		"reference":       deposit.ID.String()[:8],
	})
}

// UploadDepositProof — POST /api/v1/payments/deposit/:id/upload-proof
func (h *Handler) UploadDepositProof(c *gin.Context) {
	depositID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid deposit ID")
		return
	}
	userID := getUserUUID(c)

	var req struct {
		ProofURL string `json:"proof_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	err = locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var deposit DepositRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", depositID, userID).First(&deposit).Error; err != nil {
			return fmt.Errorf("deposit not found")
		}
		if deposit.Status != "pending" {
			return fmt.Errorf("deposit is not in pending status")
		}
		if time.Now().After(deposit.ExpiresAt) {
			tx.Model(&deposit).Update("status", "expired")
			return fmt.Errorf("deposit expired")
		}

		now := time.Now()
		proofURL := req.ProofURL
		return tx.Model(&deposit).Updates(map[string]interface{}{
			"status":            "paid",
			"proof_url":         &proofURL,
			"proof_uploaded_at": &now,
		}).Error
	})

	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"message": "proof uploaded, awaiting agent confirmation"})
}

// AgentConfirmDeposit — POST /api/v1/agent/deposit/:id/confirm
func (h *Handler) AgentConfirmDeposit(c *gin.Context) {
	depositID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid deposit ID")
		return
	}
	agentUserID := getUserUUID(c)

	// Sprint 8.5: Block frozen agents from confirming deposits
	agentUUID, _ := uuid.Parse(c.GetString("user_id"))
	if freeze.IsUserFrozen(h.db, agentUUID) {
		response.Forbidden(c)
		return
	}

	var depositOwnerID uuid.UUID
	var depositUSDAmount decimal.Decimal

	err = locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var deposit DepositRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", depositID).First(&deposit).Error; err != nil {
			return fmt.Errorf("deposit not found")
		}

		// Sprint 8.5: Also block if the deposit owner is frozen
		if freeze.IsUserFrozen(tx, deposit.UserID) {
			return fmt.Errorf("user account is frozen")
		}

		// Capture for audit log outside callback
		depositOwnerID = deposit.UserID
		depositUSDAmount = deposit.USDAmount

		agent := getAgentByUserID(tx, agentUserID)
		if agent == nil || agent.ID != deposit.AgentID {
			return fmt.Errorf("unauthorized: not the assigned agent")
		}
		if deposit.Status != "pending" && deposit.Status != "paid" {
			return fmt.Errorf("invalid deposit status: %s", deposit.Status)
		}
		if time.Now().After(deposit.ExpiresAt) {
			tx.Model(&deposit).Update("status", "expired")
			return fmt.Errorf("deposit expired")
		}

		// 1. Credit user wallet
		if err := creditWallet(tx, deposit.UserID, deposit.USDAmount, deposit.ID); err != nil {
			return fmt.Errorf("wallet credit failed: %w", err)
		}

		// 2. Update agent balance
		balanceBefore := agent.CurrentBalance
		if err := tx.Model(&PaymentAgent{}).Where("id = ?", agent.ID).
			Update("current_balance", gorm.Expr("current_balance + ?", deposit.USDAmount)).Error; err != nil {
			return err
		}

		// 3. Log liquidity event
		balanceAfter := balanceBefore.Add(deposit.USDAmount)
		tx.Create(&AgentLiquidityLog{
			AgentID:       agent.ID,
			EventType:     "deposit_confirmed",
			Amount:        deposit.USDAmount,
			BalanceBefore: balanceBefore,
			BalanceAfter:  balanceAfter,
			ReferenceID:   &depositID,
		})

		// 4. Update deposit status
		now := time.Now()
		return tx.Model(&deposit).Updates(map[string]interface{}{
			"status":                "confirmed",
			"confirmed_by_agent_at": &now,
		}).Error
	})

	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	slog.Info("payments: agent confirmed deposit", "deposit_id", depositID, "agent_user_id", agentUserID)

	// Sprint 8.5: Audit log for financial action
	freeze.LogAudit(h.db, "agent_confirm_deposit", agentUUID, depositOwnerID, fmt.Sprintf("deposit_id=%s amount=%s", depositID, depositUSDAmount.String()))

	response.OK(c, gin.H{"message": "deposit confirmed"})
}

// AgentRejectDeposit — POST /api/v1/agent/deposit/:id/reject
func (h *Handler) AgentRejectDeposit(c *gin.Context) {
	depositID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid deposit ID")
		return
	}
	agentUserID := getUserUUID(c)

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	err = locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var deposit DepositRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", depositID).First(&deposit).Error; err != nil {
			return fmt.Errorf("deposit not found")
		}

		agent := getAgentByUserID(tx, agentUserID)
		if agent == nil || agent.ID != deposit.AgentID {
			return fmt.Errorf("unauthorized: not the assigned agent")
		}
		if deposit.Status != "pending" && deposit.Status != "paid" {
			return fmt.Errorf("invalid deposit status: %s", deposit.Status)
		}

		reason := req.Reason
		return tx.Model(&deposit).Updates(map[string]interface{}{
			"status":           "rejected",
			"rejection_reason": &reason,
		}).Error
	})

	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"message": "deposit rejected"})
}

// GetDepositStatus — GET /api/v1/payments/deposit/:id/status
func (h *Handler) GetDepositStatus(c *gin.Context) {
	depositID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid deposit ID")
		return
	}
	userID := getUserUUID(c)

	var deposit DepositRequest
	if h.db.Where("id = ? AND user_id = ?", depositID, userID).First(&deposit).Error != nil {
		response.NotFound(c, "Deposit")
		return
	}
	response.OK(c, deposit)
}

// GetDepositHistory — GET /api/v1/payments/deposit/history
func (h *Handler) GetDepositHistory(c *gin.Context) {
	userID := getUserUUID(c)
	var deposits []DepositRequest
	h.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(50).Find(&deposits)
	response.OK(c, deposits)
}

// GetAgentPendingRequests — GET /api/v1/agent/requests
func (h *Handler) GetAgentPendingRequests(c *gin.Context) {
	agentUserID := getUserUUID(c)
	agent := getAgentByUserID(h.db, agentUserID)
	if agent == nil {
		response.Forbidden(c)
		return
	}

	var deposits []DepositRequest
	h.db.Where("agent_id = ? AND status IN ?", agent.ID, []string{"pending", "paid"}).
		Order("created_at ASC").Find(&deposits)

	var withdraws []WithdrawRequest
	h.db.Where("agent_id = ? AND status IN ?", agent.ID, []string{"assigned", "processing"}).
		Order("created_at ASC").Find(&withdraws)

	response.OK(c, gin.H{
		"deposits":  deposits,
		"withdraws": withdraws,
	})
}

// buildPaymentInstructions creates payment instructions for the user.
func buildPaymentInstructions(methods []PaymentMethod, amount decimal.Decimal, currency string) []string {
	var instructions []string
	for _, m := range methods {
		switch m.Type {
		case "instapay":
			instructions = append(instructions, fmt.Sprintf("Send %s %s via InstaPay to %s (%s)", amount.String(), currency, m.Identifier, m.Name))
		case "vodafone_cash":
			instructions = append(instructions, fmt.Sprintf("Send %s %s via Vodafone Cash to %s (%s)", amount.String(), currency, m.Identifier, m.Name))
		case "bank_transfer":
			instructions = append(instructions, fmt.Sprintf("Transfer %s %s to bank account %s (%s)", amount.String(), currency, m.Identifier, m.Name))
		default:
			instructions = append(instructions, fmt.Sprintf("Send %s %s via %s to %s (%s)", amount.String(), currency, m.Type, m.Identifier, m.Name))
		}
	}
	return instructions
}

// creditWallet credits a user's wallet with the given USD amount.
func creditWallet(tx *gorm.DB, userID uuid.UUID, amount decimal.Decimal, referenceID uuid.UUID) error {
	// Find wallet balance for USD
	type wbRef struct {
		ID               uuid.UUID
		AvailableBalance decimal.Decimal
	}
	var balance wbRef
	if err := tx.Table("wallet_balances wb").
		Joins("JOIN wallets w ON w.id = wb.wallet_id").
		Where("w.user_id = ? AND wb.currency = ?", userID, "USD").
		Select("wb.id, wb.available_balance").
		Scan(&balance).Error; err != nil {
		return fmt.Errorf("wallet not found for user %s", userID)
	}

	return tx.Table("wallet_balances").Where("id = ?", balance.ID).
		Update("available_balance", gorm.Expr("available_balance + ?", amount)).Error
}

// debitWallet debits a user's wallet and moves amount to pending.
func debitWallet(tx *gorm.DB, userID uuid.UUID, amount decimal.Decimal) error {
	type wbRef struct {
		ID               uuid.UUID
		AvailableBalance decimal.Decimal
	}
	var balance wbRef
	if err := tx.Table("wallet_balances wb").
		Joins("JOIN wallets w ON w.id = wb.wallet_id").
		Where("w.user_id = ? AND wb.currency = ?", userID, "USD").
		Select("wb.id, wb.available_balance").
		Scan(&balance).Error; err != nil {
		return fmt.Errorf("wallet not found for user %s", userID)
	}

	if balance.AvailableBalance.LessThan(amount) {
		return fmt.Errorf("insufficient balance")
	}

	return tx.Table("wallet_balances").Where("id = ?", balance.ID).
		Updates(map[string]interface{}{
			"available_balance": gorm.Expr("available_balance - ?", amount),
			"pending_balance":   gorm.Expr("pending_balance + ?", amount),
		}).Error
}

// releasePending moves amount from pending back to available (cancel/failed).
func releasePending(tx *gorm.DB, userID uuid.UUID, amount decimal.Decimal) error {
	type wbRef struct {
		ID uuid.UUID
	}
	var balance wbRef
	if err := tx.Table("wallet_balances wb").
		Joins("JOIN wallets w ON w.id = wb.wallet_id").
		Where("w.user_id = ? AND wb.currency = ?", userID, "USD").
		Select("wb.id").
		Scan(&balance).Error; err != nil {
		return fmt.Errorf("wallet not found for user %s", userID)
	}

	return tx.Table("wallet_balances").Where("id = ?", balance.ID).
		Updates(map[string]interface{}{
			"pending_balance":   gorm.Expr("pending_balance - ?", amount),
			"available_balance": gorm.Expr("available_balance + ?", amount),
		}).Error
}

// completeDebit moves amount from pending to fully debited (withdraw completed).
func completeDebit(tx *gorm.DB, userID uuid.UUID, amount decimal.Decimal) error {
	type wbRef struct {
		ID uuid.UUID
	}
	var balance wbRef
	if err := tx.Table("wallet_balances wb").
		Joins("JOIN wallets w ON w.id = wb.wallet_id").
		Where("w.user_id = ? AND wb.currency = ?", userID, "USD").
		Select("wb.id").
		Scan(&balance).Error; err != nil {
		return fmt.Errorf("wallet not found for user %s", userID)
	}

	return tx.Table("wallet_balances").Where("id = ?", balance.ID).
		Update("pending_balance", gorm.Expr("pending_balance - ?", amount)).Error
}
