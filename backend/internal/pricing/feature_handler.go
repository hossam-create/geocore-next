package pricing

import (
	"net/http"

	"github.com/geocore-next/backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ── Feature Pipeline Handler ──────────────────────────────────────────────────────

type FeatureHandler struct {
	db      *gorm.DB
	rdb     *redis.Client
	pipeline *PipelineService
}

func NewFeatureHandler(db *gorm.DB, rdb *redis.Client) *FeatureHandler {
	return &FeatureHandler{
		db:       db,
		rdb:      rdb,
		pipeline: NewPipelineService(db, rdb),
	}
}

// ── Enrich (POST /features/enrich) ────────────────────────────────────────────────

type EnrichReq struct {
	UserID    string `json:"user_id" binding:"required"`
	OrderID   string `json:"order_id"`
	ItemID    string `json:"item_id"`
	SessionID string `json:"session_id"`
	Geo       string `json:"geo"`
}

func (h *FeatureHandler) Enrich(c *gin.Context) {
	if !config.GetFlags().EnableDynamicPricing {
		c.JSON(http.StatusForbidden, gin.H{"error": "Dynamic pricing is not available"})
		return
	}

	var req EnrichReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(req.UserID)
	orderID, _ := uuid.Parse(req.OrderID)
	itemID, _ := uuid.Parse(req.ItemID)

	resp, err := h.pipeline.Enrich(PipelineRequest{
		UserID:    userID,
		OrderID:   orderID,
		ItemID:    itemID,
		SessionID: req.SessionID,
		Geo:       req.Geo,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ── Record Event (POST /features/event) ──────────────────────────────────────────

type EventReq struct {
	UserID       string  `json:"user_id" binding:"required"`
	ItemID       string  `json:"item_id"`
	EventType    string  `json:"event_type" binding:"required"` // view, click, purchase, cancel, claim
	TrustWeight  float64 `json:"trust_weight"` // 0-1, low-trust = less impact
}

func (h *FeatureHandler) RecordEvent(c *gin.Context) {
	var req EventReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(req.UserID)
	itemID, _ := uuid.Parse(req.ItemID)
	trustWeight := req.TrustWeight
	if trustWeight <= 0 {
		trustWeight = 1.0
	}

	if err := h.pipeline.ProcessEvent(userID, itemID, req.EventType, trustWeight); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Event processed — embeddings + features updated"})
}

// ── Retrieve (POST /features/retrieve) ────────────────────────────────────────────

type RetrieveReq struct {
	UserID       string   `json:"user_id" binding:"required"`
	ItemID       string   `json:"item_id"`
	CategoryPath string   `json:"category_path"`
	PriceMin     int64    `json:"price_min"`
	PriceMax     int64    `json:"price_max"`
	TopK         int      `json:"top_k"`
	ExcludeIDs   []string `json:"exclude_ids"`
}

func (h *FeatureHandler) Retrieve(c *gin.Context) {
	var req RetrieveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(req.UserID)
	itemID, _ := uuid.Parse(req.ItemID)

	var excludeIDs []uuid.UUID
	for _, id := range req.ExcludeIDs {
		if parsed, err := uuid.Parse(id); err == nil {
			excludeIDs = append(excludeIDs, parsed)
		}
	}

	retSvc := NewRetrievalService(h.db, NewEmbeddingService(h.db, h.rdb), NewFeatureStore(h.db, h.rdb))
	resp, err := retSvc.Retrieve(RetrievalRequest{
		UserID:       userID,
		ItemID:       itemID,
		CategoryPath: req.CategoryPath,
		PriceMin:     req.PriceMin,
		PriceMax:     req.PriceMax,
		TopK:         req.TopK,
		ExcludeIDs:   excludeIDs,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ── Get Similar Items (GET /features/similar/:id) ──────────────────────────────────

func (h *FeatureHandler) GetSimilarItems(c *gin.Context) {
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	embSvc := NewEmbeddingService(h.db, h.rdb)
	ids, err := embSvc.GetSimilarItems(itemID, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"similar_items": ids})
}

// ── Admin: Pipeline Dashboard ──────────────────────────────────────────────────────

func (h *FeatureHandler) GetDashboard(c *gin.Context) {
	dashboard := GetPipelineDashboard(h.db)
	c.JSON(http.StatusOK, dashboard)
}

// ── Admin: Refresh Features ────────────────────────────────────────────────────────

type RefreshReq struct {
	UserID string `json:"user_id"`
	ItemID string `json:"item_id"`
}

func (h *FeatureHandler) RefreshFeatures(c *gin.Context) {
	var req RefreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fs := NewFeatureStore(h.db, h.rdb)

	if req.UserID != "" {
		userID, _ := uuid.Parse(req.UserID)
		if err := fs.RefreshUserFeatures(userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if req.ItemID != "" {
		itemID, _ := uuid.Parse(req.ItemID)
		if err := fs.RefreshItemFeatures(itemID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Features refreshed"})
}

// ── Admin: Get User Features ────────────────────────────────────────────────────────

func (h *FeatureHandler) GetUserFeatures(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	fs := NewFeatureStore(h.db, h.rdb)
	features, err := fs.GetUserFeatures(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User features not found"})
		return
	}

	c.JSON(http.StatusOK, features)
}

// ── Admin: Get Item Features ────────────────────────────────────────────────────────

func (h *FeatureHandler) GetItemFeatures(c *gin.Context) {
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	fs := NewFeatureStore(h.db, h.rdb)
	features, err := fs.GetItemFeatures(itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item features not found"})
		return
	}

	c.JSON(http.StatusOK, features)
}
