package recommendations

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterRoutes mounts recommendation endpoints.
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	engine := NewEngine(db, rdb)
	h := NewHandler(engine)

	recs := r.Group("/recommendations")
	recs.Use(middleware.Auth())
	{
		recs.GET("", h.Get)
		recs.POST("/track", h.Track)
	}
}
