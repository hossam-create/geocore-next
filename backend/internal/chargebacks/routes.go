package chargebacks

import (
	"time"

	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewHandler(db)
	rl := middleware.NewRateLimiter(rdb)

	adm := r.Group("/admin/chargebacks")
	adm.Use(middleware.Auth(), middleware.AdminWithDB(db), rl.LimitByUser(60, time.Minute, "admin:chargebacks"))
	{
		adm.GET("", middleware.RequireAnyPermission(middleware.PermFinanceRead), h.ListChargebacks)
		adm.GET("/:id", middleware.RequireAnyPermission(middleware.PermFinanceRead), h.GetChargeback)
		adm.POST("/:id/evidence", middleware.RequireAnyPermission(middleware.PermFinanceRead), h.SubmitEvidence)
	}
}
