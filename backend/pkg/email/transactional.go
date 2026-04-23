package email

import (
	"context"
	"fmt"
	"time"
)

// ════════════════════════════════════════════════════════════════════════════
// Transactional Email Functions
//
// All functions use the production EmailService pipeline:
//   - HTML template rendering (embedded templates)
//   - Async delivery via SendAsync (non-blocking)
//   - Idempotency (Redis dedup)
//   - Per-user rate limiting
//   - Retry with exponential backoff + circuit breaker
//   - OpenTelemetry tracing
//   - Kafka audit event
// ════════════════════════════════════════════════════════════════════════════

// SendWelcomeEmail sends a welcome email after successful registration.
// Uses the "welcome" HTML template. Async — does not block the request lifecycle.
func SendWelcomeEmail(to, name string) error {
	if name == "" {
		name = "there"
	}
	baseURL := getEnvOr("APP_BASE_URL", "https://geocore.app")
	return Default().SendAsync(context.Background(), &Message{
		To:             to,
		ToName:         name,
		Subject:        "Welcome to GeoCore!",
		TemplateName:   "welcome",
		Data:           WelcomeData(name, baseURL),
		IdempotencyKey: "welcome:" + to,
		CreatedAt:      time.Now(),
	})
}

// SendAuctionWonEmail notifies the winner of an auction.
// Uses the "notification" HTML template. Async — non-blocking.
func SendAuctionWonEmail(to, name, auctionTitle string, amount float64, currency string) error {
	if name == "" {
		name = "there"
	}
	baseURL := getEnvOr("APP_BASE_URL", "https://geocore.app")
	return Default().SendAsync(context.Background(), &Message{
		To:           to,
		ToName:       name,
		Subject:      fmt.Sprintf("You won: %s", auctionTitle),
		TemplateName: "notification",
		Data: NotificationData(
			name,
			"Congratulations — you won the auction!",
			fmt.Sprintf("You won the auction for %s with a bid of %.2f %s. Please complete your purchase within 48 hours.", auctionTitle, amount, currency),
			"View My Bids",
			baseURL+"/my-bids",
		),
		IdempotencyKey: "auction_won:" + auctionTitle + ":" + to,
		CreatedAt:      time.Now(),
	})
}

// SendAuctionEndedSellerEmail notifies the seller when their auction ends.
// Uses the "notification" HTML template. Async — non-blocking.
func SendAuctionEndedSellerEmail(to, name, auctionTitle string, amount float64, currency string, hasWinner bool) error {
	if name == "" {
		name = "there"
	}
	baseURL := getEnvOr("APP_BASE_URL", "https://geocore.app")

	var bodyText string
	if hasWinner {
		bodyText = fmt.Sprintf("Your auction for %s ended successfully with a winning bid of %.2f %s. The buyer will complete the purchase shortly.", auctionTitle, amount, currency)
	} else {
		bodyText = fmt.Sprintf("Your auction for %s ended without a winner. You can relist the item or adjust the reserve price.", auctionTitle)
	}

	return Default().SendAsync(context.Background(), &Message{
		To:           to,
		ToName:       name,
		Subject:      fmt.Sprintf("Auction ended: %s", auctionTitle),
		TemplateName: "notification",
		Data: NotificationData(
			name,
			fmt.Sprintf("Auction %s", map[bool]string{true: "Sold!", false: "Ended"}[hasWinner]),
			bodyText,
			"My Listings",
			baseURL+"/my-listings",
		),
		IdempotencyKey: "auction_ended:" + auctionTitle + ":" + to,
		CreatedAt:      time.Now(),
	})
}

// SendPurchaseConfirmationEmail notifies a buyer of a successful purchase.
// Uses the "transaction_receipt" HTML template. Async — non-blocking.
func SendPurchaseConfirmationEmail(to, name, itemTitle string, amount float64, currency string) error {
	if name == "" {
		name = "there"
	}
	baseURL := getEnvOr("APP_BASE_URL", "https://geocore.app")
	orderID := "latest"
	orderURL := baseURL + "/orders"
	return Default().SendAsync(context.Background(), &Message{
		To:             to,
		ToName:         name,
		Subject:        fmt.Sprintf("Order confirmed: %s", itemTitle),
		TemplateName:   "transaction_receipt",
		Data:           TransactionReceiptData(name, orderID, itemTitle, amount, currency, orderURL),
		IdempotencyKey: "purchase:" + to + ":" + itemTitle,
		CreatedAt:      time.Now(),
	})
}

// SendOutbidEmail notifies a bidder they were outbid.
// Uses the "notification" HTML template. Async — non-blocking.
func SendOutbidEmail(to, name, auctionTitle string, newAmount float64, currency string) error {
	if name == "" {
		name = "there"
	}
	baseURL := getEnvOr("APP_BASE_URL", "https://geocore.app")
	return Default().SendAsync(context.Background(), &Message{
		To:           to,
		ToName:       name,
		Subject:      fmt.Sprintf("You've been outbid on: %s", auctionTitle),
		TemplateName: "notification",
		Data: NotificationData(
			name,
			"You've been outbid",
			fmt.Sprintf("Someone placed a higher bid on %s. The new leading bid is %.2f %s. Bid again to stay in the lead!", auctionTitle, newAmount, currency),
			"Place a Bid",
			baseURL+"/auctions",
		),
		IdempotencyKey: "outbid:" + auctionTitle + ":" + to,
		CreatedAt:      time.Now(),
	})
}

// SendSupportContactEmail notifies support/admin inbox about a new contact form message.
// Uses the "notification" HTML template. Async — non-blocking.
func SendSupportContactEmail(to, senderName, senderEmail, subject, message string) error {
	if senderName == "" {
		senderName = "Unknown"
	}
	if subject == "" {
		subject = "General Inquiry"
	}
	return Default().SendAsync(context.Background(), &Message{
		To:           to,
		Subject:      fmt.Sprintf("[Support] %s", subject),
		TemplateName: "notification",
		Data: NotificationData(
			"Support Team",
			fmt.Sprintf("New Support Message: %s", subject),
			fmt.Sprintf("From: %s <%s>\n\n%s", senderName, senderEmail, message),
			"",
			"",
		),
		IdempotencyKey: "support:" + senderEmail + ":" + subject,
		CreatedAt:      time.Now(),
	})
}
