package wallet

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/reputation"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TrustedAgent represents a verified KYC agent for P2P currency matching.
type TrustedAgent struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID            uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	KYCVerified       bool      `gorm:"default:false" json:"kyc_verified"`
	DepositGuarantee  float64   `gorm:"type:numeric(14,2);default:0" json:"deposit_guarantee"` // security deposit
	MaxDailyVolume    float64   `gorm:"type:numeric(14,2);default:5000" json:"max_daily_volume"`
	IsActive          bool      `gorm:"default:true" json:"is_active"`
	ApprovedBy        *uuid.UUID `gorm:"type:uuid" json:"approved_by,omitempty"`
	ApprovedAt        *time.Time `json:"approved_at,omitempty"`
	TotalTransactions int       `gorm:"default:0" json:"total_transactions"`
	TotalVolume       float64   `gorm:"type:numeric(14,2);default:0" json:"total_volume"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (TrustedAgent) TableName() string { return "trusted_agents" }

// AgentMatchRequest represents a P2P currency match request.
type AgentMatchRequest struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	AgentID     uuid.UUID `gorm:"type:uuid;not null;index" json:"agent_id"`
	Amount      float64   `gorm:"type:numeric(14,2);not null" json:"amount"`
	FromCurrency string   `gorm:"size:3;not null" json:"from_currency"`
	ToCurrency   string   `gorm:"size:3;not null" json:"to_currency"`
	Status      string    `gorm:"size:20;default:'pending'" json:"status"` // pending, approved, completed, cancelled
	ReviewedBy  *uuid.UUID `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (AgentMatchRequest) TableName() string { return "agent_match_requests" }

// IsTrustedAgent checks if a user is an approved trusted agent.
func IsTrustedAgent(db *gorm.DB, userID uuid.UUID) bool {
	var agent TrustedAgent
	if err := db.Where("user_id=? AND is_active=? AND kyc_verified=?", userID, true, true).
		First(&agent).Error; err != nil {
		return false
	}
	return true
}

// GetTrustedAgent returns the trusted agent record for a user.
func GetTrustedAgent(db *gorm.DB, userID uuid.UUID) *TrustedAgent {
	var agent TrustedAgent
	if err := db.Where("user_id=?", userID).First(&agent).Error; err != nil {
		return nil
	}
	return &agent
}

// ApproveTrustedAgent approves a user as a trusted agent (admin action).
func ApproveTrustedAgent(db *gorm.DB, agentID, approvedBy uuid.UUID) error {
	return db.Model(&TrustedAgent{}).Where("id=?", agentID).
		Updates(map[string]interface{}{
			"is_active":   true,
			"kyc_verified": true,
			"approved_by": approvedBy,
			"approved_at": time.Now(),
		}).Error
}

// RequestAgentMatch creates a P2P currency match request (not auto-matched).
func RequestAgentMatch(db *gorm.DB, userID, agentID uuid.UUID, amount float64, fromCurr, toCurr string) (*AgentMatchRequest, error) {
	// Verify agent is trusted
	if !IsTrustedAgent(db, agentID) {
		return nil, fmt.Errorf("agent is not a verified trusted agent")
	}

	// Check user trust level
	userTrust := reputation.GetOverallScore(db, userID)
	if userTrust < 30 {
		return nil, fmt.Errorf("user trust score too low for P2P matching (%.0f)", userTrust)
	}

	// Check agent daily volume limit
	agent := GetTrustedAgent(db, agentID)
	if agent != nil {
		var todayVolume float64
		db.Table("agent_match_requests").
			Where("agent_id=? AND status IN ? AND created_at>?", agentID,
				[]string{"approved", "completed"}, time.Now().Truncate(24*time.Hour)).
			Select("COALESCE(SUM(amount),0)").Scan(&todayVolume)
		if todayVolume+amount > agent.MaxDailyVolume {
			return nil, fmt.Errorf("agent daily volume limit reached ($%.2f/$%.2f)", todayVolume, agent.MaxDailyVolume)
		}
	}

	// Per-user daily limit
	userDailyLimit := getUserP2PDailyLimit(db, userID)
	var userTodayVolume float64
	db.Table("agent_match_requests").
		Where("user_id=? AND status IN ? AND created_at>?", userID,
			[]string{"approved", "completed"}, time.Now().Truncate(24*time.Hour)).
		Select("COALESCE(SUM(amount),0)").Scan(&userTodayVolume)
	if userTodayVolume+amount > userDailyLimit {
		return nil, fmt.Errorf("user daily P2P limit reached ($%.2f/$%.2f)", userTodayVolume, userDailyLimit)
	}

	req := AgentMatchRequest{
		UserID:      userID,
		AgentID:     agentID,
		Amount:      amount,
		FromCurrency: fromCurr,
		ToCurrency:   toCurr,
		Status:      "pending",
	}

	if err := db.Create(&req).Error; err != nil {
		return nil, err
	}

	slog.Info("wallet: agent match requested",
		"user_id", userID, "agent_id", agentID,
		"amount", amount, "from", fromCurr, "to", toCurr,
	)

	return &req, nil
}

// ApproveAgentMatch approves a match request (manual approval, not auto).
func ApproveAgentMatch(db *gorm.DB, requestID, reviewerID uuid.UUID) error {
	return db.Model(&AgentMatchRequest{}).Where("id=? AND status=?", requestID, "pending").
		Updates(map[string]interface{}{
			"status":      "approved",
			"reviewed_by": reviewerID,
		}).Error
}

// MonitorAgentTransactions returns recent agent transactions for monitoring.
func MonitorAgentTransactions(db *gorm.DB, agentID uuid.UUID, since time.Time) []AgentMatchRequest {
	var reqs []AgentMatchRequest
	db.Where("agent_id=? AND created_at>?", agentID, since).
		Order("created_at DESC").Limit(100).Find(&reqs)
	return reqs
}

// getUserP2PDailyLimit returns the P2P daily limit based on trust level.
func getUserP2PDailyLimit(db *gorm.DB, userID uuid.UUID) float64 {
	score := reputation.GetOverallScore(db, userID)
	level := reputation.GetTrustLevel(score)

	switch level {
	case reputation.TrustLow:
		return 100
	case reputation.TrustNormal:
		return 1000
	default: // TrustHigh
		return 5000
	}
}

// RecordAgentTransaction records a completed agent transaction for monitoring.
func RecordAgentTransaction(db *gorm.DB, requestID uuid.UUID) error {
	var req AgentMatchRequest
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id=?", requestID).First(&req).Error; err != nil {
		return err
	}

	req.Status = "completed"
	db.Save(&req)

	// Update agent stats
	db.Model(&TrustedAgent{}).Where("id=?", req.AgentID).
		Updates(map[string]interface{}{
			"total_transactions": gorm.Expr("total_transactions + 1"),
			"total_volume":      gorm.Expr("total_volume + ?", req.Amount),
		})

	slog.Info("wallet: agent transaction completed",
		"request_id", requestID, "agent_id", req.AgentID,
		"amount", req.Amount,
	)

	return nil
}
