package referral

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Handler provides HTTP handlers for referral operations
type Handler struct {
	db *gorm.DB
}

// NewHandler creates a new referral handler
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// GetMyCode — GET /api/v1/referral/code
// Returns the authenticated user's referral code and share URL.
func (h *Handler) GetMyCode(c *gin.Context) {
	userID := c.GetString("user_id")

	var row struct {
		ReferralCode string `gorm:"column:referral_code"`
	}
	if err := h.db.Table("users").Select("referral_code").
		Where("id = ?", userID).First(&row).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Ensure code exists (back-fill for any legacy user)
	if row.ReferralCode == "" {
		uid, _ := uuid.Parse(userID)
		row.ReferralCode = GenerateCode(uid)
		h.db.Table("users").Where("id = ?", userID).
			Update("referral_code", row.ReferralCode)
	}

	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "https://geocore.app"
	}

	c.JSON(http.StatusOK, gin.H{
		"code":      row.ReferralCode,
		"share_url": fmt.Sprintf("%s/register?ref=%s", baseURL, row.ReferralCode),
	})
}

// GetStats — GET /api/v1/referral/stats
// Returns referral counts, completion rate, and total points earned.
func (h *Handler) GetStats(c *gin.Context) {
	userID := c.GetString("user_id")

	var row struct {
		ReferralCode string `gorm:"column:referral_code"`
	}
	h.db.Table("users").Select("referral_code").Where("id = ?", userID).First(&row)

	type counts struct {
		Status string
		Count  int
		Total  int
	}
	var rows []counts
	h.db.Table("referrals").
		Select("status, COUNT(*) as count, COALESCE(SUM(reward_points), 0) as total").
		Where("referrer_id = ?", userID).
		Group("status").
		Scan(&rows)

	stats := ReferralStats{
		Code:     row.ReferralCode,
		ShareURL: fmt.Sprintf("%s/register?ref=%s", baseURL(), row.ReferralCode),
	}
	for _, r := range rows {
		stats.TotalReferrals += r.Count
		switch ReferralStatus(r.Status) {
		case StatusPending:
			stats.Pending = r.Count
		case StatusCompleted:
			stats.Completed = r.Count
			stats.TotalEarned += r.Total
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// CompleteReferral is called internally after a referee's first order completes.
// It marks the referral as completed, awards loyalty points, and credits the
// referrer's wallet with the configured referral bonus (from the fee engine).
func CompleteReferral(db *gorm.DB, refereeID uuid.UUID) {
	var ref Referral
	if err := db.Where("referee_id = ? AND status = ?", refereeID, StatusPending).
		First(&ref).Error; err != nil {
		return // no pending referral — nothing to do
	}

	now := time.Now()
	if err := db.Model(&ref).Updates(map[string]interface{}{
		"status":         StatusCompleted,
		"reward_paid_at": now,
	}).Error; err != nil {
		slog.Error("referral: failed to complete referral",
			"referral_id", ref.ID.String(), "error", err.Error())
		return
	}

	// Award loyalty points
	awardLoyaltyPoints(db, ref.ReferrerID, ref.RewardPoints, ref.ID)

	// Credit wallet bonus via fee engine referral config
	creditReferralWalletBonus(db, ref.ReferrerID, ref.ID)
}

// creditReferralWalletBonus credits the referrer's wallet with the fee-engine
// configured referral bonus (FeeTypeReferral / fixed amount).
func creditReferralWalletBonus(db *gorm.DB, referrerID uuid.UUID, referralID uuid.UUID) {
	// Look up referral bonus amount from fee configs
	var cfg struct {
		FeeFixed float64
	}
	db.Table("fee_configs").
		Select("fee_fixed").
		Where("fee_type = 'referral' AND is_active = true").
		Order("country DESC").
		Limit(1).
		Scan(&cfg)

	bonus := cfg.FeeFixed
	if bonus <= 0 {
		bonus = 5.0 // sensible fallback
	}

	// Credit wallet using double-entry: platform → referrer
	err := db.Exec(`
		UPDATE wallets SET available = available + ?, updated_at = NOW()
		WHERE user_id = ?
	`, bonus, referrerID).Error
	if err != nil {
		slog.Error("referral: wallet credit failed", "user_id", referrerID, "bonus", bonus, "error", err)
		return
	}

	// Record wallet transaction for audit
	db.Exec(`
		INSERT INTO wallet_transactions (id, wallet_id, type, amount, currency, description, created_at)
		SELECT uuid_generate_v4(), id, 'credit', ?, 'USD', ?, NOW()
		FROM wallets WHERE user_id = ?
	`, bonus, fmt.Sprintf("Referral bonus (referral %s)", referralID), referrerID)

	slog.Info("referral: wallet bonus credited",
		"referrer_id", referrerID, "bonus", bonus, "referral_id", referralID)
}

// awardLoyaltyPoints credits loyalty points directly in the DB.
// It mirrors the loyalty.EarnPoints logic without importing the loyalty package
// to avoid a circular dependency. The loyalty_transactions table is written
// directly; loyalty_accounts are upserted.
func awardLoyaltyPoints(db *gorm.DB, userID uuid.UUID, points int, referralID uuid.UUID) {
	// Upsert loyalty account
	err := db.Exec(`
		INSERT INTO loyalty_accounts (id, user_id, current_points, total_earned, tier, created_at, updated_at)
		VALUES (uuid_generate_v4(), ?, ?, ?, 'bronze', NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE
		  SET current_points = loyalty_accounts.current_points + EXCLUDED.current_points,
		      total_earned    = loyalty_accounts.total_earned    + EXCLUDED.total_earned,
		      updated_at      = NOW()
	`, userID, points, points).Error
	if err != nil {
		slog.Error("referral: failed to upsert loyalty account",
			"user_id", userID.String(), "error", err.Error())
		return
	}

	// Record transaction
	err = db.Exec(`
		INSERT INTO loyalty_transactions (id, user_id, action, points, description, created_at)
		VALUES (uuid_generate_v4(), ?, 'referral', ?, ?, NOW())
	`, userID, points, fmt.Sprintf("Referral reward (referral %s)", referralID.String())).Error
	if err != nil {
		slog.Error("referral: failed to insert loyalty transaction",
			"user_id", userID.String(), "error", err.Error())
	}

	slog.Info("referral: loyalty points awarded",
		"user_id", userID.String(), "points", points, "referral_id", referralID.String())
}

// LinkReferral records a referral relationship when a new user registers with a code.
// Called from the auth.Register handler.
func LinkReferral(db *gorm.DB, refereeID uuid.UUID, code string) {
	if code == "" {
		return
	}

	// Find referrer by code
	var row struct {
		ID uuid.UUID
	}
	if err := db.Table("users").Select("id").
		Where("UPPER(referral_code) = UPPER(?)", code).First(&row).Error; err != nil {
		slog.Warn("referral: code not found", "code", code)
		return
	}

	// Don't allow self-referral
	if row.ID == refereeID {
		return
	}

	ref := Referral{
		ReferrerID:   row.ID,
		RefereeID:    refereeID,
		Code:         code,
		Status:       StatusPending,
		RewardPoints: DefaultRewardPoints,
	}

	if err := db.Create(&ref).Error; err != nil {
		slog.Error("referral: failed to create referral record",
			"referee_id", refereeID.String(), "code", code, "error", err.Error())
	}
}

func baseURL() string {
	if v := os.Getenv("APP_BASE_URL"); v != "" {
		return v
	}
	return "https://geocore.app"
}
