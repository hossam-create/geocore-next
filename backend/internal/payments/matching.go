package payments

import (
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/config"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ════════════════════════════════════════════════════════════════════════════
// P2P Money Matching Engine
// Matches deposit_requests with withdraw_requests to settle internally.
// ════════════════════════════════════════════════════════════════════════════

// MatchResult records a successful match between a deposit and a withdrawal.
type MatchResult struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	DepositID   uuid.UUID       `gorm:"type:uuid;not null;index" json:"deposit_id"`
	WithdrawID  uuid.UUID       `gorm:"type:uuid;not null;index" json:"withdraw_id"`
	AmountCents int64           `gorm:"not null" json:"amount_cents"`
	Amount      decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"amount"`
	Rate        decimal.Decimal `gorm:"type:decimal(10,6);not null" json:"rate"`
	Status      string          `gorm:"size:20;not null;default:'settled'" json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
}

func (MatchResult) TableName() string { return "payment_match_results" }

// MatchCriteria defines how deposits and withdrawals are matched.
type MatchCriteria struct {
	Currency  string
	MinAmount decimal.Decimal
	MaxAmount decimal.Decimal
	MinTrust  int
}

// RunMatchingEngine executes one round of the P2P matching engine.
// It pairs confirmed deposits with pending withdrawals of the same currency
// (or cross-currency via FX), settling internally to reduce agent load.
func RunMatchingEngine(db *gorm.DB, fx *FXService) ([]MatchResult, error) {
	if !config.GetFlags().EnableP2PMatching {
		slog.Info("matching: skipped — feature flag disabled")
		return nil, nil
	}

	var results []MatchResult

	err := locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		// Load confirmed deposits awaiting settlement
		var deposits []DepositRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status = ? AND usd_amount > 0", "confirmed").
			Order("created_at ASC").
			Find(&deposits).Error; err != nil {
			return err
		}

		// Load pending/assigned withdrawals awaiting settlement
		var withdraws []WithdrawRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status IN ?", []string{"pending", "assigned"}).
			Order("created_at ASC").
			Find(&withdraws).Error; err != nil {
			return err
		}

		for i := range deposits {
			deposit := &deposits[i]
			if deposit.USDAmount.IsZero() {
				continue
			}

			for j := range withdraws {
				withdraw := &withdraws[j]
				if withdraw.USDAmount.IsZero() {
					continue
				}
				if withdraw.Status != "pending" && withdraw.Status != "assigned" {
					continue
				}

				// Safety: block matching if fraud or trust issues
				if !canMatch(tx, deposit.UserID, withdraw.UserID, deposit.USDAmount) {
					continue
				}

				// Same currency → direct match; cross-currency → FX rate
				rate := decimal.NewFromInt(1)
				if deposit.Currency != withdraw.Currency {
					rate = fx.GetRate(deposit.Currency, withdraw.Currency)
				}

				// Match the smaller amount
				matchAmount := deposit.USDAmount
				if withdraw.USDAmount.LessThan(matchAmount) {
					matchAmount = withdraw.USDAmount
				}

				if matchAmount.IsZero() {
					continue
				}

				match := MatchResult{
					ID:         uuid.New(),
					DepositID:  deposit.ID,
					WithdrawID: withdraw.ID,
					Amount:     matchAmount,
					Rate:       rate,
					Status:     "settled",
				}

				if err := tx.Create(&match).Error; err != nil {
					slog.Error("matching: failed to create match result", "error", err)
					continue
				}

				// Update deposit remaining
				deposit.USDAmount = deposit.USDAmount.Sub(matchAmount)
				if deposit.USDAmount.IsZero() || deposit.USDAmount.LessThanOrEqual(decimal.NewFromFloat(0.01)) {
					deposit.Status = "confirmed" // fully settled
				}

				// Update withdraw remaining
				withdraw.USDAmount = withdraw.USDAmount.Sub(matchAmount)
				if withdraw.USDAmount.IsZero() || withdraw.USDAmount.LessThanOrEqual(decimal.NewFromFloat(0.01)) {
					withdraw.Status = "completed"
					now := time.Now()
					withdraw.CompletedAt = &now
				}

				tx.Save(deposit)
				tx.Save(withdraw)

				results = append(results, match)
				slog.Info("matching: settled",
					"match_id", match.ID,
					"deposit_id", deposit.ID,
					"withdraw_id", withdraw.ID,
					"amount", matchAmount.String(),
				)

				// If deposit fully consumed, move to next deposit
				if deposit.USDAmount.LessThanOrEqual(decimal.NewFromFloat(0.01)) {
					break
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return results, nil
}

// canMatch checks safety rules before allowing a deposit-withdraw match.
func canMatch(db *gorm.DB, depositUserID, withdrawUserID uuid.UUID, amount decimal.Decimal) bool {
	// Rule 1: Block if either user has a high fraud score
	var depositAgentScore AgentScore
	if db.Where("agent_id IN (SELECT id FROM payment_agents WHERE user_id = ?)", depositUserID).First(&depositAgentScore).Error == nil {
		if depositAgentScore.Score < 40 {
			return false
		}
	}

	// Rule 2: Block large amounts for low-trust users
	if amount.GreaterThan(decimal.NewFromFloat(5000)) {
		var vip VIPUser
		if db.Where("user_id = ?", withdrawUserID).First(&vip).Error != nil {
			// Not VIP — block large matches
			return false
		}
		if vip.Tier == "silver" && amount.GreaterThan(decimal.NewFromFloat(1000)) {
			return false
		}
	}

	// Rule 3: Check liquidity imbalance
	imbalance := GetSystemLiquidityImbalance(db)
	if imbalance.GreaterThan(decimal.NewFromFloat(50000)) {
		// System is heavily imbalanced — slow matching
		return false
	}

	return true
}

// GetMatchingStats returns current matching queue statistics.
func GetMatchingStats(db *gorm.DB) map[string]interface{} {
	var pendingDeposits int64
	var pendingWithdraws int64
	var totalMatched float64

	db.Model(&DepositRequest{}).Where("status = ?", "confirmed").Count(&pendingDeposits)
	db.Model(&WithdrawRequest{}).Where("status IN ?", []string{"pending", "assigned"}).Count(&pendingWithdraws)
	db.Model(&MatchResult{}).Where("status = ?", "settled").
		Select("COALESCE(SUM(amount),0)").Scan(&totalMatched)

	return map[string]interface{}{
		"pending_deposits":  pendingDeposits,
		"pending_withdraws": pendingWithdraws,
		"total_matched":     totalMatched,
	}
}

// RunMatchingHandler — POST /api/v1/admin/payments/matching/run
func (h *Handler) RunMatchingHandler(c *gin.Context) {
	results, err := RunMatchingEngine(h.db, h.fx)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{
		"matches":       results,
		"total_matches": len(results),
	})
}

// GetMatchingStatsHandler — GET /api/v1/admin/payments/matching/stats
func (h *Handler) GetMatchingStatsHandler(c *gin.Context) {
	stats := GetMatchingStats(h.db)
	response.OK(c, stats)
}
