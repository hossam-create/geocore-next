package kyc

  import (
  	"fmt"
  	"net/http"
  	"time"

  	"github.com/gin-gonic/gin"
  	"github.com/google/uuid"
  	"gorm.io/gorm"
  )

  type Handler struct{ db *gorm.DB }

  func NewHandler(db *gorm.DB) *Handler { return &Handler{db: db} }

  func (h *Handler) Submit(c *gin.Context) {
  	userID := c.GetString("user_id")
  	userUUID, _ := uuid.Parse(userID)
  	var req struct {
  		FullName    string `json:"full_name" binding:"required"`
  		IDNumber    string `json:"id_number" binding:"required"`
  		Country     string `json:"country" binding:"required,len=3"`
  		Nationality string `json:"nationality" binding:"required,len=3"`
  		DateOfBirth string `json:"date_of_birth" binding:"required"`
  		Documents   []struct {
  			DocumentType string `json:"document_type" binding:"required"`
  			FileURL      string `json:"file_url" binding:"required,url"`
  			FileKey      string `json:"file_key"`
  			MimeType     string `json:"mime_type"`
  			Side         string `json:"side"`
  		} `json:"documents" binding:"required,min=2"`
  	}
  	if err := c.ShouldBindJSON(&req); err != nil {
  		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
  		return
  	}
  	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
  	if err != nil {
  		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_of_birth, use YYYY-MM-DD"})
  		return
  	}
  	var profile KYCProfile
  	if res := h.db.Where("user_id = ?", userUUID).First(&profile); res.Error == gorm.ErrRecordNotFound {
  		profile = KYCProfile{
  			UserID:      userUUID,
  			Status:      StatusPending,
  			FullName:    req.FullName,
  			IDNumber:    req.IDNumber,
  			Country:     req.Country,
  			Nationality: req.Nationality,
  			DateOfBirth: &dob,
  		}
  		h.db.Create(&profile)
  	} else {
  		h.db.Model(&profile).Updates(map[string]any{
  			"status":            StatusPending,
  			"full_name":         req.FullName,
  			"id_number":         req.IDNumber,
  			"country":           req.Country,
  			"nationality":       req.Nationality,
  			"date_of_birth":     dob,
  			"rejection_reason":  "",
  		})
  	}
  	h.db.Where("kyc_profile_id = ?", profile.ID).Delete(&KYCDocument{})
  	for _, d := range req.Documents {
  		side := d.Side
  		if side == "" { side = "front" }
  		h.db.Create(&KYCDocument{
  			KYCProfileID: profile.ID,
  			DocumentType: d.DocumentType,
  			FileURL:      d.FileURL,
  			FileKey:      d.FileKey,
  			MimeType:     d.MimeType,
  			Side:         side,
  		})
  	}
  	c.JSON(http.StatusCreated, gin.H{"message": "KYC submitted for review.", "kyc_id": profile.ID})
  }

  func (h *Handler) Status(c *gin.Context) {
  	userID := c.GetString("user_id")
  	var profile KYCProfile
  	if err := h.db.Preload("Documents").Where("user_id = ?", userID).First(&profile).Error; err != nil {
  		c.JSON(http.StatusOK, gin.H{"status": "not_submitted", "message": "No KYC submission found."})
  		return
  	}
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
  	q.Offset((page-1)*perPage).Limit(perPage).Order("created_at DESC").Find(&profiles)
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
  	c.JSON(http.StatusOK, gin.H{"data": profile})
  }

  func (h *Handler) AdminApprove(c *gin.Context) {
  	adminID, _ := uuid.Parse(c.GetString("user_id"))
  	var req struct {
  		Notes   string `json:"notes"`
  		Expires string `json:"expires"`
  	}
  	c.ShouldBindJSON(&req)
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
  	result := h.db.Model(&KYCProfile{}).Where("id = ?", c.Param("id")).
  		Updates(map[string]any{"status": StatusRejected, "rejection_reason": req.Reason})
  	if result.RowsAffected == 0 {
  		c.JSON(http.StatusNotFound, gin.H{"error": "KYC profile not found"})
  		return
  	}
  	h.db.Create(&KYCAuditLog{ProfileID: mustParseUUID(c.Param("id")), AdminID: adminID, Action: "rejected", Notes: req.Reason})
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
  