package admin

  import (
  	"time"
  	"github.com/google/uuid"
  )

  // AdminLog records every admin action for audit trail.
  type AdminLog struct {
  	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
  	AdminID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"admin_id"`
  	Action     string     `gorm:"size:100;not null" json:"action"`
  	TargetType string     `gorm:"size:50" json:"target_type"`
  	TargetID   string     `gorm:"size:128" json:"target_id"`
  	Details    string     `gorm:"type:jsonb" json:"details,omitempty"`
  	IPAddress  string     `gorm:"size:45" json:"ip_address"`
  	CreatedAt  time.Time  `json:"created_at"`
  }

  // DashboardStats is the response for GET /admin/stats
  type DashboardStats struct {
  	TotalUsers         int64   `json:"total_users"`
  	ActiveUsersToday   int64   `json:"active_users_today"`
  	TotalListings      int64   `json:"total_listings"`
  	ActiveListings     int64   `json:"active_listings"`
  	TotalAuctions      int64   `json:"total_auctions"`
  	LiveAuctions       int64   `json:"live_auctions"`
  	TotalRevenue       float64 `json:"total_revenue"`
  	RevenueToday       float64 `json:"revenue_today"`
  	PendingModeration  int64   `json:"pending_moderation"`
  	ReportsPending     int64   `json:"reports_pending"`
  	NewUsersThisWeek   int64   `json:"new_users_this_week"`
  	NewListingsToday   int64   `json:"new_listings_today"`
  }
  