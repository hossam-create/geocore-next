package trust

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler provides HTTP handlers for the trust & safety system.
type Handler struct {
	db *gorm.DB
}

// NewHandler creates a new trust handler.
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// EvaluateFraud handles POST /api/internal/fraud/evaluate
// Internal-only endpoint for rule-based fraud evaluation.
func (h *Handler) EvaluateFraud(c *gin.Context) {
	var req EvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp := Evaluate(c.Request.Context(), h.db, req)
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// ListFlags handles GET /api/admin/trust/flags
func (h *Handler) ListFlags(c *gin.Context) {
	var flags []TrustFlag
	q := h.db.Order("created_at DESC")

	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if severity := c.Query("severity"); severity != "" {
		q = q.Where("severity = ?", severity)
	}

	q.Limit(100).Find(&flags)
	c.JSON(http.StatusOK, gin.H{"data": flags})
}

// GetFlag handles GET /api/admin/trust/flags/:id
func (h *Handler) GetFlag(c *gin.Context) {
	id := c.Param("id")
	var flag TrustFlag
	if err := h.db.Where("id = ?", id).First(&flag).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flag not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": flag})
}

// ResolveFlag handles PATCH /api/admin/trust/flags/:id/resolve
func (h *Handler) ResolveFlag(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"`
		Notes  string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := h.db.Model(&TrustFlag{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": req.Status,
		"notes":  req.Notes,
	})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "flag not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "flag updated"})
}

// BulkResolve handles POST /api/admin/trust/flags/bulk-resolve
func (h *Handler) BulkResolve(c *gin.Context) {
	var req struct {
		IDs    []string `json:"ids" binding:"required"`
		Status string   `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.db.Model(&TrustFlag{}).Where("id IN ?", req.IDs).Updates(map[string]interface{}{
		"status": req.Status,
	})
	c.JSON(http.StatusOK, gin.H{"message": "flags updated", "count": len(req.IDs)})
}

// GetStats handles GET /api/admin/trust/stats
func (h *Handler) GetStats(c *gin.Context) {
	var openCount, criticalCount, autoResolvedToday int64

	h.db.Model(&TrustFlag{}).Where("status = 'open'").Count(&openCount)
	h.db.Model(&TrustFlag{}).Where("status = 'open' AND severity = 'critical'").Count(&criticalCount)
	h.db.Model(&TrustFlag{}).Where("status = 'resolved' AND source = 'rule_engine' AND updated_at >= CURRENT_DATE").Count(&autoResolvedToday)

	// Estimate fraud prevented (sum of risk scores * average transaction value)
	var fraudPrevented float64
	h.db.Model(&TrustFlag{}).
		Where("status IN ('resolved', 'false_positive') AND risk_score > 0.7").
		Select("COALESCE(SUM(risk_score * 1000), 0)").Scan(&fraudPrevented)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"open_flags":          openCount,
		"critical_flags":      criticalCount,
		"auto_resolved_today": autoResolvedToday,
		"fraud_prevented_usd": int(fraudPrevented),
	}})
}

// RegisterRoutes wires trust endpoints into the router.
func RegisterRoutes(internal, admin *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	// Internal API (not public)
	internal.POST("/fraud/evaluate", h.EvaluateFraud)

	// Admin API
	admin.GET("/trust/flags", h.ListFlags)
	admin.GET("/trust/flags/:id", h.GetFlag)
	admin.PATCH("/trust/flags/:id/resolve", h.ResolveFlag)
	admin.POST("/trust/flags/bulk-resolve", h.BulkResolve)
	admin.GET("/trust/stats", h.GetStats)
}
