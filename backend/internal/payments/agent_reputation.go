package payments

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/fraud"
	"github.com/geocore-next/backend/internal/reputation"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ════════════════════════════════════════════════════════════════════════════
// Agent Reputation System
// Tracks agent performance: success rate, dispute rate, volume, trust score.
// ════════════════════════════════════════════════════════════════════════════

// AgentScore holds the computed reputation score for a payment agent.
type AgentScore struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	AgentID     uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex" json:"agent_id"`
	SuccessRate decimal.Decimal `gorm:"type:decimal(5,2);not null;default:100" json:"success_rate"`
	DisputeRate decimal.Decimal `gorm:"type:decimal(5,2);not null;default:0" json:"dispute_rate"`
	Volume      decimal.Decimal `gorm:"type:decimal(15,2);not null;default:0" json:"volume"`
	Score       int             `gorm:"not null;default:50" json:"score"`
	TotalTx     int             `gorm:"not null;default:0" json:"total_tx"`
	SuccessTx   int             `gorm:"not null;default:0" json:"success_tx"`
	DisputeTx   int             `gorm:"not null;default:0" json:"dispute_tx"`
	FraudFlags  int             `gorm:"not null;default:0" json:"fraud_flags"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func (AgentScore) TableName() string { return "payment_agent_scores" }

// AgentScoreConfig holds thresholds for agent reputation.
type AgentScoreConfig struct {
	BlockScore       int
	WarningScore     int
	FraudDropAmount  int
	DisputePenalty   int
	SuccessBonus     int
	VolumeBonusEvery decimal.Decimal
	VolumeBonusPts   int
}

var DefaultAgentScoreConfig = AgentScoreConfig{
	BlockScore:       40,
	WarningScore:     60,
	FraudDropAmount:  30,
	DisputePenalty:   5,
	SuccessBonus:     2,
	VolumeBonusEvery: decimal.NewFromInt(10000),
	VolumeBonusPts:   1,
}

// RecordAgentSuccess increments the agent's success metrics.
func RecordAgentSuccess(db *gorm.DB, agentID uuid.UUID, amount decimal.Decimal) error {
	cfg := DefaultAgentScoreConfig

	return locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		var score AgentScore
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("agent_id = ?", agentID).First(&score).Error; err != nil {
			// Create initial score
			score = AgentScore{
				ID:      uuid.New(),
				AgentID: agentID,
				Score:   50,
			}
			if err := tx.Create(&score).Error; err != nil {
				return err
			}
		}

		score.TotalTx++
		score.SuccessTx++
		score.Volume = score.Volume.Add(amount)
		score.Score += cfg.SuccessBonus

		// Volume bonus
		if score.Volume.Div(cfg.VolumeBonusEvery).IntPart() > (score.Volume.Sub(amount)).Div(cfg.VolumeBonusEvery).IntPart() {
			score.Score += cfg.VolumeBonusPts
		}

		// Cap score at 100
		if score.Score > 100 {
			score.Score = 100
		}

		// Recalculate success rate
		if score.TotalTx > 0 {
			score.SuccessRate = decimal.NewFromInt(int64(score.SuccessTx)).
				Div(decimal.NewFromInt(int64(score.TotalTx))).Mul(decimal.NewFromInt(100))
		}

		return tx.Save(&score).Error
	})
}

// RecordAgentDispute decrements the agent's score for a dispute.
func RecordAgentDispute(db *gorm.DB, agentID uuid.UUID) error {
	cfg := DefaultAgentScoreConfig

	return locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		var score AgentScore
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("agent_id = ?", agentID).First(&score).Error; err != nil {
			return fmt.Errorf("agent score not found")
		}

		score.DisputeTx++
		score.Score -= cfg.DisputePenalty
		if score.Score < 0 {
			score.Score = 0
		}

		// Recalculate dispute rate
		if score.TotalTx > 0 {
			score.DisputeRate = decimal.NewFromInt(int64(score.DisputeTx)).
				Div(decimal.NewFromInt(int64(score.TotalTx))).Mul(decimal.NewFromInt(100))
		}

		if err := tx.Save(&score).Error; err != nil {
			return err
		}

		// Block agent if score too low
		if score.Score < cfg.BlockScore {
			var agent PaymentAgent
			if tx.Where("id = ?", agentID).First(&agent).Error == nil {
				reason := fmt.Sprintf("auto-blocked: score %d < %d", score.Score, cfg.BlockScore)
				tx.Model(&agent).Updates(map[string]interface{}{
					"status":            "suspended",
					"suspension_reason": reason,
				})
				slog.Warn("agent_reputation: agent auto-blocked", "agent_id", agentID, "score", score.Score)
			}
		}

		return nil
	})
}

// RecordAgentFraud applies a hard score drop for fraud detection.
func RecordAgentFraud(db *gorm.DB, agentID uuid.UUID, reason string) error {
	cfg := DefaultAgentScoreConfig

	return locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		var score AgentScore
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("agent_id = ?", agentID).First(&score).Error; err != nil {
			return fmt.Errorf("agent score not found")
		}

		score.FraudFlags++
		score.Score -= cfg.FraudDropAmount
		if score.Score < 0 {
			score.Score = 0
		}

		if err := tx.Save(&score).Error; err != nil {
			return err
		}

		// Suspend agent immediately on fraud
		var agent PaymentAgent
		if tx.Where("id = ?", agentID).First(&agent).Error == nil {
			tx.Model(&agent).Updates(map[string]interface{}{
				"status":            "suspended",
				"suspension_reason": "fraud_detected: " + reason,
			})
		}

		// Flag user in fraud system
		_ = fraud.FlagUser(tx, agent.UserID, "agent_fraud", fraud.SeverityHigh, reason)

		// Apply reputation penalty
		_ = reputation.ApplyPenalty(tx, nil, agent.UserID, "agent", reputation.PenaltyFraudFlagged)

		slog.Warn("agent_reputation: fraud detected, agent suspended", "agent_id", agentID, "reason", reason)
		return nil
	})
}

// GetAgentScore returns the score for a given agent.
func GetAgentScore(db *gorm.DB, agentID uuid.UUID) *AgentScore {
	var score AgentScore
	if db.Where("agent_id = ?", agentID).First(&score).Error != nil {
		return nil
	}
	return &score
}

// IsAgentBlocked returns true if the agent's score is below the block threshold.
func IsAgentBlocked(db *gorm.DB, agentID uuid.UUID) bool {
	score := GetAgentScore(db, agentID)
	if score == nil {
		return false
	}
	return score.Score < DefaultAgentScoreConfig.BlockScore
}
