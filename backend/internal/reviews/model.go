package reviews

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Review struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SellerID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"seller_id"`
	ReviewerID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"reviewer_id"`
	ReviewerName string         `gorm:"size:255;not null" json:"reviewer_name"`
	Rating       int            `gorm:"not null;check:rating >= 1 AND rating <= 5" json:"rating"`
	Comment      string         `gorm:"type:text" json:"comment,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}
