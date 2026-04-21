package cancellation

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	// Authenticated — buyer cancellation endpoints
	auth := v1.Group("")
	auth.Use(middleware.Auth())
	{
		// Preview what the cancellation fee would be (no side effects)
		auth.GET("/orders/:id/cancel-preview", h.PreviewCancellationFee)
		// Execute cancellation with smart fee
		auth.POST("/orders/:id/cancel", h.CancelOrderWithFee)

		// Cancellation Insurance (opt-in at checkout)
		auth.POST("/orders/:id/insurance", h.PurchaseInsurance)
		auth.GET("/orders/:id/insurance", h.GetOrderInsurance)
		auth.GET("/orders/:id/insurance-pricing", h.GetInsurancePricing)
	}

	// Admin — cancellation policy management + stats
	admin := v1.Group("/admin/cancellation")
	admin.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		admin.GET("/stats", h.GetAdminStats)
		admin.GET("/policies", h.ListPolicies)
		admin.PUT("/policies/:id", h.UpdatePolicy)
		admin.GET("/high-risk-users", h.ListHighRiskUsers)

		// Insurance admin
		admin.GET("/insurance/stats", h.GetAdminInsuranceStats)
		admin.POST("/insurance/disable-user/:id", h.DisableInsuranceForUser)
	}
}
