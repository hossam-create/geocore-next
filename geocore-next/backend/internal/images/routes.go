package images

  import (
  	"time"

  	"github.com/geocore-next/backend/pkg/middleware"
  	"github.com/gin-gonic/gin"
  	"github.com/redis/go-redis/v9"
  	"gorm.io/gorm"
  )

  // RegisterRoutes mounts all /images endpoints.
  //
  // Rate limits (per authenticated user):
  //   POST /images/upload   10 uploads / hour
  //   DELETE /images/:id     5 deletes / 15 min
  func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
  	h  := NewHandler(db)
  	rl := middleware.NewRateLimiter(rdb)

  	images := r.Group("/images")
  	images.Use(middleware.Auth())
  	{
  		images.POST("/upload",
  			rl.LimitByUser(10, time.Hour, "images:upload:user"),
  			h.Upload,
  		)
  		images.DELETE("/:id",
  			rl.LimitByUser(5, 15*time.Minute, "images:delete:user"),
  			h.Delete,
  		)
  		images.GET("", h.ListMine)
  	}
  }
  