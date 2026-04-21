package matching

import (
	"time"

	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterRoutes mounts matching endpoints.
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	svc := NewService(db, rdb)
	rl := middleware.NewRateLimiter(rdb)

	r.GET("/orders/:id/matches",
		middleware.Auth(),
		rl.LimitByUser(60, time.Minute, "matching:orders"),
		svc.GetMatches,
	)
}
