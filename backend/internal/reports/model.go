package reports

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TargetType string
type Status string

const (
	TargetListing TargetType = "listing"
	TargetUser    TargetType = "user"

	StatusPending   Status = "pending"
	StatusReviewed  Status = "reviewed"
	StatusDismissed Status = "dismissed"
	StatusActioned  Status = "actioned"
)

type Report struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ReporterID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"reporter_id"`
	TargetType  TargetType `gorm:"size:20;not null" json:"target_type"`
	TargetID    uuid.UUID  `gorm:"type:uuid;not null" json:"target_id"`
	Reason      string     `gorm:"size:100;not null" json:"reason"`
	Description string     `gorm:"type:text" json:"description,omitempty"`
	Status      Status     `gorm:"size:20;not null;default:'pending';index" json:"status"`
	ReviewedBy  *uuid.UUID `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	AdminNote   string     `gorm:"type:text" json:"admin_note,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Joined fields (not stored in DB)
	ReporterName string `gorm:"-" json:"reporter_name,omitempty"`
}

func (Report) TableName() string { return "reports" }

func (r *Report) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
