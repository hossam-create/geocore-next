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
        if page < 1 {
                page = 1
        }
        if perPage < 1 {
                perPage = 1
        }
        if perPage > 50 {
                perPage = 50
        }
        offset := (page - 1) * perPage

        q := h.db.Model(&Listing{}).Preload("Images").Preload("Category").
                Where("status = ?", "active")

        // Filters
        if cat := c.Query("category"); cat != "" {
                // Accept either a UUID or a slug
                if _, err := uuid.Parse(cat); err == nil {
                        q = q.Where("category_id = ?", cat)
                } else {
                        // Resolve slug → UUID via subquery
                        q = q.Where("category_id = (SELECT id FROM categories WHERE slug = ? LIMIT 1)", cat)
                }
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
        if sellerID := c.Query("seller_id"); sellerID != "" {
                if sellerID == "me" {
                        // Direct the client to the authenticated endpoint
                        response.BadRequest(c, "Use /listings/me to fetch your own listings")
                        return
                }
                if _, err := uuid.Parse(sellerID); err != nil {
                        response.BadRequest(c, "Invalid seller_id: must be a valid UUID")
                        return
                }
                q = q.Where("user_id = ?", sellerID)
        }

        // Geo filter: if lat, lng and radius are provided, filter by bounding box
        if latStr := c.Query("lat"); latStr != "" {
                if lngStr := c.Query("lng"); lngStr != "" {
                        radiusStr := c.DefaultQuery("radius", "50")
                        lat, latErr := strconv.ParseFloat(latStr, 64)
                        lng, lngErr := strconv.ParseFloat(lngStr, 64)
                        radius, radErr := strconv.ParseFloat(radiusStr, 64)
                        if latErr == nil && lngErr == nil && radErr == nil && radius > 0 {
                                // Bounding box approximation: 1 degree lat ≈ 111 km
                                latDelta := radius / 111.0
                                lngDelta := radius / (111.0 * math.Cos(lat*math.Pi/180.0))
                                q = q.Where("latitude BETWEEN ? AND ? AND longitude BETWEEN ? AND ?",
                                        lat-latDelta, lat+latDelta, lng-lngDelta, lng+lngDelta)
                        }
                }
        }

        // Sort
        switch c.DefaultQuery("sort", "newest") {
        case "price_asc":
                q = q.Order("price ASC")
        case "price_desc":
                q = q.Order("price DESC")
        case "popular":
                q = q.Order("view_count DESC")
        case "oldest":
                q = q.Order("created_at ASC")
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
        if err := h.db.Preload("Images").Preload("Category").Preload("Seller").
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
                CategoryID   string   `json:"category_id"`
                CategorySlug string   `json:"category"`
                Title        string   `json:"title" binding:"required,min=5,max=200"`
                Description  string   `json:"description" binding:"required,min=10"`
                Price        *float64 `json:"price"`
                Currency     string   `json:"currency"`
                PriceType    string   `json:"price_type"`
                Condition    string   `json:"condition"`
                Type         string   `json:"type"`
                Country      string   `json:"country" binding:"required"`
                City         string   `json:"city" binding:"required"`
                Address      string   `json:"address"`
                Latitude     *float64 `json:"latitude"`
                Longitude    *float64 `json:"longitude"`
                ImageURLs    []string `json:"image_urls"`
                Images       []string `json:"images"`
        }
        if err := c.ShouldBindJSON(&req); err != nil {
                response.BadRequest(c, err.Error())
                return
        }

        // Resolve category: accept either UUID (category_id) or slug (category)
        var catID uuid.UUID
        if req.CategoryID != "" {
                var err error
                catID, err = uuid.Parse(req.CategoryID)
                if err != nil {
                        response.BadRequest(c, "Invalid category_id: must be a valid UUID")
                        return
                }
        } else if req.CategorySlug != "" {
                var cat Category
                if err := h.db.Where("slug = ?", req.CategorySlug).First(&cat).Error; err != nil {
                        response.BadRequest(c, "Invalid category slug: "+req.CategorySlug)
                        return
                }
                catID = cat.ID
        } else {
                response.BadRequest(c, "category_id or category (slug) is required")
                return
        }

        // Validate condition enum
        validConditions := map[string]bool{"new": true, "like-new": true, "good": true, "fair": true, "for-parts": true}
        if req.Condition != "" && !validConditions[req.Condition] {
                response.BadRequest(c, "Invalid condition: must be one of new, like-new, good, fair, for-parts")
                return
        }

        // Validate type enum
        validTypes := map[string]bool{"sell": true, "buy": true, "rent": true, "auction": true, "service": true}
        if req.Type != "" && !validTypes[req.Type] {
                response.BadRequest(c, "Invalid type: must be one of sell, buy, rent, auction, service")
                return
        }

        // Merge image URLs from both fields
        imageURLs := req.ImageURLs
        if len(imageURLs) == 0 {
                imageURLs = req.Images
        }

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
        for i, url := range imageURLs {
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
        h.db.Preload("Images").Preload("Category").Where("user_id = ?", userID).
                Order("created_at DESC").Find(&listings)
        response.OK(c, listings)
}

func defaultStr(s, def string) string {
        if s == "" {
                return def
        }
        return s
}
