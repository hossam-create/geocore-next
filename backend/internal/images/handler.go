package images

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Handler handles all image upload / delete operations.
type Handler struct {
	db *gorm.DB
	r2 *R2Client
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		db: db,
		r2: NewR2ClientFromEnv(),
	}
}

// ════════════════════════════════════════════════════════════════════════════
// POST /api/v1/images/upload
// Content-Type: multipart/form-data
// Fields:
//   images     — 1–10 image files
//   listing_id — (optional) UUID to associate images with a listing
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) Upload(c *gin.Context) {
	if h.r2 == nil {
		response.InternalError(c, fmt.Errorf("image storage not configured"))
		return
	}

	userID := c.GetString("user_id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	// ── Parse multipart form (max 50 MB total) ──────────────────────────────
	if err := c.Request.ParseMultipartForm(50 << 20); err != nil {
		response.BadRequest(c, "invalid multipart form (max 50 MB total)")
		return
	}

	fileHeaders := c.Request.MultipartForm.File["images"]
	if len(fileHeaders) == 0 {
		response.BadRequest(c, "no images provided (field name: \"images\")")
		return
	}
	if len(fileHeaders) > MaxImages {
		response.BadRequest(c, fmt.Sprintf("too many images (max %d)", MaxImages))
		return
	}

	// ── Optional listing association ─────────────────────────────────────────
	var listingID *uuid.UUID
	if lid := c.PostForm("listing_id"); lid != "" {
		parsed, e := uuid.Parse(lid)
		if e != nil {
			response.BadRequest(c, "invalid listing_id")
			return
		}
		listingID = &parsed
	}

	// ── Validate all files before processing ─────────────────────────────────
	for _, fh := range fileHeaders {
		if err := ValidateImageFile(fh); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
	}

	// ── Process + upload each file ───────────────────────────────────────────
	now := time.Now().UTC()
	yearMonth := fmt.Sprintf("%d/%02d", now.Year(), now.Month())

	uploaded := make([]UploadedImage, 0, len(fileHeaders))

	for _, fh := range fileHeaders {
		groupID := uuid.New()

		// Resize + encode JPEG variants
		variants, err := ProcessImageFile(fh)
		if err != nil {
			slog.Error("image processing failed",
				"file", fh.Filename, "error", err.Error())
			response.BadRequest(c, fmt.Sprintf("could not process %q: %s", fh.Filename, err.Error()))
			return
		}

		result := UploadedImage{GroupID: groupID}
		dbRows := make([]Image, 0, len(variants))

		for _, v := range variants {
			// R2 object key: {year}/{month}/{groupID}-{size}.jpg
			key := fmt.Sprintf("%s/%s-%s.jpg", yearMonth, groupID.String(), string(v.Size))

			publicURL, uploadErr := h.r2.Put(key, v.Data, "image/jpeg")
			if uploadErr != nil {
				slog.Error("r2 upload failed",
					"key", key, "size", v.Size, "error", uploadErr.Error())
				response.InternalError(c, uploadErr)
				return
			}

			// Populate result URLs
			switch v.Size {
			case SizeOriginal:
				result.Original = publicURL
			case SizeLarge:
				result.Large = publicURL
			case SizeMedium:
				result.Medium = publicURL
			case SizeThumbnail:
				result.Thumbnail = publicURL
			}

			dbRows = append(dbRows, Image{
				UserID:    userUUID,
				ListingID: listingID,
				GroupID:   groupID,
				Size:      v.Size,
				Key:       key,
				URL:       publicURL,
				Width:     v.Width,
				Height:    v.Height,
				Bytes:     int64(len(v.Data)),
				MimeType:  "image/jpeg",
			})
		}

		// Persist all variants in one batch insert
		if err := h.db.Create(&dbRows).Error; err != nil {
			slog.Error("failed to save image records",
				"group_id", groupID.String(), "error", err.Error())
			// Don't fail the upload — images are already on R2
		}

		slog.Info("image uploaded",
			"group_id", groupID.String(),
			"user_id", userID,
			"variants", len(variants),
		)
		uploaded = append(uploaded, result)
	}

	response.Created(c, gin.H{
		"images": uploaded,
		"count":  len(uploaded),
	})
}

// ════════════════════════════════════════════════════════════════════════════
// DELETE /api/v1/images/:id
// ════════════════════════════════════════════════════════════════════════════

// Delete removes an image group (all size variants) by group_id or image_id.
// Only the owning user can delete their images.
func (h *Handler) Delete(c *gin.Context) {
	if h.r2 == nil {
		response.InternalError(c, fmt.Errorf("image storage not configured"))
		return
	}

	idParam := c.Param("id")
	userID := c.GetString("user_id")

	parsedID, err := uuid.Parse(idParam)
	if err != nil {
		response.BadRequest(c, "invalid image id")
		return
	}

	// Try group_id first (deletes all variants); fall back to single image_id
	var images []Image
	h.db.Where("(group_id = ? OR id = ?) AND user_id = ?", parsedID, parsedID, userID).Find(&images)

	if len(images) == 0 {
		response.NotFound(c, "image")
		return
	}

	// Delete from R2
	var deleteErrors []string
	for _, img := range images {
		if err := h.r2.Delete(img.Key); err != nil {
			slog.Warn("r2 delete failed (continuing)",
				"key", img.Key, "error", err.Error())
			deleteErrors = append(deleteErrors, img.Key)
		}
	}

	// Delete DB records regardless (R2 orphans are cheaper than dangling DB rows)
	h.db.Where("(group_id = ? OR id = ?) AND user_id = ?", parsedID, parsedID, userID).
		Delete(&Image{})

	if len(deleteErrors) > 0 {
		response.OK(c, gin.H{
			"message":        "Image deleted from records, but some R2 objects could not be removed.",
			"failed_objects": strings.Join(deleteErrors, ", "),
		})
		return
	}

	response.OK(c, gin.H{"message": "Image deleted successfully."})
}

// ════════════════════════════════════════════════════════════════════════════
// GET /api/v1/images — list user's uploaded images
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) ListMine(c *gin.Context) {
	userID := c.GetString("user_id")

	var images []Image
	query := h.db.Where("user_id = ? AND size = ?", userID, SizeMedium)

	if lid := c.Query("listing_id"); lid != "" {
		query = query.Where("listing_id = ?", lid)
	}

	query.Order("created_at DESC").Find(&images)
	response.OK(c, gin.H{"images": images})
}

// ════════════════════════════════════════════════════════════════════════════
// POST /api/v1/images/presign — generate presigned upload URL
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) GetPresignedUpload(svc *ImageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc == nil {
			response.InternalError(c, fmt.Errorf("image service not configured"))
			return
		}
		userID := c.GetString("user_id")
		userUUID, _ := uuid.Parse(userID)

		var req PresignedUploadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		result, err := svc.GeneratePresignedUpload(c.Request.Context(), userUUID, req)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		response.OK(c, result)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// POST /api/v1/images/presign/batch — batch presigned upload URLs
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) GetBatchPresignedUpload(svc *ImageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc == nil {
			response.InternalError(c, fmt.Errorf("image service not configured"))
			return
		}
		userID := c.GetString("user_id")
		userUUID, _ := uuid.Parse(userID)

		var req BatchPresignedUploadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		result, err := svc.GenerateBatchPresignedUploads(c.Request.Context(), userUUID, req)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		response.OK(c, result)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// POST /api/v1/images/confirm — confirm upload + process variants
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) ConfirmUpload(svc *ImageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc == nil {
			response.InternalError(c, fmt.Errorf("image service not configured"))
			return
		}
		userID := c.GetString("user_id")
		userUUID, _ := uuid.Parse(userID)

		var req ConfirmUploadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		result, err := svc.ConfirmUpload(c.Request.Context(), userUUID, req)
		if err != nil {
			slog.Error("images: confirm upload failed", "user_id", userID, "error", err)
			response.InternalError(c, err)
			return
		}
		response.Created(c, result)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// GET /api/v1/images/group/:group_id — get all variants for an image group
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) GetImageGroup(svc *ImageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc == nil {
			response.InternalError(c, fmt.Errorf("image service not configured"))
			return
		}
		groupID, err := uuid.Parse(c.Param("group_id"))
		if err != nil {
			response.BadRequest(c, "invalid group_id")
			return
		}

		result, err := svc.GetImageGroup(c.Request.Context(), groupID)
		if err != nil {
			response.NotFound(c, "image group")
			return
		}
		response.OK(c, result)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// GET /api/v1/images/listing/:listing_id — get listing images with metadata
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) GetListingImages(svc *ImageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc == nil {
			response.InternalError(c, fmt.Errorf("image service not configured"))
			return
		}
		listingID, err := uuid.Parse(c.Param("listing_id"))
		if err != nil {
			response.BadRequest(c, "invalid listing_id")
			return
		}

		imgs, err := svc.GetListingImages(c.Request.Context(), listingID)
		if err != nil {
			response.InternalError(c, err)
			return
		}
		response.OK(c, gin.H{"images": imgs})
	}
}

// ════════════════════════════════════════════════════════════════════════════
// DELETE /api/v1/images/listing/:listing_id/:image_id — remove listing image
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) DeleteListingImage(svc *ImageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc == nil {
			response.InternalError(c, fmt.Errorf("image service not configured"))
			return
		}
		listingID, err := uuid.Parse(c.Param("listing_id"))
		if err != nil {
			response.BadRequest(c, "invalid listing_id")
			return
		}
		imageID, err := uuid.Parse(c.Param("image_id"))
		if err != nil {
			response.BadRequest(c, "invalid image_id")
			return
		}

		if err := svc.DeleteListingImage(c.Request.Context(), listingID, imageID); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		response.OK(c, gin.H{"message": "Image removed from listing."})
	}
}
