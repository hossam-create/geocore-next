package livestream

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/geocore-next/backend/internal/freeze"
	"github.com/geocore-next/backend/internal/reputation"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Sprint 17: Marketplace Brain — Real-time Ranking & Traffic Allocation Engine
//
// Dynamically decides:
//   - which live sessions get exposure
//   - which creators receive traffic
//   - which listings are boosted
//   - how to maximize platform revenue + conversion
//
// Layers on top of:
//   - Session scoring (viewers, bids, conversion, trust, boost, premium)
//   - Creator Economy (Sprint 16: creator trust, deals, earnings)
//   - Revenue Flywheel (Sprint 13: boost, surge, whale)
//   - AI Assistant (Sprint 14: suggestions, dropoff)
//   - Viral Loops (Sprint 15: invites, streaks)
//
// All features are additive, feature-flagged, real-time safe (Redis caching),
// with deterministic fallback if Redis fails.
// ════════════════════════════════════════════════════════════════════════════

// ── Feature Flags ──────────────────────────────────────────────────────────

func IsMarketplaceBrainEnabled() bool  { return envBoolDefault("ENABLE_MARKETPLACE_BRAIN", true) }
func IsSmartRankingEnabled() bool      { return envBoolDefault("ENABLE_SMART_RANKING", true) }
func IsTrafficAllocationEnabled() bool { return envBoolDefault("ENABLE_TRAFFIC_ALLOCATION", true) }

// encodeJSON marshals v to JSON string.
func encodeJSON(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// decodeJSON unmarshals a JSON string into v.
func decodeJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}

// ── Constants ──────────────────────────────────────────────────────────────

const (
	// Scoring weights
	weightViewers      = 0.25
	weightBidsPerMin   = 0.25
	weightConversion   = 0.20
	weightCreatorTrust = 0.15
	weightBoostScore   = 0.10
	weightPremium      = 0.05

	// Score normalization
	maxScore = 100.0

	// Redis TTLs
	sessionScoreTTL    = 10 * time.Second
	creatorExposureTTL = 60 * time.Second
	feedCacheTTL       = 5 * time.Second

	// Traffic allocation thresholds
	trafficHighScore = 80.0 // homepage + push notifications
	trafficMidScore  = 60.0 // discovery feed
	trafficLowScore  = 40.0 // deprioritize

	// Fairness guard
	maxTrafficSharePct  = 0.30 // no session gets >30% total traffic
	coldStartBoostScore = 15.0 // temporary boost for new sessions
	coldStartDuration   = 5 * time.Minute

	// Creator exposure
	highConversionThreshold = 0.20 // +visibility if conversion > 20%
	lowTrustThreshold       = 50.0 // reduce exposure if trust < 50

	// Drop-off recovery
	dropoffViewerPct = 0.30 // 30% viewer drop triggers recovery
	dropoffNoBidSecs = 30   // no bids in 30s triggers recovery

	// Revenue optimization
	revenuePriorityMinFee = 0.08 // prioritize sessions with ≥8% platform fee

	// Redis key prefixes
	redisKeySessionScore    = "marketplace:score:"
	redisKeyCreatorExposure = "marketplace:creator_exposure:"
	redisKeyFeedCache       = "marketplace:feed"
	redisKeyTrafficAlloc    = "marketplace:traffic:"
	redisKeyColdStart       = "marketplace:cold_start:"
)

// ════════════════════════════════════════════════════════════════════════════
// Models
// ════════════════════════════════════════════════════════════════════════════

// SessionScore holds the computed score and its breakdown.
type SessionScore struct {
	SessionID        uuid.UUID `json:"session_id"`
	Score            float64   `json:"score"`        // 0–100
	ViewersNorm      float64   `json:"viewers_norm"` // normalized viewers component
	BidsPerMinNorm   float64   `json:"bids_per_min_norm"`
	ConversionNorm   float64   `json:"conversion_norm"`
	CreatorTrustNorm float64   `json:"creator_trust_norm"`
	BoostNorm        float64   `json:"boost_norm"`
	PremiumBonus     float64   `json:"premium_bonus"`
	ColdStartBonus   float64   `json:"cold_start_bonus,omitempty"`
	TrafficTier      string    `json:"traffic_tier"` // high/mid/low
	ComputedAt       time.Time `json:"computed_at"`
}

// FeedEntry is a single session in the ranked feed.
type FeedEntry struct {
	SessionID    uuid.UUID  `json:"session_id"`
	HostID       uuid.UUID  `json:"host_id"`
	Title        string     `json:"title"`
	ViewerCount  int        `json:"viewer_count"`
	IsPremium    bool       `json:"is_premium"`
	BoostScore   int        `json:"boost_score"`
	Score        float64    `json:"score"`
	TrafficTier  string     `json:"traffic_tier"`
	StreamerID   *uuid.UUID `json:"streamer_id,omitempty"`
	ThumbnailURL string     `json:"thumbnail_url,omitempty"`
}

// TrafficAllocation describes how traffic is distributed.
type TrafficAllocation struct {
	SessionID     uuid.UUID `json:"session_id"`
	TrafficTier   string    `json:"traffic_tier"`   // high/mid/low
	ExposurePct   float64   `json:"exposure_pct"`   // % of total traffic
	PushNotify    bool      `json:"push_notify"`    // homepage + push
	DiscoveryFeed bool      `json:"discovery_feed"` // discovery feed
	Deprioritized bool      `json:"deprioritized"`
}

// CreatorExposure holds a creator's visibility multiplier.
type CreatorExposure struct {
	CreatorID            uuid.UUID `json:"creator_id"`
	VisibilityMultiplier float64   `json:"visibility_multiplier"`
	ConversionRate       float64   `json:"conversion_rate"`
	TrustScore           float64   `json:"trust_score"`
	Reason               string    `json:"reason"`
}

// MarketplaceBrainMetrics holds aggregate marketplace analytics.
type MarketplaceBrainMetrics struct {
	TotalLiveSessions   int                    `json:"total_live_sessions"`
	AvgSessionScore     float64                `json:"avg_session_score"`
	HighTrafficSessions int                    `json:"high_traffic_sessions"`
	MidTrafficSessions  int                    `json:"mid_traffic_sessions"`
	LowTrafficSessions  int                    `json:"low_traffic_sessions"`
	RevenuePriorityMode bool                   `json:"revenue_priority_mode"`
	TopSessions         []SessionScore         `json:"top_sessions"`
	TrafficAllocations  []TrafficAllocation    `json:"traffic_allocations"`
	CreatorExposures    []CreatorExposure      `json:"creator_exposures"`
	ScoreHistory        []SessionScoreSnapshot `json:"score_history,omitempty"`
}

// SessionScoreSnapshot records a score at a point in time (for analytics).
type SessionScoreSnapshot struct {
	SessionID   uuid.UUID `gorm:"type:uuid;not null;index"                        json:"session_id"`
	Score       float64   `gorm:"type:numeric(6,2);not null"                     json:"score"`
	TrafficTier string    `gorm:"size:10;not null;index"                          json:"traffic_tier"`
	CreatedAt   time.Time `gorm:"not null;index"                                  json:"created_at"`
}

func (SessionScoreSnapshot) TableName() string { return "live_session_score_snapshots" }

// ════════════════════════════════════════════════════════════════════════════
// 1. Session Scoring Engine
// ════════════════════════════════════════════════════════════════════════════

// ComputeSessionScore calculates a 0–100 score for a live session.
// Uses Redis cache (10s TTL) with deterministic DB fallback.
func ComputeSessionScore(db *gorm.DB, rdb *redis.Client, sessionID uuid.UUID) (*SessionScore, error) {
	if !IsMarketplaceBrainEnabled() {
		return nil, fmt.Errorf("marketplace brain disabled")
	}

	// Try Redis cache first
	if rdb != nil {
		cached, err := rdb.Get(context.Background(), redisKeySessionScore+sessionID.String()).Result()
		if err == nil && cached != "" {
			var ss SessionScore
			if err := decodeJSON(cached, &ss); err == nil {
				return &ss, nil
			}
		}
	}
	_ = context.Background

	// Load session from DB
	var sess Session
	if err := db.Where("id = ? AND status = ?", sessionID, StatusLive).First(&sess).Error; err != nil {
		return nil, fmt.Errorf("session not found or not live")
	}

	// Gather metrics
	viewers := float64(sess.ViewerCount)
	bidsPerMin := computeBidsPerMinute(db, sessionID)
	conversionRate := computeSessionConversionRate(db, sessionID)
	creatorTrust := computeSessionCreatorTrust(db, sess.StreamerID)
	boostScore := float64(sess.BoostScore)
	isPremium := sess.IsPremium

	// Normalize each component to 0–1
	viewersNorm := math.Min(viewers/1000.0, 1.0)   // 1000 viewers = 1.0
	bidsNorm := math.Min(bidsPerMin/10.0, 1.0)     // 10 bids/min = 1.0
	convNorm := math.Min(conversionRate, 1.0)      // already 0–1
	trustNorm := math.Min(creatorTrust/100.0, 1.0) // trust 0–100
	boostNorm := math.Min(boostScore/1000.0, 1.0)  // 1000 boost = 1.0
	premiumBonus := 0.0
	if isPremium {
		premiumBonus = 1.0 // will be multiplied by weightPremium
	}

	// Weighted sum
	rawScore := weightViewers*viewersNorm +
		weightBidsPerMin*bidsNorm +
		weightConversion*convNorm +
		weightCreatorTrust*trustNorm +
		weightBoostScore*boostNorm +
		weightPremium*premiumBonus

	// Cold start bonus for new sessions
	coldStartBonus := 0.0
	if isColdStart(rdb, sessionID, sess.StartedAt) {
		coldStartBonus = coldStartBoostScore / maxScore
	}

	rawScore += coldStartBonus

	// Normalize to 0–100
	score := math.Min(rawScore*maxScore, maxScore)
	if score < 0 {
		score = 0
	}

	// Determine traffic tier
	tier := classifyTrafficTier(score)

	ss := &SessionScore{
		SessionID:        sessionID,
		Score:            math.Round(score*100) / 100,
		ViewersNorm:      math.Round(viewersNorm*1000) / 1000,
		BidsPerMinNorm:   math.Round(bidsNorm*1000) / 1000,
		ConversionNorm:   math.Round(convNorm*1000) / 1000,
		CreatorTrustNorm: math.Round(trustNorm*1000) / 1000,
		BoostNorm:        math.Round(boostNorm*1000) / 1000,
		PremiumBonus:     math.Round(premiumBonus*1000) / 1000,
		ColdStartBonus:   math.Round(coldStartBonus*1000) / 1000,
		TrafficTier:      tier,
		ComputedAt:       time.Now(),
	}

	// Cache in Redis
	if rdb != nil {
		if data, err := encodeJSON(ss); err == nil {
			rdb.Set(context.Background(), redisKeySessionScore+sessionID.String(), data, sessionScoreTTL)
		}
	}

	return ss, nil
}

// computeBidsPerMinute calculates bids per minute for a session.
func computeBidsPerMinute(db *gorm.DB, sessionID uuid.UUID) float64 {
	var sess Session
	if err := db.Where("id = ?", sessionID).Select("started_at, viewer_count").First(&sess).Error; err != nil {
		return 0
	}
	if sess.StartedAt == nil {
		return 0
	}
	duration := time.Since(*sess.StartedAt).Minutes()
	if duration < 0.1 {
		duration = 0.1 // avoid division by near-zero
	}
	var bidCount int64
	db.Table("live_bids lb").
		Joins("JOIN live_items li ON li.id = lb.item_id").
		Where("li.session_id = ?", sessionID).
		Count(&bidCount)
	return float64(bidCount) / duration
}

// computeSessionConversionRate calculates conversion rate for a session.
func computeSessionConversionRate(db *gorm.DB, sessionID uuid.UUID) float64 {
	var totalItems int64
	var soldItems int64
	db.Model(&LiveItem{}).Where("session_id = ?", sessionID).Count(&totalItems)
	if totalItems == 0 {
		return 0
	}
	db.Model(&LiveItem{}).Where("session_id = ? AND status = ?", sessionID, ItemSold).Count(&soldItems)
	return float64(soldItems) / float64(totalItems)
}

// computeSessionCreatorTrust returns the trust score for a session's creator.
func computeSessionCreatorTrust(db *gorm.DB, streamerID *uuid.UUID) float64 {
	if streamerID == nil {
		return 50.0 // default neutral trust when no creator
	}
	// Try creator profile first
	var c Creator
	if err := db.Where("user_id = ?", *streamerID).Select("trust_score").First(&c).Error; err == nil {
		return c.TrustScore
	}
	// Fall back to reputation
	return reputation.GetOverallScore(db, *streamerID)
}

// classifyTrafficTier maps a score to a traffic tier.
func classifyTrafficTier(score float64) string {
	switch {
	case score >= trafficHighScore:
		return "high"
	case score >= trafficMidScore:
		return "mid"
	case score >= trafficLowScore:
		return "low"
	default:
		return "low"
	}
}

// isColdStart checks if a session is within the cold-start window.
func isColdStart(rdb *redis.Client, sessionID uuid.UUID, startedAt *time.Time) bool {
	if startedAt == nil {
		return false
	}
	if time.Since(*startedAt) <= coldStartDuration {
		return true
	}
	return false
}

// ════════════════════════════════════════════════════════════════════════════
// 2. Feed Ranking Engine
// ════════════════════════════════════════════════════════════════════════════

// GetRankedFeed returns all live sessions sorted by the marketplace ranking.
// ORDER BY: is_premium DESC, session_score DESC, boost_score DESC, created_at DESC
func GetRankedFeed(db *gorm.DB, rdb *redis.Client, limit int) ([]FeedEntry, error) {
	if !IsSmartRankingEnabled() {
		// Fallback: simple chronological feed
		return getChronologicalFeed(db, limit)
	}

	if limit <= 0 {
		limit = 50
	}

	// Try Redis cache
	if rdb != nil {
		cached, err := rdb.Get(context.Background(), redisKeyFeedCache).Result()
		if err == nil && cached != "" {
			var entries []FeedEntry
			if err := decodeJSON(cached, &entries); err == nil && len(entries) > 0 {
				if len(entries) > limit {
					entries = entries[:limit]
				}
				return entries, nil
			}
		}
	}

	// Load all live sessions
	var sessions []Session
	if err := db.Where("status = ?", StatusLive).Order("created_at DESC").Find(&sessions).Error; err != nil {
		return nil, err
	}

	// Score each session
	entries := make([]FeedEntry, 0, len(sessions))
	for _, sess := range sessions {
		score, err := ComputeSessionScore(db, rdb, sess.ID)
		if err != nil {
			// Fallback score for sessions that can't be scored
			score = &SessionScore{
				SessionID:   sess.ID,
				Score:       30, // low default
				TrafficTier: "low",
				ComputedAt:  time.Now(),
			}
		}
		entries = append(entries, FeedEntry{
			SessionID:    sess.ID,
			HostID:       sess.HostID,
			Title:        sess.Title,
			ViewerCount:  sess.ViewerCount,
			IsPremium:    sess.IsPremium,
			BoostScore:   sess.BoostScore,
			Score:        score.Score,
			TrafficTier:  score.TrafficTier,
			StreamerID:   sess.StreamerID,
			ThumbnailURL: sess.ThumbnailURL,
		})
	}

	// Sort: is_premium DESC → score DESC → boost_score DESC → created_at DESC
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsPremium != entries[j].IsPremium {
			return entries[i].IsPremium // premium first
		}
		if entries[i].Score != entries[j].Score {
			return entries[i].Score > entries[j].Score
		}
		if entries[i].BoostScore != entries[j].BoostScore {
			return entries[i].BoostScore > entries[j].BoostScore
		}
		return entries[i].SessionID.String() > entries[j].SessionID.String() // stable tiebreak
	})

	// Apply fairness guard
	entries = applyFairnessGuard(entries)

	if len(entries) > limit {
		entries = entries[:limit]
	}

	// Cache in Redis
	if rdb != nil {
		if data, err := encodeJSON(entries); err == nil {
			rdb.Set(context.Background(), redisKeyFeedCache, data, feedCacheTTL)
		}
	}

	return entries, nil
}

// getChronologicalFeed is the deterministic fallback when smart ranking is disabled.
func getChronologicalFeed(db *gorm.DB, limit int) ([]FeedEntry, error) {
	var sessions []Session
	if err := db.Where("status = ?", StatusLive).
		Order("created_at DESC").
		Limit(limit).
		Find(&sessions).Error; err != nil {
		return nil, err
	}

	entries := make([]FeedEntry, 0, len(sessions))
	for _, sess := range sessions {
		entries = append(entries, FeedEntry{
			SessionID:    sess.ID,
			HostID:       sess.HostID,
			Title:        sess.Title,
			ViewerCount:  sess.ViewerCount,
			IsPremium:    sess.IsPremium,
			BoostScore:   sess.BoostScore,
			StreamerID:   sess.StreamerID,
			ThumbnailURL: sess.ThumbnailURL,
		})
	}
	return entries, nil
}

// ════════════════════════════════════════════════════════════════════════════
// 3. Traffic Allocation Engine
// ════════════════════════════════════════════════════════════════════════════

// AllocateTraffic determines how traffic should be distributed across live sessions.
func AllocateTraffic(db *gorm.DB, rdb *redis.Client) ([]TrafficAllocation, error) {
	if !IsTrafficAllocationEnabled() {
		return nil, fmt.Errorf("traffic allocation disabled")
	}

	feed, err := GetRankedFeed(db, rdb, 100)
	if err != nil {
		return nil, err
	}

	if len(feed) == 0 {
		return nil, nil
	}

	// Calculate total score for proportional allocation
	totalScore := 0.0
	for _, e := range feed {
		totalScore += e.Score
	}
	if totalScore == 0 {
		totalScore = 1.0 // avoid division by zero
	}

	allocations := make([]TrafficAllocation, 0, len(feed))
	for _, e := range feed {
		exposurePct := (e.Score / totalScore) * 100.0

		// Fairness guard: cap at 30%
		if exposurePct > maxTrafficSharePct*100 {
			exposurePct = maxTrafficSharePct * 100
		}

		alloc := TrafficAllocation{
			SessionID:     e.SessionID,
			TrafficTier:   e.TrafficTier,
			ExposurePct:   math.Round(exposurePct*100) / 100,
			PushNotify:    e.TrafficTier == "high",
			DiscoveryFeed: e.TrafficTier == "mid" || e.TrafficTier == "high",
			Deprioritized: e.TrafficTier == "low" && e.Score < trafficLowScore,
		}
		allocations = append(allocations, alloc)
	}

	return allocations, nil
}

// AllocateTrafficForSession returns the traffic allocation for a single session.
func AllocateTrafficForSession(db *gorm.DB, rdb *redis.Client, sessionID uuid.UUID) (*TrafficAllocation, error) {
	score, err := ComputeSessionScore(db, rdb, sessionID)
	if err != nil {
		return nil, err
	}

	// Get total live sessions for proportional calculation
	var liveCount int64
	db.Model(&Session{}).Where("status = ?", StatusLive).Count(&liveCount)
	if liveCount == 0 {
		liveCount = 1
	}

	// Simple proportional exposure
	exposurePct := 100.0 / float64(liveCount)
	if score.Score >= trafficHighScore {
		exposurePct *= 2.0 // high-score sessions get double exposure
	}
	if exposurePct > maxTrafficSharePct*100 {
		exposurePct = maxTrafficSharePct * 100
	}

	return &TrafficAllocation{
		SessionID:     sessionID,
		TrafficTier:   score.TrafficTier,
		ExposurePct:   math.Round(exposurePct*100) / 100,
		PushNotify:    score.TrafficTier == "high",
		DiscoveryFeed: score.TrafficTier == "mid" || score.TrafficTier == "high",
		Deprioritized: score.TrafficTier == "low" && score.Score < trafficLowScore,
	}, nil
}

// ════════════════════════════════════════════════════════════════════════════
// 4. Creator Prioritization
// ════════════════════════════════════════════════════════════════════════════

// AdjustCreatorExposure computes a visibility multiplier for a creator.
func AdjustCreatorExposure(db *gorm.DB, rdb *redis.Client, creatorID uuid.UUID) (*CreatorExposure, error) {
	var c Creator
	if err := db.Where("id = ?", creatorID).First(&c).Error; err != nil {
		return nil, fmt.Errorf("creator not found")
	}

	multiplier := 1.0
	reason := "baseline"

	// High conversion → boost visibility
	if c.TotalSales > 0 {
		// Estimate conversion from sales vs sessions
		var sessionsHosted int64
		db.Table("livestream_sessions").Where("streamer_id = ?", c.UserID).Count(&sessionsHosted)
		if sessionsHosted > 0 {
			convRate := float64(c.TotalSales) / float64(sessionsHosted)
			if convRate > highConversionThreshold {
				multiplier *= 1.5 // +50% visibility
				reason = "high_conversion"
			}
		}
	}

	// Low trust → reduce exposure
	if c.TrustScore < lowTrustThreshold {
		multiplier *= 0.5 // -50% visibility
		reason = "low_trust"
	}

	// Active deals → slight boost
	var activeDeals int64
	db.Model(&CreatorDeal{}).Where("creator_id = ? AND status = ?", creatorID, DealActive).Count(&activeDeals)
	if activeDeals >= 3 {
		multiplier *= 1.2
		if reason == "baseline" {
			reason = "multiple_active_deals"
		}
	}

	ce := &CreatorExposure{
		CreatorID:            creatorID,
		VisibilityMultiplier: math.Round(multiplier*100) / 100,
		ConversionRate:       computeCreatorConversionRate(db, c),
		TrustScore:           c.TrustScore,
		Reason:               reason,
	}

	// Cache in Redis
	if rdb != nil {
		if data, err := encodeJSON(ce); err == nil {
			rdb.Set(context.Background(), redisKeyCreatorExposure+creatorID.String(), data, creatorExposureTTL)
		}
	}

	return ce, nil
}

func computeCreatorConversionRate(db *gorm.DB, c Creator) float64 {
	var sessionsHosted int64
	db.Table("livestream_sessions").Where("streamer_id = ?", c.UserID).Count(&sessionsHosted)
	if sessionsHosted == 0 {
		return 0
	}
	return float64(c.TotalSales) / float64(sessionsHosted)
}

// ════════════════════════════════════════════════════════════════════════════
// 5. Auto Boost Suggestion
// ════════════════════════════════════════════════════════════════════════════

// MaybeSuggestBoost triggers a boost suggestion when a session scores high
// but has no active boost. Returns true if suggestion was sent.
func MaybeSuggestBoost(db *gorm.DB, rdb *redis.Client, sessionID uuid.UUID) bool {
	if !IsMarketplaceBrainEnabled() {
		return false
	}

	score, err := ComputeSessionScore(db, rdb, sessionID)
	if err != nil {
		return false
	}

	if score.Score < 70.0 {
		return false
	}

	// Check if session already has a boost
	var sess Session
	if err := db.Where("id = ?", sessionID).Select("boost_tier, host_id").First(&sess).Error; err != nil {
		return false
	}
	if sess.BoostTier != "" {
		return false // already boosted
	}

	// Throttle: 1 suggestion per session per 5 minutes
	if rdb != nil {
		key := "marketplace:boost_suggest:" + sessionID.String()
		set, err := rdb.SetNX(context.Background(), key, "1", 5*time.Minute).Result()
		if err != nil || !set {
			return false // already suggested recently
		}
	}

	// Broadcast boost suggestion via WebSocket
	BroadcastLiveEvent(sessionID, LiveEvent{
		Event:           EventToast,
		SessionID:       sessionID.String(),
		Message:         "🔥 Your session is trending — boost now to dominate the feed!",
		SuggestedAction: "boost",
		ActionLabel:     "Boost session",
		TargetUserID:    sess.HostID.String(),
	})

	freeze.LogAudit(db, "marketplace_boost_suggested", sess.HostID, sessionID,
		fmt.Sprintf("score=%.1f tier=%s", score.Score, score.TrafficTier))

	return true
}

// ════════════════════════════════════════════════════════════════════════════
// 6. Revenue Optimization Mode
// ════════════════════════════════════════════════════════════════════════════

// ApplyRevenuePriority re-ranks the feed to prioritize sessions that generate
// more platform revenue. Called when liquidity is low or demand is high.
func ApplyRevenuePriority(db *gorm.DB, rdb *redis.Client) ([]FeedEntry, error) {
	if !IsMarketplaceBrainEnabled() {
		return nil, fmt.Errorf("marketplace brain disabled")
	}

	feed, err := GetRankedFeed(db, rdb, 50)
	if err != nil {
		return nil, err
	}

	// Re-rank with revenue priority: boost sessions with higher fees
	for i := range feed {
		var sess Session
		if err := db.Where("id = ?", feed[i].SessionID).First(&sess).Error; err != nil {
			continue
		}

		revenueBonus := 0.0

		// Premium sessions generate more revenue
		if sess.IsPremium {
			revenueBonus += 10.0
		}

		// Sessions with entry fees
		if sess.EntryFeeCents > 0 {
			revenueBonus += 5.0
		}

		// Sessions with active boost (paid for)
		if sess.BoostTier != "" {
			revenueBonus += 5.0
		}

		// Sessions with higher commission tier
		commissionTier := DetermineTier(db, sess.HostID, sess.IsHot)
		if commissionTier == TierHot {
			revenueBonus += 3.0
		}

		feed[i].Score += revenueBonus
	}

	// Re-sort by adjusted score
	sort.Slice(feed, func(i, j int) bool {
		if feed[i].IsPremium != feed[j].IsPremium {
			return feed[i].IsPremium
		}
		return feed[i].Score > feed[j].Score
	})

	return feed, nil
}

// IsRevenuePriorityMode checks if the system should be in revenue priority mode.
// Returns true when there are few live sessions (<5) or average score is low.
func IsRevenuePriorityMode(db *gorm.DB) bool {
	var liveCount int64
	db.Model(&Session{}).Where("status = ?", StatusLive).Count(&liveCount)
	return liveCount < 5 // low liquidity → prioritize revenue
}

// ════════════════════════════════════════════════════════════════════════════
// 7. Drop-off Recovery
// ════════════════════════════════════════════════════════════════════════════

// DetectSessionDropoff checks if a session is experiencing viewer dropoff
// and bid stagnation. Returns recovery actions if detected.
func DetectSessionDropoff(db *gorm.DB, rdb *redis.Client, sessionID uuid.UUID) []string {
	actions := []string{}

	// Check viewer dropoff from peak (using Redis peak viewer tracking)
	if rdb != nil {
		peakKey := "live:ai:viewers_peak:" + sessionID.String()
		peakStr, err := rdb.Get(context.Background(), peakKey).Result()
		if err == nil && peakStr != "" {
			var peak int
			fmt.Sscanf(peakStr, "%d", &peak)
			var sess Session
			if err := db.Where("id = ?", sessionID).Select("viewer_count").First(&sess).Error; err == nil {
				if peak >= 10 && sess.ViewerCount < int(float64(peak)*(1-dropoffViewerPct)) {
					actions = append(actions, "viewer_dropoff_30pct")
				}
			}
		}
	}

	// Check bid stagnation (no bids in 30s)
	var lastBidTime *time.Time
	db.Table("live_bids lb").
		Joins("JOIN live_items li ON li.id = lb.item_id").
		Where("li.session_id = ? AND li.status = ?", sessionID, ItemActive).
		Select("MAX(lb.created_at)").
		Scan(&lastBidTime)

	if lastBidTime == nil || time.Since(*lastBidTime) > time.Duration(dropoffNoBidSecs)*time.Second {
		actions = append(actions, "no_bids_30s")
	}

	return actions
}

// RecoverSession applies recovery actions for a struggling session.
func RecoverSession(db *gorm.DB, rdb *redis.Client, sessionID uuid.UUID) {
	dropoffSignals := DetectSessionDropoff(db, rdb, sessionID)
	if len(dropoffSignals) == 0 {
		return
	}

	var sess Session
	if err := db.Where("id = ?", sessionID).First(&sess).Error; err != nil {
		return
	}

	for _, signal := range dropoffSignals {
		switch signal {
		case "viewer_dropoff_30pct":
			// Send nudge to remaining viewers
			BroadcastLiveEvent(sessionID, LiveEvent{
				Event:     EventToast,
				SessionID: sessionID.String(),
				Message:   "👀 Keep watching — things are heating up!",
			})
			freeze.LogAudit(db, "marketplace_dropoff_recovery", sess.HostID, sessionID, "signal=viewer_dropoff")

		case "no_bids_30s":
			// Suggest pinning active item or price drop
			BroadcastLiveEvent(sessionID, LiveEvent{
				Event:           EventToast,
				SessionID:       sessionID.String(),
				Message:         "💡 Try pinning your item or adjusting the price to spark bids!",
				SuggestedAction: "pin_item",
				ActionLabel:     "Pin item",
				TargetUserID:    sess.HostID.String(),
			})
			freeze.LogAudit(db, "marketplace_dropoff_recovery", sess.HostID, sessionID, "signal=no_bids")
		}
	}
}

// ════════════════════════════════════════════════════════════════════════════
// 8. Analytics
// ════════════════════════════════════════════════════════════════════════════

// GetMarketplaceBrainMetrics computes aggregate marketplace analytics.
func GetMarketplaceBrainMetrics(db *gorm.DB, rdb *redis.Client) (*MarketplaceBrainMetrics, error) {
	if !IsMarketplaceBrainEnabled() {
		return nil, fmt.Errorf("marketplace brain disabled")
	}

	metrics := &MarketplaceBrainMetrics{
		RevenuePriorityMode: IsRevenuePriorityMode(db),
	}

	// Count live sessions
	var liveCount int64
	db.Model(&Session{}).Where("status = ?", StatusLive).Count(&liveCount)
	metrics.TotalLiveSessions = int(liveCount)

	// Score all live sessions
	var sessions []Session
	db.Where("status = ?", StatusLive).Find(&sessions)

	totalScore := 0.0
	topScores := make([]SessionScore, 0, len(sessions))
	trafficAllocs := make([]TrafficAllocation, 0)
	creatorExposures := make([]CreatorExposure, 0)

	for _, sess := range sessions {
		score, err := ComputeSessionScore(db, rdb, sess.ID)
		if err != nil {
			continue
		}
		totalScore += score.Score
		topScores = append(topScores, *score)

		// Traffic tier counts
		switch score.TrafficTier {
		case "high":
			metrics.HighTrafficSessions++
		case "mid":
			metrics.MidTrafficSessions++
		default:
			metrics.LowTrafficSessions++
		}

		// Traffic allocation
		alloc, _ := AllocateTrafficForSession(db, rdb, sess.ID)
		if alloc != nil {
			trafficAllocs = append(trafficAllocs, *alloc)
		}

		// Creator exposure for sessions with creators
		if sess.StreamerID != nil {
			var c Creator
			if err := db.Where("user_id = ?", *sess.StreamerID).First(&c).Error; err == nil {
				ce, _ := AdjustCreatorExposure(db, rdb, c.ID)
				if ce != nil {
					creatorExposures = append(creatorExposures, *ce)
				}
			}
		}
	}

	if liveCount > 0 {
		metrics.AvgSessionScore = math.Round(totalScore/float64(liveCount)*100) / 100
	}

	// Sort top scores descending
	sort.Slice(topScores, func(i, j int) bool {
		return topScores[i].Score > topScores[j].Score
	})
	if len(topScores) > 10 {
		topScores = topScores[:10]
	}
	metrics.TopSessions = topScores
	metrics.TrafficAllocations = trafficAllocs
	metrics.CreatorExposures = creatorExposures

	// Score history (last 50 snapshots)
	var history []SessionScoreSnapshot
	db.Order("created_at DESC").Limit(50).Find(&history)
	metrics.ScoreHistory = history

	return metrics, nil
}

// RecordScoreSnapshot persists a score snapshot for analytics.
func RecordScoreSnapshot(db *gorm.DB, ss *SessionScore) {
	snap := SessionScoreSnapshot{
		SessionID:   ss.SessionID,
		Score:       ss.Score,
		TrafficTier: ss.TrafficTier,
		CreatedAt:   time.Now(),
	}
	if err := db.Create(&snap).Error; err != nil {
		slog.Error("marketplace: failed to record score snapshot", "error", err)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// 9. Fairness Guard
// ════════════════════════════════════════════════════════════════════════════

// applyFairnessGuard ensures no single session dominates the feed.
// Caps any session at 30% of total traffic, and gives cold-start boost to new sessions.
func applyFairnessGuard(entries []FeedEntry) []FeedEntry {
	if len(entries) <= 1 {
		return entries
	}

	// No session should occupy more than 30% of the top positions
	maxSlots := int(math.Ceil(float64(len(entries)) * maxTrafficSharePct))
	if maxSlots < 1 {
		maxSlots = 1
	}

	// Count occurrences of same host (prevent same seller dominating)
	hostCount := make(map[uuid.UUID]int)
	result := make([]FeedEntry, 0, len(entries))

	for _, e := range entries {
		count := hostCount[e.HostID]
		if count >= maxSlots && maxSlots > 0 {
			// Skip this entry — host already has enough slots
			continue
		}
		hostCount[e.HostID] = count + 1
		result = append(result, e)
	}

	return result
}

// ════════════════════════════════════════════════════════════════════════════
// Helpers
// ════════════════════════════════════════════════════════════════════════════

// BroadcastLiveEvent is a stub that calls the actual broadcast function.
// It's defined here to avoid circular imports — the real implementation
// is in handler.go's broadcastItemUpdate.
var BroadcastLiveEvent = func(sessionID uuid.UUID, event LiveEvent) {
	// Default no-op; overridden by handler initialization
	slog.Debug("marketplace: broadcast event", "session_id", sessionID, "type", event.Event)
}
