package support

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	support := r.Group("/support")
	{
		// Public routes
		support.GET("/subjects", h.GetSubjects)
		support.POST("/contact", h.SubmitContact)

		// Authenticated routes
		tickets := support.Group("/")
		tickets.Use(middleware.Auth())
		{
			tickets.GET("/tickets", h.GetTickets)
			tickets.POST("/tickets", h.CreateTicket)
			tickets.GET("/tickets/:id", h.GetTicket)
			tickets.POST("/tickets/:id/messages", h.AddTicketMessage)
			tickets.PATCH("/tickets/:id", h.UpdateTicket) // Admin only in practice
		}

		// Admin routes (future use)
		admin := support.Group("/")
		admin.Use(middleware.Auth())
		{
			admin.GET("/messages", h.GetMessages)
		}
	}
}
