package reverseauctions

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	g := v1.Group("/reverse-auctions")
	{
		// Public
		g.GET("", h.ListRequests)
		g.GET("/:id", h.GetRequest)
		g.GET("/:id/offers", h.ListOffers)

		// Authenticated
		auth := g.Group("")
		auth.Use(middleware.Auth())
		{
			auth.POST("", h.CreateRequest)
			auth.PUT("/:id", h.UpdateRequest)
			auth.DELETE("/:id", h.DeleteRequest)
			auth.POST("/:id/offers", h.CreateOffer)
			auth.PUT("/:id/offers/:offerId/accept", h.AcceptOffer)
			auth.PUT("/:id/offers/:offerId/reject", h.RejectOffer)
			auth.PUT("/:id/offers/:offerId/counter", h.CounterOffer)
			auth.PUT("/:id/offers/:offerId/respond", h.RespondToCounter)
			auth.DELETE("/:id/offers/:offerId", h.WithdrawOffer)
		}

		// My offers dashboard
		my := g.Group("/my")
		my.Use(middleware.Auth())
		{
			my.GET("/sent", h.MySentOffers)
			my.GET("/received", h.MyReceivedOffers)
		}
	}
}
