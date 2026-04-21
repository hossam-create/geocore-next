package compliance

import (
	"time"

	"github.com/google/uuid"
)

// Consent types (GDPR Art. 7 + ePrivacy Directive).
const (
	ConsentTerms     = "terms"
	ConsentPrivacy   = "privacy"
	ConsentMarketing = "marketing"
	ConsentCookies   = "cookies"
)

// ConsentRecord is an append-only record of one consent event.
// Keep every accept/withdraw event — never update previous rows.
type ConsentRecord struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_consent_user_type"  json:"user_id"`
	Type      string    `gorm:"size:32;not null;index:idx_consent_user_type"    json:"type"`
	Accepted  bool      `gorm:"not null"                                        json:"accepted"`
	Version   string    `gorm:"size:32;not null;default:'v1'"                   json:"version"`
	IPAddress string    `gorm:"size:64"                                         json:"ip_address,omitempty"`
	UserAgent string    `gorm:"size:512"                                        json:"user_agent,omitempty"`
	CreatedAt time.Time `gorm:"index"                                            json:"created_at"`
}

func (ConsentRecord) TableName() string { return "consent_records" }

// ComplianceAuditLog is an immutable audit entry used for financial and
// dispute-related actions (exchange, payouts, disputes, admin overrides).
//
// Immutability is enforced at three layers:
//  1. No UpdatedAt / DeletedAt columns (hard append-only).
//  2. Each row contains the SHA-256 hash of the previous row (chain).
//  3. `row_hash` commits the row's own canonical JSON + the chain link,
//     so any tamper retroactively invalidates every later row.
type ComplianceAuditLog struct {
	ID         int64          `gorm:"primaryKey;autoIncrement"             json:"id"`
	UserID     *uuid.UUID     `gorm:"type:uuid;index"                      json:"user_id,omitempty"`
	ActorID    *uuid.UUID     `gorm:"type:uuid;index"                      json:"actor_id,omitempty"`
	Category   string         `gorm:"size:32;not null;index"               json:"category"` // exchange | payout | dispute | admin | consent
	Action     string         `gorm:"size:64;not null;index"               json:"action"`
	ResourceID string         `gorm:"size:128;index"                       json:"resource_id,omitempty"`
	Payload    map[string]any `gorm:"type:jsonb;serializer:json;default:'{}'" json:"payload"`
	IPAddress  string         `gorm:"size:64"                              json:"ip_address,omitempty"`
	PrevHash   string         `gorm:"size:64;not null"                     json:"prev_hash"`
	RowHash    string         `gorm:"size:64;not null;uniqueIndex"         json:"row_hash"`
	CreatedAt  time.Time      `gorm:"index"                                json:"created_at"`
}

func (ComplianceAuditLog) TableName() string { return "compliance_audit_log" }
