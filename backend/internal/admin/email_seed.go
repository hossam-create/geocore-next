package admin

import (
	"log/slog"

	"gorm.io/gorm"
)

// SeedEmailTemplates inserts default email templates if the table is empty.
func SeedEmailTemplates(db *gorm.DB) {
	var count int64
	db.Model(&EmailTemplate{}).Count(&count)
	if count > 0 {
		return
	}

	defaults := []EmailTemplate{
		{
			Slug:      "welcome",
			EventType: "user.registered",
			Name:      "Welcome Email",
			Subject:   "Welcome to GeoCore, {{name}}!",
			BodyHTML:  "<h1>Welcome, {{name}}!</h1><p>Thanks for joining GeoCore. Start exploring listings <a href='{{app_url}}'>here</a>.</p>",
			BodyText:  "Welcome, {{name}}! Thanks for joining GeoCore. Start exploring at {{app_url}}",
			Variables: `["name","app_url"]`,
			IsActive:  true,
		},
		{
			Slug:      "order_confirmed",
			EventType: "order.confirmed",
			Name:      "Order Confirmed",
			Subject:   "Order #{{order_id}} Confirmed",
			BodyHTML:  "<h1>Order Confirmed</h1><p>Your order <strong>#{{order_id}}</strong> has been confirmed.</p><p>Total: {{currency}} {{amount}}</p>",
			BodyText:  "Order #{{order_id}} confirmed. Total: {{currency}} {{amount}}",
			Variables: `["order_id","currency","amount"]`,
			IsActive:  true,
		},
		{
			Slug:      "order_shipped",
			EventType: "order.shipped",
			Name:      "Order Shipped",
			Subject:   "Order #{{order_id}} Shipped",
			BodyHTML:  "<h1>Order Shipped</h1><p>Your order <strong>#{{order_id}}</strong> is on its way!</p><p>Tracking: {{tracking_number}}</p>",
			BodyText:  "Order #{{order_id}} shipped. Tracking: {{tracking_number}}",
			Variables: `["order_id","tracking_number"]`,
			IsActive:  true,
		},
		{
			Slug:      "password_reset",
			EventType: "user.password_reset",
			Name:      "Password Reset",
			Subject:   "Reset your GeoCore password",
			BodyHTML:  "<h1>Password Reset</h1><p>Click <a href='{{reset_url}}'>here</a> to reset your password. This link expires in 1 hour.</p>",
			BodyText:  "Reset your password: {{reset_url}} (expires in 1 hour)",
			Variables: `["reset_url"]`,
			IsActive:  true,
		},
		{
			Slug:      "listing_approved",
			EventType: "listing.approved",
			Name:      "Listing Approved",
			Subject:   "Your listing '{{title}}' is now live!",
			BodyHTML:  "<h1>Listing Approved</h1><p>Your listing <strong>{{title}}</strong> has been approved and is now visible to buyers.</p>",
			BodyText:  "Your listing '{{title}}' is now live!",
			Variables: `["title"]`,
			IsActive:  true,
		},
		{
			Slug:      "listing_rejected",
			EventType: "listing.rejected",
			Name:      "Listing Rejected",
			Subject:   "Your listing '{{title}}' needs revision",
			BodyHTML:  "<h1>Listing Needs Revision</h1><p>Your listing <strong>{{title}}</strong> was not approved.</p><p>Reason: {{reason}}</p>",
			BodyText:  "Your listing '{{title}}' needs revision. Reason: {{reason}}",
			Variables: `["title","reason"]`,
			IsActive:  true,
		},
		{
			Slug:      "auction_won",
			EventType: "auction.won",
			Name:      "Auction Won",
			Subject:   "You won the auction for '{{title}}'!",
			BodyHTML:  "<h1>Congratulations!</h1><p>You won the auction for <strong>{{title}}</strong> at {{currency}} {{amount}}.</p><p>Complete your purchase <a href='{{checkout_url}}'>here</a>.</p>",
			BodyText:  "You won the auction for '{{title}}' at {{currency}} {{amount}}. Checkout: {{checkout_url}}",
			Variables: `["title","currency","amount","checkout_url"]`,
			IsActive:  true,
		},
		{
			Slug:      "payout_completed",
			EventType: "payout.completed",
			Name:      "Payout Completed",
			Subject:   "Payout of {{currency}} {{amount}} completed",
			BodyHTML:  "<h1>Payout Completed</h1><p>Your payout of <strong>{{currency}} {{amount}}</strong> has been processed to {{destination}}.</p>",
			BodyText:  "Payout of {{currency}} {{amount}} completed to {{destination}}.",
			Variables: `["currency","amount","destination"]`,
			IsActive:  true,
		},
		{
			Slug:      "escrow_released",
			EventType: "escrow.released",
			Name:      "Escrow Released",
			Subject:   "Funds released for order #{{order_id}}",
			BodyHTML:  "<h1>Funds Released</h1><p>The escrow funds for order <strong>#{{order_id}}</strong> have been released to the seller.</p><p>Amount: {{currency}} {{amount}}</p>",
			BodyText:  "Funds released for order #{{order_id}}. Amount: {{currency}} {{amount}}",
			Variables: `["order_id","currency","amount"]`,
			IsActive:  true,
		},
		{
			Slug:      "refund_processed",
			EventType: "refund.processed",
			Name:      "Refund Processed",
			Subject:   "Refund of {{currency}} {{amount}} processed",
			BodyHTML:  "<h1>Refund Processed</h1><p>Your refund of <strong>{{currency}} {{amount}}</strong> for order #{{order_id}} has been processed.</p>",
			BodyText:  "Refund of {{currency}} {{amount}} for order #{{order_id}} processed.",
			Variables: `["currency","amount","order_id"]`,
			IsActive:  true,
		},
	}

	if err := db.Create(&defaults).Error; err != nil {
		slog.Error("failed to seed email templates", "error", err.Error())
	} else {
		slog.Info("seeded default email templates", "count", len(defaults))
	}
}
