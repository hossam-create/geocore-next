package redteam

import (
	"github.com/geocore-next/backend/internal/security"
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterRoutes mounts the admin-only red-team endpoints under /admin/redteam.
// Always admin-gated; behind ENABLE_REDTEAM flag at request time.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, rdb *redis.Client, ids *security.IDS) {
	sim := NewSimulator(db, rdb, ids)
	h := NewHandler(sim)

	adm := rg.Group("/admin/redteam")
	adm.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		adm.POST("/run", h.RunHandler)
		adm.GET("/runs", h.ListRunsHandler)
	}
}
