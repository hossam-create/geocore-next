package livestream

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/freeze"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Sprint 15: Live Viral Growth Engine
//
// LIVE-commerce-specific viral loops (the generic user→user referral loop
// already exists under internal/growth/referral.go and is reused here for the
// first-order reward path).
//
// Loops:
//   1. Live Invite        (viewer → viewer)
//   2. Winner Brag        (winner share card → new users)
//   3. Watcher → Bidder   (first-bid cashback)
//   4. Group Entry        (friends join same session → discounts + badge)
//   5. Streak Rewards     (consecutive live joins / bids)
//   6. Viral Triggers     (auto-share suggestions on HOT items)
//
// All rewards go through a single idempotent ledger: GrowthReward
// (reconciliation to wallet is off-band, same pattern as LiveStreamerEarning).
// ════════════════════════════════════════════════════════════════════════════

// ── Feature Flags ──────────────────────────────────────────────────────────

func IsLiveInvitesEnabled() bool   { return envBoolDefault("ENABLE_LIVE_INVITES", true) }
func IsShareRewardsEnabled() bool  { return envBoolDefault("ENABLE_SHARE_REWARDS", true) }
func IsStreaksEnabled() bool       { return envBoolDefault("ENABLE_STREAKS", true) }
func IsGroupBuyEnabled() bool      { return envBoolDefault("ENABLE_GROUP_BUY", true) }
func IsLiveReferralsEnabled() bool { return envBoolDefault("ENABLE_REFERRALS", true) }

func envBoolDefault(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v != "false" && v != "0"
}

// ── Reward Amounts (cents) ─────────────────────────────────────────────────

const (
	RewardLiveInviteBid       int64 = 200   // 2 EGP when invitee places first bid
	RewardLiveInviteWin       int64 = 2_000 // 20 EGP when invitee wins
	RewardWinShareAttribution int64 = 500   // 5 EGP when share link brings new user
	RewardFirstBidCashback    int64 = 100   // 1 EGP on first ever live bid
	RewardStreakDay3          int64 = 500   // 5 EGP at 3-day streak
	RewardStreakDay7          int64 = 2_000 // 20 EGP at 7-day streak

	GroupEntryFeeDiscountPct = 0.50 // 50% off entry fee for group of ≥3
	GroupMinSize             = 3    // minimum group size to unlock perks
)

// Max daily rewards per user (fraud cap)
const maxDailyRewardCents int64 = 50_000 // 500 EGP/day

// ════════════════════════════════════════════════════════════════════════════
// Models
// ════════════════════════════════════════════════════════════════════════════

// LiveInvite — viewer-to-viewer invite to a live session.
type LiveInvite struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	InviterID  uuid.UUID  `gorm:"type:uuid;not null;index"                        json:"inviter_id"`
	SessionID  uuid.UUID  `gorm:"type:uuid;not null;index"                        json:"session_id"`
	InviteCode string     `gorm:"size:16;not null;uniqueIndex"                    json:"invite_code"`
	InviteeID  *uuid.UUID `gorm:"type:uuid;index"                                 json:"invitee_id,omitempty"`
	Joined     bool       `gorm:"not null;default:false;index"                    json:"joined"`
	BidPlaced  bool       `gorm:"not null;default:false;index"                    json:"bid_placed"`
	Won        bool       `gorm:"not null;default:false"                          json:"won"`
	JoinedAt   *time.Time `                                                        json:"joined_at,omitempty"`
	BidAt      *time.Time `                                                        json:"bid_at,omitempty"`
	WonAt      *time.Time `                                                        json:"won_at,omitempty"`
	ClientIP   string     `gorm:"size:45"                                         json:"-"`
	CreatedAt  time.Time  `gorm:"not null;index"                                  json:"created_at"`
}

func (LiveInvite) TableName() string { return "live_invites" }

// WinShare — share card generated when a user wins an auction.
type WinShare struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index"                        json:"user_id"`
	ItemID          uuid.UUID `gorm:"type:uuid;not null;index"                        json:"item_id"`
	SessionID       uuid.UUID `gorm:"type:uuid;not null;index"                        json:"session_id"`
	ShareCode       string    `gorm:"size:16;not null;uniqueIndex"                    json:"share_code"`
	PriceCents      int64     `gorm:"not null"                                        json:"price_cents"`
	NewUsersBrought int       `gorm:"not null;default:0"                              json:"new_users_brought"`
	CreatedAt       time.Time `gorm:"not null"                                        json:"created_at"`
}

func (WinShare) TableName() string { return "live_win_shares" }

// GroupInvite — creator sends a group invite for a session.
type GroupInvite struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SessionID   uuid.UUID `gorm:"type:uuid;not null;index"                        json:"session_id"`
	CreatorID   uuid.UUID `gorm:"type:uuid;not null;index"                        json:"creator_id"`
	InviteCode  string    `gorm:"size:16;not null;uniqueIndex"                    json:"invite_code"`
	MemberCount int       `gorm:"not null;default:1"                              json:"member_count"`
	Badge       string    `gorm:"size:32;default:''"                              json:"badge,omitempty"`
	CreatedAt   time.Time `gorm:"not null"                                        json:"created_at"`
}

func (GroupInvite) TableName() string { return "live_group_invites" }

// GroupMember — a user that joined a group invite.
type GroupMember struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	GroupID  uuid.UUID `gorm:"type:uuid;not null;index:idx_group_user,unique"  json:"group_id"`
	UserID   uuid.UUID `gorm:"type:uuid;not null;index:idx_group_user,unique"  json:"user_id"`
	JoinedAt time.Time `gorm:"not null"                                        json:"joined_at"`
}

func (GroupMember) TableName() string { return "live_group_members" }

// UserStreak — tracks consecutive days of live activity per type.
type UserStreak struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID           uuid.UUID `gorm:"type:uuid;not null;index:idx_user_streak_type,unique" json:"user_id"`
	Type             string    `gorm:"size:20;not null;index:idx_user_streak_type,unique"   json:"type"` // live_join, bid
	CurrentStreak    int       `gorm:"not null;default:0"                              json:"current_streak"`
	LongestStreak    int       `gorm:"not null;default:0"                              json:"longest_streak"`
	LastActivityDate time.Time `gorm:"not null"                                        json:"last_activity_date"`
	UpdatedAt        time.Time `gorm:"not null"                                        json:"updated_at"`
}

func (UserStreak) TableName() string { return "live_user_streaks" }

// GrowthReward — idempotent ledger of viral rewards owed to users.
// Unique (user_id, reward_type, reference_id) guarantees no double credit.
type GrowthReward struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID        uuid.UUID  `gorm:"type:uuid;not null;index"                        json:"user_id"`
	RewardType    string     `gorm:"size:40;not null;index:idx_reward_unique,unique" json:"reward_type"`
	ReferenceType string     `gorm:"size:40;not null"                                json:"reference_type"`
	ReferenceID   string     `gorm:"size:64;not null;index:idx_reward_unique,unique" json:"reference_id"`
	AmountCents   int64      `gorm:"not null"                                        json:"amount_cents"`
	Status        string     `gorm:"size:20;not null;default:'pending';index"        json:"status"` // pending, paid, voided
	Metadata      string     `gorm:"type:text"                                       json:"metadata,omitempty"`
	CreatedAt     time.Time  `gorm:"not null;index"                                  json:"created_at"`
	PaidAt        *time.Time `                                                       json:"paid_at,omitempty"`
}

// Composite unique: (user_id, reward_type, reference_id) — add via AutoMigrate hook.
func (GrowthReward) TableName() string { return "live_growth_rewards" }

// Reward type constants.
const (
	RewardTypeLiveInviteBid       = "live_invite_bid"
	RewardTypeLiveInviteWin       = "live_invite_win"
	RewardTypeWinShareAttribution = "win_share_attribution"
	RewardTypeFirstBidCashback    = "first_bid_cashback"
	RewardTypeStreakBonus         = "streak_bonus"
	RewardTypeGroupEntry          = "group_entry"
)

// ════════════════════════════════════════════════════════════════════════════
// Code Generation + Ledger Helper
// ════════════════════════════════════════════════════════════════════════════

// generateShortCode produces a URL-safe 10-char uppercase code.
func generateShortCode() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	s := base32.StdEncoding.EncodeToString(b)
	return strings.TrimRight(s, "=")[:10]
}

// GrantReward atomically persists a reward if not already granted (idempotent).
// Returns (granted, err) — granted=false if the row already exists.
func GrantReward(db *gorm.DB, userID uuid.UUID, rewardType, refType, refID string, amountCents int64) (bool, error) {
	if amountCents <= 0 {
		return false, nil
	}
	// Daily cap check
	var dailyTotal int64
	dayStart := time.Now().Truncate(24 * time.Hour)
	db.Model(&GrowthReward{}).
		Where("user_id = ? AND created_at >= ?", userID, dayStart).
		Select("COALESCE(SUM(amount_cents), 0)").Scan(&dailyTotal)
	if dailyTotal+amountCents > maxDailyRewardCents {
		slog.Warn("growth: daily reward cap reached", "user_id", userID, "daily_total", dailyTotal)
		return false, nil
	}

	row := GrowthReward{
		UserID:        userID,
		RewardType:    rewardType,
		ReferenceType: refType,
		ReferenceID:   refID,
		AmountCents:   amountCents,
		Status:        "pending",
		CreatedAt:     time.Now(),
	}
	err := db.Create(&row).Error
	if err != nil {
		// Unique-violation = idempotent replay, not an error
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ════════════════════════════════════════════════════════════════════════════
// 1. Live Invite Loop (viewer → viewer)
// ════════════════════════════════════════════════════════════════════════════

// CreateLiveInvite generates a shareable invite code for a live session.
func CreateLiveInvite(db *gorm.DB, inviterID, sessionID uuid.UUID) (*LiveInvite, error) {
	if !IsLiveInvitesEnabled() {
		return nil, fmt.Errorf("live invites disabled")
	}
	// Verify session exists
	var sess Session
	if err := db.Where("id = ?", sessionID).First(&sess).Error; err != nil {
		return nil, fmt.Errorf("session not found")
	}
	inv := &LiveInvite{
		InviterID:  inviterID,
		SessionID:  sessionID,
		InviteCode: generateShortCode(),
		CreatedAt:  time.Now(),
	}
	if err := db.Create(inv).Error; err != nil {
		return nil, err
	}
	return inv, nil
}

// TrackLiveInvite records an invitee joining via an invite code.
// Idempotent per (invite_code, invitee_id).
func TrackLiveInvite(db *gorm.DB, code string, inviteeID uuid.UUID, clientIP string) (*LiveInvite, error) {
	if !IsLiveInvitesEnabled() {
		return nil, nil
	}
	var inv LiveInvite
	if err := db.Where("invite_code = ?", code).First(&inv).Error; err != nil {
		return nil, fmt.Errorf("invite not found")
	}
	// Anti-fraud: can't accept own invite
	if inv.InviterID == inviteeID {
		return nil, fmt.Errorf("self-invite blocked")
	}
	// Already tracked?
	if inv.InviteeID != nil && *inv.InviteeID == inviteeID && inv.Joined {
		return &inv, nil
	}
	// Anti-fraud: inviter and invitee from same IP
	if clientIP != "" && inv.ClientIP != "" && clientIP == inv.ClientIP {
		slog.Warn("growth: same-IP invite blocked", "code", code, "ip", clientIP)
		return nil, fmt.Errorf("same-IP abuse blocked")
	}
	now := time.Now()
	updates := map[string]interface{}{
		"invitee_id": inviteeID,
		"joined":     true,
		"joined_at":  now,
	}
	if err := db.Model(&inv).Updates(updates).Error; err != nil {
		return nil, err
	}
	inv.InviteeID = &inviteeID
	inv.Joined = true
	inv.JoinedAt = &now
	return &inv, nil
}

// RewardInviterOnBid fires when an invitee places their first bid on the invited session.
func RewardInviterOnBid(db *gorm.DB, inviteeID, sessionID uuid.UUID) {
	if !IsLiveInvitesEnabled() {
		return
	}
	var inv LiveInvite
	if err := db.Where("invitee_id = ? AND session_id = ? AND joined = ?", inviteeID, sessionID, true).
		First(&inv).Error; err != nil {
		return
	}
	if inv.BidPlaced {
		return // already rewarded
	}
	now := time.Now()
	db.Model(&inv).Updates(map[string]interface{}{"bid_placed": true, "bid_at": now})
	_, _ = GrantReward(db, inv.InviterID, RewardTypeLiveInviteBid, "live_invite", inv.ID.String(), RewardLiveInviteBid)
	freeze.LogAudit(db, "live_invite_bid_reward", inv.InviterID, inv.ID,
		fmt.Sprintf("invitee=%s session=%s cents=%d", inviteeID, sessionID, RewardLiveInviteBid))
}

// RewardInviterOnWin fires when an invitee wins an item on the invited session.
func RewardInviterOnWin(db *gorm.DB, inviteeID, sessionID uuid.UUID) {
	if !IsLiveInvitesEnabled() {
		return
	}
	var inv LiveInvite
	if err := db.Where("invitee_id = ? AND session_id = ? AND joined = ?", inviteeID, sessionID, true).
		First(&inv).Error; err != nil {
		return
	}
	if inv.Won {
		return
	}
	now := time.Now()
	db.Model(&inv).Updates(map[string]interface{}{"won": true, "won_at": now})
	_, _ = GrantReward(db, inv.InviterID, RewardTypeLiveInviteWin, "live_invite", inv.ID.String(), RewardLiveInviteWin)
}

// ════════════════════════════════════════════════════════════════════════════
// 2. Winner Brag Loop (social proof)
// ════════════════════════════════════════════════════════════════════════════

// CreateWinShare generates a share card for a won item.
func CreateWinShare(db *gorm.DB, userID, sessionID, itemID uuid.UUID, priceCents int64) (*WinShare, error) {
	if !IsShareRewardsEnabled() {
		return nil, fmt.Errorf("share rewards disabled")
	}
	ws := &WinShare{
		UserID:     userID,
		SessionID:  sessionID,
		ItemID:     itemID,
		ShareCode:  generateShortCode(),
		PriceCents: priceCents,
		CreatedAt:  time.Now(),
	}
	if err := db.Create(ws).Error; err != nil {
		return nil, err
	}
	return ws, nil
}

// AttributeShareJoin records that a new user joined via a share card.
// Grants the winner a small credit (idempotent per new_user).
func AttributeShareJoin(db *gorm.DB, shareCode string, newUserID uuid.UUID) error {
	if !IsShareRewardsEnabled() {
		return nil
	}
	var ws WinShare
	if err := db.Where("share_code = ?", shareCode).First(&ws).Error; err != nil {
		return fmt.Errorf("share card not found")
	}
	// Block self-attribution
	if ws.UserID == newUserID {
		return nil
	}
	granted, err := GrantReward(db, ws.UserID, RewardTypeWinShareAttribution, "win_share",
		ws.ID.String()+":"+newUserID.String(), RewardWinShareAttribution)
	if err != nil {
		return err
	}
	if granted {
		db.Model(&ws).UpdateColumn("new_users_brought", gorm.Expr("new_users_brought + 1"))
	}
	return nil
}

// ════════════════════════════════════════════════════════════════════════════
// 3. Watcher → Bidder Conversion Loop
// ════════════════════════════════════════════════════════════════════════════

// GrantFirstBidCashback awards 1 EGP on a user's first ever live bid.
// Idempotent: reference_id = user's own UUID (can only happen once).
func GrantFirstBidCashback(db *gorm.DB, userID uuid.UUID) {
	if !IsLiveReferralsEnabled() {
		return
	}
	// Check if user already has any bids (skip on re-entry)
	var count int64
	db.Model(&LiveBid{}).Where("user_id = ?", userID).Count(&count)
	if count > 1 {
		return // not the first bid
	}
	_, _ = GrantReward(db, userID, RewardTypeFirstBidCashback, "user", userID.String(), RewardFirstBidCashback)
}

// ════════════════════════════════════════════════════════════════════════════
// 4. Group Entry Loop
// ════════════════════════════════════════════════════════════════════════════

// CreateGroupInvite creates a group invite for a session.
func CreateGroupInvite(db *gorm.DB, creatorID, sessionID uuid.UUID) (*GroupInvite, error) {
	if !IsGroupBuyEnabled() {
		return nil, fmt.Errorf("group buy disabled")
	}
	g := &GroupInvite{
		CreatorID:   creatorID,
		SessionID:   sessionID,
		InviteCode:  generateShortCode(),
		MemberCount: 1,
		CreatedAt:   time.Now(),
	}
	if err := db.Create(g).Error; err != nil {
		return nil, err
	}
	// Add creator as first member
	_ = db.Create(&GroupMember{GroupID: g.ID, UserID: creatorID, JoinedAt: time.Now()}).Error
	return g, nil
}

// JoinGroupInvite adds a user to a group. Returns updated group.
func JoinGroupInvite(db *gorm.DB, code string, userID uuid.UUID) (*GroupInvite, error) {
	if !IsGroupBuyEnabled() {
		return nil, fmt.Errorf("group buy disabled")
	}
	var g GroupInvite
	if err := db.Where("invite_code = ?", code).First(&g).Error; err != nil {
		return nil, fmt.Errorf("group not found")
	}
	// Try insert (unique constraint guards against dupes)
	member := GroupMember{GroupID: g.ID, UserID: userID, JoinedAt: time.Now()}
	if err := db.Create(&member).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE") {
			return &g, nil // already a member
		}
		return nil, err
	}
	// Bump count
	db.Model(&g).UpdateColumn("member_count", gorm.Expr("member_count + 1"))
	g.MemberCount++

	// Unlock badge at group-min
	if g.MemberCount >= GroupMinSize && g.Badge == "" {
		db.Model(&g).Update("badge", "group_buyers")
		g.Badge = "group_buyers"
	}
	return &g, nil
}

// GroupEntryDiscount returns the discount cents the user should receive on the
// session's entry fee if they joined via a group of ≥ GroupMinSize.
// Returns 0 if no group or group is too small.
func GroupEntryDiscount(db *gorm.DB, userID, sessionID uuid.UUID, entryFeeCents int64) int64 {
	if !IsGroupBuyEnabled() || entryFeeCents <= 0 {
		return 0
	}
	var g GroupInvite
	err := db.Table("live_group_invites AS g").
		Joins("JOIN live_group_members m ON m.group_id = g.id").
		Where("g.session_id = ? AND m.user_id = ?", sessionID, userID).
		Select("g.*").Scan(&g).Error
	if err != nil || g.ID == uuid.Nil || g.MemberCount < GroupMinSize {
		return 0
	}
	return int64(float64(entryFeeCents) * GroupEntryFeeDiscountPct)
}

// ════════════════════════════════════════════════════════════════════════════
// 5. Streak & Reward System
// ════════════════════════════════════════════════════════════════════════════

// UpdateStreak advances a user's streak counter for a given activity type.
// Automatically grants streak-milestone rewards at days 3 and 7.
func UpdateStreak(db *gorm.DB, userID uuid.UUID, streakType string) *UserStreak {
	if !IsStreaksEnabled() {
		return nil
	}
	today := time.Now().Truncate(24 * time.Hour)
	var s UserStreak
	err := db.Where("user_id = ? AND type = ?", userID, streakType).First(&s).Error
	if err != nil {
		// First activity ever
		s = UserStreak{
			UserID:           userID,
			Type:             streakType,
			CurrentStreak:    1,
			LongestStreak:    1,
			LastActivityDate: today,
			UpdatedAt:        time.Now(),
		}
		_ = db.Create(&s).Error
		return &s
	}
	last := s.LastActivityDate.Truncate(24 * time.Hour)
	daysSince := int(today.Sub(last).Hours() / 24)
	switch daysSince {
	case 0:
		// Same day — no change
		return &s
	case 1:
		s.CurrentStreak++
	default:
		s.CurrentStreak = 1 // gap broke the streak
	}
	if s.CurrentStreak > s.LongestStreak {
		s.LongestStreak = s.CurrentStreak
	}
	s.LastActivityDate = today
	s.UpdatedAt = time.Now()
	_ = db.Save(&s).Error

	// Milestone rewards (idempotent via ref_id = user+type+day)
	refBase := fmt.Sprintf("%s:%s:%s", userID, streakType, today.Format("20060102"))
	switch s.CurrentStreak {
	case 3:
		_, _ = GrantReward(db, userID, RewardTypeStreakBonus, "streak_3", refBase, RewardStreakDay3)
	case 7:
		_, _ = GrantReward(db, userID, RewardTypeStreakBonus, "streak_7", refBase, RewardStreakDay7)
	}
	return &s
}

// ════════════════════════════════════════════════════════════════════════════
// 6. Viral Auto-Triggers
// ════════════════════════════════════════════════════════════════════════════

// MaybeTriggerShareSuggestion broadcasts a "share this live!" toast when an
// item is VERY_HOT or has many viewers but low bidders.
// Throttled via Redis (one per session per 2min).
func (h *LiveAuctionHandler) MaybeTriggerShareSuggestion(sessionID, itemID uuid.UUID, urgency UrgencyState, viewers, activeBidders int) {
	if !IsLiveInvitesEnabled() || h.rdb == nil {
		return
	}
	shouldFire := urgency == UrgencyVeryHot ||
		(viewers >= 20 && activeBidders <= 2)
	if !shouldFire {
		return
	}
	// Throttle: one share-suggestion per session per 2min
	key := fmt.Sprintf("live:viral:share_suggest:%s", sessionID)
	ok, err := h.rdb.SetNX(context.Background(), key, "1", 2*time.Minute).Result()
	if err != nil || !ok {
		return
	}
	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:           EventToast,
		SessionID:       sessionID.String(),
		ItemID:          itemID.String(),
		Message:         "🔥 This is trending — share it with friends to unlock rewards!",
		Icon:            "📢",
		SuggestedAction: "share_live",
		ActionLabel:     "Share live",
	})
}

// ════════════════════════════════════════════════════════════════════════════
// 7. Growth Metrics (admin)
// ════════════════════════════════════════════════════════════════════════════

// GrowthMetrics is the admin payload returned from /admin/growth/metrics.
type GrowthMetrics struct {
	InvitesSent         int64   `json:"invites_sent"`
	InvitesJoined       int64   `json:"invites_joined"`
	InvitesBidded       int64   `json:"invites_bidded"`
	InvitesWon          int64   `json:"invites_won"`
	WinSharesCreated    int64   `json:"win_shares_created"`
	WinSharesAttributed int64   `json:"win_shares_attributed"`
	GroupsCreated       int64   `json:"groups_created"`
	GroupsUnlocked      int64   `json:"groups_unlocked"`
	StreaksActive       int64   `json:"streaks_active"`
	RewardsPendingCents int64   `json:"rewards_pending_cents"`
	RewardsPaidCents    int64   `json:"rewards_paid_cents"`
	ViralCoefficient    float64 `json:"viral_coefficient"` // K = invites_joined / inviter_users
}

// ComputeGrowthMetrics aggregates all viral counters.
func ComputeGrowthMetrics(db *gorm.DB) GrowthMetrics {
	m := GrowthMetrics{}

	db.Model(&LiveInvite{}).Count(&m.InvitesSent)
	db.Model(&LiveInvite{}).Where("joined = ?", true).Count(&m.InvitesJoined)
	db.Model(&LiveInvite{}).Where("bid_placed = ?", true).Count(&m.InvitesBidded)
	db.Model(&LiveInvite{}).Where("won = ?", true).Count(&m.InvitesWon)

	db.Model(&WinShare{}).Count(&m.WinSharesCreated)
	var attributed int64
	db.Model(&WinShare{}).Select("COALESCE(SUM(new_users_brought), 0)").Scan(&attributed)
	m.WinSharesAttributed = attributed

	db.Model(&GroupInvite{}).Count(&m.GroupsCreated)
	db.Model(&GroupInvite{}).Where("member_count >= ?", GroupMinSize).Count(&m.GroupsUnlocked)

	// Active streaks: last activity within 2 days
	cutoff := time.Now().Add(-48 * time.Hour)
	db.Model(&UserStreak{}).Where("last_activity_date >= ? AND current_streak >= 2", cutoff).Count(&m.StreaksActive)

	// Reward totals
	var pending, paid int64
	db.Model(&GrowthReward{}).Where("status = ?", "pending").
		Select("COALESCE(SUM(amount_cents), 0)").Scan(&pending)
	db.Model(&GrowthReward{}).Where("status = ?", "paid").
		Select("COALESCE(SUM(amount_cents), 0)").Scan(&paid)
	m.RewardsPendingCents = pending
	m.RewardsPaidCents = paid

	// K-factor: invites_joined / distinct inviter users
	var distinctInviters int64
	db.Model(&LiveInvite{}).Distinct("inviter_id").Count(&distinctInviters)
	if distinctInviters > 0 {
		m.ViralCoefficient = float64(m.InvitesJoined) / float64(distinctInviters)
	}
	return m
}
