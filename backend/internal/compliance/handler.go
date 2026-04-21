package compliance

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler { return &Handler{db: db} }

// ─── GDPR: Data Export ──────────────────────────────────────────────────────

// DataExportHandler — GET /user/data-export
// Returns a JSON bundle of all personal data held about the caller.
// Served as a download (Content-Disposition: attachment).
func (h *Handler) DataExportHandler(c *gin.Context) {
	uid := mustUser(c)
	if uid == uuid.Nil {
		return
	}

	export, err := BuildUserExport(h.db, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Audit the export (GDPR accountability).
	_, _ = LogComplianceEvent(h.db, CategoryGDPR, "data_export", &uid, &uid, uid.String(), c.ClientIP(),
		map[string]any{"size_sections": 12})

	fname := fmt.Sprintf("data-export-%s-%s.json", uid, time.Now().UTC().Format("20060102-150405"))
	c.Header("Content-Disposition", `attachment; filename="`+fname+`"`)
	c.JSON(http.StatusOK, export)
}

// DeleteAccountHandler — DELETE /user/delete-account
// Body: {"confirm":"DELETE","reason":"optional"}
// Implements Right to Erasure: anonymises PII, preserves financial records.
func (h *Handler) DeleteAccountHandler(c *gin.Context) {
	uid := mustUser(c)
	if uid == uuid.Nil {
		return
	}

	var body struct {
		Confirm string `json:"confirm"`
		Reason  string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.Confirm != "DELETE" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "confirmation_required",
			"hint":  `send {"confirm":"DELETE"} to proceed with account erasure`,
		})
		return
	}

	anonEmail, err := AnonymizeUser(h.db, uid, &uid, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"erased":           true,
		"anonymized_email": anonEmail,
		"reason":           body.Reason,
		"notice":           "Your account is now erased. Financial records may be retained up to 7 years for regulatory purposes.",
	})
}

// ─── Consent Tracking ────────────────────────────────────────────────────────

// PostConsentHandler — POST /user/consent
// Body: {"type":"terms|privacy|marketing|cookies","accepted":true,"version":"v1.2"}
func (h *Handler) PostConsentHandler(c *gin.Context) {
	uid := mustUser(c)
	if uid == uuid.Nil {
		return
	}
	var body struct {
		Type     string `json:"type"     binding:"required"`
		Accepted bool   `json:"accepted"`
		Version  string `json:"version"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Version == "" {
		body.Version = "v1"
	}
	switch body.Type {
	case ConsentTerms, ConsentPrivacy, ConsentMarketing, ConsentCookies:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_consent_type"})
		return
	}

	rec := ConsentRecord{
		UserID:    uid,
		Type:      body.Type,
		Accepted:  body.Accepted,
		Version:   body.Version,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		CreatedAt: time.Now().UTC(),
	}
	if err := h.db.Create(&rec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Also chain into immutable audit for regulator-facing proof.
	_, _ = LogComplianceEvent(h.db, CategoryConsent, "consent_recorded",
		&uid, &uid, rec.ID.String(), c.ClientIP(),
		map[string]any{"type": body.Type, "accepted": body.Accepted, "version": body.Version})

	c.JSON(http.StatusOK, rec)
}

// GetConsentHandler — GET /user/consent
// Returns the latest state per consent type (terms / privacy / marketing / cookies).
func (h *Handler) GetConsentHandler(c *gin.Context) {
	uid := mustUser(c)
	if uid == uuid.Nil {
		return
	}
	var rows []ConsentRecord
	// DISTINCT ON — one latest row per type.
	h.db.Raw(`
		SELECT DISTINCT ON (type) *
		FROM consent_records
		WHERE user_id = ?
		ORDER BY type, created_at DESC
	`, uid).Scan(&rows)

	state := map[string]any{}
	for _, r := range rows {
		state[r.Type] = map[string]any{
			"accepted":   r.Accepted,
			"version":    r.Version,
			"created_at": r.CreatedAt,
		}
	}
	c.JSON(http.StatusOK, gin.H{"user_id": uid, "consents": state})
}

// ─── Admin: audit listing + chain verification ───────────────────────────────

// AdminAuditHandler — GET /admin/compliance/audit
// Query: ?category=exchange&action=payout_requested&user_id=...&limit=100&offset=0
func (h *Handler) AdminAuditHandler(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	q := h.db.Model(&ComplianceAuditLog{}).Order("id DESC")
	if v := c.Query("category"); v != "" {
		q = q.Where("category = ?", v)
	}
	if v := c.Query("action"); v != "" {
		q = q.Where("action = ?", v)
	}
	if v := c.Query("user_id"); v != "" {
		if uid, err := uuid.Parse(v); err == nil {
			q = q.Where("user_id = ?", uid)
		}
	}

	var rows []ComplianceAuditLog
	var total int64
	q.Count(&total)
	q.Offset(offset).Limit(limit).Find(&rows)

	c.JSON(http.StatusOK, gin.H{
		"rows":   rows,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// AdminVerifyChainHandler — GET /admin/compliance/audit/verify
// Walks the entire chain; returns first tampered row id or ok.
func (h *Handler) AdminVerifyChainHandler(c *gin.Context) {
	ok, badID, err := VerifyChain(h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"valid":        ok,
		"first_bad_id": badID,
		"verified_at":  time.Now().UTC(),
	})
}

// ─── helpers ────────────────────────────────────────────────────────────────

func mustUser(c *gin.Context) uuid.UUID {
	s := c.GetString("user_id")
	if s == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return uuid.Nil
	}
	u, err := uuid.Parse(s)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id"})
		return uuid.Nil
	}
	return u
}
