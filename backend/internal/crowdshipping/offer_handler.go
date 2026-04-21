package crowdshipping

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/geocore-next/backend/internal/freeze"
	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/internal/order"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OfferHandler struct {
	db       *gorm.DB
	notifSvc *notifications.Service
}

func NewOfferHandler(db *gorm.DB, notifSvc *notifications.Service) *OfferHandler {
	return &OfferHandler{db: db, notifSvc: notifSvc}
}

func (h *OfferHandler) notifyAsync(input notifications.NotifyInput) {
	if h.notifSvc != nil {
		go h.notifSvc.Notify(input)
	}
}

type CreateOfferReq struct {
	DeliveryRequestID string `json:"delivery_request_id" binding:"required"`
	PriceCents        int64  `json:"price_cents" binding:"required,gt=0"`
	Currency          string `json:"currency"`
	Note              string `json:"note"`
	IdempotencyKey    string `json:"idempotency_key"`
}

func (h *OfferHandler) CreateOffer(c *gin.Context) {
	var req CreateOfferReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	travelerID, _ := uuid.Parse(c.GetString("user_id"))
	deliveryReqID, _ := uuid.Parse(req.DeliveryRequestID)

	// Sprint 8.5: Block frozen users from creating offers
	if freeze.IsUserFrozen(h.db, travelerID) {
		response.Forbidden(c)
		return
	}
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	if req.IdempotencyKey != "" {
		var existing TravelerOffer
		if err := h.db.Where("idempotency_key = ? AND traveler_id = ?", req.IdempotencyKey, travelerID).First(&existing).Error; err == nil {
			response.OK(c, existing)
			return
		}
	}

	var offer TravelerOffer
	dbErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var dr DeliveryRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status IN ?", deliveryReqID, []DeliveryStatus{DeliveryPending, DeliveryMatched, DeliveryAccepted}).
			First(&dr).Error; err != nil {
			return fmt.Errorf("delivery request not available")
		}
		if dr.BuyerID == travelerID {
			return fmt.Errorf("cannot offer on own request")
		}
		var cnt int64
		tx.Model(&TravelerOffer{}).Where("delivery_request_id = ? AND status IN ?", deliveryReqID, []OfferStatus{OfferAccepted, OfferFundsHeld}).Count(&cnt)
		if cnt > 0 {
			return fmt.Errorf("already_accepted")
		}
		price := fromCents(req.PriceCents)
		platCents := req.PriceCents * 15 / 100
		offer = TravelerOffer{
			DeliveryRequestID: deliveryReqID, BuyerID: dr.BuyerID, TravelerID: travelerID,
			PriceCents: req.PriceCents, Price: price, PlatformFeeCents: platCents,
			TravelerEarningsCents: req.PriceCents - platCents, Currency: currency,
			Status: OfferPending, PaymentRetryAllowed: true,
			ExpiresAt: time.Now().Add(72 * time.Hour), Note: req.Note, IdempotencyKey: req.IdempotencyKey,
		}
		return tx.Create(&offer).Error
	})
	if dbErr != nil {
		switch dbErr.Error() {
		case "already_accepted":
			c.JSON(http.StatusConflict, gin.H{"success": false, "error": "already_accepted", "message": "Already has accepted offer"})
		case "delivery request not available":
			response.BadRequest(c, "delivery request not available")
		default:
			slog.Error("crowdshipping: create offer failed", "error", dbErr.Error())
			response.InternalError(c, dbErr)
		}
		return
	}
	h.notifyAsync(notifications.NotifyInput{
		UserID: offer.BuyerID, Type: "offer_new", Title: "New Offer",
		Body: fmt.Sprintf("A traveler offered $%.2f for your delivery", offer.Price),
		Data: map[string]string{"offer_id": offer.ID.String()},
	})
	metrics.IncBusinessEvent("offer_created")
	response.Created(c, offer)
}

type CounterOfferReq struct {
	PriceCents     int64  `json:"price_cents" binding:"required,gt=0"`
	Currency       string `json:"currency"`
	Note           string `json:"note"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (h *OfferHandler) CounterOffer(c *gin.Context) {
	offerID, _ := uuid.Parse(c.Param("id"))
	var req CounterOfferReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	var counter TravelerOffer
	dbErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var orig TravelerOffer
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", offerID).First(&orig).Error; err != nil {
			return fmt.Errorf("offer not found")
		}
		if orig.Status != OfferPending && orig.Status != OfferCountered {
			return fmt.Errorf("offer_not_counterable")
		}
		if orig.Status == OfferPending && orig.BuyerID != userID {
			return fmt.Errorf("only_buyer_can_counter")
		}
		if orig.Status == OfferCountered && orig.TravelerID != userID {
			return fmt.Errorf("only_traveler_can_counter")
		}
		tx.Model(&orig).Update("status", OfferCountered)
		price := fromCents(req.PriceCents)
		platCents := req.PriceCents * 15 / 100
		counter = TravelerOffer{
			DeliveryRequestID: orig.DeliveryRequestID, BuyerID: orig.BuyerID, TravelerID: orig.TravelerID,
			PriceCents: req.PriceCents, Price: price, PlatformFeeCents: platCents,
			TravelerEarningsCents: req.PriceCents - platCents, Currency: currency,
			Status: OfferPending, PaymentRetryAllowed: true,
			ExpiresAt: time.Now().Add(72 * time.Hour), Note: req.Note,
			CounterToID: &orig.ID, IdempotencyKey: req.IdempotencyKey,
		}
		return tx.Create(&counter).Error
	})
	if dbErr != nil {
		response.BadRequest(c, dbErr.Error())
		return
	}
	notifyUID := counter.TravelerID
	if userID == counter.TravelerID {
		notifyUID = counter.BuyerID
	}
	h.notifyAsync(notifications.NotifyInput{
		UserID: notifyUID, Type: "offer_counter", Title: "Counter Offer",
		Body: fmt.Sprintf("Counter offer of $%.2f", counter.Price),
		Data: map[string]string{"offer_id": counter.ID.String()},
	})
	metrics.IncBusinessEvent("offer_countered")
	response.Created(c, counter)
}

func (h *OfferHandler) RejectOffer(c *gin.Context) {
	offerID, _ := uuid.Parse(c.Param("id"))
	userID, _ := uuid.Parse(c.GetString("user_id"))
	dbErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var offer TravelerOffer
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND buyer_id = ?", offerID, userID).First(&offer).Error; err != nil {
			return fmt.Errorf("offer not found")
		}
		if !offer.CanTransitionTo(OfferRejected) {
			return fmt.Errorf("not rejectable: %s", offer.Status)
		}
		return tx.Model(&offer).Update("status", OfferRejected).Error
	})
	if dbErr != nil {
		response.BadRequest(c, dbErr.Error())
		return
	}
	var offer TravelerOffer
	h.db.Where("id = ?", offerID).First(&offer)
	h.notifyAsync(notifications.NotifyInput{
		UserID: offer.TravelerID, Type: "offer_rejected", Title: "Offer Rejected",
		Body: "Your offer was declined", Data: map[string]string{"offer_id": offerID.String()},
	})
	response.OK(c, gin.H{"message": "offer rejected"})
}

func (h *OfferHandler) ListOffersForDeliveryRequest(c *gin.Context) {
	drID, _ := uuid.Parse(c.Param("listing_id"))
	var offers []TravelerOffer
	q := h.db.Where("delivery_request_id = ?", drID).Order("created_at DESC")
	userID := c.GetString("user_id")
	var dr DeliveryRequest
	if h.db.Where("id = ?", drID).First(&dr).Error == nil && dr.BuyerID.String() != userID {
		q = q.Where("traveler_id = ?", userID)
	}
	if err := q.Find(&offers).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, offers)
}

// createOrderFromOffer is shared by AcceptOffer and RetryPayment.
func createOrderFromOffer(tx *gorm.DB, offer *TravelerOffer, note string) (*order.Order, error) {
	now := time.Now()
	ord := order.Order{
		BuyerID: offer.BuyerID, SellerID: offer.TravelerID,
		Status: order.StatusConfirmed, Subtotal: offer.Price,
		PlatformFee: fromCents(offer.PlatformFeeCents), Total: offer.Price,
		Currency: offer.Currency, DeliveryType: order.DeliveryTypeCrowdshipping,
		ConfirmedAt: &now,
	}
	if err := tx.Create(&ord).Error; err != nil {
		return nil, fmt.Errorf("order creation failed: %w", err)
	}
	offer.OrderID = &ord.ID
	tx.Save(offer)
	oi := order.OrderItem{
		OrderID: ord.ID, Title: "Crowdshipping Delivery",
		Quantity: 1, UnitPrice: offer.Price, TotalPrice: offer.Price,
	}
	tx.Create(&oi)
	travelerID := offer.TravelerID
	tx.Model(&DeliveryRequest{}).Where("id = ?", offer.DeliveryRequestID).
		Updates(map[string]any{"status": DeliveryAccepted, "traveler_id": travelerID})
	te := TrackingEvent{OrderID: ord.ID, TravelerID: offer.TravelerID, Status: ShipmentRequested, Note: note}
	tx.Create(&te)
	return &ord, nil
}
