package payments

import (
	"errors"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// AgentLiquidityLog Model
// ════════════════════════════════════════════════════════════════════════════

type AgentLiquidityLog struct {
	ID            int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	AgentID       uuid.UUID       `gorm:"type:uuid;not null;index:idx_all_agent" json:"agent_id"`
	EventType     string          `gorm:"size:30;not null" json:"event_type"`
	Amount        decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"amount"`
	BalanceBefore decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"balance_before"`
	BalanceAfter  decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"balance_after"`
	ReferenceID   *uuid.UUID      `gorm:"type:uuid" json:"reference_id,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

func (AgentLiquidityLog) TableName() string { return "agent_liquidity_log" }

// ════════════════════════════════════════════════════════════════════════════
// AgentUtilization
// ════════════════════════════════════════════════════════════════════════════

type AgentUtilization struct {
	AgentID        uuid.UUID       `json:"agent_id"`
	Utilization    decimal.Decimal `json:"utilization_pct"`
	Available      decimal.Decimal `json:"available"`
	Level          string          `json:"level"` // healthy, warning, critical
	BalanceLimit   decimal.Decimal `json:"balance_limit"`
	CurrentBalance decimal.Decimal `json:"current_balance"`
}

// GetAgentUtilization returns the current utilization level for an agent.
func GetAgentUtilization(db *gorm.DB, agentID uuid.UUID) AgentUtilization {
	var agent PaymentAgent
	db.First(&agent, agentID)

	utilization := decimal.Zero
	if !agent.BalanceLimit.IsZero() {
		utilization = agent.CurrentBalance.Div(agent.BalanceLimit).Mul(decimal.NewFromInt(100))
	}

	var level string
	switch {
	case utilization.GreaterThanOrEqual(decimal.NewFromInt(90)):
		level = "critical"
	case utilization.GreaterThanOrEqual(decimal.NewFromInt(70)):
		level = "warning"
	default:
		level = "healthy"
	}

	return AgentUtilization{
		AgentID:        agentID,
		Utilization:    utilization,
		Available:      agent.BalanceLimit.Sub(agent.CurrentBalance),
		Level:          level,
		BalanceLimit:   agent.BalanceLimit,
		CurrentBalance: agent.CurrentBalance,
	}
}

// checkAgentCapacity verifies an agent has enough capacity for a given amount.
func checkAgentCapacity(db *gorm.DB, agentID uuid.UUID, amount decimal.Decimal) error {
	var agent PaymentAgent
	db.First(&agent, agentID)
	if agent.CurrentBalance.Add(amount).GreaterThan(agent.BalanceLimit) {
		return errors.New("agent capacity exceeded")
	}
	return nil
}

// findBestAgent finds the best available agent for a given currency and amount.
func findBestAgent(db *gorm.DB, currency string, amount decimal.Decimal) *PaymentAgent {
	var agent PaymentAgent
	db.Where("currency = ? AND status = ?"+
		" AND (balance_limit - current_balance) >= ?"+
		" AND trust_score >= 60",
		currency, "active", amount).
		Order("trust_score DESC, (balance_limit - current_balance) DESC").
		First(&agent)
	if agent.ID == uuid.Nil {
		return nil
	}
	return &agent
}

// ════════════════════════════════════════════════════════════════════════════
// System-Wide Liquidity Guard
// Prevents system collapse from deposit/withdrawal imbalance.
// ════════════════════════════════════════════════════════════════════════════

type SystemLiquidityStatus struct {
	TotalDeposits       decimal.Decimal `json:"total_deposits"`
	TotalWithdrawals    decimal.Decimal `json:"total_withdrawals"`
	Imbalance           decimal.Decimal `json:"imbalance"`
	ImbalancePct        decimal.Decimal `json:"imbalance_pct"`
	Level               string          `json:"level"` // balanced, warning, critical
	WithdrawalFeePct    decimal.Decimal `json:"withdrawal_fee_pct"`
	MinTrustForWithdraw int             `json:"min_trust_for_withdraw"`
	WithdrawalsSlowed   bool            `json:"withdrawals_slowed"`
}

// GetSystemLiquidityImbalance returns the current system-wide imbalance.
// Positive = more deposits than withdrawals (healthy). Negative = danger.
func GetSystemLiquidityImbalance(db *gorm.DB) decimal.Decimal {
	var totalDeposits float64
	db.Table("deposit_requests").
		Where("status = ?", "confirmed").
		Select("COALESCE(SUM(usd_amount),0)").Scan(&totalDeposits)

	var totalWithdrawals float64
	db.Table("withdraw_requests").
		Where("status = ?", "completed").
		Select("COALESCE(SUM(usd_amount),0)").Scan(&totalWithdrawals)

	return decimal.NewFromFloat(totalDeposits).Sub(decimal.NewFromFloat(totalWithdrawals))
}

// GetSystemLiquidityStatus returns the full liquidity status with protective measures.
func GetSystemLiquidityStatus(db *gorm.DB) SystemLiquidityStatus {
	var totalDeposits float64
	db.Table("deposit_requests").
		Where("status = ?", "confirmed").
		Select("COALESCE(SUM(usd_amount),0)").Scan(&totalDeposits)

	var totalWithdrawals float64
	db.Table("withdraw_requests").
		Where("status = ?", "completed").
		Select("COALESCE(SUM(usd_amount),0)").Scan(&totalWithdrawals)

	deposits := decimal.NewFromFloat(totalDeposits)
	withdrawals := decimal.NewFromFloat(totalWithdrawals)
	imbalance := deposits.Sub(withdrawals)

	imbalancePct := decimal.Zero
	if !deposits.IsZero() {
		imbalancePct = imbalance.Div(deposits).Mul(decimal.NewFromInt(100))
	}

	status := SystemLiquidityStatus{
		TotalDeposits:    deposits,
		TotalWithdrawals: withdrawals,
		Imbalance:        imbalance,
		ImbalancePct:     imbalancePct,
	}

	// Determine level and protective measures
	switch {
	case imbalancePct.LessThan(decimal.NewFromFloat(-20)):
		// Withdrawals > deposits by >20% — CRITICAL
		status.Level = "critical"
		status.WithdrawalFeePct = decimal.NewFromFloat(0.03) // 3% fee
		status.MinTrustForWithdraw = 80
		status.WithdrawalsSlowed = true
	case imbalancePct.LessThan(decimal.NewFromFloat(-10)):
		// Withdrawals > deposits by 10-20% — WARNING
		status.Level = "warning"
		status.WithdrawalFeePct = decimal.NewFromFloat(0.015) // 1.5% fee
		status.MinTrustForWithdraw = 60
		status.WithdrawalsSlowed = true
	default:
		status.Level = "balanced"
		status.WithdrawalFeePct = decimal.NewFromFloat(0.005) // 0.5% standard
		status.MinTrustForWithdraw = 40
		status.WithdrawalsSlowed = false
	}

	return status
}

// ApplyLiquidityGuard checks if a withdrawal should be allowed/slowed
// based on current system liquidity status.
func ApplyLiquidityGuard(db *gorm.DB, userID uuid.UUID, amount decimal.Decimal) error {
	status := GetSystemLiquidityStatus(db)

	if status.Level == "critical" {
		// In critical imbalance, only high-trust users can withdraw
		vip := getVIPUser(db, userID)
		if vip == nil {
			return errors.New("withdrawals temporarily restricted: system liquidity critical")
		}
		if vip.Tier != "gold" && vip.Tier != "platinum" {
			return errors.New("withdrawals restricted: only gold/platinum VIP during critical liquidity")
		}
	}

	if status.Level == "warning" {
		// In warning, require higher trust
		vip := getVIPUser(db, userID)
		if vip == nil && amount.GreaterThan(decimal.NewFromFloat(500)) {
			return errors.New("large withdrawals restricted during liquidity warning")
		}
	}

	return nil
}

// GetMyLiquidity — GET /api/v1/agent/liquidity
func (h *Handler) GetMyLiquidity(c *gin.Context) {
	agentUserID := getUserUUID(c)
	agent := getAgentByUserID(h.db, agentUserID)
	if agent == nil {
		response.Forbidden(c)
		return
	}
	utilization := GetAgentUtilization(h.db, agent.ID)
	response.OK(c, utilization)
}

// GetSystemLiquidity — GET /api/v1/admin/payments/liquidity
func (h *Handler) GetSystemLiquidity(c *gin.Context) {
	status := GetSystemLiquidityStatus(h.db)
	response.OK(c, status)
}
