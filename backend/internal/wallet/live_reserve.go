package wallet

import (
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ════════════════════════════════════════════════════════════════════════════
// Live Auction Reservation Layer (Sprint 9.5)
//
// Ledger logic:
//   Reserve:   available-- , pending++   (Balance unchanged)
//   Release:   available++ , pending--   (Balance unchanged)
//   Convert:   pending--   , escrow++    (via HoldFunds, Balance unchanged)
//
// These match the existing WalletBalance invariant:
//   Balance == AvailableBalance + PendingBalance
// ════════════════════════════════════════════════════════════════════════════

// HasSufficientBalance checks if user has enough available balance.
// Does NOT lock — use for pre-flight checks only.
func HasSufficientBalance(db *gorm.DB, userID uuid.UUID, amountCents int64) bool {
	amountDec := decimal.NewFromInt(amountCents).Div(decimal.NewFromInt(100))
	var avail decimal.Decimal
	err := db.Table("wallet_balances wb").
		Joins("JOIN wallets w ON w.id = wb.wallet_id").
		Where("w.user_id = ? AND wb.currency = ?", userID, "USD").
		Select("wb.available_balance").
		Scan(&avail).Error
	if err != nil {
		return false
	}
	return avail.GreaterThanOrEqual(amountDec)
}

// ReserveFunds moves amount from available → pending within a transaction.
// This is a temporary hold during live bidding (not a full escrow).
// Caller MUST pass a *gorm.DB that is already inside a transaction.
func ReserveFunds(tx *gorm.DB, userID uuid.UUID, amountCents int64) error {
	amountDec := decimal.NewFromInt(amountCents).Div(decimal.NewFromInt(100))

	// Lock wallet → balance in deterministic order
	var w Wallet
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).First(&w).Error; err != nil {
		return fmt.Errorf("wallet not found for user %s", userID)
	}

	var bal WalletBalance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("wallet_id = ? AND currency = ?", w.ID, USD).First(&bal).Error; err != nil {
		return fmt.Errorf("USD balance not found for user %s", userID)
	}

	if bal.AvailableBalance.LessThan(amountDec) {
		return fmt.Errorf("insufficient balance: have %s, need %s", bal.AvailableBalance.String(), amountDec.String())
	}

	// Move available → pending (Balance unchanged → invariant preserved)
	applyEscrowHold(&bal, amountDec)
	if err := checkInvariant(bal); err != nil {
		return err
	}

	slog.Info("live-reserve: funds reserved",
		"user_id", userID, "amount_cents", amountCents,
		"available_after", bal.AvailableBalance.String(), "pending_after", bal.PendingBalance.String())

	return tx.Save(&bal).Error
}

// ReleaseReservedFunds moves amount from pending → available (undo reserve).
// Called when a bidder is outbid.
func ReleaseReservedFunds(tx *gorm.DB, userID uuid.UUID, amountCents int64) error {
	amountDec := decimal.NewFromInt(amountCents).Div(decimal.NewFromInt(100))

	var w Wallet
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).First(&w).Error; err != nil {
		return fmt.Errorf("wallet not found for user %s", userID)
	}

	var bal WalletBalance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("wallet_id = ? AND currency = ?", w.ID, USD).First(&bal).Error; err != nil {
		return fmt.Errorf("USD balance not found for user %s", userID)
	}

	// Move pending → available (Balance unchanged → invariant preserved)
	applyEscrowCancel(&bal, amountDec)
	if err := checkInvariant(bal); err != nil {
		return err
	}

	slog.Info("live-reserve: funds released",
		"user_id", userID, "amount_cents", amountCents,
		"available_after", bal.AvailableBalance.String(), "pending_after", bal.PendingBalance.String())

	return tx.Save(&bal).Error
}

// ConvertReserveToHold converts a pending reserve into a real escrow hold.
// Called at auction settlement when winner is confirmed.
// Creates an Escrow record and a WalletTransaction.
func ConvertReserveToHold(db *gorm.DB, buyerID, sellerID uuid.UUID, amountCents int64, refType, refID string) (*Escrow, error) {
	amountDec := decimal.NewFromInt(amountCents).Div(decimal.NewFromInt(100))
	fee := amountDec.Mul(decimal.NewFromFloat(0.025)) // 2.5% fee

	escrow := Escrow{
		BuyerID:     buyerID,
		SellerID:    sellerID,
		Amount:      amountDec,
		Currency:    USD,
		Fee:         fee,
		Status:      StatusPending,
		ReferenceID: refID,
		Type:        refType,
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		// The funds are already in pending from ReserveFunds.
		// We just create the escrow record — no balance mutation needed.
		// The pending amount stays as-is until admin releases escrow.
		if err := tx.Create(&escrow).Error; err != nil {
			return err
		}

		// Record transaction for audit trail
		var w Wallet
		if err := tx.Where("user_id = ?", buyerID).First(&w).Error; err != nil {
			return err
		}
		rt := "escrow_live_auction"
		walletTx := WalletTransaction{
			WalletID:      w.ID,
			Type:          TransactionEscrow,
			Currency:      USD,
			Amount:        amountDec.Neg(),
			Status:        StatusPending,
			ReferenceID:   &refID,
			ReferenceType: &rt,
			Description:   fmt.Sprintf("Live auction escrow for %s #%s", refType, refID),
		}
		return tx.Create(&walletTx).Error
	})

	if err != nil {
		return nil, err
	}
	return &escrow, nil
}
