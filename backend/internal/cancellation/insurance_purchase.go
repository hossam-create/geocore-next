package cancellation

import (
	"fmt"
	"time"

	"github.com/geocore-next/backend/internal/config"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PurchaseInsurance buys cancellation insurance for an order.
// Charges the user's wallet and attaches the insurance record.
func PurchaseInsurance(db *gorm.DB, orderID, userID uuid.UUID, coverage CoverageType) (*InsurancePurchaseResult, error) {
	// 1. Check feature flag
	if !IsInsuranceEnabled() {
		return nil, fmt.Errorf("cancellation insurance is not available")
	}

	// 2. Check if order already has insurance
	var existing OrderInsurance
	if err := db.Where("order_id = ? AND is_active = ?", orderID, true).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("order already has insurance")
	}

	// 3. Load order total
	var ord struct {
		Total   float64
		Status  string
		BuyerID uuid.UUID
	}
	if err := db.Table("orders").
		Select("total, status, buyer_id").
		Where("id = ?", orderID).First(&ord).Error; err != nil {
		return nil, fmt.Errorf("order not found")
	}
	if ord.BuyerID != userID {
		return nil, fmt.Errorf("only the buyer can purchase insurance")
	}
	if ord.Status != "pending" && ord.Status != "confirmed" {
		return nil, fmt.Errorf("insurance can only be purchased for pending/confirmed orders")
	}

	// 4. Anti-abuse: check if user can buy insurance
	if !canBuyInsurance(db, userID) {
		return nil, fmt.Errorf("insurance purchase temporarily unavailable for this account")
	}

	// 5. Calculate insurance price
	tier := getTier(coverage)
	if tier == nil {
		return nil, fmt.Errorf("invalid coverage type: %s", coverage)
	}

	totalCents := int64(ord.Total * 100)
	priceCents := int64(float64(totalCents) * tier.PricePercent / 100.0)
	if priceCents < 50 { // minimum 0.50
		priceCents = 50
	}

	// 6. Check if this is the user's first order → free insurance
	firstOrderFree := false
	var orderCount int64
	db.Table("orders").Where("buyer_id = ? AND status NOT IN ?", userID, []string{"cancelled"}).Count(&orderCount)
	if orderCount <= 1 {
		firstOrderFree = true
		priceCents = 0
	}

	// 7. Charge wallet (unless free)
	if !firstOrderFree && priceCents > 0 {
		if err := chargeWalletForInsurance(db, userID, priceCents, orderID); err != nil {
			return nil, fmt.Errorf("wallet charge failed: %w", err)
		}
	}

	// 8. Create insurance record
	insurance := OrderInsurance{
		OrderID:          orderID,
		UserID:           userID,
		PriceCents:       priceCents,
		CoverageType:     coverage,
		MaxFeeCoveredPct: tier.MaxFeeCoveredPct,
		IsActive:         true,
		IsUsed:           false,
		FirstOrderFree:   firstOrderFree,
	}
	if err := db.Create(&insurance).Error; err != nil {
		return nil, fmt.Errorf("insurance creation failed: %w", err)
	}

	// 9. Update usage tracking
	trackInsurancePurchase(db, userID)

	return &InsurancePurchaseResult{
		InsuranceID:    insurance.ID,
		OrderID:        orderID,
		PriceCents:     priceCents,
		CoverageType:   coverage,
		FirstOrderFree: firstOrderFree,
	}, nil
}

// chargeWalletForInsurance debits the user's wallet for insurance.
func chargeWalletForInsurance(db *gorm.DB, userID uuid.UUID, priceCents int64, orderID uuid.UUID) error {
	amount := float64(priceCents) / 100.0

	// Find user wallet
	var walletID string
	db.Table("wallets").Select("id").Where("user_id = ?", userID).Scan(&walletID)
	if walletID == "" {
		return fmt.Errorf("wallet not found")
	}

	// Check balance
	var balance struct {
		AvailableBalance float64
	}
	db.Table("wallet_balances").
		Where("wallet_id = ? AND currency = ?", walletID, "AED").
		Scan(&balance)

	if balance.AvailableBalance < amount {
		return fmt.Errorf("insufficient balance")
	}

	// Debit wallet
	now := time.Now()
	refID := fmt.Sprintf("insurance:%s", orderID)
	refType := "cancellation_insurance"

	db.Table("wallet_balances").
		Where("wallet_id = ? AND currency = ?", walletID, "AED").
		Updates(map[string]interface{}{
			"available_balance": gorm.Expr("available_balance - ?", amount),
			"balance":           gorm.Expr("balance - ?", amount),
		})

	db.Table("wallet_transactions").Create(map[string]interface{}{
		"wallet_id":      walletID,
		"type":           "payment",
		"amount":         -amount,
		"currency":       "AED",
		"status":         "completed",
		"reference_id":   refID,
		"reference_type": refType,
		"description":    fmt.Sprintf("Cancellation insurance for order %s", orderID),
		"completed_at":   now,
	})

	return nil
}

// getTier returns the pricing tier for a coverage type.
func getTier(coverage CoverageType) *InsurancePriceTier {
	for i := range InsuranceTiers {
		if InsuranceTiers[i].CoverageType == coverage {
			return &InsuranceTiers[i]
		}
	}
	return nil
}

// trackInsurancePurchase increments the monthly insurance purchase counter.
func trackInsurancePurchase(db *gorm.DB, userID uuid.UUID) {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var usage UserInsuranceUsage
	if err := db.Where("user_id = ? AND month = ?", userID, monthStart).First(&usage).Error; err != nil {
		usage = UserInsuranceUsage{
			UserID:             userID,
			Month:              monthStart,
			InsurancePurchased: 1,
		}
		db.Create(&usage)
		return
	}
	usage.InsurancePurchased++
	db.Save(&usage)
}

// IsInsuranceEnabled checks the feature flag.
func IsInsuranceEnabled() bool {
	return config.GetFlags().EnableCancellationInsurance
}
