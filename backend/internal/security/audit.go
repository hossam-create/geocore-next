package security

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SecurityAuditLog tracks authentication and security-sensitive events.
type SecurityAuditLog struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    *uuid.UUID     `gorm:"type:uuid" json:"user_id,omitempty"`
	EventType string         `gorm:"size:50;not null;index" json:"event_type"`
	IPAddress string         `gorm:"type:inet;not null" json:"ip_address"`
	UserAgent string         `gorm:"type:text" json:"user_agent,omitempty"`
	Details   map[string]any `gorm:"type:jsonb;serializer:json;default:'{}'" json:"details"`
	RiskScore int            `gorm:"default:0" json:"risk_score"`
	CreatedAt time.Time      `json:"created_at"`
}

func (SecurityAuditLog) TableName() string { return "security_audit_log" }

// Event types
const (
	EventLoginSuccess      = "login_success"
	EventLoginFailed       = "login_failed"
	EventPasswordChange    = "password_change"
	EventPasswordResetReq  = "password_reset_request"
	EventPasswordResetDone = "password_reset_complete"
	EventAccountCreated    = "account_created"
	EventAccountDeleted    = "account_deleted"
	EventSocialLogin       = "social_login"
	EventPaymentAttempt    = "payment_attempt"
	EventWalletTopUp       = "wallet_topup"
	EventRefundRequested   = "refund_requested"
	EventEscrowReleased    = "escrow_released"
	EventKYCSubmitted      = "kyc_submitted"
	EventAdminAction       = "admin_action"
	EventSessionRevoked    = "session_revoked"
	EventRateLimited       = "rate_limited"
)

// LogEvent writes a security event to the audit log.
func LogEvent(db *gorm.DB, c *gin.Context, userID *uuid.UUID, eventType string, details map[string]any) {
	if details == nil {
		details = map[string]any{}
	}
	entry := SecurityAuditLog{
		UserID:    userID,
		EventType: eventType,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Details:   details,
		RiskScore: assessRisk(eventType, details, c),
	}
	go db.Create(&entry) //nolint:errcheck
}

// LogEventDirect writes a security event without a Gin context (e.g., from background jobs).
func LogEventDirect(db *gorm.DB, userID *uuid.UUID, eventType, ip, ua string, details map[string]any) {
	if details == nil {
		details = map[string]any{}
	}
	entry := SecurityAuditLog{
		UserID:    userID,
		EventType: eventType,
		IPAddress: ip,
		UserAgent: ua,
		Details:   details,
		RiskScore: 0,
	}
	go db.Create(&entry) //nolint:errcheck
}

func assessRisk(eventType string, details map[string]any, c *gin.Context) int {
	score := 0
	switch eventType {
	case EventLoginFailed:
		score = 30
	case EventPasswordResetReq:
		score = 20
	case EventAccountDeleted:
		score = 40
	case EventRateLimited:
		score = 60
	case EventSessionRevoked:
		score = 70
		if reason, ok := details["reason"].(string); ok {
			switch reason {
			case "refresh_token_reuse_detected", "refresh_token_race_or_reuse":
				score = 90
			}
		}
	case EventAdminAction:
		score = 40
	}
	// Boost risk if user-agent looks automated
	ua := strings.ToLower(c.Request.UserAgent())
	if ua == "" || strings.Contains(ua, "curl") || strings.Contains(ua, "python") || strings.Contains(ua, "bot") {
		score += 20
	}
	if score > 100 {
		score = 100
	}
	return score
}

// MaskEmail masks an email for safe logging: ahmed@test.com → ah***@test.com
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 || len(parts[0]) < 2 {
		return "***"
	}
	return parts[0][:2] + "***@" + parts[1]
}
