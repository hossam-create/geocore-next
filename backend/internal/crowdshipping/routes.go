package crowdshipping

import (
	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB, notifSvc *notifications.Service) {
	h := NewHandler(db, notifSvc)
	oh := NewOfferHandler(db, notifSvc)

	// Public — browse trips and delivery requests
	v1.GET("/trips", h.ListTrips)
	v1.GET("/trips/search", h.SearchTrips)
	v1.GET("/trips/:id", h.GetTrip)
	v1.GET("/delivery-requests", h.ListDeliveryRequests)
	v1.GET("/delivery-requests/:id", h.GetDeliveryRequest)

	// Public — corridor config, pricing, compliance advisory
	v1.GET("/corridors", h.ListCorridors)
	v1.GET("/corridors/:origin/:dest", h.GetCorridor)
	v1.GET("/corridors/:origin/:dest/advisory", h.GetComplianceAdvisory)

	// Authenticated
	auth := v1.Group("")
	auth.Use(middleware.Auth())
	{
		// Trip management (traveler)
		auth.POST("/trips", h.CreateTrip)
		auth.DELETE("/trips/:id", h.CancelTrip)

		// Delivery request management (buyer)
		auth.POST("/delivery-requests", h.CreateDeliveryRequest)

		// Pricing is registered by internal/pricing/routes.go

		// Matching
		auth.POST("/delivery-requests/:id/find-travelers", h.FindTravelers)
		auth.POST("/delivery-requests/:id/match", h.MatchRequest)
		auth.POST("/delivery-requests/:id/accept", h.AcceptMatch)
		auth.POST("/delivery-requests/:id/reject", h.RejectMatch)
		auth.POST("/delivery-requests/:id/confirm-delivery", h.ConfirmDelivery)

		// Offer system (Sprint 2)
		auth.POST("/offers/create", oh.CreateOffer)
		auth.POST("/offers/:id/accept", oh.AcceptOffer)
		auth.POST("/offers/:id/counter", oh.CounterOffer)
		auth.POST("/offers/:id/reject", oh.RejectOffer)
		auth.POST("/offers/:id/retry-payment", oh.RetryPayment)
		auth.GET("/offers/listing/:listing_id", oh.ListOffersForDeliveryRequest)

		// Tracking (Sprint 2)
		auth.POST("/tracking/update", oh.UpdateTracking)
		auth.GET("/tracking/:order_id", oh.GetTracking)
		auth.POST("/tracking/:order_id/confirm", oh.ConfirmDelivery)
	}
}
