package livestream

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// ════════════════════════════════════════════════════════════════════════════
// Sprint 15: Viral Growth Engine — unit tests (pure logic + models)
// ════════════════════════════════════════════════════════════════════════════

// ── Short code generation ────────────────────────────────────────────────

func TestGenerateShortCode_Length(t *testing.T) {
	code := generateShortCode()
	if len(code) != 10 {
		t.Fatalf("Expected 10 chars, got %d (%q)", len(code), code)
	}
}

func TestGenerateShortCode_URLSafe(t *testing.T) {
	for i := 0; i < 20; i++ {
		code := generateShortCode()
		if strings.ContainsAny(code, "=/+ \t\n") {
			t.Fatalf("Code %q contains unsafe chars", code)
		}
	}
}

func TestGenerateShortCode_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		c := generateShortCode()
		if seen[c] {
			t.Fatalf("Collision detected at iteration %d: %s", i, c)
		}
		seen[c] = true
	}
}

// ── Reward constants (fraud safety sanity) ────────────────────────────────

func TestRewardConstants_Positive(t *testing.T) {
	rewards := []int64{
		RewardLiveInviteBid, RewardLiveInviteWin, RewardWinShareAttribution,
		RewardFirstBidCashback, RewardStreakDay3, RewardStreakDay7,
	}
	for i, r := range rewards {
		if r <= 0 {
			t.Fatalf("Reward %d must be positive, got %d", i, r)
		}
	}
}

func TestRewardHierarchy(t *testing.T) {
	// Higher-effort rewards should pay more than lower-effort ones
	if RewardLiveInviteWin <= RewardLiveInviteBid {
		t.Fatal("Win reward should exceed bid-only reward")
	}
	if RewardStreakDay7 <= RewardStreakDay3 {
		t.Fatal("7-day streak should exceed 3-day streak")
	}
	if RewardFirstBidCashback >= RewardLiveInviteWin {
		t.Fatal("First-bid cashback should be smallest")
	}
}

func TestMaxDailyRewardCap(t *testing.T) {
	if maxDailyRewardCents <= 0 {
		t.Fatal("Daily cap must be positive")
	}
	// Cap should be meaningful (≥ biggest single reward)
	if maxDailyRewardCents < RewardStreakDay7*2 {
		t.Fatal("Daily cap too low to accommodate realistic usage")
	}
}

// ── Group Entry Discount ─────────────────────────────────────────────────

func TestGroupEntryDiscountPct(t *testing.T) {
	if GroupEntryFeeDiscountPct <= 0 || GroupEntryFeeDiscountPct > 1 {
		t.Fatalf("Discount pct must be in (0, 1], got %f", GroupEntryFeeDiscountPct)
	}
}

func TestGroupMinSize(t *testing.T) {
	if GroupMinSize < 2 {
		t.Fatal("Group min size should be at least 2 (to be a 'group')")
	}
}

// ── Feature flags ────────────────────────────────────────────────────────

func TestIsLiveReferralsEnabled(t *testing.T) {
	if !IsLiveReferralsEnabled() {
		t.Fatal("Referrals should be enabled by default")
	}
}

func TestIsLiveInvitesEnabled(t *testing.T) {
	if !IsLiveInvitesEnabled() {
		t.Fatal("Live invites should be enabled by default")
	}
}

func TestIsStreaksEnabled(t *testing.T) {
	if !IsStreaksEnabled() {
		t.Fatal("Streaks should be enabled by default")
	}
}

func TestIsGroupBuyEnabled(t *testing.T) {
	if !IsGroupBuyEnabled() {
		t.Fatal("Group buy should be enabled by default")
	}
}

func TestIsShareRewardsEnabled(t *testing.T) {
	if !IsShareRewardsEnabled() {
		t.Fatal("Share rewards should be enabled by default")
	}
}

func TestFlagRespected_LiveInvites(t *testing.T) {
	t.Setenv("ENABLE_LIVE_INVITES", "false")
	if IsLiveInvitesEnabled() {
		t.Fatal("Flag should be respected when disabled")
	}
}

// ── Model sanity (table names + field persistence) ────────────────────────

func TestLiveInviteModel(t *testing.T) {
	inv := LiveInvite{
		InviterID:  uuid.New(),
		SessionID:  uuid.New(),
		InviteCode: "TEST123456",
	}
	if inv.TableName() != "live_invites" {
		t.Fatal("TableName should be live_invites")
	}
	if inv.Joined || inv.BidPlaced || inv.Won {
		t.Fatal("New invite should default to not-joined")
	}
}

func TestWinShareModel(t *testing.T) {
	ws := WinShare{
		UserID:     uuid.New(),
		SessionID:  uuid.New(),
		ItemID:     uuid.New(),
		PriceCents: 100_00,
	}
	if ws.TableName() != "live_win_shares" {
		t.Fatal("TableName should be live_win_shares")
	}
}

func TestGroupInviteModel(t *testing.T) {
	g := GroupInvite{CreatorID: uuid.New(), SessionID: uuid.New(), MemberCount: 1}
	if g.TableName() != "live_group_invites" {
		t.Fatal("TableName should be live_group_invites")
	}
	if g.MemberCount < 1 {
		t.Fatal("Creator should count as first member")
	}
}

func TestGroupMemberModel(t *testing.T) {
	m := GroupMember{GroupID: uuid.New(), UserID: uuid.New()}
	if m.TableName() != "live_group_members" {
		t.Fatal("TableName should be live_group_members")
	}
}

func TestUserStreakModel(t *testing.T) {
	s := UserStreak{UserID: uuid.New(), Type: "live_join", CurrentStreak: 5, LongestStreak: 10}
	if s.TableName() != "live_user_streaks" {
		t.Fatal("TableName should be live_user_streaks")
	}
	if s.CurrentStreak > s.LongestStreak {
		t.Fatal("Current can't exceed longest (business rule)")
	}
}

func TestGrowthRewardModel(t *testing.T) {
	r := GrowthReward{
		UserID:      uuid.New(),
		RewardType:  RewardTypeFirstBidCashback,
		ReferenceID: "test-ref",
		AmountCents: 100,
		Status:      "pending",
	}
	if r.TableName() != "live_growth_rewards" {
		t.Fatal("TableName should be live_growth_rewards")
	}
}

// ── Reward type constants ────────────────────────────────────────────────

func TestRewardTypeConstants_AllDistinct(t *testing.T) {
	types := []string{
		RewardTypeLiveInviteBid, RewardTypeLiveInviteWin,
		RewardTypeWinShareAttribution, RewardTypeFirstBidCashback,
		RewardTypeStreakBonus, RewardTypeGroupEntry,
	}
	seen := make(map[string]bool)
	for _, tp := range types {
		if tp == "" {
			t.Fatal("Reward type should not be empty")
		}
		if seen[tp] {
			t.Fatalf("Duplicate reward type: %s", tp)
		}
		seen[tp] = true
	}
}

// ── GrowthMetrics shape ──────────────────────────────────────────────────

func TestGrowthMetricsShape(t *testing.T) {
	m := GrowthMetrics{
		InvitesSent:      100,
		InvitesJoined:    40,
		InvitesBidded:    15,
		InvitesWon:       3,
		ViralCoefficient: 0.4,
	}
	if m.InvitesSent != 100 {
		t.Fatal("InvitesSent should persist")
	}
	// K-factor < 1 → not self-sustaining, > 1 → viral
	if m.ViralCoefficient < 0 || m.ViralCoefficient > 10 {
		t.Fatal("K-factor should be in realistic range")
	}
}
