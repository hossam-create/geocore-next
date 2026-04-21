package protection

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Smart Pricing Constants ────────────────────────────────────────────────────

const (
	BasePricePercent     = 1.5  // base % for standard orders
	HighRiskPricePercent = 3.0  // % for high-risk orders
	UrgentPricePercent   = 4.0  // % for urgent/time-sensitive orders
	MinPriceCents        = 50   // minimum 0.50 currency units
	RiskScoreHigh        = 50.0 // risk score threshold for higher pricing
	RiskScoreVeryHigh    = 70.0 // risk score threshold for max pricing
)

// CalculateProtectionPrice computes the smart price for order protection.
// price = base_price * (1% → 3%) + risk_factor + urgency_factor
func CalculateProtectionPrice(db *gorm.DB, orderID, userID uuid.UUID, bundleType string) (int64, float64, float64, error) {
	// 1. Load order total
	var ord struct {
		Total        float64
		Status       string
		BuyerID      uuid.UUID
		DeliveryType string
		CreatedAt    time.Time
	}
	if err := db.Table("orders").
		Select("total, status, buyer_id, delivery_type, created_at").
		Where("id = ?", orderID).First(&ord).Error; err != nil {
		return 0, 0, 0, fmt.Errorf("order not found")
	}
	if ord.BuyerID != userID {
		return 0, 0, 0, fmt.Errorf("only the buyer can purchase protection")
	}
	if ord.Status != "pending" && ord.Status != "confirmed" {
		return 0, 0, 0, fmt.Errorf("protection can only be purchased for pending/confirmed orders")
	}

	// 2. Get bundle base price
	bundle := getBundle(bundleType)
	if bundle == nil {
		return 0, 0, 0, fmt.Errorf("invalid bundle type: %s", bundleType)
	}

	basePct := bundle.PricePercent

	// 3. Calculate risk factor
	riskFactor := calculateRiskFactor(db, userID)

	// 4. Calculate urgency factor
	urgencyFactor := calculateUrgencyFactor(db, orderID)

	// 5. Apply factors to base price
	effectivePct := basePct + riskFactor + urgencyFactor
	effectivePct = math.Max(basePct, math.Min(effectivePct, UrgentPricePercent))

	// 6. Calculate price in cents
	totalCents := int64(ord.Total * 100)
	priceCents := int64(float64(totalCents) * effectivePct / 100.0)
	if priceCents < MinPriceCents {
		priceCents = MinPriceCents
	}

	// 7. First order free check
	var orderCount int64
	db.Table("orders").Where("buyer_id = ? AND status NOT IN ?", userID, []string{"cancelled"}).Count(&orderCount)
	if orderCount <= 1 {
		priceCents = 0 // first order free
	}

	return priceCents, riskFactor, urgencyFactor, nil
}

// calculateRiskFactor returns the additional % to add based on user risk.
func calculateRiskFactor(db *gorm.DB, userID uuid.UUID) float64 {
	var profile struct {
		RiskScore float64
	}
	db.Table("user_risk_profiles").
		Select("risk_score").
		Where("user_id = ?", userID).
		Scan(&profile)

	switch {
	case profile.RiskScore >= RiskScoreVeryHigh:
		return 1.5 // +1.5% for very high risk
	case profile.RiskScore >= RiskScoreHigh:
		return 0.5 // +0.5% for high risk
	default:
		return 0
	}
}

// calculateUrgencyFactor returns the additional % based on order urgency.
func calculateUrgencyFactor(db *gorm.DB, orderID uuid.UUID) float64 {
	// Check if order is crowdshipping (more urgent = higher factor)
	var ord struct {
		DeliveryType string
	}
	db.Table("orders").Select("delivery_type").Where("id = ?", orderID).Scan(&ord)

	if ord.DeliveryType == "CROWDSHIPPING" {
		// Check if there's a tight deadline on the delivery request
		var dr struct {
			Deadline *time.Time
		}
		db.Table("delivery_requests").
			Select("deadline").
			Where("buyer_id IN (SELECT buyer_id FROM orders WHERE id = ?)", orderID).
			Scan(&dr)

		if dr.Deadline != nil {
			hoursUntilDeadline := time.Until(*dr.Deadline).Hours()
			if hoursUntilDeadline < 24 {
				return 0.5 // +0.5% for urgent delivery
			}
			if hoursUntilDeadline < 72 {
				return 0.25 // +0.25% for somewhat urgent
			}
		}
	}
	return 0
}

// getBundle returns the protection bundle for a given type.
func getBundle(bundleType string) *ProtectionBundle {
	for i := range ProtectionBundles {
		if ProtectionBundles[i].Type == bundleType {
			return &ProtectionBundles[i]
		}
	}
	return nil
}

// PurchaseProtection creates an OrderProtection record and charges the wallet.
func PurchaseProtection(db *gorm.DB, orderID, userID uuid.UUID, bundleType string) (*OrderProtection, error) {
	// 1. Check if order already has protection
	var existing OrderProtection
	if err := db.Where("order_id = ?", orderID).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("order already has protection")
	}

	// 2. Calculate price
	priceCents, riskFactor, urgencyFactor, err := CalculateProtectionPrice(db, orderID, userID, bundleType)
	if err != nil {
		return nil, err
	}

	// 3. Get bundle details
	bundle := getBundle(bundleType)

	// 4. Check first order free
	var orderCount int64
	db.Table("orders").Where("buyer_id = ? AND status NOT IN ?", userID, []string{"cancelled"}).Count(&orderCount)
	firstOrderFree := orderCount <= 1

	// 5. Charge wallet (unless free)
	if !firstOrderFree && priceCents > 0 {
		if err := chargeWalletForProtection(db, userID, priceCents, orderID); err != nil {
			return nil, fmt.Errorf("wallet charge failed: %w", err)
		}
	}

	// 6. Get A/B variant for this user
	variant := GetVariantForUser(db, userID, "protection_default_on")

	// 7. Create protection record
	protection := OrderProtection{
		OrderID:         orderID,
		UserID:          userID,
		PriceCents:      priceCents,
		HasCancellation: bundle.HasCancellation,
		HasDelay:        bundle.HasDelay,
		HasFull:         bundle.HasFull,
		CoveragePercent: bundle.CoveragePercent,
		RiskFactor:      riskFactor,
		UrgencyFactor:   urgencyFactor,
		IsUsed:          false,
		FirstOrderFree:  firstOrderFree,
		ABVariant:       variant,
	}
	if err := db.Create(&protection).Error; err != nil {
		return nil, fmt.Errorf("protection creation failed: %w", err)
	}

	// 8. Track A/B event
	TrackABEvent(db, userID, "protection_default_on", variant, "protection_added", &orderID, nil)

	return &protection, nil
}

// chargeWalletForProtection debits the user's wallet for protection.
func chargeWalletForProtection(db *gorm.DB, userID uuid.UUID, priceCents int64, orderID uuid.UUID) error {
	amount := float64(priceCents) / 100.0

	var walletID string
	db.Table("wallets").Select("id").Where("user_id = ?", userID).Scan(&walletID)
	if walletID == "" {
		return fmt.Errorf("wallet not found")
	}

	var balance struct {
		AvailableBalance float64
	}
	db.Table("wallet_balances").
		Where("wallet_id = ? AND currency = ?", walletID, "AED").
		Scan(&balance)

	if balance.AvailableBalance < amount {
		return fmt.Errorf("insufficient balance")
	}

	now := time.Now()
	refID := fmt.Sprintf("protection:%s", orderID)

	db.Table("wallet_balances").
		Where("wallet_id = ? AND currency = ?", walletID, "AED").
		Updates(map[string]interface{}{
			"available_balance": gorm.Expr("available_balance - ?", amount),
			"balance":          gorm.Expr("balance - ?", amount),
		})

	db.Table("wallet_transactions").Create(map[string]interface{}{
		"wallet_id":      walletID,
		"type":           "payment",
		"amount":         -amount,
		"currency":       "AED",
		"status":         "completed",
		"reference_id":   refID,
		"reference_type": "order_protection",
		"description":    fmt.Sprintf("Order protection for order %s", orderID),
		"completed_at":   now,
	})

	return nil
}
