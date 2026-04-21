// Package locking provides deterministic lock ordering for financial transactions.
//
// RULES (MUST be followed by ALL financial handlers):
//
//  1. Always lock rows in ascending primary-key order within each resource type.
//  2. Lock resource types in this global order:
//     Escrow → Wallet → WalletBalance → WalletTransaction → OutboxEvent
//  3. When locking multiple rows of the same type (e.g. two wallets),
//     sort by primary key (UUID string comparison) and lock in that order.
//  4. Never lock a higher-order resource before a lower-order one.
//  5. Never skip a level — if you need Wallet + WalletBalance, lock Wallet first.
//
// Violating these rules creates deadlock windows under concurrent financial operations.
package locking

import (
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OrderedWalletLocks locks multiple Wallet rows in deterministic UUID order.
// Pass the *already-resolved* user IDs; this function sorts them and locks
// the corresponding wallet rows via SELECT FOR UPDATE.
//
// Returns wallets in the SAME order as the input userIDs (not sorted order),
// so callers can index into the result predictably.
func OrderedWalletLocks(tx *gorm.DB, userIDs []string) (wallets []WalletRow, err error) {
	type entry struct {
		inputIdx int
		userID   string
	}
	entries := make([]entry, len(userIDs))
	for i, uid := range userIDs {
		entries[i] = entry{inputIdx: i, userID: uid}
	}
	// Sort by user_id string (deterministic lock order)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].userID < entries[j].userID
	})

	wallets = make([]WalletRow, len(userIDs))
	for _, e := range entries {
		var w WalletRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Table("wallets").
			Where("user_id = ?", e.userID).
			First(&w).Error; err != nil {
			return nil, err
		}
		wallets[e.inputIdx] = w
	}
	return wallets, nil
}

// OrderedBalanceLocks locks multiple WalletBalance rows in deterministic wallet_id order.
// Pass pairs of (walletID, currency); this function sorts by walletID and locks.
//
// Returns balances in the SAME order as the input pairs.
func OrderedBalanceLocks(tx *gorm.DB, pairs []BalanceRef) (balances []BalanceRow, err error) {
	sorted := make([]BalanceRef, len(pairs))
	copy(sorted, pairs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].WalletID.String() < sorted[j].WalletID.String()
	})

	balances = make([]BalanceRow, len(pairs))
	// Build a lookup from walletID+currency → input index
	idxMap := make(map[string]int, len(pairs))
	for i, p := range pairs {
		idxMap[p.WalletID.String()+":"+string(p.Currency)] = i
	}

	for _, s := range sorted {
		var b BalanceRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Table("wallet_balances").
			Where("wallet_id = ? AND currency = ?", s.WalletID, s.Currency).
			First(&b).Error; err != nil {
			return nil, err
		}
		key := s.WalletID.String() + ":" + string(s.Currency)
		balances[idxMap[key]] = b
	}
	return balances, nil
}

// SortUUIDs returns UUIDs sorted by string value (ascending).
// Use this to determine lock order before issuing SELECT FOR UPDATE.
func SortUUIDs(ids []uuid.UUID) []uuid.UUID {
	sorted := make([]uuid.UUID, len(ids))
	copy(sorted, ids)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].String() < sorted[j].String()
	})
	return sorted
}

// SortStrings returns strings sorted ascending.
func SortStrings(ids []string) []string {
	sorted := make([]string, len(ids))
	copy(sorted, ids)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	return sorted
}

// ── Lightweight row types for generic locking ──────────────────────────────

type WalletRow struct {
	ID         uuid.UUID `gorm:"primaryKey"`
	UserID     string
	IsActive   bool
	DailyLimit float64
}

type BalanceRow struct {
	ID               uuid.UUID `gorm:"primaryKey"`
	WalletID         uuid.UUID
	Currency         string
	Balance          float64
	AvailableBalance float64
	PendingBalance   float64
}

type BalanceRef struct {
	WalletID uuid.UUID
	Currency string
}

// ── Deadlock Retry ──────────────────────────────────────────────────────────

const (
	// MaxRetries is the maximum number of retries for deadlock/lock_timeout errors.
	MaxRetries = 3

	// BaseDelay is the initial backoff duration for retry attempts.
	BaseDelay = 50 * time.Millisecond
)

// isDeadlockError returns true if the error is a PostgreSQL deadlock or lock timeout.
func isDeadlockError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "deadlock_detected") ||
		strings.Contains(msg, "lock_timeout") ||
		strings.Contains(msg, "could not serialize access") ||
		strings.Contains(msg, "40p01") // SQLSTATE for deadlock
}

// RetryOnDeadlock wraps a GORM transaction function with automatic retry on
// deadlock_detected and lock_timeout errors. Uses exponential backoff:
// 50ms → 100ms → 200ms. Returns the last error if all retries are exhausted.
//
// Usage:
//
//	err := locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
//	    // financial transaction with SELECT FOR UPDATE
//	    return nil
//	})
func RetryOnDeadlock(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	var lastErr error
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		lastErr = db.Transaction(fn)
		if lastErr == nil {
			return nil
		}
		if !isDeadlockError(lastErr) {
			return lastErr // non-retryable error, return immediately
		}
		if attempt < MaxRetries {
			delay := BaseDelay * time.Duration(1<<attempt) // 50ms, 100ms, 200ms
			slog.Warn("locking: deadlock detected — retrying",
				"attempt", attempt+1,
				"delay_ms", delay.Milliseconds(),
				"error", lastErr.Error())
			time.Sleep(delay)
		}
	}
	slog.Error("locking: exhausted retries on deadlock",
		"attempts", MaxRetries+1,
		"error", lastErr.Error())
	return lastErr
}
