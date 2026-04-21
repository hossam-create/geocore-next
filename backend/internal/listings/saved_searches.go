package listings

// Sprint 18 — Saved Searches.
// Users can persist search queries + filters, optionally with notify_on_match.
// Additive only. No changes to existing search behavior.

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/config"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SavedSearch persists a user's search query + filter bundle.
type SavedSearch struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null;index:idx_saved_user_created" json:"user_id"`
	Query          string         `gorm:"size:500" json:"query"`
	Label          string         `gorm:"size:200" json:"label,omitempty"`
	Filters        string         `gorm:"type:jsonb;default:'{}'" json:"-"` // raw JSON stored on disk
	NotifyOnMatch  bool           `gorm:"default:false" json:"notify_on_match"`
	LastNotifiedAt *time.Time     `json:"last_notified_at,omitempty"`
	CreatedAt      time.Time      `gorm:"index:idx_saved_user_created,sort:desc" json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (SavedSearch) TableName() string { return "saved_searches" }

// SavedSearchResponse marshals Filters back to JSON for the client.
type SavedSearchResponse struct {
	ID             uuid.UUID              `json:"id"`
	UserID         uuid.UUID              `json:"user_id"`
	Query          string                 `json:"query"`
	Label          string                 `json:"label,omitempty"`
	Filters        map[string]interface{} `json:"filters"`
	NotifyOnMatch  bool                   `json:"notify_on_match"`
	LastNotifiedAt *time.Time             `json:"last_notified_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

func toResponse(s SavedSearch) SavedSearchResponse {
	filters := map[string]interface{}{}
	if s.Filters != "" && s.Filters != "{}" {
		_ = json.Unmarshal([]byte(s.Filters), &filters)
	}
	return SavedSearchResponse{
		ID: s.ID, UserID: s.UserID, Query: s.Query, Label: s.Label,
		Filters: filters, NotifyOnMatch: s.NotifyOnMatch,
		LastNotifiedAt: s.LastNotifiedAt, CreatedAt: s.CreatedAt,
	}
}

// ════════════════════════════════════════════════════════════════════════════
// Handlers
// ════════════════════════════════════════════════════════════════════════════

type saveSearchBody struct {
	Query         string                 `json:"query"`
	Label         string                 `json:"label"`
	Filters       map[string]interface{} `json:"filters"`
	NotifyOnMatch bool                   `json:"notify_on_match"`
}

// POST /api/v1/search/save
func (h *Handler) CreateSavedSearch(c *gin.Context) {
	if !config.GetFlags().EnableSavedSearch {
		response.Conflict(c, "saved search is disabled")
		return
	}
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		response.Unauthorized(c)
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(c, "invalid user")
		return
	}

	var body saveSearchBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	body.Query = strings.TrimSpace(body.Query)
	if body.Query == "" && len(body.Filters) == 0 {
		response.BadRequest(c, "query or filters required")
		return
	}

	// Hard cap 50 saved searches per user to prevent abuse.
	var count int64
	h.writeDB().Model(&SavedSearch{}).Where("user_id = ?", userID).Count(&count)
	if count >= 50 {
		response.RateLimited(c, "max 50 saved searches per user")
		return
	}

	filtersJSON, _ := json.Marshal(body.Filters)
	s := SavedSearch{
		UserID:        userID,
		Query:         body.Query,
		Label:         strings.TrimSpace(body.Label),
		Filters:       string(filtersJSON),
		NotifyOnMatch: body.NotifyOnMatch,
	}
	if err := h.writeDB().Create(&s).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"saved_search": toResponse(s)})
}

// GET /api/v1/search/saved
func (h *Handler) ListSavedSearches(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		response.Unauthorized(c)
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(c, "invalid user")
		return
	}
	var rows []SavedSearch
	if err := h.readDB().Where("user_id = ?", userID).
		Order("created_at DESC").Limit(100).Find(&rows).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	out := make([]SavedSearchResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, toResponse(r))
	}
	response.OK(c, gin.H{"saved_searches": out, "total": len(out)})
}

// DELETE /api/v1/search/saved/:id
func (h *Handler) DeleteSavedSearch(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		response.Unauthorized(c)
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(c, "invalid user")
		return
	}
	sid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	res := h.writeDB().Where("id = ? AND user_id = ?", sid, userID).Delete(&SavedSearch{})
	if res.Error != nil {
		response.InternalError(c, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		response.NotFound(c, "saved search")
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
