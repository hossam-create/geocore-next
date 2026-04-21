package growth

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Re-engagement Engine ──────────────────────────────────────────────────────────
//
// Detects churn risk and plans targeted re-engagement actions.
// drop_off_risk = inactivity_time + low_engagement + no_recent_bids
//
// Actions: send push, send email, show special offers, invite to live session

// ReEngagementPlan is a computed plan for a user at risk.
type ReEngagementPlan struct {
	UserID       uuid.UUID `json:"user_id"`
	DropOffRisk  float64   `json:"drop_off_risk"`
	Segment      string    `json:"segment"`
	Actions      []ReEngAction `json:"actions"`
}

// ReEngAction is a specific re-engagement action.
type ReEngAction struct {
	Type        string    `json:"type"`         // push, email, special_offer, live_invite, badge
	Channel     string    `json:"channel"`      // push, email, in_app
	MessageType string    `json:"message_type"` // nudge, reminder, win, loss, promo
	Priority    string    `json:"priority"`     // high, normal, low
	ScheduledAt time.Time `json:"scheduled_at"`
	Payload     string    `json:"payload"` // JSON with action-specific data
}

// ReEngagementLog records a re-engagement action taken.
type ReEngagementLog struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Segment     string    `gorm:"size:20;not null" json:"segment"`
	ActionType  string    `gorm:"size:30;not null" json:"action_type"`
	Channel     string    `gorm:"size:20;not null" json:"channel"`
	Success     bool      `gorm:"not null;default:false" json:"success"` // did user re-engage?
	CreatedAt   time.Time `json:"created_at"`
}

func (ReEngagementLog) TableName() string { return "growth_reengagement_logs" }

// AssessChurnRisk computes churn risk and returns a re-engagement plan.
func AssessChurnRisk(db *gorm.DB, stateSvc *UserStateService, userID uuid.UUID) *ReEngagementPlan {
	state, _ := stateSvc.GetUserState(userID)

	plan := &ReEngagementPlan{
		UserID:      userID,
		DropOffRisk: state.DropOffRiskScore,
		Segment:     state.Segment,
	}

	switch {
	case state.DropOffRiskScore > 0.8:
		// Critical churn risk
		plan.Actions = []ReEngAction{
			{Type: "special_offer", Channel: "push", MessageType: "promo", Priority: "high",
				ScheduledAt: time.Now().Add(1 * time.Hour),
				Payload: `{"discount_percent": 10, "message": "Special offer just for you!"}`},
			{Type: "live_invite", Channel: "push", MessageType: "nudge", Priority: "high",
				ScheduledAt: time.Now().Add(24 * time.Hour),
				Payload: `{"message": "Live auction happening now!"}`},
			{Type: "badge", Channel: "in_app", MessageType: "win", Priority: "normal",
				ScheduledAt: time.Now().Add(48 * time.Hour),
				Payload: `{"badge": "comeback_hero", "reward": "free_boost"}`},
		}

	case state.DropOffRiskScore > 0.5:
		// Moderate risk
		plan.Actions = []ReEngAction{
			{Type: "push", Channel: "push", MessageType: "reminder", Priority: "normal",
				ScheduledAt: time.Now().Add(6 * time.Hour),
				Payload: `{"message": "Items you saved are trending!"}`},
			{Type: "live_invite", Channel: "email", MessageType: "nudge", Priority: "normal",
				ScheduledAt: time.Now().Add(48 * time.Hour),
				Payload: `{"message": "Don't miss this live session"}`},
		}

	case state.DropOffRiskScore > 0.3:
		// Mild risk — gentle nudge
		plan.Actions = []ReEngAction{
			{Type: "push", Channel: "in_app", MessageType: "reminder", Priority: "low",
				ScheduledAt: time.Now().Add(24 * time.Hour),
				Payload: `{"message": "New items in your categories"}`},
		}

	default:
		// Low risk — no action needed
		plan.Actions = []ReEngAction{}
	}

	// Dopamine-aware adjustments
	if state.DopamineScore < 40 {
		// Add an easy-win action
		plan.Actions = append(plan.Actions, ReEngAction{
			Type: "special_offer", Channel: "push", MessageType: "win", Priority: "high",
			ScheduledAt: time.Now().Add(2 * time.Hour),
			Payload: `{"cheap_items": true, "ending_soon": true, "message": "Easy wins waiting for you!"}`,
		})
	}

	return plan
}

// BatchAssessChurnRisk runs churn assessment for all at-risk users.
func BatchAssessChurnRisk(db *gorm.DB, stateSvc *UserStateService) []ReEngagementPlan {
	var atRisk []UserState
	db.Where("drop_off_risk_score > 0.3 OR segment IN ?", []string{"cold", "churn"}).
		Order("drop_off_risk_score DESC").Limit(1000).Find(&atRisk)

	plans := make([]ReEngagementPlan, 0, len(atRisk))
	for _, u := range atRisk {
		plan := AssessChurnRisk(db, stateSvc, u.UserID)
		plans = append(plans, *plan)
	}
	return plans
}

// RecordReEngagementOutcome tracks whether a re-engagement action succeeded.
func RecordReEngagementOutcome(db *gorm.DB, userID uuid.UUID, segment, actionType, channel string, success bool) {
	db.Create(&ReEngagementLog{
		UserID:     userID,
		Segment:    segment,
		ActionType: actionType,
		Channel:    channel,
		Success:    success,
	})
}

// ── Re-engagement Metrics ────────────────────────────────────────────────────────────

type ReEngagementMetrics struct {
	TotalAttempts int64   `json:"total_attempts"`
	SuccessRate   float64 `json:"success_rate"`
	BySegment     map[string]float64 `json:"by_segment"`
	ByAction      map[string]float64 `json:"by_action"`
}

func GetReEngagementMetrics(db *gorm.DB) *ReEngagementMetrics {
	var total, success int64
	db.Model(&ReEngagementLog{}).Count(&total)
	db.Model(&ReEngagementLog{}).Where("success = ?", true).Count(&success)

	successRate := 0.0
	if total > 0 {
		successRate = float64(success) / float64(total)
	}

	bySegment := map[string]float64{}
	byAction := map[string]float64{}
	var segResults []struct {
		Segment string  `json:"segment"`
		Rate    float64 `json:"rate"`
	}
	db.Model(&ReEngagementLog{}).
		Select("segment, CAST(SUM(CASE WHEN success THEN 1 ELSE 0 END) AS FLOAT) / NULLIF(COUNT(*), 0) as rate").
		Group("segment").Scan(&segResults)
	for _, r := range segResults {
		bySegment[r.Segment] = r.Rate
	}

	var actResults []struct {
		ActionType string  `json:"action_type"`
		Rate       float64 `json:"rate"`
	}
	db.Model(&ReEngagementLog{}).
		Select("action_type, CAST(SUM(CASE WHEN success THEN 1 ELSE 0 END) AS FLOAT) / NULLIF(COUNT(*), 0) as rate").
		Group("action_type").Scan(&actResults)
	for _, r := range actResults {
		byAction[r.ActionType] = r.Rate
	}

	return &ReEngagementMetrics{
		TotalAttempts: total,
		SuccessRate:   successRate,
		BySegment:     bySegment,
		ByAction:      byAction,
	}
}
