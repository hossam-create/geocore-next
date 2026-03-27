package auctions

import (
        "fmt"

        "github.com/geocore-next/backend/internal/notifications"
        "github.com/google/uuid"
)

// fmtAmount converts a float to string for notification metadata.
func fmtAmount(v float64) string { return fmt.Sprintf("%.2f", v) }

// NotificationService is an interface satisfied by *notifications.Service.
// Using an interface avoids circular imports.
type NotificationService interface {
        Notify(input notifications.NotifyInput)
}

var globalNotifSvc NotificationService

// SetNotificationService wires the notification service into this package.
// Called once from main.go after all routes are registered.
func SetNotificationService(svc NotificationService) {
        globalNotifSvc = svc
}

func notifyNewBid(auction *Auction, bidderID uuid.UUID, prevLeaderID *uuid.UUID, amount float64) {
        if globalNotifSvc == nil {
                return
        }

        auctionIDStr := auction.ID.String()

        // Notify seller about new bid
        go globalNotifSvc.Notify(notifications.NotifyInput{
                UserID: auction.SellerID,
                Type:   notifications.TypeNewBid,
                Title:  "New bid on your auction",
                Body:   fmt.Sprintf("A bid of %.2f %s was placed on your auction.", amount, auction.Currency),
                Data:   map[string]string{"auction_id": auctionIDStr},
        })

        // Notify previous leader that they were outbid (with amount + currency for email)
        if prevLeaderID != nil && *prevLeaderID != uuid.Nil && *prevLeaderID != bidderID {
                leaderID := *prevLeaderID
                go globalNotifSvc.Notify(notifications.NotifyInput{
                        UserID: leaderID,
                        Type:   notifications.TypeOutbid,
                        Title:  "You've been outbid!",
                        Body:   fmt.Sprintf("Someone placed a higher bid of %.2f %s. Bid again to win.", amount, auction.Currency),
                        Data: map[string]string{
                                "auction_id":    auctionIDStr,
                                "auction_title": fmt.Sprintf("Auction %s", auction.ID.String()[:8]),
                                "amount":        fmtAmount(amount),
                                "currency":      auction.Currency,
                        },
                })
        }
}

func notifyAuctionWon(winnerID, sellerID uuid.UUID, auctionID string, amount float64, currency string) {
        if globalNotifSvc == nil {
                return
        }

        go globalNotifSvc.Notify(notifications.NotifyInput{
                UserID: winnerID,
                Type:   notifications.TypeAuctionWon,
                Title:  "You won the auction!",
                Body:   fmt.Sprintf("Congratulations! You won with a bid of %.2f %s.", amount, currency),
                Data:   map[string]string{"auction_id": auctionID},
        })

        go globalNotifSvc.Notify(notifications.NotifyInput{
                UserID: sellerID,
                Type:   notifications.TypeAuctionEnded,
                Title:  "Your auction ended",
                Body:   fmt.Sprintf("Your auction ended. Winning bid: %.2f %s.", amount, currency),
                Data:   map[string]string{"auction_id": auctionID},
        })
}
