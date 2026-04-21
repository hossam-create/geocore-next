package kyc

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/geocore-next/backend/internal/security"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct{ db *gorm.DB }

func NewHandler(db *gorm.DB) *Handler { return &Handler{db: db} }

func (h *Handler) fieldKeyOrError() ([]byte, error) {
	key, err := security.FieldEncryptionKey()
	if err != nil {
		if os.Getenv("APP_ENV") == "production" {
			return nil, err
		}
		return nil, nil
	}
	return key, nil
}

func decryptProfilePII(profile *KYCProfile, key []byte) {
	if key == nil {
		return
	}
	if profile.FullName != "" {
		if pt, err := security.DecryptField(profile.FullName, key); err == nil {
			profile.FullName = pt
		}
	}
	if profile.IDNumber != "" {
		if pt, err := security.DecryptField(profile.IDNumber, key); err == nil {
			profile.IDNumber = pt
		}
	}
}

func (h *Handler) Submit(c *gin.Context) {
	userID := c.GetString("user_id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	var req struct {
		FullName    string `json:"full_name" binding:"required"`
		IDNumber    string `json:"id_number" binding:"required"`
		Country     string `json:"country" binding:"required,len=3"`
		Nationality string `json:"nationality" binding:"required,len=3"`
		DateOfBirth string `json:"date_of_birth" binding:"required"`

		Documents []struct {
			DocumentType string `json:"document_type" binding:"required"`
			FileURL      string `json:"file_url" binding:"required,url"`
			FileKey      string `json:"file_key"`
			MimeType     string `json:"mime_type"`
			Side         string `json:"side"`
		} `json:"documents" binding:"required,min=2"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Sanitize PII fields (defence-in-depth; still stored as plain text unless encrypted at field level)
	req.FullName = security.SanitizeText(req.FullName)
	req.IDNumber = security.SanitizeText(req.IDNumber)
	req.Country = security.SanitizeText(req.Country)
	req.Nationality = security.SanitizeText(req.Nationality)
	req.DateOfBirth = security.SanitizeText(req.DateOfBirth)
	for i := range req.Documents {
		req.Documents[i].DocumentType = security.SanitizeText(req.Documents[i].DocumentType)
		req.Documents[i].FileURL = security.SanitizeText(req.Documents[i].FileURL)
		req.Documents[i].FileKey = security.SanitizeText(req.Documents[i].FileKey)
		req.Documents[i].MimeType = security.SanitizeText(req.Documents[i].MimeType)
		req.Documents[i].Side = security.SanitizeText(req.Documents[i].Side)
	}

	if req.FullName == "" || req.IDNumber == "" {
		response.BadRequest(c, "full_name and id_number are required")
		return
	}
	if len(req.Country) != 3 || len(req.Nationality) != 3 {
		response.BadRequest(c, "country and nationality must be 3-letter codes")
		return
	}
	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		response.BadRequest(c, "invalid date_of_birth, use YYYY-MM-DD")
		return
	}

	key, keyErr := h.fieldKeyOrError()
	if keyErr != nil {
		response.InternalError(c, keyErr)
		return
	}

	encFullName, encErr := req.FullName, error(nil)
	encIDNumber := req.IDNumber
	if key != nil {
		encFullName, encErr = security.EncryptField(req.FullName, key)
		if encErr != nil {
			response.InternalError(c, encErr)
			return
		}
		encIDNumber, encErr = security.EncryptField(req.IDNumber, key)
		if encErr != nil {
			response.InternalError(c, encErr)
			return
		}
	}

	var profile KYCProfile
	if res := h.db.Where("user_id = ?", userUUID).First(&profile); res.Error == gorm.ErrRecordNotFound {
		profile = KYCProfile{
			UserID:      userUUID,
			Status:      StatusPending,
			FullName:    encFullName,
			IDNumber:    encIDNumber,
			Country:     req.Country,
			Nationality: req.Nationality,
			DateOfBirth: &dob,
		}
		h.db.Create(&profile)
	} else {
		h.db.Model(&profile).Updates(map[string]any{
			"status":           StatusPending,
			"full_name":        encFullName,
			"id_number":        encIDNumber,
			"country":          req.Country,
			"nationality":      req.Nationality,
			"date_of_birth":    dob,
			"rejection_reason": "",
		})
	}
	h.db.Where("kyc_profile_id = ?", profile.ID).Delete(&KYCDocument{})
	for _, d := range req.Documents {
		side := d.Side
		if side == "" {
			side = "front"
		}
		h.db.Create(&KYCDocument{
			KYCProfileID: profile.ID,
			DocumentType: d.DocumentType,
			FileURL:      d.FileURL,
			FileKey:      d.FileKey,
			MimeType:     d.MimeType,
			Side:         side,
		})
	}

	security.LogEvent(h.db, c, &userUUID, security.EventKYCSubmitted, map[string]any{
		"kyc_id":      profile.ID.String(),
		"country":     req.Country,
		"nationality": req.Nationality,
		"docs":        len(req.Documents),
	})

	response.Created(c, gin.H{"message": "KYC submitted for review.", "kyc_id": profile.ID})
}

func (h *Handler) Status(c *gin.Context) {
	userID := c.GetString("user_id")
	var profile KYCProfile
	if err := h.db.Preload("Documents").Where("user_id = ?", userID).First(&profile).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "not_submitted", "message": "No KYC submission found."})
		return
	}
	key, _ := h.fieldKeyOrError()
	decryptProfilePII(&profile, key)
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) AdminList(c *gin.Context) {
	page, perPage := 1, 20
	fmt.Sscan(c.DefaultQuery("page", "1"), &page)
	fmt.Sscan(c.DefaultQuery("per_page", "20"), &perPage)
	q := h.db.Model(&KYCProfile{}).Preload("Documents")
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if country := c.Query("country"); country != "" {
		q = q.Where("country = ?", country)
	}
	var total int64
	q.Count(&total)
	var profiles []KYCProfile
	q.Offset((page - 1) * perPage).Limit(perPage).Order("created_at DESC").Find(&profiles)
	key, _ := h.fieldKeyOrError()
	for i := range profiles {
		decryptProfilePII(&profiles[i], key)
	}
	c.JSON(http.StatusOK, gin.H{
		"data": profiles,
		"meta": gin.H{"total": total, "page": page, "per_page": perPage, "pages": (total + int64(perPage) - 1) / int64(perPage)},
	})
}

func (h *Handler) AdminGetOne(c *gin.Context) {
	var profile KYCProfile
	if err := h.db.Preload("Documents").Where("id = ?", c.Param("id")).First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "KYC profile not found"})
		return
	}
	key, _ := h.fieldKeyOrError()
	decryptProfilePII(&profile, key)
	c.JSON(http.StatusOK, gin.H{"data": profile})
}

func (h *Handler) AdminApprove(c *gin.Context) {
	adminID, _ := uuid.Parse(c.GetString("user_id"))
	var req struct {
		Notes   string `json:"notes"`
		Expires string `json:"expires"`
	}
	c.ShouldBindJSON(&req)
	req.Notes = security.SanitizeText(req.Notes)
	now := time.Now()
	updates := map[string]any{
		"status":           StatusApproved,
		"approved_at":      now,
		"approved_by_id":   adminID,
		"rejection_reason": "",
	}
	if req.Expires != "" {
		if exp, err := time.Parse("2006-01-02", req.Expires); err == nil {
			updates["expires_at"] = exp
		}
	}
	result := h.db.Model(&KYCProfile{}).Where("id = ?", c.Param("id")).Updates(updates)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "KYC profile not found"})
		return
	}
	h.db.Model(&KYCDocument{}).Where("kyc_profile_id = ?", c.Param("id")).Update("verified", true)
	h.db.Create(&KYCAuditLog{ProfileID: mustParseUUID(c.Param("id")), AdminID: adminID, Action: "approved", Notes: req.Notes})
	security.LogEvent(h.db, c, &adminID, security.EventAdminAction, map[string]any{
		"action":     "kyc_approved",
		"profile_id": c.Param("id"),
	})
	c.JSON(http.StatusOK, gin.H{"message": "KYC approved."})
}

func (h *Handler) AdminReject(c *gin.Context) {
	adminID, _ := uuid.Parse(c.GetString("user_id"))
	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Reason = security.SanitizeText(req.Reason)
	result := h.db.Model(&KYCProfile{}).Where("id = ?", c.Param("id")).
		Updates(map[string]any{"status": StatusRejected, "rejection_reason": req.Reason})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "KYC profile not found"})
		return
	}
	h.db.Create(&KYCAuditLog{ProfileID: mustParseUUID(c.Param("id")), AdminID: adminID, Action: "rejected", Notes: req.Reason})
	security.LogEvent(h.db, c, &adminID, security.EventAdminAction, map[string]any{
		"action":     "kyc_rejected",
		"profile_id": c.Param("id"),
	})
	c.JSON(http.StatusOK, gin.H{"message": "KYC rejected."})
}

func (h *Handler) AdminStats(c *gin.Context) {
	stats := gin.H{}
	for _, s := range []string{StatusPending, StatusUnderReview, StatusApproved, StatusRejected} {
		var count int64
		h.db.Model(&KYCProfile{}).Where("status = ?", s).Count(&count)
		stats[s] = count
	}
	var total int64
	h.db.Model(&KYCProfile{}).Count(&total)
	stats["total"] = total
	c.JSON(http.StatusOK, gin.H{"data": stats})
}

func RequireKYC(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		var profile KYCProfile
		if err := db.Select("status").Where("user_id = ? AND status = ?", userID, StatusApproved).First(&profile).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "KYC verification required. Please complete identity verification.",
				"code":  "KYC_REQUIRED",
			})
			return
		}
		c.Next()
	}
}

func mustParseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
