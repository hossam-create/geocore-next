package livestream

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/geocore-next/backend/internal/reputation"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// boostSuggestThrottle — minimum interval between boost suggestions per session.
const boostSuggestThrottle = 1 * time.Hour

// ════════════════════════════════════════════════════════════════════════════
// Sprint 11.5: Behavioral Revenue Engine
//
// Ties together the three existing layers (Behavior, Money, Safety) by making
// them actively feed each other:
//
//   FOMO urgency  →  pricing bonus        (urgency monetization)
//   Nudges        →  paid CTAs            (monetized nudges)
//   Hot items     →  boost suggestions    (seller upsell loop)
//   Price/trust   →  bidder quality gate  (fake-bid filter)
//   Funnel drops  →  auto remedies        (self-healing UX)
// ════════════════════════════════════════════════════════════════════════════

// ── Feature Flags ──────────────────────────────────────────────────────────

func IsBehavioralEngineEnabled() bool {
	v := os.Getenv("ENABLE_BEHAVIORAL_ENGINE")
	if v == "" {
		return true
	}
	return v != "false" && v != "0"
}

func IsMonetizedNudgesEnabled() bool {
	v := os.Getenv("ENABLE_MONETIZED_NUDGES")
	if v == "" {
		return true
	}
	return v != "false" && v != "0"
}

func IsBidderQualityGateEnabled() bool {
	v := os.Getenv("ENABLE_BIDDER_QUALITY_GATE")
	if v == "" {
		return true
	}
	return v != "false" && v != "0"
}

// ════════════════════════════════════════════════════════════════════════════
// 1. Urgency-Based Fee Bonus (FOMO → Pricing)
// ════════════════════════════════════════════════════════════════════════════

// UrgencyFeeBonus returns the commission % bonus derived purely from the
// current urgency state. Complementary to the surge bonus (which only triggers
// at ≥10 bids/10s) — urgency includes active-bidder count as well.
//
//	NORMAL   → 0%
//	HOT      → +2%
//	VERY_HOT → +4%
func UrgencyFeeBonus(urgency UrgencyState) float64 {
	if !IsBehavioralEngineEnabled() {
		return 0
	}
	switch urgency {
	case UrgencyVeryHot:
		return 4.0
	case UrgencyHot:
		return 2.0
	default:
		return 0
	}
}

// ════════════════════════════════════════════════════════════════════════════
// 2. Monetized Nudges (Nudges → Revenue)
// ════════════════════════════════════════════════════════════════════════════

// SuggestedActionFor maps a nudge code to a paid call-to-action.
// Returns (action, label). Empty strings mean no CTA.
func SuggestedActionFor(code NudgeCode) (string, string) {
	if !IsMonetizedNudgesEnabled() {
		return "", ""
	}
	switch code {
	case NudgeOutbid:
		return "quick_bid", "Quick bid +50 EGP"
	case NudgeItemAlmostEnd:
		return "buy_now", "Buy now — secure it"
	case NudgeBuyNowClose:
		return "buy_now", "Buy now before someone else"
	case NudgeNewHotItem:
		return "quick_bid", "Jump in — place first bid"
	case NudgeWatcherNotBidding:
		return "quick_bid", "Place a quick bid"
	default:
		return "", ""
	}
}

// ════════════════════════════════════════════════════════════════════════════
// 3. Smart Boost Injection (Hot items → seller upsell)
// ════════════════════════════════════════════════════════════════════════════

// SuggestBoostToHost pushes a host-only toast urging the seller to buy a boost
// when their item is trending but no active boost is applied.
//
// Throttled: one suggestion per session per hour (redis key if available).
func (h *LiveAuctionHandler) SuggestBoostToHost(sess *Session, urgency UrgencyState) {
	if !IsBehavioralEngineEnabled() || !IsLiveBoostEnabled() {
		return
	}
	// Only suggest for HOT/VERY_HOT items without an active boost
	if urgency == UrgencyNormal || sess.BoostTier != "" {
		return
	}
	// Redis throttle — 1h per session
	if h.rdb != nil {
		key := fmt.Sprintf("live:boost_suggest:%s", sess.ID)
		ok, err := h.rdb.SetNX(context.Background(), key, "1", boostSuggestThrottle).Result()
		if err != nil || !ok {
			return
		}
	}
	h.broadcastLiveEvent(sess.ID, LiveEvent{
		Event:           EventToast,
		SessionID:       sess.ID.String(),
		TargetUserID:    sess.HostID.String(),
		Message:         "🔥 Your item is trending — boost now to dominate the feed!",
		Icon:            "🔥",
		SuggestedAction: "boost",
		ActionLabel:     "Boost session",
	})
}

// ════════════════════════════════════════════════════════════════════════════
// 4. Bidder Quality Filter
// ════════════════════════════════════════════════════════════════════════════

const (
	bidderMinTrustScore   = 40.0      // below this → blocked
	highValueBidThreshold = 20_000_00 // 20,000 EGP in cents → mandatory deposit
)

// BidderQualityResult is returned from CheckBidderQuality.
type BidderQualityResult struct {
	Allowed        bool    `json:"allowed"`
	RequireDeposit bool    `json:"require_deposit"`
	ReasonCode     string  `json:"reason_code,omitempty"`
	UserTrustScore float64 `json:"user_trust_score,omitempty"`
}

// CheckBidderQuality gates a bid attempt based on:
//   - user's trust score (blocked if < 40)
//   - bid amount (deposit required if ≥ 20,000 EGP)
//
// Returns (allowed, require_deposit, reason_code).
func CheckBidderQuality(db *gorm.DB, userID uuid.UUID, bidAmountCents int64) BidderQualityResult {
	if !IsBidderQualityGateEnabled() {
		return BidderQualityResult{Allowed: true}
	}
	score := reputation.GetOverallScore(db, userID)
	if score < bidderMinTrustScore {
		return BidderQualityResult{
			Allowed:        false,
			ReasonCode:     "low_trust_score",
			UserTrustScore: score,
		}
	}
	return BidderQualityResult{
		Allowed:        true,
		RequireDeposit: bidAmountCents >= highValueBidThreshold,
		UserTrustScore: score,
	}
}

// ════════════════════════════════════════════════════════════════════════════
// 5. Funnel Auto-Optimization
// ════════════════════════════════════════════════════════════════════════════

// FunnelOptimization represents an automated remedy suggested by the analyzer.
type FunnelOptimization struct {
	SessionID   uuid.UUID `json:"session_id"`
	Problem     string    `json:"problem"`
	Action      string    `json:"action"`
	DropoffRate float64   `json:"dropoff_rate"`
}

// AnalyzeFunnelDropoff computes the drop-off rate between two conversion stages
// for a session. Returns dropoffRate in [0, 1].
//
// dropoff = 1 - (countTo / countFrom)   ← 0 if countFrom == 0
func AnalyzeFunnelDropoff(db *gorm.DB, sessionID uuid.UUID, from, to ConversionStage) float64 {
	var fromCount, toCount int64
	db.Model(&LiveConversionEvent{}).Where("session_id = ? AND stage = ?", sessionID, from).Count(&fromCount)
	db.Model(&LiveConversionEvent{}).Where("session_id = ? AND stage = ?", sessionID, to).Count(&toCount)
	if fromCount == 0 {
		return 0
	}
	ratio := float64(toCount) / float64(fromCount)
	if ratio > 1 {
		ratio = 1
	}
	return 1.0 - ratio
}

// AutoOptimizeSession inspects a session's funnel and returns a set of
// optimization actions (broadcasts the suggestions as toasts).
//
// Thresholds:
//   - click_bid → place_bid dropoff > 40%  →  "enable_quick_bid_prompt"
//   - view → click_bid dropoff > 70%       →  "suggest_boost"
//   - place_bid → win dropoff > 80%        →  "suggest_buy_now_hint"
func (h *LiveAuctionHandler) AutoOptimizeSession(sessionID uuid.UUID) []FunnelOptimization {
	if !IsBehavioralEngineEnabled() {
		return nil
	}
	out := make([]FunnelOptimization, 0, 3)

	if r := AnalyzeFunnelDropoff(h.db, sessionID, StageClickBid, StagePlaceBid); r > 0.40 {
		out = append(out, FunnelOptimization{
			SessionID: sessionID, Problem: "click_to_bid_dropoff",
			Action: "enable_quick_bid_prompt", DropoffRate: r,
		})
		h.BroadcastToast(sessionID, "⚡ Tap ⚡Quick Bid to place your bid in one tap", "⚡")
	}
	if r := AnalyzeFunnelDropoff(h.db, sessionID, StageView, StageClickBid); r > 0.70 {
		out = append(out, FunnelOptimization{
			SessionID: sessionID, Problem: "view_to_click_dropoff",
			Action: "suggest_boost", DropoffRate: r,
		})
		// Fetch seller and suggest boost
		var sess Session
		if err := h.db.Where("id = ?", sessionID).First(&sess).Error; err == nil {
			h.SuggestBoostToHost(&sess, UrgencyHot) // simulate hot to trigger suggestion
		}
	}
	if r := AnalyzeFunnelDropoff(h.db, sessionID, StagePlaceBid, StageWin); r > 0.80 {
		out = append(out, FunnelOptimization{
			SessionID: sessionID, Problem: "bid_to_win_dropoff",
			Action: "suggest_buy_now_hint", DropoffRate: r,
		})
		h.BroadcastToast(sessionID, "💡 Skip the war — Buy Now ends it instantly", "💡")
	}
	return out
}

// AdminAutoOptimize — GET /admin/live/:id/auto-optimize
// Runs funnel dropoff analysis and broadcasts remedies; returns the list.
func (h *LiveAuctionHandler) AdminAutoOptimize(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid session id")
		return
	}
	results := h.AutoOptimizeSession(sessionID)
	response.OK(c, gin.H{"session_id": sessionID, "optimizations": results})
}
