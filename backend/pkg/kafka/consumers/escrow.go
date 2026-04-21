package consumers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const groupEscrowService = "escrow-service"

// startEscrowConsumer listens to orders.events.
// On order.created → create escrow record.
// On order.cancelled → cancel escrow and refund buyer.
func startEscrowConsumer(ctx context.Context, db *gorm.DB) {
	consumer := kafka.NewConsumer(kafka.TopicOrders, groupEscrowService)
	go consumer.Run(ctx, func(ctx context.Context, event kafka.Event) error {
		switch event.Type {
		case "order.created":
			return handleOrderCreatedEscrow(db, event)
		case "order.cancelled":
			return handleOrderCancelledEscrow(db, event)
		default:
			return nil
		}
	})
}

func handleOrderCreatedEscrow(db *gorm.DB, event kafka.Event) error {
	if kafka.IsProcessed(db, event.EventID, groupEscrowService) {
		return nil
	}

	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid event data type")
	}

	buyerIDStr, _ := data["buyer_id"].(string)
	sellerIDStr, _ := data["seller_id"].(string)
	orderID, _ := data["order_id"].(string)
	amountFloat, _ := data["amount"].(float64)
	currency, _ := data["currency"].(string)
	if currency == "" {
		currency = "USD"
	}

	if buyerIDStr == "" || sellerIDStr == "" || orderID == "" {
		return fmt.Errorf("missing required fields in order.created event")
	}

	buyerID, _ := uuid.Parse(buyerIDStr)
	sellerID, _ := uuid.Parse(sellerIDStr)
	amountDec := decimal.NewFromFloat(amountFloat)
	fee := amountDec.Mul(decimal.NewFromFloat(0.025))

	escrow := wallet.Escrow{
		BuyerID:     buyerID,
		SellerID:    sellerID,
		Amount:      amountDec,
		Currency:    wallet.Currency(currency),
		Fee:         fee,
		Status:      wallet.StatusPending,
		ReferenceID: orderID,
		Type:        "ORDER",
	}

	if err := db.Create(&escrow).Error; err != nil {
		slog.Warn("kafka: escrow create failed", "event_id", event.EventID, "error", err)
		return err
	}

	if err := kafka.MarkProcessed(db, event.EventID, groupEscrowService); err != nil {
		slog.Warn("kafka: mark processed failed", "error", err)
	}
	slog.Info("kafka: escrow created for order",
		"order_id", orderID, "escrow_id", escrow.ID, "amount", amountDec.String())
	return nil
}

func handleOrderCancelledEscrow(db *gorm.DB, event kafka.Event) error {
	group := groupEscrowService + "-cancel"
	if kafka.IsProcessed(db, event.EventID, group) {
		return nil
	}

	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid event data type")
	}

	orderID, _ := data["order_id"].(string)
	if orderID == "" {
		return fmt.Errorf("missing order_id in order.cancelled event")
	}

	var esc wallet.Escrow
	if err := db.Where("reference_id = ? AND type = ? AND status = ?",
		orderID, "ORDER", wallet.StatusPending).First(&esc).Error; err != nil {
		slog.Warn("kafka: no pending escrow found for order cancellation",
			"order_id", orderID)
		return nil // not an error — escrow may already be released
	}

	esc.Status = wallet.StatusCancelled
	if err := db.Save(&esc).Error; err != nil {
		return err
	}

	if err := kafka.MarkProcessed(db, event.EventID, group); err != nil {
		slog.Warn("kafka: mark processed failed", "error", err)
	}
	slog.Info("kafka: escrow cancelled for order", "order_id", orderID)
	return nil
}
