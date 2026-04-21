package wallet

import (
	"fmt"

	"github.com/geocore-next/backend/pkg/locking"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// HoldFunds creates an escrow hold for a buyer→seller transaction.
// Called from other packages (e.g. reverse auctions, auctions) when a deal is accepted.
// Returns the created Escrow and any error.
func HoldFunds(db *gorm.DB, buyerID, sellerID uuid.UUID, amount float64, currency string, refType string, refID string) (*Escrow, error) {
	cur := Currency(currency)
	if cur == "" {
		cur = USD
	}
	amountDec := decimal.NewFromFloat(amount)

	// Calculate fee (2.5%)
	fee := amountDec.Mul(decimal.NewFromFloat(0.025))

	escrow := Escrow{
		BuyerID:     buyerID,
		SellerID:    sellerID,
		Amount:      amountDec,
		Currency:    cur,
		Fee:         fee,
		Status:      StatusPending,
		ReferenceID: refID,
		Type:        refType,
	}

	err := locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		// Lock wallet row FIRST for consistent lock ordering (wallet → balance).
		// Prevents deadlock with Deposit/Transfer which also lock wallet then balance.
		var buyerWallet Wallet
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", buyerID).First(&buyerWallet).Error; err != nil {
			return fmt.Errorf("buyer wallet not found")
		}

		// Lock balance row — prevents concurrent HoldFunds from both seeing sufficient balance (TOCTOU).
		var balance WalletBalance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("wallet_id = ? AND currency = ?", buyerWallet.ID, cur).
			First(&balance).Error; err != nil {
			return fmt.Errorf("currency %s not available in buyer wallet", cur)
		}
		if balance.AvailableBalance.LessThan(amountDec) {
			return fmt.Errorf("insufficient balance: have %s, need %s", balance.AvailableBalance.String(), amountDec.String())
		}

		if err := tx.Create(&escrow).Error; err != nil {
			return err
		}

		// FIX-3: use applyEscrowHold helper — preserves invariant, returns availableBefore
		// for accurate WalletTransaction BalanceBefore.
		// Old code: (a) no BalanceBefore, (b) BalanceAfter = balance.Balance (total, wrong)
		availBefore := applyEscrowHold(&balance, amountDec)
		if err := checkInvariant(balance); err != nil {
			return err
		}
		if err := tx.Save(&balance).Error; err != nil {
			return err
		}

		// Record transaction
		rt := "escrow"
		walletTx := WalletTransaction{
			WalletID:      buyerWallet.ID,
			Type:          TransactionEscrow,
			Currency:      cur,
			Amount:        amountDec.Neg(),
			BalanceBefore: availBefore,
			BalanceAfter:  balance.AvailableBalance,
			Status:        StatusPending,
			ReferenceID:   &refID,
			ReferenceType: &rt,
			Description:   fmt.Sprintf("Escrow hold for %s #%s | avail: %s→%s", refType, refID, availBefore, balance.AvailableBalance),
		}
		return tx.Create(&walletTx).Error
	})

	if err != nil {
		return nil, err
	}
	return &escrow, nil
}
