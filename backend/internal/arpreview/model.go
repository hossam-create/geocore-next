package arpreview

import (
	"time"

	"github.com/google/uuid"
)

type Listing3DModel struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ListingID     uuid.UUID `gorm:"type:uuid;not null;index"                        json:"listing_id"`
	ModelURL      string    `gorm:"type:text;not null"                              json:"model_url"`
	PosterURL     string    `gorm:"type:text"                                       json:"poster_url,omitempty"`
	Format        string    `gorm:"size:20;not null;default:'glb'"                  json:"format"`
	FileSizeBytes int64     `gorm:"default:0"                                       json:"file_size_bytes"`
	IsPrimary     bool      `gorm:"not null;default:false"                          json:"is_primary"`
	CreatedAt     time.Time `json:"created_at"`
}

func (Listing3DModel) TableName() string { return "listing_3d_models" }
