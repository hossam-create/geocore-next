package email

import "fmt"

// SendWelcomeEmail sends a welcome email after successful registration.
func SendWelcomeEmail(to, name string) error {
	cfg := loadSMTP()
	if name == "" {
		name = "there"
	}

	body := fmt.Sprintf(
		"Hi %s,\n\n"+
			"Welcome to GeoCore — the GCC's premier marketplace! 🎉\n\n"+
			"You can now:\n"+
			"  • Browse and post listings across the region\n"+
			"  • Bid in live auctions\n"+
			"  • Message sellers and buyers directly\n"+
			"  • Create your own Storefront\n\n"+
			"Get started at: %s\n\n"+
			"If you have any questions, reply to this email or visit our Help Center.\n\n"+
			"— The GeoCore Team",
		name, cfg.BaseURL,
	)

	if cfg.Host == "" || cfg.From == "" {
		fmt.Printf("[email-dev] Welcome email sent to %s (%s)\n", to, name)
		return nil
	}
	return send(cfg, to, "Welcome to GeoCore! 🎉", body)
}

// SendAuctionWonEmail notifies the winner of an auction.
func SendAuctionWonEmail(to, name, auctionTitle string, amount float64, currency string) error {
	cfg := loadSMTP()
	if name == "" {
		name = "there"
	}

	body := fmt.Sprintf(
		"Hi %s,\n\n"+
			"🏆 Congratulations — you won the auction!\n\n"+
			"  Item: %s\n"+
			"  Winning Bid: %.2f %s\n\n"+
			"Please complete your purchase within 48 hours:\n"+
			"  %s/my-bids\n\n"+
			"If you have any issues, contact the seller through GeoCore messaging.\n\n"+
			"— The GeoCore Team",
		name, auctionTitle, amount, currency, cfg.BaseURL,
	)

	if cfg.Host == "" || cfg.From == "" {
		fmt.Printf("[email-dev] Auction won email sent to %s for '%s'\n", to, auctionTitle)
		return nil
	}
	return send(cfg, to, fmt.Sprintf("You won: %s", auctionTitle), body)
}

// SendAuctionEndedSellerEmail notifies the seller when their auction ends.
func SendAuctionEndedSellerEmail(to, name, auctionTitle string, amount float64, currency string, hasWinner bool) error {
	cfg := loadSMTP()
	if name == "" {
		name = "there"
	}

	var body string
	if hasWinner {
		body = fmt.Sprintf(
			"Hi %s,\n\n"+
				"Your auction has ended successfully!\n\n"+
				"  Item: %s\n"+
				"  Winning Bid: %.2f %s\n\n"+
				"The buyer will complete the purchase shortly. You can manage your auctions at:\n"+
				"  %s/my-listings\n\n"+
				"— The GeoCore Team",
			name, auctionTitle, amount, currency, cfg.BaseURL,
		)
	} else {
		body = fmt.Sprintf(
			"Hi %s,\n\n"+
				"Your auction for '%s' ended without a winner.\n\n"+
				"You can relist the item or reduce the reserve price at:\n"+
				"  %s/my-listings\n\n"+
				"— The GeoCore Team",
			name, auctionTitle, cfg.BaseURL,
		)
	}

	if cfg.Host == "" || cfg.From == "" {
		fmt.Printf("[email-dev] Auction ended email sent to seller %s for '%s'\n", to, auctionTitle)
		return nil
	}
	return send(cfg, to, fmt.Sprintf("Auction ended: %s", auctionTitle), body)
}

// SendPurchaseConfirmationEmail notifies a buyer of a successful purchase.
func SendPurchaseConfirmationEmail(to, name, itemTitle string, amount float64, currency string) error {
	cfg := loadSMTP()
	if name == "" {
		name = "there"
	}

	body := fmt.Sprintf(
		"Hi %s,\n\n"+
			"✅ Payment confirmed — your purchase is complete!\n\n"+
			"  Item: %s\n"+
			"  Amount: %.2f %s\n\n"+
			"The seller has been notified and will arrange delivery. You can track your orders at:\n"+
			"  %s/orders\n\n"+
			"Need help? Contact support@geocore.app\n\n"+
			"— The GeoCore Team",
		name, itemTitle, amount, currency, cfg.BaseURL,
	)

	if cfg.Host == "" || cfg.From == "" {
		fmt.Printf("[email-dev] Purchase confirmation sent to %s for '%s'\n", to, itemTitle)
		return nil
	}
	return send(cfg, to, fmt.Sprintf("Order confirmed: %s", itemTitle), body)
}

// SendOutbidEmail notifies a bidder they were outbid.
func SendOutbidEmail(to, name, auctionTitle string, newAmount float64, currency string) error {
	cfg := loadSMTP()
	if name == "" {
		name = "there"
	}

	body := fmt.Sprintf(
		"Hi %s,\n\n"+
			"Someone placed a higher bid on an auction you're watching.\n\n"+
			"  Item: %s\n"+
			"  New Leading Bid: %.2f %s\n\n"+
			"Bid again to stay in the lead:\n"+
			"  %s/auctions\n\n"+
			"— The GeoCore Team",
		name, auctionTitle, newAmount, currency, cfg.BaseURL,
	)

	if cfg.Host == "" || cfg.From == "" {
		fmt.Printf("[email-dev] Outbid email sent to %s for '%s'\n", to, auctionTitle)
		return nil
	}
	return send(cfg, to, fmt.Sprintf("You've been outbid on: %s", auctionTitle), body)
}

// SendSupportContactEmail notifies support/admin inbox about a new contact form message.
func SendSupportContactEmail(to, senderName, senderEmail, subject, message string) error {
	cfg := loadSMTP()
	if senderName == "" {
		senderName = "Unknown"
	}
	if subject == "" {
		subject = "General Inquiry"
	}

	body := fmt.Sprintf(
		"New support contact message received:\n\n"+
			"From: %s <%s>\n"+
			"Subject: %s\n\n"+
			"Message:\n%s\n",
		senderName, senderEmail, subject, message,
	)

	if cfg.Host == "" || cfg.From == "" {
		fmt.Printf("[email-dev] Support contact forwarded to %s from %s <%s>\n", to, senderName, senderEmail)
		return nil
	}
	return send(cfg, to, fmt.Sprintf("[Support] %s", subject), body)
}
