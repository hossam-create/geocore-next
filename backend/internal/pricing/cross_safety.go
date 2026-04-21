package pricing

import (
	"time"

	"gorm.io/gorm"
)

// ── Cross-System Safety ──────────────────────────────────────────────────────────
//
// Global guardrails + consistency rules + emergency mode + rollout.
// These protect across ALL subsystems (pricing + ranking + recs).

// ActivateCrossEmergency activates emergency mode across all systems.
func ActivateCrossEmergency(db *gorm.DB) {
	db.Model(&CrossConfig{}).Where("is_active = ?", true).
		Updates(map[string]interface{}{
			"emergency_mode_active": true,
			"updated_at":           time.Now(),
		})
	// Also activate hybrid emergency (cascading safety)
	ActivateEmergencyMode(db)
}

func DeactivateCrossEmergency(db *gorm.DB) {
	db.Model(&CrossConfig{}).Where("is_active = ?", true).
		Updates(map[string]interface{}{
			"emergency_mode_active": false,
			"updated_at":           time.Now(),
		})
	DeactivateEmergencyMode(db)
}

// ── Cross Rollout ──────────────────────────────────────────────────────────────────

func AdvanceCrossRollout(db *gorm.DB) int {
	config := loadCrossConfig(db)
	newPercent := config.RolloutPercent
	switch newPercent {
	case 5:
		newPercent = 25
	case 25:
		newPercent = 50
	case 50:
		newPercent = 100
	default:
		if newPercent < 100 {
			newPercent = 100
		}
	}

	db.Model(&CrossConfig{}).Where("id = ?", config.ID).
		Update("rollout_percent", newPercent)
	return newPercent
}

func RollbackCrossRollout(db *gorm.DB) int {
	config := loadCrossConfig(db)
	newPercent := config.RolloutPercent
	switch newPercent {
	case 100:
		newPercent = 50
	case 50:
		newPercent = 25
	case 25:
		newPercent = 5
	default:
		newPercent = 5
	}

	db.Model(&CrossConfig{}).Where("id = ?", config.ID).
		Update("rollout_percent", newPercent)
	return newPercent
}

// ── Cross Conversion Check ────────────────────────────────────────────────────────

func CheckCrossConversion(db *gorm.DB) (bool, float64) {
	config := loadCrossConfig(db)

	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	oneDayAgo := now.Add(-24 * time.Hour)

	var recentTotal int64
	var recentBought int64
	db.Model(&CrossEvent{}).Where("created_at > ?", oneHourAgo).Count(&recentTotal)
	db.Model(&CrossEvent{}).Where("created_at > ? AND did_buy = ?", oneHourAgo, true).Count(&recentBought)

	var baselineTotal int64
	var baselineBought int64
	db.Model(&CrossEvent{}).Where("created_at > ? AND created_at <= ?", oneDayAgo, oneHourAgo).Count(&baselineTotal)
	db.Model(&CrossEvent{}).Where("created_at > ? AND created_at <= ? AND did_buy = ?", oneDayAgo, oneHourAgo, true).Count(&baselineBought)

	if recentTotal < 20 || baselineTotal < 50 {
		return false, 0
	}

	recentRate := float64(recentBought) / float64(recentTotal)
	baselineRate := float64(baselineBought) / float64(baselineTotal)

	if baselineRate-recentRate > config.ConversionDropThreshold {
		ActivateCrossEmergency(db)
		return true, recentRate
	}

	return false, recentRate
}

// ── Cross Dashboard ──────────────────────────────────────────────────────────────

func GetCrossDashboard(db *gorm.DB) *CrossDashboard {
	config := loadCrossConfig(db)

	var totalDecisions int64
	db.Model(&CrossEvent{}).Count(&totalDecisions)

	bySource := make(map[string]int64)
	sources := []string{"rl", "blend", "fallback", "rules", "emergency"}
	for _, s := range sources {
		var count int64
		db.Model(&CrossEvent{}).Where("source_pricing = ? OR source_ranking = ? OR source_recs = ?", s, s, s).Count(&count)
		if count > 0 {
			bySource[s] = count
		}
	}

	var avgConf struct{ Avg float64 }
	db.Model(&CrossEvent{}).Select("COALESCE(AVG(confidence), 0) as avg").Scan(&avgConf)

	var boughtCount, clickCount int64
	db.Model(&CrossEvent{}).Where("did_buy = ?", true).Count(&boughtCount)
	db.Model(&CrossEvent{}).Where("did_click = ?", true).Count(&clickCount)

	attachRate := 0.0
	clickRate := 0.0
	if totalDecisions > 0 {
		attachRate = float64(boughtCount) / float64(totalDecisions)
		clickRate = float64(clickCount) / float64(totalDecisions)
	}

	var avgReward struct{ Avg float64 }
	db.Model(&CrossTransition{}).Where("reward_total != 0").
		Select("COALESCE(AVG(reward_total), 0) as avg").Scan(&avgReward)

	// Consistency violations (from guardrails log)
	var violations int64
	db.Model(&CrossEvent{}).
		Where("guardrails_json LIKE ?", "%consistency%").Count(&violations)

	return &CrossDashboard{
		Config:                config,
		TotalDecisions:        totalDecisions,
		DecisionsBySource:     bySource,
		AvgConfidence:         avgConf.Avg,
		AttachRate:            attachRate,
		ClickRate:             clickRate,
		AvgReward:             avgReward.Avg,
		EmergencyModeActive:   config.EmergencyModeActive,
		RolloutPercent:        config.RolloutPercent,
		ConsistencyViolations: violations,
	}
}
