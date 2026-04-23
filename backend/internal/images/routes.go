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
//
//	POST /images/upload          10 uploads / hour
//	POST /images/presign         20 presign / hour
//	POST /images/confirm         20 confirms / hour
//	DELETE /images/:id            5 deletes / 15 min
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewHandler(db)
	svc := NewImageService(db)
	rl := middleware.NewRateLimiter(rdb)

	img := r.Group("/images")
	img.Use(middleware.Auth())
	{
		// Server-side multipart upload (legacy, still supported)
		img.POST("/upload",
			rl.LimitByUser(10, time.Hour, "images:upload:user"),
			h.Upload,
		)
		// Presigned upload flow (new — client uploads directly to S3)
		img.POST("/presign",
			rl.LimitByUser(20, time.Hour, "images:presign:user"),
			h.GetPresignedUpload(svc),
		)
		img.POST("/presign/batch",
			rl.LimitByUser(10, time.Hour, "images:presign:batch:user"),
			h.GetBatchPresignedUpload(svc),
		)
		img.POST("/confirm",
			rl.LimitByUser(20, time.Hour, "images:confirm:user"),
			h.ConfirmUpload(svc),
		)
		// Image group variants
		img.GET("/group/:group_id", h.GetImageGroup(svc))
		// Listing image management
		img.GET("/listing/:listing_id", h.GetListingImages(svc))
		img.DELETE("/listing/:listing_id/:image_id",
			rl.LimitByUser(10, 15*time.Minute, "images:delete:user"),
			h.DeleteListingImage(svc),
		)
		// Legacy endpoints
		img.DELETE("/:id",
			rl.LimitByUser(5, 15*time.Minute, "images:delete:user"),
			h.Delete,
		)
		img.GET("", h.ListMine)
	}

	// Media/upload-url endpoint (existing presign flow)
	media := r.Group("/media")
	media.Use(middleware.Auth())
	{
		media.POST("/upload-url", h.GetUploadURL)
	}
}
