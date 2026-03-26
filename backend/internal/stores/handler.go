package stores

import (
	"regexp"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/listings"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db}
}

// List — GET /api/v1/stores
// Returns all active storefronts, paginated.
func (h *Handler) List(c *gin.Context) {
	var stores []Storefront
	h.db.Where("is_active = true").
		Order("views DESC").
		Limit(50).
		Find(&stores)
	response.OK(c, stores)
}

// GetBySlug — GET /api/v1/stores/:slug
// Returns a storefront with its active listings.
func (h *Handler) GetBySlug(c *gin.Context) {
	slug := c.Param("slug")

	var store Storefront
	if err := h.db.Where("slug = ? AND is_active = true", slug).First(&store).Error; err != nil {
		response.NotFound(c, "Storefront")
		return
	}

	// Increment view count asynchronously
	go h.db.Model(&store).UpdateColumn("views", gorm.Expr("views + 1"))

	// Load seller's active listings
	var storeListings []listings.Listing
	h.db.Where("user_id = ? AND status = ?", store.UserID, "active").
		Preload("Images").
		Preload("Category").
		Order("created_at DESC").
		Limit(48).
		Find(&storeListings)

	// Refresh view count for response
	store.Views++

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"storefront": store,
			"listings":   storeListings,
		},
	})
}

// GetMyStore — GET /api/v1/stores/me (auth required)
func (h *Handler) GetMyStore(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	var store Storefront
	if err := h.db.Where("user_id = ?", userID).First(&store).Error; err != nil {
		response.NotFound(c, "Storefront")
		return
	}
	response.OK(c, store)
}

// Create — POST /api/v1/stores (auth required)
func (h *Handler) Create(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))

	// Check if user already has a storefront
	var existing Storefront
	if h.db.Where("user_id = ?", userID).First(&existing).Error == nil {
		response.Conflict(c, "You already have a storefront")
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required,min=2,max=120"`
		Description string `json:"description"`
		WelcomeMsg  string `json:"welcome_msg"`
		LogoURL     string `json:"logo_url"`
		BannerURL   string `json:"banner_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	slug := generateSlug(req.Name)
	// Ensure slug uniqueness
	var count int64
	h.db.Model(&Storefront{}).Where("slug = ?", slug).Count(&count)
	if count > 0 {
		slug = slug + "-" + time.Now().Format("0601")
	}

	store := Storefront{
		UserID:      userID,
		Slug:        slug,
		Name:        req.Name,
		Description: req.Description,
		WelcomeMsg:  req.WelcomeMsg,
		LogoURL:     req.LogoURL,
		BannerURL:   req.BannerURL,
	}

	if err := h.db.Create(&store).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.Created(c, store)
}

// Update — PUT /api/v1/stores/me (auth required)
func (h *Handler) Update(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))

	var store Storefront
	if err := h.db.Where("user_id = ?", userID).First(&store).Error; err != nil {
		response.NotFound(c, "Storefront")
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		WelcomeMsg  *string `json:"welcome_msg"`
		LogoURL     *string `json:"logo_url"`
		BannerURL   *string `json:"banner_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.WelcomeMsg != nil {
		updates["welcome_msg"] = *req.WelcomeMsg
	}
	if req.LogoURL != nil {
		updates["logo_url"] = *req.LogoURL
	}
	if req.BannerURL != nil {
		updates["banner_url"] = *req.BannerURL
	}

	h.db.Model(&store).Updates(updates)
	response.OK(c, store)
}

// ── helpers ──────────────────────────────────────────────────────────────────

var nonAlphanumRE = regexp.MustCompile(`[^a-z0-9]+`)

func generateSlug(name string) string {
	s := strings.ToLower(name)
	s = nonAlphanumRE.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}
