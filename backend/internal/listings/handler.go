package listings

import (
	"math"
	"strconv"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Handler struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
	return &Handler{db, rdb}
}

func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if perPage > 50 {
		perPage = 50
	}
	offset := (page - 1) * perPage

	q := h.db.Model(&Listing{}).Preload("Images").Preload("Category").
		Where("status = ?", "active")

	// Filters
	if cat := c.Query("category"); cat != "" {
		q = q.Where("category_id = ?", cat)
	}
	if country := c.Query("country"); country != "" {
		q = q.Where("country = ?", country)
	}
	if city := c.Query("city"); city != "" {
		q = q.Where("city ILIKE ?", "%"+city+"%")
	}
	if t := c.Query("type"); t != "" {
		q = q.Where("type = ?", t)
	}
	if condition := c.Query("condition"); condition != "" {
		q = q.Where("condition = ?", condition)
	}
	if minPrice := c.Query("min_price"); minPrice != "" {
		q = q.Where("price >= ?", minPrice)
	}
	if maxPrice := c.Query("max_price"); maxPrice != "" {
		q = q.Where("price <= ?", maxPrice)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("title ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Sort
	switch c.DefaultQuery("sort", "newest") {
	case "price_asc":
		q = q.Order("price ASC")
	case "price_desc":
		q = q.Order("price DESC")
	case "popular":
		q = q.Order("view_count DESC")
	default:
		q = q.Order("is_featured DESC, created_at DESC")
	}

	var total int64
	q.Count(&total)

	var listings []Listing
	q.Offset(offset).Limit(perPage).Find(&listings)

	response.OKMeta(c, listings, response.Meta{
		Total:   total,
		Page:    page,
		PerPage: perPage,
		Pages:   int64(math.Ceil(float64(total) / float64(perPage))),
	})
}

func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}
	var listing Listing
	if err := h.db.Preload("Images").Preload("Category").
		First(&listing, "id = ? AND status = ?", id, "active").Error; err != nil {
		response.NotFound(c, "Listing")
		return
	}
	// Increment view count async
	go h.db.Model(&listing).UpdateColumn("view_count", gorm.Expr("view_count + 1"))
	response.OK(c, listing)
}

func (h *Handler) Create(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	var req struct {
		CategoryID  string   `json:"category_id" binding:"required"`
		Title       string   `json:"title" binding:"required,min=5,max=200"`
		Description string   `json:"description" binding:"required,min=10"`
		Price       *float64 `json:"price"`
		Currency    string   `json:"currency"`
		PriceType   string   `json:"price_type"`
		Condition   string   `json:"condition"`
		Type        string   `json:"type"`
		Country     string   `json:"country" binding:"required"`
		City        string   `json:"city" binding:"required"`
		Address     string   `json:"address"`
		Latitude    *float64 `json:"latitude"`
		Longitude   *float64 `json:"longitude"`
		ImageURLs   []string `json:"image_urls"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	catID, _ := uuid.Parse(req.CategoryID)
	expires := time.Now().AddDate(0, 2, 0) // 2 months
	listing := Listing{
		ID:          uuid.New(),
		UserID:      userID,
		CategoryID:  catID,
		Title:       req.Title,
		Description: req.Description,
		Price:       req.Price,
		Currency:    defaultStr(req.Currency, "USD"),
		PriceType:   defaultStr(req.PriceType, "fixed"),
		Condition:   req.Condition,
		Type:        defaultStr(req.Type, "sell"),
		Country:     req.Country,
		City:        req.City,
		Address:     req.Address,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		Status:      "active",
		ExpiresAt:   &expires,
	}

	if err := h.db.Create(&listing).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Save images
	for i, url := range req.ImageURLs {
		h.db.Create(&ListingImage{
			ID:        uuid.New(),
			ListingID: listing.ID,
			URL:       url,
			SortOrder: i,
			IsCover:   i == 0,
		})
	}

	response.Created(c, listing)
}

func (h *Handler) Update(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}
	var listing Listing
	if err := h.db.First(&listing, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		response.NotFound(c, "Listing")
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	// Allow only safe fields to update
	allowed := []string{"title", "description", "price", "currency", "price_type", "condition", "country", "city", "address", "status"}
	updates := map[string]interface{}{}
	for _, k := range allowed {
		if v, ok := req[k]; ok {
			updates[k] = v
		}
	}
	h.db.Model(&listing).Updates(updates)
	response.OK(c, listing)
}

func (h *Handler) Delete(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}
	result := h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&Listing{})
	if result.RowsAffected == 0 {
		response.NotFound(c, "Listing")
		return
	}
	response.OK(c, gin.H{"message": "Listing deleted"})
}

func (h *Handler) GetCategories(c *gin.Context) {
	var cats []Category
	h.db.Where("parent_id IS NULL AND is_active = true").
		Preload("Children").
		Order("sort_order").
		Find(&cats)
	response.OK(c, cats)
}

func (h *Handler) ToggleFavorite(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	listingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}
	var fav Favorite
	result := h.db.Where("user_id = ? AND listing_id = ?", userID, listingID).First(&fav)
	if result.Error == nil {
		h.db.Delete(&fav)
		h.db.Model(&Listing{}).Where("id = ?", listingID).
			UpdateColumn("favorite_count", gorm.Expr("favorite_count - 1"))
		response.OK(c, gin.H{"favorited": false})
	} else {
		h.db.Create(&Favorite{ID: uuid.New(), UserID: userID, ListingID: listingID})
		h.db.Model(&Listing{}).Where("id = ?", listingID).
			UpdateColumn("favorite_count", gorm.Expr("favorite_count + 1"))
		response.OK(c, gin.H{"favorited": true})
	}
}

func (h *Handler) GetMyListings(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	var listings []Listing
	h.db.Preload("Images").Where("user_id = ?", userID).
		Order("created_at DESC").Find(&listings)
	response.OK(c, listings)
}

func defaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
