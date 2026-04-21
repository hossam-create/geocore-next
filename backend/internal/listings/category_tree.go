package listings

// Sprint 18 — Category Tree / Breadcrumb / Slug discovery layer.
// Additive on top of existing Category model. No breaking changes.

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/config"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Types
// ════════════════════════════════════════════════════════════════════════════

// BreadcrumbNode is a single node in a category breadcrumb chain.
type BreadcrumbNode struct {
	ID     uuid.UUID `json:"id"`
	Slug   string    `json:"slug"`
	NameEn string    `json:"name_en"`
	NameAr string    `json:"name_ar,omitempty"`
	Level  int       `json:"level"`
}

// ════════════════════════════════════════════════════════════════════════════
// Feature flag helpers (shared across search/suggestions/tree).
// ════════════════════════════════════════════════════════════════════════════

func searchFlagEnabled() bool       { return config.GetFlags().EnableSearch }
func autocompleteFlagEnabled() bool { return config.GetFlags().EnableAutocomplete }

// ════════════════════════════════════════════════════════════════════════════
// Backfill — compute level + path for every row based on parent chain.
// Idempotent. Safe to call at startup after AutoMigrate.
// ════════════════════════════════════════════════════════════════════════════

// BackfillCategoryTree computes Level + Path for all categories.
// Rows whose Path/Level are already correct are skipped implicitly (single UPDATE batch).
func BackfillCategoryTree(db *gorm.DB) error {
	var cats []Category
	if err := db.Order("sort_order ASC, name_en ASC").Find(&cats).Error; err != nil {
		return err
	}
	byID := make(map[uuid.UUID]*Category, len(cats))
	for i := range cats {
		byID[cats[i].ID] = &cats[i]
	}

	// compute level + path via memoized recursion with cycle guard
	type meta struct {
		level int
		path  string
	}
	computed := make(map[uuid.UUID]meta, len(cats))

	var resolve func(c *Category, seen map[uuid.UUID]bool) meta
	resolve = func(c *Category, seen map[uuid.UUID]bool) meta {
		if m, ok := computed[c.ID]; ok {
			return m
		}
		if c.ParentID == nil {
			m := meta{level: 0, path: c.Slug}
			computed[c.ID] = m
			return m
		}
		if seen[c.ID] { // cycle — degrade gracefully
			m := meta{level: 0, path: c.Slug}
			computed[c.ID] = m
			return m
		}
		seen[c.ID] = true
		parent, ok := byID[*c.ParentID]
		if !ok { // dangling parent ref — treat as root
			m := meta{level: 0, path: c.Slug}
			computed[c.ID] = m
			return m
		}
		pm := resolve(parent, seen)
		m := meta{level: pm.level + 1, path: pm.path + "/" + c.Slug}
		computed[c.ID] = m
		return m
	}

	updated := 0
	for i := range cats {
		m := resolve(&cats[i], map[uuid.UUID]bool{})
		if cats[i].Level == m.level && cats[i].Path == m.path {
			continue
		}
		if err := db.Model(&Category{}).Where("id = ?", cats[i].ID).
			Updates(map[string]interface{}{"level": m.level, "path": m.path}).Error; err != nil {
			slog.Warn("category backfill update failed", "id", cats[i].ID, "err", err)
			continue
		}
		updated++
	}
	slog.Info("✅ category tree backfill complete", "total", len(cats), "updated", updated)
	return nil
}

// ════════════════════════════════════════════════════════════════════════════
// Tree builder — nest flat list by ParentID.
// ════════════════════════════════════════════════════════════════════════════

// BuildCategoryTree returns active categories as a nested tree rooted at parent_id IS NULL.
// maxDepth caps recursion (5 levels by default).
func BuildCategoryTree(db *gorm.DB, maxDepth int) ([]Category, error) {
	if maxDepth <= 0 {
		maxDepth = 5
	}
	var flat []Category
	if err := db.Where("is_active = true").
		Order("sort_order ASC, name_en ASC").
		Find(&flat).Error; err != nil {
		return nil, err
	}

	byParent := make(map[string][]Category, len(flat))
	for _, c := range flat {
		key := "root"
		if c.ParentID != nil {
			key = c.ParentID.String()
		}
		byParent[key] = append(byParent[key], c)
	}

	var attach func(nodes []Category, depth int) []Category
	attach = func(nodes []Category, depth int) []Category {
		for i := range nodes {
			if depth >= maxDepth {
				nodes[i].Children = nil
				continue
			}
			children := byParent[nodes[i].ID.String()]
			if len(children) > 0 {
				nodes[i].Children = attach(children, depth+1)
			} else {
				nodes[i].Children = nil
			}
		}
		return nodes
	}

	return attach(byParent["root"], 0), nil
}

// ════════════════════════════════════════════════════════════════════════════
// Handlers
// ════════════════════════════════════════════════════════════════════════════

// GetCategoryTree — GET /api/v1/categories/tree
// Returns a fully nested tree of active categories (cached in Redis 10min).
func (h *Handler) GetCategoryTree(c *gin.Context) {
	if !config.GetFlags().EnableCategoryTree {
		response.OK(c, gin.H{"tree": []Category{}, "disabled": true})
		return
	}

	const cacheKey = "categories:tree:v1"
	if h.rdb != nil {
		if cached, err := h.rdb.Get(context.Background(), cacheKey).Bytes(); err == nil {
			var tree []Category
			if json.Unmarshal(cached, &tree) == nil {
				response.OK(c, gin.H{"tree": tree, "cached": true})
				return
			}
		}
	}

	tree, err := BuildCategoryTree(h.readDB(), 5)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	if h.rdb != nil {
		if b, err := json.Marshal(tree); err == nil {
			h.rdb.Set(context.Background(), cacheKey, b, 10*time.Minute)
		}
	}
	response.OK(c, gin.H{"tree": tree})
}

// GetCategoryBySlug — GET /api/v1/categories/:slug
// Returns a single category with its direct children.
func (h *Handler) GetCategoryBySlug(c *gin.Context) {
	if !config.GetFlags().EnableCategoryTree {
		response.NotFound(c, "category")
		return
	}
	slug := strings.ToLower(strings.TrimSpace(c.Param("slug")))
	if slug == "" {
		response.BadRequest(c, "slug is required")
		return
	}

	var cat Category
	if err := h.readDB().Where("slug = ? AND is_active = true", slug).
		Preload("Children", "is_active = true").
		First(&cat).Error; err != nil {
		response.NotFound(c, "category")
		return
	}
	response.OK(c, gin.H{"category": cat})
}

// BreadcrumbPayload is the cached shape returned by GetCategoryBreadcrumb.
type BreadcrumbPayload struct {
	Slug       string           `json:"slug"`
	Breadcrumb []BreadcrumbNode `json:"breadcrumb"`
	Path       string           `json:"path"`
	Level      int              `json:"level"`
}

// GetCategoryBreadcrumb — GET /api/v1/categories/:slug/breadcrumb
// Returns the parent chain from root → this category (Redis-cached 10min).
func (h *Handler) GetCategoryBreadcrumb(c *gin.Context) {
	slug := strings.ToLower(strings.TrimSpace(c.Param("slug")))
	if slug == "" {
		response.BadRequest(c, "slug is required")
		return
	}

	// Sprint 18.1 — Redis precompute cache.
	cacheKey := "breadcrumb:v1:" + slug
	if h.rdb != nil {
		if cached, err := h.rdb.Get(context.Background(), cacheKey).Bytes(); err == nil {
			var p BreadcrumbPayload
			if json.Unmarshal(cached, &p) == nil {
				response.OK(c, p)
				return
			}
		}
	}

	var target Category
	if err := h.readDB().Where("slug = ? AND is_active = true", slug).First(&target).Error; err != nil {
		response.NotFound(c, "category")
		return
	}

	chain := []BreadcrumbNode{}
	current := &target
	// walk up via ParentID (bounded by Level+1 to prevent infinite loops)
	guard := target.Level + 2
	for current != nil && guard > 0 {
		chain = append([]BreadcrumbNode{{
			ID: current.ID, Slug: current.Slug,
			NameEn: current.NameEn, NameAr: current.NameAr, Level: current.Level,
		}}, chain...)
		if current.ParentID == nil {
			break
		}
		var parent Category
		if err := h.readDB().Where("id = ?", *current.ParentID).First(&parent).Error; err != nil {
			break
		}
		current = &parent
		guard--
	}

	payload := BreadcrumbPayload{
		Slug:       target.Slug,
		Breadcrumb: chain,
		Path:       target.Path,
		Level:      target.Level,
	}
	if h.rdb != nil {
		if b, err := json.Marshal(payload); err == nil {
			h.rdb.Set(context.Background(), cacheKey, b, 10*time.Minute)
		}
	}
	response.OK(c, payload)
}

// GetCategoryListings — GET /api/v1/categories/:slug/listings
// Thin wrapper over Search that forces category_slug filter + includes descendants.
// Reuses existing ranking (boost + rep + relevance).
func (h *Handler) GetCategoryListings(c *gin.Context) {
	slug := strings.ToLower(strings.TrimSpace(c.Param("slug")))
	if slug == "" {
		response.BadRequest(c, "slug is required")
		return
	}
	// Resolve the category by slug so we can match descendants by Path prefix.
	var cat Category
	if err := h.readDB().Where("slug = ? AND is_active = true", slug).First(&cat).Error; err != nil {
		response.NotFound(c, "category")
		return
	}

	// Inject into query params and delegate to Search.
	// Replace category_slug with category_path to include descendants.
	q := c.Request.URL.Query()
	q.Del("category")
	q.Del("category_id")
	q.Set("category_path", cat.Path)
	c.Request.URL.RawQuery = q.Encode()
	h.Search(c)
}

// ════════════════════════════════════════════════════════════════════════════
// Registration — invoked from routes.go
// ════════════════════════════════════════════════════════════════════════════
