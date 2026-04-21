package growth

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ── User State Engine ─────────────────────────────────────────────────────────────
//
// Real-time session brain: tracks every user's state across sessions.
// Redis (hot, 30min TTL) + Postgres (cold, source of truth).
// Updated on every event (view, bid, click, purchase).

// UserState represents the real-time state of a user.
type UserState struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID            uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	LastActiveAt      *time.Time `json:"last_active_at"`
	SessionDurationSec float64   `gorm:"type:numeric(12,2);not null;default:0" json:"session_duration_sec"`
	ActionsLast5m     int       `gorm:"not null;default:0" json:"actions_last_5m"`
	ActionsLast1h     int       `gorm:"not null;default:0" json:"actions_last_1h"`
	BidsCount         int       `gorm:"not null;default:0" json:"bids_count"`
	PurchasesCount    int       `gorm:"not null;default:0" json:"purchases_count"`
	ViewsCount        int       `gorm:"not null;default:0" json:"views_count"`
	SavesCount        int       `gorm:"not null;default:0" json:"saves_count"`
	LossesCount       int       `gorm:"not null;default:0" json:"losses_count"` // outbid / lost auction
	DropOffRiskScore  float64   `gorm:"type:numeric(5,4);not null;default:0" json:"drop_off_risk_score"` // 0-1
	EngagementScore   float64   `gorm:"type:numeric(8,2);not null;default:50" json:"engagement_score"` // 0-100
	DopamineScore     float64   `gorm:"type:numeric(8,2);not null;default:50" json:"dopamine_score"`   // 0-100
	Segment           string    `gorm:"size:20;not null;default:'active'" json:"segment"` // active/warm/cold/churn/vip
	PreferredChannel  string    `gorm:"size:20;not null;default:'push'" json:"preferred_channel"`
	ExperimentGroup   string    `gorm:"size:20;not null;default:'control'" json:"experiment_group"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (UserState) TableName() string { return "growth_user_states" }

// UserActionEvent records a raw action for analytics.
type UserActionEvent struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Action    string    `gorm:"size:30;not null;index" json:"action"` // view, click, bid, purchase, save, outbid, win, lose
	ItemID    uuid.UUID `gorm:"type:uuid" json:"item_id"`
	SessionID string    `gorm:"size:50" json:"session_id"`
	Metadata  string    `gorm:"type:text" json:"metadata"` // JSON
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

func (UserActionEvent) TableName() string { return "growth_action_events" }

// ── User State Service ──────────────────────────────────────────────────────────────

type UserStateService struct {
	db  *gorm.DB
	rdb *redis.Client
	ttl time.Duration
}

func NewUserStateService(db *gorm.DB, rdb *redis.Client) *UserStateService {
	return &UserStateService{
		db:  db,
		rdb: rdb,
		ttl: 30 * time.Minute,
	}
}

// GetUserState fetches user state: Redis first, then DB.
func (s *UserStateService) GetUserState(userID uuid.UUID) (*UserState, error) {
	ctx := context.Background()
	key := fmt.Sprintf("growth:state:%s", userID)

	// Try Redis
	val, err := s.rdb.Get(ctx, key).Result()
	if err == nil {
		var state UserState
		if json.Unmarshal([]byte(val), &state) == nil {
			return &state, nil
		}
	}

	// Fallback to DB
	var state UserState
	if err := s.db.Where("user_id = ?", userID).First(&state).Error; err != nil {
		// Create default state
		state = UserState{
			UserID:          userID,
			EngagementScore: 50,
			DopamineScore:   50,
			Segment:         "active",
		}
		s.db.Create(&state)
	}

	// Write-through to Redis
	s.cacheState(&state)
	return &state, nil
}

// RecordAction updates user state based on an action event.
func (s *UserStateService) RecordAction(userID uuid.UUID, action string, itemID uuid.UUID, sessionID string, metadata string) (*UserState, error) {
	// Record raw event
	event := UserActionEvent{
		UserID:    userID,
		Action:    action,
		ItemID:    itemID,
		SessionID: sessionID,
		Metadata:  metadata,
	}
	s.db.Create(&event)

	// Load current state
	state, _ := s.GetUserState(userID)

	// Update counters
	now := time.Now()
	state.LastActiveAt = &now
	state.UpdatedAt = now

	switch action {
	case "view":
		state.ViewsCount++
		state.ActionsLast5m++
		state.ActionsLast1h++
	case "click":
		state.ActionsLast5m++
		state.ActionsLast1h++
	case "bid":
		state.BidsCount++
		state.ActionsLast5m++
		state.ActionsLast1h++
	case "purchase":
		state.PurchasesCount++
		state.ActionsLast5m++
		state.ActionsLast1h++
	case "save":
		state.SavesCount++
		state.ActionsLast5m++
		state.ActionsLast1h++
	case "outbid", "lose":
		state.LossesCount++
	case "win":
		// Handled in dopamine engine
	}

	// Recompute engagement score
	state.EngagementScore = computeEngagementScore(state)

	// Recompute drop-off risk
	state.DropOffRiskScore = computeDropOffRisk(state)

	// Recompute segment
	state.Segment = computeSegment(state)

	// Save to DB
	s.db.Save(state)

	// Update Redis
	s.cacheState(state)

	return state, nil
}

// DecayActionCounts reduces short-term counters (call periodically).
func (s *UserStateService) DecayActionCounts() {
	// Decay actions_last_5m every 5 minutes
	s.db.Model(&UserState{}).Where("actions_last_5m > 0").
		Update("actions_last_5m", gorm.Expr("GREATEST(actions_last_5m - 1, 0)"))

	// Decay actions_last_1h every hour
	s.db.Model(&UserState{}).Where("actions_last_1h > 0").
		Update("actions_last_1h", gorm.Expr("GREATEST(actions_last_1h - 1, 0)"))
}

// ── Score Computations ────────────────────────────────────────────────────────────────

func computeEngagementScore(state *UserState) float64 {
	// Engagement = weighted combination of recent activity
	score := 0.0

	// Recent actions (0-40 points)
	score += math.Min(float64(state.ActionsLast5m)*8, 40)

	// Bids (0-20 points)
	score += math.Min(float64(state.BidsCount)*2, 20)

	// Purchases (0-20 points)
	score += math.Min(float64(state.PurchasesCount)*4, 20)

	// Session duration (0-20 points)
	score += math.Min(state.SessionDurationSec/60.0*0.5, 20)

	// Penalty for losses (0-10 points subtracted)
	score -= math.Min(float64(state.LossesCount)*2, 10)

	return math.Max(0, math.Min(100, score))
}

func computeDropOffRisk(state *UserState) float64 {
	risk := 0.0

	// Inactivity (0-0.4)
	if state.LastActiveAt != nil {
		hoursInactive := time.Since(*state.LastActiveAt).Hours()
		risk += math.Min(hoursInactive/24.0, 1.0) * 0.4
	}

	// Low engagement (0-0.3)
	risk += (1.0 - state.EngagementScore/100.0) * 0.3

	// No recent bids (0-0.3)
	if state.BidsCount == 0 {
		risk += 0.3
	} else if state.ActionsLast1h == 0 {
		risk += 0.15
	}

	return math.Max(0, math.Min(1, risk))
}

func computeSegment(state *UserState) string {
	if state.LastActiveAt == nil {
		return "cold"
	}

	hoursInactive := time.Since(*state.LastActiveAt).Hours()

	// VIP: high engagement + purchases
	if state.EngagementScore > 80 && state.PurchasesCount >= 3 {
		return "vip"
	}

	switch {
	case hoursInactive <= 1 && state.EngagementScore > 60:
		return "active"
	case hoursInactive <= 24:
		return "warm"
	case hoursInactive <= 168: // 7 days
		return "cold"
	default:
		return "churn"
	}
}

// ── Cache Helpers ──────────────────────────────────────────────────────────────────────

func (s *UserStateService) cacheState(state *UserState) {
	ctx := context.Background()
	key := fmt.Sprintf("growth:state:%s", state.UserID)
	data, _ := json.Marshal(state)
	s.rdb.Set(ctx, key, data, s.ttl)
}

func (s *UserStateService) invalidateCache(userID uuid.UUID) {
	ctx := context.Background()
	s.rdb.Del(ctx, fmt.Sprintf("growth:state:%s", userID))
}
