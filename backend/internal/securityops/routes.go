package securityops

import (
	"github.com/geocore-next/backend/internal/controltower"
	"github.com/geocore-next/backend/internal/security"
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterRoutes mounts all Sprint 23 security-observability endpoints and
// wires the EmitHook so LogSecurityEvent broadcasts into the controltower bus.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, rdb *redis.Client, ids *security.IDS) {
	// Ensure the event bus is initialised (idempotent).
	controltower.InitEventBus(rdb)

	// Inject shared Redis into the security package for burst / multi-IP rules.
	security.InitSecurityOps(rdb)

	// Wire security events → controltower live feed.
	security.EmitHook = func(p security.EventPayload) {
		controltower.Emit(
			p.EventType,
			mapSeverity(p.Severity),
			p.Message,
			p.UserID,
			p.IP,
		)
	}

	h := NewHandler(db, rdb, ids)

	adm := rg.Group("/admin/security")
	adm.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		// Part 3 — live activity stream (+ last 100)
		adm.GET("/live", h.LiveHandler)

		// Part 4/11 — user risk profile
		adm.GET("/users", h.UsersHandler)
		adm.GET("/users/:userId", h.UserRiskHandler)

		// Part 6 — admin control actions
		adm.POST("/freeze/:userId", h.FreezeHandler)
		adm.POST("/unfreeze/:userId", h.UnfreezeHandler)
		adm.POST("/block-ip", h.BlockIPHandler)
		adm.POST("/unblock-ip", h.UnblockIPHandler)

		// Part 8 — threat overview
		adm.GET("/overview", h.OverviewHandler)
	}

	// Part 7 — system health (sits under /admin/system, matches spec).
	sys := rg.Group("/admin/system")
	sys.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	sys.GET("/health", h.HealthHandler)
}

func mapSeverity(s string) controltower.SeverityLevel {
	switch s {
	case security.SevSecCritical:
		return controltower.SevCritical
	case security.SevSecHigh, security.SevSecMedium:
		return controltower.SevWarning
	default:
		return controltower.SevInfo
	}
}
