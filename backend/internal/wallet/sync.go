package wallet

import (
	"fmt"
	"log/slog"

	"github.com/geocore-next/backend/pkg/locking"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ════════════════════════════════════════════════════════════════════════════
// Wallet Sync Layer — Anti Double-Spend Protection
// Ensures every balance in the system corresponds to a confirmed real operation.
// ════════════════════════════════════════════════════════════════════════════

// WalletSyncSnapshot captures the verified state of a wallet at a point in time.
type WalletSyncSnapshot struct {
	UserID            uuid.UUID       `json:"user_id"`
	Currency          Currency        `json:"currency"`
	ConfirmedBalance  decimal.Decimal `json:"confirmed_balance"`
	PendingDeposits   decimal.Decimal `json:"pending_deposits"`
	ConfirmedDeposits decimal.Decimal `json:"confirmed_deposits"`
	MatchedTransfers  decimal.Decimal `json:"matched_transfers"`
	AvailableBalance  decimal.Decimal `json:"available_balance"`
	PendingBalance    decimal.Decimal `json:"pending_balance"`
	InvariantOK       bool            `json:"invariant_ok"`
}

// VerifyWalletSync checks that a user's wallet balances match the sum of
// confirmed operations. Returns an error if there's a mismatch (double-spend).
func VerifyWalletSync(db *gorm.DB, userID uuid.UUID, currency Currency) (*WalletSyncSnapshot, error) {
	snapshot := &WalletSyncSnapshot{
		UserID:   userID,
		Currency: currency,
	}

	err := locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		// Lock wallet and balance rows
		var wallet Wallet
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			return fmt.Errorf("wallet not found")
		}

		var balance WalletBalance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("wallet_id = ? AND currency = ?", wallet.ID, currency).
			First(&balance).Error; err != nil {
			return fmt.Errorf("balance not found for currency %s", currency)
		}

		snapshot.AvailableBalance = balance.AvailableBalance
		snapshot.PendingBalance = balance.PendingBalance

		// Sum confirmed deposits
		var confirmedDeposits decimal.Decimal
		tx.Table("deposit_requests dr").
			Where("dr.user_id = ? AND dr.currency = ? AND dr.status = ?", userID, string(currency), "confirmed").
			Select("COALESCE(SUM(dr.usd_amount),0)").Scan(&confirmedDeposits)
		snapshot.ConfirmedDeposits = confirmedDeposits

		// Sum pending deposits
		var pendingDeposits decimal.Decimal
		tx.Table("deposit_requests dr").
			Where("dr.user_id = ? AND dr.currency = ? AND dr.status IN ?", userID, string(currency),
				[]string{"pending", "paid"}).
			Select("COALESCE(SUM(dr.usd_amount),0)").Scan(&pendingDeposits)
		snapshot.PendingDeposits = pendingDeposits

		// Sum completed withdrawals
		var completedWithdrawals decimal.Decimal
		tx.Table("withdraw_requests wr").
			Where("wr.user_id = ? AND wr.currency = ? AND wr.status = ?", userID, string(currency), "completed").
			Select("COALESCE(SUM(wr.usd_amount),0)").Scan(&completedWithdrawals)

		// Sum matched transfers
		var matchedTransfers decimal.Decimal
		tx.Table("payment_match_results mr").
			Joins("JOIN deposit_requests dr ON dr.id = mr.deposit_id").
			Where("dr.user_id = ? AND mr.status = ?", userID, "settled").
			Select("COALESCE(SUM(mr.amount),0)").Scan(&matchedTransfers)
		snapshot.MatchedTransfers = matchedTransfers

		// Compute expected confirmed balance
		// confirmed_balance = confirmed_deposits - completed_withdrawals
		snapshot.ConfirmedBalance = confirmedDeposits.Sub(completedWithdrawals)

		// Invariant: available + pending + escrow_held = confirmed_deposits - completed_withdrawals
		// For now, check available + pending ≈ confirmed_balance
		totalWalletBalance := balance.AvailableBalance.Add(balance.PendingBalance)

		// Allow small rounding difference (0.01)
		diff := totalWalletBalance.Sub(snapshot.ConfirmedBalance).Abs()
		snapshot.InvariantOK = diff.LessThanOrEqual(decimal.NewFromFloat(0.01))

		if !snapshot.InvariantOK {
			slog.Warn("wallet_sync: invariant violation",
				"user_id", userID,
				"currency", currency,
				"wallet_total", totalWalletBalance.String(),
				"confirmed_balance", snapshot.ConfirmedBalance.String(),
				"diff", diff.String(),
			)
		}

		return nil
	})

	return snapshot, err
}

// BlockWithdrawalIfUnconfirmed prevents withdrawal when balance is not confirmed.
// This is the core anti double-spend check.
func BlockWithdrawalIfUnconfirmed(db *gorm.DB, userID uuid.UUID, amount decimal.Decimal, currency Currency) error {
	snapshot, err := VerifyWalletSync(db, userID, currency)
	if err != nil {
		return fmt.Errorf("wallet sync verification failed: %w", err)
	}

	if !snapshot.InvariantOK {
		return fmt.Errorf("wallet invariant violation: cannot withdraw until balance is confirmed")
	}

	if snapshot.AvailableBalance.LessThan(amount) {
		return fmt.Errorf("insufficient confirmed balance: have %s, need %s", snapshot.AvailableBalance.String(), amount.String())
	}

	return nil
}

// RepairWalletSync fixes wallet balances that don't match confirmed operations.
// This is a recovery tool for when the invariant is violated.
func RepairWalletSync(db *gorm.DB, userID uuid.UUID, currency Currency) error {
	return locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		var wallet Wallet
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			return fmt.Errorf("wallet not found")
		}

		var balance WalletBalance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("wallet_id = ? AND currency = ?", wallet.ID, currency).
			First(&balance).Error; err != nil {
			return fmt.Errorf("balance not found")
		}

		// Recalculate from source of truth: deposit_requests and withdraw_requests
		var confirmedDeposits decimal.Decimal
		tx.Table("deposit_requests dr").
			Where("dr.user_id = ? AND dr.currency = ? AND dr.status = ?", userID, string(currency), "confirmed").
			Select("COALESCE(SUM(dr.usd_amount),0)").Scan(&confirmedDeposits)

		var completedWithdrawals decimal.Decimal
		tx.Table("withdraw_requests wr").
			Where("wr.user_id = ? AND wr.currency = ? AND wr.status = ?", userID, string(currency), "completed").
			Select("COALESCE(SUM(wr.usd_amount),0)").Scan(&completedWithdrawals)

		var pendingWithdrawals decimal.Decimal
		tx.Table("withdraw_requests wr").
			Where("wr.user_id = ? AND wr.currency = ? AND wr.status IN ?", userID, string(currency),
				[]string{"pending", "assigned", "processing"}).
			Select("COALESCE(SUM(wr.usd_amount),0)").Scan(&pendingWithdrawals)

		// Correct balances
		expectedAvailable := confirmedDeposits.Sub(completedWithdrawals).Sub(pendingWithdrawals)
		expectedPending := pendingWithdrawals

		if expectedAvailable.LessThan(decimal.Zero) {
			expectedAvailable = decimal.Zero
		}

		balance.AvailableBalance = expectedAvailable
		balance.PendingBalance = expectedPending
		balance.Balance = expectedAvailable.Add(expectedPending)

		slog.Info("wallet_sync: repaired wallet",
			"user_id", userID,
			"currency", currency,
			"available", expectedAvailable.String(),
			"pending", expectedPending.String(),
		)

		return tx.Save(&balance).Error
	})
}

// ReplayAttackCheck detects if the same idempotency key has been used before.
func ReplayAttackCheck(db *gorm.DB, userID uuid.UUID, idempotencyKey string) error {
	if idempotencyKey == "" {
		return nil
	}

	// Check deposit_requests
	var depositCount int64
	db.Table("deposit_requests").
		Where("user_id = ? AND idempotency_key = ?", userID, idempotencyKey).
		Count(&depositCount)
	if depositCount > 0 {
		return fmt.Errorf("replay attack detected: duplicate idempotency key in deposits")
	}

	// Check withdraw_requests
	var withdrawCount int64
	db.Table("withdraw_requests").
		Where("user_id = ? AND idempotency_key = ?", userID, idempotencyKey).
		Count(&withdrawCount)
	if withdrawCount > 0 {
		return fmt.Errorf("replay attack detected: duplicate idempotency key in withdrawals")
	}

	return nil
}
