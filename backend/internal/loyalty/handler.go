package loyalty

import (
	"crypto/rand"
	"encoding/hex"
	"math"
	"strconv"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// GetAccount returns the user's loyalty account
func (h *Handler) GetAccount(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))

	var account LoyaltyAccount
	if err := h.db.First(&account, "user_id = ?", userID).Error; err != nil {
		// Create account if not exists
		account = LoyaltyAccount{
			ID:             uuid.New(),
			UserID:         userID,
			CurrentPoints:  0,
			LifetimePoints: 0,
			Tier:           TierBronze,
			ReferralCode:   GenerateReferralCode(),
			CreatedAt:      time.Now(),
		}
		h.db.Create(&account)
	}

	nextTier, pointsNeeded := GetNextTier(account.Tier, account.LifetimePoints)
	multiplier := TierMultipliers[account.Tier]

	response.OK(c, gin.H{
		"account":       account,
		"next_tier":     nextTier,
		"points_needed": pointsNeeded,
		"multiplier":    multiplier,
	})
}

// GetTransactions returns points transaction history
func (h *Handler) GetTransactions(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage := 20

	var account LoyaltyAccount
	if err := h.db.First(&account, "user_id = ?", userID).Error; err != nil {
		response.NotFound(c, "Loyalty account")
		return
	}

	var transactions []PointsTransaction
	var total int64

	h.db.Model(&PointsTransaction{}).Where("account_id = ?", account.ID).Count(&total)
	h.db.Where("account_id = ?", account.ID).
		Offset((page - 1) * perPage).Limit(perPage).
		Order("created_at DESC").Find(&transactions)

	response.OKMeta(c, transactions, gin.H{"total": total, "page": page})
}

// EarnPoints adds points to the account
func (h *Handler) EarnPoints(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))

	var req struct {
		Action      PointsAction `json:"action" binding:"required"`
		BasePoints  int          `json:"base_points" binding:"required,min=1"`
		ReferenceID *string      `json:"reference_id"`
		Description string       `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var account LoyaltyAccount
	if err := h.db.First(&account, "user_id = ?", userID).Error; err != nil {
		response.NotFound(c, "Loyalty account")
		return
	}

	// Apply tier multiplier
	multiplier := TierMultipliers[account.Tier]
	earnedPoints := int(math.Round(float64(req.BasePoints) * multiplier))

	// Update account
	account.CurrentPoints += earnedPoints
	account.LifetimePoints += earnedPoints

	// Check for tier upgrade
	newTier := GetTierForPoints(account.LifetimePoints)
	tierUpgraded := newTier != account.Tier
	if tierUpgraded {
		account.Tier = newTier
		// Set tier expiry (1 year)
		expiry := time.Now().AddDate(1, 0, 0)
		account.TierExpiresAt = &expiry
	}

	h.db.Save(&account)

	// Create transaction
	expiresAt := time.Now().AddDate(1, 0, 0) // Points expire in 1 year
	transaction := PointsTransaction{
		ID:          uuid.New(),
		AccountID:   account.ID,
		Action:      req.Action,
		Points:      earnedPoints,
		Balance:     account.CurrentPoints,
		Multiplier:  multiplier,
		ReferenceID: req.ReferenceID,
		Description: req.Description,
		ExpiresAt:   &expiresAt,
		CreatedAt:   time.Now(),
	}
	h.db.Create(&transaction)

	result := gin.H{
		"points_earned": earnedPoints,
		"multiplier":    multiplier,
		"new_balance":   account.CurrentPoints,
		"tier":          account.Tier,
	}
	if tierUpgraded {
		result["tier_upgraded"] = true
		result["new_tier"] = newTier
	}

	response.OK(c, result)
}

// ClaimDailyBonus claims daily login bonus
func (h *Handler) ClaimDailyBonus(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))

	var account LoyaltyAccount
	if err := h.db.First(&account, "user_id = ?", userID).Error; err != nil {
		response.NotFound(c, "Loyalty account")
		return
	}

	// Check if already claimed today
	if account.LastLoginBonus != nil {
		lastClaim := account.LastLoginBonus.Truncate(24 * time.Hour)
		today := time.Now().Truncate(24 * time.Hour)
		if lastClaim.Equal(today) {
			response.BadRequest(c, "Daily bonus already claimed today")
			return
		}
	}

	// Update streak
	var streak Streak
	if err := h.db.First(&streak, "account_id = ?", account.ID).Error; err != nil {
		streak = Streak{
			ID:            uuid.New(),
			AccountID:     account.ID,
			CurrentStreak: 0,
			LongestStreak: 0,
			LastActivity:  time.Now(),
		}
	}

	// Check if streak continues (within 48 hours of last activity)
	if time.Since(streak.LastActivity) <= 48*time.Hour {
		streak.CurrentStreak++
	} else {
		streak.CurrentStreak = 1
	}
	if streak.CurrentStreak > streak.LongestStreak {
		streak.LongestStreak = streak.CurrentStreak
	}
	streak.LastActivity = time.Now()
	h.db.Save(&streak)

	// Calculate bonus (base 10 + streak bonus)
	baseBonus := 10
	streakBonus := min(streak.CurrentStreak*2, 50) // Max 50 streak bonus
	totalBonus := baseBonus + streakBonus

	// Apply tier multiplier
	multiplier := TierMultipliers[account.Tier]
	earnedPoints := int(math.Round(float64(totalBonus) * multiplier))

	// Update account
	now := time.Now()
	account.CurrentPoints += earnedPoints
	account.LifetimePoints += earnedPoints
	account.LastLoginBonus = &now
	h.db.Save(&account)

	// Create transaction
	transaction := PointsTransaction{
		ID:          uuid.New(),
		AccountID:   account.ID,
		Action:      ActionDailyLogin,
		Points:      earnedPoints,
		Balance:     account.CurrentPoints,
		Multiplier:  multiplier,
		Description: "Daily login bonus",
		CreatedAt:   time.Now(),
	}
	h.db.Create(&transaction)

	response.OK(c, gin.H{
		"points_earned":  earnedPoints,
		"current_streak": streak.CurrentStreak,
		"longest_streak": streak.LongestStreak,
		"new_balance":    account.CurrentPoints,
	})
}

// GetRewards returns available rewards
func (h *Handler) GetRewards(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))

	var account LoyaltyAccount
	if err := h.db.First(&account, "user_id = ?", userID).Error; err != nil {
		response.NotFound(c, "Loyalty account")
		return
	}

	var rewards []Reward
	now := time.Now()
	h.db.Where("is_active = ? AND valid_from <= ?", true, now).
		Where("valid_until IS NULL OR valid_until >= ?", now).
		Order("points_cost ASC").Find(&rewards)

	// Mark which rewards are redeemable
	type RewardWithStatus struct {
		Reward
		CanRedeem bool   `json:"can_redeem"`
		Reason    string `json:"reason,omitempty"`
	}

	result := make([]RewardWithStatus, len(rewards))
	for i, r := range rewards {
		result[i].Reward = r
		result[i].CanRedeem = true

		if account.CurrentPoints < r.PointsCost {
			result[i].CanRedeem = false
			result[i].Reason = "Insufficient points"
		} else if tierOrder(account.Tier) < tierOrder(r.MinTier) {
			result[i].CanRedeem = false
			result[i].Reason = "Tier requirement not met"
		} else if r.MaxRedemptions != nil && r.CurrentRedemptions >= *r.MaxRedemptions {
			result[i].CanRedeem = false
			result[i].Reason = "Sold out"
		}
	}

	response.OK(c, gin.H{
		"rewards":        result,
		"current_points": account.CurrentPoints,
		"tier":           account.Tier,
	})
}

// RedeemReward redeems a reward
func (h *Handler) RedeemReward(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	rewardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid reward ID")
		return
	}

	var account LoyaltyAccount
	if err := h.db.First(&account, "user_id = ?", userID).Error; err != nil {
		response.NotFound(c, "Loyalty account")
		return
	}

	var reward Reward
	if err := h.db.First(&reward, "id = ? AND is_active = ?", rewardID, true).Error; err != nil {
		response.NotFound(c, "Reward")
		return
	}

	// Validate
	if account.CurrentPoints < reward.PointsCost {
		response.BadRequest(c, "Insufficient points")
		return
	}
	if tierOrder(account.Tier) < tierOrder(reward.MinTier) {
		response.BadRequest(c, "Tier requirement not met")
		return
	}
	if reward.MaxRedemptions != nil && reward.CurrentRedemptions >= *reward.MaxRedemptions {
		response.BadRequest(c, "Reward sold out")
		return
	}

	// Generate redemption code
	codeBytes := make([]byte, 8)
	rand.Read(codeBytes)
	code := hex.EncodeToString(codeBytes)

	// Create redemption
	redemption := RewardRedemption{
		ID:          uuid.New(),
		AccountID:   account.ID,
		RewardID:    rewardID,
		PointsSpent: reward.PointsCost,
		Code:        code,
		Status:      "active",
		ExpiresAt:   time.Now().AddDate(0, 3, 0), // 3 months validity
		CreatedAt:   time.Now(),
	}

	// Deduct points
	account.CurrentPoints -= reward.PointsCost
	reward.CurrentRedemptions++

	// Transaction
	h.db.Transaction(func(tx *gorm.DB) error {
		tx.Save(&account)
		tx.Save(&reward)
		tx.Create(&redemption)

		// Log transaction
		tx.Create(&PointsTransaction{
			ID:          uuid.New(),
			AccountID:   account.ID,
			Action:      ActionRedemption,
			Points:      -reward.PointsCost,
			Balance:     account.CurrentPoints,
			ReferenceID: ptrString(redemption.ID.String()),
			Description: "Redeemed: " + reward.Name,
			CreatedAt:   time.Now(),
		})
		return nil
	})

	response.OK(c, gin.H{
		"redemption":  redemption,
		"code":        code,
		"new_balance": account.CurrentPoints,
	})
}

// GetMyRedemptions returns user's redemptions
func (h *Handler) GetMyRedemptions(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))

	var account LoyaltyAccount
	if err := h.db.First(&account, "user_id = ?", userID).Error; err != nil {
		response.NotFound(c, "Loyalty account")
		return
	}

	var redemptions []RewardRedemption
	h.db.Preload("Reward").Where("account_id = ?", account.ID).
		Order("created_at DESC").Find(&redemptions)

	response.OK(c, redemptions)
}

// ApplyReferral applies a referral code
func (h *Handler) ApplyReferral(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))

	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var account LoyaltyAccount
	if err := h.db.First(&account, "user_id = ?", userID).Error; err != nil {
		response.NotFound(c, "Loyalty account")
		return
	}

	if account.ReferredBy != nil {
		response.BadRequest(c, "Already used a referral code")
		return
	}

	// Find referrer
	var referrer LoyaltyAccount
	if err := h.db.First(&referrer, "referral_code = ?", req.Code).Error; err != nil {
		response.NotFound(c, "Referral code")
		return
	}

	if referrer.UserID == userID {
		response.BadRequest(c, "Cannot refer yourself")
		return
	}

	// Apply referral
	account.ReferredBy = &referrer.ID
	referrer.TotalReferrals++

	// Bonus points
	referralBonus := 500
	referrerBonus := 250

	account.CurrentPoints += referralBonus
	account.LifetimePoints += referralBonus
	referrer.CurrentPoints += referrerBonus
	referrer.LifetimePoints += referrerBonus

	h.db.Transaction(func(tx *gorm.DB) error {
		tx.Save(&account)
		tx.Save(&referrer)

		// Log transactions
		tx.Create(&PointsTransaction{
			ID:          uuid.New(),
			AccountID:   account.ID,
			Action:      ActionReferral,
			Points:      referralBonus,
			Balance:     account.CurrentPoints,
			Description: "Referral bonus",
			CreatedAt:   time.Now(),
		})
		tx.Create(&PointsTransaction{
			ID:          uuid.New(),
			AccountID:   referrer.ID,
			Action:      ActionReferral,
			Points:      referrerBonus,
			Balance:     referrer.CurrentPoints,
			Description: "Referral reward",
			CreatedAt:   time.Now(),
		})
		return nil
	})

	response.OK(c, gin.H{
		"message":       "Referral applied successfully",
		"points_earned": referralBonus,
		"new_balance":   account.CurrentPoints,
	})
}

// GetBadges returns user's badges
func (h *Handler) GetBadges(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))

	var account LoyaltyAccount
	if err := h.db.First(&account, "user_id = ?", userID).Error; err != nil {
		response.NotFound(c, "Loyalty account")
		return
	}

	var userBadges []UserBadge
	h.db.Preload("Badge").Where("account_id = ?", account.ID).Find(&userBadges)

	var allBadges []Badge
	h.db.Where("is_active = ?", true).Find(&allBadges)

	// Mark earned badges
	earnedMap := make(map[uuid.UUID]time.Time)
	for _, ub := range userBadges {
		earnedMap[ub.BadgeID] = ub.EarnedAt
	}

	type BadgeWithStatus struct {
		Badge
		Earned   bool       `json:"earned"`
		EarnedAt *time.Time `json:"earned_at,omitempty"`
	}

	result := make([]BadgeWithStatus, len(allBadges))
	for i, b := range allBadges {
		result[i].Badge = b
		if earnedAt, ok := earnedMap[b.ID]; ok {
			result[i].Earned = true
			result[i].EarnedAt = &earnedAt
		}
	}

	response.OK(c, result)
}

// GetLeaderboard returns top users by points
func (h *Handler) GetLeaderboard(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit > 100 {
		limit = 100
	}

	var accounts []LoyaltyAccount
	h.db.Order("lifetime_points DESC").Limit(limit).Find(&accounts)

	type LeaderboardEntry struct {
		Rank           int       `json:"rank"`
		UserID         uuid.UUID `json:"user_id"`
		LifetimePoints int       `json:"lifetime_points"`
		Tier           TierLevel `json:"tier"`
	}

	result := make([]LeaderboardEntry, len(accounts))
	for i, a := range accounts {
		result[i] = LeaderboardEntry{
			Rank:           i + 1,
			UserID:         a.UserID,
			LifetimePoints: a.LifetimePoints,
			Tier:           a.Tier,
		}
	}

	response.OK(c, result)
}

func tierOrder(t TierLevel) int {
	switch t {
	case TierBronze:
		return 1
	case TierSilver:
		return 2
	case TierGold:
		return 3
	case TierPlatinum:
		return 4
	case TierDiamond:
		return 5
	default:
		return 0
	}
}

func ptrString(s string) *string {
	return &s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
