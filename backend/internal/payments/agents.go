package payments

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/kyc"
	"github.com/geocore-next/backend/internal/security"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ════════════════════════════════════════════════════════════════════════════
// PaymentAgent Model
// ════════════════════════════════════════════════════════════════════════════

type PaymentAgent struct {
	ID               uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID           uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	Status           string          `gorm:"size:20;not null;default:'pending'" json:"status"`
	Country          string          `gorm:"size:3;not null" json:"country"`
	Currency         string          `gorm:"size:3;not null" json:"currency"`
	BalanceLimit     decimal.Decimal `gorm:"type:decimal(15,2);not null;default:10000" json:"balance_limit"`
	CurrentBalance   decimal.Decimal `gorm:"type:decimal(15,2);not null;default:0" json:"current_balance"`
	CollateralHeld   decimal.Decimal `gorm:"type:decimal(15,2);not null;default:0" json:"collateral_held"`
	TrustScore       int             `gorm:"not null;default:50" json:"trust_score"`
	PaymentMethods   string          `gorm:"type:jsonb;default:'[]'" json:"payment_methods"`
	ApprovedBy       *uuid.UUID      `gorm:"type:uuid" json:"approved_by,omitempty"`
	ApprovedAt       *time.Time      `json:"approved_at,omitempty"`
	SuspensionReason *string         `gorm:"type:text" json:"suspension_reason,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

func (PaymentAgent) TableName() string { return "payment_agents" }

// PaymentMethod represents a single payment method for an agent.
type PaymentMethod struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
	Name       string `json:"name"`
}

// AgentPublicView is the safe public view of an agent returned to users.
type AgentPublicView struct {
	ID                uuid.UUID       `json:"id"`
	TrustScore        int             `json:"trust_score"`
	PaymentMethods    []PaymentMethod `json:"payment_methods"`
	AvailableCapacity decimal.Decimal `json:"available_capacity"`
}

// ════════════════════════════════════════════════════════════════════════════
// Agent Handlers
// ════════════════════════════════════════════════════════════════════════════

// RegisterAgent — POST /api/v1/admin/payments/agents/register
func (h *Handler) RegisterAgent(c *gin.Context) {
	var req struct {
		UserID           uuid.UUID       `json:"user_id" binding:"required"`
		Country          string          `json:"country" binding:"required"`
		Currency         string          `json:"currency" binding:"required"`
		BalanceLimit     decimal.Decimal `json:"balance_limit" binding:"required"`
		CollateralAmount decimal.Decimal `json:"collateral_amount" binding:"required"`
		PaymentMethods   []PaymentMethod `json:"payment_methods" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Verify KYC approved
	var kycProfile kyc.KYCProfile
	if err := h.db.Where("user_id = ? AND status = ?", req.UserID, kyc.StatusApproved).
		First(&kycProfile).Error; err != nil {
		response.BadRequest(c, "agent must have approved KYC")
		return
	}

	methodsJSON, _ := json.Marshal(req.PaymentMethods)

	err := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var count int64
		tx.Model(&PaymentAgent{}).Where("user_id = ?", req.UserID).Count(&count)
		if count > 0 {
			return fmt.Errorf("user is already a payment agent")
		}

		agent := PaymentAgent{
			ID:             uuid.New(),
			UserID:         req.UserID,
			Status:         "pending",
			Country:        req.Country,
			Currency:       req.Currency,
			BalanceLimit:   req.BalanceLimit,
			CollateralHeld: req.CollateralAmount,
			TrustScore:     50,
			PaymentMethods: string(methodsJSON),
		}
		if err := tx.Create(&agent).Error; err != nil {
			return err
		}

		// Hold collateral in escrow
		slog.Info("payments: holding agent collateral", "user_id", req.UserID, "agent_id", agent.ID, "amount", req.CollateralAmount.String())

		security.LogEvent(tx, c, &req.UserID, security.EventAdminAction, map[string]any{
			"action":     "agent_registered",
			"agent_id":   agent.ID.String(),
			"collateral": req.CollateralAmount.String(),
		})
		return nil
	})

	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Created(c, gin.H{"message": "agent registered, pending approval"})
}

// ApproveAgent — PUT /api/v1/admin/payments/agents/:id/approve
func (h *Handler) ApproveAgent(c *gin.Context) {
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid agent ID")
		return
	}
	adminIDStr, _ := c.Get("user_id")
	adminID, _ := uuid.Parse(adminIDStr.(string))

	err = locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var agent PaymentAgent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", agentID).First(&agent).Error; err != nil {
			return fmt.Errorf("agent not found")
		}
		if agent.Status != "pending" {
			return fmt.Errorf("agent is not in pending status (current: %s)", agent.Status)
		}

		now := time.Now()
		return tx.Model(&agent).Updates(map[string]interface{}{
			"status":      "active",
			"approved_by": adminID,
			"approved_at": &now,
		}).Error
	})

	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	security.LogEvent(h.db, c, &adminID, security.EventAdminAction, map[string]any{
		"action":   "agent_approved",
		"agent_id": agentID.String(),
	})
	response.OK(c, gin.H{"message": "agent approved"})
}

// SuspendAgent — PUT /api/v1/admin/payments/agents/:id/suspend
func (h *Handler) SuspendAgent(c *gin.Context) {
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid agent ID")
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "suspension reason required")
		return
	}

	adminIDStr, _ := c.Get("user_id")
	adminID, _ := uuid.Parse(adminIDStr.(string))

	err = locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var agent PaymentAgent
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", agentID).First(&agent).Error; err != nil {
			return fmt.Errorf("agent not found")
		}
		if agent.Status != "active" {
			return fmt.Errorf("agent is not active (current: %s)", agent.Status)
		}

		reason := req.Reason
		return tx.Model(&agent).Updates(map[string]interface{}{
			"status":            "suspended",
			"suspension_reason": &reason,
		}).Error
	})

	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Freeze pending requests for this agent
	h.db.Model(&DepositRequest{}).
		Where("agent_id = ? AND status = ?", agentID, "pending").
		Update("status", "expired")
	h.db.Model(&WithdrawRequest{}).
		Where("agent_id = ? AND status IN ?", agentID, []string{"pending", "assigned"}).
		Update("status", "cancelled")

	security.LogEvent(h.db, c, &adminID, security.EventAdminAction, map[string]any{
		"action":   "agent_suspended",
		"agent_id": agentID.String(),
		"reason":   req.Reason,
	})
	response.OK(c, gin.H{"message": "agent suspended, pending requests frozen"})
}

// GetAvailableAgents — GET /api/v1/payments/agents/available
func (h *Handler) GetAvailableAgents(c *gin.Context) {
	country := c.Query("country")
	currency := c.Query("currency")
	amountStr := c.Query("amount")
	amount, _ := decimal.NewFromString(amountStr)
	if amount.IsZero() {
		amount = decimal.NewFromInt(999999)
	}

	var agents []PaymentAgent
	h.db.Where("country = ? AND currency = ? AND status = ?"+
		" AND (balance_limit - current_balance) >= ?",
		country, currency, "active", amount).
		Order("trust_score DESC, (balance_limit - current_balance) DESC").
		Find(&agents)

	var views []AgentPublicView
	for _, a := range agents {
		var methods []PaymentMethod
		json.Unmarshal([]byte(a.PaymentMethods), &methods)
		views = append(views, AgentPublicView{
			ID:                a.ID,
			TrustScore:        a.TrustScore,
			PaymentMethods:    methods,
			AvailableCapacity: a.BalanceLimit.Sub(a.CurrentBalance),
		})
	}
	if views == nil {
		views = []AgentPublicView{}
	}
	response.OK(c, views)
}

// ListAllAgents — GET /api/v1/admin/payments/agents
func (h *Handler) ListAllAgents(c *gin.Context) {
	var agents []PaymentAgent
	h.db.Order("created_at DESC").Find(&agents)
	response.OK(c, agents)
}

// GetAgentUtilization — GET /api/v1/admin/payments/agents/:id/utilization
func (h *Handler) GetAgentUtilization(c *gin.Context) {
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid agent ID")
		return
	}
	utilization := GetAgentUtilization(h.db, agentID)
	response.OK(c, utilization)
}

// GetPaymentDashboard — GET /api/v1/admin/payments/dashboard
func (h *Handler) GetPaymentDashboard(c *gin.Context) {
	type DashboardStats struct {
		TotalAgents      int64   `json:"total_agents"`
		ActiveAgents     int64   `json:"active_agents"`
		PendingDeposits  int64   `json:"pending_deposits"`
		PendingWithdraws int64   `json:"pending_withdraws"`
		TotalDeposited   float64 `json:"total_deposited_24h"`
		TotalWithdrawn   float64 `json:"total_withdrawn_24h"`
	}
	var stats DashboardStats
	h.db.Model(&PaymentAgent{}).Count(&stats.TotalAgents)
	h.db.Model(&PaymentAgent{}).Where("status = ?", "active").Count(&stats.ActiveAgents)
	h.db.Model(&DepositRequest{}).Where("status = ?", "pending").Count(&stats.PendingDeposits)
	h.db.Model(&WithdrawRequest{}).Where("status IN ?", []string{"pending", "assigned"}).Count(&stats.PendingWithdraws)

	yesterday := time.Now().Add(-24 * time.Hour)
	h.db.Model(&DepositRequest{}).
		Where("status = ? AND confirmed_by_agent_at > ?", "confirmed", yesterday).
		Select("COALESCE(SUM(usd_amount),0)").Scan(&stats.TotalDeposited)
	h.db.Model(&WithdrawRequest{}).
		Where("status = ? AND completed_at > ?", "completed", yesterday).
		Select("COALESCE(SUM(usd_amount),0)").Scan(&stats.TotalWithdrawn)

	response.OK(c, stats)
}

// getAgentByUserID returns the PaymentAgent for a given user.
func getAgentByUserID(db *gorm.DB, userID uuid.UUID) *PaymentAgent {
	var agent PaymentAgent
	if err := db.Where("user_id = ?", userID).First(&agent).Error; err != nil {
		return nil
	}
	return &agent
}

// getUserUUID extracts user ID from gin context as uuid.UUID.
func getUserUUID(c *gin.Context) uuid.UUID {
	idStr, _ := c.MustGet("user_id").(string)
	id, _ := uuid.Parse(idStr)
	return id
}
