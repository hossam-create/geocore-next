package payments

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/fraud"
	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/internal/order"
	"github.com/geocore-next/backend/internal/security"
	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ════════════════════════════════════════════════════════════════════════════
// Handler
// ════════════════════════════════════════════════════════════════════════════

type Handler struct {
	db       *gorm.DB
	rdb      *redis.Client
	fx       *FXService
	notifSvc *notifications.Service
}

func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
	return &Handler{db: db, rdb: rdb, fx: NewFXService(db, rdb)}
}

func NewHandlerWithNotifications(db *gorm.DB, rdb *redis.Client, notifSvc *notifications.Service) *Handler {
	return &Handler{db: db, rdb: rdb, fx: NewFXService(db, rdb), notifSvc: notifSvc}
}

// ════════════════════════════════════════════════════════════════════════════
// Request types
// ════════════════════════════════════════════════════════════════════════════

type CreatePaymentIntentReq struct {
	ListingID   *string `json:"listing_id"`
	AuctionID   *string `json:"auction_id"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
}

type ConfirmPaymentReq struct {
	PaymentIntentID string `json:"payment_intent_id" binding:"required"`
}

type ReleaseEscrowReq struct {
	EscrowID string `json:"escrow_id" binding:"required"`
	Notes    string `json:"notes"`
}

type RefundReq struct {
	PaymentID string `json:"payment_id" binding:"required"`
	Reason    string `json:"reason"`
}

type AddPaymentMethodReq struct {
	PaymentMethodID string `json:"payment_method_id" binding:"required"`
	SetDefault      bool   `json:"set_default"`
}

// ════════════════════════════════════════════════════════════════════════════
// CreatePaymentIntent — POST /api/v1/payments/create-payment-intent
// ════════════════════════════════════════════════════════════════════════════

// Minimal structs for server-side price/seller resolution
type listingRef struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Price  *float64
}

type auctionRef struct {
	ID         uuid.UUID
	ListingID  uuid.UUID
	SellerID   uuid.UUID
	WinnerID   *uuid.UUID
	CurrentBid float64
	StartPrice float64
	Status     string
}

// CreatePaymentIntent creates a Stripe PaymentIntent and saves a pending Payment record.
// Amount and seller are derived server-side from the listing or auction — never trusted from client.
func (h *Handler) CreatePaymentIntent(c *gin.Context) {
	var req CreatePaymentIntentReq
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

	// ── Load buyer ────────────────────────────────────────────────────────────
	buyerID := c.GetString("user_id")
	var buyer users.User
	if err := h.db.First(&buyer, "id = ?", buyerID).Error; err != nil {
		response.NotFound(c, "user")
		return
	}

	// ── Derive authoritative amount and seller from listing or auction ─────────
	var authorAmount float64
	var sellerID uuid.UUID
	var listingID *uuid.UUID
	var auctionID *uuid.UUID
	var desc string
	var paymentKind PaymentKind

	if req.AuctionID != nil {
		id, e := uuid.Parse(*req.AuctionID)
		if e != nil {
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
		// Auction payments are only allowed for settled (ended) auctions
		if auction.Status != "ended" {
			response.BadRequest(c, "auction payment requires auction to be ended")
			return
		}
		// Only the winner may pay
		if auction.WinnerID == nil || auction.WinnerID.String() != buyerID {
			response.Forbidden(c)
			return
		}
		sellerID = auction.SellerID
		if auction.CurrentBid > 0 {
			authorAmount = auction.CurrentBid
		} else {
			authorAmount = auction.StartPrice
		}
		auctionID = &id
		lid := auction.ListingID
		listingID = &lid
		desc = fmt.Sprintf("Auction payment (auction %s)", id.String())
		paymentKind = PaymentKindAuctionPayment
	} else {
		id, e := uuid.Parse(*req.ListingID)
		if e != nil {
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
		authorAmount = *listing.Price
		listingID = &id
		desc = fmt.Sprintf("Purchase (listing %s)", id.String())
		paymentKind = PaymentKindPurchase
	}

	// ── Buyer cannot be the seller ────────────────────────────────────────────
	if sellerID.String() == buyerID {
		response.BadRequest(c, "buyer and seller cannot be the same user")
		return
	}

	// ── Fraud detection baseline (STEP 9) ───────────────────────────
	{
		var profile fraud.UserRiskProfile
		h.db.Where("user_id = ?", buyerID).First(&profile)
		accountAgeHours := time.Since(buyer.CreatedAt).Hours()
		risk := fraud.AnalyzeTransaction(authorAmount, profile.TotalOrders, profile.AvgOrderValue, accountAgeHours)
		if risk.RiskScore >= 80 {
			security.LogEvent(h.db, c, &buyer.ID, security.EventPaymentAttempt, map[string]any{
				"fraud_declined": true,
				"risk_score":     risk.RiskScore,
				"risk_level":     risk.RiskLevel,
				"amount":         authorAmount,
			})
			response.BadRequest(c, "transaction declined by fraud prevention")
			return
		}
		if risk.RiskScore >= 50 {
			security.LogEvent(h.db, c, &buyer.ID, security.EventPaymentAttempt, map[string]any{
				"fraud_flagged": true,
				"risk_score":    risk.RiskScore,
				"risk_level":    risk.RiskLevel,
				"amount":        authorAmount,
			})
		}
	}

	// ── Ensure buyer has a Stripe customer record ─────────────────────────────
	stripeCustomerID, err := h.ensureStripeCustomer(&buyer)
	if err != nil {
		slog.Error("failed to ensure Stripe customer",
			"user_id", buyer.ID.String(), "error", err.Error())
		response.InternalError(c, err)
		return
	}

	// ── Normalise currency ────────────────────────────────────────────────────
	currency := strings.ToLower(req.Currency)
	if currency == "" {
		currency = "aed"
	}

	// ── Build metadata ────────────────────────────────────────────────────────
	meta := map[string]string{
		"buyer_id":  buyer.ID.String(),
		"seller_id": sellerID.String(),
		"platform":  "geocore",
	}
	if listingID != nil {
		meta["listing_id"] = listingID.String()
	}
	if auctionID != nil {
		meta["auction_id"] = auctionID.String()
	}

	// ── Create Stripe PaymentIntent (skipped when Stripe is not configured) ──
	var piID, clientSecret string
	if os.Getenv("STRIPE_SECRET_KEY") != "" {
		pi, err := createPaymentIntent(authorAmount, currency, stripeCustomerID, desc, meta)
		if err != nil {
			slog.Error("Stripe: failed to create payment intent",
				"user_id", buyer.ID.String(), "amount", authorAmount, "error", err.Error())
			response.BadRequest(c, stripeErrMsg(err))
			return
		}
		piID = pi.ID
		clientSecret = pi.ClientSecret
	} else {
		// Non-Stripe dev environment: use a local placeholder ID
		piID = "local_" + uuid.New().String()
		slog.Warn("Stripe not configured — payment intent is local placeholder",
			"pi_id", piID, "user_id", buyer.ID.String())
	}

	// ── Persist pending payment record ────────────────────────────────────────
	payment := Payment{
		UserID:                buyer.ID,
		ListingID:             listingID,
		AuctionID:             auctionID,
		Kind:                  paymentKind,
		StripePaymentIntentID: piID,
		StripeClientSecret:    clientSecret,
		Amount:                authorAmount,
		Currency:              strings.ToUpper(currency),
		Status:                PaymentStatusPending,
		Description:           desc,
	}
	if err := h.db.Create(&payment).Error; err != nil {
		slog.Error("failed to save payment record",
			"stripe_pi", piID, "error", err.Error())
		response.InternalError(c, err)
		return
	}

	security.LogEvent(h.db, c, &buyer.ID, security.EventPaymentAttempt, map[string]any{
		"payment_id": payment.ID.String(),
		"kind":       string(paymentKind),
		"amount":     authorAmount,
		"currency":   strings.ToUpper(currency),
		"listing_id": func() string {
			if listingID != nil {
				return listingID.String()
			}
			return ""
		}(),
		"auction_id": func() string {
			if auctionID != nil {
				return auctionID.String()
			}
			return ""
		}(),
		"seller_id": sellerID.String(),
	})

	slog.Info("payment intent created",
		"payment_id", payment.ID.String(),
		"stripe_pi", piID,
		"amount", authorAmount,
		"currency", currency,
		"buyer_id", buyer.ID.String(),
	)

	response.Created(c, gin.H{
		"payment_id":        payment.ID,
		"payment_intent_id": piID,
		"client_secret":     clientSecret,
		"amount":            authorAmount,
		"currency":          strings.ToUpper(currency),
	})
}

// ════════════════════════════════════════════════════════════════════════════
// ConfirmPayment — POST /api/v1/payments/confirm
// ════════════════════════════════════════════════════════════════════════════

// ConfirmPayment checks the latest PaymentIntent status from Stripe.
// If the payment succeeded, it creates an EscrowAccount record and marks
// the payment as succeeded.
//
// Note: For a production system, status should primarily be updated via
// Stripe webhooks (Task 2.2).  This endpoint provides a fallback for clients
// that want to poll status after the Stripe.js confirmation flow.
func (h *Handler) ConfirmPayment(c *gin.Context) {
	var req ConfirmPaymentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// ── Load local payment record ─────────────────────────────────────────────
	var payment Payment
	if err := h.db.Where("stripe_payment_intent_id = ?", req.PaymentIntentID).
		First(&payment).Error; err != nil {
		response.NotFound(c, "payment")
		return
	}

	// Verify this payment belongs to the authenticated user
	buyerID := c.GetString("user_id")
	if payment.UserID.String() != buyerID {
		response.Forbidden(c)
		return
	}

	// If already processed, return current status
	if payment.Status == PaymentStatusSucceeded {
		response.OK(c, gin.H{"status": payment.Status, "payment_id": payment.ID})
		return
	}

	// ── Fetch latest status from Stripe ──────────────────────────────────────
	pi, err := retrievePaymentIntent(req.PaymentIntentID)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	switch pi.Status {
	case stripe.PaymentIntentStatusSucceeded:
		if err := h.handlePaymentSuccess(c, &payment, pi); err != nil {
			response.InternalError(c, err)
			return
		}
		response.OK(c, gin.H{
			"status":     "succeeded",
			"payment_id": payment.ID,
			"message":    "Payment successful. Funds are held in escrow.",
		})

	case stripe.PaymentIntentStatusRequiresAction:
		response.OK(c, gin.H{
			"status":        "requires_action",
			"client_secret": pi.ClientSecret,
			"message":       "Additional authentication required (3D Secure).",
		})

	case stripe.PaymentIntentStatusRequiresPaymentMethod:
		h.db.Model(&payment).Update("status", PaymentStatusFailed)
		response.BadRequest(c, "Payment failed. Please try again with a different payment method.")

	default:
		response.OK(c, gin.H{
			"status":  string(pi.Status),
			"message": "Payment is being processed.",
		})
	}
}

// ════════════════════════════════════════════════════════════════════════════
// ReleaseEscrow — POST /api/v1/payments/release-escrow
// ════════════════════════════════════════════════════════════════════════════

// ReleaseEscrow marks an escrow account as released.
// Only the buyer (the one who paid) can trigger a release.
// After release, the seller receives their funds (handled by Stripe Connect
// or manual payout — depending on the business model).
func (h *Handler) ReleaseEscrow(c *gin.Context) {
	var req ReleaseEscrowReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	req.Notes = security.SanitizeText(req.Notes)

	buyerID := c.GetString("user_id")
	buyerUUID, _ := uuid.Parse(buyerID)

	var escrow EscrowAccount
	var now time.Time

	// ── Atomic: lock + validate state + update in one transaction ──────────────
	// FOR UPDATE prevents two concurrent release requests both passing the
	// EscrowStatusHeld check and double-releasing the same escrow.
	dbErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Payment").First(&escrow, "id = ?", req.EscrowID).Error; err != nil {
			return fmt.Errorf("not_found")
		}
		if escrow.BuyerID.String() != buyerID {
			return fmt.Errorf("forbidden")
		}
		if escrow.Status != EscrowStatusHeld {
			return fmt.Errorf("already_%s", escrow.Status)
		}
		now = time.Now()
		return tx.Model(&escrow).Updates(map[string]any{
			"status":      EscrowStatusReleased,
			"released_at": now,
			"notes":       req.Notes,
		}).Error
	})

	if dbErr != nil {
		switch dbErr.Error() {
		case "not_found":
			response.NotFound(c, "escrow")
		case "forbidden":
			response.Forbidden(c)
		default:
			response.BadRequest(c, dbErr.Error())
		}
		return
	}

	if buyerUUID != uuid.Nil {
		security.LogEvent(h.db, c, &buyerUUID, security.EventEscrowReleased, map[string]any{
			"escrow_id":  escrow.ID.String(),
			"payment_id": escrow.PaymentID.String(),
			"seller_id":  escrow.SellerID.String(),
			"amount":     escrow.Amount,
			"currency":   escrow.Currency,
		})
	}
	slog.Info("escrow released",
		"escrow_id", escrow.ID.String(),
		"buyer_id", buyerID,
		"seller_id", escrow.SellerID.String(),
		"amount", escrow.Amount,
	)
	response.OK(c, gin.H{
		"escrow_id":   escrow.ID,
		"status":      EscrowStatusReleased,
		"released_at": now,
		"message":     "Funds released to seller.",
	})
}

// ════════════════════════════════════════════════════════════════════════════
// RequestRefund — POST /api/v1/payments/request-refund
// ════════════════════════════════════════════════════════════════════════════

// RequestRefund issues a full refund for a payment via Stripe.
// Only the buyer can request a refund, and only when escrow is still held.
func (h *Handler) RequestRefund(c *gin.Context) {
	var req RefundReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Sanitize free-text reason
	req.Reason = security.SanitizeText(req.Reason)

	buyerID := c.GetString("user_id")
	paymentUUID, err := uuid.Parse(req.PaymentID)
	if err != nil {
		response.BadRequest(c, "invalid payment_id")
		return
	}

	var payment Payment
	if err := h.db.Preload("Escrow").First(&payment, "id = ?", paymentUUID).Error; err != nil {
		response.NotFound(c, "payment")
		return
	}

	if payment.UserID.String() != buyerID {
		response.Forbidden(c)
		return
	}

	if payment.Status != PaymentStatusSucceeded {
		response.BadRequest(c, "only succeeded payments can be refunded")
		return
	}

	// Check escrow is still held (not released)
	if payment.Escrow != nil && payment.Escrow.Status != EscrowStatusHeld {
		response.BadRequest(c, "cannot refund: escrow funds have already been released to the seller")
		return
	}

	// ── Issue Stripe refund ───────────────────────────────────────────────────
	_, refundErr := issueRefund(payment.StripePaymentIntentID, nil)
	if refundErr != nil {
		slog.Error("Stripe refund failed",
			"payment_id", payment.ID.String(), "error", refundErr.Error())
		response.BadRequest(c, stripeErrMsg(refundErr))
		return
	}

	// ── Update local records ──────────────────────────────────────────────────
	now := time.Now()
	h.db.Model(&payment).Updates(map[string]any{
		"status":      PaymentStatusRefunded,
		"refunded_at": now,
	})
	if payment.Escrow != nil {
		h.db.Model(payment.Escrow).Update("status", EscrowStatusRefunded)
	}

	slog.Info("payment refunded",
		"payment_id", payment.ID.String(),
		"buyer_id", buyerID,
		"amount", payment.Amount,
	)

	response.OK(c, gin.H{
		"payment_id":  payment.ID,
		"status":      PaymentStatusRefunded,
		"refunded_at": now,
		"message":     "Refund initiated. It may take 5–10 business days to appear on your statement.",
	})
}

// ════════════════════════════════════════════════════════════════════════════
// GetPaymentMethods — GET /api/v1/payments/payment-methods
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) GetPaymentMethods(c *gin.Context) {
	buyerID := c.GetString("user_id")

	var user users.User
	if err := h.db.First(&user, "id = ?", buyerID).Error; err != nil {
		response.NotFound(c, "user")
		return
	}

	if user.StripeCustomerID == "" {
		response.OK(c, gin.H{"payment_methods": []gin.H{}})
		return
	}

	methods, err := listPaymentMethods(user.StripeCustomerID)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	// Also load saved methods from DB (includes is_default flag)
	var saved []SavedPaymentMethod
	h.db.Where("user_id = ?", buyerID).Find(&saved)
	savedMap := make(map[string]SavedPaymentMethod, len(saved))
	for _, s := range saved {
		savedMap[s.StripeMethodID] = s
	}

	out := make([]gin.H, 0, len(methods))
	for _, m := range methods {
		entry := gin.H{
			"id":         m.ID,
			"brand":      string(m.Card.Brand),
			"last4":      m.Card.Last4,
			"exp_month":  m.Card.ExpMonth,
			"exp_year":   m.Card.ExpYear,
			"is_default": false,
		}
		if db, ok := savedMap[m.ID]; ok {
			entry["is_default"] = db.IsDefault
		}
		out = append(out, entry)
	}

	response.OK(c, gin.H{"payment_methods": out})
}

// ════════════════════════════════════════════════════════════════════════════
// AddPaymentMethod — POST /api/v1/payments/add-payment-method
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) AddPaymentMethod(c *gin.Context) {
	var req AddPaymentMethodReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	buyerID := c.GetString("user_id")
	var user users.User
	if err := h.db.First(&user, "id = ?", buyerID).Error; err != nil {
		response.NotFound(c, "user")
		return
	}

	// Ensure Stripe customer exists
	stripeCustomerID, err := h.ensureStripeCustomer(&user)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	// Attach payment method to customer in Stripe
	pm, err := attachPaymentMethod(req.PaymentMethodID, stripeCustomerID)
	if err != nil {
		response.BadRequest(c, stripeErrMsg(err))
		return
	}

	// If set as default, unset previous defaults
	if req.SetDefault {
		h.db.Model(&SavedPaymentMethod{}).
			Where("user_id = ?", buyerID).
			Update("is_default", false)
	}

	// Upsert saved payment method in DB
	userUUID, _ := uuid.Parse(buyerID)
	savedPM := SavedPaymentMethod{
		UserID:         userUUID,
		StripeMethodID: pm.ID,
		Brand:          string(pm.Card.Brand),
		Last4:          pm.Card.Last4,
		ExpMonth:       int(pm.Card.ExpMonth),
		ExpYear:        int(pm.Card.ExpYear),
		IsDefault:      req.SetDefault,
	}
	h.db.Where("stripe_method_id = ?", pm.ID).FirstOrCreate(&savedPM)

	response.Created(c, gin.H{
		"id":         pm.ID,
		"brand":      string(pm.Card.Brand),
		"last4":      pm.Card.Last4,
		"exp_month":  pm.Card.ExpMonth,
		"exp_year":   pm.Card.ExpYear,
		"is_default": req.SetDefault,
	})
}

// ════════════════════════════════════════════════════════════════════════════
// DeletePaymentMethod — DELETE /api/v1/payments/payment-methods/:id
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) DeletePaymentMethod(c *gin.Context) {
	pmID := c.Param("id")
	buyerID := c.GetString("user_id")

	// Verify ownership in DB before detaching
	var saved SavedPaymentMethod
	if err := h.db.Where("stripe_method_id = ? AND user_id = ?", pmID, buyerID).
		First(&saved).Error; err != nil {
		response.NotFound(c, "payment method")
		return
	}

	if err := detachPaymentMethod(pmID); err != nil {
		response.BadRequest(c, stripeErrMsg(err))
		return
	}

	h.db.Delete(&saved)

	response.OK(c, gin.H{"message": "Payment method removed."})
}

// ════════════════════════════════════════════════════════════════════════════
// GetPaymentHistory — GET /api/v1/payments
// ════════════════════════════════════════════════════════════════════════════

// PaymentPublic is the safe outbound shape for payment history.
// StripePaymentIntentID and StripeClientSecret are intentionally excluded —
// internal Stripe references must not leak to callers.
type PaymentPublic struct {
	ID            uuid.UUID      `json:"id"`
	Kind          PaymentKind    `json:"kind"`
	Amount        float64        `json:"amount"`
	Currency      string         `json:"currency"`
	Status        PaymentStatus  `json:"status"`
	Description   string         `json:"description,omitempty"`
	FailureReason string         `json:"failure_reason,omitempty"`
	RefundedAt    *time.Time     `json:"refunded_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	Escrow        *EscrowAccount `json:"escrow,omitempty"`
}

func toPublic(p Payment) PaymentPublic {
	return PaymentPublic{
		ID: p.ID, Kind: p.Kind, Amount: p.Amount, Currency: p.Currency,
		Status: p.Status, Description: p.Description, FailureReason: p.FailureReason,
		RefundedAt: p.RefundedAt, CreatedAt: p.CreatedAt, Escrow: p.Escrow,
	}
}

func (h *Handler) GetPaymentHistory(c *gin.Context) {
	buyerID := c.GetString("user_id")

	var payments []Payment
	query := h.db.Where("user_id = ?", buyerID).
		Preload("Escrow").
		Order("created_at DESC")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Model(&Payment{}).Count(&total)

	page, perPage := paginationParams(c)
	query.Offset((page - 1) * perPage).Limit(perPage).Find(&payments)

	pub := make([]PaymentPublic, len(payments))
	for i, p := range payments {
		pub[i] = toPublic(p)
	}

	response.OKMeta(c, pub, response.Meta{
		Total:   total,
		Page:    page,
		PerPage: perPage,
		Pages:   (total + int64(perPage) - 1) / int64(perPage),
	})
}

// ════════════════════════════════════════════════════════════════════════════
// Internal helpers
// ════════════════════════════════════════════════════════════════════════════

// ensureStripeCustomer gets or creates a Stripe customer for the user.
// If the user already has a stripe_customer_id, it returns it directly.
// Returns empty string (no error) when Stripe is not configured so callers
// can degrade gracefully rather than 500-ing in non-Stripe environments.
func (h *Handler) ensureStripeCustomer(user *users.User) (string, error) {
	if user.StripeCustomerID != "" {
		return user.StripeCustomerID, nil
	}
	// Stripe not configured — return empty string without error so callers
	// can fall through to the local-ID / no-Stripe path
	if os.Getenv("STRIPE_SECRET_KEY") == "" {
		return "", nil
	}

	custID, err := createStripeCustomer(user.Email, user.Name, user.Phone)
	if err != nil {
		return "", err
	}

	if err := h.db.Model(user).Update("stripe_customer_id", custID).Error; err != nil {
		slog.Warn("saved Stripe customer ID to DB failed",
			"user_id", user.ID.String(), "cust_id", custID)
	}
	user.StripeCustomerID = custID
	return custID, nil
}

// handlePaymentSuccess transitions a payment to succeeded.
// For wallet top-ups (type=wallet_topup in PI metadata) no escrow is created.
// For regular purchases/auction wins, escrow is created with seller from PI metadata.
// Called both from ConfirmPayment and from the webhook handler.
func (h *Handler) handlePaymentSuccess(c *gin.Context, payment *Payment, pi *stripe.PaymentIntent) error {
	if payment.Status == PaymentStatusSucceeded {
		return nil // idempotent — already processed
	}

	// Mark payment as succeeded
	if err := h.db.Model(payment).Updates(map[string]any{
		"status":         PaymentStatusSucceeded,
		"payment_method": "card",
	}).Error; err != nil {
		return err
	}

	// Wallet top-ups: no escrow needed
	if pi.Metadata["type"] == "wallet_topup" {
		slog.Info("wallet top-up succeeded",
			"payment_id", payment.ID.String(),
			"amount", payment.Amount,
			"currency", payment.Currency,
		)
		return nil
	}

	// Regular purchase: parse seller and create escrow
	sellerIDStr := pi.Metadata["seller_id"]
	sellerUUID, err := uuid.Parse(sellerIDStr)
	if err != nil {
		return fmt.Errorf("invalid seller_id in payment intent metadata")
	}

	escrow := EscrowAccount{
		PaymentID: payment.ID,
		SellerID:  sellerUUID,
		BuyerID:   payment.UserID,
		Amount:    payment.Amount,
		Currency:  payment.Currency,
		Status:    EscrowStatusHeld,
	}
	if err := h.db.Where("payment_id = ?", payment.ID).
		FirstOrCreate(&escrow).Error; err != nil {
		return err
	}

	slog.Info("payment succeeded, escrow created",
		"payment_id", payment.ID.String(),
		"escrow_id", escrow.ID.String(),
		"amount", payment.Amount,
		"currency", payment.Currency,
	)

	// Create the order record for this payment
	if err := h.createOrderFromPayment(payment, sellerUUID); err != nil {
		slog.Error("failed to create order from payment",
			"payment_id", payment.ID.String(), "error", err.Error())
		// Non-fatal: order can be reconciled later
	}

	return nil
}

// createOrderFromPayment creates an Order record after a successful payment.
// It is idempotent: if an order for this payment_intent_id already exists it is a no-op.
func (h *Handler) createOrderFromPayment(payment *Payment, sellerID uuid.UUID) error {
	// Idempotency check
	var existing order.Order
	if err := h.db.Where("payment_intent_id = ?", payment.StripePaymentIntentID).
		First(&existing).Error; err == nil {
		return nil
	}

	// Resolve listing title for the order item snapshot
	title := payment.Description
	if payment.ListingID != nil {
		var row struct{ Title string }
		if err := h.db.Table("listings").Select("title").
			Where("id = ?", payment.ListingID).First(&row).Error; err == nil {
			title = row.Title
		}
	} else if payment.AuctionID != nil {
		var row struct{ Title string }
		if err := h.db.Table("listings").
			Select("listings.title").
			Joins("JOIN auctions ON auctions.listing_id = listings.id").
			Where("auctions.id = ?", payment.AuctionID).
			First(&row).Error; err == nil {
			title = row.Title
		}
	}

	o := order.Order{
		BuyerID:         payment.UserID,
		SellerID:        sellerID,
		PaymentIntentID: payment.StripePaymentIntentID,
		PaymentID:       &payment.ID,
		Status:          order.StatusPending,
		Subtotal:        payment.Amount,
		Total:           payment.Amount,
		Currency:        payment.Currency,
		Items: []order.OrderItem{
			{
				ListingID:  payment.ListingID,
				AuctionID:  payment.AuctionID,
				Title:      title,
				Quantity:   1,
				UnitPrice:  payment.Amount,
				TotalPrice: payment.Amount,
			},
		},
	}

	if err := h.db.Create(&o).Error; err != nil {
		return fmt.Errorf("createOrderFromPayment: %w", err)
	}

	slog.Info("order created from payment",
		"order_id", o.ID.String(),
		"payment_id", payment.ID.String(),
	)
	return nil
}

// ════════════════════════════════════════════════════════════════════════════
// GetWalletBalance — GET /api/v1/wallet/balance
// ════════════════════════════════════════════════════════════════════════════

// GetWalletBalance returns the user's wallet balance.
// In this architecture, purchases and auction payments are card-charged separately —
// they do NOT debit the wallet. The wallet is a top-up ledger only:
//
//	balance = sum of succeeded wallet_topup payments
//
// This gives an unambiguous wallet balance independent of purchase history.
// total_card_spent is reported separately for user information.
func (h *Handler) GetWalletBalance(c *gin.Context) {
	userID := c.GetString("user_id")

	// Wallet balance: only succeeded top-ups count as wallet credits
	var balance float64
	h.db.Model(&Payment{}).
		Where("user_id = ? AND status = ? AND kind = ?",
			userID, PaymentStatusSucceeded, PaymentKindWalletTopUp).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&balance)

	// Total card spend (purchases + auction payments) — informational only, does not affect wallet
	var totalCardSpent float64
	h.db.Model(&Payment{}).
		Where("user_id = ? AND status = ? AND kind IN ?",
			userID, PaymentStatusSucceeded, []PaymentKind{PaymentKindPurchase, PaymentKindAuctionPayment}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalCardSpent)

	// Total refunded card payments — informational only
	var totalRefunded float64
	h.db.Model(&Payment{}).
		Where("user_id = ? AND status = ? AND kind IN ?",
			userID, PaymentStatusRefunded, []PaymentKind{PaymentKindPurchase, PaymentKindAuctionPayment}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalRefunded)

	response.OK(c, gin.H{
		"balance":          balance,
		"total_card_spent": totalCardSpent,
		"total_refunded":   totalRefunded,
		"currency":         "AED",
	})
}

// ════════════════════════════════════════════════════════════════════════════
// WalletTopUp — POST /api/v1/wallet/top-up
// ════════════════════════════════════════════════════════════════════════════

type WalletTopUpReq struct {
	Amount   float64 `json:"amount"   binding:"required,gt=0"`
	Currency string  `json:"currency"`
}

// WalletTopUp creates a Stripe PaymentIntent specifically for adding funds
// to the user's wallet (no seller involved).
func (h *Handler) WalletTopUp(c *gin.Context) {
	var req WalletTopUpReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if req.Currency == "" {
		req.Currency = "AED"
	}

	req.Currency = strings.ToUpper(security.SanitizeText(req.Currency))
	if len(req.Currency) != 3 {
		response.BadRequest(c, "currency must be a 3-letter code")
		return
	}

	buyerID := c.GetString("user_id")
	var buyer users.User
	if err := h.db.First(&buyer, "id = ?", buyerID).Error; err != nil {
		response.NotFound(c, "user")
		return
	}

	stripeCustomerID, err := h.ensureStripeCustomer(&buyer)
	if err != nil {
		slog.Error("failed to ensure Stripe customer",
			"user_id", buyer.ID.String(), "error", err.Error())
		response.InternalError(c, err)
		return
	}

	// Create the Stripe PaymentIntent (if Stripe is configured)
	clientSecret := ""
	piID := ""
	if stripeCustomerID != "" {
		amountCents := int64(req.Amount * 100)
		piParams := &stripe.PaymentIntentParams{
			Amount:   stripe.Int64(amountCents),
			Currency: stripe.String(strings.ToLower(req.Currency)),
			Customer: stripe.String(stripeCustomerID),
		}
		piParams.AddMetadata("user_id", buyerID)
		piParams.AddMetadata("type", "wallet_topup")
		pi, piErr := paymentintent.New(piParams)
		if piErr != nil {
			slog.Error("Stripe PI creation failed for wallet top-up",
				"user_id", buyerID, "error", piErr.Error())
			response.InternalError(c, fmt.Errorf("failed to create payment intent: %w", piErr))
			return
		}
		clientSecret = pi.ClientSecret
		piID = pi.ID
	}

	// When Stripe is not configured, use a unique local ID to satisfy the uniqueIndex
	if piID == "" {
		piID = "local_" + uuid.New().String()
	}

	// Persist a pending payment record
	payment := Payment{
		UserID:                buyer.ID,
		Kind:                  PaymentKindWalletTopUp,
		Amount:                req.Amount,
		Currency:              req.Currency,
		Status:                PaymentStatusPending,
		Description:           fmt.Sprintf("Wallet top-up — %.0f %s", req.Amount, req.Currency),
		StripePaymentIntentID: piID,
	}
	if err := h.db.Create(&payment).Error; err != nil {
		slog.Error("failed to create wallet top-up payment", "error", err.Error())
		response.InternalError(c, err)
		return
	}

	security.LogEvent(h.db, c, &buyer.ID, security.EventWalletTopUp, map[string]any{
		"payment_id": payment.ID.String(),
		"amount":     req.Amount,
		"currency":   req.Currency,
		"stripe":     clientSecret != "",
	})

	// Expose payment_intent_id only when it is a real Stripe ID
	exposedPIID := ""
	if clientSecret != "" {
		exposedPIID = piID
	}

	slog.Info("wallet top-up initiated",
		"user_id", buyerID,
		"amount", req.Amount,
		"currency", req.Currency,
		"stripe", clientSecret != "",
	)

	response.Created(c, gin.H{
		"payment_id":        payment.ID,
		"payment_intent_id": exposedPIID,
		"client_secret":     clientSecret,
		"amount":            req.Amount,
		"currency":          req.Currency,
		"stripe_configured": clientSecret != "",
	})
}

// ════════════════════════════════════════════════════════════════════════════
// GetWalletTransactions — GET /api/v1/wallet/transactions
// ════════════════════════════════════════════════════════════════════════════

// GetWalletTransactions returns the user's payment history as wallet transactions.
func (h *Handler) GetWalletTransactions(c *gin.Context) {
	userID := c.GetString("user_id")

	var payments []Payment
	page, perPage := paginationParams(c)
	h.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Find(&payments)

	type WalletTx struct {
		ID          interface{} `json:"id"`
		Amount      float64     `json:"amount"`
		Kind        string      `json:"kind"`
		Status      string      `json:"status"`
		Description string      `json:"description"`
		Currency    string      `json:"currency"`
		CreatedAt   interface{} `json:"created_at"`
	}

	txs := make([]WalletTx, 0, len(payments))
	for _, p := range payments {
		note := p.Description
		// Determine transaction kind and sign from authoritative Kind field
		txType := string(p.Kind)
		if txType == "" {
			txType = "payment"
		}
		// Positive amounts are credits; negative are debits
		amount := -p.Amount // default: debit
		switch p.Kind {
		case PaymentKindWalletTopUp:
			amount = p.Amount // always a credit
			if p.Status == PaymentStatusPending {
				note = fmt.Sprintf("%s (pending)", p.Description)
			}
		case PaymentKindRefund:
			amount = p.Amount // credits the buyer
			txType = "refund"
			note = fmt.Sprintf("Refund: %s", p.Description)
		case PaymentKindAuctionPayment, PaymentKindPurchase:
			amount = -p.Amount // debits the buyer
		}
		// Status overrides for non-kind-based cases
		if p.Kind == "" || p.Kind == PaymentKindPurchase || p.Kind == PaymentKindAuctionPayment {
			switch p.Status {
			case PaymentStatusRefunded:
				txType = "refund"
				amount = p.Amount
				note = fmt.Sprintf("Refund: %s", p.Description)
			case PaymentStatusFailed:
				txType = "failed"
			case PaymentStatusCancelled:
				txType = "cancelled"
			}
		}
		txs = append(txs, WalletTx{
			ID:          p.ID,
			Amount:      amount,
			Kind:        txType,
			Status:      string(p.Status),
			Description: note,
			Currency:    p.Currency,
			CreatedAt:   p.CreatedAt,
		})
	}

	response.OK(c, txs)
}

// paginationParams extracts page and per_page from query string.
func paginationParams(c *gin.Context) (page, perPage int) {
	page = 1
	perPage = 20
	if p := c.Query("page"); p != "" {
		fmt.Sscan(p, &page)
	}
	if pp := c.Query("per_page"); pp != "" {
		fmt.Sscan(pp, &perPage)
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return
}
