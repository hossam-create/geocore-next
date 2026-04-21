package compliance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Categories used across the audit log. Stable strings — do NOT rename
// without a backfill script since analytics queries depend on them.
const (
	CategoryExchange = "exchange"
	CategoryPayout   = "payout"
	CategoryDispute  = "dispute"
	CategoryAdmin    = "admin"
	CategoryConsent  = "consent"
	CategoryGDPR     = "gdpr"
)

// genesisHash is the 32-byte zero hash used as prev_hash for the first row.
const genesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// Mutex serialising chain appends. Without this, two concurrent writers could
// read the same "latest row" and produce duplicate prev_hash, silently breaking
// the chain. DB-level advisory lock would be ideal; a process-level mutex is
// sufficient for single-instance deployments and is a safe default elsewhere.
var appendMu sync.Mutex

// LogComplianceEvent appends an immutable audit row.
// Safe to call from any handler; returns error so callers can surface failures
// to the user when appropriate (rare — most audit failures should not block UX).
func LogComplianceEvent(
	db *gorm.DB,
	category, action string,
	userID, actorID *uuid.UUID,
	resourceID, ip string,
	payload map[string]any,
) (*ComplianceAuditLog, error) {
	if db == nil {
		return nil, fmt.Errorf("compliance: nil db")
	}
	if payload == nil {
		payload = map[string]any{}
	}

	appendMu.Lock()
	defer appendMu.Unlock()

	// Load the previous row's hash — genesis if table is empty.
	prevHash := genesisHash
	var last ComplianceAuditLog
	if err := db.Order("id DESC").Limit(1).Find(&last).Error; err == nil && last.ID != 0 {
		prevHash = last.RowHash
	}

	row := ComplianceAuditLog{
		UserID:     userID,
		ActorID:    actorID,
		Category:   category,
		Action:     action,
		ResourceID: resourceID,
		Payload:    payload,
		IPAddress:  ip,
		PrevHash:   prevHash,
		CreatedAt:  time.Now().UTC(),
	}
	row.RowHash = computeRowHash(row)

	if err := db.Create(&row).Error; err != nil {
		return nil, fmt.Errorf("compliance: persist audit row: %w", err)
	}
	return &row, nil
}

// computeRowHash canonicalises the row and returns SHA-256 hex.
// Order matters; keep it stable. Do NOT include id/row_hash themselves
// (id is AUTO, row_hash is the output).
func computeRowHash(r ComplianceAuditLog) string {
	payloadBytes, _ := json.Marshal(r.Payload)
	parts := []string{
		r.PrevHash,
		uidStr(r.UserID),
		uidStr(r.ActorID),
		r.Category,
		r.Action,
		r.ResourceID,
		string(payloadBytes),
		r.IPAddress,
		r.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0x1e}) // ASCII record-separator to disambiguate concatenation
	}
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyChain walks the entire audit log and returns the first corrupted row,
// if any. Use from admin UI / nightly job to prove tamper-freedom.
func VerifyChain(db *gorm.DB) (valid bool, firstBadID int64, err error) {
	const batch = 1000
	prev := genesisHash
	offset := 0
	for {
		var rows []ComplianceAuditLog
		if err := db.Order("id ASC").Offset(offset).Limit(batch).Find(&rows).Error; err != nil {
			return false, 0, err
		}
		if len(rows) == 0 {
			return true, 0, nil
		}
		for _, r := range rows {
			if r.PrevHash != prev {
				return false, r.ID, nil
			}
			if computeRowHash(r) != r.RowHash {
				return false, r.ID, nil
			}
			prev = r.RowHash
		}
		offset += len(rows)
	}
}

func uidStr(u *uuid.UUID) string {
	if u == nil || *u == uuid.Nil {
		return ""
	}
	return u.String()
}
