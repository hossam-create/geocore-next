package wholesale

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	db.AutoMigrate(&WholesaleListing{}, &WholesaleOrder{}, &WholesaleSeller{})

	h := NewHandler(db)

	// Authenticated endpoints
	ws := rg.Group("/wholesale")
	{
		// Seller
		ws.POST("/sellers", h.RegisterSeller)
		ws.GET("/sellers/me", h.GetMySellerProfile)

		// Listings
		ws.POST("/listings", h.CreateListing)
		ws.GET("/listings", h.ListListings)
		ws.GET("/listings/:id", h.GetListing)

		// Orders
		ws.POST("/orders", h.CreateOrder)
		ws.GET("/orders", h.ListMyOrders)
		ws.PATCH("/orders/:id/respond", h.RespondToOrder)
	}

	// Admin endpoints
	admin := rg.Group("/admin/wholesale")
	{
		admin.GET("/sellers", h.AdminListSellers)
		admin.PATCH("/sellers/:id/verify", h.AdminVerifySeller)
		admin.GET("/listings", h.AdminListListings)
	}
}
