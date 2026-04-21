package auctions

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/geocore-next/backend/internal/freeze"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// DutchPriceUpdate is the WebSocket message broadcast on each price tick.
type DutchPriceUpdate struct {
	Type            string  `json:"type"`
	AuctionID       string  `json:"auction_id"`
	CurrentPrice    float64 `json:"current_price"`
	NextDecrementAt string  `json:"next_decrement_at"`
	ReservePrice    float64 `json:"reserve_price"`
}

// DutchAuctionManager manages background tickers for all active dutch auctions.
type DutchAuctionManager struct {
	db      *gorm.DB
	rdb     *redis.Client
	tickers map[uuid.UUID]context.CancelFunc
	mu      sync.Mutex
}

// NewDutchAuctionManager creates a new manager instance.
func NewDutchAuctionManager(db *gorm.DB, rdb *redis.Client) *DutchAuctionManager {
	return &DutchAuctionManager{
		db:      db,
		rdb:     rdb,
		tickers: make(map[uuid.UUID]context.CancelFunc),
	}
}

// RestoreOnStartup finds all active dutch auctions and starts their tickers.
func (m *DutchAuctionManager) RestoreOnStartup() {
	var auctions []Auction
	m.db.Where("type = ? AND status = ? AND ends_at > ?", AuctionTypeDutch, StatusActive, time.Now()).
		Find(&auctions)

	for i := range auctions {
		m.StartTicker(auctions[i].ID)
	}
	if len(auctions) > 0 {
		slog.Info("dutch-ticker: restored tickers on startup", "count", len(auctions))
	}
}

// StartTicker begins a background goroutine that decrements the price at the
// configured interval and broadcasts updates via Redis pub/sub.
func (m *DutchAuctionManager) StartTicker(auctionID uuid.UUID) {
	m.mu.Lock()
	// If already running, skip
	if _, exists := m.tickers[auctionID]; exists {
		m.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.tickers[auctionID] = cancel
	m.mu.Unlock()

	go m.runTicker(ctx, auctionID)
	slog.Info("dutch-ticker: started", "auction_id", auctionID)
}

// StopTicker stops the background ticker for a specific auction.
func (m *DutchAuctionManager) StopTicker(auctionID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cancel, exists := m.tickers[auctionID]; exists {
		cancel()
		delete(m.tickers, auctionID)
		slog.Info("dutch-ticker: stopped", "auction_id", auctionID)
	}
}

func (m *DutchAuctionManager) runTicker(ctx context.Context, auctionID uuid.UUID) {
	// Load auction to get interval
	var auction Auction
	if err := m.db.First(&auction, "id = ?", auctionID).Error; err != nil {
		slog.Error("dutch-ticker: auction not found", "auction_id", auctionID, "error", err)
		return
	}

	intervalMin := 5
	if auction.DutchDropInterval != nil && *auction.DutchDropInterval > 0 {
		intervalMin = *auction.DutchDropInterval
	}
	interval := time.Duration(intervalMin) * time.Minute

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Broadcast the initial price immediately
	m.broadcastPrice(&auction)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Reload auction to check current status
			if err := m.db.First(&auction, "id = ?", auctionID).Error; err != nil {
				slog.Error("dutch-ticker: reload failed", "auction_id", auctionID)
				return
			}

			// Stop if auction is no longer active
			if auction.Status != StatusActive || time.Now().After(auction.EndsAt) {
				m.StopTicker(auctionID)
				return
			}

			currentPrice := auction.GetCurrentDutchPrice()

			// Update current_bid in DB to reflect the decremented price
			m.db.Model(&auction).Update("current_bid", currentPrice)

			// Check if price hit the floor (reserve/end price)
			if auction.DutchEndPrice != nil && currentPrice <= *auction.DutchEndPrice {
				slog.Info("dutch-ticker: reached end price, ending auction", "auction_id", auctionID, "price", currentPrice)
				m.db.Model(&auction).Update("status", StatusEnded)
				// Sprint 8.5: Audit log for auction_ended
				freeze.LogAudit(m.db, "auction_ended", uuid.Nil, auction.ID, fmt.Sprintf("method=dutch_floor price=%.2f auction_id=%s", currentPrice, auctionID))
				m.broadcastEnded(&auction, currentPrice)
				m.StopTicker(auctionID)
				return
			}

			m.broadcastPrice(&auction)
		}
	}
}

func (m *DutchAuctionManager) broadcastPrice(auction *Auction) {
	currentPrice := auction.GetCurrentDutchPrice()

	intervalMin := 5
	if auction.DutchDropInterval != nil && *auction.DutchDropInterval > 0 {
		intervalMin = *auction.DutchDropInterval
	}
	nextDecrement := time.Now().Add(time.Duration(intervalMin) * time.Minute)

	reservePrice := 0.0
	if auction.DutchEndPrice != nil {
		reservePrice = *auction.DutchEndPrice
	}

	msg := DutchPriceUpdate{
		Type:            "dutch_price_update",
		AuctionID:       auction.ID.String(),
		CurrentPrice:    currentPrice,
		NextDecrementAt: nextDecrement.Format(time.RFC3339),
		ReservePrice:    reservePrice,
	}

	data, _ := json.Marshal(msg)
	channel := fmt.Sprintf("auction:%s", auction.ID)
	m.rdb.Publish(context.Background(), channel, string(data))
}

func (m *DutchAuctionManager) broadcastEnded(auction *Auction, finalPrice float64) {
	msg := map[string]interface{}{
		"type":        "dutch_ended",
		"auction_id":  auction.ID.String(),
		"final_price": finalPrice,
		"reason":      "reached_reserve_price",
	}
	data, _ := json.Marshal(msg)
	channel := fmt.Sprintf("auction:%s", auction.ID)
	m.rdb.Publish(context.Background(), channel, string(data))
}
