package livestream

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ════════════════════════════════════════════════════════════════════════════
// Sprint 14: AI-Powered Live Seller Assistant
//
// Rule-based (no ML) real-time decision engine. Consumes signals from:
//   - FOMO Engine       (urgency, bid velocity)
//   - Conversion Engine (funnel dropoff)
//   - Monetization      (boost state, premium mode, entry fee)
//   - Wallet            (read-only context)
//
// Emits:
//   - WebSocket event `live_ai_suggestion`
//   - Persisted LiveAIEvent row (learning layer)
//
// Philosophy: non-intrusive, human-in-the-loop, feature-flagged, additive.
// ════════════════════════════════════════════════════════════════════════════

// ── Feature Flags ──────────────────────────────────────────────────────────

func IsAIAssistantEnabled() bool {
	v := os.Getenv("ENABLE_AI_ASSISTANT")
	if v == "" {
		return true
	}
	return v != "false" && v != "0"
}

func IsAIMonetizationHintsEnabled() bool {
	v := os.Getenv("ENABLE_AI_MONETIZATION_HINTS")
	if v == "" {
		return true
	}
	return v != "false" && v != "0"
}

func IsAIDropPreventionEnabled() bool {
	v := os.Getenv("ENABLE_AI_DROP_PREVENTION")
	if v == "" {
		return true
	}
	return v != "false" && v != "0"
}

// ── Constants ──────────────────────────────────────────────────────────────

const (
	aiSuggestThrottle = 90 * time.Second // max 1 of same suggestion per session+item per 90s
	aiNoBidsDropoff   = 20 * time.Second // no-bid-for trigger
	aiViewerDropPct   = 0.30             // 30% viewer drop threshold
)

// ── Suggestion Types ───────────────────────────────────────────────────────

const (
	AISuggestPriceDrop         = "price_drop"
	AISuggestEnableBuyNow      = "enable_buy_now"
	AISuggestExtendTimer       = "extend_timer"
	AISuggestIncreaseIncrement = "increase_increment"
	AISuggestPushBuyNow        = "push_buy_now"
	AISuggestPinItem           = "pin_item"
	AISuggestBoostSession      = "boost_session"
	AISuggestEnablePremium     = "enable_premium"
	AISuggestSetEntryFeeNext   = "set_entry_fee_next"
	AISuggestDropoffRemedy     = "dropoff_remedy"
)

// ── LiveAIEvent Model (Learning Layer) ─────────────────────────────────────

// LiveAIEvent is the audit/learning row for every suggestion the AI surfaces.
type LiveAIEvent struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SessionID      uuid.UUID  `gorm:"type:uuid;not null;index"                        json:"session_id"`
	ItemID         *uuid.UUID `gorm:"type:uuid;index"                                 json:"item_id,omitempty"`
	SuggestionType string     `gorm:"size:50;not null;index"                          json:"suggestion_type"`
	Message        string     `gorm:"type:text;not null"                              json:"message"`
	Confidence     float64    `gorm:"type:numeric(4,3);not null;default:0.5"          json:"confidence"`
	Accepted       bool       `gorm:"not null;default:false;index"                    json:"accepted"`
	AcceptedAt     *time.Time `                                                       json:"accepted_at,omitempty"`
	ImpactScore    float64    `gorm:"type:numeric(6,2);not null;default:0"            json:"impact_score"` // filled post-hoc
	CreatedAt      time.Time  `gorm:"not null;index"                                  json:"created_at"`
}

func (LiveAIEvent) TableName() string { return "live_ai_events" }

// ── Data Structures ────────────────────────────────────────────────────────

// LiveInsights is the real-time snapshot exposed to the UI / internal decision loop.
type LiveInsights struct {
	SessionID       uuid.UUID    `json:"session_id"`
	ItemID          uuid.UUID    `json:"item_id"`
	UrgencyLevel    UrgencyState `json:"urgency_level"`
	BidVelocity     int          `json:"bid_velocity"` // bids in last 10s
	BidsLast30s     int          `json:"bids_last_30s"`
	ActiveBidders   int          `json:"active_bidders"`
	ViewerCount     int          `json:"viewer_count"`
	SecondsLeft     int          `json:"seconds_left"`
	BuyNowProgress  float64      `json:"buy_now_progress"`
	DropoffRisk     float64      `json:"dropoff_risk"`     // 0.0–1.0
	OptimalAction   string       `json:"optimal_action"`   // suggestion_type
	ConfidenceScore float64      `json:"confidence_score"` // 0.0–1.0
}

// SellerSuggestion is what the AI emits to sellers/viewers.
type SellerSuggestion struct {
	Type       string  `json:"type"`
	Message    string  `json:"message"`
	Action     string  `json:"action"` // CTA code (same domain as SuggestedAction)
	Confidence float64 `json:"confidence"`
}

// SuggestionContext is the input to GenerateSellerSuggestion.
// All fields are optional — rules guard against missing data.
type SuggestionContext struct {
	Session             *Session
	Item                *LiveItem
	Insights            LiveInsights
	SecondsSinceLastBid int
	BoostActive         bool
}

// ════════════════════════════════════════════════════════════════════════════
// 1. Real-Time Decision Engine — GetLiveInsights
// ════════════════════════════════════════════════════════════════════════════

// GetLiveInsights returns the current decision-engine snapshot for an item.
// Safe to call from any handler — read-only.
func (h *LiveAuctionHandler) GetLiveInsights(sessionID, itemID uuid.UUID) LiveInsights {
	ins := LiveInsights{SessionID: sessionID, ItemID: itemID}

	// Bid velocity (FOMO engine)
	bids10 := countBidsInWindow(h.rdb, itemID, fomoWindow10s)
	bids30 := countBidsInWindow(h.rdb, itemID, fomoWindow30s)
	ins.BidVelocity = bids10
	ins.BidsLast30s = bids30
	ins.UrgencyLevel = computeUrgency(bids10)
	ins.ActiveBidders = countActiveBidders(h.db, itemID)

	// Item state
	var item LiveItem
	if err := h.db.Where("id = ?", itemID).First(&item).Error; err == nil {
		if item.EndsAt != nil {
			sec := int(time.Until(*item.EndsAt).Seconds())
			if sec < 0 {
				sec = 0
			}
			ins.SecondsLeft = sec
		}
		ins.BuyNowProgress = ComputeBuyNowProgress(item.CurrentBidCents, item.BuyNowPriceCents)
	}

	// Viewer count
	ins.ViewerCount = h.getViewerCount(sessionID)

	// Dropoff risk — simple heuristic:
	//   +0.4 if no bids in last 30s and item is active
	//   +0.3 if viewer count < 5 and item is active
	//   +0.3 if urgency == NORMAL and seconds_left < 30
	ins.DropoffRisk = h.computeDropoffRisk(ins)

	// Optimal action — run the rule engine (stateless)
	action, conf := h.selectOptimalAction(ins)
	ins.OptimalAction = action
	ins.ConfidenceScore = conf

	return ins
}

// computeDropoffRisk returns a heuristic 0.0–1.0 risk score.
func (h *LiveAuctionHandler) computeDropoffRisk(ins LiveInsights) float64 {
	risk := 0.0
	if ins.BidsLast30s == 0 && ins.SecondsLeft > 0 {
		risk += 0.4
	}
	if ins.ViewerCount < 5 && ins.SecondsLeft > 0 {
		risk += 0.3
	}
	if ins.UrgencyLevel == UrgencyNormal && ins.SecondsLeft > 0 && ins.SecondsLeft < 30 {
		risk += 0.3
	}
	if risk > 1.0 {
		risk = 1.0
	}
	return risk
}

// selectOptimalAction applies the rule ladder and returns (action, confidence).
func (h *LiveAuctionHandler) selectOptimalAction(ins LiveInsights) (string, float64) {
	switch {
	case ins.BuyNowProgress >= 0.90:
		return AISuggestPushBuyNow, 0.95
	case ins.UrgencyLevel == UrgencyVeryHot && ins.ActiveBidders >= 3:
		return AISuggestExtendTimer, 0.85
	case ins.UrgencyLevel == UrgencyHot && ins.ActiveBidders >= 2:
		return AISuggestIncreaseIncrement, 0.75
	case ins.BidsLast30s == 0 && ins.SecondsLeft > 30:
		return AISuggestEnableBuyNow, 0.70
	case ins.BidsLast30s == 0 && ins.ViewerCount < 10:
		return AISuggestPinItem, 0.65
	case ins.DropoffRisk >= 0.6:
		return AISuggestDropoffRemedy, 0.70
	default:
		return "", 0
	}
}

// ════════════════════════════════════════════════════════════════════════════
// 2. Smart Seller Suggestions — GenerateSellerSuggestion
// ════════════════════════════════════════════════════════════════════════════

// GenerateSellerSuggestion returns the strongest actionable suggestion
// given a context snapshot. Returns nil if no suggestion applies.
func GenerateSellerSuggestion(c SuggestionContext) *SellerSuggestion {
	ins := c.Insights

	// ── Rule ladder (highest confidence first) ─────────────────────────────

	if ins.BuyNowProgress >= 0.90 && c.Item != nil && c.Item.BuyNowPriceCents != nil {
		return &SellerSuggestion{
			Type:       AISuggestPushBuyNow,
			Message:    "🔥 Bidding is at 90%+ of buy-now — push buyers to secure it instantly",
			Action:     "broadcast_buy_now_cta",
			Confidence: 0.95,
		}
	}

	if ins.UrgencyLevel == UrgencyVeryHot && ins.ActiveBidders >= 3 && ins.SecondsLeft > 0 && ins.SecondsLeft < 60 {
		return &SellerSuggestion{
			Type:       AISuggestExtendTimer,
			Message:    "⏰ Huge demand — extend the timer to let more bids pile in",
			Action:     "extend_timer",
			Confidence: 0.85,
		}
	}

	if ins.UrgencyLevel == UrgencyHot && ins.ActiveBidders >= 2 {
		return &SellerSuggestion{
			Type:       AISuggestIncreaseIncrement,
			Message:    "📈 Hot item with multiple bidders — raise the minimum increment",
			Action:     "increase_increment",
			Confidence: 0.75,
		}
	}

	if c.SecondsSinceLastBid >= int(aiNoBidsDropoff/time.Second) && ins.SecondsLeft > 30 {
		if c.Item != nil && c.Item.BuyNowPriceCents == nil {
			return &SellerSuggestion{
				Type:       AISuggestEnableBuyNow,
				Message:    "💡 No bids for 20s — enable Buy Now to close the sale",
				Action:     "enable_buy_now",
				Confidence: 0.70,
			}
		}
		return &SellerSuggestion{
			Type:       AISuggestPriceDrop,
			Message:    "💡 No bids for 20s — consider lowering the reserve",
			Action:     "lower_reserve",
			Confidence: 0.65,
		}
	}

	if ins.BidsLast30s == 0 && ins.ViewerCount < 10 && c.Item != nil && !c.Item.IsPinned {
		return &SellerSuggestion{
			Type:       AISuggestPinItem,
			Message:    "📌 Low engagement — pin this item to keep it on-screen",
			Action:     "pin_item",
			Confidence: 0.65,
		}
	}

	return nil
}

// ════════════════════════════════════════════════════════════════════════════
// 3. Auto-Monetization Triggers — GenerateRevenueOpportunity
// ════════════════════════════════════════════════════════════════════════════

// GenerateRevenueOpportunity returns a monetization-focused suggestion, or nil.
func GenerateRevenueOpportunity(c SuggestionContext) *SellerSuggestion {
	if !IsAIMonetizationHintsEnabled() {
		return nil
	}
	ins := c.Insights

	// HOT + no boost → suggest boost
	if ins.UrgencyLevel != UrgencyNormal && !c.BoostActive && c.Session != nil && c.Session.BoostTier == "" {
		return &SellerSuggestion{
			Type:       AISuggestBoostSession,
			Message:    "🚀 Your session is trending — boost it now to capture the momentum",
			Action:     "boost_session",
			Confidence: 0.80,
		}
	}

	// High traffic + high-value item → premium mode
	if c.Session != nil && !c.Session.IsPremium && ins.ViewerCount >= 50 &&
		c.Item != nil && c.Item.CurrentBidCents >= 10_000_00 { // 10k EGP
		return &SellerSuggestion{
			Type:       AISuggestEnablePremium,
			Message:    "💎 High-value item attracting a crowd — enable Premium Mode to stand out",
			Action:     "enable_premium",
			Confidence: 0.75,
		}
	}

	// Many bidders & high competition → entry fee for next session
	if ins.ActiveBidders >= 8 && ins.UrgencyLevel != UrgencyNormal &&
		c.Session != nil && c.Session.EntryFeeCents == 0 {
		return &SellerSuggestion{
			Type:       AISuggestSetEntryFeeNext,
			Message:    "🎯 Competitive auction — set an entry fee for your next session to monetize the crowd",
			Action:     "set_entry_fee_next",
			Confidence: 0.70,
		}
	}

	return nil
}

// ════════════════════════════════════════════════════════════════════════════
// 4. Drop-off Prevention
// ════════════════════════════════════════════════════════════════════════════

// TrackViewerSnapshot stores the current viewer count in Redis so future
// calls can compare and detect a drop.
func (h *LiveAuctionHandler) TrackViewerSnapshot(sessionID uuid.UUID, viewers int) {
	if h.rdb == nil {
		return
	}
	key := "live:ai:viewers_peak:" + sessionID.String()
	// Track peak via SETNX + compare
	ctx := context.Background()
	existing, err := h.rdb.Get(ctx, key).Int()
	if err != nil || viewers > existing {
		h.rdb.Set(ctx, key, viewers, 2*time.Hour)
	}
}

// DetectDropoff returns true if viewers dropped by >= aiViewerDropPct since peak.
func (h *LiveAuctionHandler) DetectDropoff(sessionID uuid.UUID, currentViewers int) bool {
	if !IsAIDropPreventionEnabled() || h.rdb == nil {
		return false
	}
	key := "live:ai:viewers_peak:" + sessionID.String()
	peak, err := h.rdb.Get(context.Background(), key).Int()
	if err != nil || peak < 10 { // ignore noise under 10 viewers
		return false
	}
	dropRatio := 1.0 - (float64(currentViewers) / float64(peak))
	return dropRatio >= aiViewerDropPct
}

// ════════════════════════════════════════════════════════════════════════════
// 5. Main Evaluation Entry Point — wired into PlaceBid + publishUrgencyUpdate
// ════════════════════════════════════════════════════════════════════════════

// EvaluateAndSuggest runs the full decision engine for an item and, if a
// high-confidence suggestion is produced, broadcasts it via WebSocket and
// persists a LiveAIEvent row. Throttled per (session, item, type).
//
// Non-blocking, safe to call after every key event (bid, urgency change).
func (h *LiveAuctionHandler) EvaluateAndSuggest(sessionID, itemID uuid.UUID) {
	if !IsAIAssistantEnabled() {
		return
	}

	// Build context
	ins := h.GetLiveInsights(sessionID, itemID)

	var sess Session
	if err := h.db.Where("id = ?", sessionID).First(&sess).Error; err != nil {
		return
	}
	var item LiveItem
	if err := h.db.Where("id = ?", itemID).First(&item).Error; err != nil {
		return
	}

	// Seconds since last bid
	secSinceBid := h.secondsSinceLastBid(itemID)

	ctx := SuggestionContext{
		Session:             &sess,
		Item:                &item,
		Insights:            ins,
		SecondsSinceLastBid: secSinceBid,
		BoostActive:         sess.BoostTier != "",
	}

	// Prefer seller suggestion; if none, try revenue opportunity.
	sugg := GenerateSellerSuggestion(ctx)
	if sugg == nil {
		sugg = GenerateRevenueOpportunity(ctx)
	}
	if sugg == nil {
		return
	}

	// Throttle identical suggestions per session+item+type
	if !h.aiThrottleAllow(sessionID, itemID, sugg.Type) {
		return
	}

	// Persist + broadcast
	row := LiveAIEvent{
		SessionID:      sessionID,
		ItemID:         &itemID,
		SuggestionType: sugg.Type,
		Message:        sugg.Message,
		Confidence:     sugg.Confidence,
		CreatedAt:      time.Now(),
	}
	if err := h.db.Create(&row).Error; err != nil {
		return
	}

	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:           EventLiveAISuggestion,
		SessionID:       sessionID.String(),
		ItemID:          itemID.String(),
		TargetUserID:    sess.HostID.String(), // seller-facing
		Message:         sugg.Message,
		SuggestionID:    row.ID.String(),
		SuggestionType:  sugg.Type,
		SuggestedAction: sugg.Action,
		Confidence:      sugg.Confidence,
	})
}

// secondsSinceLastBid returns time since the latest bid, or very large if none.
func (h *LiveAuctionHandler) secondsSinceLastBid(itemID uuid.UUID) int {
	var bid LiveBid
	if err := h.db.Where("item_id = ?", itemID).
		Order("created_at DESC").Limit(1).First(&bid).Error; err != nil {
		return 9999
	}
	return int(time.Since(bid.CreatedAt).Seconds())
}

// aiThrottleAllow uses Redis SetNX to limit identical suggestions.
func (h *LiveAuctionHandler) aiThrottleAllow(sessionID, itemID uuid.UUID, sType string) bool {
	if h.rdb == nil {
		return true
	}
	key := fmt.Sprintf("live:ai:suggest:%s:%s:%s", sessionID, itemID, sType)
	ok, err := h.rdb.SetNX(context.Background(), key, "1", aiSuggestThrottle).Result()
	if err != nil {
		return true // fail open — don't block suggestions on Redis errors
	}
	return ok
}

// ════════════════════════════════════════════════════════════════════════════
// 6. HTTP Endpoints
// ════════════════════════════════════════════════════════════════════════════

// AIInsights — GET /livestream/:id/items/:itemId/ai-insights
// Seller-facing real-time snapshot.
func (h *LiveAuctionHandler) AIInsights(c *gin.Context) {
	if !IsAIAssistantEnabled() {
		response.BadRequest(c, "AI assistant is not enabled")
		return
	}
	sessionID, _ := uuid.Parse(c.Param("id"))
	itemID, _ := uuid.Parse(c.Param("itemId"))
	ins := h.GetLiveInsights(sessionID, itemID)
	response.OK(c, ins)
}

// AcceptAISuggestion — POST /livestream/:id/ai-suggestions/:suggestionId/accept
// Marks a suggestion as accepted (for the learning layer).
func (h *LiveAuctionHandler) AcceptAISuggestion(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))
	suggID, err := uuid.Parse(c.Param("suggestionId"))
	if err != nil {
		response.BadRequest(c, "invalid suggestion id")
		return
	}

	// Only the session host can accept suggestions
	var sess Session
	if err := h.db.Where("id = ? AND host_id = ?", sessionID, userID).First(&sess).Error; err != nil {
		response.Forbidden(c)
		return
	}

	now := time.Now()
	res := h.db.Model(&LiveAIEvent{}).
		Where("id = ? AND session_id = ?", suggID, sessionID).
		Updates(map[string]interface{}{
			"accepted":    true,
			"accepted_at": now,
		})
	if res.Error != nil || res.RowsAffected == 0 {
		response.NotFound(c, "suggestion")
		return
	}
	response.OK(c, gin.H{"accepted": true, "suggestion_id": suggID})
}

// AdminAIPerformance — GET /admin/live/ai-performance
// Aggregate analytics for the learning layer.
func (h *LiveAuctionHandler) AdminAIPerformance(c *gin.Context) {
	type topRow struct {
		SuggestionType string  `json:"suggestion_type"`
		Shown          int64   `json:"shown"`
		Accepted       int64   `json:"accepted"`
		AcceptanceRate float64 `json:"acceptance_rate"`
	}

	var total, accepted int64
	h.db.Model(&LiveAIEvent{}).Count(&total)
	h.db.Model(&LiveAIEvent{}).Where("accepted = ?", true).Count(&accepted)

	// Per-type breakdown
	var rows []struct {
		SuggestionType string
		Shown          int64
		Accepted       int64
	}
	h.db.Model(&LiveAIEvent{}).
		Select("suggestion_type, COUNT(*) AS shown, SUM(CASE WHEN accepted THEN 1 ELSE 0 END) AS accepted").
		Group("suggestion_type").Scan(&rows)

	top := make([]topRow, 0, len(rows))
	for _, r := range rows {
		rate := 0.0
		if r.Shown > 0 {
			rate = float64(r.Accepted) / float64(r.Shown)
		}
		top = append(top, topRow{
			SuggestionType: r.SuggestionType,
			Shown:          r.Shown,
			Accepted:       r.Accepted,
			AcceptanceRate: rate,
		})
	}

	// Revenue impact estimate — sum of impact_score on accepted rows
	var impact struct{ Total float64 }
	h.db.Model(&LiveAIEvent{}).
		Where("accepted = ?", true).
		Select("COALESCE(SUM(impact_score),0) AS total").Scan(&impact)

	accRate := 0.0
	if total > 0 {
		accRate = float64(accepted) / float64(total)
	}

	response.OK(c, gin.H{
		"suggestions_count":         total,
		"accepted_count":            accepted,
		"acceptance_rate":           accRate,
		"revenue_increase_estimate": impact.Total,
		"top_performing":            top,
	})
}
