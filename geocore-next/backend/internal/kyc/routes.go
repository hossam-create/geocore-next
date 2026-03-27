package kyc

  import (
  	"github.com/geocore-next/backend/pkg/middleware"
  	"github.com/gin-gonic/gin"
  	"gorm.io/gorm"
  )

  func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
  	h := NewHandler(db)
  	kyc := r.Group("/kyc")
  	{
  		user := kyc.Group("").Use(middleware.Auth())
  		user.POST("/submit", h.Submit)
  		user.GET("/status",  h.Status)

  		adm := kyc.Group("/admin").Use(middleware.Auth(), middleware.AdminWithDB(db))
  		adm.GET("/list",        h.AdminList)
  		adm.GET("/stats",       h.AdminStats)
  		adm.GET("/:id",         h.AdminGetOne)
  		adm.PUT("/:id/approve", h.AdminApprove)
  		adm.PUT("/:id/reject",  h.AdminReject)
  	}
  }
  