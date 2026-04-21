package cms

import (
	"log/slog"

	"gorm.io/gorm"
)

// SeedCMS inserts default CMS data if tables are empty.
func SeedCMS(db *gorm.DB) {
	seedSettings(db)
	seedContentBlocks(db)
	seedNavMenus(db)
	seedSlides(db)
}

func seedSettings(db *gorm.DB) {
	var count int64
	db.Model(&SiteSetting{}).Count(&count)
	if count > 0 {
		return
	}

	settings := []SiteSetting{
		// Branding
		{Key: "site_name", Value: "GeoCore", Group: "branding", Label: "Site Name", Type: "text"},
		{Key: "site_logo", Value: "/uploads/branding/logo.svg", Group: "branding", Label: "Logo", Type: "image"},
		{Key: "site_favicon", Value: "/uploads/branding/favicon.ico", Group: "branding", Label: "Favicon", Type: "image"},
		{Key: "primary_color", Value: "#6366f1", Group: "branding", Label: "Primary Color", Type: "color"},
		{Key: "secondary_color", Value: "#8b5cf6", Group: "branding", Label: "Secondary Color", Type: "color"},
		{Key: "accent_color", Value: "#f59e0b", Group: "branding", Label: "Accent Color", Type: "color"},
		{Key: "font_family", Value: "Inter", Group: "branding", Label: "Font Family", Type: "text"},
		// Contact
		{Key: "contact_email", Value: "support@geocore.app", Group: "contact", Label: "Contact Email", Type: "email"},
		{Key: "contact_phone", Value: "+971-4-XXX-XXXX", Group: "contact", Label: "Phone", Type: "text"},
		{Key: "contact_address", Value: "Dubai, UAE", Group: "contact", Label: "Address", Type: "textarea"},
		{Key: "whatsapp_number", Value: "+97150XXXXXXX", Group: "contact", Label: "WhatsApp", Type: "text"},
		// Social
		{Key: "social_instagram", Value: "https://instagram.com/geocore", Group: "social", Label: "Instagram", Type: "url"},
		{Key: "social_twitter", Value: "https://twitter.com/geocore", Group: "social", Label: "Twitter/X", Type: "url"},
		{Key: "social_facebook", Value: "https://facebook.com/geocore", Group: "social", Label: "Facebook", Type: "url"},
		{Key: "social_youtube", Value: "", Group: "social", Label: "YouTube", Type: "url"},
		{Key: "social_tiktok", Value: "", Group: "social", Label: "TikTok", Type: "url"},
		// SEO
		{Key: "seo_title", Value: "GeoCore — Premier Marketplace", Group: "seo", Label: "Default Title", Type: "text"},
		{Key: "seo_description", Value: "Discover unique items, auctions, and deals on GeoCore marketplace.", Group: "seo", Label: "Meta Description", Type: "textarea"},
		{Key: "seo_keywords", Value: "marketplace, auctions, deals, shopping", Group: "seo", Label: "Keywords", Type: "text"},
		{Key: "og_image", Value: "/uploads/branding/og-image.jpg", Group: "seo", Label: "OG Image", Type: "image"},
		// General
		{Key: "currency", Value: "AED", Group: "general", Label: "Default Currency", Type: "text"},
		{Key: "language", Value: "en", Group: "general", Label: "Default Language", Type: "text"},
		{Key: "maintenance_mode", Value: "false", Group: "general", Label: "Maintenance Mode", Type: "boolean"},
		{Key: "footer_text", Value: "© 2025 GeoCore. All rights reserved.", Group: "general", Label: "Footer Text", Type: "textarea"},
		{Key: "cookie_consent_text", Value: "We use cookies to improve your experience.", Group: "general", Label: "Cookie Consent", Type: "textarea"},
	}

	if err := db.Create(&settings).Error; err != nil {
		slog.Error("failed to seed site settings", "error", err.Error())
	} else {
		slog.Info("seeded site settings", "count", len(settings))
	}
}

func seedContentBlocks(db *gorm.DB) {
	var count int64
	db.Model(&ContentBlock{}).Count(&count)
	if count > 0 {
		return
	}

	blocks := []ContentBlock{
		{
			Slug: "homepage_hero", Title: "Homepage Hero", Type: ContentBlockHero,
			Content: "Discover Unique Treasures", Content2: "The premier marketplace for collectors, enthusiasts, and deal seekers.",
			ImageURL: "/uploads/banners/hero-default.jpg",
			LinkURL:  "/listings",
			Page:     "home", Section: "hero", Position: 0, IsActive: true,
		},
		{
			Slug: "homepage_features", Title: "Platform Features", Type: ContentBlockFeatures,
			Content: `{"items":[{"icon":"Shield","title":"Buyer Protection","desc":"Full refund if item not as described"},{"icon":"Zap","title":"Instant Bidding","desc":"Real-time auction experience"},{"icon":"Truck","title":"Fast Shipping","desc":"Tracked delivery to your door"},{"icon":"Star","title":"Verified Sellers","desc":"Trust scores & reviews"}]}`,
			Page:    "home", Section: "features", Position: 1, IsActive: true,
		},
		{
			Slug: "homepage_cta", Title: "Start Selling", Type: ContentBlockCTA,
			Content: "Ready to sell?", Content2: "List your first item in minutes and reach thousands of buyers.",
			LinkURL: "/sell",
			Page:    "home", Section: "cta", Position: 2, IsActive: true,
		},
		{
			Slug: "homepage_testimonials", Title: "What Our Users Say", Type: ContentBlockTestimonial,
			Content: `{"items":[{"name":"Ahmed K.","text":"Found rare coins I couldn't find anywhere else!","rating":5},{"name":"Sara M.","text":"Selling is so easy. Made my first sale within a day.","rating":5}]}`,
			Page:    "home", Section: "testimonials", Position: 3, IsActive: true,
		},
		{
			Slug: "footer_about", Title: "About GeoCore", Type: ContentBlockHTML,
			Content: "<p>GeoCore is the premier marketplace connecting buyers and sellers across the region. Discover unique items, participate in live auctions, and enjoy a secure shopping experience.</p>",
			Page:    "global", Section: "footer", Position: 0, IsActive: true,
		},
	}

	if err := db.Create(&blocks).Error; err != nil {
		slog.Error("failed to seed content blocks", "error", err.Error())
	} else {
		slog.Info("seeded content blocks", "count", len(blocks))
	}
}

func seedNavMenus(db *gorm.DB) {
	var count int64
	db.Model(&NavMenu{}).Count(&count)
	if count > 0 {
		return
	}

	items := []NavMenu{
		{Location: "header", Label: "Home", URL: "/", Position: 0, IsActive: true},
		{Location: "header", Label: "Listings", URL: "/listings", Position: 1, IsActive: true},
		{Location: "header", Label: "Auctions", URL: "/auctions", Position: 2, IsActive: true},
		{Location: "header", Label: "Deals", URL: "/deals", Position: 3, IsActive: true},
		{Location: "header", Label: "Sell", URL: "/sell", Position: 4, IsActive: true},
		{Location: "footer", Label: "About", URL: "/about", Position: 0, IsActive: true},
		{Location: "footer", Label: "Help", URL: "/help", Position: 1, IsActive: true},
		{Location: "footer", Label: "Buyer Protection", URL: "/buyer-protection", Position: 2, IsActive: true},
		{Location: "footer", Label: "Seller Protection", URL: "/seller-protection", Position: 3, IsActive: true},
		{Location: "footer", Label: "Fees", URL: "/fees", Position: 4, IsActive: true},
		{Location: "footer", Label: "Contact", URL: "/support/contact", Position: 5, IsActive: true},
	}

	if err := db.Create(&items).Error; err != nil {
		slog.Error("failed to seed nav menus", "error", err.Error())
	} else {
		slog.Info("seeded nav menus", "count", len(items))
	}
}

func seedSlides(db *gorm.DB) {
	var count int64
	db.Model(&HeroSlide{}).Count(&count)
	if count > 0 {
		return
	}

	slides := []HeroSlide{
		{
			Title: "Summer Collection 2025", Subtitle: "Up to 40% off on selected items",
			ImageURL: "/uploads/banners/summer-sale.jpg", LinkURL: "/listings?tag=summer",
			LinkLabel: "Shop Now", Badge: "SALE", Position: 0, IsActive: true,
		},
		{
			Title: "Live Auctions", Subtitle: "Bid on rare and exclusive items in real-time",
			ImageURL: "/uploads/banners/auctions.jpg", LinkURL: "/auctions",
			LinkLabel: "View Auctions", Badge: "LIVE", Position: 1, IsActive: true,
		},
		{
			Title: "Start Selling Today", Subtitle: "List your items and reach thousands of buyers",
			ImageURL: "/uploads/banners/sell-now.jpg", LinkURL: "/sell",
			LinkLabel: "Get Started", Position: 2, IsActive: true,
		},
	}

	if err := db.Create(&slides).Error; err != nil {
		slog.Error("failed to seed hero slides", "error", err.Error())
	} else {
		slog.Info("seeded hero slides", "count", len(slides))
	}
}
