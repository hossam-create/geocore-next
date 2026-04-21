package aiops

import (
	"context"

	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers the AIOps API and starts the background engine loop.
//
// Endpoints (admin-only):
//
//	GET  /api/v1/aiops/health                        — engine status
//	GET  /api/v1/aiops/incidents                     — list incidents (?status=open&severity=P0)
//	GET  /api/v1/aiops/incidents/:id                 — get single incident
//	POST /api/v1/aiops/incidents/:id/resolve         — mark resolved (human approval)
//	POST /api/v1/aiops/incidents/:id/ignore          — dismiss incident
//	POST /api/v1/aiops/analyze                       — on-demand analysis
func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB, ctx context.Context) {
	engine := NewEngine()
	engine.Start(ctx)

	h := NewHandler(engine)

	g := v1.Group("/aiops",
		middleware.Auth(),
		middleware.AdminWithDB(db),
	)
	{
		g.GET("/health", h.Health)
		g.GET("/incidents", h.ListIncidents)
		g.GET("/incidents/:id", h.GetIncident)
		g.POST("/incidents/:id/resolve", h.ResolveIncident)
		g.POST("/incidents/:id/ignore", h.IgnoreIncident)
		g.POST("/analyze", h.Analyze)
	}
}
