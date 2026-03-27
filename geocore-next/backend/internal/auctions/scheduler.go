package auctions

import (
        "context"
        "fmt"
        "log"
        "time"

        "github.com/geocore-next/backend/internal/notifications"
        "github.com/geocore-next/backend/pkg/email"
        "github.com/google/uuid"
        "gorm.io/gorm"
        "gorm.io/gorm/clause"
)

// StartAuctionEndWorker runs every 60 seconds. It marks auctions whose
// ends_at has passed as "ended", sets the winner_id to the top bidder,
// sends winner/seller emails, and broadcasts an auction_ended WebSocket event.
func StartAuctionEndWorker(ctx context.Context, db *gorm.DB, hub *Hub) {
        ticker := time.NewTicker(60 * time.Second)
        defer ticker.Stop()

        log.Println("[auction-scheduler] auction-end worker started")

        for {
                select {
                case <-ctx.Done():
                        log.Println("[auction-scheduler] auction-end worker stopped")
                        return
                case <-ticker.C:
                        processEndedAuctions(db, hub)
                }
        }
}

// ProcessEndedAuctions is exported for testing. The unexported processEndedAuctions
// delegates to it internally so tests can call it directly.
func ProcessEndedAuctions(db *gorm.DB, hub *Hub) {
        processEndedAuctions(db, hub)
}

func processEndedAuctions(db *gorm.DB, hub *Hub) {
        // Process in batches of 100 to avoid loading unbounded rows into memory.
        // Each tick handles at most 100 ended auctions; remaining are caught on the next tick.
        var ended []Auction
        if err := db.Where("status = ? AND ends_at <= ?", "active", time.Now()).
                Limit(100).Find(&ended).Error; err != nil {
                log.Printf("[auction-scheduler] error querying ended auctions: %v", err)
                return
        }

        for _, auction := range ended {
                finalizeAuction(db, hub, auction)
        }
}

// userRecord is a minimal struct for fetching user email/name from the DB.
type userRecord struct {
        ID    uuid.UUID
        Name  string
        Email string
}

func finalizeAuction(db *gorm.DB, hub *Hub, auction Auction) {
        var winnerID *uuid.UUID
        var finalBid float64

        err := db.Transaction(func(tx *gorm.DB) error {
                // Re-fetch with pessimistic lock (FOR UPDATE) to avoid concurrent finalization races.
                // The clause.Locking approach is idiomatic in GORM and is a no-op in SQLite.
                var a Auction
                if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
                        First(&a, "id = ? AND status = ?", auction.ID, "active").Error; err != nil {
                        return err
                }

                updates := map[string]interface{}{
                        "status": "ended",
                }

                // Find the top bidder; on tied amounts, the earliest bid wins (first-bidder-wins rule).
                var topBid Bid
                if err := tx.Where("auction_id = ?", a.ID).
                        Order("amount DESC, placed_at ASC").
                        First(&topBid).Error; err == nil {
                        updates["winner_id"] = topBid.UserID
                        wID := topBid.UserID
                        winnerID = &wID
                        finalBid = topBid.Amount
                }

                if err := tx.Model(&a).Updates(updates).Error; err != nil {
                        return err
                }

                return nil
        })

        if err != nil {
                log.Printf("[auction-scheduler] error finalizing auction %s: %v", auction.ID, err)
                return
        }

        // Re-fetch the finalized auction for notification data
        var finalized Auction
        if err := db.First(&finalized, "id = ?", auction.ID).Error; err != nil {
                log.Printf("[auction-scheduler] error re-fetching auction %s: %v", auction.ID, err)
                return
        }

        // Broadcast auction_ended event via WebSocket hub
        winnerStr := "null"
        if winnerID != nil {
                winnerStr = fmt.Sprintf(`"%s"`, winnerID.String())
        }
        payload := fmt.Sprintf(
                `{"event": "auction_ended", "auction_id": "%s", "winner_id": %s, "final_bid": %.2f}`,
                finalized.ID,
                winnerStr,
                finalBid,
        )
        if hub != nil {
                hub.Broadcast(&BroadcastMsg{
                        AuctionID: finalized.ID.String(),
                        Data:      []byte(payload),
                })
        }

        // Look up listing title for emails (best-effort)
        auctionTitle := finalized.ID.String()
        var listing struct {
                Title string
        }
        if db.Table("listings").Select("title").Where("id = ?", finalized.ListingID).Scan(&listing).Error == nil && listing.Title != "" {
                auctionTitle = listing.Title
        }

        // Notify winner and seller
        if winnerID != nil {
                notifyAuctionWon(*winnerID, finalized.SellerID, finalized.ID.String(), finalBid, finalized.Currency)

                // Send winner email (best-effort)
                go sendAuctionWonEmail(db, *winnerID, auctionTitle, finalBid, finalized.Currency)

                // Send seller ended-with-winner email (best-effort)
                go sendAuctionEndedSellerEmail(db, finalized.SellerID, auctionTitle, finalBid, finalized.Currency, true)
        } else {
                // No bids — notify seller the auction ended with no winner
                go notifyAuctionEndedNoWinner(finalized.SellerID, finalized.ID.String())

                // Send seller ended-no-winner email
                go sendAuctionEndedSellerEmail(db, finalized.SellerID, auctionTitle, 0, finalized.Currency, false)
        }

        log.Printf("[auction-scheduler] auction %s ended, winner: %v, final bid: %.2f",
                finalized.ID, winnerID, finalBid)
}

func notifyAuctionEndedNoWinner(sellerID uuid.UUID, auctionID string) {
        if globalNotifSvc == nil {
                return
        }
        globalNotifSvc.Notify(notifications.NotifyInput{
                UserID: sellerID,
                Type:   notifications.TypeAuctionEnded,
                Title:  "Your auction ended without a winner",
                Body:   "Your auction ended with no bids. You can relist the item or adjust the reserve price.",
                Data:   map[string]string{"auction_id": auctionID},
        })
}

func sendAuctionWonEmail(db *gorm.DB, winnerID uuid.UUID, auctionTitle string, amount float64, currency string) {
        var u userRecord
        if err := db.Table("users").Select("id, name, email").Where("id = ?", winnerID).Scan(&u).Error; err != nil {
                log.Printf("[auction-scheduler] could not look up winner email for %s: %v", winnerID, err)
                return
        }
        if u.Email == "" {
                return
        }
        if err := email.SendAuctionWonEmail(u.Email, u.Name, auctionTitle, amount, currency); err != nil {
                log.Printf("[auction-scheduler] winner email error for %s: %v", winnerID, err)
        }
}

func sendAuctionEndedSellerEmail(db *gorm.DB, sellerID uuid.UUID, auctionTitle string, amount float64, currency string, hasWinner bool) {
        var u userRecord
        if err := db.Table("users").Select("id, name, email").Where("id = ?", sellerID).Scan(&u).Error; err != nil {
                log.Printf("[auction-scheduler] could not look up seller email for %s: %v", sellerID, err)
                return
        }
        if u.Email == "" {
                return
        }
        if err := email.SendAuctionEndedSellerEmail(u.Email, u.Name, auctionTitle, amount, currency, hasWinner); err != nil {
                log.Printf("[auction-scheduler] seller email error for %s: %v", sellerID, err)
        }
}

// ApplyAuctionIndexes creates database indexes needed for efficient auction queries.
// Called once during application startup (idempotent).
func ApplyAuctionIndexes(db *gorm.DB) {
        indexes := []string{
                `CREATE INDEX IF NOT EXISTS idx_auctions_status_ends_at ON auctions(status, ends_at)`,
                `CREATE INDEX IF NOT EXISTS idx_auctions_seller_id ON auctions(seller_id)`,
                `CREATE INDEX IF NOT EXISTS idx_auctions_listing_id ON auctions(listing_id)`,
                `CREATE INDEX IF NOT EXISTS idx_auctions_current_bid ON auctions(current_bid)`,
                `CREATE INDEX IF NOT EXISTS idx_auctions_bid_count ON auctions(bid_count)`,
                `CREATE INDEX IF NOT EXISTS idx_auctions_created_at ON auctions(created_at DESC)`,
                `CREATE INDEX IF NOT EXISTS idx_bids_auction_id_amount ON bids(auction_id, amount DESC)`,
                `CREATE INDEX IF NOT EXISTS idx_bids_user_id ON bids(user_id)`,
                `CREATE INDEX IF NOT EXISTS idx_bids_is_auto ON bids(auction_id, is_auto) WHERE is_auto = true`,
        }
        for _, idx := range indexes {
                if err := db.Exec(idx).Error; err != nil {
                        log.Printf("[auction-indexes] index creation skipped: %v", err)
                }
        }
        log.Println("[auction-indexes] auction indexes ready")
}
