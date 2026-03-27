package stores

import (
        "context"
        "encoding/json"
        "fmt"
        "regexp"
        "strconv"
        "strings"
        "time"

        "github.com/geocore-next/backend/internal/listings"
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

const storeListCacheKeyPrefix = "stores:list:"
const storeListCacheTTL = 5 * time.Minute

// storeCacheKey returns a cache key scoped to the page and per_page values.
func storeCacheKey(page, perPage int) string {
        return fmt.Sprintf("%s%d:%d", storeListCacheKeyPrefix, page, perPage)
}

// List — GET /api/v1/stores
// Returns active storefronts with configurable pagination (default 20, max 100).
// Results are cached in Redis for 5 minutes per page/per_page combination.
func (h *Handler) List(c *gin.Context) {
        page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
        perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
        if page < 1 {
                page = 1
        }
        if perPage < 1 || perPage > 100 {
                perPage = 20
        }
        offset := (page - 1) * perPage
        cacheKey := storeCacheKey(page, perPage)

        type pagedResult struct {
                Stores []Storefront `json:"stores"`
                Total  int64        `json:"total"`
        }

        // Try Redis cache
        if h.rdb != nil {
                if cached, err := h.rdb.Get(context.Background(), cacheKey).Bytes(); err == nil {
                        var res pagedResult
                        if json.Unmarshal(cached, &res) == nil {
                                pages := (res.Total + int64(perPage) - 1) / int64(perPage)
                                response.OKMeta(c, res.Stores, gin.H{"total": res.Total, "page": page, "per_page": perPage, "pages": pages})
                                return
                        }
                }
        }

        var stores []Storefront
        var total int64
        h.db.Model(&Storefront{}).Where("is_active = true").Count(&total)
        h.db.Where("is_active = true").
                Order("views DESC").
                Limit(perPage).
                Offset(offset).
                Find(&stores)

        // Cache result
        if h.rdb != nil {
                if data, err := json.Marshal(pagedResult{Stores: stores, Total: total}); err == nil {
                        h.rdb.Set(context.Background(), cacheKey, data, storeListCacheTTL)
                }
        }

        pages := int64(1)
        if perPage > 0 {
                pages = (total + int64(perPage) - 1) / int64(perPage)
        }
        response.OKMeta(c, stores, gin.H{"total": total, "page": page, "per_page": perPage, "pages": pages})
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

        // Invalidate all paginated store list cache keys
        h.invalidateStoreListCache()

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

        // Invalidate all paginated store list cache keys
        h.invalidateStoreListCache()

        response.OK(c, store)
}

// ── helpers ──────────────────────────────────────────────────────────────────

// invalidateStoreListCache removes all paginated store list cache keys from Redis.
// It uses Scan+Del so it works regardless of how many page/per_page combinations are cached.
func (h *Handler) invalidateStoreListCache() {
        if h.rdb == nil {
                return
        }
        ctx := context.Background()
        pattern := storeListCacheKeyPrefix + "*"
        var cursor uint64
        for {
                keys, next, err := h.rdb.Scan(ctx, cursor, pattern, 100).Result()
                if err != nil {
                        break
                }
                if len(keys) > 0 {
                        h.rdb.Del(ctx, keys...)
                }
                cursor = next
                if cursor == 0 {
                        break
                }
        }
}

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
