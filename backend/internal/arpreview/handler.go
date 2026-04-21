package arpreview

import (
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct{ db *gorm.DB }

func NewHandler(db *gorm.DB) *Handler { return &Handler{db: db} }

// GET /api/v1/listings/:id/3d-models
func (h *Handler) ListModels(c *gin.Context) {
	var models []Listing3DModel
	if err := h.db.Where("listing_id = ?", c.Param("id")).Order("is_primary DESC, created_at ASC").Find(&models).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, models)
}

// POST /api/v1/listings/:id/3d-models
func (h *Handler) AddModel(c *gin.Context) {
	var body struct {
		ModelURL      string `json:"model_url" binding:"required"`
		PosterURL     string `json:"poster_url"`
		Format        string `json:"format"`
		FileSizeBytes int64  `json:"file_size_bytes"`
		IsPrimary     bool   `json:"is_primary"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	format := body.Format
	if format == "" {
		format = "glb"
	}

	m := Listing3DModel{
		ModelURL:      body.ModelURL,
		PosterURL:     body.PosterURL,
		Format:        format,
		FileSizeBytes: body.FileSizeBytes,
		IsPrimary:     body.IsPrimary,
	}

	if err := h.db.Raw("SELECT id FROM listings WHERE id = ?", c.Param("id")).Scan(&m.ListingID).Error; err != nil || m.ListingID.String() == "00000000-0000-0000-0000-000000000000" {
		response.NotFound(c, "listing")
		return
	}

	if body.IsPrimary {
		h.db.Model(&Listing3DModel{}).Where("listing_id = ?", m.ListingID).Update("is_primary", false)
	}

	if err := h.db.Create(&m).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.Created(c, m)
}

// DELETE /api/v1/listings/:id/3d-models/:modelId
func (h *Handler) DeleteModel(c *gin.Context) {
	result := h.db.Where("id = ? AND listing_id = ?", c.Param("modelId"), c.Param("id")).Delete(&Listing3DModel{})
	if result.RowsAffected == 0 {
		response.NotFound(c, "3d model")
		return
	}
	response.OK(c, gin.H{"message": "model deleted"})
}
