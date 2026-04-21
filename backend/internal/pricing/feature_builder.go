package pricing

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Feature Builder ──────────────────────────────────────────────────────────────
// Extracts features from DB + Redis to build a PricingContext.

// BuildPricingContext assembles all features needed for a pricing decision.
func BuildPricingContext(db *gorm.DB, userID, orderID uuid.UUID) (*PricingContext, error) {
	ctx := &PricingContext{
		UserID:  userID,
		OrderID: orderID,
	}

	// ── Order Features ──────────────────────────────────────────────────────
	var ord struct {
		Total        float64
		DeliveryType string
		BuyerID      uuid.UUID
		SellerID     uuid.UUID
		CreatedAt    time.Time
	}
	if err := db.Table("orders").
		Select("total, delivery_type, buyer_id, seller_id, created_at").
		Where("id = ?", orderID).First(&ord).Error; err != nil {
		return nil, err
	}

	ctx.OrderPriceCents = int64(ord.Total * 100)
	ctx.DeliveryType = ord.DeliveryType

	// ── User Features ────────────────────────────────────────────────────────
	var profile struct {
		RiskScore     float64
		TotalOrders   int
		AvgOrderValue float64
	}
	db.Table("user_risk_profiles").
		Select("risk_score, total_orders, avg_order_value").
		Where("user_id = ?", userID).Scan(&profile)

	ctx.TrustScore = 100 - profile.RiskScore // invert: low risk = high trust
	ctx.AvgOrderValue = profile.AvgOrderValue * 100

	// Cancellation rate from cancellation stats
	var cancelStats struct {
		CancelRate float64
	}
	db.Table("user_cancellation_stats").
		Select("cancel_rate").
		Where("user_id = ?", userID).Scan(&cancelStats)
	ctx.CancellationRate = cancelStats.CancelRate

	// Account age
	var userCreatedAt time.Time
	db.Table("users").Select("created_at").Where("id = ?", userID).Scan(&userCreatedAt)
	ctx.AccountAgeDays = time.Since(userCreatedAt).Hours() / 24.0

	// Abuse flags
	var abuseCount int64
	db.Table("guarantee_claims").
		Where("user_id = ? AND status IN ?", userID, []string{"auto_approved", "approved"}).
		Count(&abuseCount)
	ctx.AbuseFlags = int(abuseCount)

	// Past insurance usage
	var insuranceCount int64
	db.Table("order_protections").
		Where("user_id = ?", userID).Count(&insuranceCount)
	ctx.PastInsuranceUsage = int(insuranceCount)

	// Insurance buy rate
	var totalOrders int64
	db.Table("orders").Where("buyer_id = ? AND status NOT IN ?", userID, []string{"cancelled"}).Count(&totalOrders)
	if totalOrders > 0 {
		ctx.InsuranceBuyRate = float64(insuranceCount) / float64(totalOrders)
	}

	// Last insurance price
	var lastPrice struct {
		PriceCents int64
	}
	db.Table("order_protections").
		Select("price_cents").
		Where("user_id = ? AND price_cents > 0", userID).
		Order("created_at DESC").Limit(1).Scan(&lastPrice)
	ctx.LastInsurancePrice = lastPrice.PriceCents

	// Price sensitivity: derived from buy rate and last price
	// High buy rate + low last price = sensitive
	if ctx.InsuranceBuyRate > 0 {
		ctx.PriceSensitivity = 1.0 - ctx.InsuranceBuyRate
	} else {
		ctx.PriceSensitivity = 0.5 // default moderate
	}

	// ── Order/Traveler Features ──────────────────────────────────────────────
	// Traveler rating
	var travelerRating struct {
		Score float64
	}
	db.Table("geo_scores").
		Select("score").
		Where("user_id = ?", ord.SellerID).Scan(&travelerRating)
	ctx.TravelerRating = travelerRating.Score / 20.0 // normalize 0-100 → 0-5

	// Delivery risk: based on route and past delays
	var delayClaims int64
	db.Table("guarantee_claims").
		Where("traveler_id = ? AND type = ?", ord.SellerID, "delay").
		Count(&delayClaims)
	ctx.DeliveryRiskScore = float64(delayClaims) / 10.0
	if ctx.DeliveryRiskScore > 1.0 {
		ctx.DeliveryRiskScore = 1.0
	}

	// Route risk: based on delivery request
	var dr struct {
		Deadline *time.Time
	}
	db.Table("delivery_requests").
		Select("deadline").
		Where("buyer_id = ?", userID).
		Order("created_at DESC").Limit(1).Scan(&dr)

	if dr.Deadline != nil {
		hoursUntil := time.Until(*dr.Deadline).Hours()
		if hoursUntil < 24 {
			ctx.RouteRisk = 0.8
			ctx.UrgencyScore = 0.9
		} else if hoursUntil < 72 {
			ctx.RouteRisk = 0.4
			ctx.UrgencyScore = 0.5
		} else {
			ctx.RouteRisk = 0.1
			ctx.UrgencyScore = 0.1
		}
	}

	// Category from order items
	var firstItem struct {
		Name string
	}
	db.Table("order_items").
		Select("name").
		Where("order_id = ?", orderID).
		Order("created_at ASC").Limit(1).Scan(&firstItem)
	ctx.Category = categorizeItem(firstItem.Name)

	// ── Context Features ────────────────────────────────────────────────────
	now := time.Now()
	ctx.TimeOfDay = now.Hour()
	ctx.IsRushHour = isRushHour(now)

	// Live demand: placeholder — in production, read from Redis
	ctx.LiveDemand = 0.5 // default moderate

	return ctx, nil
}

// categorizeItem maps item names to risk categories.
func categorizeItem(name string) string {
	// Simple heuristic — in production, use a category lookup
	highValue := []string{"iphone", "macbook", "laptop", "camera", "electronics"}
	for _, h := range highValue {
		if len(name) > 0 && containsIgnoreCase(name, h) {
			return "electronics"
		}
	}
	return "general"
}

func containsIgnoreCase(s, substr string) bool {
	sLower := toLower(s)
	subLower := toLower(substr)
	for i := 0; i <= len(sLower)-len(subLower); i++ {
		if sLower[i:i+len(subLower)] == subLower {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result[i] = c
	}
	return string(result)
}

// isRushHour checks if current time is during peak shopping hours.
func isRushHour(t time.Time) bool {
	hour := t.Hour()
	// Peak: 10-12, 18-22
	return (hour >= 10 && hour <= 12) || (hour >= 18 && hour <= 22)
}
