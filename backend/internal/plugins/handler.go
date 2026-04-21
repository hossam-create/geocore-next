package plugins

import (
	"log/slog"
	"strings"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct{ db *gorm.DB }

func NewHandler(db *gorm.DB) *Handler { return &Handler{db: db} }

// ── Browse ──────────────────────────────────────────────────────────────────

// GET /api/v1/plugins
func (h *Handler) List(c *gin.Context) {
	var items []Plugin
	q := h.db.Where("status = ?", PluginPublished).Order("install_count DESC").Limit(50)
	if cat := c.Query("category"); cat != "" {
		q = q.Where("category = ?", cat)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("name ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if free := c.Query("free"); free == "true" {
		q = q.Where("is_free = true")
	}
	if err := q.Find(&items).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, items)
}

// GET /api/v1/plugins/:slug
func (h *Handler) Get(c *gin.Context) {
	var p Plugin
	if err := h.db.Where("slug = ?", c.Param("slug")).First(&p).Error; err != nil {
		response.NotFound(c, "plugin")
		return
	}
	response.OK(c, p)
}

// ── Author ──────────────────────────────────────────────────────────────────

type CreateReq struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	IconURL     string  `json:"icon_url"`
	RepoURL     string  `json:"repo_url"`
	Price       float64 `json:"price"`
	IsFree      bool    `json:"is_free"`
}

// POST /api/v1/plugins
func (h *Handler) Create(c *gin.Context) {
	var req CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	uid, _ := uuid.Parse(c.GetString("user_id"))

	slug := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(req.Name), " ", "-"))
	cat := req.Category
	if cat == "" {
		cat = "general"
	}

	p := Plugin{
		AuthorID:    uid,
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		Category:    cat,
		IconURL:     req.IconURL,
		RepoURL:     req.RepoURL,
		Price:       req.Price,
		IsFree:      req.IsFree,
		Status:      PluginDraft,
	}
	if err := h.db.Create(&p).Error; err != nil {
		slog.Error("plugins: create failed", "error", err.Error())
		response.InternalError(c, err)
		return
	}
	response.Created(c, p)
}

// PATCH /api/v1/plugins/:slug
func (h *Handler) Update(c *gin.Context) {
	uid := c.GetString("user_id")
	var p Plugin
	if err := h.db.Where("slug = ? AND author_id = ?", c.Param("slug"), uid).First(&p).Error; err != nil {
		response.NotFound(c, "plugin")
		return
	}
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	delete(body, "id")
	delete(body, "author_id")
	delete(body, "install_count")
	h.db.Model(&p).Updates(body)
	response.OK(c, p)
}

// POST /api/v1/plugins/:slug/publish
func (h *Handler) Publish(c *gin.Context) {
	uid := c.GetString("user_id")
	var p Plugin
	if err := h.db.Where("slug = ? AND author_id = ?", c.Param("slug"), uid).First(&p).Error; err != nil {
		response.NotFound(c, "plugin")
		return
	}
	h.db.Model(&p).Update("status", PluginPublished)
	response.OK(c, gin.H{"message": "plugin published"})
}

// ── Install/Uninstall ───────────────────────────────────────────────────────

// POST /api/v1/plugins/:slug/install
func (h *Handler) Install(c *gin.Context) {
	uid, _ := uuid.Parse(c.GetString("user_id"))
	var p Plugin
	if err := h.db.Where("slug = ? AND status = ?", c.Param("slug"), PluginPublished).First(&p).Error; err != nil {
		response.NotFound(c, "published plugin")
		return
	}

	install := PluginInstall{PluginID: p.ID, UserID: uid, IsActive: true}
	result := h.db.Where("plugin_id = ? AND user_id = ?", p.ID, uid).FirstOrCreate(&install)
	if result.RowsAffected > 0 {
		h.db.Model(&p).Update("install_count", gorm.Expr("install_count + 1"))
	}
	response.OK(c, gin.H{"message": "plugin installed"})
}

// POST /api/v1/plugins/:slug/uninstall
func (h *Handler) Uninstall(c *gin.Context) {
	uid := c.GetString("user_id")
	var p Plugin
	if err := h.db.Where("slug = ?", c.Param("slug")).First(&p).Error; err != nil {
		response.NotFound(c, "plugin")
		return
	}
	result := h.db.Where("plugin_id = ? AND user_id = ?", p.ID, uid).Delete(&PluginInstall{})
	if result.RowsAffected > 0 {
		h.db.Model(&p).Update("install_count", gorm.Expr("GREATEST(install_count - 1, 0)"))
	}
	response.OK(c, gin.H{"message": "plugin uninstalled"})
}

// GET /api/v1/plugins/installed
func (h *Handler) MyInstalled(c *gin.Context) {
	uid := c.GetString("user_id")
	var installs []PluginInstall
	h.db.Where("user_id = ? AND is_active = true", uid).Find(&installs)

	var pluginIDs []uuid.UUID
	for _, i := range installs {
		pluginIDs = append(pluginIDs, i.PluginID)
	}
	if len(pluginIDs) == 0 {
		response.OK(c, []Plugin{})
		return
	}

	var items []Plugin
	h.db.Where("id IN ?", pluginIDs).Find(&items)
	response.OK(c, items)
}
