package fraud

import (
	"log/slog"
	"time"

	"github.com/geocore-next/backend/pkg/events"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct{ db *gorm.DB }

func NewHandler(db *gorm.DB) *Handler { return &Handler{db: db} }

// ══════════════════════════════════════════════════════════════════════════════
// Alerts
// ══════════════════════════════════════════════════════════════════════════════

// GET /api/v1/admin/fraud/alerts
func (h *Handler) ListAlerts(c *gin.Context) {
	var alerts []FraudAlert
	q := h.db.Order("created_at DESC").Limit(100)

	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	if sev := c.Query("severity"); sev != "" {
		q = q.Where("severity = ?", sev)
	}
	if tt := c.Query("target_type"); tt != "" {
		q = q.Where("target_type = ?", tt)
	}

	if err := q.Find(&alerts).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, alerts)
}

// GET /api/v1/admin/fraud/alerts/:id
func (h *Handler) GetAlert(c *gin.Context) {
	var alert FraudAlert
	if err := h.db.Where("id = ?", c.Param("id")).First(&alert).Error; err != nil {
		response.NotFound(c, "fraud alert")
		return
	}
	response.OK(c, alert)
}

// PATCH /api/v1/admin/fraud/alerts/:id
func (h *Handler) UpdateAlert(c *gin.Context) {
	var body struct {
		Status     string `json:"status"`
		Resolution string `json:"resolution"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var alert FraudAlert
	if err := h.db.Where("id = ?", c.Param("id")).First(&alert).Error; err != nil {
		response.NotFound(c, "fraud alert")
		return
	}

	updates := map[string]any{}
	if body.Status != "" {
		updates["status"] = body.Status
	}
	if body.Resolution != "" {
		updates["resolution"] = body.Resolution
	}

	reviewerID, _ := uuid.Parse(c.GetString("user_id"))
	now := time.Now()
	updates["reviewed_by"] = reviewerID
	updates["reviewed_at"] = now

	h.db.Model(&alert).Updates(updates)
	response.OK(c, gin.H{"message": "alert updated"})
}

// GET /api/v1/admin/fraud/stats
func (h *Handler) Stats(c *gin.Context) {
	type CountResult struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var byStatus []CountResult
	h.db.Model(&FraudAlert{}).Select("status, count(*) as count").Group("status").Scan(&byStatus)

	var bySeverity []struct {
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}
	h.db.Model(&FraudAlert{}).Select("severity, count(*) as count").Group("severity").Scan(&bySeverity)

	var total int64
	h.db.Model(&FraudAlert{}).Count(&total)

	var last24h int64
	h.db.Model(&FraudAlert{}).Where("created_at > ?", time.Now().Add(-24*time.Hour)).Count(&last24h)

	response.OK(c, gin.H{
		"total":       total,
		"last_24h":    last24h,
		"by_status":   byStatus,
		"by_severity": bySeverity,
	})
}

// ══════════════════════════════════════════════════════════════════════════════
// Rules
// ══════════════════════════════════════════════════════════════════════════════

// GET /api/v1/admin/fraud/rules
func (h *Handler) ListRules(c *gin.Context) {
	var rules []FraudRule
	if err := h.db.Order("created_at ASC").Find(&rules).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, rules)
}

// PATCH /api/v1/admin/fraud/rules/:id
func (h *Handler) UpdateRule(c *gin.Context) {
	var body struct {
		IsActive *bool   `json:"is_active"`
		Severity *string `json:"severity"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	var rule FraudRule
	if err := h.db.Where("id = ?", c.Param("id")).First(&rule).Error; err != nil {
		response.NotFound(c, "fraud rule")
		return
	}
	updates := map[string]any{}
	if body.IsActive != nil {
		updates["is_active"] = *body.IsActive
	}
	if body.Severity != nil {
		updates["severity"] = *body.Severity
	}
	h.db.Model(&rule).Updates(updates)
	response.OK(c, gin.H{"message": "rule updated"})
}

// ══════════════════════════════════════════════════════════════════════════════
// Risk Profiles
// ══════════════════════════════════════════════════════════════════════════════

// GET /api/v1/admin/fraud/risk-profiles
func (h *Handler) ListRiskProfiles(c *gin.Context) {
	var profiles []UserRiskProfile
	q := h.db.Order("risk_score DESC").Limit(50)
	if level := c.Query("risk_level"); level != "" {
		q = q.Where("risk_level = ?", level)
	}
	if err := q.Find(&profiles).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, profiles)
}

// ══════════════════════════════════════════════════════════════════════════════
// Analyze endpoint (callable from other services)
// ══════════════════════════════════════════════════════════════════════════════

// POST /api/v1/fraud/analyze
func (h *Handler) Analyze(c *gin.Context) {
	var body struct {
		Amount          float64 `json:"amount" binding:"required"`
		UserID          string  `json:"user_id" binding:"required"`
		AccountAgeHours float64 `json:"account_age_hours"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var profile UserRiskProfile
	h.db.Where("user_id = ?", body.UserID).First(&profile)

	result := AnalyzeTransaction(body.Amount, profile.TotalOrders, profile.AvgOrderValue, body.AccountAgeHours)

	if result.RiskScore >= 70 {
		targetID, _ := uuid.Parse(body.UserID)
		alert := FraudAlert{
			TargetType: TargetUser,
			TargetID:   targetID,
			AlertType:  "automated_risk_check",
			Severity:   Severity(result.RiskLevel),
			RiskScore:  result.RiskScore,
			DetectedBy: "rule_engine",
			Confidence: result.RiskScore / 100,
			Indicators: "[]",
			Status:     AlertPending,
		}
		if err := h.db.Create(&alert).Error; err != nil {
			slog.Error("fraud: failed to create alert", "error", err.Error())
		}
	}

	// Publish domain event for in-process consumers
	events.Publish(events.Event{
		Type: events.EventFraudChecked,
		Payload: map[string]interface{}{
			"user_id":    body.UserID,
			"risk_score": result.RiskScore,
			"decision":   result.Decision,
		},
	})

	// Transactional outbox for Kafka delivery
	_ = kafka.WriteOutbox(h.db, kafka.TopicFraud, kafka.New(
		"fraud.checked",
		body.UserID,
		"fraud",
		kafka.Actor{Type: "system", ID: "rule-engine"},
		map[string]interface{}{
			"user_id":    body.UserID,
			"risk_score": result.RiskScore,
			"decision":   result.Decision,
		},
		kafka.EventMeta{Source: "fraud-service"},
	))

	response.OK(c, result)
}
