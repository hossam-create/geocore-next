package reports

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers public + admin report routes.
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	// Authenticated users submit reports
	auth := r.Group("/reports")
	auth.Use(middleware.Auth())
	{
		auth.POST("", h.CreateReport)
	}

	// Admin routes registered separately via admin/routes.go using handler methods directly
	_ = h // handler used by admin package via RegisterAdminRoutes
}

// RegisterAdminRoutes registers admin-only report routes under an already-auth'd group.
func RegisterAdminRoutes(adm *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)
	adm.GET("/reports", middleware.RequireAnyPermission(middleware.PermReportsReview), h.AdminListReports)
	adm.GET("/reports/stats", middleware.RequireAnyPermission(middleware.PermReportsReview), h.AdminGetStats)
	adm.PATCH("/reports/:id", middleware.RequireAnyPermission(middleware.PermReportsReview), h.AdminReviewReport)
}
