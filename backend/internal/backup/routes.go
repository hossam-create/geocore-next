package backup

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes mounts all backup admin endpoints.
// All routes require super_admin / admin role.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	cfg := BackupConfigFromEnv()
	h := NewHandler(db, cfg)

	adm := rg.Group("/admin/system")
	adm.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		adm.GET("/backups", h.ListBackupsHandler)
		adm.POST("/backups/trigger", h.TriggerBackupHandler)
		adm.POST("/backups/validate", h.ValidateHandler)
		adm.POST("/restore", h.RestoreHandler)
	}
}
