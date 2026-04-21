package pricing

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── State Builder ────────────────────────────────────────────────────────────────

// BuildRLState constructs the full RL state from pricing context + session history.
func BuildRLState(db *gorm.DB, ctx *PricingContext) *RLState {
	state := &RLState{
		UserTrust:        ctx.TrustScore,
		CancelRate:       ctx.CancellationRate,
		InsuranceBuyRate: ctx.InsuranceBuyRate,
		RiskScore:        1.0 - ctx.TrustScore/100.0, // invert trust → risk
		AccountAgeDays:   ctx.AccountAgeDays,
		OrderValueCents:  ctx.OrderPriceCents,
		DeliveryRisk:     ctx.DeliveryRiskScore,
		Category:         ctx.Category,
		UrgencyScore:     ctx.UrgencyScore,
		TimeOfDay:        ctx.TimeOfDay,
		LiveDemand:       ctx.LiveDemand,
	}

	// Load session history
	session := loadOrCreateSession(db, ctx.UserID, ctx.OrderID)
	if session != nil {
		state.SessionStep = session.CurrentStep
		state.RefusalCount = session.RefusalCount
		state.LastAccepted = session.RefusalCount == 0 && session.CurrentStep > 0

		// Parse previous offers
		if session.PreviousOffers != "" && session.PreviousOffers != "[]" {
			var offers []float64
			if err := json.Unmarshal([]byte(session.PreviousOffers), &offers); err == nil {
				state.PreviousOffers = offers
			}
		}
	}

	return state
}

// loadOrCreateSession gets or creates an RL session for a user+order.
func loadOrCreateSession(db *gorm.DB, userID, orderID uuid.UUID) *RLSession {
	var session RLSession
	if err := db.Where("user_id = ? AND order_id = ?", userID, orderID).First(&session).Error; err != nil {
		// Create new session
		episodeID := fmt.Sprintf("ep_%s_%d", userID.String()[:8], time.Now().UnixNano())
		session = RLSession{
			UserID:         userID,
			OrderID:        orderID,
			EpisodeID:      episodeID,
			CurrentStep:    0,
			PreviousOffers: "[]",
			RefusalCount:   0,
			LastActionIdx:  -1,
		}
		db.Create(&session)
	}
	return &session
}

// UpdateSessionAfterAction updates the session after an action is taken.
func UpdateSessionAfterAction(db *gorm.DB, userID, orderID uuid.UUID, actionIdx int, pricePercent float64, didBuy bool) {
	session := loadOrCreateSession(db, userID, orderID)
	if session == nil {
		return
	}

	// Append to previous offers
	var offers []float64
	if session.PreviousOffers != "" && session.PreviousOffers != "[]" {
		json.Unmarshal([]byte(session.PreviousOffers), &offers)
	}
	offers = append(offers, pricePercent)
	offersJSON, _ := json.Marshal(offers)

	updates := map[string]interface{}{
		"current_step":    session.CurrentStep + 1,
		"previous_offers": string(offersJSON),
		"last_action_idx": actionIdx,
	}

	if !didBuy {
		updates["refusal_count"] = session.RefusalCount + 1
	}

	if didBuy || session.CurrentStep+1 >= 3 {
		updates["is_complete"] = true
	}

	db.Model(&RLSession{}).Where("id = ?", session.ID).Updates(updates)
}

// CompleteSession marks a session as complete and computes total reward.
func CompleteSession(db *gorm.DB, userID, orderID uuid.UUID, totalReward float64, didChurn bool) {
	session := loadOrCreateSession(db, userID, orderID)
	if session == nil {
		return
	}

	updates := map[string]interface{}{
		"is_complete":  true,
		"total_reward": totalReward,
	}
	db.Model(&RLSession{}).Where("id = ?", session.ID).Updates(updates)
}

// GetSessionCount returns how many active sessions a user has.
func GetSessionCount(db *gorm.DB, userID uuid.UUID) int64 {
	var count int64
	db.Model(&RLSession{}).Where("user_id = ? AND is_complete = ?", userID, false).Count(&count)
	return count
}

// Ensure fmt is used
var _ = fmt.Sprintf
