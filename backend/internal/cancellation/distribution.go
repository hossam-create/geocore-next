package cancellation

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ApplyCancellationFee records the fee in the ledger, credits the traveler,
// and debits the buyer. Called AFTER the order status is updated to cancelled.
func ApplyCancellationFee(db *gorm.DB, orderID, buyerID, travelerID uuid.UUID, result *CancellationFeeResult, reason string) error {
	if result.FeeCents <= 0 {
		// Record zero-fee entry for audit
		db.Create(&CancellationLedger{
			OrderID:             orderID,
			UserID:              buyerID,
			FeeCents:            0,
			TravelerCompensation: 0,
			PlatformFee:          0,
			FeePercent:          0,
			AbuseMultiplier:     result.AbuseMultiplier,
			TokenUsed:           result.TokenUsed,
			SecondsSinceAccept:  result.SecondsSinceAccept,
			Tier:                result.Tier,
			Reason:              reason,
		})
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// 1. Record in ledger
		ledger := CancellationLedger{
			OrderID:             orderID,
			UserID:              buyerID,
			FeeCents:            result.FeeCents,
			TravelerCompensation: result.TravelerCompensation,
			PlatformFee:          result.PlatformFee,
			FeePercent:          result.FeePercent,
			AbuseMultiplier:     result.AbuseMultiplier,
			TokenUsed:           result.TokenUsed,
			SecondsSinceAccept:  result.SecondsSinceAccept,
			Tier:                result.Tier,
			Reason:              reason,
		}
		if err := tx.Create(&ledger).Error; err != nil {
			return fmt.Errorf("ledger write failed: %w", err)
		}

		// 2. Credit traveler compensation to traveler wallet
		if result.TravelerCompensation > 0 {
			if err := tx.Table("wallet_transactions").
				Create(map[string]interface{}{
					"wallet_id": getWalletID(tx, travelerID),
					"type":      "credit",
					"amount":    float64(result.TravelerCompensation) / 100.0,
					"currency":  "AED",
					"reference": fmt.Sprintf("cancel_comp:%s", orderID),
					"note":      "Cancellation compensation from buyer",
				}).Error; err != nil {
				return fmt.Errorf("traveler credit failed: %w", err)
			}
		}

		// 3. Credit platform fee to platform wallet
		if result.PlatformFee > 0 {
			if err := tx.Table("wallet_transactions").
				Create(map[string]interface{}{
					"wallet_id": getPlatformWalletID(tx),
					"type":      "credit",
					"amount":    float64(result.PlatformFee) / 100.0,
					"currency":  "AED",
					"reference": fmt.Sprintf("cancel_platform_fee:%s", orderID),
					"note":      "Cancellation platform fee",
				}).Error; err != nil {
				return fmt.Errorf("platform fee credit failed: %w", err)
			}
		}

		// 4. Update user cancellation stats
		updateStatsAfterCancel(tx, buyerID)

		return nil
	})
}

// getWalletID resolves a user's wallet ID.
func getWalletID(tx *gorm.DB, userID uuid.UUID) string {
	var wid string
	tx.Table("wallets").Select("id").Where("user_id = ?", userID).Scan(&wid)
	if wid == "" {
		return userID.String() // fallback
	}
	return wid
}

// getPlatformWalletID resolves the platform wallet.
func getPlatformWalletID(tx *gorm.DB) string {
	var wid string
	tx.Table("wallets").Select("id").Where("user_id IS NULL AND type = 'platform'").Scan(&wid)
	if wid == "" {
		return "platform"
	}
	return wid
}
