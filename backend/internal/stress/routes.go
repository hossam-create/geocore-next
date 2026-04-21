package stress

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes mounts the stress simulation API under /api/v1/stress.
// All endpoints require admin authentication.
//
// Endpoints:
//
//	GET  /api/v1/stress/scenarios              — list built-in scenarios
//	POST /api/v1/stress/run/:scenario_id       — start a named scenario
//	GET  /api/v1/stress/status                 — live run status
//	GET  /api/v1/stress/report                 — last completed report
//	POST /api/v1/stress/stop                   — abort running test
func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	o := newOrchestrator()
	h := newHandler(o)

	g := v1.Group("/stress",
		middleware.Auth(),
		middleware.AdminWithDB(db),
	)
	{
		g.GET("/scenarios", h.ListScenarios)
		g.POST("/run/:scenario_id", h.RunScenario)
		g.GET("/status", h.Status)
		g.GET("/report", h.Report)
		g.POST("/stop", h.Stop)
	}
}
