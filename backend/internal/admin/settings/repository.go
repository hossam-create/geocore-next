package settings

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ── Settings ────────────────────────────────────────────────────────────────

func (r *Repository) GetAllSettings() ([]AdminSetting, error) {
	var s []AdminSetting
	err := r.db.Order("category ASC, key ASC").Find(&s).Error
	return s, err
}

func (r *Repository) GetSettingsByCategory(cat string) ([]AdminSetting, error) {
	var s []AdminSetting
	err := r.db.Where("category = ?", cat).Order("key ASC").Find(&s).Error
	return s, err
}

func (r *Repository) GetPublicSettings() ([]AdminSetting, error) {
	var s []AdminSetting
	err := r.db.Where("is_public = ?", true).Find(&s).Error
	return s, err
}

func (r *Repository) GetSetting(key string) (*AdminSetting, error) {
	var s AdminSetting
	err := r.db.Where("key = ?", key).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) UpdateSetting(key, value string, adminID uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&AdminSetting{}).Where("key = ?", key).
		Updates(map[string]interface{}{
			"value":      value,
			"updated_at": now,
			"updated_by": adminID,
		}).Error
}

func (r *Repository) BulkUpdateSettings(updates map[string]string, adminID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		for key, val := range updates {
			if err := tx.Model(&AdminSetting{}).Where("key = ?", key).
				Updates(map[string]interface{}{
					"value":      val,
					"updated_at": now,
					"updated_by": adminID,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GroupedSettings returns settings organized by category.
func (r *Repository) GroupedSettings() ([]CategoryGroup, error) {
	all, err := r.GetAllSettings()
	if err != nil {
		return nil, err
	}
	catMap := make(map[string][]AdminSetting)
	var order []string
	for _, s := range all {
		if _, exists := catMap[s.Category]; !exists {
			order = append(order, s.Category)
		}
		catMap[s.Category] = append(catMap[s.Category], s)
	}
	groups := make([]CategoryGroup, 0, len(order))
	for _, cat := range order {
		groups = append(groups, CategoryGroup{Category: cat, Settings: catMap[cat]})
	}
	return groups, nil
}

// ── Feature Flags ───────────────────────────────────────────────────────────

func (r *Repository) GetAllFlags() ([]FeatureFlag, error) {
	var f []FeatureFlag
	err := r.db.Order("key ASC").Find(&f).Error
	return f, err
}

func (r *Repository) GetFlag(key string) (*FeatureFlag, error) {
	var f FeatureFlag
	err := r.db.Where("key = ?", key).First(&f).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *Repository) UpdateFlag(key string, enabled bool, rolloutPct int, allowedGroups []string) error {
	return r.db.Model(&FeatureFlag{}).Where("key = ?", key).
		Updates(map[string]interface{}{
			"enabled":        enabled,
			"rollout_pct":    rolloutPct,
			"allowed_groups": allowedGroups,
		}).Error
}

// GetPublicFlags returns flags relevant to a user given their group.
// If rollout_pct < 100, uses a simple hash to decide inclusion.
func (r *Repository) GetPublicFlags(userID string, userGroup string) (map[string]bool, error) {
	flags, err := r.GetAllFlags()
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(flags))
	for _, f := range flags {
		if !f.Enabled {
			result[f.Key] = false
			continue
		}
		// Check group restriction
		if len(f.AllowedGroups) > 0 && userGroup != "" {
			allowed := false
			for _, g := range f.AllowedGroups {
				if strings.EqualFold(g, userGroup) {
					allowed = true
					break
				}
			}
			if !allowed {
				result[f.Key] = false
				continue
			}
		}
		// Check rollout percentage
		if f.RolloutPct < 100 && userID != "" {
			hash := simpleHash(userID + f.Key)
			if hash%100 >= uint32(f.RolloutPct) {
				result[f.Key] = false
				continue
			}
		}
		result[f.Key] = true
	}
	return result, nil
}

func simpleHash(s string) uint32 {
	var h uint32
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return h
}

// ── Seed ────────────────────────────────────────────────────────────────────

func (r *Repository) SeedDefaults() {
	settingSeeds := []AdminSetting{
		// Site
		{Key: "site.name", Value: `"GeoCore Next"`, Type: "string", Category: "site", Label: "Site Name", IsPublic: true},
		{Key: "site.tagline", Value: `""`, Type: "string", Category: "site", Label: "Tagline", IsPublic: true},
		{Key: "site.logo_url", Value: `""`, Type: "string", Category: "site", Label: "Logo URL", IsPublic: true},
		{Key: "site.favicon_url", Value: `""`, Type: "string", Category: "site", Label: "Favicon URL", IsPublic: true},
		{Key: "site.maintenance_mode", Value: "false", Type: "bool", Category: "site", Label: "Maintenance Mode", Description: "Puts the entire site into maintenance mode"},
		{Key: "site.on", Value: "true", Type: "bool", Category: "site", Label: "Site Enabled", Description: "Master kill switch", IsPublic: true},
		{Key: "site.default_currency", Value: `"USD"`, Type: "select", Category: "site", Label: "Default Currency", IsPublic: true, Options: strPtr(`[{"value":"USD","label":"US Dollar"},{"value":"AED","label":"UAE Dirham"},{"value":"EUR","label":"Euro"},{"value":"GBP","label":"British Pound"}]`)},
		{Key: "site.default_language", Value: `"en"`, Type: "select", Category: "site", Label: "Default Language", IsPublic: true, Options: strPtr(`[{"value":"en","label":"English"},{"value":"ar","label":"Arabic"}]`)},
		{Key: "site.contact_email", Value: `""`, Type: "string", Category: "site", Label: "Contact Email"},
		{Key: "site.google_analytics", Value: `""`, Type: "string", Category: "site", Label: "Google Analytics ID", IsPublic: true},

		// Listings
		{Key: "listings.require_approval", Value: "false", Type: "bool", Category: "listings", Label: "Require Approval", Description: "New listings must be approved by admin"},
		{Key: "listings.max_images", Value: "10", Type: "number", Category: "listings", Label: "Max Images Per Listing"},
		{Key: "listings.max_duration_days", Value: "60", Type: "number", Category: "listings", Label: "Max Listing Duration (days)"},
		{Key: "listings.allow_anonymous", Value: "false", Type: "bool", Category: "listings", Label: "Allow Anonymous Listings"},
		{Key: "listings.profanity_filter", Value: "true", Type: "bool", Category: "listings", Label: "Profanity Filter", Description: "Automatically filter profanity from listing titles and descriptions"},
		{Key: "listings.banned_keywords", Value: "[]", Type: "json", Category: "listings", Label: "Banned Keywords", Description: "JSON array of keywords that are blocked from listings"},

		// Auctions
		{Key: "auctions.proxy_bidding", Value: "true", Type: "bool", Category: "auctions", Label: "Proxy Bidding", IsPublic: true},
		{Key: "auctions.min_bid_increment", Value: "1.0", Type: "number", Category: "auctions", Label: "Min Bid Increment", IsPublic: true},
		{Key: "auctions.extension_minutes", Value: "5", Type: "number", Category: "auctions", Label: "Anti-Snipe Extension (min)", IsPublic: true},
		{Key: "auctions.snipe_protection", Value: "true", Type: "bool", Category: "auctions", Label: "Snipe Protection", IsPublic: true, Description: "Automatically extend auctions when bids are placed in final seconds"},

		// Search Ranking
		{Key: "search.featured_boost", Value: "2.0", Type: "number", Category: "search", Label: "Featured Boost Multiplier", Description: "Score multiplier for featured/promoted listings"},
		{Key: "search.recency_weight", Value: "0.3", Type: "number", Category: "search", Label: "Recency Weight", Description: "Weight for listing freshness in ranking (0-1)"},
		{Key: "search.review_weight", Value: "0.4", Type: "number", Category: "search", Label: "Review Score Weight", Description: "Weight for seller review score in ranking (0-1)"},
		{Key: "search.sales_weight", Value: "0.3", Type: "number", Category: "search", Label: "Sales Volume Weight", Description: "Weight for historical sales volume in ranking (0-1)"},

		// Payments
		{Key: "payments.stripe_enabled", Value: "false", Type: "bool", Category: "payments", Label: "Enable Stripe"},
		{Key: "payments.stripe_pk", Value: `""`, Type: "string", Category: "payments", Label: "Stripe Publishable Key", IsPublic: true},
		{Key: "payments.stripe_sk", Value: `""`, Type: "secret", Category: "payments", Label: "Stripe Secret Key", IsSecret: true, Description: "Never exposed publicly"},
		{Key: "payments.stripe_webhook_secret", Value: `""`, Type: "secret", Category: "payments", Label: "Stripe Webhook Secret", IsSecret: true},
		{Key: "payments.paymob_enabled", Value: "false", Type: "bool", Category: "payments", Label: "Enable PayMob"},
		{Key: "payments.paymob_api_key", Value: `""`, Type: "secret", Category: "payments", Label: "PayMob API Key", IsSecret: true},
		{Key: "payments.paypal_enabled", Value: "false", Type: "bool", Category: "payments", Label: "Enable PayPal"},
		{Key: "payments.paypal_client_id", Value: `""`, Type: "string", Category: "payments", Label: "PayPal Client ID", IsPublic: true},
		{Key: "payments.paypal_client_secret", Value: `""`, Type: "secret", Category: "payments", Label: "PayPal Client Secret", IsSecret: true},
		{Key: "payments.platform_fee_pct", Value: "5.0", Type: "number", Category: "payments", Label: "Platform Fee %", IsPublic: true, Description: "Percentage charged on each sale"},
		{Key: "payments.payout_schedule", Value: `"weekly"`, Type: "select", Category: "payments", Label: "Payout Schedule", Options: strPtr(`[{"value":"daily","label":"Daily"},{"value":"weekly","label":"Weekly"},{"value":"biweekly","label":"Bi-Weekly"},{"value":"monthly","label":"Monthly"}]`)},

		// Pricing
		{Key: "pricing.featured_listing_cost", Value: "2.99", Type: "number", Category: "pricing", Label: "Featured Listing Cost", IsPublic: true},
		{Key: "pricing.success_fee_default", Value: "5.0", Type: "number", Category: "pricing", Label: "Success Fee — Default %", IsPublic: true},
		{Key: "pricing.success_fee_vehicles", Value: "2.5", Type: "number", Category: "pricing", Label: "Success Fee — Vehicles %", IsPublic: true},
		{Key: "pricing.success_fee_real_estate", Value: "1.0", Type: "number", Category: "pricing", Label: "Success Fee — Real Estate %", IsPublic: true},

		// Shipping
		{Key: "shipping.dhl_enabled", Value: "false", Type: "bool", Category: "shipping", Label: "Enable DHL"},
		{Key: "shipping.dhl_api_key", Value: `""`, Type: "secret", Category: "shipping", Label: "DHL API Key", IsSecret: true},
		{Key: "shipping.free_shipping_min", Value: "0", Type: "number", Category: "shipping", Label: "Free Shipping Minimum", IsPublic: true, Description: "Minimum order amount for free shipping (0 = disabled)"},

		// Email
		{Key: "email.provider", Value: `"smtp"`, Type: "select", Category: "email", Label: "Email Provider", Options: strPtr(`[{"value":"smtp","label":"SMTP"},{"value":"resend","label":"Resend"},{"value":"sendgrid","label":"SendGrid"}]`)},
		{Key: "email.from_address", Value: `""`, Type: "string", Category: "email", Label: "From Address"},
		{Key: "email.resend_api_key", Value: `""`, Type: "secret", Category: "email", Label: "Resend API Key", IsSecret: true},
		{Key: "email.sendgrid_api_key", Value: `""`, Type: "secret", Category: "email", Label: "SendGrid API Key", IsSecret: true},
		{Key: "email.smtp_host", Value: `""`, Type: "string", Category: "email", Label: "SMTP Host"},
		{Key: "email.smtp_port", Value: "587", Type: "number", Category: "email", Label: "SMTP Port"},
		{Key: "email.smtp_user", Value: `""`, Type: "string", Category: "email", Label: "SMTP Username"},
		{Key: "email.smtp_pass", Value: `""`, Type: "secret", Category: "email", Label: "SMTP Password", IsSecret: true},

		// OAuth / Social Login
		{Key: "oauth.google_enabled", Value: "false", Type: "bool", Category: "oauth", Label: "Enable Google Login"},
		{Key: "oauth.google_client_id", Value: `""`, Type: "string", Category: "oauth", Label: "Google Client ID", IsPublic: true},
		{Key: "oauth.google_client_secret", Value: `""`, Type: "secret", Category: "oauth", Label: "Google Client Secret", IsSecret: true},
		{Key: "oauth.apple_enabled", Value: "false", Type: "bool", Category: "oauth", Label: "Enable Apple Login"},
		{Key: "oauth.apple_client_id", Value: `""`, Type: "string", Category: "oauth", Label: "Apple Client ID (Service ID)", IsPublic: true},
		{Key: "oauth.apple_team_id", Value: `""`, Type: "string", Category: "oauth", Label: "Apple Team ID"},
		{Key: "oauth.apple_key_id", Value: `""`, Type: "string", Category: "oauth", Label: "Apple Key ID"},
		{Key: "oauth.apple_private_key", Value: `""`, Type: "secret", Category: "oauth", Label: "Apple Private Key (PEM)", IsSecret: true},
		{Key: "oauth.facebook_enabled", Value: "false", Type: "bool", Category: "oauth", Label: "Enable Facebook Login"},
		{Key: "oauth.facebook_app_id", Value: `""`, Type: "string", Category: "oauth", Label: "Facebook App ID", IsPublic: true},
		{Key: "oauth.facebook_app_secret", Value: `""`, Type: "secret", Category: "oauth", Label: "Facebook App Secret", IsSecret: true},

		// Storage
		{Key: "storage.provider", Value: `"local"`, Type: "select", Category: "storage", Label: "Storage Provider", Options: strPtr(`[{"value":"local","label":"Local"},{"value":"r2","label":"Cloudflare R2"},{"value":"s3","label":"AWS S3"}]`)},
		{Key: "storage.r2_account_id", Value: `""`, Type: "string", Category: "storage", Label: "R2 Account ID"},
		{Key: "storage.r2_access_key", Value: `""`, Type: "secret", Category: "storage", Label: "R2 Access Key", IsSecret: true},
		{Key: "storage.r2_secret_key", Value: `""`, Type: "secret", Category: "storage", Label: "R2 Secret Key", IsSecret: true},
		{Key: "storage.r2_bucket", Value: `""`, Type: "string", Category: "storage", Label: "R2 Bucket Name"},
		{Key: "storage.r2_public_url", Value: `""`, Type: "string", Category: "storage", Label: "R2 Public URL", IsPublic: true},
		{Key: "storage.s3_region", Value: `"us-east-1"`, Type: "string", Category: "storage", Label: "AWS S3 Region"},
		{Key: "storage.s3_bucket", Value: `""`, Type: "string", Category: "storage", Label: "AWS S3 Bucket"},
		{Key: "storage.s3_access_key", Value: `""`, Type: "secret", Category: "storage", Label: "AWS Access Key ID", IsSecret: true},
		{Key: "storage.s3_secret_key", Value: `""`, Type: "secret", Category: "storage", Label: "AWS Secret Access Key", IsSecret: true},

		// Maps
		{Key: "maps.provider", Value: `"google"`, Type: "select", Category: "maps", Label: "Maps Provider", Options: strPtr(`[{"value":"google","label":"Google Maps"},{"value":"mapbox","label":"Mapbox"}]`), IsPublic: true},
		{Key: "maps.google_api_key", Value: `""`, Type: "string", Category: "maps", Label: "Google Maps API Key", IsPublic: true},
		{Key: "maps.mapbox_token", Value: `""`, Type: "string", Category: "maps", Label: "Mapbox Public Token", IsPublic: true},

		// SMS
		{Key: "sms.provider", Value: `"twilio"`, Type: "select", Category: "sms", Label: "SMS Provider", Options: strPtr(`[{"value":"twilio","label":"Twilio"},{"value":"vonage","label":"Vonage"}]`)},
		{Key: "sms.twilio_account_sid", Value: `""`, Type: "string", Category: "sms", Label: "Twilio Account SID"},
		{Key: "sms.twilio_auth_token", Value: `""`, Type: "secret", Category: "sms", Label: "Twilio Auth Token", IsSecret: true},
		{Key: "sms.twilio_from_number", Value: `""`, Type: "string", Category: "sms", Label: "Twilio From Number"},
		{Key: "sms.vonage_api_key", Value: `""`, Type: "secret", Category: "sms", Label: "Vonage API Key", IsSecret: true},
		{Key: "sms.vonage_api_secret", Value: `""`, Type: "secret", Category: "sms", Label: "Vonage API Secret", IsSecret: true},

		// Push Notifications
		{Key: "push.fcm_server_key", Value: `""`, Type: "secret", Category: "push", Label: "Firebase FCM Server Key", IsSecret: true},
		{Key: "push.fcm_project_id", Value: `""`, Type: "string", Category: "push", Label: "Firebase Project ID"},
		{Key: "push.apns_key_id", Value: `""`, Type: "string", Category: "push", Label: "APNs Key ID"},
		{Key: "push.apns_team_id", Value: `""`, Type: "string", Category: "push", Label: "APNs Team ID"},
		{Key: "push.apns_private_key", Value: `""`, Type: "secret", Category: "push", Label: "APNs Private Key (PEM)", IsSecret: true},

		// Additional Payments
		{Key: "payments.tabby_enabled", Value: "false", Type: "bool", Category: "payments", Label: "Enable Tabby (BNPL)"},
		{Key: "payments.tabby_public_key", Value: `""`, Type: "string", Category: "payments", Label: "Tabby Public Key", IsPublic: true},
		{Key: "payments.tabby_secret_key", Value: `""`, Type: "secret", Category: "payments", Label: "Tabby Secret Key", IsSecret: true},
		{Key: "payments.tamara_enabled", Value: "false", Type: "bool", Category: "payments", Label: "Enable Tamara (BNPL)"},
		{Key: "payments.tamara_token", Value: `""`, Type: "secret", Category: "payments", Label: "Tamara API Token", IsSecret: true},
		{Key: "payments.tamara_notification_key", Value: `""`, Type: "secret", Category: "payments", Label: "Tamara Notification Key", IsSecret: true},
		{Key: "payments.crypto_wallet_address", Value: `""`, Type: "string", Category: "payments", Label: "Crypto Payout Wallet Address"},
		{Key: "payments.financial_mode_paused", Value: "false", Type: "bool", Category: "payments", Label: "Financial Kill Switch", Description: "Halt ALL money movement (deposits, withdrawals, transfers, escrow). Emergency use only."},

		// Trust & Safety
		{Key: "trust.auto_ban_threshold", Value: "5", Type: "number", Category: "trust", Label: "Auto-Ban Threshold", Description: "Number of confirmed flags before automatic ban"},
		{Key: "trust.ml_fraud_enabled", Value: "false", Type: "bool", Category: "trust", Label: "ML Fraud Detection", Description: "Enable machine-learning based fraud scoring"},
		{Key: "trust.require_phone_verify", Value: "false", Type: "bool", Category: "trust", Label: "Require Phone Verification"},
		{Key: "trust.require_id_verify", Value: "false", Type: "bool", Category: "trust", Label: "Require ID Verification"},
		{Key: "trust.max_reports_before_review", Value: "3", Type: "number", Category: "trust", Label: "Max Reports Before Review", Description: "Number of user reports that trigger manual review"},

		// AWS
		{Key: "aws.cloudwatch_enabled", Value: "false", Type: "bool", Category: "aws", Label: "Enable CloudWatch"},
		{Key: "aws.s3_bucket", Value: `""`, Type: "string", Category: "aws", Label: "S3 Bucket Name"},
		{Key: "aws.s3_access_key", Value: `""`, Type: "secret", Category: "aws", Label: "AWS Access Key ID", IsSecret: true},
		{Key: "aws.s3_secret_key", Value: `""`, Type: "secret", Category: "aws", Label: "AWS Secret Access Key", IsSecret: true},
		{Key: "aws.cloudfront_url", Value: `""`, Type: "string", Category: "aws", Label: "CloudFront Distribution URL", IsPublic: true},
	}

	for _, s := range settingSeeds {
		r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&s)
	}

	flagSeeds := []FeatureFlag{
		// Commerce
		{Key: "feature.dutch_auction", Description: "Enable Dutch (descending price) auctions", Category: "commerce"},
		{Key: "feature.reverse_auction", Description: "Enable reverse auctions (buyer sets price, sellers compete)", Category: "commerce"},
		{Key: "feature.storefronts", Description: "Enable seller storefronts", Category: "commerce"},
		{Key: "feature.wallet", Description: "Enable in-platform wallet", Category: "commerce"},
		// Growth
		{Key: "feature.loyalty_program", Description: "Enable loyalty points system", Category: "growth"},
		{Key: "feature.referral_program", Description: "Enable referral / affiliate program", Category: "growth"},
		{Key: "feature.deals_promotions", Description: "Enable deals & promotions engine", Category: "growth"},
		{Key: "feature.subscription_plans", Description: "Enable subscription / membership plans", Category: "growth"},
		// Auctions
		{Key: "feature.live_streaming", Description: "Enable live streaming auctions", Category: "auctions"},
		// Payments
		{Key: "feature.paypal", Description: "Enable PayPal payment gateway", Category: "payments"},
		{Key: "feature.crypto_payments", Description: "Enable cryptocurrency payments", Category: "payments"},
		{Key: "feature.bnpl", Description: "Enable Buy Now Pay Later integration", Category: "payments"},
		// Future / Experimental
		{Key: "feature.ai_chatbot", Description: "Enable AI chatbot assistant", Category: "future"},
		{Key: "feature.ai_fraud_detection", Description: "Enable AI-powered fraud detection", Category: "future"},
		{Key: "feature.ai_pricing", Description: "Enable AI-powered dynamic pricing suggestions", Category: "future"},
		{Key: "feature.ar_preview", Description: "Enable AR product preview", Category: "future"},
		{Key: "feature.crowdshipping", Description: "Enable crowd-shipping / traveler delivery", Category: "future"},
		{Key: "feature.blockchain", Description: "Enable blockchain escrow", Category: "future"},
		{Key: "feature.plugin_marketplace", Description: "Enable third-party plugin marketplace", Category: "future"},
		{Key: "feature.p2p_exchange", Description: "Enable peer-to-peer currency exchange", Category: "future"},
	}

	for _, f := range flagSeeds {
		r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&f)
	}
}

func strPtr(s string) *string { return &s }
