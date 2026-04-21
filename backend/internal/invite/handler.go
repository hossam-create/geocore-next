package invite

import (
	"net/http"

	"github.com/geocore-next/backend/internal/config"
	"github.com/geocore-next/backend/internal/reputation"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Handler provides HTTP handlers for the invite system.
type Handler struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
	return &Handler{db: db, rdb: rdb}
}

// CreateInviteHandler — POST /invite
func (h *Handler) CreateInviteHandler(c *gin.Context) {
	if !config.GetFlags().EnableInviteOnly && !config.GetFlags().EnableReferrals {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "invite system is disabled"})
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))
	var body struct {
		TTLDays int `json:"ttl_days"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.TTLDays == 0 {
		body.TTLDays = 30
	}
	inv, err := CreateInvite(h.db, userID, body.TTLDays)
	if err != nil {
		switch err {
		case ErrNotEligible:
			c.JSON(http.StatusForbidden, gin.H{"error": "Your trust score is too low. Build more reputation first."})
		case ErrQuotaExceeded:
			c.JSON(http.StatusForbidden, gin.H{"error": "You have reached your invite quota."})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"invite":     inv,
		"share_code": inv.InviteCode,
		"message":    "Share this code with trusted people only.",
	})
}

// GetMyInvitesHandler — GET /invite
func (h *Handler) GetMyInvitesHandler(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	var invites []Invite
	h.db.Where("inviter_id = ?", userID).Order("created_at DESC").Find(&invites)
	score := reputation.GetOverallScore(h.db, userID)
	quota := AllowedInviteCount(score)
	c.JSON(http.StatusOK, gin.H{
		"invites":     invites,
		"quota":       quota,
		"trust_score": score,
		"eligible":    quota > 0,
	})
}

// GetMyRewardsHandler — GET /invite/rewards
func (h *Handler) GetMyRewardsHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	var rewards []ReferralReward
	h.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&rewards)
	c.JSON(http.StatusOK, gin.H{"rewards": rewards})
}

// AdminInviteAnalyticsHandler — GET /admin/invites
func (h *Handler) AdminInviteAnalyticsHandler(c *gin.Context) {
	var total, usedTotal int64
	h.db.Model(&Invite{}).Count(&total)
	h.db.Model(&InviteUsage{}).Count(&usedTotal)

	usageRate := 0.0
	if total > 0 {
		usageRate = float64(usedTotal) / float64(total*3) * 100
	}

	type topRow struct {
		InviterID      string  `json:"inviter_id"`
		TotalInvites   int     `json:"total_invites"`
		TotalUsed      int     `json:"total_used"`
		ConversionRate float64 `json:"conversion_rate"`
	}
	var top []topRow
	h.db.Raw(`
		SELECT
			inviter_id::text,
			COUNT(*) AS total_invites,
			COALESCE(SUM(used_count),0) AS total_used,
			CASE WHEN SUM(max_uses)>0
				THEN ROUND(CAST(SUM(used_count) AS NUMERIC)/CAST(SUM(max_uses) AS NUMERIC)*100,2)
				ELSE 0
			END AS conversion_rate
		FROM invites
		GROUP BY inviter_id
		ORDER BY total_used DESC
		LIMIT 20
	`).Scan(&top)

	c.JSON(http.StatusOK, gin.H{
		"total_invites": total,
		"total_used":    usedTotal,
		"usage_rate":    usageRate,
		"top_inviters":  top,
	})
}
