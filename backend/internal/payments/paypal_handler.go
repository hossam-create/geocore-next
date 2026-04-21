package payments

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CreatePayPalOrderReq struct {
	ListingID   *string `json:"listing_id"`
	AuctionID   *string `json:"auction_id"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
	ReturnURL   string  `json:"return_url"`
	CancelURL   string  `json:"cancel_url"`
}

type CapturePayPalOrderReq struct {
	OrderID string `json:"order_id" binding:"required"`
}

func (h *Handler) CreatePayPalOrder(c *gin.Context) {
	var req CreatePayPalOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if req.ListingID == nil && req.AuctionID == nil {
		response.BadRequest(c, "either listing_id or auction_id is required")
		return
	}
	if req.ListingID != nil && req.AuctionID != nil {
		response.BadRequest(c, "only one of listing_id or auction_id may be provided")
		return
	}

	buyerID := c.GetString("user_id")
	var buyer users.User
	if err := h.db.First(&buyer, "id = ?", buyerID).Error; err != nil {
		response.NotFound(c, "user")
		return
	}

	var amount float64
	var sellerID uuid.UUID
	var listingID *uuid.UUID
	var auctionID *uuid.UUID
	var desc string
	var paymentKind PaymentKind

	if req.AuctionID != nil {
		id, err := uuid.Parse(*req.AuctionID)
		if err != nil {
			response.BadRequest(c, "invalid auction_id")
			return
		}
		var auction auctionRef
		if err := h.db.Table("auctions").
			Select("id, listing_id, seller_id, winner_id, current_bid, start_price, status").
			Where("id = ? AND deleted_at IS NULL", id).
			First(&auction).Error; err != nil {
			response.NotFound(c, "auction")
			return
		}
		if auction.Status != "ended" && auction.Status != "sold" {
			response.BadRequest(c, "auction payment requires auction to be ended")
			return
		}
		if auction.WinnerID == nil || auction.WinnerID.String() != buyerID {
			response.Forbidden(c)
			return
		}

		sellerID = auction.SellerID
		if auction.CurrentBid > 0 {
			amount = auction.CurrentBid
		} else {
			amount = auction.StartPrice
		}
		auctionID = &id
		lid := auction.ListingID
		listingID = &lid
		desc = fmt.Sprintf("PayPal auction payment (%s)", id.String())
		paymentKind = PaymentKindAuctionPayment
	} else {
		id, err := uuid.Parse(*req.ListingID)
		if err != nil {
			response.BadRequest(c, "invalid listing_id")
			return
		}
		var listing listingRef
		if err := h.db.Table("listings").
			Select("id, user_id, price").
			Where("id = ? AND deleted_at IS NULL", id).
			First(&listing).Error; err != nil {
			response.NotFound(c, "listing")
			return
		}
		if listing.Price == nil || *listing.Price <= 0 {
			response.BadRequest(c, "listing price not available")
			return
		}

		sellerID = listing.UserID
		amount = *listing.Price
		listingID = &id
		desc = fmt.Sprintf("PayPal purchase (%s)", id.String())
		paymentKind = PaymentKindPurchase
	}

	if sellerID.String() == buyerID {
		response.BadRequest(c, "buyer and seller cannot be the same user")
		return
	}

	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "AED"
	}
	if req.Description != "" {
		desc = req.Description
	}

	orderResult, err := createPayPalOrder(amount, currency, desc, req.ReturnURL, req.CancelURL)
	if err != nil {
		slog.Error("paypal create order failed", "user_id", buyer.ID.String(), "error", err.Error())
		response.InternalError(c, err)
		return
	}

	payment := Payment{
		UserID:                buyer.ID,
		ListingID:             listingID,
		AuctionID:             auctionID,
		Kind:                  paymentKind,
		StripePaymentIntentID: orderResult.OrderID,
		Amount:                amount,
		Currency:              currency,
		Status:                PaymentStatusPending,
		PaymentMethod:         "paypal",
		Description:           desc,
	}
	if err := h.db.Create(&payment).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	response.Created(c, gin.H{
		"payment_id":     payment.ID,
		"order_id":       orderResult.OrderID,
		"approval_url":   orderResult.ApprovalURL,
		"amount":         amount,
		"currency":       currency,
		"payment_method": "paypal",
	})
}

func (h *Handler) CapturePayPalOrder(c *gin.Context) {
	var req CapturePayPalOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	buyerID := c.GetString("user_id")
	var payment Payment
	if err := h.db.Where("stripe_payment_intent_id = ? AND user_id = ?", req.OrderID, buyerID).First(&payment).Error; err != nil {
		response.NotFound(c, "payment")
		return
	}
	if payment.Status == PaymentStatusSucceeded {
		response.OK(c, gin.H{"payment_id": payment.ID, "status": payment.Status, "order_id": req.OrderID})
		return
	}

	captureResult, err := capturePayPalOrder(req.OrderID)
	if err != nil {
		slog.Error("paypal capture failed", "order_id", req.OrderID, "error", err.Error())
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.finalizePayPalPayment(req.OrderID); err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, gin.H{
		"payment_id":    payment.ID,
		"order_id":      captureResult.OrderID,
		"capture_id":    captureResult.CaptureID,
		"status":        PaymentStatusSucceeded,
		"paypal_status": captureResult.Status,
	})
}

func (h *Handler) PayPalWebhook(c *gin.Context) {
	payload, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		slog.Error("paypal webhook: failed to read body", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid payload"})
		return
	}

	headers := payPalWebhookHeaders{
		TransmissionID:   c.GetHeader("Paypal-Transmission-Id"),
		TransmissionTime: c.GetHeader("Paypal-Transmission-Time"),
		CertURL:          c.GetHeader("Paypal-Cert-Url"),
		AuthAlgo:         c.GetHeader("Paypal-Auth-Algo"),
		TransmissionSig:  c.GetHeader("Paypal-Transmission-Sig"),
	}
	if err := verifyPayPalWebhook(headers, payload); err != nil {
		slog.Warn("paypal webhook: signature verification failed", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"message": "webhook verification failed"})
		return
	}

	var event struct {
		EventType string         `json:"event_type"`
		Resource  map[string]any `json:"resource"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		slog.Error("paypal webhook: invalid event json", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid event"})
		return
	}

	switch event.EventType {
	case "CHECKOUT.ORDER.APPROVED":
		if orderID := extractPayPalOrderID(event.Resource); orderID != "" {
			if err := h.finalizePayPalPayment(orderID); err != nil {
				slog.Error("paypal webhook: finalize approved order failed", "order_id", orderID, "error", err.Error())
			}
		}
	case "PAYMENT.CAPTURE.COMPLETED":
		if orderID := extractPayPalOrderID(event.Resource); orderID != "" {
			if err := h.finalizePayPalPayment(orderID); err != nil {
				slog.Error("paypal webhook: finalize captured order failed", "order_id", orderID, "error", err.Error())
			}
		}
	default:
		slog.Info("paypal webhook: unhandled event", "event_type", event.EventType)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) finalizePayPalPayment(orderID string) error {
	var payment Payment
	if err := h.db.Where("stripe_payment_intent_id = ?", orderID).First(&payment).Error; err != nil {
		return err
	}
	if payment.Status == PaymentStatusSucceeded {
		return nil
	}

	sellerID, err := h.resolveSellerIDForPayment(&payment)
	if err != nil {
		return err
	}

	now := time.Now()
	tx := h.db.Begin()
	if err := tx.Model(&payment).Updates(map[string]any{
		"status":         PaymentStatusSucceeded,
		"payment_method": "paypal",
		"updated_at":     now,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	escrow := EscrowAccount{
		PaymentID: payment.ID,
		SellerID:  sellerID,
		BuyerID:   payment.UserID,
		Amount:    payment.Amount,
		Currency:  payment.Currency,
		Status:    EscrowStatusHeld,
	}
	if err := tx.Where("payment_id = ?", payment.ID).FirstOrCreate(&escrow).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func extractPayPalOrderID(resource map[string]any) string {
	if resource == nil {
		return ""
	}
	if id, ok := resource["id"].(string); ok && strings.TrimSpace(id) != "" {
		return id
	}
	if supp, ok := resource["supplementary_data"].(map[string]any); ok {
		if rel, ok := supp["related_ids"].(map[string]any); ok {
			if id, ok := rel["order_id"].(string); ok {
				return id
			}
		}
	}
	return ""
}

func (h *Handler) resolveSellerIDForPayment(payment *Payment) (uuid.UUID, error) {
	if payment.AuctionID != nil {
		var auction struct {
			SellerID uuid.UUID
		}
		if err := h.db.Table("auctions").Select("seller_id").Where("id = ?", *payment.AuctionID).First(&auction).Error; err != nil {
			return uuid.Nil, err
		}
		return auction.SellerID, nil
	}
	if payment.ListingID != nil {
		var listing struct {
			UserID uuid.UUID
		}
		if err := h.db.Table("listings").Select("user_id").Where("id = ?", *payment.ListingID).First(&listing).Error; err != nil {
			return uuid.Nil, err
		}
		return listing.UserID, nil
	}
	return uuid.Nil, fmt.Errorf("payment missing listing_id and auction_id")
}
