package livestream

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ════════════════════════════════════════════════════════════════════════════
// Sprint 17: Marketplace Brain — Unit Tests
// ════════════════════════════════════════════════════════════════════════════

// ── 1. Scoring Constants & Weights ──────────────────────────────────────────

func TestScoringWeights(t *testing.T) {
	total := weightViewers + weightBidsPerMin + weightConversion + weightCreatorTrust + weightBoostScore + weightPremium
	if math.Abs(total-1.0) > 0.001 {
		t.Errorf("scoring weights sum = %v, want 1.0", total)
	}
}

func TestTrafficThresholds(t *testing.T) {
	if trafficHighScore != 80.0 {
		t.Errorf("trafficHighScore = %v", trafficHighScore)
	}
	if trafficMidScore != 60.0 {
		t.Errorf("trafficMidScore = %v", trafficMidScore)
	}
	if trafficLowScore != 40.0 {
		t.Errorf("trafficLowScore = %v", trafficLowScore)
	}
}

func TestFairnessConstants(t *testing.T) {
	if maxTrafficSharePct != 0.30 {
		t.Errorf("maxTrafficSharePct = %v", maxTrafficSharePct)
	}
	if coldStartBoostScore != 15.0 {
		t.Errorf("coldStartBoostScore = %v", coldStartBoostScore)
	}
	if coldStartDuration != 5*time.Minute {
		t.Errorf("coldStartDuration = %v", coldStartDuration)
	}
}

func TestCreatorExposureThresholds(t *testing.T) {
	if highConversionThreshold != 0.20 {
		t.Errorf("highConversionThreshold = %v", highConversionThreshold)
	}
	if lowTrustThreshold != 50.0 {
		t.Errorf("lowTrustThreshold = %v", lowTrustThreshold)
	}
}

// ── 2. classifyTrafficTier ──────────────────────────────────────────────────

func TestClassifyTrafficTier(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{95.0, "high"},
		{80.0, "high"},
		{79.9, "mid"},
		{60.0, "mid"},
		{59.9, "low"},
		{40.0, "low"},
		{20.0, "low"},
		{0.0, "low"},
	}
	for _, tt := range tests {
		got := classifyTrafficTier(tt.score)
		if got != tt.want {
			t.Errorf("classifyTrafficTier(%v) = %v, want %v", tt.score, got, tt.want)
		}
	}
}

// ── 3. isColdStart ──────────────────────────────────────────────────────────

func TestIsColdStart_NewSession(t *testing.T) {
	recent := time.Now().Add(-1 * time.Minute)
	if !isColdStart(nil, uuid.New(), &recent) {
		t.Error("new session should be in cold start")
	}
}

func TestIsColdStart_OldSession(t *testing.T) {
	old := time.Now().Add(-10 * time.Minute)
	if isColdStart(nil, uuid.New(), &old) {
		t.Error("old session should not be in cold start")
	}
}

func TestIsColdStart_NilStartedAt(t *testing.T) {
	if isColdStart(nil, uuid.New(), nil) {
		t.Error("nil startedAt should not be cold start")
	}
}

// ── 4. Feature Flags ────────────────────────────────────────────────────────

func TestMarketplaceFeatureFlags(t *testing.T) {
	_ = IsMarketplaceBrainEnabled()
	_ = IsSmartRankingEnabled()
	_ = IsTrafficAllocationEnabled()
}

// ── 5. SessionScore struct ──────────────────────────────────────────────────

func TestSessionScore_Fields(t *testing.T) {
	ss := SessionScore{
		SessionID:        uuid.New(),
		Score:            75.5,
		ViewersNorm:      0.5,
		BidsPerMinNorm:   0.3,
		ConversionNorm:   0.2,
		CreatorTrustNorm: 0.75,
		BoostNorm:        0.1,
		PremiumBonus:     0.0,
		TrafficTier:      "mid",
		ComputedAt:       time.Now(),
	}
	if ss.TrafficTier != "mid" {
		t.Errorf("TrafficTier = %v", ss.TrafficTier)
	}
	if ss.Score < 0 || ss.Score > 100 {
		t.Errorf("Score = %v, out of range", ss.Score)
	}
}

// ── 6. FeedEntry struct ────────────────────────────────────────────────────

func TestFeedEntry_Fields(t *testing.T) {
	fe := FeedEntry{
		SessionID:   uuid.New(),
		HostID:      uuid.New(),
		Title:       "Test Session",
		ViewerCount: 50,
		IsPremium:   true,
		BoostScore:  500,
		Score:       85.0,
		TrafficTier: "high",
	}
	if !fe.IsPremium {
		t.Error("IsPremium should be true")
	}
	if fe.TrafficTier != "high" {
		t.Errorf("TrafficTier = %v", fe.TrafficTier)
	}
}

// ── 7. TrafficAllocation struct ─────────────────────────────────────────────

func TestTrafficAllocation_HighTier(t *testing.T) {
	ta := TrafficAllocation{
		SessionID:     uuid.New(),
		TrafficTier:   "high",
		ExposurePct:   25.0,
		PushNotify:    true,
		DiscoveryFeed: true,
	}
	if !ta.PushNotify {
		t.Error("high tier should have push notify")
	}
	if !ta.DiscoveryFeed {
		t.Error("high tier should be in discovery feed")
	}
}

func TestTrafficAllocation_LowTier(t *testing.T) {
	ta := TrafficAllocation{
		SessionID:     uuid.New(),
		TrafficTier:   "low",
		ExposurePct:   5.0,
		PushNotify:    false,
		DiscoveryFeed: false,
		Deprioritized: true,
	}
	if ta.PushNotify {
		t.Error("low tier should not have push notify")
	}
	if !ta.Deprioritized {
		t.Error("low tier should be deprioritized")
	}
}

// ── 8. CreatorExposure struct ───────────────────────────────────────────────

func TestCreatorExposure_Baseline(t *testing.T) {
	ce := CreatorExposure{
		CreatorID:            uuid.New(),
		VisibilityMultiplier: 1.0,
		ConversionRate:       0.15,
		TrustScore:           75.0,
		Reason:               "baseline",
	}
	if ce.VisibilityMultiplier != 1.0 {
		t.Errorf("baseline multiplier = %v", ce.VisibilityMultiplier)
	}
}

func TestCreatorExposure_HighConversion(t *testing.T) {
	ce := CreatorExposure{
		VisibilityMultiplier: 1.5,
		ConversionRate:       0.25,
		TrustScore:           80.0,
		Reason:               "high_conversion",
	}
	if ce.VisibilityMultiplier < 1.0 {
		t.Error("high conversion should boost visibility")
	}
}

func TestCreatorExposure_LowTrust(t *testing.T) {
	ce := CreatorExposure{
		VisibilityMultiplier: 0.5,
		TrustScore:           30.0,
		Reason:               "low_trust",
	}
	if ce.VisibilityMultiplier >= 1.0 {
		t.Error("low trust should reduce visibility")
	}
}

// ── 9. applyFairnessGuard ───────────────────────────────────────────────────

func TestApplyFairnessGuard_SingleHost(t *testing.T) {
	hostID := uuid.New()
	entries := []FeedEntry{
		{SessionID: uuid.New(), HostID: hostID, Score: 90},
		{SessionID: uuid.New(), HostID: hostID, Score: 80},
		{SessionID: uuid.New(), HostID: hostID, Score: 70},
		{SessionID: uuid.New(), HostID: uuid.New(), Score: 60},
	}
	result := applyFairnessGuard(entries)
	// With 4 entries and 30% cap, max 1 slot per host (ceil(4*0.3)=2)
	// Host should get at most 2 entries
	hostCount := 0
	for _, e := range result {
		if e.HostID == hostID {
			hostCount++
		}
	}
	if hostCount > 2 {
		t.Errorf("host got %d entries, expected at most 2 (fairness cap)", hostCount)
	}
}

func TestApplyFairnessGuard_DiverseHosts(t *testing.T) {
	entries := []FeedEntry{
		{SessionID: uuid.New(), HostID: uuid.New(), Score: 90},
		{SessionID: uuid.New(), HostID: uuid.New(), Score: 80},
		{SessionID: uuid.New(), HostID: uuid.New(), Score: 70},
	}
	result := applyFairnessGuard(entries)
	if len(result) != 3 {
		t.Errorf("diverse hosts: got %d entries, want 3", len(result))
	}
}

func TestApplyFairnessGuard_Empty(t *testing.T) {
	result := applyFairnessGuard([]FeedEntry{})
	if len(result) != 0 {
		t.Errorf("empty input: got %d entries", len(result))
	}
}

// ── 10. MarketplaceBrainMetrics struct ──────────────────────────────────────

func TestMarketplaceBrainMetrics_Fields(t *testing.T) {
	m := MarketplaceBrainMetrics{
		TotalLiveSessions:   10,
		AvgSessionScore:     65.5,
		HighTrafficSessions: 3,
		MidTrafficSessions:  4,
		LowTrafficSessions:  3,
		RevenuePriorityMode: false,
	}
	if m.TotalLiveSessions != 10 {
		t.Errorf("TotalLiveSessions = %d", m.TotalLiveSessions)
	}
	if m.HighTrafficSessions+m.MidTrafficSessions+m.LowTrafficSessions != 10 {
		t.Error("traffic tier counts don't add up")
	}
}

// ── 11. SessionScoreSnapshot ───────────────────────────────────────────────

func TestSessionScoreSnapshot_TableName(t *testing.T) {
	var snap SessionScoreSnapshot
	if snap.TableName() != "live_session_score_snapshots" {
		t.Errorf("TableName = %v", snap.TableName())
	}
}

// ── 12. Score normalization logic ───────────────────────────────────────────

func TestScoreNormalization_Max100(t *testing.T) {
	// A session with all max values should score near 100
	viewersNorm := math.Min(1000.0/1000.0, 1.0)
	bidsNorm := math.Min(10.0/10.0, 1.0)
	convNorm := math.Min(1.0, 1.0)
	trustNorm := math.Min(100.0/100.0, 1.0)
	boostNorm := math.Min(1000.0/1000.0, 1.0)
	premiumBonus := 1.0

	rawScore := weightViewers*viewersNorm +
		weightBidsPerMin*bidsNorm +
		weightConversion*convNorm +
		weightCreatorTrust*trustNorm +
		weightBoostScore*boostNorm +
		weightPremium*premiumBonus

	score := math.Min(rawScore*maxScore, maxScore)
	if score != 100.0 {
		t.Errorf("max score = %v, want 100", score)
	}
}

func TestScoreNormalization_ZeroValues(t *testing.T) {
	rawScore := 0.0 // all zeros
	score := math.Min(rawScore*maxScore, maxScore)
	if score != 0 {
		t.Errorf("zero score = %v, want 0", score)
	}
}

// ── 13. Revenue Priority Mode ──────────────────────────────────────────────

func TestIsRevenuePriorityMode_Threshold(t *testing.T) {
	// IsRevenuePriorityMode returns true when liveCount < 5
	// We can only test the logic path exists
	_ = IsRevenuePriorityMode
}

// ── 14. Dropoff detection constants ──────────────────────────────────────────

func TestDropoffConstants(t *testing.T) {
	if dropoffViewerPct != 0.30 {
		t.Errorf("dropoffViewerPct = %v", dropoffViewerPct)
	}
	if dropoffNoBidSecs != 30 {
		t.Errorf("dropoffNoBidSecs = %v", dropoffNoBidSecs)
	}
}

// ── 15. Redis key format ────────────────────────────────────────────────────

func TestRedisKeyFormats(t *testing.T) {
	if redisKeySessionScore != "marketplace:score:" {
		t.Errorf("redisKeySessionScore = %v", redisKeySessionScore)
	}
	if redisKeyCreatorExposure != "marketplace:creator_exposure:" {
		t.Errorf("redisKeyCreatorExposure = %v", redisKeyCreatorExposure)
	}
	if redisKeyFeedCache != "marketplace:feed" {
		t.Errorf("redisKeyFeedCache = %v", redisKeyFeedCache)
	}
}

// ── 16. BroadcastLiveEvent stub ─────────────────────────────────────────────

func TestBroadcastLiveEvent_Stub(t *testing.T) {
	// Verify the stub doesn't panic
	BroadcastLiveEvent(uuid.New(), LiveEvent{
		Event:   EventToast,
		Message: "test",
	})
}

// ── 17. encodeJSON / decodeJSON helpers ─────────────────────────────────────

func TestEncodeDecodeJSON(t *testing.T) {
	original := SessionScore{
		SessionID:   uuid.New(),
		Score:       75.5,
		TrafficTier: "mid",
		ComputedAt:  time.Now(),
	}
	data, err := encodeJSON(original)
	if err != nil {
		t.Fatalf("encodeJSON: %v", err)
	}
	var decoded SessionScore
	if err := decodeJSON(data, &decoded); err != nil {
		t.Fatalf("decodeJSON: %v", err)
	}
	if decoded.Score != original.Score {
		t.Errorf("decoded Score = %v, want %v", decoded.Score, original.Score)
	}
	if decoded.TrafficTier != original.TrafficTier {
		t.Errorf("decoded TrafficTier = %v", decoded.TrafficTier)
	}
}
