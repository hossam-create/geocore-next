package growth

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Unified Decision Engine ───────────────────────────────────────────────────────
//
// DecideNextBestAction(userID) → Action
//
// Input: UserState + Dopamine + Re-engagement + Experiments + Bandit outputs
// Output: single best action for this user right now
//
// Possible actions:
//   show_live_auction
//   send_notification
//   recommend_item
//   do_nothing

// DecisionAction is the output of the decision engine.
type DecisionAction struct {
	UserID     uuid.UUID `json:"user_id"`
	Action     string    `json:"action"`     // show_live_auction, send_notification, recommend_item, do_nothing
	Confidence float64   `json:"confidence"` // 0-1
	Reason     string    `json:"reason"`     // why this action was chosen
	Channel    string    `json:"channel"`    // push, email, in_app
	Payload    string    `json:"payload"`    // JSON with action-specific data
	Sources    []string  `json:"sources"`    // which engines contributed
}

// DecisionLog records every decision for audit.
type DecisionLog struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Action     string    `gorm:"size:30;not null" json:"action"`
	Confidence float64   `gorm:"type:numeric(5,4);not null" json:"confidence"`
	Reason     string    `gorm:"size:200" json:"reason"`
	Sources    string    `gorm:"type:text" json:"sources"`                 // JSON array
	Outcome    string    `gorm:"size:20;default:'pending'" json:"outcome"` // pending, success, fail
	CreatedAt  int64     `gorm:"autoCreateTime" json:"created_at"`
}

func (DecisionLog) TableName() string { return "growth_decision_logs" }

// DecideNextBestAction computes the single best action for a user right now.
func DecideNextBestAction(db *gorm.DB, stateSvc *UserStateService, userID uuid.UUID) *DecisionAction {
	state, _ := stateSvc.GetUserState(userID)
	dopamineAction := GetDopamineAction(db, stateSvc, userID)
	reengagementPlan := AssessChurnRisk(db, stateSvc, userID)

	sources := []string{}
	bestAction := &DecisionAction{
		UserID: userID,
	}

	// ── Priority 1: Critical churn risk ──────────────────────────────────────
	if state.DropOffRiskScore > 0.8 {
		bestAction.Action = "send_notification"
		bestAction.Channel = "push"
		bestAction.Confidence = 0.9
		bestAction.Reason = "critical_churn_risk"
		bestAction.Payload = `{"type": "promo", "message": "We miss you! Special offer inside"}`
		sources = append(sources, "reengagement")
	}

	// ── Priority 2: Critical dopamine low ──────────────────────────────────────
	if state.DopamineScore < 20 {
		bestAction.Action = "recommend_item"
		bestAction.Channel = "in_app"
		bestAction.Confidence = 0.85
		bestAction.Reason = "dopamine_critical_low"
		bestAction.Payload = `{"sort": "ending_soon", "max_price": 500, "inject_cheap": true}`
		sources = append(sources, "dopamine")
	}

	// ── Priority 3: Active user with high dopamine ──────────────────────────────
	if state.Segment == "active" && state.DopamineScore > 70 && bestAction.Action == "" {
		bestAction.Action = "show_live_auction"
		bestAction.Channel = "in_app"
		bestAction.Confidence = 0.8
		bestAction.Reason = "active_high_dopamine"
		bestAction.Payload = `{"filter": "trending", "suggest_boost": true}`
		sources = append(sources, "dopamine", "user_state")
	}

	// ── Priority 4: Warm user with recent bids ──────────────────────────────────
	if state.Segment == "warm" && state.BidsCount > 0 && bestAction.Action == "" {
		bestAction.Action = "send_notification"
		bestAction.Channel = "push"
		bestAction.Confidence = 0.7
		bestAction.Reason = "warm_with_bids"
		bestAction.Payload = `{"type": "reminder", "message": "Auction ending soon!"}`
		sources = append(sources, "reengagement", "user_state")
	}

	// ── Priority 5: Re-engagement plan has actions ──────────────────────────────
	if len(reengagementPlan.Actions) > 0 && bestAction.Action == "" {
		first := reengagementPlan.Actions[0]
		bestAction.Action = "send_notification"
		bestAction.Channel = first.Channel
		bestAction.Confidence = 0.6
		bestAction.Reason = fmt.Sprintf("reengagement_%s", first.Type)
		bestAction.Payload = first.Payload
		sources = append(sources, "reengagement")
	}

	// ── Priority 6: Dopamine action ──────────────────────────────────────────────
	if dopamineAction.Type != "balanced" && bestAction.Action == "" {
		switch dopamineAction.Type {
		case "show_easy_wins", "inject_cheap":
			bestAction.Action = "recommend_item"
			bestAction.Channel = "in_app"
		case "show_premium", "suggest_vip":
			bestAction.Action = "show_live_auction"
			bestAction.Channel = "in_app"
		}
		bestAction.Confidence = 0.5
		bestAction.Reason = dopamineAction.Reason
		bestAction.Payload = dopamineAction.Payload
		sources = append(sources, "dopamine")
	}

	// ── Default: do nothing ──────────────────────────────────────────────────────
	if bestAction.Action == "" {
		bestAction.Action = "do_nothing"
		bestAction.Confidence = 1.0
		bestAction.Reason = "no_action_needed"
		bestAction.Payload = "{}"
	}

	bestAction.Sources = sources

	// Log the decision
	db.Create(&DecisionLog{
		UserID:     userID,
		Action:     bestAction.Action,
		Confidence: bestAction.Confidence,
		Reason:     bestAction.Reason,
		Sources:    fmt.Sprintf("%v", sources),
	})

	return bestAction
}

// RecordDecisionOutcome updates a decision with its outcome.
func RecordDecisionOutcome(db *gorm.DB, decisionID uuid.UUID, outcome string) {
	db.Model(&DecisionLog{}).Where("id = ?", decisionID).
		Update("outcome", outcome)
}

// ── Decision Metrics ────────────────────────────────────────────────────────────────

type DecisionMetrics struct {
	TotalDecisions int64            `json:"total_decisions"`
	ActionCounts   map[string]int64 `json:"action_counts"`
	SuccessRate    float64          `json:"success_rate"`
	AvgConfidence  float64          `json:"avg_confidence"`
}

func GetDecisionMetrics(db *gorm.DB) *DecisionMetrics {
	var total int64
	db.Model(&DecisionLog{}).Count(&total)

	var success int64
	db.Model(&DecisionLog{}).Where("outcome = ?", "success").Count(&success)

	successRate := 0.0
	if total > 0 {
		successRate = float64(success) / float64(total)
	}

	var avgConf struct{ Avg float64 }
	db.Model(&DecisionLog{}).Select("COALESCE(AVG(confidence), 0) as avg").Scan(&avgConf)

	actionCounts := map[string]int64{}
	var actResults []struct {
		Action string `json:"action"`
		Count  int64  `json:"count"`
	}
	db.Model(&DecisionLog{}).Select("action, COUNT(*) as count").Group("action").Scan(&actResults)
	for _, r := range actResults {
		actionCounts[r.Action] = r.Count
	}

	return &DecisionMetrics{
		TotalDecisions: total,
		ActionCounts:   actionCounts,
		SuccessRate:    successRate,
		AvgConfidence:  avgConf.Avg,
	}
}

// ── Helper ──────────────────────────────────────────────────────────────────────────

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for _, s := range ss[1:] {
		result += sep + s
	}
	return result
}
