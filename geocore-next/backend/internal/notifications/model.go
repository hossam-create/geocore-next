package notifications

  import (
  	"time"

  	"github.com/google/uuid"
  	"gorm.io/gorm"
  )

  // ════════════════════════════════════════════════════════════════════════════
  // Notification types
  // ════════════════════════════════════════════════════════════════════════════

  const (
  	TypeNewBid          = "new_bid"
  	TypeOutbid          = "outbid"
  	TypeAuctionWon      = "auction_won"
  	TypeAuctionEnded    = "auction_ended"
  	TypeNewMessage      = "new_message"
  	TypeListingApproved = "listing_approved"
  	TypeListingRejected = "listing_rejected"
  	TypePaymentSuccess  = "payment_success"
  	TypePaymentFailed   = "payment_failed"
  	TypeEscrowReleased  = "escrow_released"
  	TypeNewReview       = "new_review"
  )

  // ════════════════════════════════════════════════════════════════════════════
  // Models
  // ════════════════════════════════════════════════════════════════════════════

  // Notification is a single in-app / push / email notification record.
  type Notification struct {
  	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
  	UserID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
  	Type      string         `gorm:"size:50;not null;index" json:"type"`
  	Title     string         `gorm:"size:255" json:"title"`
  	Body      string         `gorm:"type:text" json:"body"`
  	Data      string         `gorm:"type:jsonb" json:"data,omitempty"` // arbitrary JSON payload
  	Read      bool           `gorm:"default:false;index" json:"read"`
  	ReadAt    *time.Time     `json:"read_at,omitempty"`
  	CreatedAt time.Time      `json:"created_at"`
  	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
  }

  // NotificationPreference holds per-user notification channel preferences.
  type NotificationPreference struct {
  	UserID               uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
  	EmailNewBid          bool      `gorm:"default:true" json:"email_new_bid"`
  	EmailOutbid          bool      `gorm:"default:true" json:"email_outbid"`
  	EmailMessage         bool      `gorm:"default:true" json:"email_message"`
  	EmailListingApproved bool      `gorm:"default:true" json:"email_listing_approved"`
  	PushNewBid           bool      `gorm:"default:true" json:"push_new_bid"`
  	PushOutbid           bool      `gorm:"default:true" json:"push_outbid"`
  	PushMessage          bool      `gorm:"default:true" json:"push_message"`
  	InAppEnabled         bool      `gorm:"default:true" json:"in_app_enabled"`
  	CreatedAt            time.Time `json:"created_at"`
  	UpdatedAt            time.Time `json:"updated_at"`
  }

  // PushToken stores an FCM device token for a user.
  type PushToken struct {
  	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
  	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
  	Token     string    `gorm:"type:text;not null;uniqueIndex" json:"token"`
  	Platform  string    `gorm:"size:20" json:"platform"` // web | ios | android
  	CreatedAt time.Time `json:"created_at"`
  }
  