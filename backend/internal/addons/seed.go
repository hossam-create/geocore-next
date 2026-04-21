package addons

import (
	"log/slog"

	"gorm.io/gorm"
)

// SeedAddons inserts default marketplace addons if the table is empty.
func SeedAddons(db *gorm.DB) {
	var count int64
	db.Model(&Addon{}).Count(&count)
	if count > 0 {
		return
	}

	defaults := []Addon{
		{
			Slug: "stripe-payments", Name: "Stripe Payments Pro",
			Description: "Advanced Stripe payment integration with Apple Pay, Google Pay, and 3D Secure support. Includes webhook management and chargeback handling.",
			Category: "payments", Tags: `["stripe","payments","apple-pay","3d-secure"]`,
			Author: "GeoCore Team", Version: "2.1.0",
			IsFree: false, Price: 29.00, Currency: "AED",
			IsVerified: true, IsOfficial: true,
			Permissions: `["payment.process","payment.refund"]`,
			Hooks:        `["order:onPaymentRequired","subscription:onRenewal"]`,
			Status:       AddonStatusAvailable,
		},
		{
			Slug: "google-analytics", Name: "Google Analytics 4",
			Description: "Full GA4 integration with e-commerce tracking, custom events, and real-time dashboard. Track listings, auctions, and checkout funnels.",
			Category: "analytics", Tags: `["analytics","google","tracking","ecommerce"]`,
			Author: "GeoCore Team", Version: "1.3.0",
			IsFree: true,
			IsVerified: true, IsOfficial: true,
			Permissions: `["analytics.read","analytics.write"]`,
			Hooks:        `["listing:onView","order:onComplete","auction:onBid"]`,
			Status:       AddonStatusAvailable,
		},
		{
			Slug: "mailchimp-integration", Name: "Mailchimp Marketing",
			Description: "Sync customer data to Mailchimp lists. Automated campaigns for abandoned carts, order confirmations, and win-back emails.",
			Category: "marketing", Tags: `["email","mailchimp","marketing","automation"]`,
			Author: "GeoCore Team", Version: "1.0.2",
			IsFree: true,
			IsVerified: true, IsOfficial: false,
			Permissions: `["email.send","user.read"]`,
			Hooks:        `["user:onRegister","order:onAbandon"]`,
			Status:       AddonStatusAvailable,
		},
		{
			Slug: "ar-product-viewer", Name: "AR Product Viewer",
			Description: "Augmented Reality product preview. Buyers can visualize items in their space before purchasing. Supports USDZ and glTF models.",
			Category: "experience", Tags: `["ar","3d","preview","visualization"]`,
			Author: "Spatial Labs", Version: "1.0.0",
			IsFree: false, Price: 49.00, Currency: "AED",
			IsVerified: true, IsOfficial: false,
			Permissions: `["listing.read"]`,
			Hooks:        `["listing:onDetailView"]`,
			Status:       AddonStatusAvailable,
		},
		{
			Slug: "whatsapp-notifications", Name: "WhatsApp Business Notifications",
			Description: "Send order updates, shipping alerts, and promotional messages via WhatsApp Business API. Supports templates and media messages.",
			Category: "notifications", Tags: `["whatsapp","notifications","messaging"]`,
			Author: "MsgBridge", Version: "1.2.0",
			IsFree: false, Price: 19.00, Currency: "AED",
			IsVerified: true, IsOfficial: false,
			Permissions: `["notification.send","user.read"]`,
			Hooks:        `["order:onStatusChange","shipment:onUpdate"]`,
			Status:       AddonStatusAvailable,
		},
		{
			Slug: "loyalty-engine", Name: "Advanced Loyalty Engine",
			Description: "Points, tiers, streaks, and referral rewards. Customizable rules engine for loyalty programs with real-time balance tracking.",
			Category: "engagement", Tags: `["loyalty","rewards","points","tiers","gamification"]`,
			Author: "GeoCore Team", Version: "2.0.0",
			IsFree: false, Price: 39.00, Currency: "AED",
			IsVerified: true, IsOfficial: true,
			Permissions: `["loyalty.manage","user.write"]`,
			Hooks:        `["order:onComplete","review:onSubmit","user:onRegister"]`,
			Status:       AddonStatusAvailable,
		},
		{
			Slug: "seo-optimizer", Name: "SEO Optimizer Pro",
			Description: "Auto-generate meta tags, structured data, sitemaps, and Open Graph images. Includes keyword analysis for listing titles.",
			Category: "marketing", Tags: `["seo","meta","sitemap","structured-data"]`,
			Author: "RankForge", Version: "1.1.0",
			IsFree: true,
			IsVerified: false, IsOfficial: false,
			Permissions: `["listing.read","listing.write"]`,
			Hooks:        `["listing:onCreate","listing:onUpdate"]`,
			Status:       AddonStatusAvailable,
		},
		{
			Slug: "fraud-shield", Name: "Fraud Shield AI",
			Description: "Machine learning-based fraud detection. Analyzes transaction patterns, device fingerprints, and behavioral signals in real-time.",
			Category: "security", Tags: `["fraud","ai","security","detection"]`,
			Author: "GeoCore Team", Version: "1.0.0",
			IsFree: false, Price: 59.00, Currency: "AED",
			IsVerified: true, IsOfficial: true,
			Permissions: `["fraud.detect","transaction.read","user.read"]`,
			Hooks:        `["payment:onBeforeProcess","order:onCreate"]`,
			Status:       AddonStatusAvailable,
		},
	}

	if err := db.Create(&defaults).Error; err != nil {
		slog.Error("failed to seed addons", "error", err.Error())
	} else {
		slog.Info("seeded default marketplace addons", "count", len(defaults))
	}
}
