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

const groupWalletService = "wallet-service"

// startWalletConsumer listens to orders.events and escrow.events.
// On order.created → debit buyer wallet (escrow hold).
// On escrow.released → credit seller wallet.
func startWalletConsumer(ctx context.Context, db *gorm.DB) {
	// ── Orders consumer: debit buyer on order.created ─────────────────────────
	ordersConsumer := kafka.NewConsumer(kafka.TopicOrders, groupWalletService)
	go ordersConsumer.Run(ctx, func(ctx context.Context, event kafka.Event) error {
		if event.Type != "order.created" {
			return nil // only care about order.created
		}

		if kafka.IsProcessed(db, event.EventID, groupWalletService) {
			slog.Debug("kafka: wallet already processed event", "event_id", event.EventID)
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

		buyerID, err := uuid.Parse(buyerIDStr)
		if err != nil {
			return fmt.Errorf("invalid buyer_id: %w", err)
		}
		sellerID, _ := uuid.Parse(sellerIDStr)

		_, err = wallet.HoldFunds(db, buyerID, sellerID, amountFloat, currency, "ORDER", orderID)
		if err != nil {
			slog.Warn("kafka: wallet hold failed",
				"event_id", event.EventID, "order_id", orderID, "error", err)
			return err // will re-deliver
		}

		if err := kafka.MarkProcessed(db, event.EventID, groupWalletService); err != nil {
			slog.Warn("kafka: mark processed failed", "error", err)
		}
		slog.Info("kafka: wallet hold created for order",
			"order_id", orderID, "buyer_id", buyerIDStr, "amount", amountFloat)
		return nil
	})

	// ── Escrow consumer: credit seller on escrow.released ─────────────────────
	escrowConsumer := kafka.NewConsumer(kafka.TopicEscrow, groupWalletService)
	go escrowConsumer.Run(ctx, func(ctx context.Context, event kafka.Event) error {
		if event.Type != "escrow.released" {
			return nil
		}

		// Use composite key to avoid collision with orders consumer
		group := groupWalletService + "-escrow"
		if kafka.IsProcessed(db, event.EventID, group) {
			return nil
		}

		data, ok := event.Data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid event data type")
		}

		escrowIDStr, _ := data["escrow_id"].(string)
		sellerPaid, _ := data["seller_paid"].(bool)

		if escrowIDStr == "" || !sellerPaid {
			return nil // not ready or missing data
		}

		escrowID, err := uuid.Parse(escrowIDStr)
		if err != nil {
			return fmt.Errorf("invalid escrow_id: %w", err)
		}

		// Load escrow to get seller + amount
		var esc wallet.Escrow
		if err := db.First(&esc, "id = ?", escrowID).Error; err != nil {
			return fmt.Errorf("escrow not found: %w", err)
		}

		// Credit seller wallet — find seller wallet and add amount
		var sellerWallet wallet.Wallet
		if err := db.Where("user_id = ?", esc.SellerID).First(&sellerWallet).Error; err != nil {
			return fmt.Errorf("seller wallet not found: %w", err)
		}

		var balance wallet.WalletBalance
		if err := db.Where("wallet_id = ? AND currency = ?", sellerWallet.ID, esc.Currency).First(&balance).Error; err != nil {
			return fmt.Errorf("seller balance not found: %w", err)
		}

		sellerAmount := esc.Amount
		balance.Balance = balance.Balance.Add(sellerAmount)
		balance.AvailableBalance = balance.AvailableBalance.Add(sellerAmount)
		if err := db.Save(&balance).Error; err != nil {
			return err
		}

		refType := "escrow_release_kafka"
		now := ctx.Value("now") // best-effort
		_ = now
		if err := db.Create(&wallet.WalletTransaction{
			WalletID:      sellerWallet.ID,
			Type:          wallet.TransactionRelease,
			Currency:      esc.Currency,
			Amount:        sellerAmount,
			BalanceBefore: balance.Balance.Sub(sellerAmount),
			BalanceAfter:  balance.Balance,
			Fee:           esc.Fee,
			Status:        wallet.StatusCompleted,
			ReferenceID:   &escrowIDStr,
			ReferenceType: &refType,
			Description:   "Escrow release (Kafka consumer) for ORDER #" + esc.ReferenceID,
		}).Error; err != nil {
			return err
		}

		if err := kafka.MarkProcessed(db, event.EventID, group); err != nil {
			slog.Warn("kafka: mark processed failed", "error", err)
		}
		slog.Info("kafka: seller credited from escrow release",
			"escrow_id", escrowIDStr, "amount", sellerAmount.String())
		return nil
	})

	_ = decimal.Decimal{} // imported for potential future use
}
