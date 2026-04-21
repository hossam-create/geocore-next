package waitlist

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// JoinResult is the enriched payload returned to the caller on join.
type JoinResult struct {
	User          *WaitlistUser
	IsNew         bool
	ShareLink     string
	Message       string
	NextMilestone string
}

// UXData is the enriched position payload returned on status requests.
type UXData struct {
	Position        int     `json:"position"`
	PeopleBehind    int64   `json:"people_behind"`
	MovedToday      int     `json:"moved_today"`
	ProgressPercent float64 `json:"progress_percent"`
	NextUnlock      string  `json:"next_unlock"`
	ReferralCount   int     `json:"referral_count"`
	Status          Status  `json:"status"`
}

// Join registers an email on the waitlist.
// Returns (result, nil) whether the email is new or already present.
func Join(db *gorm.DB, rdb *redis.Client, email, referredBy, ip, deviceID, baseURL string) (*JoinResult, error) {
	// Return existing record without error — idempotent.
	var existing WaitlistUser
	if err := db.Where("email = ?", email).First(&existing).Error; err == nil {
		return buildResult(&existing, false, baseURL), nil
	}

	// Anti-abuse: rapid IP detection.
	suspicious := CheckRapidJoin(context.Background(), rdb, ip)

	// Anti-gaming: block same-device referral.
	validRef := referredBy
	if deviceID != "" && referredBy != "" {
		var sameDevice WaitlistUser
		if db.Where("device_id = ? AND referral_code = ?", deviceID, referredBy).First(&sameDevice).Error == nil {
			slog.Warn("waitlist: same-device referral rejected", "device", deviceID)
			validRef = "" // invalidate referral, still allow join
		}
	}

	// Assign next sequential position.
	var lastPos int
	db.Model(&WaitlistUser{}).Select("COALESCE(MAX(position), 0)").Scan(&lastPos)
	pos := lastPos + 1

	// Unique code with up to 5 collision retries.
	code := GenerateReferralCode()
	for i := 0; i < 5; i++ {
		var check WaitlistUser
		if db.Where("referral_code = ?", code).First(&check).Error != nil {
			break
		}
		code = GenerateReferralCode()
	}

	user := WaitlistUser{
		Email:            email,
		Position:         pos,
		PreviousPosition: pos,
		ReferralCode:     code,
		PriorityScore:    float64(pos),
		Status:           StatusWaiting,
		IPAddress:        ip,
		DeviceID:         deviceID,
	}
	if validRef != "" {
		user.ReferredBy = &validRef
	}
	if suspicious {
		user.PriorityScore = float64(pos) + 5000
		slog.Warn("waitlist: suspicious rapid join", "ip", ip, "email", email)
	}

	if err := db.Create(&user).Error; err != nil {
		return nil, err
	}

	// Boost inviter asynchronously (only if referral was valid).
	if validRef != "" {
		go ApplyReferralBoost(db, validRef, false)
	}

	slog.Info("waitlist: new signup", "email", email, "position", pos, "ref", validRef)
	return buildResult(&user, true, baseURL), nil
}

// ApplyReferralBoost increments referral_count and tightens priority_score.
// trusted=true applies an extra -5 bonus (e.g. from a verified user referral).
func ApplyReferralBoost(db *gorm.DB, inviterCode string, trusted bool) {
	var inviter WaitlistUser
	if err := db.Where("referral_code = ?", inviterCode).First(&inviter).Error; err != nil {
		return
	}
	if inviter.Status != StatusWaiting {
		return
	}
	inviter.ReferralCount++
	// priority_score = base_position - (referral_count * 3) - (trusted_referrals * 5)
	trustedBonus := 0.0
	if trusted {
		trustedBonus = 5
	}
	inviter.PriorityScore = float64(inviter.Position) -
		float64(inviter.ReferralCount)*3 -
		trustedBonus
	if inviter.PriorityScore < 1 {
		inviter.PriorityScore = 1
	}
	db.Save(&inviter)
	slog.Info("waitlist: referral boost", "code", inviterCode,
		"referral_count", inviter.ReferralCount, "priority_score", inviter.PriorityScore)
	go RecalculatePositions(db)
}

// RecalculatePositions re-ranks all waiting users by priority_score ASC.
// Snapshots PreviousPosition before writing new positions (used for moved_today).
func RecalculatePositions(db *gorm.DB) {
	var users []WaitlistUser
	db.Where("status = ?", StatusWaiting).
		Order("priority_score ASC, created_at ASC").
		Find(&users)
	for i, u := range users {
		newPos := i + 1
		db.Model(&WaitlistUser{}).
			Where("id = ?", u.ID).
			Updates(map[string]interface{}{
				"previous_position": u.Position,
				"position":          newPos,
			})
	}
	slog.Info("waitlist: positions recalculated", "count", len(users))
}

// ReleaseInvites marks the top n waiting users as invited.
// Enforces the daily scarcity limit before proceeding.
func ReleaseInvites(db *gorm.DB, n int) ([]WaitlistUser, error) {
	if err := CheckAndRecordRelease(db, n); err != nil {
		return nil, err
	}
	var top []WaitlistUser
	if err := db.Where("status = ?", StatusWaiting).
		Order("position ASC").Limit(n).Find(&top).Error; err != nil {
		return nil, err
	}
	for i := range top {
		top[i].Status = StatusInvited
		db.Save(&top[i])
		// Seed onboarding record for the conversion bridge.
		db.Create(&OnboardingState{UserEmail: top[i].Email})
		slog.Info("waitlist: invited", "email", top[i].Email, "position", top[i].Position)
	}
	return top, nil
}

// CheckRapidJoin returns true when >5 signups arrive from the same IP within 1 hour.
func CheckRapidJoin(ctx context.Context, rdb *redis.Client, ip string) bool {
	if rdb == nil || ip == "" {
		return false
	}
	key := fmt.Sprintf("waitlist:ip:%s", ip)
	n, _ := rdb.Incr(ctx, key).Result()
	if n == 1 {
		rdb.Expire(ctx, key, time.Hour)
	}
	return n > 5
}

// FlagUser marks a user as flagged and sends them to the back of the queue.
func FlagUser(db *gorm.DB, userID string, reason string) {
	db.Model(&WaitlistUser{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"status":         StatusFlagged,
		"priority_score": 9999999,
	})
	slog.Warn("waitlist: user flagged", "user_id", userID, "reason", reason)
}

// Stats returns aggregate social-proof metrics for the landing page.
func Stats(db *gorm.DB) map[string]interface{} {
	var total, joinedToday, invitesSent int64
	var avgReferrals float64

	db.Model(&WaitlistUser{}).Count(&total)
	db.Model(&WaitlistUser{}).
		Where("created_at >= ?", time.Now().Truncate(24*time.Hour)).
		Count(&joinedToday)
	db.Model(&WaitlistUser{}).Where("status = ?", StatusInvited).Count(&invitesSent)
	db.Model(&WaitlistUser{}).Select("COALESCE(AVG(referral_count), 0)").Scan(&avgReferrals)

	return map[string]interface{}{
		"total_users":   total,
		"joined_today":  joinedToday,
		"invites_sent":  invitesSent,
		"avg_referrals": avgReferrals,
	}
}

// UXPayload builds rich position data for the status endpoint.
func UXPayload(db *gorm.DB, user *WaitlistUser) *UXData {
	var total int64
	db.Model(&WaitlistUser{}).Where("status = ?", StatusWaiting).Count(&total)
	peopleBehind := total - int64(user.Position)
	if peopleBehind < 0 {
		peopleBehind = 0
	}
	movedToday := user.PreviousPosition - user.Position
	if movedToday < 0 {
		movedToday = 0
	}
	progress := 0.0
	if total > 0 {
		progress = float64(total-int64(user.Position)) / float64(total) * 100
	}
	return &UXData{
		Position:        user.Position,
		PeopleBehind:    peopleBehind,
		MovedToday:      movedToday,
		ProgressPercent: progress,
		NextUnlock:      NextUnlock(user.Position),
		ReferralCount:   user.ReferralCount,
		Status:          user.Status,
	}
}

// NextUnlock returns the next benefit threshold for the user.
func NextUnlock(position int) string {
	switch {
	case position <= 100:
		return "Early access unlocked — you're in the top 100!"
	case position <= 500:
		return fmt.Sprintf("%d more spots to early access (top 500)", position-100)
	case position <= 1000:
		return fmt.Sprintf("%d more spots to priority onboarding (top 1000)", position-500)
	default:
		return fmt.Sprintf("%d spots until priority onboarding", position-1000)
	}
}

// BoostByActivity applies a dynamic priority bonus based on trust score and onboarding.
// Called externally when a user completes a meaningful action post-invite.
func BoostByActivity(db *gorm.DB, email string, trustScore float64) {
	var user WaitlistUser
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		return
	}
	// Bonus: trust score contribution (0–10) + onboarding complete (+5).
	bonus := trustScore / 10
	var ob OnboardingState
	if db.Where("user_email = ? AND is_complete = true", email).First(&ob).Error == nil {
		bonus += 5
	}
	user.PriorityScore -= bonus
	if user.PriorityScore < 1 {
		user.PriorityScore = 1
	}
	db.Save(&user)
	slog.Info("waitlist: activity boost", "email", email, "bonus", bonus, "score", user.PriorityScore)
	go RecalculatePositions(db)
}

// NextMilestone returns contextual copy based on the user's current position.
func NextMilestone(position int) string {
	switch {
	case position <= 100:
		return "You're in the top 100! Early access is coming very soon."
	case position <= 500:
		return "Top 500 gets early access."
	case position <= 1000:
		return "Top 1000 gets priority onboarding."
	default:
		return "Invite friends to move up faster."
	}
}

// buildResult assembles a JoinResult from a WaitlistUser.
func buildResult(u *WaitlistUser, isNew bool, baseURL string) *JoinResult {
	msg := "You're already on the waitlist."
	if isNew {
		msg = "You're on the waitlist! Invite friends to move up."
	}
	return &JoinResult{
		User:          u,
		IsNew:         isNew,
		ShareLink:     fmt.Sprintf("%s/waitlist?ref=%s", baseURL, u.ReferralCode),
		Message:       msg,
		NextMilestone: NextMilestone(u.Position),
	}
}
