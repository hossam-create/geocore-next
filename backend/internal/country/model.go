package country

import (
	"time"

	"github.com/google/uuid"
)

// ── Country Configuration ──────────────────────────────────────────────────────
// Each country has its own rules for currency, taxes, KYC limits, payment methods,
// and feature availability. This model drives the Country Layer Service.

// CountryConfig holds all country-specific rules and settings.
type CountryConfig struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Code         string    `gorm:"size:3;uniqueIndex;not null" json:"code"`          // ISO 3166-1 alpha-2 (e.g. "EG", "SA", "AE")
	NameEn       string    `gorm:"size:128;not null" json:"name_en"`
	NameAr       string    `gorm:"size:128" json:"name_ar,omitempty"`
	Currency     string    `gorm:"size:3;not null" json:"currency"`                  // ISO 4217 (e.g. "EGP", "SAR", "AED")
	CurrencyAr   string    `gorm:"size:32" json:"currency_ar,omitempty"`              // "جنيه", "ريال", "درهم"
	CurrencySymbol string  `gorm:"size:8" json:"currency_symbol,omitempty"`           // "E£", "﷼", "د.إ"

	// Tax & Fees
	TaxRate         float64 `gorm:"type:numeric(5,4);default:0" json:"tax_rate"`            // VAT rate (0.14 = 14%)
	TaxLabel        string  `gorm:"size:64;default:'VAT'" json:"tax_label"`                  // "VAT", "ض.ق.م"
	TaxInclusive    bool    `gorm:"default:true" json:"tax_inclusive"`                       // prices include tax?
	ServiceFeeRate  float64 `gorm:"type:numeric(5,4);default:0.05" json:"service_fee_rate"` // platform fee
	WithholdingRate float64 `gorm:"type:numeric(5,4);default:0" json:"withholding_rate"`   // tax withholding on payouts

	// KYC Limits (in local currency cents)
	KYCTier1LimitCents  int64 `gorm:"default:500000" json:"kyc_tier1_limit_cents"`   // max transaction without KYC (e.g. 5000 EGP = 500000)
	KYCTier2LimitCents  int64 `gorm:"default:5000000" json:"kyc_tier2_limit_cents"`  // max with basic KYC
	KYCTier3LimitCents  int64 `gorm:"default:0" json:"kyc_tier3_limit_cents"`        // 0 = unlimited with full KYC
	MaxListingPriceCents int64 `gorm:"default:0" json:"max_listing_price_cents"`    // 0 = no limit

	// Payment Methods available in this country
	PaymentMethods []string `gorm:"type:jsonb;serializer:json;default:'[\"card\",\"cash_on_delivery\"]'" json:"payment_methods"`
	// Possible: card, cash_on_delivery, bank_transfer, wallet, bnpl, crypto, paymob, stripe

	// Feature Flags per country
	EnableAuctions    bool `gorm:"default:true" json:"enable_auctions"`
	EnableLive        bool `gorm:"default:true" json:"enable_live"`
	EnableBNPL        bool `gorm:"default:false" json:"enable_bnpl"`
	EnableP2P         bool `gorm:"default:false" json:"enable_p2p"`
	EnableCrypto      bool `gorm:"default:false" json:"enable_crypto"`
	EnableCrowdship   bool `gorm:"default:false" json:"enable_crowdship"`
	EnableWholesale   bool `gorm:"default:false" json:"enable_wholesale"`
	EnableRealEstate  bool `gorm:"default:false" json:"enable_real_estate"`

	// Shipping
	DefaultShippingCents int64    `gorm:"default:5000" json:"default_shipping_cents"`
	FreeShippingThresholdCents int64 `gorm:"default:100000" json:"free_shipping_threshold_cents"`
	ShippingZones     []string `gorm:"type:jsonb;serializer:json;default:'[]'" json:"shipping_zones"`

	// Locale
	Locale           string `gorm:"size:10;default:'en'" json:"locale"`              // "en", "ar", "ar-EG"
	DateFormat       string `gorm:"size:20;default:'DD/MM/YYYY'" json:"date_format"`
	NumberFormat     string `gorm:"size:20;default:'1,234.56'" json:"number_format"`

	// Regulatory
	RequireNationalID   bool `gorm:"default:false" json:"require_national_id"`
	RequireAddressProof bool `gorm:"default:false" json:"require_address_proof"`
	MinAge              int  `gorm:"default:18" json:"min_age"`
	MaxReturnDays       int  `gorm:"default:14" json:"max_return_days"`

	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (CountryConfig) TableName() string { return "country_configs" }

// ── Country Override (per-user or per-category) ─────────────────────────────────

// CountryOverride allows overriding specific country rules for a user or category.
// For example: a premium seller gets higher listing limits, or a specific category
// has different tax rates in a country.
type CountryOverride struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	CountryCode  string    `gorm:"size:3;not null;index:idx_country_override" json:"country_code"`
	TargetType   string    `gorm:"size:32;not null;index:idx_country_override" json:"target_type"` // "user", "category", "seller_tier"
	TargetID     string    `gorm:"size:128;not null;index:idx_country_override" json:"target_id"`
	Field        string    `gorm:"size:64;not null" json:"field"`                                  // e.g. "kyc_tier1_limit_cents", "tax_rate"
	Value        string    `gorm:"type:text;not null" json:"value"`                                // JSON-encoded value
	Reason       string    `gorm:"size:256" json:"reason,omitempty"`
	CreatedBy    *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (CountryOverride) TableName() string { return "country_overrides" }

// ── Resolved Config (computed result) ───────────────────────────────────────────

// ResolvedConfig is the final computed config for a user in a country,
// after applying all overrides. This is what the API returns.
type ResolvedConfig struct {
	CountryCode              string   `json:"country_code"`
	Currency                 string   `json:"currency"`
	CurrencyAr               string   `json:"currency_ar,omitempty"`
	CurrencySymbol           string   `json:"currency_symbol,omitempty"`
	TaxRate                  float64  `json:"tax_rate"`
	TaxLabel                 string   `json:"tax_label"`
	TaxInclusive             bool     `json:"tax_inclusive"`
	ServiceFeeRate           float64  `json:"service_fee_rate"`
	WithholdingRate          float64  `json:"withholding_rate"`
	KYCTier1LimitCents       int64    `json:"kyc_tier1_limit_cents"`
	KYCTier2LimitCents       int64    `json:"kyc_tier2_limit_cents"`
	KYCTier3LimitCents       int64    `json:"kyc_tier3_limit_cents"`
	MaxListingPriceCents     int64    `json:"max_listing_price_cents"`
	PaymentMethods           []string `json:"payment_methods"`
	EnableAuctions           bool     `json:"enable_auctions"`
	EnableLive               bool     `json:"enable_live"`
	EnableBNPL               bool     `json:"enable_bnpl"`
	EnableP2P                bool     `json:"enable_p2p"`
	EnableCrypto             bool     `json:"enable_crypto"`
	EnableCrowdship          bool     `json:"enable_crowdship"`
	EnableWholesale          bool     `json:"enable_wholesale"`
	EnableRealEstate         bool     `json:"enable_real_estate"`
	DefaultShippingCents     int64    `json:"default_shipping_cents"`
	FreeShippingThresholdCents int64  `json:"free_shipping_threshold_cents"`
	Locale                   string   `json:"locale"`
	RequireNationalID        bool     `json:"require_national_id"`
	RequireAddressProof      bool     `json:"require_address_proof"`
	MinAge                   int      `json:"min_age"`
	MaxReturnDays            int      `json:"max_return_days"`
}
