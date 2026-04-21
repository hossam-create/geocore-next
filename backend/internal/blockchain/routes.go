package blockchain

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	auth := v1.Group("/blockchain/escrow")
	auth.Use(middleware.Auth())
	{
		auth.POST("", h.CreateEscrow)
		auth.GET("", h.List)
		auth.GET("/:id", h.Get)
		auth.POST("/:id/fund", h.Fund)
		auth.POST("/:id/release", h.Release)
		auth.POST("/:id/refund", h.Refund)
		auth.POST("/:id/dispute", h.Dispute)
	}
}
