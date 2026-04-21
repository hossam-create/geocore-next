package protection

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── A/B Test Variants ────────────────────────────────────────────────────────────

const (
	ExperimentProtectionDefault = "protection_default_on"

	VariantControl     = "control"      // protection OFF by default
	VariantOptOut      = "opt_out"      // protection ON by default (opt-out)
	VariantSocialProof = "social_proof" // show "98% of users choose protection"
)

var Experiments = []struct {
	Name     string
	Variants []string
}{
	{ExperimentProtectionDefault, []string{VariantControl, VariantOptOut, VariantSocialProof}},
}

// GetVariantForUser returns the A/B variant assigned to a user for an experiment.
// Uses deterministic hash for consistent assignment, falls back to DB record.
func GetVariantForUser(db *gorm.DB, userID uuid.UUID, experiment string) string {
	// 1. Check if user already has a variant assignment
	var assignment ABVariantAssignment
	if err := db.Where("user_id = ? AND experiment = ?", userID, experiment).First(&assignment).Error; err == nil {
		return assignment.Variant
	}

	// 2. Deterministic assignment based on user ID hash
	variant := deterministicVariant(userID, experiment)

	// 3. Persist the assignment
	assignment = ABVariantAssignment{
		UserID:     userID,
		Experiment: experiment,
		Variant:    variant,
	}
	db.Create(&assignment)

	return variant
}

// deterministicVariant uses a hash of userID + experiment to assign variant.
// Ensures the same user always gets the same variant.
func deterministicVariant(userID uuid.UUID, experiment string) string {
	h := fnv.New32a()
	h.Write([]byte(userID.String() + experiment))
	hash := h.Sum32()

	// Get variants for this experiment
	variants := []string{VariantControl, VariantOptOut, VariantSocialProof}
	for _, exp := range Experiments {
		if exp.Name == experiment {
			variants = exp.Variants
			break
		}
	}

	idx := hash % uint32(len(variants))
	return variants[idx]
}

// TrackABEvent records an A/B test event for metrics tracking.
func TrackABEvent(db *gorm.DB, userID uuid.UUID, experiment, variant, eventType string, orderID *uuid.UUID, metadata map[string]interface{}) {
	metaJSON := "{}"
	if metadata != nil {
		if b, err := json.Marshal(metadata); err == nil {
			metaJSON = string(b)
		}
	}

	event := ABEvent{
		UserID:     userID,
		Experiment: experiment,
		Variant:    variant,
		EventType:  eventType,
		OrderID:    orderID,
		Metadata:   metaJSON,
	}
	db.Create(&event)
}

// ── A/B Test Results ────────────────────────────────────────────────────────────

// GetABTestResults computes metrics for each variant of an experiment.
func GetABTestResults(db *gorm.DB, experiment string) (*ABTestResults, error) {
	results := &ABTestResults{
		Experiment: experiment,
	}

	// Get all variants for this experiment
	var variants []string
	for _, exp := range Experiments {
		if exp.Name == experiment {
			variants = exp.Variants
			break
		}
	}
	if len(variants) == 0 {
		return nil, fmt.Errorf("experiment not found: %s", experiment)
	}

	for _, v := range variants {
		metrics := ABVariantMetrics{Variant: v}

		// Users in this variant
		db.Model(&ABVariantAssignment{}).
			Where("experiment = ? AND variant = ?", experiment, v).
			Count(&metrics.Users)

		// Attach rate: % of users who added protection
		var attachCount int64
		db.Model(&ABEvent{}).
			Where("experiment = ? AND variant = ? AND event_type = ?", experiment, v, "protection_added").
			Count(&attachCount)
		if metrics.Users > 0 {
			metrics.AttachRate = float64(attachCount) / float64(metrics.Users)
		}

		// Conversion rate: % of users who placed an order
		var orderCount int64
		db.Model(&ABEvent{}).
			Where("experiment = ? AND variant = ? AND event_type = ?", experiment, v, "order_placed").
			Count(&orderCount)
		if metrics.Users > 0 {
			metrics.ConversionRate = float64(orderCount) / float64(metrics.Users)
		}

		// Cancellation rate
		var cancelCount int64
		db.Model(&ABEvent{}).
			Where("experiment = ? AND variant = ? AND event_type = ?", experiment, v, "cancelled").
			Count(&cancelCount)
		if orderCount > 0 {
			metrics.CancellationRate = float64(cancelCount) / float64(orderCount)
		}

		// Average revenue per order
		var avgRevenue struct {
			Avg float64
		}
		db.Table("order_protections").
			Where("ab_variant = ? AND price_cents > 0", v).
			Select("COALESCE(AVG(price_cents), 0) as avg").
			Scan(&avgRevenue)
		metrics.AvgRevenuePerOrder = avgRevenue.Avg

		results.Variants = append(results.Variants, metrics)
	}

	// Determine winner (highest conversion rate)
	bestRate := 0.0
	for _, v := range results.Variants {
		if v.ConversionRate > bestRate {
			bestRate = v.ConversionRate
			results.Winner = v.Variant
		}
	}

	return results, nil
}

// ── Daily Metrics Aggregation ────────────────────────────────────────────────────

// AggregateDailyMetrics computes and stores daily protection metrics.
func AggregateDailyMetrics(db *gorm.DB) error {
	today := time.Now().Truncate(24 * time.Hour)

	var metrics ProtectionDailyMetrics
	if err := db.Where("date = ?", today).First(&metrics).Error; err != nil {
		metrics = ProtectionDailyMetrics{Date: today}
	}

	// Total orders today
	db.Table("orders").Where("DATE(created_at) = ?", today.Format("2006-01-02")).Count(&metrics.TotalOrders)

	// Protection attached today
	db.Table("order_protections").Where("DATE(created_at) = ?", today.Format("2006-01-02")).Count(&metrics.ProtectionAttached)

	// Attach rate
	if metrics.TotalOrders > 0 {
		metrics.AttachRate = float64(metrics.ProtectionAttached) / float64(metrics.TotalOrders)
	}

	// Revenue
	var revenue struct{ Total int64 }
	db.Table("order_protections").
		Where("DATE(created_at) = ?", today.Format("2006-01-02")).
		Select("COALESCE(SUM(price_cents), 0) as total").
		Scan(&revenue)
	metrics.RevenueCents = revenue.Total

	// Claims filed today
	db.Table("guarantee_claims").Where("DATE(created_at) = ?", today.Format("2006-01-02")).Count(&metrics.ClaimsFiled)

	// Claims approved today
	db.Table("guarantee_claims").
		Where("DATE(resolved_at) = ? AND status IN ?", today.Format("2006-01-02"),
			[]string{"auto_approved", "approved"}).Count(&metrics.ClaimsApproved)

	// Payouts
	var payouts struct{ Total int64 }
	db.Table("guarantee_claims").
		Where("DATE(resolved_at) = ? AND status IN ?", today.Format("2006-01-02"),
			[]string{"auto_approved", "approved"}).
		Select("COALESCE(SUM(refund_cents + compensation_cents), 0) as total").
		Scan(&payouts)
	metrics.PayoutsCents = payouts.Total

	// Net revenue
	metrics.NetRevenueCents = metrics.RevenueCents - metrics.PayoutsCents

	// Average risk factor
	var avgRisk struct{ Avg float64 }
	db.Table("order_protections").
		Where("DATE(created_at) = ?", today.Format("2006-01-02")).
		Select("COALESCE(AVG(risk_factor), 0) as avg").
		Scan(&avgRisk)
	metrics.AvgRiskFactor = avgRisk.Avg

	// Upsert
	if metrics.ID == uuid.Nil {
		db.Create(&metrics)
	} else {
		db.Save(&metrics)
	}

	return nil
}
