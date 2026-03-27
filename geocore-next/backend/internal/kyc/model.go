package kyc

  import (
  	"time"

  	"github.com/google/uuid"
  	"gorm.io/gorm"
  )

  const (
  	StatusPending     = "pending"
  	StatusUnderReview = "under_review"
  	StatusApproved    = "approved"
  	StatusRejected    = "rejected"
  	StatusExpired     = "expired"
  )

  const (
  	DocEmiratesID     = "emirates_id"
  	DocPassport       = "passport"
  	DocNationalID     = "national_id"
  	DocResidenceVisa  = "residence_visa"
  	DocDrivingLicense = "driving_license"
  	DocSelfie         = "selfie"
  )

  type KYCProfile struct {
  	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
  	UserID          uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
  	Status          string         `gorm:"size:20;not null;default:'pending';index" json:"status"`
  	RejectionReason string         `gorm:"type:text" json:"rejection_reason,omitempty"`
  	ApprovedAt      *time.Time     `json:"approved_at,omitempty"`
  	ApprovedByID    *uuid.UUID     `gorm:"type:uuid" json:"approved_by_id,omitempty"`
  	ExpiresAt       *time.Time     `json:"expires_at,omitempty"`
  	RiskLevel       string         `gorm:"size:10;default:'low'" json:"risk_level"`
  	Country         string         `gorm:"size:3" json:"country"`
  	Nationality     string         `gorm:"size:3" json:"nationality"`
  	DateOfBirth     *time.Time     `json:"date_of_birth,omitempty"`
  	FullName        string         `gorm:"size:255" json:"full_name"`
  	IDNumber        string         `gorm:"size:100" json:"id_number"`
  	CreatedAt       time.Time      `json:"created_at"`
  	UpdatedAt       time.Time      `json:"updated_at"`
  	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
  	Documents       []KYCDocument  `gorm:"foreignKey:KYCProfileID" json:"documents,omitempty"`
  }

  type KYCDocument struct {
  	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
  	KYCProfileID uuid.UUID `gorm:"type:uuid;not null;index" json:"kyc_profile_id"`
  	DocumentType string    `gorm:"size:50;not null" json:"document_type"`
  	FileURL      string    `gorm:"type:text;not null" json:"file_url"`
  	FileKey      string    `gorm:"type:text" json:"-"`
  	MimeType     string    `gorm:"size:50" json:"mime_type"`
  	Side         string    `gorm:"size:10;default:'front'" json:"side"`
  	Verified     bool      `gorm:"default:false" json:"verified"`
  	CreatedAt    time.Time `json:"created_at"`
  }

  type KYCAuditLog struct {
  	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
  	ProfileID uuid.UUID `gorm:"type:uuid;not null;index" json:"profile_id"`
  	AdminID   uuid.UUID `gorm:"type:uuid;not null" json:"admin_id"`
  	Action    string    `gorm:"size:50;not null" json:"action"`
  	Notes     string    `gorm:"type:text" json:"notes,omitempty"`
  	CreatedAt time.Time `json:"created_at"`
  }
  