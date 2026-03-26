package auctions

  import (
  	"github.com/geocore-next/backend/pkg/middleware"
  	"github.com/gin-gonic/gin"
  	"github.com/redis/go-redis/v9"
  	"gorm.io/gorm"
  )

  func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
  	h := NewHandler(db, rdb)

  	a := r.Group("/auctions")
  	{
  		// Public — anyone can view auctions and bids
  		a.GET("", h.List)
  		a.GET("/:id", h.Get)
  		a.GET("/:id/bids", h.GetBids)

  		// Auth + email verified — required to create auctions or place bids
  		verified := a.Group("")
  		verified.Use(middleware.Auth(), middleware.EmailVerified(db))
  		{
  			verified.POST("", h.Create)
  			verified.POST("/:id/bid", h.PlaceBid)
  		}
  	}
  }
  