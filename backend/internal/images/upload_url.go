package images

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ════════════════════════════════════════════════════════════════════════════
// POST /api/v1/media/upload-url
// Returns a presigned PUT URL for direct-to-storage uploads.
// When R2 is not configured (dev environment), returns a mock response
// with a placeholder public URL so the upload flow can still be exercised.
// ════════════════════════════════════════════════════════════════════════════

var allowedMediaTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
	"image/gif":  "gif",
}

const maxUploadSize = 10 * 1024 * 1024 // 10 MB

func (h *Handler) GetUploadURL(c *gin.Context) {
	var req struct {
		Filename    string `json:"filename"     binding:"required"`
		ContentType string `json:"content_type" binding:"required"`
		Folder      string `json:"folder"`
		Size        int64  `json:"size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	ext, ok := allowedMediaTypes[req.ContentType]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": fmt.Sprintf("unsupported content type %q; allowed: image/jpeg, image/png, image/webp, image/gif", req.ContentType),
		})
		return
	}

	if req.Size > 0 && req.Size > maxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "file too large; max 10 MB"})
		return
	}

	folder := req.Folder
	if folder == "" {
		folder = "uploads"
	}

	// Generate unique key
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to generate key"})
		return
	}
	uniqueID := hex.EncodeToString(b)
	key := fmt.Sprintf("%s/%s.%s", folder, uniqueID, ext)

	// R2 configured — return a real presigned upload URL
	if h.r2 != nil {
		uploadURL, err := h.r2.PresignPutURL(key, req.ContentType, 300)
		if err == nil {
			publicURL := fmt.Sprintf("%s/%s", h.r2.PublicURL, key)
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": gin.H{
					"upload_url":     uploadURL,
					"public_url":     publicURL,
					"key":            key,
					"expires_in":     300,
					"max_size_bytes": maxUploadSize,
				},
			})
			return
		}
	}

	// R2 not configured — return mock response for dev/testing
	publicURL := fmt.Sprintf("https://picsum.photos/seed/%s/800/600", uniqueID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"upload_url":     fmt.Sprintf("https://mock-r2.dev/upload/%s?token=mock", key),
			"public_url":     publicURL,
			"key":            key,
			"expires_in":     300,
			"max_size_bytes": maxUploadSize,
			"_mock":          true,
		},
	})
}
