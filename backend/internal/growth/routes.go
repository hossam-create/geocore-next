package growth

import (
	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Handler for growth endpoints.
type Handler struct {
	db       *gorm.DB
	notifSvc *notifications.Service
}

func NewHandler(db *gorm.DB, notifSvc *notifications.Service) *Handler {
	return &Handler{db: db, notifSvc: notifSvc}
}

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, notifSvc *notifications.Service) {
	h := NewHandler(db, notifSvc)

	// User-facing referral routes
	growth := r.Group("/growth")
	growth.Use(middleware.Auth())
	{
		growth.GET("/referral/stats", h.GetReferralStats)
		growth.POST("/referral/invite-traveler", h.InviteTravelerHandler)
		growth.POST("/referral/register", h.RegisterInviteHandler)
	}

	// Admin growth routes
	adminGrowth := r.Group("/admin/growth")
	adminGrowth.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		adminGrowth.POST("/bootstrap/ghost-listings", h.SeedGhostListingsHandler)
		adminGrowth.POST("/bootstrap/platform-travelers", h.SeedPlatformTravelersHandler)
		adminGrowth.GET("/bootstrap/stats", h.GetBootstrapStatsHandler)
		adminGrowth.POST("/bootstrap/cleanup", h.CleanupGhostDataHandler)
		adminGrowth.POST("/auto-fill", h.TriggerAutoFillHandler)
		adminGrowth.POST("/conversion/detect-stale", h.DetectStaleListingsHandler)
		adminGrowth.GET("/conversion/stats", h.GetConversionStatsHandler)
		adminGrowth.POST("/retention/weekly-digest", h.SendWeeklyDigestHandler)
		adminGrowth.POST("/retention/reengage", h.ReEngageUsersHandler)
	}
}

// ── User-facing handlers ──────────────────────────────────────────────────

func (h *Handler) GetReferralStats(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	stats := GetReferralStats(h.db, userID)
	response.OK(c, stats)
}

func (h *Handler) InviteTravelerHandler(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))
	invite, err := InviteTraveler(h.db, userID, req.Email)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Created(c, invite)
}

func (h *Handler) RegisterInviteHandler(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))
	if err := RegisterTravelerInvite(h.db, req.Code, userID); err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "invite registered"})
}

// ── Admin handlers ─────────────────────────────────────────────────────────

func (h *Handler) SeedGhostListingsHandler(c *gin.Context) {
	var listings []GhostListing
	if err := c.ShouldBindJSON(&listings); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	adminID, _ := uuid.Parse(c.GetString("user_id"))
	if err := SeedGhostListings(h.db, adminID, listings); err != nil {
		response.InternalError(c, err)
		return
	}
	response.Created(c, gin.H{"count": len(listings)})
}

func (h *Handler) SeedPlatformTravelersHandler(c *gin.Context) {
	var travelers []PlatformTraveler
	if err := c.ShouldBindJSON(&travelers); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := SeedPlatformTravelers(h.db, travelers); err != nil {
		response.InternalError(c, err)
		return
	}
	response.Created(c, gin.H{"count": len(travelers)})
}

func (h *Handler) GetBootstrapStatsHandler(c *gin.Context) {
	stats := GetBootstrapStats(h.db)
	response.OK(c, stats)
}

func (h *Handler) CleanupGhostDataHandler(c *gin.Context) {
	if err := CleanupGhostData(h.db); err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "ghost data cleanup complete"})
}

func (h *Handler) TriggerAutoFillHandler(c *gin.Context) {
	if err := SmartAutoFill(h.db, h.notifSvc); err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "auto-fill triggered"})
}

func (h *Handler) DetectStaleListingsHandler(c *gin.Context) {
	if err := DetectStaleListings(h.db, h.notifSvc); err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "stale listing detection complete"})
}

func (h *Handler) GetConversionStatsHandler(c *gin.Context) {
	stats := GetConversionStats(h.db)
	response.OK(c, stats)
}

func (h *Handler) SendWeeklyDigestHandler(c *gin.Context) {
	if err := SendWeeklyDigest(h.db, h.notifSvc); err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "weekly digest sent"})
}

func (h *Handler) ReEngageUsersHandler(c *gin.Context) {
	if err := ReEngageInactiveUsers(h.db, h.notifSvc); err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "re-engagement notifications sent"})
}
