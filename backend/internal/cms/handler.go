package cms

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db        *gorm.DB
	uploadDir string
}

func NewHandler(db *gorm.DB) *Handler {
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	return &Handler{db: db, uploadDir: uploadDir}
}

// ════════════════════════════════════════════════════════════════════════════
// Hero Slides
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) ListSlides(c *gin.Context) {
	var slides []HeroSlide
	h.db.Where("is_active = true").Order("position ASC").Find(&slides)
	response.OK(c, slides)
}

func (h *Handler) CreateSlide(c *gin.Context) {
	var req struct {
		Title     string `json:"title"`
		Subtitle  string `json:"subtitle"`
		ImageURL  string `json:"image_url" binding:"required"`
		LinkURL   string `json:"link_url"`
		LinkLabel string `json:"link_label"`
		Badge     string `json:"badge"`
		Position  int    `json:"position"`
		IsActive  *bool  `json:"is_active"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	slide := HeroSlide{
		Title: req.Title, Subtitle: req.Subtitle, ImageURL: req.ImageURL,
		LinkURL: req.LinkURL, LinkLabel: req.LinkLabel, Badge: req.Badge,
		Position: req.Position, IsActive: req.IsActive != nil && *req.IsActive,
	}
	if req.StartDate != "" {
		t, _ := time.Parse("2006-01-02", req.StartDate)
		slide.StartDate = &t
	}
	if req.EndDate != "" {
		t, _ := time.Parse("2006-01-02", req.EndDate)
		slide.EndDate = &t
	}
	if err := h.db.Create(&slide).Error; err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, slide)
}

func (h *Handler) UpdateSlide(c *gin.Context) {
	id := c.Param("id")
	var slide HeroSlide
	if err := h.db.Where("id = ?", id).First(&slide).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "slide not found"})
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	// Parse date fields
	for _, df := range []string{"start_date", "end_date"} {
		if v, ok := req[df]; ok && v != nil && v != "" {
			t, _ := time.Parse("2006-01-02", fmt.Sprintf("%v", v))
			req[df] = &t
		} else if ok {
			req[df] = nil
		}
	}
	h.db.Model(&slide).Updates(req)
	h.db.Where("id = ?", id).First(&slide)
	response.OK(c, slide)
}

func (h *Handler) DeleteSlide(c *gin.Context) {
	id := c.Param("id")
	h.db.Where("id = ?", id).Delete(&HeroSlide{})
	response.OK(c, gin.H{"deleted": true})
}

func (h *Handler) ReorderSlides(c *gin.Context) {
	var req struct {
		Order []string `json:"order" binding:"required"` // array of slide IDs in desired order
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	for i, id := range req.Order {
		h.db.Model(&HeroSlide{}).Where("id = ?", id).Update("position", i)
	}
	response.OK(c, gin.H{"reordered": true})
}

// ════════════════════════════════════════════════════════════════════════════
// Content Blocks
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) ListContentBlocks(c *gin.Context) {
	var blocks []ContentBlock
	q := h.db.Model(&ContentBlock{})
	if page := c.Query("page"); page != "" {
		q = q.Where("page = ?", page)
	}
	if section := c.Query("section"); section != "" {
		q = q.Where("section = ?", section)
	}
	q.Where("is_active = true").Order("position ASC").Find(&blocks)
	response.OK(c, blocks)
}

func (h *Handler) GetContentBlock(c *gin.Context) {
	slug := c.Param("slug")
	var block ContentBlock
	if err := h.db.Where("slug = ?", slug).First(&block).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "content block not found"})
		return
	}
	response.OK(c, block)
}

func (h *Handler) CreateContentBlock(c *gin.Context) {
	var block ContentBlock
	if err := c.ShouldBindJSON(&block); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	block.ID = uuid.Nil // let DB generate
	if err := h.db.Create(&block).Error; err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, block)
}

func (h *Handler) UpdateContentBlock(c *gin.Context) {
	slug := c.Param("slug")
	var block ContentBlock
	if err := h.db.Where("slug = ?", slug).First(&block).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "content block not found"})
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.db.Model(&block).Updates(req)
	h.db.Where("slug = ?", slug).First(&block)
	response.OK(c, block)
}

func (h *Handler) DeleteContentBlock(c *gin.Context) {
	slug := c.Param("slug")
	h.db.Where("slug = ?", slug).Delete(&ContentBlock{})
	response.OK(c, gin.H{"deleted": true})
}

// ════════════════════════════════════════════════════════════════════════════
// Media Library
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) ListMedia(c *gin.Context) {
	var files []MediaFile
	q := h.db.Model(&MediaFile{})
	if folder := c.Query("folder"); folder != "" {
		q = q.Where("folder = ?", folder)
	}
	if mediaType := c.Query("type"); mediaType != "" {
		q = q.Where("type = ?", mediaType)
	}
	q.Order("created_at DESC").Limit(100).Find(&files)
	response.OK(c, files)
}

func (h *Handler) UploadMedia(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file required")
		return
	}

	folder := c.DefaultPostForm("folder", "general")
	alt := c.DefaultPostForm("alt", "")

	// Create folder if needed
	dir := filepath.Join(h.uploadDir, folder)
	os.MkdirAll(dir, 0755)

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	name := uuid.New().String() + ext
	fullPath := filepath.Join(dir, name)

	if err := c.SaveUploadedFile(file, fullPath); err != nil {
		response.BadRequest(c, "failed to save file")
		return
	}

	// Determine media type
	mediaType := MediaTypeDocument
	if strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
		mediaType = MediaTypeImage
	} else if strings.HasPrefix(file.Header.Get("Content-Type"), "video/") {
		mediaType = MediaTypeVideo
	}

	// Build URL
	urlPath := "/uploads/" + folder + "/" + name

	userID, _ := c.Get("user_id")
	uid, _ := uuid.Parse(fmt.Sprintf("%v", userID))

	media := MediaFile{
		FileName:   file.Filename,
		FilePath:   fullPath,
		URL:        urlPath,
		MimeType:   file.Header.Get("Content-Type"),
		SizeBytes:  file.Size,
		Type:       mediaType,
		Alt:        alt,
		Folder:     folder,
		UploadedBy: uid,
	}
	h.db.Create(&media)
	response.OK(c, media)
}

func (h *Handler) DeleteMedia(c *gin.Context) {
	id := c.Param("id")
	var media MediaFile
	if err := h.db.Where("id = ?", id).First(&media).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "media not found"})
		return
	}
	os.Remove(media.FilePath)
	h.db.Delete(&media)
	response.OK(c, gin.H{"deleted": true})
}

// ════════════════════════════════════════════════════════════════════════════
// Site Settings
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) ListSettings(c *gin.Context) {
	var settings []SiteSetting
	q := h.db.Model(&SiteSetting{})
	if group := c.Query("group"); group != "" {
		q = q.Where("\"group\" = ?", group)
	}
	q.Order("\"group\", key ASC").Find(&settings)
	response.OK(c, settings)
}

func (h *Handler) GetSetting(c *gin.Context) {
	key := c.Param("key")
	var setting SiteSetting
	if err := h.db.Where("key = ?", key).First(&setting).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "setting not found"})
		return
	}
	response.OK(c, setting)
}

func (h *Handler) UpdateSetting(c *gin.Context) {
	key := c.Param("key")
	var req struct {
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	var setting SiteSetting
	result := h.db.Where("key = ?", key).First(&setting)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "setting not found"})
		return
	}
	h.db.Model(&setting).Update("value", req.Value)
	h.db.Where("key = ?", key).First(&setting)
	response.OK(c, setting)
}

func (h *Handler) BulkUpdateSettings(c *gin.Context) {
	var req map[string]string // key → value
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	for key, value := range req {
		h.db.Model(&SiteSetting{}).Where("key = ?", key).Update("value", value)
	}
	response.OK(c, gin.H{"updated": len(req)})
}

// ════════════════════════════════════════════════════════════════════════════
// Navigation Menus
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) ListNavMenus(c *gin.Context) {
	location := c.DefaultQuery("location", "header")
	var menus []NavMenu
	h.db.Where("location = ? AND is_active = true", location).Order("position ASC").Find(&menus)
	response.OK(c, menus)
}

func (h *Handler) CreateNavItem(c *gin.Context) {
	var item NavMenu
	if err := c.ShouldBindJSON(&item); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	item.ID = uuid.Nil
	if err := h.db.Create(&item).Error; err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, item)
}

func (h *Handler) UpdateNavItem(c *gin.Context) {
	id := c.Param("id")
	var item NavMenu
	if err := h.db.Where("id = ?", id).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "nav item not found"})
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.db.Model(&item).Updates(req)
	h.db.Where("id = ?", id).First(&item)
	response.OK(c, item)
}

func (h *Handler) DeleteNavItem(c *gin.Context) {
	id := c.Param("id")
	h.db.Where("id = ?", id).Delete(&NavMenu{})
	response.OK(c, gin.H{"deleted": true})
}

func (h *Handler) ReorderNav(c *gin.Context) {
	var req struct {
		Order []string `json:"order" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	for i, id := range req.Order {
		h.db.Model(&NavMenu{}).Where("id = ?", id).Update("position", i)
	}
	response.OK(c, gin.H{"reordered": true})
}

// ════════════════════════════════════════════════════════════════════════════
// Public API — for frontend to consume CMS data (no auth required)
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) PublicSlides(c *gin.Context) {
	var slides []HeroSlide
	now := time.Now()
	h.db.Where("is_active = true AND (start_date IS NULL OR start_date <= ?) AND (end_date IS NULL OR end_date >= ?)", now, now).
		Order("position ASC").Find(&slides)
	response.OK(c, slides)
}

func (h *Handler) PublicContentBlocks(c *gin.Context) {
	page := c.Param("page")
	var blocks []ContentBlock
	h.db.Where("page = ? AND is_active = true", page).Order("position ASC").Find(&blocks)
	response.OK(c, blocks)
}

func (h *Handler) PublicSettings(c *gin.Context) {
	var settings []SiteSetting
	h.db.Where("\"group\" IN ?", []string{"branding", "contact", "social", "seo", "general"}).Order("key ASC").Find(&settings)
	// Return as key→value map for easy frontend consumption
	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	response.OK(c, result)
}

func (h *Handler) PublicNav(c *gin.Context) {
	location := c.Param("location")
	var menus []NavMenu
	h.db.Where("location = ? AND is_active = true", location).Order("position ASC").Find(&menus)
	response.OK(c, menus)
}

// ServeUploads serves uploaded files from the upload directory.
func ServeUploads() http.Handler {
	return http.FileServer(http.Dir("./uploads"))
}

// Ensure we use io and strconv
var _, _ = io.EOF, strconv.Itoa(0)
