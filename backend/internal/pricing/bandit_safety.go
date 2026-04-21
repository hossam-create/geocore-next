package pricing

import (
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Bandit Safety Layer ──────────────────────────────────────────────────────────
//
// 1. Price boundaries: min 1%, max 4%
// 2. Cooldown: same user → same price within session
// 3. Kill switch: if conversion drops > threshold → fallback to static
// 4. Conversion guard: monitors attach rate and auto-activates kill switch

// CheckConversionDrop monitors the overall conversion rate and activates
// the kill switch if it drops below the threshold.
func CheckConversionDrop(db *gorm.DB) (bool, float64) {
	config := loadBanditConfig(db)

	// Compare recent conversion rate (last 1 hour) vs baseline (last 24 hours)
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)

	// Recent rate
	var recentTotal int64
	var recentConversions int64
	db.Model(&BanditEvent{}).Where("created_at > ?", oneHourAgo).Count(&recentTotal)
	db.Model(&BanditEvent{}).Where("created_at > ? AND did_buy = ?", oneHourAgo, true).Count(&recentConversions)

	var recentRate float64
	if recentTotal > 0 {
		recentRate = float64(recentConversions) / float64(recentTotal)
	}

	// Baseline rate
	var baselineTotal int64
	var baselineConversions int64
	db.Model(&BanditEvent{}).Where("created_at > ? AND created_at <= ?", twentyFourHoursAgo, oneHourAgo).Count(&baselineTotal)
	db.Model(&BanditEvent{}).Where("created_at > ? AND created_at <= ? AND did_buy = ?", twentyFourHoursAgo, oneHourAgo, true).Count(&baselineConversions)

	var baselineRate float64
	if baselineTotal > 0 {
		baselineRate = float64(baselineConversions) / float64(baselineTotal)
	}

	// Need minimum data points to make a decision
	if recentTotal < 20 || baselineTotal < 50 {
		return false, recentRate
	}

	// Check for conversion drop
	drop := baselineRate - recentRate
	if drop > config.ConversionDropThreshold {
		// Activate kill switch
		ActivateBanditKillSwitch(db, "conversion_drop")
		return true, recentRate
	}

	return false, recentRate
}

// ActivateBanditKillSwitch turns on the kill switch, falling back to static pricing.
func ActivateBanditKillSwitch(db *gorm.DB, reason string) {
	db.Model(&BanditConfig{}).Where("is_active = ?", true).
		Updates(map[string]interface{}{
			"kill_switch_active": true,
			"updated_at":        time.Now(),
		})

	// Log the kill switch activation
	db.Table("bandit_events").Create(map[string]interface{}{
		"id":         uuid.New(),
		"segment":    "system",
		"algorithm":  "kill_switch",
		"price_cents": 0,
		"reward":     0,
		"created_at": time.Now(),
	})
}

// DeactivateBanditKillSwitch turns off the kill switch, resuming bandit pricing.
func DeactivateBanditKillSwitch(db *gorm.DB) {
	db.Model(&BanditConfig{}).Where("is_active = ?", true).
		Updates(map[string]interface{}{
			"kill_switch_active": false,
			"updated_at":        time.Now(),
		})
}

// ClampPrice ensures the price stays within safe boundaries.
func ClampPrice(priceCents int64, orderPriceCents int64, minPct, maxPct float64) int64 {
	minCents := int64(float64(orderPriceCents) * minPct / 100.0)
	maxCents := int64(float64(orderPriceCents) * maxPct / 100.0)

	if minCents < 50 {
		minCents = 50
	}
	if maxCents < 50 {
		maxCents = 50
	}

	return int64(math.Max(float64(minCents), math.Min(float64(maxCents), float64(priceCents))))
}

// GetConversionTrend returns the current conversion trend.
func GetConversionTrend(db *gorm.DB) string {
	now := time.Now()

	// Last 3 hours
	var recentTotal int64
	var recentConversions int64
	db.Model(&BanditEvent{}).Where("created_at > ?", now.Add(-3*time.Hour)).Count(&recentTotal)
	db.Model(&BanditEvent{}).Where("created_at > ? AND did_buy = ?", now.Add(-3*time.Hour), true).Count(&recentConversions)

	// 3-6 hours ago
	var prevTotal int64
	var prevConversions int64
	db.Model(&BanditEvent{}).Where("created_at > ? AND created_at <= ?",
		now.Add(-6*time.Hour), now.Add(-3*time.Hour)).Count(&prevTotal)
	db.Model(&BanditEvent{}).Where("created_at > ? AND created_at <= ? AND did_buy = ?",
		now.Add(-6*time.Hour), now.Add(-3*time.Hour), true).Count(&prevConversions)

	if recentTotal < 10 || prevTotal < 10 {
		return "stable" // not enough data
	}

	recentRate := float64(recentConversions) / float64(recentTotal)
	prevRate := float64(prevConversions) / float64(prevTotal)

	diff := recentRate - prevRate
	if diff > 0.05 {
		return "improving"
	} else if diff < -0.05 {
		return "dropping"
	}
	return "stable"
}

// GetBanditDashboard returns the full bandit dashboard for admin view.
func GetBanditDashboard(db *gorm.DB) *BanditDashboard {
	config := loadBanditConfig(db)

	// Load all arms
	var arms []BanditArm
	db.Find(&arms)

	// Build stats per arm
	armStats := make([]BanditArmStats, 0, len(arms))
	var totalImpressions, totalConversions int64
	var totalReward float64
	var bestArm *BanditArmStats
	bestAvgReward := math.Inf(-1)

	for _, arm := range arms {
		stats := BanditArmStats{
			Segment:       arm.Segment,
			PricePercent:  arm.PricePercent,
			Impressions:   arm.Impressions,
			Conversions:   arm.Conversions,
			TotalReward:   arm.TotalReward,
			Alpha:         arm.Alpha,
			Beta:          arm.Beta,
		}
		if arm.Impressions > 0 {
			stats.ConversionRate = float64(arm.Conversions) / float64(arm.Impressions)
			stats.AvgReward = arm.TotalReward / float64(arm.Impressions)
		}
		stats.SampleValue = SampleArm(&arm)

		totalImpressions += arm.Impressions
		totalConversions += arm.Conversions
		totalReward += arm.TotalReward

		if stats.AvgReward > bestAvgReward && arm.Impressions >= int64(config.MinImpressionsBeforeExploit) {
			bestAvgReward = stats.AvgReward
			bestArm = &stats
		}

		armStats = append(armStats, stats)
	}

	overallAttachRate := 0.0
	if totalImpressions > 0 {
		overallAttachRate = float64(totalConversions) / float64(totalImpressions)
	}

	return &BanditDashboard{
		Config:            config,
		Arms:              armStats,
		TotalImpressions:  totalImpressions,
		TotalConversions:  totalConversions,
		OverallAttachRate: overallAttachRate,
		OverallRevenue:    totalReward,
		KillSwitchActive:  config.KillSwitchActive,
		BestArm:           bestArm,
		ConversionTrend:   GetConversionTrend(db),
	}
}

// ResetSegmentArms resets all arm statistics for a segment (fresh start).
func ResetSegmentArms(db *gorm.DB, segment string) {
	db.Model(&BanditArm{}).Where("segment = ?", segment).
		Updates(map[string]interface{}{
			"impressions":  0,
			"conversions":  0,
			"total_reward": 0,
			"alpha":        1,
			"beta":         1,
			"updated_at":   time.Now(),
		})
}
