package payments

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/geocore-next/backend/pkg/events"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ════════════════════════════════════════════════════════════════════════════
// PayMob handler — payment init + HMAC-verified webhook
// ════════════════════════════════════════════════════════════════════════════

// InitPayMob initializes the PayMob client singleton.
var paymobClient *PayMobClient

func InitPayMob() {
	if paymobClient == nil {
		paymobClient = NewPayMobClient()
	}
}

// CreatePayMobPayment initiates a PayMob payment:
//  1. Authenticate with PayMob API
//  2. Register order
//  3. Generate payment key
//  4. Return iframe URL for frontend redirect
func (h *Handler) CreatePayMobPayment(c *gin.Context) {
	InitPayMob()
	if !paymobClient.IsConfigured() {
		response.BadRequest(c, "PayMob not configured")
		return
	}

	userID, err := uuid.Parse(c.MustGet("user_id").(string))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req PayMobInitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if req.Currency == "" {
		req.Currency = "EGP"
	}

	// Idempotency check
	if req.IdempotencyKey != "" {
		var existing PayMobOrder
		if err := h.db.Where("idempotency_key = ?", req.IdempotencyKey).First(&existing).Error; err == nil {
			// Already processed — return existing iframe URL
			iframeURL := paymobClient.IframeURL(existing.PaymentKey)
			response.OK(c, gin.H{
				"paymob_order_id": existing.PayMobOrderID,
				"iframe_url":      iframeURL,
				"status":          existing.Status,
			})
			return
		}
	}

	ctx := c.Request.Context()

	// Step 1: Get auth token
	authToken, err := paymobClient.GetAuthToken(ctx)
	if err != nil {
		slog.Error("PayMob auth failed", "error", err)
		response.InternalError(c, err)
		return
	}

	// Step 2: Register order
	merchantOrderID := fmt.Sprintf("GC-%s-%d", userID.String()[:8], time.Now().UnixNano())
	paymobOrderID, err := paymobClient.RegisterOrder(ctx, authToken, req.AmountCents, req.Currency, merchantOrderID)
	if err != nil {
		slog.Error("PayMob register order failed", "error", err)
		response.InternalError(c, err)
		return
	}

	// Step 3: Get payment key
	paymentKey, err := paymobClient.GetPaymentKey(ctx, authToken, paymobOrderID, req.AmountCents, req.Currency, nil)
	if err != nil {
		slog.Error("PayMob payment key failed", "error", err)
		response.InternalError(c, err)
		return
	}

	// Step 4: Store order record
	order := PayMobOrder{
		UserID:         userID,
		PayMobOrderID:  paymobOrderID,
		PaymentKey:     paymentKey,
		AmountCents:    req.AmountCents,
		Currency:       req.Currency,
		Status:         PaymentStatusPending,
		IdempotencyKey: req.IdempotencyKey,
	}
	if err := h.db.Create(&order).Error; err != nil {
		slog.Error("PayMob order save failed", "error", err)
		response.InternalError(c, err)
		return
	}

	// Also create a Payment record for unified tracking
	var listingID, auctionID *uuid.UUID
	if req.ListingID != nil {
		if lid, e := uuid.Parse(*req.ListingID); e == nil {
			listingID = &lid
		}
	}
	if req.AuctionID != nil {
		if aid, e := uuid.Parse(*req.AuctionID); e == nil {
			auctionID = &aid
		}
	}
	payment := Payment{
		UserID:                userID,
		ListingID:             listingID,
		AuctionID:             auctionID,
		Kind:                  PaymentKindPurchase,
		Amount:                float64(req.AmountCents) / 100.0,
		Currency:              req.Currency,
		Status:                PaymentStatusPending,
		StripePaymentIntentID: fmt.Sprintf("paymob_%d", paymobOrderID),
	}
	if err := h.db.Create(&payment).Error; err != nil {
		slog.Warn("PayMob payment record creation failed", "error", err)
	}

	iframeURL := paymobClient.IframeURL(paymentKey)
	metrics.IncWalletOp("paymob_init", "success")
	response.Created(c, gin.H{
		"paymob_order_id": paymobOrderID,
		"iframe_url":      iframeURL,
		"payment_id":      payment.ID,
		"status":          "pending",
	})
}

// PayMobWebhook handles HMAC-verified webhook callbacks from PayMob.
// This is the critical path for confirming payments — must be idempotent.
func (h *Handler) PayMobWebhook(c *gin.Context) {
	InitPayMob()

	// Read raw body for HMAC verification
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.BadRequest(c, "failed to read body")
		return
	}

	// Verify HMAC signature
	receivedHMAC := c.GetHeader("X-Paymob-HMAC")
	if !paymobClient.VerifyHMAC(body, receivedHMAC) {
		slog.Warn("PayMob webhook HMAC verification failed",
			"hmac_header", receivedHMAC,
		)
		response.BadRequest(c, "HMAC verification failed")
		return
	}

	var payload PayMobWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		response.BadRequest(c, "invalid payload")
		return
	}

	txnID := payload.Object.ID
	slog.Info("PayMob webhook received",
		"type", payload.Type,
		"txn_id", txnID,
		"order_id", payload.Object.OrderID,
		"success", payload.Object.Success,
	)

	// Idempotency: check if we already processed this transaction
	result := h.db.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&ProcessedPayMobEvent{
			PayMobTxnID:  txnID,
			EventType:    payload.Type,
			ResponseCode: 200,
			ProcessedAt:  time.Now(),
		})
	if result.Error != nil {
		slog.Error("PayMob event dedup check failed", "error", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "dedup check failed"})
		return
	}

	// If no rows affected, event was already processed
	if result.RowsAffected == 0 {
		slog.Info("PayMob webhook already processed", "txn_id", txnID)
		c.JSON(http.StatusOK, gin.H{"status": "already_processed"})
		return
	}

	// Find the PayMob order
	var pmOrder PayMobOrder
	if err := h.db.Where("paymob_order_id = ?", payload.Object.OrderID).First(&pmOrder).Error; err != nil {
		slog.Error("PayMob order not found", "order_id", payload.Object.OrderID, "error", err)
		c.JSON(http.StatusOK, gin.H{"status": "order_not_found"})
		return
	}

	// Process based on type
	switch {
	case payload.Object.Success && !payload.Object.IsRefunded:
		// Payment succeeded
		pmOrder.Status = PaymentStatusSucceeded
		h.db.Save(&pmOrder)

		// Update the unified Payment record
		h.db.Model(&Payment{}).
			Where("stripe_payment_intent_id = ?", fmt.Sprintf("paymob_%d", pmOrder.PayMobOrderID)).
			Updates(map[string]interface{}{
				"status":         PaymentStatusSucceeded,
				"payment_method": "paymob",
			})

		// Create escrow for the payment
		escrow := EscrowAccount{
			PaymentID: getPaymentIDByPayMobOrder(h.db, pmOrder.PayMobOrderID),
			SellerID:  uuid.Nil, // will be set when order is linked
			BuyerID:   pmOrder.UserID,
			Amount:    float64(pmOrder.AmountCents) / 100.0,
			Currency:  pmOrder.Currency,
			Status:    EscrowStatusHeld,
		}
		h.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&escrow)

		metrics.IncWalletOp("paymob_success", "success")
		slog.Info("PayMob payment succeeded",
			"order_id", pmOrder.PayMobOrderID,
			"amount_cents", pmOrder.AmountCents,
		)

		// Publish domain event for in-process consumers
		events.Publish(events.Event{
			Type: events.EventPaymentCompleted,
			Payload: map[string]interface{}{
				"payment_id": escrow.PaymentID,
				"user_id":    pmOrder.UserID,
				"amount":     float64(pmOrder.AmountCents) / 100.0,
				"currency":   pmOrder.Currency,
				"provider":   "paymob",
			},
		})
		metrics.IncPaymentsProcessed()

		// Transactional outbox for Kafka delivery
		paymentID := getPaymentIDByPayMobOrder(h.db, pmOrder.PayMobOrderID)
		_ = kafka.WriteOutbox(h.db, kafka.TopicPayments, kafka.New(
			"payment.succeeded",
			paymentID.String(),
			"payment",
			kafka.Actor{Type: "user", ID: pmOrder.UserID.String()},
			map[string]interface{}{
				"payment_id": paymentID.String(),
				"user_id":    pmOrder.UserID.String(),
				"amount":     float64(pmOrder.AmountCents) / 100.0,
				"currency":   pmOrder.Currency,
				"provider":   "paymob",
			},
			kafka.EventMeta{Source: "webhook-service"},
		))

	case payload.Object.IsRefunded:
		pmOrder.Status = PaymentStatusRefunded
		h.db.Save(&pmOrder)
		h.db.Model(&Payment{}).
			Where("stripe_payment_intent_id = ?", fmt.Sprintf("paymob_%d", pmOrder.PayMobOrderID)).
			Update("status", PaymentStatusRefunded)
		metrics.IncWalletOp("paymob_refund", "success")

	case payload.Object.ErrorOccurred:
		pmOrder.Status = PaymentStatusFailed
		h.db.Save(&pmOrder)
		h.db.Model(&Payment{}).
			Where("stripe_payment_intent_id = ?", fmt.Sprintf("paymob_%d", pmOrder.PayMobOrderID)).
			Update("status", PaymentStatusFailed)
		metrics.IncWalletOp("paymob_failed", "error")
	}

	c.JSON(http.StatusOK, gin.H{"status": "processed"})
}

// getPaymentIDByPayMobOrder looks up the Payment UUID from the PayMob order ID.
func getPaymentIDByPayMobOrder(db *gorm.DB, paymobOrderID int64) uuid.UUID {
	var p Payment
	if err := db.Select("id").
		Where("stripe_payment_intent_id = ?", fmt.Sprintf("paymob_%d", paymobOrderID)).
		First(&p).Error; err != nil {
		return uuid.Nil
	}
	return p.ID
}

// GetPayMobPaymentStatus returns the status of a PayMob payment.
func (h *Handler) GetPayMobPaymentStatus(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid order id")
		return
	}

	var pmOrder PayMobOrder
	if err := h.db.Where("paymob_order_id = ?", orderID).First(&pmOrder).Error; err != nil {
		response.NotFound(c, "PayMob order")
		return
	}

	response.OK(c, gin.H{
		"paymob_order_id": pmOrder.PayMobOrderID,
		"status":          pmOrder.Status,
		"amount_cents":    pmOrder.AmountCents,
		"currency":        pmOrder.Currency,
	})
}
