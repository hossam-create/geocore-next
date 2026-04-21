package arpreview

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	v1.GET("/listings/:id/3d-models", h.ListModels)

	auth := v1.Group("")
	auth.Use(middleware.Auth())
	auth.POST("/listings/:id/3d-models", h.AddModel)
	auth.DELETE("/listings/:id/3d-models/:modelId", h.DeleteModel)
}
