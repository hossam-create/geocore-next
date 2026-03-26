package reviews

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)
	// GET /users/:id/reviews — public, no auth
	r.GET("/users/:id/reviews", h.List)
	// POST /users/:id/reviews — requires auth
	r.POST("/users/:id/reviews", middleware.Auth(), h.Create)
}
