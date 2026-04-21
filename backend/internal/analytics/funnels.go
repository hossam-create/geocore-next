package analytics

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Activation Funnels
// Track user progression through buyer/seller/traveler funnels.
// ════════════════════════════════════════════════════════════════════════════

// FunnelEvent tracks a single step in a user's activation funnel.
type FunnelEvent struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_funnel_user" json:"user_id"`
	Funnel    string    `gorm:"size:30;not null;index:idx_funnel_type" json:"funnel"` // buyer, seller, traveler
	Step      string    `gorm:"size:50;not null" json:"step"`
	StepOrder int       `gorm:"not null" json:"step_order"`
	Metadata  string    `gorm:"type:text" json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (FunnelEvent) TableName() string { return "funnel_events" }

// Funnel definitions
var Funnels = map[string][]FunnelStep{
	"buyer": {
		{Step: "signup", Order: 1},
		{Step: "search", Order: 2},
		{Step: "request", Order: 3},
		{Step: "offer_received", Order: 4},
		{Step: "pay", Order: 5},
	},
	"seller": {
		{Step: "signup", Order: 1},
		{Step: "create_listing", Order: 2},
		{Step: "boost", Order: 3},
		{Step: "get_offers", Order: 4},
	},
	"traveler": {
		{Step: "signup", Order: 1},
		{Step: "add_trip", Order: 2},
		{Step: "receive_request", Order: 3},
		{Step: "send_offer", Order: 4},
	},
}

// FunnelStep defines a single step in a funnel.
type FunnelStep struct {
	Step  string
	Order int
}

// RecordFunnelEvent records a user's progression through a funnel step.
func RecordFunnelEvent(db *gorm.DB, userID uuid.UUID, funnel, step string) error {
	steps, ok := Funnels[funnel]
	if !ok {
		return nil // unknown funnel — silently skip
	}

	var stepOrder int
	for _, s := range steps {
		if s.Step == step {
			stepOrder = s.Order
			break
		}
	}
	if stepOrder == 0 {
		return nil // unknown step — silently skip
	}

	event := FunnelEvent{
		ID:        uuid.New(),
		UserID:    userID,
		Funnel:    funnel,
		Step:      step,
		StepOrder: stepOrder,
	}

	// Only record if this step hasn't been recorded yet (idempotent)
	result := db.Where("user_id = ? AND funnel = ? AND step = ?", userID, funnel, step).
		FirstOrCreate(&event)
	return result.Error
}

// FunnelDropoff represents drop-off analysis for a funnel.
type FunnelDropoff struct {
	Funnel     string          `json:"funnel"`
	Step       string          `json:"step"`
	StepOrder  int             `json:"step_order"`
	Count      int64           `json:"count"`
	Reached    int64           `json:"reached"`
	Converted  int64           `json:"converted"`
	DropoffPct decimal.Decimal `json:"dropoff_pct"`
}

// GetFunnelDropoffs returns drop-off analysis for all funnels.
func GetFunnelDropoffs(db *gorm.DB) []FunnelDropoff {
	var results []FunnelDropoff

	for funnelName, steps := range Funnels {
		for i, step := range steps {
			var reached int64
			db.Model(&FunnelEvent{}).Where("funnel = ? AND step_order >= ?", funnelName, step.Order).
				Distinct("user_id").Count(&reached)

			var converted int64
			if i < len(steps)-1 {
				db.Model(&FunnelEvent{}).Where("funnel = ? AND step_order >= ?", funnelName, steps[i+1].Order).
					Distinct("user_id").Count(&converted)
			} else {
				converted = reached // last step
			}

			dropoffPct := decimal.Zero
			if reached > 0 {
				dropoffPct = decimal.NewFromInt(reached - converted).Div(decimal.NewFromInt(reached)).Mul(decimal.NewFromInt(100))
			}

			results = append(results, FunnelDropoff{
				Funnel:     funnelName,
				Step:       step.Step,
				StepOrder:  step.Order,
				Reached:    reached,
				Converted:  converted,
				DropoffPct: dropoffPct,
			})
		}
	}

	return results
}

// GetUserFunnelProgress returns a user's progress through all funnels.
func GetUserFunnelProgress(db *gorm.DB, userID uuid.UUID) map[string]interface{} {
	progress := make(map[string]interface{})

	for funnelName, steps := range Funnels {
		var events []FunnelEvent
		db.Where("user_id = ? AND funnel = ?", userID, funnelName).
			Order("step_order ASC").Find(&events)

		completedSteps := len(events)
		totalSteps := len(steps)
		currentStep := ""
		if completedSteps > 0 {
			currentStep = events[completedSteps-1].Step
		}

		progress[funnelName] = map[string]interface{}{
			"completed_steps": completedSteps,
			"total_steps":     totalSteps,
			"current_step":    currentStep,
			"pct_complete":    decimal.NewFromInt(int64(completedSteps)).Div(decimal.NewFromInt(int64(totalSteps))).Mul(decimal.NewFromInt(100)).StringFixed(1),
		}
	}

	return progress
}
