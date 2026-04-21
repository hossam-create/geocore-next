package compliance

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DataExport is the GDPR "Subject Access Request" bundle.
// Captured with raw maps / anys so we don't force coupling to every domain
// model; each section is a slice of rows fetched from its respective table.
type DataExport struct {
	ExportedAt   time.Time        `json:"exported_at"`
	UserID       uuid.UUID        `json:"user_id"`
	Notice       string           `json:"notice"`
	User         map[string]any   `json:"user"`
	Listings     []map[string]any `json:"listings,omitempty"`
	Orders       []map[string]any `json:"orders,omitempty"`
	Payments     []map[string]any `json:"payments,omitempty"`
	Withdrawals  []map[string]any `json:"withdrawals,omitempty"`
	Deposits     []map[string]any `json:"deposits,omitempty"`
	ChatMessages []map[string]any `json:"chat_messages,omitempty"`
	Reviews      []map[string]any `json:"reviews,omitempty"`
	Exchange     []map[string]any `json:"exchange_requests,omitempty"`
	Disputes     []map[string]any `json:"disputes,omitempty"`
	Consents     []ConsentRecord  `json:"consents,omitempty"`
	AuditLog     []map[string]any `json:"audit_log,omitempty"`
}

const exportNotice = "This archive contains personal data we hold about you under GDPR Art. 15 / CCPA §1798.100. Financial and audit records may be retained for up to 7 years per AMLD / BSA obligations even after account deletion."

// BuildUserExport assembles all user data from every table that stores it.
// Uses best-effort queries (Scan into map) so missing tables don't break export.
func BuildUserExport(db *gorm.DB, userID uuid.UUID) (DataExport, error) {
	out := DataExport{
		ExportedAt: time.Now().UTC(),
		UserID:     userID,
		Notice:     exportNotice,
	}

	// Core user — best-effort; if user table is missing we still return empty shell.
	user := map[string]any{}
	db.Raw(`SELECT * FROM users WHERE id = ?`, userID).Scan(&user)
	out.User = user

	// Helper for simple `WHERE user_id = ?` SELECTs.
	fetchByUser := func(table, col string) []map[string]any {
		rows := []map[string]any{}
		db.Raw("SELECT * FROM "+table+" WHERE "+col+" = ?", userID).Scan(&rows)
		return rows
	}

	out.Listings = fetchByUser("listings", "user_id")
	out.Orders = fetchByUser("orders", "buyer_id")
	out.Payments = fetchByUser("payments", "user_id")
	out.Withdrawals = fetchByUser("withdraw_requests", "user_id")
	out.Deposits = fetchByUser("deposit_requests", "user_id")
	out.Reviews = fetchByUser("reviews", "reviewer_id")
	out.Exchange = fetchByUser("exchange_requests", "user_id")
	out.Disputes = fetchByUser("disputes", "user_id")

	// Chat — messages where user is sender OR recipient.
	chat := []map[string]any{}
	db.Raw(`SELECT * FROM chat_messages WHERE sender_id = ? OR recipient_id = ?`, userID, userID).Scan(&chat)
	out.ChatMessages = chat

	// Consents — typed fetch so the enum is preserved.
	var consents []ConsentRecord
	db.Where("user_id = ?", userID).Order("created_at ASC").Find(&consents)
	out.Consents = consents

	// Audit — only rows relevant to the user (their own actions).
	audit := []map[string]any{}
	db.Raw(`SELECT id, category, action, resource_id, payload, created_at
	        FROM compliance_audit_log WHERE user_id = ?
	        ORDER BY created_at ASC`, userID).Scan(&audit)
	out.AuditLog = audit

	return out, nil
}
