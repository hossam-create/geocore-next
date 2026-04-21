package addons

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// ListAddons returns all addons with optional filters.
// GET /admin/addons?category=&status=&q=&page=&per_page=
func (h *Handler) ListAddons(c *gin.Context) {
	page, perPage := 1, 20
	fmt.Sscan(c.DefaultQuery("page", "1"), &page)
	fmt.Sscan(c.DefaultQuery("per_page", "20"), &perPage)
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	q := h.db.Model(&Addon{})
	if cat := c.Query("category"); cat != "" {
		q = q.Where("category = ?", cat)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("name ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	q.Count(&total)

	var addons []Addon
	q.Offset((page - 1) * perPage).Limit(perPage).Order("download_count DESC, avg_rating DESC").Find(&addons)

	response.OKMeta(c, addons, response.Meta{
		Total: total, Page: page, PerPage: perPage,
		Pages: (total + int64(perPage) - 1) / int64(perPage),
	})
}

// GetAddon returns a single addon by ID.
// GET /admin/addons/:id
func (h *Handler) GetAddon(c *gin.Context) {
	id := c.Param("id")
	var addon Addon
	if err := h.db.Where("id = ?", id).First(&addon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "addon not found"})
		return
	}
	response.OK(c, addon)
}

// InstallAddon marks an addon as installed.
// POST /admin/addons/:id/install
func (h *Handler) InstallAddon(c *gin.Context) {
	id := c.Param("id")
	var addon Addon
	if err := h.db.Where("id = ?", id).First(&addon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "addon not found"})
		return
	}
	if addon.Status == AddonStatusEnabled || addon.Status == AddonStatusInstalled {
		c.JSON(http.StatusConflict, gin.H{"error": "addon already installed"})
		return
	}

	now := time.Now()
	h.db.Model(&addon).Updates(map[string]interface{}{
		"status":       AddonStatusInstalled,
		"installed_at": now,
		"download_count": gorm.Expr("download_count + 1"),
	})
	addon.Status = AddonStatusInstalled
	addon.InstalledAt = &now

	slog.Info("addon installed", "addon_id", addon.ID.String(), "slug", addon.Slug)
	response.OK(c, addon)
}

// UninstallAddon removes an addon installation.
// POST /admin/addons/:id/uninstall
func (h *Handler) UninstallAddon(c *gin.Context) {
	id := c.Param("id")
	var addon Addon
	if err := h.db.Where("id = ?", id).First(&addon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "addon not found"})
		return
	}

	h.db.Model(&addon).Updates(map[string]interface{}{
		"status":       AddonStatusAvailable,
		"installed_at": nil,
		"config":       nil,
	})
	addon.Status = AddonStatusAvailable
	addon.InstalledAt = nil

	slog.Info("addon uninstalled", "addon_id", addon.ID.String(), "slug", addon.Slug)
	response.OK(c, addon)
}

// EnableAddon enables an installed addon.
// POST /admin/addons/:id/enable
func (h *Handler) EnableAddon(c *gin.Context) {
	id := c.Param("id")
	var addon Addon
	if err := h.db.Where("id = ?", id).First(&addon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "addon not found"})
		return
	}
	if addon.Status != AddonStatusInstalled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "addon must be installed first"})
		return
	}

	h.db.Model(&addon).Update("status", AddonStatusEnabled)
	addon.Status = AddonStatusEnabled

	slog.Info("addon enabled", "addon_id", addon.ID.String(), "slug", addon.Slug)
	response.OK(c, addon)
}

// DisableAddon disables an enabled addon.
// POST /admin/addons/:id/disable
func (h *Handler) DisableAddon(c *gin.Context) {
	id := c.Param("id")
	var addon Addon
	if err := h.db.Where("id = ?", id).First(&addon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "addon not found"})
		return
	}
	if addon.Status != AddonStatusEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "addon is not enabled"})
		return
	}

	h.db.Model(&addon).Update("status", AddonStatusInstalled)
	addon.Status = AddonStatusInstalled

	slog.Info("addon disabled", "addon_id", addon.ID.String(), "slug", addon.Slug)
	response.OK(c, addon)
}

// UpdateAddonConfig updates the configuration for an installed addon.
// PUT /admin/addons/:id/config
func (h *Handler) UpdateAddonConfig(c *gin.Context) {
	id := c.Param("id")
	var addon Addon
	if err := h.db.Where("id = ?", id).First(&addon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "addon not found"})
		return
	}

	var req struct {
		Config string `json:"config" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.db.Model(&addon).Update("config", req.Config)
	addon.Config = req.Config

	response.OK(c, addon)
}

// ListAddonReviews returns reviews for an addon.
// GET /admin/addons/:id/reviews
func (h *Handler) ListAddonReviews(c *gin.Context) {
	id := c.Param("id")
	var reviews []AddonReview
	h.db.Where("addon_id = ?", id).Order("created_at DESC").Limit(50).Find(&reviews)
	response.OK(c, reviews)
}

// AddAddonReview adds a rating/review for an addon.
// POST /admin/addons/:id/reviews
func (h *Handler) AddAddonReview(c *gin.Context) {
	addonID := c.Param("id")
	var addon Addon
	if err := h.db.Where("id = ?", addonID).First(&addon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "addon not found"})
		return
	}

	var req struct {
		Rating  int    `json:"rating" binding:"required,min=1,max=5"`
		Review  string `json:"review"`
		Version string `json:"version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	uid, _ := uuid.Parse(fmt.Sprintf("%v", userID))

	review := AddonReview{
		AddonID: addon.ID,
		UserID:  uid,
		Rating:  req.Rating,
		Review:  req.Review,
		Version: req.Version,
	}
	if err := h.db.Create(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create review"})
		return
	}

	// Recalculate average rating
	var result struct {
		Avg   float64
		Count int
	}
	h.db.Model(&AddonReview{}).Where("addon_id = ?", addon.ID).Select("AVG(rating) as avg, COUNT(*) as count").Scan(&result)
	h.db.Model(&addon).Updates(map[string]interface{}{
		"avg_rating":   result.Avg,
		"rating_count": result.Count,
	})

	response.OK(c, review)
}

// GetMarketplaceStats returns marketplace overview statistics.
// GET /admin/addons/stats
func (h *Handler) GetMarketplaceStats(c *gin.Context) {
	var totalAddons int64
	h.db.Model(&Addon{}).Count(&totalAddons)

	var installedCount int64
	h.db.Model(&Addon{}).Where("status IN ?", []AddonStatus{AddonStatusInstalled, AddonStatusEnabled}).Count(&installedCount)

	var enabledCount int64
	h.db.Model(&Addon{}).Where("status = ?", AddonStatusEnabled).Count(&enabledCount)

	var totalDownloads int64
	h.db.Model(&Addon{}).Select("COALESCE(SUM(download_count),0)").Scan(&totalDownloads)

	type CatCount struct {
		Category string
		Count    int
	}
	var categories []CatCount
	h.db.Model(&Addon{}).Select("category, COUNT(*) as count").Group("category").Order("count DESC").Limit(10).Scan(&categories)

	response.OK(c, gin.H{
		"total_addons":    totalAddons,
		"installed_count": installedCount,
		"enabled_count":   enabledCount,
		"total_downloads": totalDownloads,
		"categories":      categories,
	})
}
