package order

import (
	"time"

	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterRoutes registers all order routes under /api/v1
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb ...*redis.Client) {
	repo := NewRepository(db)
	handler := NewHandler(repo)

	var rl *middleware.RateLimiter
	if len(rdb) > 0 && rdb[0] != nil {
		rl = middleware.NewRateLimiter(rdb[0])
	}

	orders := r.Group("/orders")
	{
		// Create order — rate limited to prevent abuse
		if rl != nil {
			orders.POST("", rl.LimitByUser(20, time.Hour, "orders:create:user"), handler.CreateOrder)
			orders.POST("/guest", rl.Limit(10, time.Hour, "orders:guest:ip"), handler.CreateGuestOrder)
		} else {
			orders.POST("", handler.CreateOrder)
			orders.POST("/guest", handler.CreateGuestOrder)
		}

		// Buyer order list (critical read — always use primary DB)
		orders.GET("", middleware.CriticalRead(), handler.ListBuyerOrders)

		// Seller order list (critical read)
		orders.GET("/selling", middleware.CriticalRead(), handler.ListSellerOrders)

		// Single order operations (critical read)
		orders.GET("/:id", middleware.CriticalRead(), handler.GetOrder)

		// Status transitions
		orders.PATCH("/:id/confirm", handler.ConfirmOrder)
		orders.PATCH("/:id/ship", handler.ShipOrder)
		orders.PATCH("/:id/deliver", handler.DeliverOrder)
		orders.PATCH("/:id/cancel", handler.CancelOrder)
	}
}
