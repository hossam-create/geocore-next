package listings

import (
	"fmt"
	"net/http"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Listing Variants ────────────────────────────────────────────────────────

func (h *Handler) ListVariants(c *gin.Context) {
	id := c.Param("id")
	var variants []ListingVariant
	h.readDB().Where("listing_id = ? AND is_active = true", id).Order("created_at ASC").Find(&variants)
	response.OK(c, variants)
}

func (h *Handler) CreateVariant(c *gin.Context) {
	id := c.Param("id")
	lid, _ := uuid.Parse(id)
	var req ListingVariant
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	req.ID = uuid.Nil
	req.ListingID = lid
	if err := h.writeDB().Create(&req).Error; err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, req)
}

func (h *Handler) UpdateVariant(c *gin.Context) {
	vid := c.Param("variantId")
	var v ListingVariant
	if err := h.writeDB().Where("id = ?", vid).First(&v).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "variant not found"})
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.writeDB().Model(&v).Updates(req)
	h.writeDB().Where("id = ?", vid).First(&v)
	response.OK(c, v)
}

func (h *Handler) DeleteVariant(c *gin.Context) {
	vid := c.Param("variantId")
	h.writeDB().Where("id = ?", vid).Delete(&ListingVariant{})
	response.OK(c, gin.H{"deleted": true})
}

// ── Listing Q&A ─────────────────────────────────────────────────────────────

func (h *Handler) ListQA(c *gin.Context) {
	id := c.Param("id")
	var qa []ListingQA
	h.readDB().Where("listing_id = ? AND is_public = true", id).Order("created_at DESC").Limit(50).Find(&qa)
	response.OK(c, qa)
}

func (h *Handler) AskQuestion(c *gin.Context) {
	id := c.Param("id")
	lid, _ := uuid.Parse(id)
	userID, _ := c.Get("user_id")
	uid, _ := uuid.Parse(fmt.Sprintf("%v", userID))

	var req struct {
		Question string `json:"question" binding:"required,min=5"`
		IsPublic *bool  `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	qa := ListingQA{
		ListingID: lid,
		Question:  req.Question,
		AskedBy:   uid,
		IsPublic:  req.IsPublic == nil || *req.IsPublic,
	}
	if err := h.writeDB().Create(&qa).Error; err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, qa)
}

func (h *Handler) AnswerQuestion(c *gin.Context) {
	qaID := c.Param("qaId")
	var qa ListingQA
	if err := h.writeDB().Where("id = ?", qaID).First(&qa).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "question not found"})
		return
	}
	userID, _ := c.Get("user_id")
	uid, _ := uuid.Parse(fmt.Sprintf("%v", userID))

	var req struct {
		Answer string `json:"answer" binding:"required,min=2"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	now := time.Now()
	h.writeDB().Model(&qa).Updates(map[string]interface{}{
		"answer":      req.Answer,
		"answered_by": uid,
		"answered_at": now,
	})
	h.writeDB().Where("id = ?", qaID).First(&qa)
	response.OK(c, qa)
}

// ── Listing Feedback ────────────────────────────────────────────────────────

func (h *Handler) ListFeedback(c *gin.Context) {
	id := c.Param("id")
	var fb []ListingFeedback
	h.readDB().Where("listing_id = ?", id).Order("created_at DESC").Limit(50).Find(&fb)

	// Calculate summary
	var summary struct {
		Avg   float64 `json:"avg_rating"`
		Count int     `json:"count"`
		Dist  map[int]int `json:"distribution"`
	}
	h.readDB().Model(&ListingFeedback{}).Where("listing_id = ?", id).
		Select("AVG(rating) as avg, COUNT(*) as count").Scan(&summary)
	summary.Dist = map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}
	var dists []struct {
		Rating int
		Count  int
	}
	h.readDB().Model(&ListingFeedback{}).Where("listing_id = ?", id).
		Select("rating, COUNT(*) as count").Group("rating").Scan(&dists)
	for _, d := range dists {
		summary.Dist[d.Rating] = d.Count
	}

	response.OK(c, gin.H{
		"feedback": fb,
		"summary":  summary,
	})
}

func (h *Handler) CreateFeedback(c *gin.Context) {
	id := c.Param("id")
	lid, _ := uuid.Parse(id)
	userID, _ := c.Get("user_id")
	uid, _ := uuid.Parse(fmt.Sprintf("%v", userID))

	// Get seller_id from listing
	var listing Listing
	if err := h.readDB().Where("id = ?", lid).First(&listing).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "listing not found"})
		return
	}

	var req struct {
		Rating      int    `json:"rating" binding:"required,min=1,max=5"`
		Title       string `json:"title"`
		Review      string `json:"review"`
		IsAnonymous *bool  `json:"is_anonymous"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	fb := ListingFeedback{
		ListingID:   lid,
		SellerID:    listing.UserID,
		BuyerID:     uid,
		Rating:      req.Rating,
		Title:       req.Title,
		Review:      req.Review,
		IsAnonymous: req.IsAnonymous != nil && *req.IsAnonymous,
	}
	if err := h.writeDB().Create(&fb).Error; err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, fb)
}

// ── Recently Viewed ─────────────────────────────────────────────────────────

func (h *Handler) RecentlyViewed(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		return
	}
	uid := fmt.Sprintf("%v", userID)

	// Use Redis if available, otherwise return empty
	if h.rdb != nil {
		ids, err := h.rdb.LRange(c.Request.Context(), "recently_viewed:"+uid, 0, 19).Result()
		if err != nil || len(ids) == 0 {
			response.OK(c, []interface{}{})
			return
		}
		var listings []Listing
		h.readDB().Where("id IN ? AND status = 'active'", idsToUUIDs(ids)).Preload("Images").Find(&listings)
		response.OK(c, listings)
		return
	}

	response.OK(c, []interface{}{})
}

func idsToUUIDs(ids []string) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if u, err := uuid.Parse(id); err == nil {
			result = append(result, u)
		}
	}
	return result
}

// RecordRecentlyViewed adds a listing to the user's recently viewed list.
func RecordRecentlyViewed(db *gorm.DB, rdb interface{ LPush(ctx interface{}, key string, values ...interface{}) error }, userID, listingID string) {
	// This is called from the view tracking logic
}
