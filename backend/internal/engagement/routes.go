package engagement

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes mounts engagement endpoints on the router group.
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewEngagementHandler(db)

	// ── Authenticated endpoints ──────────────────────────────────────────────
	e := r.Group("/engagement")
	e.Use(middleware.Auth())
	{
		// Session momentum
		e.POST("/momentum/action", h.RecordAction)
		e.GET("/momentum/:session_id", h.GetMomentum)

		// Notification AI
		e.POST("/notification/decide", h.Decide)
		e.POST("/notification/send", h.Send)
		e.POST("/notification/outcome", h.RecordOutcome)

		// Re-engagement
		e.GET("/segment/:user_id", h.GetSegment)
		e.POST("/reengage/:user_id", h.PlanReEngagement)

		// Timing
		e.POST("/activity", h.RecordActivity)
		e.GET("/best-time/:user_id", h.GetBestTime)

		// User preferences
		e.PUT("/preferences/:user_id", h.UpdatePreferences)
	}

	// ── Admin endpoints ──────────────────────────────────────────────────────
	admin := r.Group("/admin/engagement")
	admin.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		admin.GET("/dashboard", h.GetDashboard)
		admin.POST("/kill-switch", h.ActivateKillSwitch)
		admin.DELETE("/kill-switch", h.DeactivateKillSwitch)
		admin.POST("/segment-all", h.SegmentAll)
		admin.POST("/process-touches", h.ProcessTouches)
	}
}
