package listings

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/freeze"
	"github.com/geocore-next/backend/internal/images"
	"github.com/geocore-next/backend/internal/moderation"
	"github.com/geocore-next/backend/internal/security"
	"github.com/geocore-next/backend/internal/subscriptions"
	"github.com/geocore-next/backend/pkg/cache"
	"github.com/geocore-next/backend/pkg/events"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/geocore-next/backend/pkg/rwtracker"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Handler struct {
	db      *gorm.DB
	dbWrite *gorm.DB
	dbRead  *gorm.DB
	rdb     *redis.Client
	cache   *cache.Cache
	rwt     *rwtracker.RecentWriteTracker
}

func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
	return &Handler{db: db, dbWrite: db, dbRead: db, rdb: rdb, cache: cache.New(rdb), rwt: rwtracker.NewRecentWriteTracker(rdb)}
}

func NewHandlerReadWrite(dbWrite *gorm.DB, dbRead *gorm.DB, rdb *redis.Client) *Handler {
	if dbRead == nil {
		dbRead = dbWrite
	}
	return &Handler{db: dbWrite, dbWrite: dbWrite, dbRead: dbRead, rdb: rdb, cache: cache.New(rdb), rwt: rwtracker.NewRecentWriteTracker(rdb)}
}

// readDB returns the read replica unless the current user recently performed a
// write (read-after-write consistency). Falls back to write DB when no replica.
func (h *Handler) readDB() *gorm.DB {
	if h.rwt != nil {
		// Try to extract user_id from gin context if available
		// This is a no-op when called outside a request context
	}
	if h.dbRead != nil {
		return h.dbRead
	}
	return h.dbWrite
}

// readDBForUser routes reads to primary if the user wrote recently, else replica.
func (h *Handler) readDBForUser(userID string) *gorm.DB {
	if h.rwt != nil && h.rwt.ShouldReadPrimary(userID) {
		return h.dbWrite
	}
	if h.dbRead != nil {
		return h.dbRead
	}
	return h.dbWrite
}

func (h *Handler) writeDB() *gorm.DB {
	if h.dbWrite != nil {
		return h.dbWrite
	}
	return h.dbRead
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

	// Use primary DB if user recently wrote, else replica
	readDB := h.readDB()
	if uid, exists := c.Get("user_id"); exists {
		if uidStr, ok := uid.(string); ok {
			readDB = h.readDBForUser(uidStr)
		}
	}
	q := readDB.Model(&Listing{}).Preload("Images").Preload("Category").
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
		safeCity := security.SanitizeSearchQuery(city)
		if safeCity != "" {
			q = q.Where("city ILIKE ?", "%"+safeCity+"%")
		}
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
		safeSearch := security.SanitizeSearchQuery(search)
		if safeSearch != "" {
			q = q.Where("title ILIKE ? OR description ILIKE ?", "%"+safeSearch+"%", "%"+safeSearch+"%")
		}
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

	// Custom fields filter: ?cf[transmission]=automatic, ?cf[year_min]=2020, ?cf[year_max]=2023
	for key, values := range c.Request.URL.Query() {
		if !strings.HasPrefix(key, "cf[") || !strings.HasSuffix(key, "]") {
			continue
		}
		fieldName := key[3 : len(key)-1]
		val := values[0]
		if val == "" {
			continue
		}
		if strings.HasSuffix(fieldName, "_min") {
			realField := fieldName[:len(fieldName)-4]
			q = q.Where("(custom_fields->>?)::numeric >= ?", realField, val)
		} else if strings.HasSuffix(fieldName, "_max") {
			realField := fieldName[:len(fieldName)-4]
			q = q.Where("(custom_fields->>?)::numeric <= ?", realField, val)
		} else {
			q = q.Where("custom_fields->>? = ?", fieldName, val)
		}
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

	// ── Cache: only cache page 1 with no filters (hot path) ────────────────
	cacheKey := ""
	if c.Request.URL.RawQuery == "" || c.Request.URL.RawQuery == fmt.Sprintf("page=1&per_page=%d", perPage) {
		cacheKey = fmt.Sprintf("listings:list:p%d:pp%d", page, perPage)
	}
	type listResult struct {
		Listings []Listing     `json:"listings"`
		Meta     response.Meta `json:"meta"`
	}
	if cacheKey != "" {
		var cached listResult
		if h.cache.Get(c.Request.Context(), cacheKey, &cached) {
			response.OKMeta(c, cached.Listings, cached.Meta)
			return
		}
	}

	var total int64
	q.Count(&total)

	var listings []Listing
	q.Offset(offset).Limit(perPage).Find(&listings)

	meta := response.Meta{
		Total:   total,
		Page:    page,
		PerPage: perPage,
		Pages:   int64(math.Ceil(float64(total) / float64(perPage))),
	}
	if cacheKey != "" {
		h.cache.Set(c.Request.Context(), cacheKey, listResult{listings, meta}, 2*time.Minute)
	}
	response.OKMeta(c, listings, meta)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	// ── Cache: serve cached listing for public (unauthenticated) reads ───────
	cacheKey := fmt.Sprintf("listings:detail:%s", id.String())
	if c.GetHeader("Authorization") == "" {
		var cached Listing
		if h.cache.Get(c.Request.Context(), cacheKey, &cached) {
			response.OK(c, cached)
			return
		}
	}

	var listing Listing
	if err := h.readDB().Preload("Images").Preload("Category").Preload("Seller").
		First(&listing, "id = ? AND status = ?", id, "active").Error; err != nil {
		response.NotFound(c, "Listing")
		return
	}
	// Cache the plain listing for public reads (5-min TTL)
	if c.GetHeader("Authorization") == "" {
		h.cache.Set(c.Request.Context(), cacheKey, listing, 5*time.Minute)
	}
	// Increment view count async
	go h.writeDB().Model(&listing).UpdateColumn("view_count", gorm.Expr("view_count + 1"))

	// Include is_watched only when caller is authenticated.
	isWatched := false
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if userID, tokenErr := middleware.ValidateToken(token); tokenErr == nil {
			var cnt int64
			if qErr := h.db.Table("watchlist_items").
				Where("user_id = ? AND listing_id = ?", userID, listing.ID).
				Count(&cnt).Error; qErr == nil {
				isWatched = cnt > 0
			}

			listing.IsWatched = &isWatched
			response.OK(c, listing)
			return
		}
	}

	response.OK(c, listing)
}

func (h *Handler) Create(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))

	// Sprint 8.5: Block frozen users from creating listings
	if freeze.IsUserFrozen(h.db, userID) {
		response.Forbidden(c)
		return
	}

	// Enforce plan listing limit
	limit := subscriptions.GetUserPlanLimits(h.db, userID)
	if limit > 0 {
		var activeCount int64
		h.db.Model(&Listing{}).Where("user_id = ? AND status = ?", userID, "active").Count(&activeCount)
		if int(activeCount) >= limit {
			response.BadRequest(c, fmt.Sprintf("Listing limit reached (%d/%d). Upgrade your plan to post more listings.", activeCount, limit))
			return
		}
	}

	var req struct {
		CategoryID    string            `json:"category_id"`
		CategorySlug  string            `json:"category"`
		Title         string            `json:"title" binding:"required,min=5,max=200"`
		Description   string            `json:"description" binding:"required,min=10"`
		Price         *float64          `json:"price"`
		Currency      string            `json:"currency"`
		PriceType     string            `json:"price_type"`
		Condition     string            `json:"condition"`
		Type          string            `json:"type"`
		Country       string            `json:"country" binding:"required"`
		City          string            `json:"city" binding:"required"`
		Address       string            `json:"address"`
		Latitude      *float64          `json:"latitude"`
		Longitude     *float64          `json:"longitude"`
		ImageURLs     []string          `json:"image_urls"`      // legacy: raw URL strings
		Images        []string          `json:"images"`          // legacy: alias for image_urls
		ImageGroupIDs []string          `json:"image_group_ids"` // new: group IDs from /images/upload
		CustomFields  map[string]string `json:"custom_fields"`
		Attributes    map[string]string `json:"attributes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Sanitize user-controlled text fields before validation and persistence.
	req.Title = security.SanitizeText(req.Title)
	req.Description = security.SanitizeHTML(req.Description)
	req.Country = security.SanitizeText(req.Country)
	req.City = security.SanitizeText(req.City)
	req.Address = security.SanitizeText(req.Address)
	req.CategorySlug = strings.ToLower(strings.TrimSpace(req.CategorySlug))

	if len(req.Title) < 5 || len(req.Title) > 200 {
		response.BadRequest(c, "title must be between 5 and 200 characters after sanitization")
		return
	}
	if len(req.Description) < 10 {
		response.BadRequest(c, "description must be at least 10 characters after sanitization")
		return
	}
	if req.Country == "" || req.City == "" {
		response.BadRequest(c, "country and city are required")
		return
	}

	if blocked, reason := moderation.CheckContent(req.Title, req.Description); blocked {
		response.BadRequest(c, reason)
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
		if err := h.readDB().Where("slug = ?", req.CategorySlug).First(&cat).Error; err != nil {
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

	// Merge image URLs from both legacy fields
	imageURLs := req.ImageURLs
	if len(imageURLs) == 0 {
		imageURLs = req.Images
	}

	// Parse new-style image group IDs
	var imageGroupIDs []uuid.UUID
	for _, gid := range req.ImageGroupIDs {
		parsed, e := uuid.Parse(gid)
		if e == nil {
			imageGroupIDs = append(imageGroupIDs, parsed)
		}
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

	// Content moderation check
	if blocked, reason := moderation.CheckContent(req.Title, req.Description); blocked {
		// Transactional outbox: moderation.blocked event
		_ = kafka.WriteOutbox(h.writeDB(), kafka.TopicModeration, kafka.New(
			"moderation.blocked",
			listing.ID.String(),
			"moderation",
			kafka.Actor{Type: "user", ID: userID.String()},
			map[string]interface{}{
				"entity_type": "listing",
				"entity_id":   listing.ID.String(),
				"user_id":     userID.String(),
				"reason":      reason,
			},
			kafka.EventMeta{Source: "api-service"},
		))
		response.BadRequest(c, reason)
		return
	}

	// Merge custom_fields from either field
	cf := req.CustomFields
	if len(cf) == 0 {
		cf = req.Attributes
	}
	if len(cf) > 0 {
		for k, v := range cf {
			delete(cf, k)
			cleanKey := security.SanitizeText(k)
			if cleanKey == "" {
				continue
			}
			cf[cleanKey] = security.SanitizeText(v)
		}
		cfJSON, _ := json.Marshal(cf)
		listing.CustomFields = string(cfJSON)
	}

	if err := h.writeDB().Create(&listing).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Save images — prefer new group_id flow, fall back to legacy URL flow
	if len(imageGroupIDs) > 0 {
		// New flow: look up image variants from the images table
		for i, gid := range imageGroupIDs {
			var img images.Image
			// Prefer large variant for listing display
			if err := h.writeDB().Where("group_id = ? AND size = ?", gid, images.SizeLarge).First(&img).Error; err != nil {
				// Fallback to medium, then any
				if err := h.writeDB().Where("group_id = ? AND size = ?", gid, images.SizeMedium).First(&img).Error; err != nil {
					if err := h.writeDB().Where("group_id = ?", gid).Order("size ASC").First(&img).Error; err != nil {
						continue
					}
				}
			}
			h.writeDB().Create(&ListingImage{
				ID:        uuid.New(),
				ListingID: listing.ID,
				GroupID:   gid,
				ImageID:   img.ID,
				URL:       img.URL,
				Width:     img.Width,
				Height:    img.Height,
				Bytes:     img.Bytes,
				MimeType:  img.MimeType,
				Variant:   string(img.Size),
				SortOrder: i,
				IsCover:   i == 0,
			})
		}
	} else {
		// Legacy flow: raw URL strings (backward compatible)
		for i, url := range imageURLs {
			h.writeDB().Create(&ListingImage{
				ID:        uuid.New(),
				ListingID: listing.ID,
				URL:       url,
				Variant:   "original",
				MimeType:  "image/jpeg",
				SortOrder: i,
				IsCover:   i == 0,
			})
		}
	}

	// Invalidate search result caches on new listing
	if h.rdb != nil {
		ctx := context.Background()
		iter := h.rdb.Scan(ctx, 0, "listings:search:*", 100).Iterator()
		for iter.Next(ctx) {
			h.rdb.Del(ctx, iter.Val())
		}
	}
	// Mark recent write for read-after-write consistency
	if h.rwt != nil {
		h.rwt.MarkWrite(userID.String())
	}

	// Publish domain event for in-process consumers
	events.Publish(events.Event{
		Type: events.EventListingCreated,
		Payload: map[string]interface{}{
			"listing_id":  listing.ID.String(),
			"user_id":     userID.String(),
			"title":       listing.Title,
			"price":       listing.Price,
			"currency":    listing.Currency,
			"category_id": listing.CategoryID.String(),
		},
	})

	// Transactional outbox for Kafka delivery
	_ = kafka.WriteOutbox(h.writeDB(), kafka.TopicModeration, kafka.New(
		"listing.created",
		listing.ID.String(),
		"listing",
		kafka.Actor{Type: "user", ID: userID.String()},
		map[string]interface{}{
			"listing_id":  listing.ID.String(),
			"user_id":     userID.String(),
			"title":       listing.Title,
			"price":       listing.Price,
			"currency":    listing.Currency,
			"category_id": listing.CategoryID.String(),
		},
		kafka.EventMeta{Source: "api-service"},
	))

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
	if err := h.writeDB().First(&listing, "id = ? AND user_id = ?", id, userID).Error; err != nil {
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
			switch k {
			case "title", "country", "city", "address":
				if s, ok := v.(string); ok {
					updates[k] = security.SanitizeText(s)
				}
			case "description":
				if s, ok := v.(string); ok {
					updates[k] = security.SanitizeHTML(s)
				}
			default:
				updates[k] = v
			}
		}
	}
	if v, ok := updates["condition"]; ok {
		if cond, ok := v.(string); ok {
			validConditions := map[string]bool{"new": true, "like-new": true, "good": true, "fair": true, "for-parts": true}
			if cond != "" && !validConditions[cond] {
				response.BadRequest(c, "Invalid condition: must be one of new, like-new, good, fair, for-parts")
				return
			}
		}
	}
	finalTitle := listing.Title
	if v, ok := updates["title"].(string); ok {
		finalTitle = v
	}
	finalDescription := listing.Description
	if v, ok := updates["description"].(string); ok {
		finalDescription = v
	}
	if blocked, reason := moderation.CheckContent(finalTitle, finalDescription); blocked {
		response.BadRequest(c, reason)
		return
	}
	h.writeDB().Model(&listing).Updates(updates)
	// Invalidate cached listing detail and search caches
	ctx := c.Request.Context()
	h.cache.Del(ctx, fmt.Sprintf("listings:detail:%s", id.String()))
	if h.rdb != nil {
		iter := h.rdb.Scan(ctx, 0, "listings:search:*", 100).Iterator()
		for iter.Next(ctx) {
			h.rdb.Del(ctx, iter.Val())
		}
	}
	// Mark recent write for read-after-write consistency
	if h.rwt != nil {
		h.rwt.MarkWrite(userID)
	}
	response.OK(c, listing)
}

func (h *Handler) Delete(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}
	result := h.writeDB().Where("id = ? AND user_id = ?", id, userID).Delete(&Listing{})
	if result.RowsAffected == 0 {
		response.NotFound(c, "Listing")
		return
	}
	// Invalidate cached listing detail and search caches
	ctx := c.Request.Context()
	h.cache.Del(ctx, fmt.Sprintf("listings:detail:%s", id.String()))
	if h.rdb != nil {
		iter := h.rdb.Scan(ctx, 0, "listings:search:*", 100).Iterator()
		for iter.Next(ctx) {
			h.rdb.Del(ctx, iter.Val())
		}
	}
	// Mark recent write for read-after-write consistency
	if h.rwt != nil {
		h.rwt.MarkWrite(userID)
	}
	response.OK(c, gin.H{"message": "Listing deleted"})
}

func (h *Handler) GetCategories(c *gin.Context) {
	var cats []Category
	h.readDB().Where("parent_id IS NULL AND is_active = true").
		Preload("Children").
		Order("sort_order").
		Find(&cats)
	response.OK(c, cats)
}

// GetCategoryFields returns the custom fields for a given category.
func (h *Handler) GetCategoryFields(c *gin.Context) {
	categoryID := c.Param("id")
	var fields []CategoryField
	h.readDB().Where("category_id = ? AND is_active = true", categoryID).
		Order("sort_order").Find(&fields)
	response.OK(c, fields)
}

func (h *Handler) ToggleFavorite(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	listingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}
	var fav Favorite
	result := h.writeDB().Where("user_id = ? AND listing_id = ?", userID, listingID).First(&fav)
	if result.Error == nil {
		h.writeDB().Delete(&fav)
		h.writeDB().Model(&Listing{}).Where("id = ?", listingID).
			UpdateColumn("favorite_count", gorm.Expr("favorite_count - 1"))
		response.OK(c, gin.H{"favorited": false})
	} else {
		h.writeDB().Create(&Favorite{ID: uuid.New(), UserID: userID, ListingID: listingID})
		h.writeDB().Model(&Listing{}).Where("id = ?", listingID).
			UpdateColumn("favorite_count", gorm.Expr("favorite_count + 1"))
		response.OK(c, gin.H{"favorited": true})
	}
}

func (h *Handler) GetMyListings(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	var listings []Listing
	h.readDB().Preload("Images").Preload("Category").Where("user_id = ?", userID).
		Order("created_at DESC").Find(&listings)
	response.OK(c, listings)
}

func defaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
