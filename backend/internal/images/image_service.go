package images

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"log/slog"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// ImageService — production S3 image management with presigned uploads,
// WebP conversion, multi-variant processing, CDN integration, and
// listing image association.
// ════════════════════════════════════════════════════════════════════════════

// ImageService orchestrates the full image lifecycle.
type ImageService struct {
	db *gorm.DB
	r2 *R2Client
}

// NewImageService creates a new ImageService. Returns nil if R2 is not configured.
func NewImageService(db *gorm.DB) *ImageService {
	r2 := NewR2ClientFromEnv()
	if r2 == nil {
		slog.Warn("images: R2 not configured — ImageService disabled")
		return nil
	}
	return &ImageService{db: db, r2: r2}
}

// ── Presigned Upload Flow ─────────────────────────────────────────────────────

// PresignedUploadRequest is the input for generating a presigned upload URL.
type PresignedUploadRequest struct {
	Filename    string `json:"filename"     binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
	Folder      string `json:"folder"`
	Size        int64  `json:"size"`
	ListingID   string `json:"listing_id"` // optional — associate after upload
}

// PresignedUploadResponse contains the presigned URL and metadata.
type PresignedUploadResponse struct {
	UploadURL string `json:"upload_url"`
	PublicURL string `json:"public_url"`
	Key       string `json:"key"`
	ExpiresIn int    `json:"expires_in"`
	MaxSize   int64  `json:"max_size_bytes"`
	UploadID  string `json:"upload_id"` // for confirming upload later
}

// GeneratePresignedUpload creates a presigned PUT URL for direct-to-S3 upload.
func (s *ImageService) GeneratePresignedUpload(ctx context.Context, userID uuid.UUID, req PresignedUploadRequest) (*PresignedUploadResponse, error) {
	ext, ok := allowedMediaTypes[req.ContentType]
	if !ok {
		return nil, fmt.Errorf("unsupported content type %q; allowed: jpeg, png, webp, gif", req.ContentType)
	}
	if req.Size > 0 && req.Size > maxUploadSize {
		return nil, fmt.Errorf("file too large; max %d MB", maxUploadSize/1024/1024)
	}

	folder := req.Folder
	if folder == "" {
		folder = "uploads"
	}

	now := time.Now().UTC()
	yearMonth := fmt.Sprintf("%d/%02d", now.Year(), now.Month())
	uniqueID := uuid.New()
	key := fmt.Sprintf("%s/%s-%s.%s", yearMonth, folder, uniqueID.String()[:8], ext)

	uploadURL, err := s.r2.PresignPutURL(key, req.ContentType, 300)
	if err != nil {
		return nil, fmt.Errorf("presign: %w", err)
	}

	publicURL := fmt.Sprintf("%s/%s", s.r2.PublicURL, key)

	return &PresignedUploadResponse{
		UploadURL: uploadURL,
		PublicURL: publicURL,
		Key:       key,
		ExpiresIn: 300,
		MaxSize:   maxUploadSize,
		UploadID:  uniqueID.String(),
	}, nil
}

// ── Confirm Upload + Process Variants ─────────────────────────────────────────

// ConfirmUploadRequest is sent by the client after a successful direct upload.
type ConfirmUploadRequest struct {
	Key       string `json:"key"        binding:"required"`
	ListingID string `json:"listing_id"` // optional
}

// ConfirmUploadResponse contains the processed image variants.
type ConfirmUploadResponse struct {
	GroupID   uuid.UUID      `json:"group_id"`
	Variants  []ImageVariant `json:"variants"`
	ListingID *uuid.UUID     `json:"listing_id,omitempty"`
}

// ImageVariant represents one size variant of a processed image.
type ImageVariant struct {
	Size     ImageSize `json:"size"`
	URL      string    `json:"url"`
	Width    int       `json:"width"`
	Height   int       `json:"height"`
	Bytes    int64     `json:"bytes"`
	MimeType string    `json:"mime_type"`
}

// ConfirmUpload fetches the uploaded original from R2, processes it into
// variants (thumbnail, medium, large, original), re-uploads them as WebP,
// and creates DB records.
func (s *ImageService) ConfirmUpload(ctx context.Context, userID uuid.UUID, req ConfirmUploadRequest) (*ConfirmUploadResponse, error) {
	// 1. Download the original from R2
	originalData, err := s.r2.Get(req.Key)
	if err != nil {
		return nil, fmt.Errorf("fetch original: %w", err)
	}

	// 2. Decode the image
	src, format, err := DecodeBytes(originalData)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	// 3. Process into variants
	groupID := uuid.New()
	now := time.Now().UTC()
	yearMonth := fmt.Sprintf("%d/%02d", now.Year(), now.Month())

	var listingID *uuid.UUID
	if req.ListingID != "" {
		parsed, e := uuid.Parse(req.ListingID)
		if e == nil {
			listingID = &parsed
		}
	}

	type spec struct {
		size ImageSize
		max  int
	}
	specs := []spec{
		{SizeOriginal, MaxOriginalDim},
		{SizeLarge, MaxLargeDim},
		{SizeMedium, MaxMediumDim},
		{SizeThumbnail, MaxThumbnailDim},
	}

	var variants []ImageVariant
	var dbRows []Image

	for _, sp := range specs {
		resized := resizeFit(src, sp.max)

		// Encode as WebP (preferred) or JPEG fallback
		var data []byte
		var mimeType string
		var ext string

		if WebPSupported() {
			data, err = EncodeWebP(resized, WebPQuality)
			if err != nil {
				// Fallback to JPEG if WebP encoding fails
				slog.Warn("images: WebP encode failed, falling back to JPEG", "size", sp.size, "error", err)
				data, err = encodeJPEG(resized)
				if err != nil {
					return nil, fmt.Errorf("encode %s: %w", sp.size, err)
				}
				mimeType = "image/jpeg"
				ext = "jpg"
			} else {
				mimeType = "image/webp"
				ext = "webp"
			}
		} else {
			data, err = encodeJPEG(resized)
			if err != nil {
				return nil, fmt.Errorf("encode %s: %w", sp.size, err)
			}
			mimeType = "image/jpeg"
			ext = "jpg"
		}

		key := fmt.Sprintf("%s/%s-%s.%s", yearMonth, groupID.String()[:8], string(sp.size), ext)
		publicURL, uploadErr := s.r2.Put(key, data, mimeType)
		if uploadErr != nil {
			slog.Error("images: R2 upload failed", "key", key, "error", uploadErr)
			return nil, fmt.Errorf("upload %s: %w", sp.size, uploadErr)
		}

		b := resized.Bounds()
		variant := ImageVariant{
			Size:     sp.size,
			URL:      publicURL,
			Width:    b.Dx(),
			Height:   b.Dy(),
			Bytes:    int64(len(data)),
			MimeType: mimeType,
		}
		variants = append(variants, variant)

		dbRows = append(dbRows, Image{
			UserID:    userID,
			ListingID: listingID,
			GroupID:   groupID,
			Size:      sp.size,
			Key:       key,
			URL:       publicURL,
			Width:     b.Dx(),
			Height:    b.Dy(),
			Bytes:     int64(len(data)),
			MimeType:  mimeType,
		})
	}

	// 4. Persist DB records
	if err := s.db.Create(&dbRows).Error; err != nil {
		slog.Error("images: failed to save image records", "group_id", groupID, "error", err)
	}

	// 5. If listing_id provided, also create ListingImage records
	if listingID != nil {
		for i, v := range variants {
			if v.Size == SizeMedium || v.Size == SizeLarge {
				s.db.Create(&ListingImageAssoc{
					ID:        uuid.New(),
					ListingID: *listingID,
					ImageID:   dbRows[i].ID,
					GroupID:   groupID,
					URL:       v.URL,
					Width:     v.Width,
					Height:    v.Height,
					Bytes:     v.Bytes,
					MimeType:  v.MimeType,
					Variant:   string(v.Size),
					SortOrder: 0,
					IsCover:   v.Size == SizeLarge,
				})
				break // only one ListingImage per group (use large as default)
			}
		}
	}

	slog.Info("images: upload confirmed and processed",
		"group_id", groupID,
		"user_id", userID,
		"original_format", format,
		"variants", len(variants),
	)

	return &ConfirmUploadResponse{
		GroupID:   groupID,
		Variants:  variants,
		ListingID: listingID,
	}, nil
}

// ── Server-side Upload (multipart) ────────────────────────────────────────────

// UploadAndProcess handles a multipart file upload, processes it into variants,
// uploads to R2, and creates DB records. Returns the grouped result.
func (s *ImageService) UploadAndProcess(ctx context.Context, userID uuid.UUID, fh *multipart.FileHeader, listingID *uuid.UUID) (*UploadedImage, error) {
	if err := ValidateImageFile(fh); err != nil {
		return nil, err
	}

	groupID := uuid.New()
	now := time.Now().UTC()
	yearMonth := fmt.Sprintf("%d/%02d", now.Year(), now.Month())

	// Process into variants using WebP when available
	variants, err := ProcessImageFile(fh)
	if err != nil {
		return nil, fmt.Errorf("process: %w", err)
	}

	result := UploadedImage{GroupID: groupID}
	dbRows := make([]Image, 0, len(variants))

	for _, v := range variants {
		// Re-encode as WebP if supported
		src, _, decodeErr := DecodeBytes(v.Data)
		if decodeErr != nil {
			// Use JPEG variant as-is
			src = nil
		}

		var data []byte
		var mimeType string
		var ext string

		if src != nil && WebPSupported() {
			resized := resizeFit(src, maxDimForSize(v.Size))
			webpData, webpErr := EncodeWebP(resized, WebPQuality)
			if webpErr == nil {
				data = webpData
				mimeType = "image/webp"
				ext = "webp"
			} else {
				data = v.Data
				mimeType = "image/jpeg"
				ext = "jpg"
			}
		} else {
			data = v.Data
			mimeType = "image/jpeg"
			ext = "jpg"
		}

		key := fmt.Sprintf("%s/%s-%s.%s", yearMonth, groupID.String()[:8], string(v.Size), ext)
		publicURL, uploadErr := s.r2.Put(key, data, mimeType)
		if uploadErr != nil {
			slog.Error("images: R2 upload failed", "key", key, "error", uploadErr)
			return nil, fmt.Errorf("upload %s: %w", v.Size, uploadErr)
		}

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
			UserID:    userID,
			ListingID: listingID,
			GroupID:   groupID,
			Size:      v.Size,
			Key:       key,
			URL:       publicURL,
			Width:     v.Width,
			Height:    v.Height,
			Bytes:     int64(len(data)),
			MimeType:  mimeType,
		})
	}

	if err := s.db.Create(&dbRows).Error; err != nil {
		slog.Error("images: failed to save image records", "group_id", groupID, "error", err)
	}

	return &result, nil
}

// ── Listing Image Association ─────────────────────────────────────────────────

// ListingImageAssoc is the enhanced listing-image join table with metadata.
type ListingImageAssoc struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ListingID uuid.UUID `gorm:"type:uuid;not null;index" json:"listing_id"`
	ImageID   uuid.UUID `gorm:"type:uuid;not null;index" json:"image_id"`
	GroupID   uuid.UUID `gorm:"type:uuid;not null;index" json:"group_id"`
	URL       string    `gorm:"not null" json:"url"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Bytes     int64     `json:"bytes"`
	MimeType  string    `gorm:"size:50;default:'image/webp'" json:"mime_type"`
	Variant   string    `gorm:"size:20;not null;default:'large'" json:"variant"` // thumbnail|medium|large|original
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	IsCover   bool      `gorm:"default:false" json:"is_cover"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName overrides the GORM table name.
func (ListingImageAssoc) TableName() string { return "listing_images" }

// AssociateImagesWithListing creates listing_images records for a listing from
// a list of image group IDs. It picks the "large" variant for each group.
func (s *ImageService) AssociateImagesWithListing(ctx context.Context, listingID uuid.UUID, groupIDs []uuid.UUID) error {
	for i, gid := range groupIDs {
		// Find the large variant for this group
		var img Image
		if err := s.db.Where("group_id = ? AND size = ?", gid, SizeLarge).First(&img).Error; err != nil {
			// Fallback to medium, then original
			if err := s.db.Where("group_id = ? AND size = ?", gid, SizeMedium).First(&img).Error; err != nil {
				if err := s.db.Where("group_id = ?", gid).Order("size ASC").First(&img).Error; err != nil {
					slog.Warn("images: group not found, skipping", "group_id", gid)
					continue
				}
			}
		}

		assoc := ListingImageAssoc{
			ID:        uuid.New(),
			ListingID: listingID,
			ImageID:   img.ID,
			GroupID:   gid,
			URL:       img.URL,
			Width:     img.Width,
			Height:    img.Height,
			Bytes:     img.Bytes,
			MimeType:  img.MimeType,
			Variant:   string(img.Size),
			SortOrder: i,
			IsCover:   i == 0,
		}
		if err := s.db.Create(&assoc).Error; err != nil {
			slog.Error("images: failed to associate image with listing",
				"listing_id", listingID, "group_id", gid, "error", err)
		}
	}
	return nil
}

// GetListingImages returns all images for a listing with full metadata.
func (s *ImageService) GetListingImages(ctx context.Context, listingID uuid.UUID) ([]ListingImageAssoc, error) {
	var images []ListingImageAssoc
	if err := s.db.Where("listing_id = ?", listingID).
		Order("sort_order ASC, created_at ASC").
		Find(&images).Error; err != nil {
		return nil, err
	}
	return images, nil
}

// GetListingImageVariants returns all size variants for a specific image group
// attached to a listing.
func (s *ImageService) GetListingImageVariants(ctx context.Context, listingID, groupID uuid.UUID) ([]Image, error) {
	var images []Image
	if err := s.db.Where("group_id = ?", groupID).
		Order("size ASC").
		Find(&images).Error; err != nil {
		return nil, err
	}
	return images, nil
}

// DeleteListingImage removes a listing image association and optionally deletes
// the R2 objects if the image is not associated with any other listing.
func (s *ImageService) DeleteListingImage(ctx context.Context, listingID, imageID uuid.UUID) error {
	var assoc ListingImageAssoc
	if err := s.db.Where("listing_id = ? AND id = ?", listingID, imageID).First(&assoc).Error; err != nil {
		return fmt.Errorf("listing image not found")
	}

	// Delete the association
	if err := s.db.Delete(&assoc).Error; err != nil {
		return err
	}

	// Check if this group is still referenced by other listings
	var count int64
	s.db.Model(&ListingImageAssoc{}).Where("group_id = ?", assoc.GroupID).Count(&count)
	if count == 0 {
		// No other listings reference this group — delete from R2 + images table
		var groupImages []Image
		s.db.Where("group_id = ?", assoc.GroupID).Find(&groupImages)
		for _, img := range groupImages {
			if err := s.r2.Delete(img.Key); err != nil {
				slog.Warn("images: R2 delete failed", "key", img.Key, "error", err)
			}
		}
		s.db.Where("group_id = ?", assoc.GroupID).Delete(&Image{})
	}

	return nil
}

// ── CDN Helpers ────────────────────────────────────────────────────────────────

// CDNConfig holds CDN-related configuration.
type CDNConfig struct {
	PublicURL string // e.g. https://cdn.geocore.com
	WebPBase  string // e.g. https://cdn.geocore.com — same as PublicURL for R2
	Enabled   bool
}

// GetCDNConfig returns the CDN configuration from R2 env vars.
func (s *ImageService) GetCDNConfig() CDNConfig {
	if s.r2 == nil {
		return CDNConfig{Enabled: false}
	}
	return CDNConfig{
		PublicURL: s.r2.PublicURL,
		WebPBase:  s.r2.PublicURL,
		Enabled:   s.r2.PublicURL != "",
	}
}

// RewriteToWebPURL takes a JPEG image URL and returns the WebP equivalent.
// If the CDN is configured, it rewrites to the CDN URL.
// This is useful for <picture> elements in the frontend.
func RewriteToWebPURL(jpegURL, cdnBase string) string {
	if cdnBase == "" {
		return jpegURL
	}
	// Replace .jpg with .webp extension
	url := jpegURL
	if len(url) > 4 && url[len(url)-4:] == ".jpg" {
		url = url[:len(url)-4] + ".webp"
	}
	if len(url) > 5 && url[len(url)-5:] == ".jpeg" {
		url = url[:len(url)-5] + ".webp"
	}
	return url
}

// ── Migration Helper ──────────────────────────────────────────────────────────

// MigrateTextArrayImages migrates the legacy TEXT[] images column on the
// listings table to the new listing_images table. This is idempotent — it
// skips listings that already have listing_images records.
func (s *ImageService) MigrateTextArrayImages(ctx context.Context, batchSize int) (int, error) {
	if batchSize <= 0 {
		batchSize = 100
	}

	// Find listings that have images in the TEXT[] column but no listing_images
	type row struct {
		ID     uuid.UUID
		Images string // TEXT[] comes as a string like {url1,url2}
	}

	var rows []row
	err := s.db.Raw(`
		SELECT l.id, l.images::text
		FROM listings l
		WHERE l.images IS NOT NULL
		  AND array_length(l.images, 1) > 0
		  AND NOT EXISTS (
		    SELECT 1 FROM listing_images li WHERE li.listing_id = l.id
		  )
		LIMIT ?
	`, batchSize).Scan(&rows).Error
	if err != nil {
		return 0, fmt.Errorf("query listings: %w", err)
	}

	migrated := 0
	for _, r := range rows {
		urls := parseTextArray(r.Images)
		if len(urls) == 0 {
			continue
		}

		for i, u := range urls {
			assoc := ListingImageAssoc{
				ID:        uuid.New(),
				ListingID: r.ID,
				ImageID:   uuid.Nil, // no Image record for legacy URLs
				GroupID:   uuid.New(),
				URL:       u,
				Variant:   "original",
				SortOrder: i,
				IsCover:   i == 0,
				MimeType:  "image/jpeg", // assume JPEG for legacy
			}
			if err := s.db.Create(&assoc).Error; err != nil {
				slog.Error("images: migration failed for listing",
					"listing_id", r.ID, "error", err)
			}
		}
		migrated++
	}

	slog.Info("images: TEXT[] migration batch complete", "migrated", migrated, "batch", batchSize)
	return migrated, nil
}

// parseTextArray parses PostgreSQL TEXT[] format "{url1,url2,url3}" into []string.
func parseTextArray(raw string) []string {
	if len(raw) < 2 {
		return nil
	}
	// Strip leading { and trailing }
	if raw[0] == '{' {
		raw = raw[1:]
	}
	if len(raw) > 0 && raw[len(raw)-1] == '}' {
		raw = raw[:len(raw)-1]
	}
	if raw == "" {
		return nil
	}

	// Simple comma split (doesn't handle escaped commas, but URLs don't have them)
	var result []string
	start := 0
	for i := 0; i <= len(raw); i++ {
		if i == len(raw) || raw[i] == ',' {
			part := raw[start:i]
			if part != "" && part != "NULL" {
				result = append(result, part)
			}
			start = i + 1
		}
	}
	return result
}

// ── Image Group URLs ──────────────────────────────────────────────────────────

// ImageGroupResult contains all variant URLs for an image group.
type ImageGroupResult struct {
	GroupID   uuid.UUID `json:"group_id"`
	Original  string    `json:"original"`
	Large     string    `json:"large"`
	Medium    string    `json:"medium"`
	Thumbnail string    `json:"thumbnail"`
}

// GetImageGroup returns all variant URLs for a given group ID.
func (s *ImageService) GetImageGroup(ctx context.Context, groupID uuid.UUID) (*ImageGroupResult, error) {
	var images []Image
	if err := s.db.Where("group_id = ?", groupID).Find(&images).Error; err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("image group not found")
	}

	result := &ImageGroupResult{GroupID: groupID}
	for _, img := range images {
		switch img.Size {
		case SizeOriginal:
			result.Original = img.URL
		case SizeLarge:
			result.Large = img.URL
		case SizeMedium:
			result.Medium = img.URL
		case SizeThumbnail:
			result.Thumbnail = img.URL
		}
	}
	return result, nil
}

// ── Batch Upload URL Generation ───────────────────────────────────────────────

// BatchPresignedUploadRequest allows generating multiple presigned URLs at once.
type BatchPresignedUploadRequest struct {
	Files []PresignedUploadRequest `json:"files" binding:"required"`
}

// BatchPresignedUploadResponse contains all presigned URLs.
type BatchPresignedUploadResponse struct {
	Uploads []PresignedUploadResponse `json:"uploads"`
	Count   int                       `json:"count"`
}

// GenerateBatchPresignedUploads creates multiple presigned PUT URLs.
func (s *ImageService) GenerateBatchPresignedUploads(ctx context.Context, userID uuid.UUID, req BatchPresignedUploadRequest) (*BatchPresignedUploadResponse, error) {
	if len(req.Files) > MaxImages {
		return nil, fmt.Errorf("too many files (max %d)", MaxImages)
	}

	uploads := make([]PresignedUploadResponse, 0, len(req.Files))
	for _, f := range req.Files {
		resp, err := s.GeneratePresignedUpload(ctx, userID, f)
		if err != nil {
			return nil, fmt.Errorf("file %q: %w", f.Filename, err)
		}
		uploads = append(uploads, *resp)
	}

	return &BatchPresignedUploadResponse{
		Uploads: uploads,
		Count:   len(uploads),
	}, nil
}

// ── Helpers ────────────────────────────────────────────────────────────────────

func maxDimForSize(size ImageSize) int {
	switch size {
	case SizeThumbnail:
		return MaxThumbnailDim
	case SizeMedium:
		return MaxMediumDim
	case SizeLarge:
		return MaxLargeDim
	default:
		return MaxOriginalDim
	}
}

// DecodeBytes decodes image bytes into an image.Image.
func DecodeBytes(data []byte) (img image.Image, format string, err error) {
	return image.Decode(bytes.NewReader(data))
}

// MarshalJSON helps serialize the CDN config for API responses.
func (c CDNConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"public_url": c.PublicURL,
		"webp_base":  c.WebPBase,
		"enabled":    c.Enabled,
	})
}
