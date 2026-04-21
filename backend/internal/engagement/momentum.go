package engagement

import (
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Session Momentum Engine ────────────────────────────────────────────────────────
//
// Tracks real-time session engagement quality and adapts feed intensity.
//
// High momentum → show decisive items (Buy Now, auctions ending soon)
// Low momentum → reduce pressure, show variety, suggest break/save

// UpdateMomentum recalculates session momentum from raw counters.
func UpdateMomentum(db *gorm.DB, userID uuid.UUID, sessionID string) (*SessionMomentum, error) {
	var m SessionMomentum
	if err := db.Where("session_id = ?", sessionID).First(&m).Error; err != nil {
		m = SessionMomentum{
			UserID:    userID,
			SessionID: sessionID,
		}
	}

	// ── Compute rates ────────────────────────────────────────────────────────
	if m.ViewsCount > 0 {
		m.ClickRate = float64(m.ClicksCount) / float64(m.ViewsCount)
		m.BidRate = float64(m.BidsCount) / float64(m.ViewsCount)
		m.ScrollVelocity = float64(m.ViewsCount) / math.Max(1, time.Since(m.CreatedAt).Minutes())
	}

	// ── Compute friction (exits + backs / total actions) ──────────────────────
	totalActions := m.ClicksCount + m.BidsCount + m.SavesCount + m.PurchasesCount
	if totalActions > 0 {
		m.Friction = float64(m.BacksCount) / float64(totalActions)
	}

	// ── Composite momentum score ──────────────────────────────────────────────
	// Weighted: click_rate (30%) + bid_rate (30%) - friction (20%) + scroll (20%)
	m.MomentumScore = 0.3*m.ClickRate + 0.3*m.BidRate - 0.2*m.Friction + 0.2*math.Min(m.ScrollVelocity/10.0, 1.0)
	m.MomentumScore = math.Max(0, math.Min(1, m.MomentumScore))

	// ── Determine feed intensity ──────────────────────────────────────────────
	config := loadEngagementConfig(db)
	switch {
	case m.MomentumScore >= config.MomentumHighThreshold:
		m.FeedIntensity = "high" // decisive items, urgency
	case m.MomentumScore <= config.MomentumLowThreshold:
		m.FeedIntensity = "low" // variety, save-for-later, gentle
	default:
		m.FeedIntensity = "balanced"
	}

	m.UpdatedAt = time.Now()
	db.Save(&m)

	return &m, nil
}

// RecordMomentumAction updates session counters for a specific action type.
func RecordMomentumAction(db *gorm.DB, sessionID string, action string) error {
	updates := map[string]interface{}{}
	switch action {
	case "view":
		updates["views_count"] = gorm.Expr("views_count + 1")
	case "click":
		updates["views_count"] = gorm.Expr("views_count + 1")
		updates["clicks_count"] = gorm.Expr("clicks_count + 1")
	case "bid":
		updates["bids_count"] = gorm.Expr("bids_count + 1")
	case "save":
		updates["saves_count"] = gorm.Expr("saves_count + 1")
	case "purchase":
		updates["purchases_count"] = gorm.Expr("purchases_count + 1")
	case "back":
		updates["backs_count"] = gorm.Expr("backs_count + 1")
	default:
		return nil
	}

	updates["updated_at"] = time.Now()
	return db.Model(&SessionMomentum{}).Where("session_id = ?", sessionID).Updates(updates).Error
}

// GetMomentum retrieves the current session momentum.
func GetMomentum(db *gorm.DB, sessionID string) (*SessionMomentum, error) {
	var m SessionMomentum
	if err := db.Where("session_id = ?", sessionID).First(&m).Error; err != nil {
		return &SessionMomentum{SessionID: sessionID, FeedIntensity: "balanced"}, nil
	}
	return &m, nil
}

// GetFeedRecommendation returns what type of content to show based on momentum.
type FeedRecommendation struct {
	FeedIntensity     string   `json:"feed_intensity"`
	ShowUrgencyItems  bool     `json:"show_urgency_items"`  // auctions ending, limited stock
	ShowBuyNow        bool     `json:"show_buy_now"`        // decisive CTAs
	ShowSaveForLater  bool     `json:"show_save_for_later"` // gentle save prompts
	ShowVariety       bool     `json:"show_variety"`        // diverse content
	ShowBreak         bool     `json:"show_break"`          // suggest taking a break
	ExplorationRatio  float64  `json:"exploration_ratio"`  // % of novel content
}

func GetFeedRecommendation(momentum *SessionMomentum, config EngagementConfig) *FeedRecommendation {
	rec := &FeedRecommendation{
		FeedIntensity:    momentum.FeedIntensity,
		ExplorationRatio: float64(config.ExplorationPercent) / 100.0,
	}

	switch momentum.FeedIntensity {
	case "high":
		rec.ShowUrgencyItems = true
		rec.ShowBuyNow = true
		rec.ExplorationRatio = 0.05 // less exploration when decisive
	case "low":
		rec.ShowSaveForLater = true
		rec.ShowVariety = true
		rec.ShowBreak = momentum.Friction > 0.3
		rec.ExplorationRatio = 0.2 // more exploration when browsing
	default: // balanced
		rec.ShowUrgencyItems = momentum.MomentumScore > 0.5
		rec.ShowSaveForLater = momentum.MomentumScore < 0.4
	}

	return rec
}
