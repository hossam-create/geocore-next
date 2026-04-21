package warroom

import (
	"context"
	"net/http"

	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type apiHandler struct{ ctrl *Controller }

// GET /api/v1/warroom/state
func (h *apiHandler) State(c *gin.Context) {
	response.OK(c, gin.H{"state": h.ctrl.State()})
}

// GET /api/v1/warroom/dashboard
func (h *apiHandler) Dashboard(c *gin.Context) {
	response.OK(c, h.ctrl.Dashboard())
}

// POST /api/v1/warroom/lockdown
func (h *apiHandler) Lockdown(c *gin.Context) {
	var body struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.ctrl.ManualTransition(StateLockdown, body.Reason)
	response.OK(c, gin.H{"message": "system locked down", "state": StateLockdown})
}

// POST /api/v1/warroom/recover
func (h *apiHandler) Recover(c *gin.Context) {
	h.ctrl.ManualTransition(StateNormal, "operator initiated recovery")
	response.OK(c, gin.H{"message": "system recovering", "state": StateNormal})
}

// POST /api/v1/warroom/degrade
func (h *apiHandler) Degrade(c *gin.Context) {
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.Reason == "" {
		body.Reason = "manual degradation"
	}
	h.ctrl.ManualTransition(StateDegraded, body.Reason)
	response.OK(c, gin.H{"message": "system degraded", "state": StateDegraded})
}

// POST /api/v1/warroom/actions/:id/approve
func (h *apiHandler) ApproveAction(c *gin.Context) {
	if !h.ctrl.ApproveAction(c.Param("id")) {
		c.JSON(http.StatusNotFound, gin.H{"error": "action not found or not pending"})
		return
	}
	response.OK(c, gin.H{"message": "action approved and executed"})
}

// POST /api/v1/warroom/actions/:id/reject
func (h *apiHandler) RejectAction(c *gin.Context) {
	if !h.ctrl.RejectAction(c.Param("id")) {
		c.JSON(http.StatusNotFound, gin.H{"error": "action not found or not pending"})
		return
	}
	response.OK(c, gin.H{"message": "action rejected"})
}

// RegisterRoutes mounts the War Room control plane under /api/v1/warroom (admin-only).
//
//	GET  /api/v1/warroom/state                   — current system state
//	GET  /api/v1/warroom/dashboard               — full dashboard view
//	POST /api/v1/warroom/lockdown                — enter lockdown
//	POST /api/v1/warroom/recover                 — exit to NORMAL
//	POST /api/v1/warroom/degrade                 — force DEGRADED state
//	POST /api/v1/warroom/actions/:id/approve     — approve a pending action
//	POST /api/v1/warroom/actions/:id/reject      — reject a pending action
func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB, ctx context.Context) {
	ctrl := newController()
	ctrl.Start(ctx)

	h := &apiHandler{ctrl: ctrl}

	g := v1.Group("/warroom",
		middleware.Auth(),
		middleware.AdminWithDB(db),
	)
	{
		g.GET("/state", h.State)
		g.GET("/dashboard", h.Dashboard)
		g.POST("/lockdown", h.Lockdown)
		g.POST("/recover", h.Recover)
		g.POST("/degrade", h.Degrade)
		g.POST("/actions/:id/approve", h.ApproveAction)
		g.POST("/actions/:id/reject", h.RejectAction)
	}
}
