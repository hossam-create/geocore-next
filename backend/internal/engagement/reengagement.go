package engagement

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Re-engagement Engine ──────────────────────────────────────────────────────────
//
// Segments users by activity and plans targeted, low-frequency campaigns.
//
// Active (0-24h): don't bother — focus inside session
// Warm (1-3 days): "real opportunity" (auction ending / good deal)
// Cold (3-7 days): strong content + social proof
// Churn risk (7+ days): simple incentive (discount / limited boost)

// SegmentUser classifies a user into an engagement segment.
func SegmentUser(db *gorm.DB, userID uuid.UUID) UserSegmentType {
	profile := loadOrCreateProfile(db, userID)

	if profile.LastActiveAt == nil {
		return SegmentCold // never active = cold
	}

	hoursSinceActivity := time.Since(*profile.LastActiveAt).Hours()

	switch {
	case hoursSinceActivity <= 24:
		return SegmentActive
	case hoursSinceActivity <= 72:
		return SegmentWarm
	case hoursSinceActivity <= 168: // 7 days
		return SegmentCold
	default:
		return SegmentChurnRisk
	}
}

// PlanReEngagement creates a re-engagement plan for a user based on their segment.
func PlanReEngagement(db *gorm.DB, userID uuid.UUID) []PlannedTouch {
	segment := SegmentUser(db, userID)
	profile := loadOrCreateProfile(db, userID)
	config := loadEngagementConfig(db)

	// Update profile segment
	profile.Segment = segment
	db.Save(&profile)

	var touches []PlannedTouch

	switch segment {
	case SegmentActive:
		// Don't send re-engagement — focus on in-session momentum
		// Maybe one gentle "saved for you" in-app after 4h idle
		touches = append(touches, PlannedTouch{
			UserID:      userID,
			Segment:     segment,
			Channel:     "in_app",
			MessageType: "discovery",
			ScheduledAt: time.Now().Add(4 * time.Hour),
			Status:      "planned",
		})

	case SegmentWarm:
		// "Real opportunity" — auction ending, price drop, good deal
		touches = append(touches, PlannedTouch{
			UserID:      userID,
			Segment:     segment,
			Channel:     selectChannel(profile, "price_drop"),
			MessageType: "opportunity",
			ScheduledAt: findBestSendTime(db, userID, 1),
			Status:      "planned",
		})

	case SegmentCold:
		// Strong content + social proof
		touches = append(touches, PlannedTouch{
			UserID:      userID,
			Segment:     segment,
			Channel:     selectChannel(profile, "social_proof"),
			MessageType: "social_proof",
			ScheduledAt: findBestSendTime(db, userID, 2),
			Status:      "planned",
		})
		// Second touch: discovery
		touches = append(touches, PlannedTouch{
			UserID:      userID,
			Segment:     segment,
			Channel:     "in_app",
			MessageType: "discovery",
			ScheduledAt: findBestSendTime(db, userID, 4),
			Status:      "planned",
		})

	case SegmentChurnRisk:
		// Simple incentive
		if profile.NotificationsThisWeek < config.MaxNotificationsPerWeek {
			touches = append(touches, PlannedTouch{
				UserID:      userID,
				Segment:     segment,
				Channel:     selectChannel(profile, "incentive"),
				MessageType: "incentive",
				ScheduledAt: findBestSendTime(db, userID, 1),
				Status:      "planned",
			})
		}
	}

	// Save planned touches
	for i := range touches {
		db.Create(&touches[i])
	}

	return touches
}

// ── Segment All Users (batch job) ──────────────────────────────────────────────────

// SegmentAllUsers updates segments for all engagement profiles.
func SegmentAllUsers(db *gorm.DB) map[UserSegmentType]int64 {
	var profiles []UserEngagementProfile
	db.Find(&profiles)

	counts := map[UserSegmentType]int64{
		SegmentActive:    0,
		SegmentWarm:      0,
		SegmentCold:      0,
		SegmentChurnRisk: 0,
	}

	for _, p := range profiles {
		segment := SegmentUser(db, p.UserID)
		p.Segment = segment
		db.Save(&p)
		counts[segment]++
	}

	return counts
}

// ── Process Planned Touches (cron job) ──────────────────────────────────────────────

// ProcessPlannedTouches sends due notifications from the planned touches queue.
func ProcessPlannedTouches(db *gorm.DB) int {
	now := time.Now()
	var touches []PlannedTouch
	db.Where("status = ? AND scheduled_at <= ?", "planned", now).Find(&touches)

	sent := 0
	for _, t := range touches {
		// Check if user still in same segment (might have re-engaged)
		currentSegment := SegmentUser(db, t.UserID)
		if currentSegment == SegmentActive && t.Segment != SegmentActive {
			// User re-engaged on their own — cancel this touch
			db.Model(&t).Updates(map[string]interface{}{
				"status": "cancelled",
			})
			continue
		}

		// Send the notification
		decision := Decide(db, NotifyEvent{
			UserID:     t.UserID,
			EventType:  mapMessageTypeToEvent(t.MessageType),
			ValueScore: segmentValueScore(t.Segment),
		})

		if decision.ShouldSend {
			SendNotification(db, NotifyEvent{
				UserID:     t.UserID,
				EventType:  mapMessageTypeToEvent(t.MessageType),
				ValueScore: segmentValueScore(t.Segment),
			}, decision)

			now := time.Now()
			db.Model(&t).Updates(map[string]interface{}{
				"status":  "sent",
				"sent_at": now,
				"channel": decision.Channel,
			})
			sent++
		} else {
			db.Model(&t).Updates(map[string]interface{}{
				"status": "cancelled",
			})
		}
	}

	return sent
}

// ── Helpers ──────────────────────────────────────────────────────────────────────────

func mapMessageTypeToEvent(msgType string) string {
	mapping := map[string]string{
		"opportunity":   "price_drop",
		"social_proof":  "saved_match",
		"incentive":     "price_drop",
		"discovery":     "saved_match",
	}
	if e, ok := mapping[msgType]; ok {
		return e
	}
	return "saved_match"
}

func segmentValueScore(segment UserSegmentType) float64 {
	// Higher value score for segments that need more convincing
	scores := map[UserSegmentType]float64{
		SegmentActive:    0.2, // don't need much
		SegmentWarm:      0.5, // moderate
		SegmentCold:      0.7, // need strong value
		SegmentChurnRisk: 0.9, // need very strong value
	}
	if s, ok := scores[segment]; ok {
		return s
	}
	return 0.5
}

// findBestSendTime finds the optimal time to send a notification.
// Uses user activity hours histogram + offset days from now.
func findBestSendTime(db *gorm.DB, userID uuid.UUID, daysFromNow int) time.Time {
	var hours []UserActivityHour
	db.Where("user_id = ?", userID).Order("score DESC").Limit(3).Find(&hours)

	if len(hours) == 0 {
		// Default: send at 10am local time
		target := time.Now().AddDate(0, 0, daysFromNow)
		return time.Date(target.Year(), target.Month(), target.Day(), 10, 0, 0, 0, target.Location())
	}

	// Use the best hour
	bestHour := hours[0].Hour
	target := time.Now().AddDate(0, 0, daysFromNow)
	return time.Date(target.Year(), target.Month(), target.Day(), bestHour, 0, 0, 0, target.Location())
}
