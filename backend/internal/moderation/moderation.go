package moderation

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/geocore-next/backend/internal/ops"
	"github.com/geocore-next/backend/pkg/jobs"
	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const keywordsCacheKey = "moderation:keywords"
const keywordsCacheTTL = 5 * time.Minute

var (
	storeMu       sync.RWMutex
	globalDB      *gorm.DB
	globalRDB     *redis.Client
	localKeywords []RestrictedKeyword
	localExpiry   time.Time
)

func InitStore(db *gorm.DB, rdb *redis.Client) {
	storeMu.Lock()
	defer storeMu.Unlock()
	globalDB = db
	globalRDB = rdb
	localKeywords = nil
	localExpiry = time.Time{}
}

// RestrictedKeyword defines a blocked/flagged keyword.
type RestrictedKeyword struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Keyword   string `gorm:"size:100;not null" json:"keyword"`
	Severity  string `gorm:"size:20;not null" json:"severity"` // block | flag
	MessageEn string `gorm:"size:300" json:"message_en"`
	MessageAr string `gorm:"size:300" json:"message_ar"`
	Category  string `gorm:"size:50" json:"category,omitempty"`
	IsActive  bool   `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time
}

func (RestrictedKeyword) TableName() string { return "restricted_keywords" }

// ModerationLog records moderation actions.
type ModerationLog struct {
	ID          uint       `gorm:"primaryKey;autoIncrement"`
	TargetType  string     `gorm:"size:20;not null" json:"target_type"` // listing | auction | request
	TargetID    uuid.UUID  `gorm:"type:uuid;not null" json:"target_id"`
	Action      string     `gorm:"size:20;not null" json:"action"` // blocked | flagged | approved | rejected
	Reason      string     `gorm:"type:text" json:"reason,omitempty"`
	ModeratorID *uuid.UUID `gorm:"type:uuid" json:"moderator_id,omitempty"`
	CreatedAt   time.Time
}

func (ModerationLog) TableName() string { return "moderation_logs" }

func getModerationMode() string {
	mode := strings.ToUpper(strings.TrimSpace(ops.ConfigGet("feature.moderation_auto")))
	if mode == "" {
		mode = strings.ToUpper(strings.TrimSpace(os.Getenv("FEATURE_MODERATION_AUTO")))
	}
	switch mode {
	case "OFF", "WARN", "BLOCK":
		return mode
	default:
		return "OFF"
	}
}

func loadKeywords() []RestrictedKeyword {
	storeMu.RLock()
	db := globalDB
	rdb := globalRDB
	if len(localKeywords) > 0 && time.Now().Before(localExpiry) {
		cached := make([]RestrictedKeyword, len(localKeywords))
		copy(cached, localKeywords)
		storeMu.RUnlock()
		return cached
	}
	storeMu.RUnlock()

	if rdb != nil {
		if raw, err := rdb.Get(context.Background(), keywordsCacheKey).Result(); err == nil && raw != "" {
			var cached []RestrictedKeyword
			if json.Unmarshal([]byte(raw), &cached) == nil {
				storeMu.Lock()
				localKeywords = cached
				localExpiry = time.Now().Add(keywordsCacheTTL)
				storeMu.Unlock()
				return cached
			}
		}
	}

	var keywords []RestrictedKeyword
	if db != nil {
		db.Where("is_active = true").Find(&keywords)

		// Also load from admin_settings "listings.banned_keywords" (JSON array)
		var settingVal struct{ Value string }
		if db.Table("admin_settings").Where("key = ?", "listings.banned_keywords").Select("value").Scan(&settingVal).Error == nil && settingVal.Value != "" {
			var extraKeywords []string
			raw := strings.Trim(settingVal.Value, `"`)
			if json.Unmarshal([]byte(raw), &extraKeywords) == nil {
				for _, kw := range extraKeywords {
					kw = strings.TrimSpace(kw)
					if kw != "" {
						keywords = append(keywords, RestrictedKeyword{
							Keyword:   kw,
							Severity:  "block",
							MessageEn: "Listing contains banned keyword: " + kw,
							IsActive:  true,
						})
					}
				}
			}
		}
	}

	b, _ := json.Marshal(keywords)
	if rdb != nil {
		rdb.Set(context.Background(), keywordsCacheKey, string(b), keywordsCacheTTL)
	}
	storeMu.Lock()
	localKeywords = keywords
	localExpiry = time.Now().Add(keywordsCacheTTL)
	storeMu.Unlock()

	return keywords
}

func checkRawContent(title, description string) (matched bool, reason string) {
	combined := strings.ToLower(title + " " + description)
	keywords := loadKeywords()

	for _, kw := range keywords {
		if strings.Contains(combined, strings.ToLower(kw.Keyword)) {
			return true, kw.MessageEn
		}
	}

	// Suspicious patterns (hardcoded fallback)
	suspiciousPatterns := []string{
		"cheap fake", "cheap replica",
		"call me now", "whatsapp me", "email me now",
	}
	for _, p := range suspiciousPatterns {
		if strings.Contains(combined, p) {
			return true, "Content flagged for review"
		}
	}
	return false, ""
}

// CheckContent checks title + description according to feature.moderation_auto mode.
// Modes:
// OFF => always allow
// WARN => log only
// BLOCK => reject matched content
func CheckContent(title, description string) (blocked bool, reason string) {
	mode := getModerationMode()
	if mode == "OFF" {
		return false, ""
	}

	matched, reason := checkRawContent(title, description)
	if !matched {
		return false, ""
	}
	if mode == "WARN" {
		slog.Warn("moderation: content matched in WARN mode",
			"reason", reason,
			"mode", mode,
		)
		return false, ""
	}

	slog.Warn("moderation: content blocked",
		"reason", reason,
		"mode", mode,
	)
	metrics.IncModerationBlocksTotal()
	return true, reason
}

// LogAction records a moderation action in the database (async via job queue for scaling).
func LogAction(db *gorm.DB, targetType string, targetID uuid.UUID, action string, reason string, moderatorID *uuid.UUID) {
	// Enqueue async job to offload DB write from hot path
	_ = jobs.EnqueueDefault(&jobs.Job{
		Type:        jobs.JobTypeModerationLog,
		Priority:    7,
		MaxAttempts: 3,
		Payload: map[string]interface{}{
			"target_type":  targetType,
			"target_id":    targetID.String(),
			"action":       action,
			"reason":       reason,
			"moderator_id": moderatorID,
		},
	})
}

// LogActionSync performs synchronous moderation logging (fallback for critical paths).
func LogActionSync(db *gorm.DB, targetType string, targetID uuid.UUID, action string, reason string, moderatorID *uuid.UUID) {
	db.Create(&ModerationLog{
		TargetType:  targetType,
		TargetID:    targetID,
		Action:      action,
		Reason:      reason,
		ModeratorID: moderatorID,
	})
}
