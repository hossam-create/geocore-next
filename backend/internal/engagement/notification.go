package engagement

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Notification AI ─────────────────────────────────────────────────────────────────
//
// Event-driven notifications: send only if there's clear value.
// Decision = P(action | user, event, time) * value_score > threshold
//
// Guardrails:
// - Frequency caps (daily/weekly)
// - Quiet hours (per user timezone)
// - Per-user opt-out (granular: push/email/all)
// - Kill switch (global)
// - Audit log for every notification

// NotificationEvent represents an incoming event that might trigger a notification.
type NotifyEvent struct {
	UserID    uuid.UUID `json:"user_id"`
	EventType string    `json:"event_type"` // outbid, price_drop, live_hot, saved_match, auction_ending
	Payload   string    `json:"payload"`    // JSON with event details
	ValueScore float64  `json:"value_score"` // how valuable is this notification to the user?
}

// Decide determines whether to send a notification.
func Decide(db *gorm.DB, event NotifyEvent) NotificationDecision {
	config := loadEngagementConfig(db)

	// ── 1. Kill switch check ──────────────────────────────────────────────────
	if config.KillSwitchActive {
		return NotificationDecision{
			ShouldSend: false,
			Reason:     "kill_switch_active",
		}
	}

	// ── 2. Load user profile ──────────────────────────────────────────────────
	profile := loadOrCreateProfile(db, event.UserID)

	// ── 3. Opt-out check ──────────────────────────────────────────────────────
	if profile.OptOutAll {
		return NotificationDecision{
			ShouldSend: false,
			Reason:     "user_opted_out_all",
		}
	}

	// ── 4. Determine best channel ──────────────────────────────────────────────
	channel := selectChannel(profile, event.EventType)
	if channel == "push" && profile.OptOutPush {
		channel = "in_app"
	}
	if channel == "email" && profile.OptOutEmail {
		channel = "in_app"
	}

	// ── 5. Quiet hours check ──────────────────────────────────────────────────
	if isInQuietHours(profile, config) {
		return NotificationDecision{
			ShouldSend: false,
			Channel:    channel,
			Reason:     "quiet_hours",
		}
	}

	// ── 6. Frequency cap check ────────────────────────────────────────────────
	if profile.NotificationsToday >= config.MaxNotificationsPerDay {
		return NotificationDecision{
			ShouldSend: false,
			Channel:    channel,
			Reason:     fmt.Sprintf("daily_cap_reached (%d)", profile.NotificationsToday),
		}
	}
	if profile.NotificationsThisWeek >= config.MaxNotificationsPerWeek {
		return NotificationDecision{
			ShouldSend: false,
			Channel:    channel,
			Reason:     fmt.Sprintf("weekly_cap_reached (%d)", profile.NotificationsThisWeek),
		}
	}

	// ── 7. Compute P(action) from user's historical open/act rates ────────────
	pAction := computePAction(profile, event.EventType)

	// ── 8. Decision: P(action) * value_score > threshold ──────────────────────
	score := pAction * event.ValueScore
	if score < config.NotificationScoreThreshold {
		return NotificationDecision{
			ShouldSend: false,
			Channel:    channel,
			Score:      score,
			Reason:     fmt.Sprintf("score_below_threshold (%.3f < %.3f)", score, config.NotificationScoreThreshold),
		}
	}

	// ── 9. Determine priority ────────────────────────────────────────────────
	priority := "normal"
	if score > 0.7 {
		priority = "high"
	} else if score < 0.4 {
		priority = "low"
	}

	return NotificationDecision{
		ShouldSend: true,
		Channel:    channel,
		Score:      score,
		Reason:     fmt.Sprintf("p_action=%.2f * value=%.2f = %.3f", pAction, event.ValueScore, score),
		Priority:   priority,
	}
}

// SendNotification records a sent notification (audit trail).
func SendNotification(db *gorm.DB, event NotifyEvent, decision NotificationDecision) error {
	now := time.Now()
	notif := NotificationEvent{
		UserID:    event.UserID,
		EventType: event.EventType,
		Channel:   decision.Channel,
		Score:     decision.Score,
		Reason:    decision.Reason,
		SentAt:    &now,
	}
	db.Create(&notif)

	// Update profile counters
	db.Model(&UserEngagementProfile{}).Where("user_id = ?", event.UserID).
		Updates(map[string]interface{}{
			"notifications_today":     gorm.Expr("notifications_today + 1"),
			"notifications_this_week": gorm.Expr("notifications_this_week + 1"),
			"total_notifications_sent": gorm.Expr("total_notifications_sent + 1"),
		})

	return nil
}

// RecordNotificationOutcome records whether the user opened/acted on a notification.
func RecordNotificationOutcome(db *gorm.DB, notifID uuid.UUID, opened, acted, optedOut bool) error {
	now := time.Now()
	updates := map[string]interface{}{
		"opened":   opened,
		"acted":    acted,
		"opted_out": optedOut,
	}
	if opened {
		updates["opened_at"] = now
	}
	db.Model(&NotificationEvent{}).Where("id = ?", notifID).Updates(updates)

	// Update user profile rates
	var notif NotificationEvent
	db.Where("id = ?", notifID).First(&notif)

	profile := loadOrCreateProfile(db, notif.UserID)
	if opened {
		profile.TotalOpened++
	}
	if acted {
		profile.TotalActed++
	}
	if optedOut {
		profile.OptOutAll = true
	}
	if profile.TotalNotificationsSent > 0 {
		profile.OpenRate = float64(profile.TotalOpened) / float64(profile.TotalNotificationsSent)
		profile.ActRate = float64(profile.TotalActed) / float64(profile.TotalNotificationsSent)
	}
	db.Save(&profile)

	return nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────────────

func selectChannel(profile *UserEngagementProfile, eventType string) string {
	// Urgent events → push, informational → email/in_app
	urgentEvents := map[string]bool{
		"outbid":          true,
		"auction_ending":  true,
		"price_drop":      true,
	}
	if urgentEvents[eventType] && !profile.OptOutPush {
		return "push"
	}
	if !profile.OptOutEmail && (eventType == "saved_match" || eventType == "weekly_digest") {
		return "email"
	}
	return "in_app"
}

func isInQuietHours(profile *UserEngagementProfile, config EngagementConfig) bool {
	start := profile.QuietHoursStart
	end := profile.QuietHoursEnd
	if start == 0 && end == 0 {
		start = config.QuietHoursDefaultStart
		end = config.QuietHoursDefaultEnd
	}

	// Simple check: assume user's local hour
	// In production: use user's timezone
	hour := time.Now().Hour()
	if start > end { // crosses midnight (e.g., 22-8)
		return hour >= start || hour < end
	}
	return hour >= start && hour < end
}

func computePAction(profile *UserEngagementProfile, eventType string) float64 {
	// Base: user's overall open rate
	p := profile.OpenRate
	if p == 0 {
		p = 0.3 // default for new users
	}

	// Adjust by event type (some events have higher action rates)
	eventMultipliers := map[string]float64{
		"outbid":          1.5, // people click outbid alerts
		"price_drop":      1.3,
		"auction_ending":  1.4,
		"live_hot":        1.1,
		"saved_match":     1.2,
	}
	if m, ok := eventMultipliers[eventType]; ok {
		p = math.Min(p*m, 1.0)
	}

	return p
}

func loadOrCreateProfile(db *gorm.DB, userID uuid.UUID) *UserEngagementProfile {
	var profile UserEngagementProfile
	if err := db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		profile = UserEngagementProfile{
			UserID:            userID,
			Segment:           SegmentActive,
			PreferredChannels: "push,in_app",
		}
		db.Create(&profile)
	}
	return &profile
}

func loadEngagementConfig(db *gorm.DB) EngagementConfig {
	var config EngagementConfig
	if err := db.Where("is_active = ?", true).Order("created_at DESC").First(&config).Error; err != nil {
		return EngagementConfig{
			MaxNotificationsPerDay:   3,
			MaxNotificationsPerWeek:  12,
			NotificationScoreThreshold: 0.3,
			QuietHoursDefaultStart:  22,
			QuietHoursDefaultEnd:    8,
			ExplorationPercent:       10,
			MomentumHighThreshold:    0.7,
			MomentumLowThreshold:     0.3,
		}
	}
	return config
}
