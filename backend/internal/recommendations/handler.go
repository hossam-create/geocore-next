package recommendations

import (
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler exposes recommendation HTTP endpoints.
type Handler struct {
	engine *Engine
}

// NewHandler creates a recommendation handler.
func NewHandler(engine *Engine) *Handler {
	return &Handler{engine: engine}
}

// GET /api/v1/recommendations?context=home_page&limit=10&reference_id=xxx
func (h *Handler) Get(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	rctx := Context(c.DefaultQuery("context", string(ContextHomePage)))
	limitStr := c.DefaultQuery("limit", "10")
	var limit int
	if _, err := uuid.Parse(limitStr); err != nil {
		limit = 10
	}
	if n := parseInt(limitStr); n > 0 {
		limit = n
	} else {
		limit = 10
	}

	var refID *uuid.UUID
	if ref := c.Query("reference_id"); ref != "" {
		if id, err := uuid.Parse(ref); err == nil {
			refID = &id
		}
	}

	resp := h.engine.GetRecommendations(c.Request.Context(), userID, rctx, refID, limit)
	response.OK(c, resp)
}

// POST /api/v1/recommendations/track
func (h *Handler) Track(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req struct {
		ListingID       string `json:"listing_id" binding:"required"`
		InteractionType string `json:"interaction_type" binding:"required"`
		CategoryID      string `json:"category_id"`
		SessionID       string `json:"session_id"`
		DwellTimeMs     int    `json:"dwell_time_ms"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	listingID, err := uuid.Parse(req.ListingID)
	if err != nil {
		response.BadRequest(c, "invalid listing_id")
		return
	}

	var catID *uuid.UUID
	if req.CategoryID != "" {
		if id, err := uuid.Parse(req.CategoryID); err == nil {
			catID = &id
		}
	}

	h.engine.TrackInteraction(userID, listingID, req.InteractionType, catID, req.SessionID, req.DwellTimeMs)
	response.OK(c, gin.H{"tracked": true})
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			return 0
		}
	}
	return n
}
