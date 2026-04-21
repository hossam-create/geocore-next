package waitlist

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrDailyLimitReached = errors.New("waitlist: daily invite limit reached")

// WaitlistConfig holds operator-controlled scarcity knobs (single-row table).
type WaitlistConfig struct {
	ID                  uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	DailyInviteLimit    int       `gorm:"not null;default:50"      json:"daily_invite_limit"`
	InvitesReleasedToday int      `gorm:"not null;default:0"       json:"invites_released_today"`
	LastResetAt         time.Time `gorm:"not null"                 json:"last_reset_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (WaitlistConfig) TableName() string { return "waitlist_config" }

// GetConfig fetches (or seeds) the single config row.
func GetConfig(db *gorm.DB) (*WaitlistConfig, error) {
	var cfg WaitlistConfig
	err := db.First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cfg = WaitlistConfig{
			DailyInviteLimit:    50,
			InvitesReleasedToday: 0,
			LastResetAt:         time.Now().Truncate(24 * time.Hour),
		}
		db.Create(&cfg)
		return &cfg, nil
	}
	return &cfg, err
}

var configMu sync.Mutex

// CheckAndRecordRelease validates and records a bulk invite release atomically.
// Returns ErrDailyLimitReached if the daily cap would be exceeded.
func CheckAndRecordRelease(db *gorm.DB, n int) error {
	configMu.Lock()
	defer configMu.Unlock()

	return db.Transaction(func(tx *gorm.DB) error {
		var cfg WaitlistConfig
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&cfg).Error; err != nil {
			return err
		}
		// Reset counter if the stored date is stale (new calendar day).
		today := time.Now().Truncate(24 * time.Hour)
		if cfg.LastResetAt.Before(today) {
			cfg.InvitesReleasedToday = 0
			cfg.LastResetAt = today
			slog.Info("waitlist: daily invite counter reset")
		}
		if cfg.DailyInviteLimit > 0 && cfg.InvitesReleasedToday+n > cfg.DailyInviteLimit {
			return ErrDailyLimitReached
		}
		cfg.InvitesReleasedToday += n
		return tx.Save(&cfg).Error
	})
}

// SetDailyLimit lets an admin update the daily release cap.
func SetDailyLimit(db *gorm.DB, limit int) error {
	return db.Model(&WaitlistConfig{}).Where("1 = 1").Update("daily_invite_limit", limit).Error
}

// ─── Admin Analytics ───────────────────────────────────────────────────────

// DailyRelease is a row in the per-day invite release breakdown.
type DailyRelease struct {
	Day      string `json:"day"`
	Released int    `json:"released"`
}

// TopInviter is a row in the top-referrers leaderboard.
type TopInviter struct {
	Email         string  `json:"email"`
	ReferralCount int     `json:"referral_count"`
	Qualified     int     `json:"qualified"`
	ConversionPct float64 `json:"conversion_pct"`
}

// AdminAnalytics returns invite usage per day, conversion rate, and top inviters.
func AdminAnalytics(db *gorm.DB) map[string]interface{} {
	// Daily release for last 14 days.
	var daily []DailyRelease
	db.Raw(`
		SELECT TO_CHAR(created_at, 'YYYY-MM-DD') AS day,
		       COUNT(*) AS released
		FROM waitlist_users
		WHERE status IN ('invited','joined')
		  AND created_at >= NOW() - INTERVAL '14 days'
		GROUP BY day
		ORDER BY day
	`).Scan(&daily)

	// Referral conversion rate: qualified / total referred.
	var totalReferred, totalQualified int64
	db.Model(&WaitlistUser{}).Where("referred_by IS NOT NULL").Count(&totalReferred)
	// Qualified = those who have joined (status = joined).
	db.Model(&WaitlistUser{}).Where("referred_by IS NOT NULL AND status = 'joined'").Count(&totalQualified)
	convRate := 0.0
	if totalReferred > 0 {
		convRate = float64(totalQualified) / float64(totalReferred) * 100
	}

	// Top 20 inviters.
	var top []TopInviter
	db.Raw(`
		SELECT w.email,
		       w.referral_count,
		       COUNT(r.id) FILTER (WHERE r.status = 'joined') AS qualified,
		       CASE WHEN w.referral_count > 0
		            THEN ROUND(COUNT(r.id) FILTER (WHERE r.status = 'joined')::NUMERIC
		                       / w.referral_count * 100, 2)
		            ELSE 0
		       END AS conversion_pct
		FROM waitlist_users w
		LEFT JOIN waitlist_users r ON r.referred_by = w.referral_code
		WHERE w.referral_count > 0
		GROUP BY w.id, w.email, w.referral_count
		ORDER BY qualified DESC, w.referral_count DESC
		LIMIT 20
	`).Scan(&top)

	cfg, _ := GetConfig(db)
	return map[string]interface{}{
		"daily_releases":          daily,
		"referral_conversion_pct": convRate,
		"total_referred":          totalReferred,
		"total_qualified":         totalQualified,
		"top_inviters":            top,
		"daily_config":            cfg,
	}
}
