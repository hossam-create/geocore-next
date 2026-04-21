package ads

import (
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Handler holds the DB reference for ad management.
type Handler struct {
	db *gorm.DB
}

// NewHandler creates a new ads handler.
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// ════════════════════════════════════════════════════════════════════════════
// Admin endpoints (require auth + admin role)
// ════════════════════════════════════════════════════════════════════════════

// ListAds returns all ads for admin, filterable by placement and status.
// GET /admin/ads
func (h *Handler) ListAds(c *gin.Context) {
	q := h.db.Model(&Ad{}).Order("placement ASC, position ASC, created_at DESC")

	if p := c.Query("placement"); p != "" {
		q = q.Where("placement = ?", p)
	}
	if s := c.Query("status"); s != "" {
		switch s {
		case "active":
			q = q.Where("enabled = true")
		case "disabled":
			q = q.Where("enabled = false")
		case "scheduled":
			q = q.Where("enabled = true AND start_date > ?", time.Now())
		case "expired":
			q = q.Where("end_date IS NOT NULL AND end_date < ?", time.Now())
		}
	}

	var ads []Ad
	q.Find(&ads)
	response.OK(c, ads)
}

// CreateAd creates a new banner ad.
// POST /admin/ads
func (h *Handler) CreateAd(c *gin.Context) {
	var req AdCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ad := Ad{
		Title:     req.Title,
		ImageURL:  req.ImageURL,
		LinkURL:   req.LinkURL,
		Placement: req.Placement,
		Position:  req.Position,
		Enabled:   true,
	}

	if req.Enabled != nil {
		ad.Enabled = *req.Enabled
	}

	if req.StartDate != nil {
		t, err := time.Parse(time.RFC3339, *req.StartDate)
		if err != nil {
			response.BadRequest(c, "Invalid start_date format (use RFC3339)")
			return
		}
		ad.StartDate = &t
	}
	if req.EndDate != nil {
		t, err := time.Parse(time.RFC3339, *req.EndDate)
		if err != nil {
			response.BadRequest(c, "Invalid end_date format (use RFC3339)")
			return
		}
		ad.EndDate = &t
	}

	if ad.StartDate != nil && ad.EndDate != nil && ad.EndDate.Before(*ad.StartDate) {
		response.BadRequest(c, "end_date must be after start_date")
		return
	}

	if userID := c.GetString("user_id"); userID != "" {
		uid, _ := uuid.Parse(userID)
		ad.CreatedBy = &uid
	}

	if err := h.db.Create(&ad).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.Created(c, ad)
}

// UpdateAd updates an existing ad.
// PUT /admin/ads/:id
func (h *Handler) UpdateAd(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ad ID")
		return
	}

	var ad Ad
	if err := h.db.First(&ad, "id = ?", id).Error; err != nil {
		response.NotFound(c, "Ad")
		return
	}

	var req AdUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updates := map[string]interface{}{}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.ImageURL != nil {
		updates["image_url"] = *req.ImageURL
	}
	if req.LinkURL != nil {
		updates["link_url"] = *req.LinkURL
	}
	if req.Placement != nil {
		updates["placement"] = *req.Placement
	}
	if req.Position != nil {
		updates["position"] = *req.Position
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.StartDate != nil {
		t, err := time.Parse(time.RFC3339, *req.StartDate)
		if err != nil {
			response.BadRequest(c, "Invalid start_date format")
			return
		}
		updates["start_date"] = t
	}
	if req.EndDate != nil {
		t, err := time.Parse(time.RFC3339, *req.EndDate)
		if err != nil {
			response.BadRequest(c, "Invalid end_date format")
			return
		}
		updates["end_date"] = t
	}

	h.db.Model(&ad).Updates(updates)
	h.db.First(&ad, "id = ?", id) // reload
	response.OK(c, ad)
}

// DeleteAd soft-deletes an ad.
// DELETE /admin/ads/:id
func (h *Handler) DeleteAd(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ad ID")
		return
	}

	result := h.db.Where("id = ?", id).Delete(&Ad{})
	if result.RowsAffected == 0 {
		response.NotFound(c, "Ad")
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

// ToggleAd enables or disables an ad.
// PATCH /admin/ads/:id/toggle
func (h *Handler) ToggleAd(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ad ID")
		return
	}

	var ad Ad
	if err := h.db.First(&ad, "id = ?", id).Error; err != nil {
		response.NotFound(c, "Ad")
		return
	}

	ad.Enabled = !ad.Enabled
	h.db.Model(&ad).Update("enabled", ad.Enabled)
	response.OK(c, ad)
}

// ════════════════════════════════════════════════════════════════════════════
// Public endpoints
// ════════════════════════════════════════════════════════════════════════════

// GetPublicAds returns active ads for a given placement.
// GET /ads?placement=hero
func (h *Handler) GetPublicAds(c *gin.Context) {
	placement := c.Query("placement")
	if placement == "" {
		response.BadRequest(c, "placement query parameter is required")
		return
	}

	now := time.Now()
	var ads []Ad
	h.db.Where("enabled = true AND placement = ? AND (start_date IS NULL OR start_date <= ?) AND (end_date IS NULL OR end_date >= ?)",
		placement, now, now).
		Order("position ASC").
		Find(&ads)

	// Increment view counts asynchronously
	ids := make([]uuid.UUID, len(ads))
	for i, a := range ads {
		ids[i] = a.ID
	}
	if len(ids) > 0 {
		go func() {
			h.db.Model(&Ad{}).Where("id IN ?", ids).UpdateColumn("view_count", gorm.Expr("view_count + 1"))
		}()
	}

	response.OK(c, ads)
}

// TrackClick increments the click counter for an ad.
// POST /ads/:id/click
func (h *Handler) TrackClick(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ad ID")
		return
	}

	result := h.db.Model(&Ad{}).Where("id = ? AND enabled = true", id).
		UpdateColumn("click_count", gorm.Expr("click_count + 1"))
	if result.RowsAffected == 0 {
		response.NotFound(c, "Ad")
		return
	}
	response.OK(c, gin.H{"tracked": true})
}
