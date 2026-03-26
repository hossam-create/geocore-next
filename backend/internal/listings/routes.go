package listings

  import (
  	"github.com/geocore-next/backend/pkg/middleware"
  	"github.com/gin-gonic/gin"
  	"github.com/redis/go-redis/v9"
  	"gorm.io/gorm"
  )

  func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
  	h := NewHandler(db, rdb)

  	r.GET("/categories", h.GetCategories)

  	// ── Search endpoints (public) ──────────────────────────────────────────────
  	r.GET("/listings/search",           h.Search)
  	r.GET("/listings/suggestions",      h.Suggestions)

  	listings := r.Group("/listings")
  	{
  		// Public — anyone can browse listings
  		listings.GET("",    h.List)
  		listings.GET("/:id", h.Get)

  		// Auth required
  		authed := listings.Group("")
  		authed.Use(middleware.Auth())
  		{
  			authed.GET("/me",                   h.GetMyListings)
  			authed.GET("/recent-searches",      h.RecentSearches)
  			authed.PUT("/:id",                  h.Update)
  			authed.DELETE("/:id",               h.Delete)
  			authed.POST("/:id/favorite",        h.ToggleFavorite)
  		}

  		// Auth + email verified — posting requires verified email
  		verified := listings.Group("")
  		verified.Use(middleware.Auth(), middleware.EmailVerified(db))
  		{
  			verified.POST("", h.Create)
  		}
  	}
  }
  