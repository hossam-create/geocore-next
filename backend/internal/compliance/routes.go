package compliance

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes mounts GDPR / consent / audit endpoints.
//
// Public:  GET  /meta/disclaimer
// User:    GET  /user/data-export             (Auth)
//          DELETE /user/delete-account         (Auth)
//          POST /user/consent                  (Auth)
//          GET  /user/consent                  (Auth)
// Admin:   GET  /admin/compliance/audit        (Admin)
//          GET  /admin/compliance/audit/verify (Admin)
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	// Public
	rg.GET("/meta/disclaimer", DisclaimerHandler)

	// User-facing (must be authenticated)
	user := rg.Group("/user")
	user.Use(middleware.Auth())
	{
		user.GET("/data-export", h.DataExportHandler)
		user.DELETE("/delete-account", h.DeleteAccountHandler)
		user.POST("/consent", h.PostConsentHandler)
		user.GET("/consent", h.GetConsentHandler)
	}

	// Admin
	adm := rg.Group("/admin/compliance")
	adm.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		adm.GET("/audit", h.AdminAuditHandler)
		adm.GET("/audit/verify", h.AdminVerifyChainHandler)
	}
}
