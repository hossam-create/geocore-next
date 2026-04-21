package country

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	repo := NewRepository(db, rdb)
	h := NewHandler(repo)

	// Auto-migrate
	db.AutoMigrate(&CountryConfig{}, &CountryOverride{})

	// Public endpoints
	countries := rg.Group("/country")
	{
		countries.GET("", h.ListConfigs)
		countries.GET("/:code", h.GetConfig)
		countries.GET("/:code/overrides", h.ListOverrides)
	}

	// Admin endpoints
	admin := rg.Group("/admin/country")
	{
		admin.POST("", h.UpsertConfig)
		admin.DELETE("/:code", h.DeleteConfig)
		admin.POST("/:code/overrides", h.CreateOverride)
		admin.DELETE("/overrides/:id", h.DeleteOverride)
	}
}

// NewMiddleware creates a CountryMiddleware with the given dependencies.
// Call this separately in main.go to inject the middleware into the router chain.
func NewMiddleware(db *gorm.DB, rdb *redis.Client) gin.HandlerFunc {
	repo := NewRepository(db, rdb)
	return CountryMiddleware(repo)
}
