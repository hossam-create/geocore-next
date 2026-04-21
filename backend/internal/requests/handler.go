package requests

import (
	"net/http"
	"strconv"
	"time"

	"github.com/geocore-next/backend/internal/moderation"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Handler provides HTTP handlers for product request operations
type Handler struct {
	db *gorm.DB
}

// NewHandler creates a new requests handler
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// CreateRequest — POST /api/v1/requests
// Authenticated buyers submit a new product request.
type CreateRequestReq struct {
	Title       string   `json:"title"       binding:"required,min=3,max=200"`
	Description string   `json:"description"`
	CategoryID  *string  `json:"category_id"`
	Budget      *float64 `json:"budget"`
	Currency    string   `json:"currency"`
}

func (h *Handler) CreateRequest(c *gin.Context) {
	var req CreateRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if blocked, reason := moderation.CheckContent(req.Title, req.Description); blocked {
		response.BadRequest(c, reason)
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))
	currency := req.Currency
	if currency == "" {
		currency = "AED"
	}

	pr := ProductRequest{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Budget:      req.Budget,
		Currency:    currency,
		Status:      StatusOpen,
	}

	if req.CategoryID != nil {
		catID, err := uuid.Parse(*req.CategoryID)
		if err == nil {
			pr.CategoryID = &catID
		}
	}

	expires := time.Now().Add(30 * 24 * time.Hour)
	pr.ExpiresAt = &expires

	if err := h.db.Create(&pr).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	response.Created(c, pr)
}

func (h *Handler) UpdateRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid request id")
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))
	var pr ProductRequest
	if err := h.db.First(&pr, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		response.NotFound(c, "request")
		return
	}
	if pr.Status != StatusOpen {
		response.BadRequest(c, "only open requests can be updated")
		return
	}

	var req struct {
		Title       *string  `json:"title"`
		Description *string  `json:"description"`
		Budget      *float64 `json:"budget"`
		Currency    *string  `json:"currency"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	finalTitle := pr.Title
	if req.Title != nil {
		finalTitle = *req.Title
	}
	finalDescription := pr.Description
	if req.Description != nil {
		finalDescription = *req.Description
	}
	if blocked, reason := moderation.CheckContent(finalTitle, finalDescription); blocked {
		response.BadRequest(c, reason)
		return
	}

	updates := map[string]any{}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Budget != nil {
		updates["budget"] = *req.Budget
	}
	if req.Currency != nil {
		updates["currency"] = *req.Currency
	}

	if len(updates) == 0 {
		response.OK(c, pr)
		return
	}
	if err := h.db.Model(&pr).Updates(updates).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, pr)
}

// ListRequests — GET /api/v1/requests
// Public endpoint — no auth required. Filterable by category and status.
func (h *Handler) ListRequests(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	q := h.db.Model(&ProductRequest{}).Where("status = ?", StatusOpen)

	if cat := c.Query("category_id"); cat != "" {
		q = q.Where("category_id = ?", cat)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("title ILIKE ?", "%"+search+"%")
	}

	var total int64
	q.Count(&total)

	var list []ProductRequest
	q.Order("created_at DESC").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Find(&list)

	// Enrich with response counts and user/category names
	for i := range list {
		var cnt int64
		h.db.Model(&ProductRequestResponse{}).Where("request_id = ?", list[i].ID).Count(&cnt)
		list[i].ResponseCount = int(cnt)

		var row struct{ Name string }
		h.db.Table("users").Select("name").Where("id = ?", list[i].UserID).First(&row)
		list[i].UserName = row.Name

		if list[i].CategoryID != nil {
			h.db.Table("categories").Select("name").Where("id = ?", list[i].CategoryID).First(&row)
			list[i].CategoryName = row.Name
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": list,
		"pagination": gin.H{
			"total":    total,
			"page":     page,
			"per_page": perPage,
			"pages":    (total + int64(perPage) - 1) / int64(perPage),
		},
	})
}

// GetRequest — GET /api/v1/requests/:id
func (h *Handler) GetRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid request id")
		return
	}

	var pr ProductRequest
	if err := h.db.First(&pr, "id = ?", id).Error; err != nil {
		response.NotFound(c, "request")
		return
	}

	// Load responses
	var responses []ProductRequestResponse
	h.db.Where("request_id = ?", id).Order("created_at ASC").Find(&responses)
	for i := range responses {
		var row struct{ Name string }
		h.db.Table("users").Select("name").Where("id = ?", responses[i].SellerID).First(&row)
		responses[i].SellerName = row.Name
	}

	// Enrich user/category
	var row struct{ Name string }
	h.db.Table("users").Select("name").Where("id = ?", pr.UserID).First(&row)
	pr.UserName = row.Name
	if pr.CategoryID != nil {
		h.db.Table("categories").Select("name").Where("id = ?", pr.CategoryID).First(&row)
		pr.CategoryName = row.Name
	}

	c.JSON(http.StatusOK, gin.H{"data": pr, "responses": responses})
}

// RespondToRequest — POST /api/v1/requests/:id/respond
// Sellers respond to a product request with an optional listing link and message.
type RespondReq struct {
	ListingID *string `json:"listing_id"`
	Message   string  `json:"message" binding:"required,min=5"`
}

func (h *Handler) RespondToRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid request id")
		return
	}

	var req RespondReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var pr ProductRequest
	if err := h.db.First(&pr, "id = ?", id).Error; err != nil {
		response.NotFound(c, "request")
		return
	}
	if pr.Status != StatusOpen {
		response.BadRequest(c, "this request is no longer open")
		return
	}

	sellerID, _ := uuid.Parse(c.GetString("user_id"))
	resp := ProductRequestResponse{
		RequestID: id,
		SellerID:  sellerID,
		Message:   req.Message,
	}
	if req.ListingID != nil {
		lid, err := uuid.Parse(*req.ListingID)
		if err == nil {
			resp.ListingID = &lid
		}
	}

	if err := h.db.Create(&resp).Error; err != nil {
		response.BadRequest(c, "you have already responded to this request")
		return
	}

	response.Created(c, resp)
}

// CancelRequest — DELETE /api/v1/requests/:id
// Only the creator can cancel their own request.
func (h *Handler) CancelRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid request id")
		return
	}

	userID := c.GetString("user_id")
	var pr ProductRequest
	if err := h.db.First(&pr, "id = ?", id).Error; err != nil {
		response.NotFound(c, "request")
		return
	}
	if pr.UserID.String() != userID {
		response.Forbidden(c)
		return
	}

	h.db.Model(&pr).Update("status", StatusCancelled)
	response.OK(c, gin.H{"message": "Request cancelled"})
}

// MyRequests — GET /api/v1/requests/mine
// Returns the authenticated user's product requests.
func (h *Handler) MyRequests(c *gin.Context) {
	userID := c.GetString("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	var total int64
	h.db.Model(&ProductRequest{}).Where("user_id = ?", userID).Count(&total)

	var list []ProductRequest
	h.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Find(&list)

	for i := range list {
		var cnt int64
		h.db.Model(&ProductRequestResponse{}).Where("request_id = ?", list[i].ID).Count(&cnt)
		list[i].ResponseCount = int(cnt)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": list,
		"pagination": gin.H{
			"total":    total,
			"page":     page,
			"per_page": perPage,
			"pages":    (total + int64(perPage) - 1) / int64(perPage),
		},
	})
}
