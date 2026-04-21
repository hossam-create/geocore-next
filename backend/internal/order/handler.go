package order

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/geocore-next/backend/internal/referral"
	"github.com/geocore-next/backend/pkg/events"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler provides HTTP handlers for order operations
type Handler struct {
	repo *Repository
}

// NewHandler creates a new order handler
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateOrderRequest represents a request to create an order
// Typically called from payment webhook after successful payment
type CreateOrderRequest struct {
	PaymentIntentID string         `json:"payment_intent_id" binding:"required"`
	BuyerID         uuid.UUID      `json:"buyer_id"`
	SellerID        uuid.UUID      `json:"seller_id" binding:"required"`
	Items           []OrderItemReq `json:"items" binding:"required,min=1"`
	Subtotal        float64        `json:"subtotal" binding:"required,gt=0"`
	PlatformFee     float64        `json:"platform_fee"`
	PaymentFee      float64        `json:"payment_fee"`
	Total           float64        `json:"total" binding:"required,gt=0"`
	Currency        string         `json:"currency" default:"AED"`
	ShippingAddress *Address       `json:"shipping_address"`
	Notes           string         `json:"notes"`
	IsGuest         bool           `json:"is_guest"`
	GuestEmail      string         `json:"guest_email"`
	GuestFirstName  string         `json:"guest_first_name"`
	GuestLastName   string         `json:"guest_last_name"`
	GuestPhone      string         `json:"guest_phone"`
	DeliveryType    DeliveryType   `json:"delivery_type"`
}

type CreateGuestOrderRequest struct {
	PaymentIntentID string         `json:"payment_intent_id" binding:"required"`
	SellerID        uuid.UUID      `json:"seller_id" binding:"required"`
	Items           []OrderItemReq `json:"items" binding:"required,min=1"`
	Subtotal        float64        `json:"subtotal" binding:"required,gt=0"`
	PlatformFee     float64        `json:"platform_fee"`
	PaymentFee      float64        `json:"payment_fee"`
	Total           float64        `json:"total" binding:"required,gt=0"`
	Currency        string         `json:"currency" default:"AED"`
	ShippingAddress *Address       `json:"shipping_address"`
	Notes           string         `json:"notes"`
	GuestEmail      string         `json:"guest_email" binding:"required,email"`
	GuestFirstName  string         `json:"guest_first_name" binding:"required"`
	GuestLastName   string         `json:"guest_last_name"`
	GuestPhone      string         `json:"guest_phone" binding:"required"`
	DeliveryType    DeliveryType   `json:"delivery_type"`
}

// OrderItemReq represents an item in the create order request
type OrderItemReq struct {
	ListingID  *uuid.UUID             `json:"listing_id"`
	AuctionID  *uuid.UUID             `json:"auction_id"`
	Title      string                 `json:"title" binding:"required"`
	Quantity   int                    `json:"quantity" binding:"required,min=1"`
	UnitPrice  float64                `json:"unit_price" binding:"required,gt=0"`
	Condition  string                 `json:"condition"`
	Attributes map[string]interface{} `json:"attributes"`
}

// CreateOrder creates a new order (typically called by payment webhook)
func (h *Handler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.IsGuest {
		if req.GuestEmail == "" || req.GuestFirstName == "" || req.GuestPhone == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "guest_email, guest_first_name and guest_phone are required for guest orders"})
			return
		}
	} else if req.BuyerID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "buyer_id is required for non-guest orders"})
		return
	}

	if req.DeliveryType == "" {
		req.DeliveryType = DeliveryTypeStandard
	}

	// Calculate total prices for items
	items := make([]OrderItem, len(req.Items))
	for i, itemReq := range req.Items {
		items[i] = OrderItem{
			ListingID:  itemReq.ListingID,
			AuctionID:  itemReq.AuctionID,
			Title:      itemReq.Title,
			Quantity:   itemReq.Quantity,
			UnitPrice:  itemReq.UnitPrice,
			TotalPrice: itemReq.UnitPrice * float64(itemReq.Quantity),
			Condition:  itemReq.Condition,
			Attributes: itemReq.Attributes,
		}
	}

	var guestToken *uuid.UUID
	var guestEmail, guestFirstName, guestLastName, guestPhone, guestFingerprintHash *string
	if req.IsGuest {
		tok := uuid.New()
		guestToken = &tok
		fp := buildGuestTokenFingerprintHash(c, tok)
		guestFingerprintHash = &fp
		guestEmail = &req.GuestEmail
		guestFirstName = &req.GuestFirstName
		if req.GuestLastName != "" {
			guestLastName = &req.GuestLastName
		}
		guestPhone = &req.GuestPhone
	}

	order := Order{
		PaymentIntentID:           req.PaymentIntentID,
		BuyerID:                   req.BuyerID,
		SellerID:                  req.SellerID,
		Status:                    StatusPending,
		Items:                     items,
		Subtotal:                  req.Subtotal,
		PlatformFee:               req.PlatformFee,
		PaymentFee:                req.PaymentFee,
		Total:                     req.Total,
		Currency:                  req.Currency,
		ShippingAddress:           req.ShippingAddress,
		Notes:                     req.Notes,
		IsGuestOrder:              req.IsGuest,
		GuestEmail:                guestEmail,
		GuestFirstName:            guestFirstName,
		GuestLastName:             guestLastName,
		GuestPhone:                guestPhone,
		GuestToken:                guestToken,
		GuestTokenFingerprintHash: guestFingerprintHash,
		DeliveryType:              req.DeliveryType,
	}

	if err := h.repo.CreateWithOutbox(c.Request.Context(), &order, c.GetString("request_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}
	if req.IsGuest {
		metrics.IncGuestOrdersTotal()
	}
	metrics.IncOrdersCreated()

	// Publish domain event
	events.Publish(events.Event{
		Type:      events.EventOrderCreated,
		RequestID: c.GetString("request_id"),
		Payload: map[string]interface{}{
			"order_id":  order.ID,
			"buyer_id":  order.BuyerID,
			"seller_id": order.SellerID,
			"total":     order.Total,
			"currency":  order.Currency,
		},
	})

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Order created successfully",
		"data":        order,
		"guest_token": guestToken,
	})
}

// CreateGuestOrder creates guest checkout order without auth middleware.
func (h *Handler) CreateGuestOrder(c *gin.Context) {
	var req CreateGuestOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.DeliveryType == "" {
		req.DeliveryType = DeliveryTypeStandard
	}

	guestToken := uuid.New()
	shim := CreateOrderRequest{
		PaymentIntentID: req.PaymentIntentID,
		BuyerID:         uuid.New(),
		SellerID:        req.SellerID,
		Items:           req.Items,
		Subtotal:        req.Subtotal,
		PlatformFee:     req.PlatformFee,
		PaymentFee:      req.PaymentFee,
		Total:           req.Total,
		Currency:        req.Currency,
		ShippingAddress: req.ShippingAddress,
		Notes:           req.Notes,
		IsGuest:         true,
		GuestEmail:      req.GuestEmail,
		GuestFirstName:  req.GuestFirstName,
		GuestLastName:   req.GuestLastName,
		GuestPhone:      req.GuestPhone,
		DeliveryType:    req.DeliveryType,
	}

	items := make([]OrderItem, len(shim.Items))
	for i, itemReq := range shim.Items {
		items[i] = OrderItem{
			ListingID:  itemReq.ListingID,
			AuctionID:  itemReq.AuctionID,
			Title:      itemReq.Title,
			Quantity:   itemReq.Quantity,
			UnitPrice:  itemReq.UnitPrice,
			TotalPrice: itemReq.UnitPrice * float64(itemReq.Quantity),
			Condition:  itemReq.Condition,
			Attributes: itemReq.Attributes,
		}
	}

	guestEmail := shim.GuestEmail
	guestFirstName := shim.GuestFirstName
	guestLastName := shim.GuestLastName
	guestPhone := shim.GuestPhone
	fingerprintHash := buildGuestTokenFingerprintHash(c, guestToken)
	order := Order{
		PaymentIntentID:           shim.PaymentIntentID,
		BuyerID:                   shim.BuyerID,
		SellerID:                  shim.SellerID,
		Status:                    StatusPending,
		Items:                     items,
		Subtotal:                  shim.Subtotal,
		PlatformFee:               shim.PlatformFee,
		PaymentFee:                shim.PaymentFee,
		Total:                     shim.Total,
		Currency:                  shim.Currency,
		ShippingAddress:           shim.ShippingAddress,
		Notes:                     shim.Notes,
		IsGuestOrder:              true,
		GuestEmail:                &guestEmail,
		GuestFirstName:            &guestFirstName,
		GuestLastName:             &guestLastName,
		GuestPhone:                &guestPhone,
		GuestToken:                &guestToken,
		GuestTokenFingerprintHash: &fingerprintHash,
		DeliveryType:              shim.DeliveryType,
	}

	if err := h.repo.CreateWithOutbox(c.Request.Context(), &order, c.GetString("request_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create guest order"})
		return
	}
	metrics.IncGuestOrdersTotal()

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Guest order created successfully",
		"data":        order,
		"guest_token": guestToken,
	})
}

func getActorUUID(c *gin.Context) (uuid.UUID, bool) {
	if raw, ok := c.Get("userID"); ok {
		if id, ok := raw.(uuid.UUID); ok {
			return id, true
		}
	}
	if raw := c.GetString("user_id"); raw != "" {
		if id, err := uuid.Parse(raw); err == nil {
			return id, true
		}
	}
	return uuid.Nil, false
}

// GetOrder retrieves a single order by ID
func (h *Handler) GetOrder(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	order, err := h.repo.GetByID(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order"})
		return
	}
	if order == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.IsGuestOrder {
		headToken := c.GetHeader("X-Guest-Token")
		if headToken == "" || order.GuestToken == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "guest token required"})
			return
		}
		provided, parseErr := uuid.Parse(headToken)
		if parseErr != nil || provided != *order.GuestToken {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid guest token"})
			return
		}
		if order.GuestTokenFingerprintHash != nil && *order.GuestTokenFingerprintHash != "" {
			currentFP := buildGuestTokenFingerprintHash(c, provided)
			if currentFP != *order.GuestTokenFingerprintHash {
				c.JSON(http.StatusForbidden, gin.H{"error": "guest token fingerprint mismatch"})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"data": order})
		return
	}

	userID, exists := getActorUUID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Verify user is either buyer or seller
	if order.BuyerID != userID && order.SellerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": order})
}

func buildGuestTokenFingerprintHash(c *gin.Context, token uuid.UUID) string {
	raw := c.ClientIP() + "|" + c.GetHeader("User-Agent") + "|" + token.String()
	digest := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(digest[:])
}

// ListBuyerOrders returns paginated orders for the authenticated buyer
func (h *Handler) ListBuyerOrders(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	orders, total, err := h.repo.ListByBuyer(c.Request.Context(), userID.(uuid.UUID), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": orders,
		"pagination": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// ListSellerOrders returns paginated orders for the authenticated seller
func (h *Handler) ListSellerOrders(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	orders, total, err := h.repo.ListBySeller(c.Request.Context(), userID.(uuid.UUID), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": orders,
		"pagination": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// ConfirmOrderRequest represents a request to confirm an order
type ConfirmOrderRequest struct {
	Note string `json:"note"`
}

// ConfirmOrder allows seller to confirm an order
func (h *Handler) ConfirmOrder(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	order, err := h.repo.GetByID(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order"})
		return
	}
	if order == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Only seller can confirm
	if order.SellerID != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only seller can confirm this order"})
		return
	}

	// Validate status transition
	if !CanTransition(order.Status, StatusConfirmed) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot confirm order in current status: " + string(order.Status)})
		return
	}

	var req ConfirmOrderRequest
	c.ShouldBindJSON(&req)

	if err := h.repo.UpdateStatus(c.Request.Context(), orderID, StatusConfirmed, userID.(uuid.UUID).String(), req.Note); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order confirmed successfully"})
}

// ShipOrderRequest represents a request to mark order as shipped
type ShipOrderRequest struct {
	TrackingNumber string `json:"tracking_number" binding:"required"`
	Carrier        string `json:"carrier" binding:"required"`
}

// ShipOrder allows seller to mark order as shipped
func (h *Handler) ShipOrder(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var req ShipOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.repo.GetByID(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order"})
		return
	}
	if order == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Only seller can ship
	if order.SellerID != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only seller can ship this order"})
		return
	}

	// Validate status transition
	if !CanTransition(order.Status, StatusShipped) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot ship order in current status: " + string(order.Status)})
		return
	}

	if err := h.repo.UpdateShipping(c.Request.Context(), orderID, req.TrackingNumber, req.Carrier); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update shipping"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Order marked as shipped",
		"tracking_number": req.TrackingNumber,
		"carrier":         req.Carrier,
	})
}

// DeliverOrderRequest represents a request to mark order as delivered
type DeliverOrderRequest struct {
	Note string `json:"note"`
}

// DeliverOrder allows buyer to confirm delivery (triggers escrow release)
func (h *Handler) DeliverOrder(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	order, err := h.repo.GetByID(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order"})
		return
	}
	if order == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Only buyer can confirm delivery
	if order.BuyerID != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only buyer can confirm delivery"})
		return
	}

	// Validate status transition
	if !CanTransition(order.Status, StatusCompleted) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot confirm delivery in current status: " + string(order.Status)})
		return
	}

	var req DeliverOrderRequest
	c.ShouldBindJSON(&req)

	// First mark as delivered
	if err := h.repo.UpdateStatus(c.Request.Context(), orderID, StatusDelivered, userID.(uuid.UUID).String(), req.Note); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
		return
	}

	// Then complete the order (in production, this would trigger escrow release job)
	if err := h.repo.UpdateStatus(c.Request.Context(), orderID, StatusCompleted, "system", "Delivery confirmed by buyer"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete order"})
		return
	}

	// Complete referral if this is the buyer's first completed order
	go referral.CompleteReferral(h.repo.db, order.BuyerID)

	if order.PaymentIntentID != "" {
		now := time.Now()
		if err := h.repo.db.
			Table("escrow_accounts AS ea").
			Joins("JOIN payments p ON p.id = ea.payment_id").
			Where("p.stripe_payment_intent_id = ? AND ea.status = ?", order.PaymentIntentID, "held").
			Updates(map[string]interface{}{
				"status":      "released",
				"released_at": now,
				"updated_at":  now,
				"notes":       "Released after buyer delivery confirmation",
			}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Order completed but failed to release escrow"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Order delivered and completed. Funds will be released to seller.",
	})
}

// CancelOrderRequest represents a request to cancel an order
type CancelOrderRequest struct {
	Reason string `json:"reason"`
}

// CancelOrder allows cancellation if order is still pending
func (h *Handler) CancelOrder(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var req CancelOrderRequest
	c.ShouldBindJSON(&req)

	order, err := h.repo.GetByID(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order"})
		return
	}
	if order == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Both buyer and seller can cancel, but only in certain statuses
	currentUserID := userID.(uuid.UUID)
	if order.BuyerID != currentUserID && order.SellerID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Validate status transition
	if !CanTransition(order.Status, StatusCancelled) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot cancel order in current status: " + string(order.Status)})
		return
	}

	if err := h.repo.UpdateStatus(c.Request.Context(), orderID, StatusCancelled, currentUserID.String(), req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel order"})
		return
	}

	if order.PaymentIntentID != "" {
		var payment struct {
			ID     uuid.UUID
			Status string
		}
		if err := h.repo.db.Table("payments").
			Select("id, status").
			Where("stripe_payment_intent_id = ?", order.PaymentIntentID).
			First(&payment).Error; err == nil {
			now := time.Now()
			switch payment.Status {
			case "succeeded":
				h.repo.db.Table("payments").
					Where("id = ?", payment.ID).
					Updates(map[string]interface{}{"status": "refunded", "refunded_at": now, "updated_at": now})
				h.repo.db.Table("escrow_accounts").
					Where("payment_id = ?", payment.ID).
					Updates(map[string]interface{}{"status": "refunded", "updated_at": now})
			case "pending":
				h.repo.db.Table("payments").
					Where("id = ?", payment.ID).
					Updates(map[string]interface{}{"status": "cancelled", "updated_at": now})
			}
		}
	}

	// Transactional outbox: order.cancelled event
	_ = kafka.WriteOutbox(h.repo.DB(), kafka.TopicOrders, kafka.New(
		"order.cancelled",
		orderID.String(),
		"order",
		kafka.Actor{Type: "user", ID: currentUserID.String()},
		map[string]interface{}{
			"order_id":  orderID.String(),
			"buyer_id":  order.BuyerID.String(),
			"seller_id": order.SellerID.String(),
			"reason":    req.Reason,
		},
		kafka.EventMeta{Source: "api-service"},
	))

	c.JSON(http.StatusOK, gin.H{"message": "Order cancelled successfully"})
}
