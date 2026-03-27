package images

  import (
  	"time"

  	"github.com/google/uuid"
  )

  // ImageSize enumerates the processed variants stored for each uploaded file.
  type ImageSize string

  const (
  	SizeOriginal  ImageSize = "original"  // re-encoded JPEG, original dimensions
  	SizeLarge     ImageSize = "large"     // max 1200px on longest side
  	SizeMedium    ImageSize = "medium"    // max 600px on longest side
  	SizeThumbnail ImageSize = "thumbnail" // max 200px on longest side
  )

  // Image records every uploaded image and its R2 storage paths.
  // One upload creates up to 4 Image rows (one per size variant).
  type Image struct {
  	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
  	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
  	ListingID *uuid.UUID `gorm:"type:uuid;index" json:"listing_id,omitempty"`
  	GroupID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"group_id"`  // groups the 4 size variants
  	Size      ImageSize  `gorm:"size:20;not null" json:"size"`
  	Key       string     `gorm:"size:512;not null" json:"key"`               // R2 object key
  	URL       string     `gorm:"size:1024;not null" json:"url"`              // public CDN URL
  	Width     int        `json:"width"`
  	Height    int        `json:"height"`
  	Bytes     int64      `json:"bytes"`
  	MimeType  string     `gorm:"size:50;default:'image/jpeg'" json:"mime_type"`
  	CreatedAt time.Time  `json:"created_at"`
  }

  // UploadedImage is the response returned to the client after a successful upload.
  // It contains the public URLs for all size variants in one convenient struct.
  type UploadedImage struct {
  	GroupID   uuid.UUID `json:"group_id"`
  	Original  string    `json:"original"`
  	Large     string    `json:"large"`
  	Medium    string    `json:"medium"`
  	Thumbnail string    `json:"thumbnail"`
  }
  