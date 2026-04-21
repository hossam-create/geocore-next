package wallet

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ── Invariant ─────────────────────────────────────────────────────────────────
//
// Every WalletBalance row must satisfy:
//
//	Balance == AvailableBalance + PendingBalance
//
// checkInvariant returns an error and logs CRITICAL if violated.
// Call this inside every DB transaction that mutates a WalletBalance row,
// after the mutation, before tx.Save().
func checkInvariant(b WalletBalance) error {
	expected := b.AvailableBalance.Add(b.PendingBalance)
	if !b.Balance.Equal(expected) {
		metrics.IncWalletInvariantViolation()
		slog.Error("WALLET INVARIANT VIOLATED",
			"severity", "CRITICAL",
			"wallet_id", b.WalletID,
			"balance", b.Balance.String(),
			"available", b.AvailableBalance.String(),
			"pending", b.PendingBalance.String(),
			"expected_balance", expected.String(),
			"drift", b.Balance.Sub(expected).String(),
		)
		return fmt.Errorf(
			"wallet invariant violation: balance=%s != available=%s + pending=%s",
			b.Balance, b.AvailableBalance, b.PendingBalance,
		)
	}
	return nil
}

// ── Balance snapshots ─────────────────────────────────────────────────────────

type balSnap struct {
	balance   decimal.Decimal
	available decimal.Decimal
	pending   decimal.Decimal
}

func snapBalance(b *WalletBalance) balSnap {
	return balSnap{b.Balance, b.AvailableBalance, b.PendingBalance}
}

// ── Mutation helpers ──────────────────────────────────────────────────────────
// All helpers mutate the WalletBalance in-place AND preserve the invariant.
// They do NOT touch the DB — callers call tx.Save(b) after.

// applyDeposit increases Balance and AvailableBalance by amount.
// PendingBalance is untouched — existing escrow holds are unaffected.
// Returns (balanceBefore, availableBefore) for WalletTransaction records.
func applyDeposit(b *WalletBalance, amount decimal.Decimal) (balanceBefore, availableBefore decimal.Decimal) {
	snap := snapBalance(b)
	b.Balance = b.Balance.Add(amount)
	b.AvailableBalance = b.AvailableBalance.Add(amount)
	b.UpdatedAt = time.Now()
	logBalanceOp("deposit", b.WalletID.String(), snap, b, amount)
	return snap.balance, snap.available
}

// applyWithdrawal decreases Balance and AvailableBalance by amount.
// PendingBalance is untouched.
// Caller must verify AvailableBalance >= amount before calling.
// Returns (balanceBefore, availableBefore) for WalletTransaction records.
func applyWithdrawal(b *WalletBalance, amount decimal.Decimal) (balanceBefore, availableBefore decimal.Decimal) {
	snap := snapBalance(b)
	b.Balance = b.Balance.Sub(amount)
	b.AvailableBalance = b.AvailableBalance.Sub(amount)
	b.UpdatedAt = time.Now()
	logBalanceOp("withdrawal", b.WalletID.String(), snap, b, amount)
	return snap.balance, snap.available
}

// applyEscrowHold moves amount from AvailableBalance to PendingBalance.
// Balance (total) is unchanged.
// Returns (availableBefore) for WalletTransaction BalanceBefore.
func applyEscrowHold(b *WalletBalance, amount decimal.Decimal) (availableBefore decimal.Decimal) {
	snap := snapBalance(b)
	b.AvailableBalance = b.AvailableBalance.Sub(amount)
	b.PendingBalance = b.PendingBalance.Add(amount)
	b.UpdatedAt = time.Now()
	logBalanceOp("escrow_hold", b.WalletID.String(), snap, b, amount)
	return snap.available
}

// applyEscrowRelease (buyer side) moves amount from PendingBalance and deducts
// it from Balance. AvailableBalance is unchanged.
func applyEscrowRelease(b *WalletBalance, amount decimal.Decimal) {
	snap := snapBalance(b)
	b.PendingBalance = b.PendingBalance.Sub(amount)
	b.Balance = b.Balance.Sub(amount)
	b.UpdatedAt = time.Now()
	logBalanceOp("escrow_release_buyer", b.WalletID.String(), snap, b, amount)
}

// applyEscrowCancel moves amount from PendingBalance back to AvailableBalance.
// Balance (total) is unchanged.
// Returns availableBefore for WalletTransaction BalanceBefore.
func applyEscrowCancel(b *WalletBalance, amount decimal.Decimal) (availableBefore decimal.Decimal) {
	snap := snapBalance(b)
	b.PendingBalance = b.PendingBalance.Sub(amount)
	b.AvailableBalance = b.AvailableBalance.Add(amount)
	b.UpdatedAt = time.Now()
	logBalanceOp("escrow_cancel", b.WalletID.String(), snap, b, amount)
	return snap.available
}

// applyEscrowReleaseApproval records 1st/2nd admin approvals for escrow release.
// Returns readyToRelease=true only after a distinct second admin approves.
func applyEscrowReleaseApproval(e *Escrow, adminID uuid.UUID, now time.Time) (readyToRelease bool, err error) {
	if e.Approval1By == nil {
		e.Approval1By = &adminID
		e.Approval1At = &now
		return false, nil
	}
	if e.Approval2By == nil {
		if *e.Approval1By == adminID {
			return false, fmt.Errorf("second_approval_must_be_distinct_admin")
		}
		e.Approval2By = &adminID
		e.Approval2At = &now
	}
	return true, nil
}

// ── Logging ───────────────────────────────────────────────────────────────────

func logBalanceOp(op, walletID string, before balSnap, after *WalletBalance, amount decimal.Decimal) {
	slog.Info("wallet_balance_mutation",
		"op", op,
		"wallet_id", walletID,
		"amount", amount.String(),
		"balance_before", before.balance.String(),
		"available_before", before.available.String(),
		"pending_before", before.pending.String(),
		"balance_after", after.Balance.String(),
		"available_after", after.AvailableBalance.String(),
		"pending_after", after.PendingBalance.String(),
	)
}
