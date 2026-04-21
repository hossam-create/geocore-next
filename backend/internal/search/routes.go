package search

import (
	"time"

	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterRoutes mounts search endpoints under the given router group.

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewHandler(db)
	rl := middleware.NewRateLimiter(rdb)

	// Semantic AI search
	rg.POST("/search", rl.Limit(100, time.Minute, "search:query:ip"), h.Search)

	// Autocomplete suggestions
	rg.GET("/search/suggest", rl.Limit(100, time.Minute, "search:suggest:ip"), h.Suggest)

	// Trending queries
	rg.GET("/search/trending", rl.Limit(100, time.Minute, "search:trending:ip"), h.Trending)

	// On-demand embedding for a specific listing (admin/indexer use)
	rg.POST("/listings/:id/embed", h.EmbedListing)
}
