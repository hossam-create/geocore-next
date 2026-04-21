package payments

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Payment Trust Gate — Fraud checks for deposit/withdraw operations
// ════════════════════════════════════════════════════════════════════════════

type PaymentTrustGateResult struct {
	Allowed   bool     `json:"allowed"`
	RiskScore int      `json:"risk_score"`
	Action    string   `json:"action"` // allow, review, block
	Flags     []string `json:"flags"`
}

// CheckPaymentTrustGate performs fraud checks before deposit/withdraw operations.
func CheckPaymentTrustGate(db *gorm.DB, rdb *redis.Client, userID uuid.UUID, operation string, amount decimal.Decimal) PaymentTrustGateResult {
	result := PaymentTrustGateResult{
		Allowed: true,
		Flags:   []string{},
	}

	ctx := context.Background()

	switch operation {
	case "deposit":
		// Rule 1: More than 3 deposits in 1 hour
		if rdb != nil {
			key := fmt.Sprintf("deposit_count:%s", userID.String())
			count, _ := rdb.Incr(ctx, key).Result()
			rdb.Expire(ctx, key, time.Hour)
			if count > 3 {
				result.RiskScore += 40
				result.Flags = append(result.Flags, "repeated_deposits")
			}
		}

		// Rule 2: Same agent always (agent abuse)
		if checkSameAgentAbuse(db, userID) {
			result.RiskScore += 30
			result.Flags = append(result.Flags, "same_agent_abuse")
		}

	case "withdraw":
		// Rule 3: Fast withdraw after deposit (< 2 hours)
		if checkFastWithdrawAfterDeposit(db, userID) {
			result.RiskScore += 50
			result.Flags = append(result.Flags, "fast_withdraw_after_deposit")
		}

		// Rule 4: Large amount for low-trust user
		if amount.GreaterThan(decimal.NewFromFloat(5000)) {
			result.RiskScore += 20
			result.Flags = append(result.Flags, "large_amount")
		}
	}

	// Determine action
	switch {
	case result.RiskScore >= 70:
		result.Allowed = false
		result.Action = "block"
	case result.RiskScore >= 40:
		result.Action = "review"
	default:
		result.Action = "allow"
	}

	return result
}

// checkSameAgentAbuse detects if a user always deposits through the same agent.
func checkSameAgentAbuse(db *gorm.DB, userID uuid.UUID) bool {
	type agentCount struct {
		AgentID uuid.UUID
		Count   int64
	}
	var results []agentCount
	db.Model(&DepositRequest{}).
		Select("agent_id, count(*) as count").
		Where("user_id = ? AND status = ?", userID, "confirmed").
		Group("agent_id").
		Order("count DESC").
		Limit(1).
		Scan(&results)

	if len(results) > 0 && results[0].Count >= 5 {
		// Check if this agent is >80% of all deposits
		var total int64
		db.Model(&DepositRequest{}).
			Where("user_id = ? AND status = ?", userID, "confirmed").
			Count(&total)
		if total > 0 && float64(results[0].Count)/float64(total) > 0.8 {
			return true
		}
	}
	return false
}

// checkFastWithdrawAfterDeposit detects if a user tries to withdraw shortly after depositing.
func checkFastWithdrawAfterDeposit(db *gorm.DB, userID uuid.UUID) bool {
	var lastDeposit DepositRequest
	if db.Where("user_id = ? AND status = ?", userID, "confirmed").
		Order("confirmed_by_agent_at DESC").First(&lastDeposit).Error != nil {
		return false
	}
	if lastDeposit.ConfirmedByAgentAt == nil {
		return false
	}

	// If last confirmed deposit was less than 2 hours ago
	twoHoursAgo := time.Now().Add(-2 * time.Hour)
	return lastDeposit.ConfirmedByAgentAt.After(twoHoursAgo)
}

// checkFastWithdrawAbuse prevents immediate withdrawal after deposit for new users.
func checkFastWithdrawAbuse(db *gorm.DB, rdb *redis.Client, userID uuid.UUID) error {
	ctx := context.Background()
	if rdb != nil {
		key := fmt.Sprintf("last_deposit:%s", userID.String())
		lastDepositStr, err := rdb.Get(ctx, key).Result()
		if err == nil && lastDepositStr != "" {
			lastDeposit, err := time.Parse(time.RFC3339, lastDepositStr)
			if err == nil && time.Since(lastDeposit) < 2*time.Hour {
				return fmt.Errorf("withdrawal cooldown: please wait 2 hours after deposit")
			}
		}
	}
	return nil
}
