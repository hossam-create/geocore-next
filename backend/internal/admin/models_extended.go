package admin

import (
	"time"

	"github.com/google/uuid"
)

// ── Section 2: User Groups ──────────────────────────────────────────────────

type UserGroup struct {
	ID                int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name              string    `gorm:"size:100;not null" json:"name"`
	Slug              string    `gorm:"size:100;uniqueIndex" json:"slug"`
	Description       string    `json:"description,omitempty"`
	PricePlanID       *int      `json:"price_plan_id,omitempty"`
	Permissions       string    `gorm:"type:jsonb;default:'{}'" json:"permissions"`
	MaxActiveListings int       `gorm:"default:10" json:"max_active_listings"`
	CanPlaceAuctions  bool      `gorm:"default:true" json:"can_place_auctions"`
	RequiresApproval  bool      `gorm:"default:false" json:"requires_approval"`
	IsDefault         bool      `gorm:"default:false" json:"is_default"`
	SortOrder         int       `gorm:"default:0" json:"sort_order"`
	CreatedAt         time.Time `json:"created_at"`
}

func (UserGroup) TableName() string { return "user_groups" }

// ── Section 2: Custom User Fields ───────────────────────────────────────────

type UserCustomField struct {
	ID          int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	Label       string    `gorm:"size:100;not null" json:"label"`
	LabelEn     string    `gorm:"size:100" json:"label_en,omitempty"`
	FieldType   string    `gorm:"size:20;not null" json:"field_type"`
	Options     string    `gorm:"type:jsonb;default:'[]'" json:"options"`
	IsRequired  bool      `gorm:"default:false" json:"is_required"`
	Placeholder string    `gorm:"size:200" json:"placeholder,omitempty"`
	SortOrder   int       `gorm:"default:0" json:"sort_order"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

func (UserCustomField) TableName() string { return "user_custom_fields" }

// ── Section 3: Listing Extras ───────────────────────────────────────────────

type ListingExtra struct {
	ID           int     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string  `gorm:"size:100" json:"name"`
	Description  string  `json:"description,omitempty"`
	Type         string  `gorm:"size:20" json:"type"`
	Price        float64 `gorm:"type:decimal(10,2);default:0" json:"price"`
	DurationDays *int    `json:"duration_days,omitempty"`
	IsActive     bool    `gorm:"default:true" json:"is_active"`
}

func (ListingExtra) TableName() string { return "listing_extras" }

type ListingExtraPurchase struct {
	ID          int          `gorm:"primaryKey;autoIncrement" json:"id"`
	ListingID   uuid.UUID    `gorm:"type:uuid;not null;index" json:"listing_id"`
	ExtraID     int          `gorm:"not null" json:"extra_id"`
	PurchasedAt time.Time    `json:"purchased_at"`
	ExpiresAt   *time.Time   `json:"expires_at,omitempty"`
	Extra       ListingExtra `gorm:"foreignKey:ExtraID" json:"extra,omitempty"`
}

func (ListingExtraPurchase) TableName() string { return "listing_extra_purchases" }

// ── Section 6: Payment Gateways ─────────────────────────────────────────────

type PaymentGateway struct {
	ID                  int     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                string  `gorm:"size:100" json:"name"`
	Slug                string  `gorm:"size:50;uniqueIndex" json:"slug"`
	DisplayName         string  `gorm:"size:100" json:"display_name"`
	IsActive            bool    `gorm:"default:false" json:"is_active"`
	IsSandbox           bool    `gorm:"default:true" json:"is_sandbox"`
	Config              string  `gorm:"type:jsonb;default:'{}'" json:"config"`
	SupportedCurrencies string  `gorm:"type:jsonb;default:'[\"EGP\",\"USD\"]'" json:"supported_currencies"`
	FeePercent          float64 `gorm:"type:decimal(5,2);default:0" json:"fee_percent"`
	FeeFixed            float64 `gorm:"type:decimal(10,2);default:0" json:"fee_fixed"`
	SortOrder           int     `gorm:"default:0" json:"sort_order"`
}

func (PaymentGateway) TableName() string { return "payment_gateways" }

// ── Section 6: Invoices ─────────────────────────────────────────────────────

type Invoice struct {
	ID               int        `gorm:"primaryKey;autoIncrement" json:"id"`
	InvoiceNumber    string     `gorm:"size:20;uniqueIndex" json:"invoice_number"`
	UserID           *uuid.UUID `gorm:"type:uuid" json:"user_id,omitempty"`
	Items            string     `gorm:"type:jsonb;not null;default:'[]'" json:"items"`
	Subtotal         float64    `gorm:"type:decimal(10,2);default:0" json:"subtotal"`
	Discount         float64    `gorm:"type:decimal(10,2);default:0" json:"discount"`
	Tax              float64    `gorm:"type:decimal(10,2);default:0" json:"tax"`
	Total            float64    `gorm:"type:decimal(10,2);default:0" json:"total"`
	Status           string     `gorm:"size:20;default:'pending'" json:"status"`
	GatewayID        *int       `json:"gateway_id,omitempty"`
	GatewayReference string     `gorm:"size:200" json:"gateway_reference,omitempty"`
	Notes            string     `json:"notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	PaidAt           *time.Time `json:"paid_at,omitempty"`
}

func (Invoice) TableName() string { return "invoices" }

// ── Section 6: Discount Codes ───────────────────────────────────────────────

type DiscountCode struct {
	ID             int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Code           string     `gorm:"size:50;uniqueIndex;not null" json:"code"`
	Description    string     `json:"description,omitempty"`
	DiscountType   string     `gorm:"size:20" json:"discount_type"`
	DiscountValue  float64    `gorm:"type:decimal(10,2)" json:"discount_value"`
	AppliesTo      string     `gorm:"size:20;default:'all'" json:"applies_to"`
	MinOrderAmount float64    `gorm:"type:decimal(10,2);default:0" json:"min_order_amount"`
	MaxUses        *int       `json:"max_uses,omitempty"`
	UsesPerUser    int        `gorm:"default:1" json:"uses_per_user"`
	CurrentUses    int        `gorm:"default:0" json:"current_uses"`
	UserGroupID    *int       `json:"user_group_id,omitempty"`
	ValidFrom      *time.Time `json:"valid_from,omitempty"`
	ValidUntil     *time.Time `json:"valid_until,omitempty"`
	IsActive       bool       `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
}

func (DiscountCode) TableName() string { return "discount_codes" }

// ── Section 7: Email Templates ──────────────────────────────────────────────

type EmailTemplate struct {
	ID        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Slug      string    `gorm:"size:100;uniqueIndex" json:"slug"`
	EventType string    `gorm:"size:100;index" json:"event_type"`
	Name      string    `gorm:"size:200" json:"name"`
	Subject   string    `gorm:"size:300" json:"subject"`
	BodyHTML  string    `gorm:"type:text" json:"body_html"`
	BodyText  string    `gorm:"type:text" json:"body_text,omitempty"`
	Variables string    `gorm:"type:jsonb;default:'[]'" json:"variables"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	UpdatedBy string    `gorm:"size:100" json:"updated_by,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (EmailTemplate) TableName() string { return "email_templates" }

// ── Section 7: Static Pages ────────────────────────────────────────────────

type StaticPage struct {
	ID              int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Title           string    `gorm:"size:200" json:"title"`
	Slug            string    `gorm:"size:200;uniqueIndex" json:"slug"`
	Content         string    `gorm:"type:text" json:"content"`
	MetaTitle       string    `gorm:"size:200" json:"meta_title,omitempty"`
	MetaDescription string    `gorm:"type:text" json:"meta_description,omitempty"`
	IsPublished     bool      `gorm:"default:false" json:"is_published"`
	ShowInFooter    bool      `gorm:"default:false" json:"show_in_footer"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (StaticPage) TableName() string { return "static_pages" }

// ── Section 7: Announcements ────────────────────────────────────────────────

type Announcement struct {
	ID              int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Title           string     `gorm:"size:200" json:"title"`
	Content         string     `gorm:"type:text" json:"content"`
	Type            string     `gorm:"size:20;default:'info'" json:"type"`
	DisplayLocation string     `gorm:"size:20;default:'homepage'" json:"display_location"`
	TargetGroupID   *int       `json:"target_group_id,omitempty"`
	StartsAt        *time.Time `json:"starts_at,omitempty"`
	EndsAt          *time.Time `json:"ends_at,omitempty"`
	IsActive        bool       `gorm:"default:true" json:"is_active"`
	CreatedAt       time.Time  `json:"created_at"`
}

func (Announcement) TableName() string { return "announcements" }

// ── Section 8: Geography ────────────────────────────────────────────────────

type GeoRegion struct {
	ID        int         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string      `gorm:"size:100;not null" json:"name"`
	NameAr    string      `gorm:"size:100" json:"name_ar,omitempty"`
	Code      string      `gorm:"size:10" json:"code,omitempty"`
	Type      string      `gorm:"size:20" json:"type"`
	ParentID  *int        `json:"parent_id,omitempty"`
	Latitude  *float64    `gorm:"type:decimal(10,8)" json:"latitude,omitempty"`
	Longitude *float64    `gorm:"type:decimal(11,8)" json:"longitude,omitempty"`
	IsActive  bool        `gorm:"default:true" json:"is_active"`
	SortOrder int         `gorm:"default:0" json:"sort_order"`
	Children  []GeoRegion `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

func (GeoRegion) TableName() string { return "geo_regions" }
