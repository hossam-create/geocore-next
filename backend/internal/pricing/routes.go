package pricing

import (
	"time"

	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterRoutes mounts pricing endpoints on the router group.
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	svc := NewService(db, rdb)
	rl := middleware.NewRateLimiter(rdb)
	h := NewPricingHandler(db)
	bh := NewBanditHandler(db)

	p := r.Group("/pricing")
	p.Use(middleware.Auth())
	{
		// Shipping route pricing (existing)
		p.POST("/calculate",
			rl.LimitByUser(30, time.Minute, "pricing:calculate"),
			svc.Calculate,
		)

		// Insurance dynamic pricing (rule-based + AI)
		p.GET("/insurance/:id", h.GetInsurancePrice)
		p.GET("/insurance/variant", h.GetPricingVariant)
		p.POST("/insurance/:id/outcome", h.RecordOutcome)

		// Bandit pricing (real-time optimization)
		p.GET("/insurance/bandit/:id", bh.SelectInsurancePrice)
		p.POST("/insurance/bandit/feedback", bh.RecordFeedback)

		// RL pricing (sequential decisions)
		rh := NewRLHandler(db)
		p.GET("/insurance/rl/:id", rh.SelectPrice)
		p.POST("/insurance/rl/feedback", rh.RecordFeedback)

		// Hybrid pricing (RL + Bandit + Rules guardrails)
		hh := NewHybridHandler(db)
		p.GET("/insurance/hybrid/:id", hh.SelectPrice)
		p.POST("/insurance/hybrid/feedback", hh.RecordFeedback)

		// Cross-system RL (pricing + ranking + recommendations)
		ch := NewCrossHandler(db)
		p.GET("/insurance/cross/:id", ch.Select)
		p.POST("/insurance/cross/feedback", ch.Feedback)

		// Feature pipeline (enrich + events + retrieval)
		fh := NewFeatureHandler(db, rdb)
		p.POST("/features/enrich", fh.Enrich)
		p.POST("/features/event", fh.RecordEvent)
		p.POST("/features/retrieve", fh.Retrieve)
		p.GET("/features/similar/:id", fh.GetSimilarItems)
	}

	// Admin — insurance pricing management
	admin := r.Group("/admin/pricing")
	admin.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		admin.GET("/metrics", h.GetAdminMetrics)
		admin.POST("/model", h.LoadModel)
		admin.PUT("/config", h.UpdateConfig)
		admin.GET("/gaming/:id", h.DetectGaming)

		// Bandit admin
		admin.GET("/bandit", bh.GetDashboard)
		admin.PUT("/bandit/config", bh.UpdateConfig)
		admin.POST("/bandit/kill-switch", bh.ActivateKillSwitch)
		admin.DELETE("/bandit/kill-switch", bh.DeactivateKillSwitch)
		admin.POST("/bandit/reset", bh.ResetSegment)
		admin.GET("/bandit/conversion-check", bh.CheckConversion)

		// RL admin
		rh := NewRLHandler(db)
		admin.GET("/rl", rh.GetDashboard)
		admin.PUT("/rl/config", rh.UpdateConfig)
		admin.POST("/rl/kill-switch", rh.ActivateKillSwitch)
		admin.DELETE("/rl/kill-switch", rh.DeactivateKillSwitch)
		admin.POST("/rl/rollout/advance", rh.AdvanceRollout)
		admin.POST("/rl/rollout/rollback", rh.RollbackRollout)
		admin.POST("/rl/q-table/save", rh.SaveQTable)
		admin.GET("/rl/q-table/stats", rh.GetQTableStats)
		admin.GET("/rl/conversion-check", rh.CheckConversion)

		// Hybrid admin
		hh := NewHybridHandler(db)
		admin.GET("/hybrid", hh.GetDashboard)
		admin.PUT("/hybrid/config", hh.UpdateConfig)
		admin.POST("/hybrid/emergency", hh.ActivateEmergency)
		admin.DELETE("/hybrid/emergency", hh.DeactivateEmergency)
		admin.POST("/hybrid/rollout/advance", hh.AdvanceRollout)
		admin.POST("/hybrid/rollout/rollback", hh.RollbackRollout)
		admin.GET("/hybrid/conversion-check", hh.CheckConversion)

		// Cross-system admin
		ch := NewCrossHandler(db)
		admin.GET("/cross", ch.GetDashboard)
		admin.PUT("/cross/config", ch.UpdateConfig)
		admin.POST("/cross/emergency", ch.ActivateEmergency)
		admin.DELETE("/cross/emergency", ch.DeactivateEmergency)
		admin.POST("/cross/rollout/advance", ch.AdvanceRollout)
		admin.POST("/cross/rollout/rollback", ch.RollbackRollout)
		admin.POST("/cross/q-tables/save", ch.SaveQTables)
		admin.GET("/cross/conversion-check", ch.CheckConversion)

		// Feature pipeline admin
		fh := NewFeatureHandler(db, rdb)
		admin.GET("/features/dashboard", fh.GetDashboard)
		admin.POST("/features/refresh", fh.RefreshFeatures)
		admin.GET("/features/user/:id", fh.GetUserFeatures)
		admin.GET("/features/item/:id", fh.GetItemFeatures)
	}
}
