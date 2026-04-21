package crypto

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	v1.GET("/crypto/providers", h.Providers)
	v1.POST("/crypto/coinbase/webhook", h.CoinbaseWebhook)

	cp := v1.Group("/crypto")
	cp.Use(middleware.Auth())
	{
		cp.POST("/create-charge", h.CreateCharge)
	}
}
