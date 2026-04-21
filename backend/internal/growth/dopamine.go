package growth

import (
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Dopamine Loop Engine ──────────────────────────────────────────────────────────
//
// Tracks dopamine score and triggers actions based on emotional state.
// High dopamine → premium items, VIP boosts
// Low dopamine → easier wins, cheap items, re-engagement
//
// Events that increase dopamine:
//   winning bid → +30
//   near win → +15
//   new hot item → +10
//   reward / badge → +20
//
// Events that decrease:
//   inactivity → -10
//   losing multiple times → -15

// DopamineEvent records a dopamine-modifying event.
type DopamineEvent struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	EventType string    `gorm:"size:30;not null;index" json:"event_type"` // win, near_win, hot_item, badge, inactivity, multi_loss
	Delta     float64   `gorm:"type:numeric(8,2);not null" json:"delta"`
	OldScore  float64   `gorm:"type:numeric(8,2);not null" json:"old_score"`
	NewScore  float64   `gorm:"type:numeric(8,2);not null" json:"new_score"`
	ItemID    uuid.UUID `gorm:"type:uuid" json:"item_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (DopamineEvent) TableName() string { return "growth_dopamine_events" }

// DopamineAction represents an action triggered by dopamine state.
type DopamineAction struct {
	Type    string  `json:"type"`    // show_easy_wins, show_premium, send_reengage, inject_cheap, suggest_vip
	Reason  string  `json:"reason"`  // why this action was chosen
	Score   float64 `json:"score"`   // current dopamine score
	Payload string  `json:"payload"` // JSON with action-specific data
}

// DopamineDeltas maps event types to score changes.
var DopamineDeltas = map[string]float64{
	"win":         +30, // winning bid
	"near_win":    +15, // was leading but lost at the end
	"hot_item":    +10, // new trending item in their category
	"badge":       +20, // earned a reward/badge
	"purchase":    +15, // completed a purchase
	"save":        +5,  // saved an item (mild positive)
	"inactivity":  -10, // no action for a while
	"multi_loss":  -15, // lost multiple bids in a row
	"outbid":      -8,  // just got outbid
	"claim_denied": -5, // insurance claim denied
}

// UpdateDopamine applies a dopamine event and returns the new score + recommended actions.
func UpdateDopamine(db *gorm.DB, stateSvc *UserStateService, userID uuid.UUID, eventType string, itemID uuid.UUID) (*DopamineAction, error) {
	delta, ok := DopamineDeltas[eventType]
	if !ok {
		delta = 0
	}

	// Load current state
	state, err := stateSvc.GetUserState(userID)
	if err != nil {
		return nil, err
	}

	oldScore := state.DopamineScore

	// Apply delta with diminishing returns (large scores change less)
	adjustedDelta := delta
	if delta > 0 && oldScore > 70 {
		adjustedDelta = delta * 0.7 // harder to increase when already high
	} else if delta < 0 && oldScore < 30 {
		adjustedDelta = delta * 0.7 // harder to decrease when already low
	}

	newScore := oldScore + adjustedDelta
	newScore = math.Max(0, math.Min(100, newScore))

	// Natural decay toward baseline (50) over time
	if state.LastActiveAt != nil {
		hoursInactive := time.Since(*state.LastActiveAt).Hours()
		decayFactor := math.Min(hoursInactive*2, 20) // max 20 points decay
		if newScore > 50 {
			newScore = math.Max(50, newScore-decayFactor)
		} else if newScore < 50 {
			newScore = math.Min(50, newScore+decayFactor*0.5) // slower recovery
		}
	}

	// Update state
	state.DopamineScore = newScore
	db.Save(state)

	// Record event
	db.Create(&DopamineEvent{
		UserID:    userID,
		EventType: eventType,
		Delta:     adjustedDelta,
		OldScore:  oldScore,
		NewScore:  newScore,
		ItemID:    itemID,
	})

	// Determine action based on new score
	action := decideDopamineAction(newScore, eventType)

	return action, nil
}

// decideDopamineAction chooses an action based on dopamine score.
func decideDopamineAction(score float64, eventType string) *DopamineAction {
	switch {
	case score < 20:
		// Critical low — urgent re-engagement
		return &DopamineAction{
			Type:   "show_easy_wins",
			Reason: "dopamine_critical_low",
			Score:  score,
			Payload: `{"max_price": 500, "sort": "ending_soon", "inject_cheap": true}`,
		}
	case score < 40:
		// Low — re-engage with easier content
		return &DopamineAction{
			Type:   "inject_cheap",
			Reason: "dopamine_low",
			Score:  score,
			Payload: `{"max_price": 2000, "sort": "popular", "show_badges": true}`,
		}
	case score < 60:
		// Balanced — normal experience
		return &DopamineAction{
			Type:   "balanced",
			Reason: "dopamine_balanced",
			Score:  score,
			Payload: `{}`,
		}
	case score < 80:
		// High — show premium, suggest boosts
		return &DopamineAction{
			Type:   "show_premium",
			Reason: "dopamine_high",
			Score:  score,
			Payload: `{"min_price": 5000, "suggest_boost": true, "show_vip": false}`,
		}
	default:
		// Very high — VIP mode
		return &DopamineAction{
			Type:   "suggest_vip",
			Reason: "dopamine_very_high",
			Score:  score,
			Payload: `{"show_vip": true, "show_exclusive": true, "suggest_boost": true}`,
		}
	}
}

// GetDopamineAction returns the current recommended action for a user.
func GetDopamineAction(db *gorm.DB, stateSvc *UserStateService, userID uuid.UUID) *DopamineAction {
	state, err := stateSvc.GetUserState(userID)
	if err != nil {
		return &DopamineAction{Type: "balanced", Reason: "default", Score: 50}
	}
	return decideDopamineAction(state.DopamineScore, "")
}

// ── Dopamine Metrics ────────────────────────────────────────────────────────────────

type DopamineMetrics struct {
	AvgScore        float64           `json:"avg_score"`
	UsersBelow40    int64             `json:"users_below_40"`
	UsersAbove80    int64             `json:"users_above_80"`
	TopEvents       []DopamineEventStats `json:"top_events"`
}

type DopamineEventStats struct {
	EventType string  `json:"event_type"`
	Count     int64   `json:"count"`
	AvgDelta  float64 `json:"avg_delta"`
}

func GetDopamineMetrics(db *gorm.DB) *DopamineMetrics {
	var avg struct{ Avg float64 }
	db.Model(&UserState{}).Select("COALESCE(AVG(dopamine_score), 50) as avg").Scan(&avg)

	var below40, above80 int64
	db.Model(&UserState{}).Where("dopamine_score < 40").Count(&below40)
	db.Model(&UserState{}).Where("dopamine_score > 80").Count(&above80)

	var eventStats []DopamineEventStats
	db.Model(&DopamineEvent{}).
		Select("event_type, COUNT(*) as count, AVG(delta) as avg_delta").
		Group("event_type").Order("count DESC").Limit(10).Scan(&eventStats)

	return &DopamineMetrics{
		AvgScore:     avg.Avg,
		UsersBelow40: below40,
		UsersAbove80: above80,
		TopEvents:    eventStats,
	}
}
