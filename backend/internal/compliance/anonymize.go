package compliance

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AnonymizeUser implements GDPR Art. 17 ("right to erasure") while respecting
// regulatory retention obligations (AMLD / BSA — financial records must be
// kept up to 7 years).
//
// Strategy:
//   * Rewrite PII columns in `users` to deterministic anonymous placeholders
//     (email/phone become unique-per-user stubs to preserve uniqueness indexes).
//   * Soft-delete (deleted_at) the user row.
//   * Null out PII on linked domain tables (chat, listings, reviews).
//   * Leave financial rows (payments, exchange, audit, withdraw) intact —
//     they are referenced by immutable audit chain and legal retention.
//   * Append a GDPR audit event so deletion itself is provable later.
//
// Returns the anonymised email so the caller can echo it to the requester
// ("your account has been erased; reference id: deleted+<uuid>@…").
func AnonymizeUser(db *gorm.DB, userID uuid.UUID, actorID *uuid.UUID, ip string) (string, error) {
	if userID == uuid.Nil {
		return "", fmt.Errorf("compliance: nil user id")
	}

	anonEmail := fmt.Sprintf("deleted+%s@anon.local", userID.String())
	anonName := "Deleted User"

	err := db.Transaction(func(tx *gorm.DB) error {
		// 1. Rewrite PII in users.
		if err := tx.Exec(`
			UPDATE users
			SET name = ?,
			    email = ?,
			    phone = '',
			    avatar_url = '',
			    bio = '',
			    location = '',
			    password_hash = '',
			    ban_reason = 'gdpr_erasure',
			    is_active = false,
			    is_banned = true,
			    google_id = '',
			    apple_id = '',
			    facebook_id = '',
			    verification_token = '',
			    deleted_at = NOW(),
			    updated_at = NOW()
			WHERE id = ?
		`, anonName, anonEmail, userID).Error; err != nil {
			return fmt.Errorf("users rewrite: %w", err)
		}

		// 2. Redact chat message bodies authored by user — keep timestamps + IDs
		// for counterparty continuity, drop PII payload.
		tx.Exec(`UPDATE chat_messages SET content = '[redacted:gdpr]' WHERE sender_id = ?`, userID)

		// 3. Reviews — keep numeric rating for counterparty aggregates, wipe text.
		tx.Exec(`UPDATE reviews SET comment = '[redacted:gdpr]' WHERE reviewer_id = ?`, userID)

		// 4. Listings — mark inactive + wipe description / title PII.
		tx.Exec(`UPDATE listings SET title = '[removed]', description = '[removed]', status = 'removed' WHERE user_id = ?`, userID)

		// 5. Sessions / tokens — revoke everything active.
		tx.Exec(`DELETE FROM user_sessions WHERE user_id = ?`, userID)
		tx.Exec(`DELETE FROM refresh_tokens WHERE user_id = ?`, userID)

		return nil
	})
	if err != nil {
		return "", err
	}

	// 6. Append to immutable audit chain OUTSIDE the tx so chain order matches
	// wall-clock. If this fails we still consider erasure complete but log the
	// gap for manual reconciliation.
	_, auditErr := LogComplianceEvent(db, CategoryGDPR, "erasure_completed",
		&userID, actorID, userID.String(), ip,
		map[string]any{
			"anonymized_email": anonEmail,
			"reason":           "user_request",
		})
	if auditErr != nil {
		// Not fatal; erasure already happened. Surface for ops.
		return anonEmail, fmt.Errorf("compliance: erasure done, audit append failed: %w", auditErr)
	}

	return anonEmail, nil
}
