package fraud

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterPredictRoutes mounts Sprint 24 admin endpoints under /admin/system/risk.
// These are separate from the existing /admin/fraud endpoints registered by
// RegisterRoutes (Sprint 5) and focus specifically on the Predictor subsystem.
func RegisterPredictRoutes(rg *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewPredictHandler(db, rdb)

	adm := rg.Group("/admin/system/risk")
	adm.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		adm.GET("/users", h.RiskUsersHandler)
		adm.GET("/users/:userId", h.PredictUserHandler)
	}
}
