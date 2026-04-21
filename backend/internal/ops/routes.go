package ops

import (
	"github.com/geocore-next/backend/pkg/jobs"
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterRoutes mounts all /ops Control Center endpoints (admin only).
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client, jq *jobs.JobQueue) (*CronScheduler, *AlertEngine) {
	scheduler := NewCronScheduler(db, jq)
	alertEng := NewAlertEngine(db, rdb)

	SeedDefaultSchedules(db)

	h := NewHandler(db, rdb, scheduler, alertEng)

	ops := r.Group("/ops")
	ops.Use(middleware.Auth(), middleware.AdminWithDB(db))
	{
		// System status / health
		ops.GET("/status", middleware.RequireAnyPermission(middleware.PermOpsRead), h.GetStatus)

		// Cron schedules
		ops.GET("/cron", middleware.RequireAnyPermission(middleware.PermOpsRead), h.ListCron)
		ops.POST("/cron", middleware.RequireAnyPermission(middleware.PermOpsManage), h.CreateCron)
		ops.PUT("/cron/:id", middleware.RequireAnyPermission(middleware.PermOpsManage), h.UpdateCron)
		ops.DELETE("/cron/:id", middleware.RequireAnyPermission(middleware.PermOpsManage), h.DeleteCron)

		// Alert rules
		ops.GET("/alerts", middleware.RequireAnyPermission(middleware.PermOpsRead), h.ListAlerts)
		ops.POST("/alerts", middleware.RequireAnyPermission(middleware.PermOpsManage), h.CreateAlert)
		ops.PUT("/alerts/:id", middleware.RequireAnyPermission(middleware.PermOpsManage), h.UpdateAlert)
		ops.DELETE("/alerts/:id", middleware.RequireAnyPermission(middleware.PermOpsManage), h.DeleteAlert)
		ops.GET("/alerts/history", middleware.RequireAnyPermission(middleware.PermOpsRead), h.GetAlertHistory)
		ops.GET("/alerts/metrics", middleware.RequireAnyPermission(middleware.PermOpsRead), h.GetAlertMetrics)

		// Runtime config (payment keys, feature flags, etc.)
		ops.GET("/config", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.ListConfig)
		ops.POST("/config", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.SetConfig)
		ops.POST("/config/bulk", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.BulkSetConfig)
		ops.DELETE("/config/:key", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.DeleteConfig)

		// Job queue management
		ops.GET("/jobs/stats", middleware.RequireAnyPermission(middleware.PermOpsRead), h.GetJobStats)
		ops.GET("/jobs/failed", middleware.RequireAnyPermission(middleware.PermOpsRead), h.GetFailedJobs)
		ops.POST("/jobs/retry", middleware.RequireAnyPermission(middleware.PermOpsManage), h.RetryFailedJobs)
	}

	return scheduler, alertEng
}
