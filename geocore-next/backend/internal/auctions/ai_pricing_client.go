package auctions

  import (
  	"bytes"
  	"context"
  	"encoding/json"
  	"fmt"
  	"io"
  	"log/slog"
  	"net/http"
  	"os"
  	"time"
  )

  // AIPricingClient calls the GeoCore AI Pricing microservice.
  // Inspired by T51-AI-Bidding-and-Auction-Pricing-Agent (DQN bidding concepts).
  type AIPricingClient struct {
  	baseURL    string
  	httpClient *http.Client
  }

  type BidPredictRequest struct {
  	CurrentPrice     float64 `json:"current_price"`
  	StartingPrice    float64 `json:"starting_price"`
  	ReservePrice     float64 `json:"reserve_price,omitempty"`
  	EstimatedValue   float64 `json:"estimated_value,omitempty"`
  	SecondsRemaining int     `json:"seconds_remaining"`
  	TotalSeconds     int     `json:"total_seconds"`
  	BidCount         int     `json:"bid_count"`
  	Category         string  `json:"category"`
  	Strategy         string  `json:"strategy,omitempty"`
  	Currency         string  `json:"currency,omitempty"`
  	AuctionID        string  `json:"auction_id,omitempty"`
  }

  type BidPredictResponse struct {
  	ShouldBid            bool    `json:"should_bid"`
  	OptimalBid           float64 `json:"optimal_bid"`
  	MaxBid               float64 `json:"max_bid"`
  	MinIncrement         float64 `json:"min_increment"`
  	EstimatedMarketValue float64 `json:"estimated_market_value"`
  	WinProbability       float64 `json:"win_probability"`
  	UrgencyScore         float64 `json:"urgency_score"`
  	CompetitionScore     float64 `json:"competition_score"`
  	Strategy             string  `json:"strategy"`
  	Explanation          string  `json:"explanation"`
  	Currency             string  `json:"currency"`
  }

  func NewAIPricingClient() *AIPricingClient {
  	baseURL := os.Getenv("AI_PRICING_URL")
  	if baseURL == "" {
  		baseURL = "http://localhost:8090"
  	}
  	return &AIPricingClient{
  		baseURL:    baseURL,
  		httpClient: &http.Client{Timeout: 5 * time.Second},
  	}
  }

  func (c *AIPricingClient) Predict(ctx context.Context, req BidPredictRequest) (*BidPredictResponse, error) {
  	body, _ := json.Marshal(req)
  	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/predict", bytes.NewReader(body))
  	if err != nil {
  		return nil, err
  	}
  	httpReq.Header.Set("Content-Type", "application/json")
  	resp, err := c.httpClient.Do(httpReq)
  	if err != nil {
  		slog.Warn("AI pricing service unavailable", "error", err.Error())
  		return nil, fmt.Errorf("ai pricing unavailable: %w", err)
  	}
  	defer resp.Body.Close()
  	var result struct {
  		Success bool               `json:"success"`
  		Data    BidPredictResponse `json:"data"`
  		Error   string             `json:"error"`
  	}
  	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
  		return nil, err
  	}
  	if !result.Success {
  		return nil, fmt.Errorf("ai pricing error: %s", result.Error)
  	}
  	return &result.Data, nil
  }

  func (c *AIPricingClient) IsHealthy(ctx context.Context) bool {
  	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
  	resp, err := c.httpClient.Do(req)
  	if err != nil { return false }
  	defer func() { io.Copy(io.Discard, resp.Body); resp.Body.Close() }()
  	return resp.StatusCode == http.StatusOK
  }
  