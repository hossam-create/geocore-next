package analytics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// PostHogClient handles analytics tracking via PostHog
type PostHogClient struct {
	apiKey     string
	host       string
	httpClient *http.Client
}

// Event represents an analytics event
type Event struct {
	Event      string                 `json:"event"`
	DistinctID string                 `json:"distinct_id"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Timestamp  time.Time              `json:"timestamp,omitempty"`
}

// NewPostHogClient creates a new PostHog client
func NewPostHogClient() *PostHogClient {
	host := os.Getenv("POSTHOG_HOST")
	if host == "" {
		host = "https://app.posthog.com"
	}

	return &PostHogClient{
		apiKey: os.Getenv("POSTHOG_API_KEY"),
		host:   host,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// IsConfigured checks if PostHog is configured
func (p *PostHogClient) IsConfigured() bool {
	return p.apiKey != ""
}

// Capture sends an event to PostHog
func (p *PostHogClient) Capture(event Event) error {
	if !p.IsConfigured() {
		return nil // Silently skip if not configured
	}

	if event.Properties == nil {
		event.Properties = make(map[string]interface{})
	}
	event.Properties["$lib"] = "geocore-go"

	payload := map[string]interface{}{
		"api_key":     p.apiKey,
		"event":       event.Event,
		"distinct_id": event.DistinctID,
		"properties":  event.Properties,
		"timestamp":   event.Timestamp.Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", p.host+"/capture/", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("posthog error: status %d", resp.StatusCode)
	}

	return nil
}

// Identify associates user properties with a distinct ID
func (p *PostHogClient) Identify(distinctID string, properties map[string]interface{}) error {
	if !p.IsConfigured() {
		return nil
	}

	payload := map[string]interface{}{
		"api_key":     p.apiKey,
		"event":       "$identify",
		"distinct_id": distinctID,
		"$set":        properties,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", p.host+"/capture/", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ===== Pre-built Event Helpers =====

// TrackSignup tracks user signup
func (p *PostHogClient) TrackSignup(userID, email, signupMethod string) {
	p.Capture(Event{
		Event:      "user_signed_up",
		DistinctID: userID,
		Properties: map[string]interface{}{
			"email":         email,
			"signup_method": signupMethod,
		},
		Timestamp: time.Now(),
	})
}

// TrackLogin tracks user login
func (p *PostHogClient) TrackLogin(userID, loginMethod string) {
	p.Capture(Event{
		Event:      "user_logged_in",
		DistinctID: userID,
		Properties: map[string]interface{}{
			"login_method": loginMethod,
		},
		Timestamp: time.Now(),
	})
}

// TrackProductViewed tracks product view
func (p *PostHogClient) TrackProductViewed(userID, productID, productName, category string, price float64) {
	p.Capture(Event{
		Event:      "product_viewed",
		DistinctID: userID,
		Properties: map[string]interface{}{
			"product_id":   productID,
			"product_name": productName,
			"category":     category,
			"price":        price,
		},
		Timestamp: time.Now(),
	})
}

// TrackAuctionViewed tracks auction view
func (p *PostHogClient) TrackAuctionViewed(userID, auctionID, auctionType string, currentBid float64) {
	p.Capture(Event{
		Event:      "auction_viewed",
		DistinctID: userID,
		Properties: map[string]interface{}{
			"auction_id":   auctionID,
			"auction_type": auctionType,
			"current_bid":  currentBid,
		},
		Timestamp: time.Now(),
	})
}

// TrackBidPlaced tracks bid placement
func (p *PostHogClient) TrackBidPlaced(userID, auctionID string, bidAmount, previousBid float64, isAutoBid bool) {
	p.Capture(Event{
		Event:      "bid_placed",
		DistinctID: userID,
		Properties: map[string]interface{}{
			"auction_id":   auctionID,
			"bid_amount":   bidAmount,
			"previous_bid": previousBid,
			"is_auto_bid":  isAutoBid,
		},
		Timestamp: time.Now(),
	})
}

// TrackAuctionWon tracks auction win
func (p *PostHogClient) TrackAuctionWon(userID, auctionID string, winningBid float64) {
	p.Capture(Event{
		Event:      "auction_won",
		DistinctID: userID,
		Properties: map[string]interface{}{
			"auction_id":  auctionID,
			"winning_bid": winningBid,
		},
		Timestamp: time.Now(),
	})
}

// TrackBuyNow tracks Buy Now purchase
func (p *PostHogClient) TrackBuyNow(userID, auctionID string, price float64) {
	p.Capture(Event{
		Event:      "buy_now_completed",
		DistinctID: userID,
		Properties: map[string]interface{}{
			"auction_id": auctionID,
			"price":      price,
		},
		Timestamp: time.Now(),
	})
}

// TrackSearch tracks search queries
func (p *PostHogClient) TrackSearch(userID, query string, resultsCount int, filters map[string]interface{}) {
	props := map[string]interface{}{
		"search_query":  query,
		"results_count": resultsCount,
	}
	for k, v := range filters {
		props["filter_"+k] = v
	}

	p.Capture(Event{
		Event:      "search_performed",
		DistinctID: userID,
		Properties: props,
		Timestamp:  time.Now(),
	})
}

// TrackCheckoutStarted tracks checkout initiation
func (p *PostHogClient) TrackCheckoutStarted(userID string, cartTotal float64, itemCount int) {
	p.Capture(Event{
		Event:      "checkout_started",
		DistinctID: userID,
		Properties: map[string]interface{}{
			"cart_total": cartTotal,
			"item_count": itemCount,
		},
		Timestamp: time.Now(),
	})
}

// TrackPurchaseCompleted tracks successful purchase
func (p *PostHogClient) TrackPurchaseCompleted(userID, orderID string, total float64, paymentMethod string) {
	p.Capture(Event{
		Event:      "purchase_completed",
		DistinctID: userID,
		Properties: map[string]interface{}{
			"order_id":       orderID,
			"total":          total,
			"payment_method": paymentMethod,
		},
		Timestamp: time.Now(),
	})
}

// TrackWalletDeposit tracks wallet deposit
func (p *PostHogClient) TrackWalletDeposit(userID string, amount float64, currency string) {
	p.Capture(Event{
		Event:      "wallet_deposit",
		DistinctID: userID,
		Properties: map[string]interface{}{
			"amount":   amount,
			"currency": currency,
		},
		Timestamp: time.Now(),
	})
}

// TrackWalletWithdraw tracks wallet withdrawal
func (p *PostHogClient) TrackWalletWithdraw(userID string, amount float64, currency string) {
	p.Capture(Event{
		Event:      "wallet_withdraw",
		DistinctID: userID,
		Properties: map[string]interface{}{
			"amount":   amount,
			"currency": currency,
		},
		Timestamp: time.Now(),
	})
}

// TrackEscrowCreated tracks escrow creation
func (p *PostHogClient) TrackEscrowCreated(userID, auctionID string, amount float64) {
	p.Capture(Event{
		Event:      "escrow_created",
		DistinctID: userID,
		Properties: map[string]interface{}{
			"auction_id": auctionID,
			"amount":     amount,
		},
		Timestamp: time.Now(),
	})
}

// TrackEscrowReleased tracks escrow release
func (p *PostHogClient) TrackEscrowReleased(userID, auctionID string, amount, fee float64) {
	p.Capture(Event{
		Event:      "escrow_released",
		DistinctID: userID,
		Properties: map[string]interface{}{
			"auction_id":   auctionID,
			"amount":       amount,
			"platform_fee": fee,
		},
		Timestamp: time.Now(),
	})
}

// TrackKYCSubmitted tracks KYC submission
func (p *PostHogClient) TrackKYCSubmitted(userID, documentType string) {
	p.Capture(Event{
		Event:      "kyc_submitted",
		DistinctID: userID,
		Properties: map[string]interface{}{
			"document_type": documentType,
		},
		Timestamp: time.Now(),
	})
}

// TrackKYCApproved tracks KYC approval
func (p *PostHogClient) TrackKYCApproved(userID string) {
	p.Capture(Event{
		Event:      "kyc_approved",
		DistinctID: userID,
		Timestamp:  time.Now(),
	})
}

// TrackSubscription tracks subscription purchase
func (p *PostHogClient) TrackSubscription(userID, planID, planName string, price float64) {
	p.Capture(Event{
		Event:      "subscription_purchased",
		DistinctID: userID,
		Properties: map[string]interface{}{
			"plan_id":   planID,
			"plan_name": planName,
			"price":     price,
		},
		Timestamp: time.Now(),
	})
}

// TrackListingCreated tracks new listing creation
func (p *PostHogClient) TrackListingCreated(userID, listingID, category string, price float64, hasAuction bool) {
	p.Capture(Event{
		Event:      "listing_created",
		DistinctID: userID,
		Properties: map[string]interface{}{
			"listing_id":  listingID,
			"category":    category,
			"price":       price,
			"has_auction": hasAuction,
		},
		Timestamp: time.Now(),
	})
}

// TrackError tracks application errors
func (p *PostHogClient) TrackError(userID, errorType, errorMessage, endpoint string) {
	p.Capture(Event{
		Event:      "error_occurred",
		DistinctID: userID,
		Properties: map[string]interface{}{
			"error_type":    errorType,
			"error_message": errorMessage,
			"endpoint":      endpoint,
		},
		Timestamp: time.Now(),
	})
}
