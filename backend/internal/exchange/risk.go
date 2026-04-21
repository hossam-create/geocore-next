package exchange

// risk.go — Part 6: Risk Engine.
//
// Detects velocity abuse, collusion (same-pair repeat matching),
// circular trades, and fake confirmations. Never touches wallet or escrow.

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	RiskLow    = "low"
	RiskMedium = "medium"
	RiskHigh   = "high"

	velocityWindowMins  = 60
	velocityMaxRequests = 10
	collusionThreshold  = 3 // same user-pair matched > 3 times
)

// ExchangeRiskFlag records a risk signal for a user or match.
type ExchangeRiskFlag struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"                        json:"user_id"`
	MatchID   *uuid.UUID `gorm:"type:uuid;index"                                 json:"match_id,omitempty"`
	FlagType  string     `gorm:"size:30;not null;index"                          json:"flag_type"` // velocity|collusion|circular|fake_proof
	Severity  string     `gorm:"size:10;not null"                                json:"severity"`
	Detail    string     `gorm:"type:text"                                       json:"detail"`
	Resolved  bool       `gorm:"not null;default:false"                          json:"resolved"`
	CreatedAt time.Time  `json:"created_at"`
}

func (ExchangeRiskFlag) TableName() string { return "exchange_risk_flags" }

// RiskCheckResult is the output of CheckExchangeRisk / CheckMatchRisk.
type RiskCheckResult struct {
	Allowed   bool     `json:"allowed"`
	RiskLevel string   `json:"risk_level"`
	Flags     []string `json:"flags,omitempty"`
	Reason    string   `json:"reason,omitempty"`
}

// CheckExchangeRisk runs request-level risk signals (velocity + active flags).
func CheckExchangeRisk(db *gorm.DB, userID uuid.UUID) RiskCheckResult {
	result := RiskCheckResult{Allowed: true, RiskLevel: RiskLow}

	if detectVelocity(db, userID) {
		result.Flags = append(result.Flags, "velocity")
		result.RiskLevel = RiskHigh
		result.Allowed = false
		result.Reason = "Too many requests in a short time. Please wait before creating more."
		_ = recordFlag(db, userID, nil, "velocity", RiskHigh, "velocity threshold exceeded")
		return result
	}

	var activeHighFlags int64
	db.Model(&ExchangeRiskFlag{}).
		Where("user_id=? AND severity=? AND resolved=?", userID, RiskHigh, false).
		Count(&activeHighFlags)
	if activeHighFlags > 0 {
		result.Flags = append(result.Flags, "active_high_risk_flag")
		result.RiskLevel = RiskHigh
		result.Allowed = false
		result.Reason = "Account has unresolved risk flags. Please contact support."
	}
	return result
}

// CheckMatchRisk runs match-level signals (collusion + circular trades).
func CheckMatchRisk(db *gorm.DB, reqA, reqB *ExchangeRequest) RiskCheckResult {
	result := RiskCheckResult{Allowed: true, RiskLevel: RiskLow}

	if detectCollusion(db, reqA.UserID, reqB.UserID) {
		result.Flags = append(result.Flags, "collusion")
		if result.RiskLevel == RiskLow {
			result.RiskLevel = RiskMedium
		}
		_ = recordFlag(db, reqA.UserID, nil, "collusion", RiskMedium, "repeated matching with same counterparty")
	}

	if detectCircular(db, reqA.UserID, reqB.UserID) {
		result.Flags = append(result.Flags, "circular_trade")
		result.RiskLevel = RiskHigh
		result.Allowed = false
		result.Reason = "Circular trade pattern detected. This match has been blocked."
		_ = recordFlag(db, reqA.UserID, nil, "circular_trade", RiskHigh, "circular exchange pattern detected")
	}
	return result
}

// FlagFakeProof records a suspicious proof upload and logs it.
func FlagFakeProof(db *gorm.DB, matchID, userID uuid.UUID, detail string) {
	_ = recordFlag(db, userID, &matchID, "fake_proof", RiskHigh, detail)
	slog.Warn("exchange: fake proof flagged", "match_id", matchID, "user_id", userID)
}

// RiskLevelForUser returns the aggregate risk level based on unresolved flags.
func RiskLevelForUser(db *gorm.DB, userID uuid.UUID) string {
	var highCount, medCount int64
	db.Model(&ExchangeRiskFlag{}).Where("user_id=? AND severity=? AND resolved=?", userID, RiskHigh, false).Count(&highCount)
	db.Model(&ExchangeRiskFlag{}).Where("user_id=? AND severity=? AND resolved=?", userID, RiskMedium, false).Count(&medCount)
	switch {
	case highCount > 0:
		return RiskHigh
	case medCount > 1:
		return RiskMedium
	default:
		return RiskLow
	}
}

// ─── internal detectors ──────────────────────────────────────────────────────

func detectVelocity(db *gorm.DB, userID uuid.UUID) bool {
	window := time.Now().Add(-velocityWindowMins * time.Minute)
	var count int64
	db.Model(&ExchangeRequest{}).
		Where("user_id=? AND created_at>? AND is_system_generated=?", userID, window, false).
		Count(&count)
	return count >= velocityMaxRequests
}

func detectCollusion(db *gorm.DB, userA, userB uuid.UUID) bool {
	var count int64
	db.Raw(`SELECT COUNT(*) FROM exchange_matches em
		JOIN exchange_requests ra ON ra.id=em.request_a_id
		JOIN exchange_requests rb ON rb.id=em.request_b_id
		WHERE (ra.user_id=? AND rb.user_id=?) OR (ra.user_id=? AND rb.user_id=?)`,
		userA, userB, userB, userA).Scan(&count)
	return count > int64(collusionThreshold)
}

func detectCircular(db *gorm.DB, userA, userB uuid.UUID) bool {
	// 2-hop check: A→B→A within 24h
	var count int64
	db.Raw(`SELECT COUNT(*) FROM exchange_matches em1
		JOIN exchange_requests ra1 ON ra1.id=em1.request_a_id
		JOIN exchange_requests rb1 ON rb1.id=em1.request_b_id
		JOIN exchange_matches em2 ON em2.id != em1.id
		JOIN exchange_requests ra2 ON ra2.id=em2.request_a_id
		JOIN exchange_requests rb2 ON rb2.id=em2.request_b_id
		WHERE ra1.user_id=? AND rb1.user_id=? AND ra2.user_id=? AND rb2.user_id=?
		AND em1.created_at > NOW() - INTERVAL '24 hours'`,
		userA, userB, userB, userA).Scan(&count)
	return count > 0
}

func recordFlag(db *gorm.DB, userID uuid.UUID, matchID *uuid.UUID, flagType, severity, detail string) error {
	return db.Create(&ExchangeRiskFlag{
		ID:       uuid.New(),
		UserID:   userID,
		MatchID:  matchID,
		FlagType: flagType,
		Severity: severity,
		Detail:   detail,
	}).Error
}
