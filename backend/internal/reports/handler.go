package reports

import (
	"fmt"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// CreateReport — POST /api/v1/reports
// Authenticated users submit a report on a listing or user.
func (h *Handler) CreateReport(c *gin.Context) {
	reporterID, _ := uuid.Parse(c.GetString("user_id"))

	var req struct {
		TargetType  string `json:"target_type" binding:"required,oneof=listing user"`
		TargetID    string `json:"target_id"   binding:"required"`
		Reason      string `json:"reason"      binding:"required,min=3,max=100"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		response.BadRequest(c, "invalid target_id")
		return
	}

	report := Report{
		ReporterID:  reporterID,
		TargetType:  TargetType(req.TargetType),
		TargetID:    targetID,
		Reason:      req.Reason,
		Description: req.Description,
		Status:      StatusPending,
	}

	if err := h.db.Create(&report).Error; err != nil {
		response.BadRequest(c, "you have already reported this item")
		return
	}

	response.Created(c, report)
}

// AdminListReports — GET /api/v1/admin/reports
// Returns paginated reports with optional status filter.
func (h *Handler) AdminListReports(c *gin.Context) {
	page, perPage := 1, 20
	if p := c.Query("page"); p != "" {
		fmt.Sscan(p, &page)
	}
	if pp := c.Query("per_page"); pp != "" {
		fmt.Sscan(pp, &perPage)
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	q := h.db.Model(&Report{})
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if targetType := c.Query("target_type"); targetType != "" {
		q = q.Where("target_type = ?", targetType)
	}

	var total int64
	q.Count(&total)

	type ReportWithReporter struct {
		Report
		ReporterName string `json:"reporter_name"`
	}

	var rows []ReportWithReporter
	q.Select("reports.*, u.name as reporter_name").
		Joins("LEFT JOIN users u ON u.id = reports.reporter_id").
		Offset((page - 1) * perPage).Limit(perPage).
		Order("reports.created_at DESC").
		Scan(&rows)

	response.OKMeta(c, rows, response.Meta{
		Total:   total,
		Page:    page,
		PerPage: perPage,
		Pages:   (total + int64(perPage) - 1) / int64(perPage),
	})
}

// AdminReviewReport — PATCH /api/v1/admin/reports/:id
// Admin updates report status and adds an optional note.
func (h *Handler) AdminReviewReport(c *gin.Context) {
	id := c.Param("id")
	adminID, _ := uuid.Parse(c.GetString("user_id"))

	var req struct {
		Status    string `json:"status" binding:"required,oneof=reviewed dismissed actioned"`
		AdminNote string `json:"admin_note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	now := time.Now()
	result := h.db.Model(&Report{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      req.Status,
		"admin_note":  req.AdminNote,
		"reviewed_by": adminID,
		"reviewed_at": now,
		"updated_at":  now,
	})
	if result.RowsAffected == 0 {
		response.NotFound(c, "report")
		return
	}

	response.OK(c, gin.H{"message": "Report updated", "status": req.Status})
}

// AdminGetStats — GET /api/v1/admin/reports/stats
// Returns counts by status for the admin dashboard badge.
func (h *Handler) AdminGetStats(c *gin.Context) {
	type stat struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var stats []stat
	h.db.Model(&Report{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&stats)

	response.OK(c, stats)
}
