package engagement

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Timing Engine ─────────────────────────────────────────────────────────────────
//
// Learns the best send-time per user using activity histogram + bandit.
// Best time = when user is usually active AND there's a valuable event AND low notification congestion.
//
// Simple: histogram of user activity by hour
// Advanced: bandit tries different hours and learns which gets best open rate

// RecordActivity records that a user was active at a specific hour.
func RecordActivity(db *gorm.DB, userID uuid.UUID) error {
	hour := time.Now().Hour()
	dayOfWeek := int(time.Now().Weekday())

	var ah UserActivityHour
	if err := db.Where("user_id = ? AND hour = ? AND (day_of_week = ? OR day_of_week = -1)",
		userID, hour, dayOfWeek).First(&ah).Error; err != nil {
		// Create new entry
		ah = UserActivityHour{
			UserID:    userID,
			Hour:      hour,
			DayOfWeek: -1, // aggregate all days initially
			Count:     1,
		}
		db.Create(&ah)
	} else {
		db.Model(&ah).Updates(map[string]interface{}{
			"count":      ah.Count + 1,
			"updated_at": time.Now(),
		})
	}

	// Recompute scores for this user
	go recomputeHourScores(db, userID)

	return nil
}

// recomputeHourScores normalizes activity counts to 0-1 scores.
func recomputeHourScores(db *gorm.DB, userID uuid.UUID) {
	var hours []UserActivityHour
	db.Where("user_id = ? AND day_of_week = -1", userID).Find(&hours)

	if len(hours) == 0 {
		return
	}

	// Find max count
	maxCount := 0
	for _, h := range hours {
		if h.Count > maxCount {
			maxCount = h.Count
		}
	}

	if maxCount == 0 {
		return
	}

	// Normalize
	for _, h := range hours {
		score := float64(h.Count) / float64(maxCount)
		db.Model(&UserActivityHour{}).Where("id = ?", h.ID).
			Updates(map[string]interface{}{
				"score":      score,
				"updated_at": time.Now(),
			})
	}
}

// GetBestSendTime returns the optimal hour to send a notification to a user.
func GetBestSendTime(db *gorm.DB, userID uuid.UUID) (int, float64) {
	var hours []UserActivityHour
	db.Where("user_id = ? AND day_of_week = -1", userID).
		Order("score DESC").Limit(3).Find(&hours)

	if len(hours) == 0 {
		return 10, 0.5 // default: 10am with moderate confidence
	}

	return hours[0].Hour, hours[0].Score
}

// ── Send-Time Bandit ────────────────────────────────────────────────────────────────
//
// Instead of just using the histogram, try different hours and learn from outcomes.
// This is a lightweight bandit: each hour is an "arm", open rate is the reward.

// SendTimeBandit tracks open rates per hour for send-time optimization.
type SendTimeBandit struct {
	db *gorm.DB
}

func NewSendTimeBandit(db *gorm.DB) *SendTimeBandit {
	return &SendTimeBandit{db: db}
}

// SelectHour chooses an hour to send a notification using ε-greedy.
func (b *SendTimeBandit) SelectHour(userID uuid.UUID, epsilon float64) int {
	var hours []UserActivityHour
	b.db.Where("user_id = ? AND day_of_week = -1", userID).Find(&hours)

	if len(hours) == 0 {
		return 10 // default
	}

	// Build arm data from activity hours
	type arm struct {
		hour    int
		score   float64
		samples int
	}

	arms := make([]arm, 0, len(hours))
	for _, h := range hours {
		// Score is based on open rate (approximated by activity score)
		// In production: use actual notification open rates per hour
		arms = append(arms, arm{
			hour:    h.Hour,
			score:   h.Score,
			samples: h.Count,
		})
	}

	// ε-greedy: explore epsilon of the time
	if len(arms) > 0 && randFloat() < epsilon {
		// Random hour from user's active hours
		return arms[randInt(len(arms))].hour
	}

	// Exploit: best scoring hour
	bestHour := 10
	bestScore := 0.0
	for _, a := range arms {
		if a.score > bestScore {
			bestScore = a.score
			bestHour = a.hour
		}
	}

	return bestHour
}

// RecordSendOutcome updates the bandit with the outcome of a send at a specific hour.
func (b *SendTimeBandit) RecordSendOutcome(userID uuid.UUID, hour int, opened bool) {
	// Update the activity hour score based on notification outcome
	var ah UserActivityHour
	if err := b.db.Where("user_id = ? AND hour = ? AND day_of_week = -1",
		userID, hour).First(&ah).Error; err != nil {
		// Create if not exists
		ah = UserActivityHour{
			UserID:    userID,
			Hour:      hour,
			DayOfWeek: -1,
			Count:     1,
		}
		if opened {
			ah.Score = 0.5
		}
		b.db.Create(&ah)
		return
	}

	// Update score with exponential moving average
	alpha := 0.1
	reward := 0.0
	if opened {
		reward = 1.0
	}
	newScore := ah.Score + alpha*(reward-ah.Score)
	b.db.Model(&ah).Updates(map[string]interface{}{
		"score":      newScore,
		"count":      ah.Count + 1,
		"updated_at": time.Now(),
	})
}

// ── Simple RNG helpers (avoid import cycle with pricing package) ────────────────────

func randFloat() float64 {
	return float64(time.Now().UnixNano()%10000) / 10000.0
}

func randInt(max int) int {
	return int(time.Now().UnixNano()) % max
}
