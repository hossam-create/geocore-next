package listings

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/events"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ── Negotiation types ──────────────────────────────────────────────────────────

type NegotiationStatus string

const (
	NegotiationOpen           NegotiationStatus = "open"
	NegotiationCountered      NegotiationStatus = "countered"
	NegotiationPendingPayment NegotiationStatus = "pending_payment_lock" // escrow in progress
	NegotiationAccepted       NegotiationStatus = "accepted"             // escrow held successfully
	NegotiationPaymentFailed  NegotiationStatus = "payment_failed"       // escrow failed, retry allowed
	NegotiationRejected       NegotiationStatus = "rejected"
	NegotiationExpired        NegotiationStatus = "expired"
	NegotiationClosed         NegotiationStatus = "closed" // converted to order
)

type NegotiationThread struct {
	ID                  uuid.UUID            `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ListingID           uuid.UUID            `gorm:"type:uuid;not null;index" json:"listing_id"`
	BuyerID             uuid.UUID            `gorm:"type:uuid;not null;index" json:"buyer_id"`
	SellerID            uuid.UUID            `gorm:"type:uuid;not null;index" json:"seller_id"`
	Status              NegotiationStatus    `gorm:"type:varchar(20);not null;default:'open';index" json:"status"`
	AgreedCents         int64                `gorm:"default:0" json:"agreed_cents"` // final agreed price in cents
	AgreedPrice         float64              `gorm:"default:0" json:"agreed_price"` // final agreed price in USD
	Currency            string               `gorm:"default:'USD'" json:"currency"`
	PaymentRetryAllowed bool                 `gorm:"default:true" json:"payment_retry_allowed"` // buyer can retry payment
	ExpiresAt           *time.Time           `json:"expires_at,omitempty"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
	DeletedAt           gorm.DeletedAt       `gorm:"index" json:"-"`
	Messages            []NegotiationMessage `gorm:"foreignKey:ThreadID" json:"messages,omitempty"`
}

func (NegotiationThread) TableName() string { return "negotiation_threads" }

type OfferAction string

const (
	OfferActionOffer   OfferAction = "offer"
	OfferActionAccept  OfferAction = "accept"
	OfferActionReject  OfferAction = "reject"
	OfferActionCounter OfferAction = "counter"
)

type NegotiationMessage struct {
	ID               uuid.UUID   `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ThreadID         uuid.UUID   `gorm:"type:uuid;not null;index" json:"thread_id"`
	SenderID         uuid.UUID   `gorm:"type:uuid;not null;index" json:"sender_id"`
	Action           OfferAction `gorm:"type:varchar(20);not null" json:"action"`
	PriceCents       int64       `gorm:"not null" json:"price_cents"`
	Price            float64     `gorm:"not null" json:"price"`
	DeliveryFeeCents int64       `gorm:"default:0" json:"delivery_fee_cents"`
	DeliveryFee      float64     `gorm:"default:0" json:"delivery_fee"`
	Currency         string      `gorm:"default:'USD'" json:"currency"`
	Breakdown        string      `gorm:"type:jsonb;default:'{}'" json:"breakdown,omitempty"` // pricing breakdown JSON
	Note             string      `gorm:"type:text" json:"note,omitempty"`
	AutoDecision     bool        `gorm:"default:false" json:"auto_decision"` // true if auto-accept/reject
	CreatedAt        time.Time   `json:"created_at"`
}

func (NegotiationMessage) TableName() string { return "negotiation_messages" }

// ── Notification integration ────────────────────────────────────────────────────

// NotificationService is an interface satisfied by *notifications.Service.
// Using an interface avoids circular imports.
type NotificationService interface {
	Notify(input notifications.NotifyInput)
}

var globalNotifSvc NotificationService

// SetNotificationService wires the notification service into this package.
func SetNotificationService(svc NotificationService) {
	globalNotifSvc = svc
}

func notifyOffer(listingID uuid.UUID, recipientID uuid.UUID, nType string, title string, body string) {
	if globalNotifSvc == nil {
		return
	}
	go globalNotifSvc.Notify(notifications.NotifyInput{
		UserID: recipientID,
		Type:   nType,
		Title:  title,
		Body:   body,
		Data:   map[string]string{"listing_id": listingID.String()},
	})
}

// ── Core negotiation logic ──────────────────────────────────────────────────────

// toCents converts USD float to integer cents (same pattern as crowdshipping).
func toCents(usd float64) int64     { return int64(usd*100 + 0.5) }
func fromCents(cents int64) float64 { return float64(cents) / 100.0 }

// EvaluateOffer applies auto-accept/reject rules based on ListingTradeConfig.
// Returns the action the system should take automatically, or empty string if no auto-action.
func EvaluateOffer(listingPriceCents int64, offerCents int64, cfg ListingTradeConfig) OfferAction {
	if listingPriceCents <= 0 {
		return "" // no price to compare against
	}
	minCents := int64(float64(listingPriceCents) * cfg.MinOfferPercent)
	autoAcceptCents := int64(float64(listingPriceCents) * cfg.AutoAcceptPercent)

	if offerCents < minCents {
		return OfferActionReject
	}
	if offerCents >= autoAcceptCents {
		return OfferActionAccept
	}
	return "" // manual review needed
}

// ── HTTP handlers ───────────────────────────────────────────────────────────────

// SubmitOffer creates a new offer or counter-offer on a listing.
// POST /api/v1/listings/:id/offer
func (h *Handler) SubmitOffer(c *gin.Context) {
	buyerID, _ := uuid.Parse(c.MustGet("user_id").(string))
	listingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid listing ID")
		return
	}

	var req struct {
		Price       float64                `json:"price" binding:"required,gt=0"`
		DeliveryFee float64                `json:"delivery_fee"`
		Note        string                 `json:"note"`
		ThreadID    *string                `json:"thread_id"` // nil = new thread, non-nil = counter-offer
		Breakdown   map[string]interface{} `json:"breakdown"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	offerCents := toCents(req.Price)
	deliveryCents := toCents(req.DeliveryFee)

	// Load listing with FOR UPDATE to prevent concurrent modifications
	var listing Listing
	if err := h.writeDB().Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&listing, "id = ? AND status = ?", listingID, "active").Error; err != nil {
		response.NotFound(c, "Listing")
		return
	}

	if listing.UserID == buyerID {
		response.BadRequest(c, "Cannot make offer on your own listing")
		return
	}

	// ── RISK 2 FIX: explicit status guard ──
	if listing.Status != "active" {
		response.BadRequest(c, "Listing is no longer available (already sold or expired)")
		return
	}

	cfg := listing.GetTradeConfig()
	if !cfg.OfferEnabled && listing.ListingType != ListingTypeNegotiation && listing.ListingType != ListingTypeHybrid {
		response.BadRequest(c, "Offers are not enabled for this listing")
		return
	}

	listingPriceCents := listing.PriceCents
	if listingPriceCents == 0 && listing.Price != nil {
		listingPriceCents = toCents(*listing.Price)
	}

	var thread NegotiationThread
	var actionTaken OfferAction

	dbErr := h.writeDB().Transaction(func(tx *gorm.DB) error {
		if req.ThreadID != nil {
			// Counter-offer on existing thread
			threadID, parseErr := uuid.Parse(*req.ThreadID)
			if parseErr != nil {
				return fmt.Errorf("invalid thread_id")
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&thread, "id = ? AND buyer_id = ? AND status IN ?", threadID, buyerID,
					[]NegotiationStatus{NegotiationOpen, NegotiationCountered}).Error; err != nil {
				return fmt.Errorf("thread_not_found")
			}
		} else {
			// New thread — check if buyer already has an open thread
			existingThread := NegotiationThread{}
			if err := tx.Where("listing_id = ? AND buyer_id = ? AND status IN ?",
				listingID, buyerID, []NegotiationStatus{NegotiationOpen, NegotiationCountered}).
				First(&existingThread).Error; err == nil {
				thread = existingThread
			} else {
				// Create new thread
				expiresAt := time.Now().Add(time.Duration(cfg.OfferExpiryHours) * time.Hour)
				thread = NegotiationThread{
					ID:        uuid.New(),
					ListingID: listingID,
					BuyerID:   buyerID,
					SellerID:  listing.UserID,
					Status:    NegotiationOpen,
					Currency:  listing.Currency,
					ExpiresAt: &expiresAt,
				}
				if err := tx.Create(&thread).Error; err != nil {
					return err
				}
			}
		}

		// Evaluate auto-accept/reject
		autoAction := EvaluateOffer(listingPriceCents, offerCents, cfg)
		msgAction := OfferActionOffer
		autoDecision := false

		switch autoAction {
		case OfferActionReject:
			msgAction = OfferActionReject
			autoDecision = true
			thread.Status = NegotiationRejected
		case OfferActionAccept:
			// ── RISK 1 FIX: HoldFunds BEFORE marking ACCEPTED ──
			// 1. Set state to PENDING_PAYMENT_LOCK first
			thread.Status = NegotiationPendingPayment
			thread.AgreedCents = offerCents
			thread.AgreedPrice = fromCents(offerCents)

			// 2. Attempt escrow hold within the same transaction
			total := fromCents(offerCents)
			_, escrowErr := wallet.HoldFunds(tx, thread.BuyerID, thread.SellerID, total, thread.Currency,
				"negotiation", thread.ID.String())
			if escrowErr != nil {
				// Escrow failed → PAYMENT_FAILED, not ACCEPTED
				thread.Status = NegotiationPaymentFailed
				thread.PaymentRetryAllowed = true
				msgAction = OfferActionOffer // record as offer, not accept
				autoDecision = true
				autoAction = OfferActionOffer // override so we notify about payment failure
				slog.Error("trading: auto-accept escrow failed",
					"thread_id", thread.ID, "error", escrowErr)
			} else {
				// Escrow succeeded → ACCEPTED
				msgAction = OfferActionAccept
				autoDecision = true
				thread.Status = NegotiationAccepted
			}
		default:
			thread.Status = NegotiationOpen
		}
		actionTaken = autoAction

		breakdownJSON, _ := json.Marshal(req.Breakdown)
		msg := NegotiationMessage{
			ID:               uuid.New(),
			ThreadID:         thread.ID,
			SenderID:         buyerID,
			Action:           msgAction,
			PriceCents:       offerCents,
			Price:            fromCents(offerCents),
			DeliveryFeeCents: deliveryCents,
			DeliveryFee:      fromCents(deliveryCents),
			Currency:         listing.Currency,
			Breakdown:        string(breakdownJSON),
			Note:             req.Note,
			AutoDecision:     autoDecision,
		}
		if err := tx.Create(&msg).Error; err != nil {
			return err
		}

		return tx.Save(&thread).Error
	})

	if dbErr != nil {
		response.BadRequest(c, dbErr.Error())
		return
	}

	// Notifications + events (post-commit)
	switch actionTaken {
	case OfferActionAccept:
		notifyOffer(listingID, buyerID, "offer_accepted", "Offer Accepted",
			fmt.Sprintf("Your offer of %.2f %s was auto-accepted!", fromCents(offerCents), listing.Currency))
		notifyOffer(listingID, listing.UserID, "offer_accepted", "Offer Auto-Accepted",
			fmt.Sprintf("An offer of %.2f %s met your auto-accept threshold.", fromCents(offerCents), listing.Currency))
	case OfferActionReject:
		notifyOffer(listingID, buyerID, "offer_rejected", "Offer Rejected",
			fmt.Sprintf("Your offer of %.2f %s was below the minimum threshold.", fromCents(offerCents), listing.Currency))
	default:
		notifyOffer(listingID, listing.UserID, "offer_created", "New Offer",
			fmt.Sprintf("You received an offer of %.2f %s on your listing.", fromCents(offerCents), listing.Currency))
	}

	slog.Info("trading: offer submitted",
		"listing_id", listingID,
		"thread_id", thread.ID,
		"buyer_id", buyerID,
		"offer_cents", offerCents,
		"auto_action", string(actionTaken),
	)

	// Publish domain event
	events.Publish(events.Event{
		Type: events.EventListingCreated, // reuse event bus — will add specific types later
		Payload: map[string]interface{}{
			"event_type":  "offer.created",
			"listing_id":  listingID.String(),
			"thread_id":   thread.ID.String(),
			"buyer_id":    buyerID.String(),
			"offer_cents": offerCents,
			"auto_action": string(actionTaken),
		},
	})

	response.Created(c, gin.H{
		"thread":      thread,
		"auto_action": actionTaken,
	})
}

// RespondOffer allows the seller to accept, reject, or counter an offer.
// POST /api/v1/listings/:id/offer/respond
func (h *Handler) RespondOffer(c *gin.Context) {
	sellerID, _ := uuid.Parse(c.MustGet("user_id").(string))
	listingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid listing ID")
		return
	}

	var req struct {
		ThreadID string  `json:"thread_id" binding:"required"`
		Action   string  `json:"action" binding:"required"` // accept | reject | counter
		Price    float64 `json:"price"`                     // required for counter
		Note     string  `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	threadID, _ := uuid.Parse(req.ThreadID)
	action := OfferAction(req.Action)
	if action != OfferActionAccept && action != OfferActionReject && action != OfferActionCounter {
		response.BadRequest(c, "Invalid action: must be accept, reject, or counter")
		return
	}

	var thread NegotiationThread
	var listing Listing

	dbErr := h.writeDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&thread, "id = ? AND seller_id = ? AND status IN ?", threadID, sellerID,
				[]NegotiationStatus{NegotiationOpen, NegotiationCountered}).Error; err != nil {
			return fmt.Errorf("thread_not_found")
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&listing, "id = ? AND status = ?", listingID, "active").Error; err != nil {
			return fmt.Errorf("listing_not_found")
		}

		switch action {
		case OfferActionAccept:
			// Find the latest offer from buyer
			var lastOffer NegotiationMessage
			if err := tx.Where("thread_id = ? AND sender_id = ? AND action IN ?",
				threadID, thread.BuyerID, []OfferAction{OfferActionOffer, OfferActionCounter}).
				Order("created_at DESC").First(&lastOffer).Error; err != nil {
				return fmt.Errorf("no_offer_found")
			}
			// ── RISK 1 FIX: HoldFunds BEFORE marking ACCEPTED ──
			thread.Status = NegotiationPendingPayment
			thread.AgreedCents = lastOffer.PriceCents
			thread.AgreedPrice = lastOffer.Price

			total := fromCents(lastOffer.PriceCents)
			_, escrowErr := wallet.HoldFunds(tx, thread.BuyerID, thread.SellerID, total, thread.Currency,
				"negotiation", thread.ID.String())
			if escrowErr != nil {
				thread.Status = NegotiationPaymentFailed
				thread.PaymentRetryAllowed = true
				slog.Error("trading: seller-accept escrow failed",
					"thread_id", thread.ID, "error", escrowErr)
			} else {
				thread.Status = NegotiationAccepted
			}
		case OfferActionReject:
			thread.Status = NegotiationRejected
		case OfferActionCounter:
			if req.Price <= 0 {
				return fmt.Errorf("counter_price_required")
			}
			thread.Status = NegotiationCountered
		}

		counterCents := int64(0)
		counterPrice := 0.0
		if action == OfferActionCounter {
			counterCents = toCents(req.Price)
			counterPrice = fromCents(counterCents)
		}

		msg := NegotiationMessage{
			ID:         uuid.New(),
			ThreadID:   thread.ID,
			SenderID:   sellerID,
			Action:     action,
			PriceCents: counterCents,
			Price:      counterPrice,
			Currency:   listing.Currency,
			Note:       req.Note,
		}
		if err := tx.Create(&msg).Error; err != nil {
			return err
		}

		return tx.Save(&thread).Error
	})

	if dbErr != nil {
		response.BadRequest(c, dbErr.Error())
		return
	}

	// Notifications
	switch action {
	case OfferActionAccept:
		notifyOffer(listingID, thread.BuyerID, "offer_accepted", "Offer Accepted!",
			fmt.Sprintf("The seller accepted your offer of %.2f %s!", thread.AgreedPrice, thread.Currency))
	case OfferActionReject:
		notifyOffer(listingID, thread.BuyerID, "offer_rejected", "Offer Rejected",
			"The seller rejected your offer.")
	case OfferActionCounter:
		notifyOffer(listingID, thread.BuyerID, "offer_countered", "Counter Offer",
			fmt.Sprintf("The seller countered with %.2f %s.", fromCents(toCents(req.Price)), thread.Currency))
	}

	slog.Info("trading: offer response",
		"listing_id", listingID,
		"thread_id", threadID,
		"seller_id", sellerID,
		"action", string(action),
	)

	response.OK(c, gin.H{"thread": thread, "action": action})
}

// GetNegotiationThread retrieves a negotiation thread with messages.
// GET /api/v1/listings/:id/negotiation/:thread_id
func (h *Handler) GetNegotiationThread(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	threadID, err := uuid.Parse(c.Param("thread_id"))
	if err != nil {
		response.BadRequest(c, "Invalid thread ID")
		return
	}

	var thread NegotiationThread
	if err := h.readDB().Preload("Messages", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at ASC")
	}).First(&thread, "id = ? AND (buyer_id = ? OR seller_id = ?)", threadID, userID, userID).Error; err != nil {
		response.NotFound(c, "Negotiation thread")
		return
	}

	response.OK(c, thread)
}

// ListNegotiationThreads lists all negotiation threads for a listing (seller view).
// GET /api/v1/listings/:id/negotiations
func (h *Handler) ListNegotiationThreads(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	listingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid listing ID")
		return
	}

	var threads []NegotiationThread
	if err := h.readDB().Where("listing_id = ? AND seller_id = ?", listingID, userID).
		Order("updated_at DESC").Find(&threads).Error; err != nil {
		response.NotFound(c, "Negotiation threads")
		return
	}

	response.OK(c, threads)
}

// ConvertAcceptedOfferToOrder converts an accepted negotiation to an order + escrow.
// Called internally after offer acceptance.
func ConvertAcceptedOfferToOrder(db *gorm.DB, thread *NegotiationThread, listing *Listing) error {
	if thread.Status != NegotiationAccepted {
		return fmt.Errorf("thread not in accepted state")
	}

	totalCents := thread.AgreedCents
	platformCents := totalCents * 15 / 100
	travelerCents := totalCents - platformCents
	total := fromCents(totalCents)

	// Hold escrow funds
	_, err := wallet.HoldFunds(db, thread.BuyerID, thread.SellerID, total, thread.Currency,
		"negotiation", thread.ID.String())
	if err != nil {
		return fmt.Errorf("escrow failed: %w", err)
	}

	// Mark thread as closed
	now := time.Now()
	thread.Status = NegotiationClosed
	db.Save(thread)

	// Mark listing as sold
	db.Model(listing).Updates(map[string]interface{}{
		"status":  "sold",
		"sold_at": now,
	})

	slog.Info("trading: offer converted to order",
		"thread_id", thread.ID,
		"listing_id", listing.ID,
		"total_cents", totalCents,
		"platform_cents", platformCents,
		"traveler_cents", travelerCents,
	)

	_ = kafka.WriteOutbox(db, kafka.TopicOrders, kafka.New(
		"order.created",
		thread.ID.String(),
		"order",
		kafka.Actor{Type: "user", ID: thread.BuyerID.String()},
		map[string]interface{}{
			"thread_id":      thread.ID.String(),
			"listing_id":     listing.ID.String(),
			"buyer_id":       thread.BuyerID.String(),
			"seller_id":      thread.SellerID.String(),
			"total_cents":    totalCents,
			"platform_cents": platformCents,
			"currency":       thread.Currency,
			"source":         "negotiation",
		},
		kafka.EventMeta{Source: "api-service"},
	))

	return nil
}

// ExpireStaleOffers marks expired negotiation threads and sends notifications.
// Should be called periodically — use StartExpiryScheduler for automatic scheduling.
func ExpireStaleOffers(db *gorm.DB) {
	now := time.Now()
	var expired []NegotiationThread
	db.Where("status IN ? AND expires_at < ?",
		[]NegotiationStatus{NegotiationOpen, NegotiationCountered}, now).
		Find(&expired)

	if len(expired) == 0 {
		return
	}

	ids := make([]uuid.UUID, len(expired))
	for i, t := range expired {
		ids[i] = t.ID
	}
	db.Model(&NegotiationThread{}).Where("id IN ?", ids).
		Update("status", NegotiationExpired)

	// Send expiry notifications
	for _, t := range expired {
		notifyOffer(t.ListingID, t.BuyerID, "offer_expired", "Offer Expired",
			fmt.Sprintf("Your offer on this listing has expired."))
		notifyOffer(t.ListingID, t.SellerID, "offer_expired", "Offer Expired",
			fmt.Sprintf("An offer on your listing has expired."))
	}

	slog.Info("trading: expired stale offers", "count", len(expired))
}

// StartExpiryScheduler runs ExpireStaleOffers every 5 minutes until ctx is cancelled.
// Call this from main.go on startup.
func StartExpiryScheduler(ctx context.Context, db *gorm.DB) {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				ExpireStaleOffers(db)
			}
		}
	}()
	slog.Info("trading: expiry scheduler started (5m interval)")
}

// RetryPayment allows a buyer to retry escrow on a PAYMENT_FAILED thread.
// POST /api/v1/listings/:id/negotiation/:thread_id/retry-payment
func (h *Handler) RetryPayment(c *gin.Context) {
	buyerID, _ := uuid.Parse(c.MustGet("user_id").(string))
	listingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid listing ID")
		return
	}
	threadID, err := uuid.Parse(c.Param("thread_id"))
	if err != nil {
		response.BadRequest(c, "Invalid thread ID")
		return
	}

	var thread NegotiationThread
	var listing Listing

	dbErr := h.writeDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&thread, "id = ? AND buyer_id = ? AND status = ? AND payment_retry_allowed = ?",
				threadID, buyerID, NegotiationPaymentFailed, true).Error; err != nil {
			return fmt.Errorf("thread_not_retryable")
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&listing, "id = ? AND status = ?", listingID, "active").Error; err != nil {
			return fmt.Errorf("listing_not_available")
		}

		// ── Same RISK 1 pattern: PENDING_PAYMENT_LOCK → HoldFunds → ACCEPTED/FAILED ──
		thread.Status = NegotiationPendingPayment
		total := fromCents(thread.AgreedCents)
		_, escrowErr := wallet.HoldFunds(tx, thread.BuyerID, thread.SellerID, total, thread.Currency,
			"negotiation", thread.ID.String())
		if escrowErr != nil {
			thread.Status = NegotiationPaymentFailed
			slog.Error("trading: payment retry escrow failed",
				"thread_id", thread.ID, "error", escrowErr)
		} else {
			thread.Status = NegotiationAccepted
		}
		return tx.Save(&thread).Error
	})

	if dbErr != nil {
		response.BadRequest(c, dbErr.Error())
		return
	}

	if thread.Status == NegotiationAccepted {
		notifyOffer(listingID, buyerID, "offer_accepted", "Payment Successful!",
			fmt.Sprintf("Your payment of %.2f %s was processed. Your offer is now accepted!", thread.AgreedPrice, thread.Currency))
		notifyOffer(listingID, listing.UserID, "offer_accepted", "Offer Payment Confirmed",
			fmt.Sprintf("The buyer's payment of %.2f %s was confirmed.", thread.AgreedPrice, thread.Currency))
	}

	response.OK(c, gin.H{"thread_id": threadID, "status": thread.Status})
}

// handleAutoAcceptedOffer converts auto-accepted offers to orders immediately.
func handleAutoAcceptedOffer(db *gorm.DB, thread *NegotiationThread, listing *Listing) {
	if thread.Status != NegotiationAccepted {
		return
	}
	if err := ConvertAcceptedOfferToOrder(db, thread, listing); err != nil {
		slog.Error("trading: auto-accept conversion failed",
			"thread_id", thread.ID, "error", err)
	}
}
