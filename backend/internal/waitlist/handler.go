package waitlist

import (
	"fmt"
	"net/http"

	"github.com/geocore-next/backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const defaultBaseURL = "https://geocore.app"

// Handler holds dependencies for waitlist HTTP handlers.
type Handler struct {
	db      *gorm.DB
	rdb     *redis.Client
	baseURL string
}

func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
	return &Handler{db: db, rdb: rdb, baseURL: defaultBaseURL}
}

// JoinHandler — POST /waitlist/join
// Body: {"email": "user@example.com"}
// Query: ?ref=CODE (optional referral)
func (h *Handler) JoinHandler(c *gin.Context) {
	if !config.GetFlags().EnableWaitlist {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "waitlist_closed",
			"message": "The waitlist is not currently open.",
		})
		return
	}

	var body struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	refCode := c.Query("ref")
	ip := c.ClientIP()
	deviceID := c.GetHeader("X-Device-ID") // optional anti-gaming fingerprint

	result, err := Join(h.db, h.rdb, body.Email, refCode, ip, deviceID, h.baseURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"position":       result.User.Position,
		"referral_code":  result.User.ReferralCode,
		"share_link":     result.ShareLink,
		"message":        result.Message,
		"next_milestone": result.NextMilestone,
		"is_new":         result.IsNew,
	})
}

// StatusHandler — GET /waitlist/status?email=...
func (h *Handler) StatusHandler(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email query param required"})
		return
	}
	var user WaitlistUser
	if err := h.db.Where("email = ?", email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "email not found on waitlist"})
		return
	}
	ux := UXPayload(h.db, &user)
	shareLink := fmt.Sprintf("%s/waitlist?ref=%s", h.baseURL, user.ReferralCode)
	c.JSON(http.StatusOK, gin.H{
		"position":       ux.Position,
		"people_behind":  ux.PeopleBehind,
		"moved_today":    ux.MovedToday,
		"progress_pct":   ux.ProgressPercent,
		"next_unlock":    ux.NextUnlock,
		"referral_code":  user.ReferralCode,
		"referral_count": ux.ReferralCount,
		"share_link":     shareLink,
		"status":         ux.Status,
		"next_milestone": NextMilestone(user.Position),
	})
}

// StatsHandler — GET /waitlist/stats (public social proof)
func (h *Handler) StatsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, Stats(h.db))
}

// AdminReleaseHandler — POST /admin/waitlist/release
// Body: {"count": 50}
func (h *Handler) AdminReleaseHandler(c *gin.Context) {
	var body struct {
		Count int `json:"count" binding:"required,min=1,max=1000"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	released, err := ReleaseInvites(h.db, body.Count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"released": len(released),
		"message":  fmt.Sprintf("%d invites released", len(released)),
	})
}

// AdminRecalcHandler — POST /admin/waitlist/recalc
func (h *Handler) AdminRecalcHandler(c *gin.Context) {
	go RecalculatePositions(h.db)
	c.JSON(http.StatusAccepted, gin.H{"message": "position recalculation queued"})
}

// AdminFlagHandler — POST /admin/waitlist/flag
// Body: {"user_id": "...", "reason": "..."}
func (h *Handler) AdminFlagHandler(c *gin.Context) {
	var body struct {
		UserID string `json:"user_id" binding:"required"`
		Reason string `json:"reason"  binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	FlagUser(h.db, body.UserID, body.Reason)
	c.JSON(http.StatusOK, gin.H{"message": "user flagged"})
}

// AdminAnalyticsHandler — GET /admin/waitlist/analytics
func (h *Handler) AdminAnalyticsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, AdminAnalytics(h.db))
}

// AdminSetLimitHandler — POST /admin/waitlist/limit
// Body: {"daily_limit": 100}
func (h *Handler) AdminSetLimitHandler(c *gin.Context) {
	var body struct {
		DailyLimit int `json:"daily_limit" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := SetDailyLimit(h.db, body.DailyLimit); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"daily_limit": body.DailyLimit, "message": "daily limit updated"})
}
