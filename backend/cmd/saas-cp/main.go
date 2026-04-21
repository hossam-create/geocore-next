// cmd/saas-cp — GeoCore SaaS Control Plane HTTP server.
//
// Manages tenant provisioning, API key lifecycle, billing, and plan administration.
// Runs on port 9000 (SAAS_CP_PORT) and is protected by CONTROL_PLANE_SECRET.
//
// Env vars:
//   DATABASE_URL          — PostgreSQL connection string
//   CONTROL_PLANE_SECRET  — master bearer token for all routes (required in prod)
//   SAAS_CP_PORT          — listening port (default: 9000)
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/geocore-next/backend/internal/authz"
	"github.com/geocore-next/backend/internal/billing"
	"github.com/geocore-next/backend/internal/tenant"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	db := mustConnectDB()
	autoMigrate(db)

	meter := billing.NewMeter(db)
	billing.GlobalMeter = meter
	meter.Start(ctx)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), masterAuth())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "saas-control-plane"})
	})

	mgr := tenant.NewManager(db)
	v1 := r.Group("/v1")
	registerTenants(v1, db, mgr)
	registerBilling(v1, db)
	registerAuth(v1, db)

	port := os.Getenv("SAAS_CP_PORT")
	if port == "" {
		port = "9000"
	}
	srv := &http.Server{Addr: ":" + port, Handler: r, ReadTimeout: 15 * time.Second}

	slog.Info("saas-cp: listening", "port", port)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("saas-cp: server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("saas-cp: shutting down")
	shut, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	srv.Shutdown(shut)
}

// ── Middleware ────────────────────────────────────────────────────────────────

func masterAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := os.Getenv("CONTROL_PLANE_SECRET")
		if secret == "" {
			c.Next()
			return
		}
		auth := c.GetHeader("Authorization")
		// Accept master bearer token
		if auth == "Bearer "+secret {
			c.Next()
			return
		}
		// Accept signed control-plane JWT
		if len(auth) > 7 {
			if claims, err := authz.ValidateToken(auth[7:]); err == nil {
				c.Set("cp_tenant_id", claims.TenantID)
				c.Set("cp_role", claims.Role)
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid control plane credentials"})
	}
}

// ── Route registration ────────────────────────────────────────────────────────

func registerTenants(v1 *gin.RouterGroup, db *gorm.DB, mgr *tenant.Manager) {
	g := v1.Group("/tenants")

	g.POST("", func(c *gin.Context) {
		var body struct {
			Name   string `json:"name"  binding:"required"`
			Email  string `json:"email" binding:"required"`
			PlanID string `json:"plan_id"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		if body.PlanID == "" {
			body.PlanID = "starter"
		}
		t, err := mgr.Create(body.Name, body.Email, body.PlanID)
		if err != nil {
			response.InternalError(c, err)
			return
		}
		key, _ := authz.CreateKey(db, t.ID, "default", "owner")
		response.OK(c, gin.H{"tenant": t, "api_key": key})
	})

	g.GET("", func(c *gin.Context) {
		tenants, count, err := mgr.List(50, 0)
		if err != nil {
			response.InternalError(c, err)
			return
		}
		response.OK(c, gin.H{"tenants": tenants, "total": count})
	})

	g.GET("/:id", func(c *gin.Context) {
		t, err := mgr.Get(c.Param("id"))
		if err != nil {
			response.NotFound(c, "tenant")
			return
		}
		response.OK(c, t)
	})

	g.PATCH("/:id/plan", func(c *gin.Context) {
		var body struct {
			PlanID string `json:"plan_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		if err := mgr.UpdatePlan(c.Param("id"), body.PlanID); err != nil {
			response.InternalError(c, err)
			return
		}
		response.OK(c, gin.H{"message": "plan updated", "plan_id": body.PlanID})
	})

	g.POST("/:id/suspend", func(c *gin.Context) {
		if err := mgr.Suspend(c.Param("id")); err != nil {
			response.InternalError(c, err)
			return
		}
		response.OK(c, gin.H{"message": "tenant suspended"})
	})

	g.GET("/:id/usage", func(c *gin.Context) {
		end := time.Now()
		start := end.Add(-30 * 24 * time.Hour)
		response.OK(c, billing.Summarize(db, c.Param("id"), start, end))
	})
}

func registerBilling(v1 *gin.RouterGroup, db *gorm.DB) {
	g := v1.Group("/billing")

	g.GET("/plans", func(c *gin.Context) {
		plans := make([]billing.Plan, 0, len(billing.Catalog))
		for _, p := range billing.Catalog {
			plans = append(plans, p)
		}
		sort.Slice(plans, func(i, j int) bool {
			return plans[i].MonthlyPrice < plans[j].MonthlyPrice
		})
		response.OK(c, plans)
	})

	g.GET("/invoice/:tenant_id", func(c *gin.Context) {
		var t tenant.Tenant
		if err := db.First(&t, "id = ?", c.Param("tenant_id")).Error; err != nil {
			response.NotFound(c, "tenant")
			return
		}
		inv, err := billing.CurrentInvoice(db, t.ID, billing.Get(billing.PlanID(t.Plan)))
		if err != nil {
			response.InternalError(c, err)
			return
		}
		response.OK(c, inv)
	})

	g.POST("/usage", func(c *gin.Context) {
		var body struct {
			TenantID  string `json:"tenant_id"  binding:"required"`
			EventType string `json:"event_type" binding:"required"`
			Quantity  int64  `json:"quantity"   binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		if billing.GlobalMeter != nil {
			billing.GlobalMeter.Record(body.TenantID, billing.EventType(body.EventType), body.Quantity)
		}
		response.OK(c, gin.H{"recorded": true})
	})
}

func registerAuth(v1 *gin.RouterGroup, db *gorm.DB) {
	g := v1.Group("/tenants/:id/api-keys")

	g.POST("", func(c *gin.Context) {
		var body struct {
			Name string `json:"name" binding:"required"`
			Role string `json:"role"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		if body.Role == "" {
			body.Role = "dev"
		}
		key, err := authz.CreateKey(db, c.Param("id"), body.Name, body.Role)
		if err != nil {
			response.InternalError(c, err)
			return
		}
		response.OK(c, key)
	})

	g.GET("", func(c *gin.Context) {
		keys, err := authz.ListKeys(db, c.Param("id"))
		if err != nil {
			response.InternalError(c, err)
			return
		}
		response.OK(c, keys)
	})

	g.DELETE("/:key_id", func(c *gin.Context) {
		if err := authz.RevokeKey(db, c.Param("key_id"), c.Param("id")); err != nil {
			response.NotFound(c, "api_key")
			return
		}
		response.OK(c, gin.H{"message": "api key revoked"})
	})

	// POST /v1/tenants/:id/api-keys/:key_id/token — issue JWT for the key's role
	v1.POST("/tenants/:id/token", func(c *gin.Context) {
		var body struct {
			Role string `json:"role"`
		}
		_ = c.ShouldBindJSON(&body)
		if body.Role == "" {
			body.Role = "dev"
		}
		tok, err := authz.IssueToken(c.Param("id"), body.Role, 24*time.Hour)
		if err != nil {
			response.InternalError(c, err)
			return
		}
		response.OK(c, gin.H{"token": tok, "expires_in": "24h"})
	})
}

// ── Infrastructure ────────────────────────────────────────────────────────────

func mustConnectDB() *gorm.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/geocore_dev?sslmode=disable"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		slog.Error("saas-cp: db connection failed", "error", err)
		os.Exit(1)
	}
	return db
}

func autoMigrate(db *gorm.DB) {
	if err := db.AutoMigrate(
		&tenant.Tenant{},
		&authz.APIKey{},
		&billing.UsageEvent{},
		&billing.Invoice{},
	); err != nil {
		slog.Warn("saas-cp: auto-migrate warning", "error", fmt.Sprintf("%v", err))
	}
}
