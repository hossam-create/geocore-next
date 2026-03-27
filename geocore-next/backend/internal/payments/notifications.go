package payments

import (
        "fmt"

        "github.com/geocore-next/backend/internal/notifications"
        "github.com/google/uuid"
)

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

func notifyPaymentConfirmed(buyerID, sellerID uuid.UUID, amount float64, currency, itemTitle string) {
        if globalNotifSvc == nil {
                return
        }

        amountStr := fmt.Sprintf("%.2f %s", amount, currency)

        // Notify buyer: payment confirmed, funds in escrow
        go globalNotifSvc.Notify(notifications.NotifyInput{
                UserID: buyerID,
                Type:   notifications.TypePaymentSuccess,
                Title:  "Payment confirmed",
                Body:   fmt.Sprintf("Your payment of %s for \"%s\" is held in escrow.", amountStr, itemTitle),
                Data:   map[string]string{"item_title": itemTitle, "amount": amountStr},
        })

        // Notify seller: new order received
        go globalNotifSvc.Notify(notifications.NotifyInput{
                UserID: sellerID,
                Type:   notifications.TypePaymentSuccess,
                Title:  "New order received",
                Body:   fmt.Sprintf("You have a new order for \"%s\" — %s is held in escrow.", itemTitle, amountStr),
                Data:   map[string]string{"item_title": itemTitle, "amount": amountStr},
        })
}

func notifyEscrowReleased(sellerID uuid.UUID, amount float64, currency string) {
        if globalNotifSvc == nil {
                return
        }

        go globalNotifSvc.Notify(notifications.NotifyInput{
                UserID: sellerID,
                Type:   notifications.TypeEscrowReleased,
                Title:  "Funds released",
                Body:   fmt.Sprintf("%.2f %s has been released from escrow to your account.", amount, currency),
                Data:   map[string]string{"amount": fmt.Sprintf("%.2f", amount), "currency": currency},
        })
}
