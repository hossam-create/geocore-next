package reverseauctions

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/geocore-next/backend/internal/moderation"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct{ db *gorm.DB }

func NewHandler(db *gorm.DB) *Handler { return &Handler{db: db} }

// ── Requests ────────────────────────────────────────────────────────────────

type CreateRequestReq struct {
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description"`
	CategoryID  string   `json:"category_id"`
	MaxBudget   *float64 `json:"max_budget"`
	Deadline    string   `json:"deadline" binding:"required"`
	Images      string   `json:"images"`
}

// POST /api/v1/reverse-auctions
func (h *Handler) CreateRequest(c *gin.Context) {
	var req CreateRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	buyerID, _ := uuid.Parse(c.GetString("user_id"))

	deadline, err := time.Parse(time.RFC3339, req.Deadline)
	if err != nil {
		response.BadRequest(c, "deadline must be RFC3339 format")
		return
	}
	if deadline.Before(time.Now()) {
		response.BadRequest(c, "deadline must be in the future")
		return
	}

	// Content moderation check
	if blocked, reason := moderation.CheckContent(req.Title, req.Description); blocked {
		response.BadRequest(c, reason)
		return
	}

	status := RequestOpen

	r := ReverseAuctionRequest{
		BuyerID:     buyerID,
		Title:       req.Title,
		Description: req.Description,
		MaxBudget:   req.MaxBudget,
		Deadline:    deadline,
		Status:      status,
		Images:      "[]",
	}
	if req.CategoryID != "" {
		catID, _ := uuid.Parse(req.CategoryID)
		r.CategoryID = &catID
	}
	if req.Images != "" {
		r.Images = req.Images
	}

	if err := h.db.Create(&r).Error; err != nil {
		slog.Error("reverse-auctions: create request failed", "error", err)
		response.InternalError(c, err)
		return
	}
	response.Created(c, r)
}

// GET /api/v1/reverse-auctions
func (h *Handler) ListRequests(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage := 20
	status := c.DefaultQuery("status", "open")

	var items []ReverseAuctionRequest
	var total int64
	q := h.db.Model(&ReverseAuctionRequest{}).Where("status = ?", status)

	if cat := c.Query("category_id"); cat != "" {
		q = q.Where("category_id = ?", cat)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("title ILIKE ?", "%"+search+"%")
	}

	q.Count(&total)
	q.Order("created_at DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&items)
	response.OKMeta(c, items, gin.H{"total": total, "page": page})
}

// GET /api/v1/reverse-auctions/:id
func (h *Handler) GetRequest(c *gin.Context) {
	var r ReverseAuctionRequest
	if err := h.db.Preload("Offers").Where("id = ?", c.Param("id")).First(&r).Error; err != nil {
		response.NotFound(c, "reverse auction request")
		return
	}
	response.OK(c, r)
}

// PUT /api/v1/reverse-auctions/:id
func (h *Handler) UpdateRequest(c *gin.Context) {
	uid := c.GetString("user_id")
	var r ReverseAuctionRequest
	if err := h.db.Where("id = ? AND buyer_id = ?", c.Param("id"), uid).First(&r).Error; err != nil {
		response.NotFound(c, "reverse auction request")
		return
	}
	if r.Status != RequestOpen {
		response.BadRequest(c, "can only update open requests")
		return
	}
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	delete(body, "id")
	delete(body, "buyer_id")
	delete(body, "status")
	h.db.Model(&r).Updates(body)
	response.OK(c, r)
}

// DELETE /api/v1/reverse-auctions/:id
func (h *Handler) DeleteRequest(c *gin.Context) {
	uid := c.GetString("user_id")
	var r ReverseAuctionRequest
	if err := h.db.Where("id = ? AND buyer_id = ?", c.Param("id"), uid).First(&r).Error; err != nil {
		response.NotFound(c, "reverse auction request")
		return
	}
	h.db.Model(&r).Update("status", RequestClosed)
	// Reject all pending offers
	h.db.Model(&ReverseAuctionOffer{}).Where("request_id = ? AND status = ?", r.ID, OfferPending).
		Update("status", OfferRejected)
	response.OK(c, gin.H{"message": "request closed"})
}

// ── Offers ──────────────────────────────────────────────────────────────────

type CreateOfferReq struct {
	Price        float64 `json:"price" binding:"required,gt=0"`
	Description  string  `json:"description"`
	DeliveryDays *int    `json:"delivery_days"`
}

// POST /api/v1/reverse-auctions/:id/offers
func (h *Handler) CreateOffer(c *gin.Context) {
	var req CreateOfferReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	sellerID, _ := uuid.Parse(c.GetString("user_id"))
	requestID, _ := uuid.Parse(c.Param("id"))

	var r ReverseAuctionRequest
	if err := h.db.Where("id = ? AND status = ?", requestID, RequestOpen).First(&r).Error; err != nil {
		response.NotFound(c, "open reverse auction request")
		return
	}

	if r.BuyerID == sellerID {
		response.BadRequest(c, "cannot make an offer on your own request")
		return
	}

	if r.MaxBudget != nil && req.Price > *r.MaxBudget {
		response.BadRequest(c, "offer price exceeds buyer's max budget")
		return
	}

	expiresAt := time.Now().Add(48 * time.Hour)
	offer := ReverseAuctionOffer{
		RequestID:    requestID,
		SellerID:     sellerID,
		Price:        req.Price,
		Description:  req.Description,
		DeliveryDays: req.DeliveryDays,
		ExpiresAt:    &expiresAt,
		Status:       OfferPending,
	}

	if err := h.db.Create(&offer).Error; err != nil {
		// Likely unique constraint violation
		response.BadRequest(c, "you already made an offer on this request")
		return
	}
	response.Created(c, offer)
}

// GET /api/v1/reverse-auctions/:id/offers
func (h *Handler) ListOffers(c *gin.Context) {
	requestID := c.Param("id")
	var offers []ReverseAuctionOffer
	h.db.Where("request_id = ?", requestID).Order("price ASC").Find(&offers)
	response.OK(c, offers)
}

// PUT /api/v1/reverse-auctions/:id/offers/:offerId/accept
func (h *Handler) AcceptOffer(c *gin.Context) {
	uid := c.GetString("user_id")
	requestID, _ := uuid.Parse(c.Param("id"))
	offerID, _ := uuid.Parse(c.Param("offerId"))

	var r ReverseAuctionRequest
	if err := h.db.Where("id = ? AND buyer_id = ? AND status = ?", requestID, uid, RequestOpen).First(&r).Error; err != nil {
		response.NotFound(c, "open request owned by you")
		return
	}

	var offer ReverseAuctionOffer
	if err := h.db.Where("id = ? AND request_id = ? AND status IN ?", offerID, requestID, []OfferStatus{OfferPending, OfferCountered}).First(&offer).Error; err != nil {
		response.NotFound(c, "pending or countered offer")
		return
	}

	// Check expiry
	if offer.ExpiresAt != nil && time.Now().After(*offer.ExpiresAt) {
		h.db.Model(&offer).Update("status", OfferExpired)
		response.BadRequest(c, "offer has expired")
		return
	}

	now := time.Now()
	// Use counter price if it was countered
	finalPrice := offer.Price
	if offer.Status == OfferCountered && offer.CounterPrice != nil {
		finalPrice = *offer.CounterPrice
	}

	buyerID, _ := uuid.Parse(uid)
	// ── Atomic: accept + reject others + hold funds in ONE transaction ─────────
	// A deal with no secured escrow must not proceed (M-02 fix).
	err := h.db.Transaction(func(tx *gorm.DB) error {
		tx.Model(&offer).Updates(map[string]any{
			"status":       OfferAccepted,
			"price":        finalPrice,
			"responded_at": now,
		})
		tx.Model(&ReverseAuctionOffer{}).
			Where("request_id = ? AND id != ? AND status IN ?", requestID, offerID, []OfferStatus{OfferPending, OfferCountered}).
			Updates(map[string]any{"status": OfferRejected, "responded_at": now})
		tx.Model(&r).Update("status", RequestFulfilled)
		_, err := wallet.HoldFunds(tx, buyerID, offer.SellerID, finalPrice, "USD", "REVERSE_AUCTION", requestID.String())
		if err != nil {
			return fmt.Errorf("escrow_hold_failed: %w", err)
		}
		return nil
	})
	if err != nil {
		if len(err.Error()) > 19 && err.Error()[:19] == "escrow_hold_failed:" {
			response.BadRequest(c, "Could not secure buyer funds for this deal")
		} else {
			response.InternalError(c, err)
		}
		return
	}

	response.OK(c, gin.H{"message": "offer accepted", "offer": offer})
}

// ...
func (h *Handler) RejectOffer(c *gin.Context) {
	uid := c.GetString("user_id")
	requestID, _ := uuid.Parse(c.Param("id"))
	offerID, _ := uuid.Parse(c.Param("offerId"))

	var r ReverseAuctionRequest
	if err := h.db.Where("id = ? AND buyer_id = ?", requestID, uid).First(&r).Error; err != nil {
		response.NotFound(c, "request owned by you")
		return
	}

	result := h.db.Model(&ReverseAuctionOffer{}).
		Where("id = ? AND request_id = ? AND status = ?", offerID, requestID, OfferPending).
		Update("status", OfferRejected)
	if result.RowsAffected == 0 {
		response.NotFound(c, "pending offer")
		return
	}
	response.OK(c, gin.H{"message": "offer rejected"})
}

// DELETE /api/v1/reverse-auctions/:id/offers/:offerId  (seller withdraws own offer)
func (h *Handler) WithdrawOffer(c *gin.Context) {
	uid := c.GetString("user_id")
	offerID, _ := uuid.Parse(c.Param("offerId"))
	requestID, _ := uuid.Parse(c.Param("id"))

	result := h.db.Model(&ReverseAuctionOffer{}).
		Where("id = ? AND request_id = ? AND seller_id = ? AND status IN ?", offerID, requestID, uid, []OfferStatus{OfferPending, OfferCountered}).
		Update("status", OfferWithdrawn)
	if result.RowsAffected == 0 {
		response.NotFound(c, "pending offer owned by you")
		return
	}
	response.OK(c, gin.H{"message": "offer withdrawn"})
}

// PUT /api/v1/reverse-auctions/:id/offers/:offerId/counter — buyer counters a seller's offer
func (h *Handler) CounterOffer(c *gin.Context) {
	uid := c.GetString("user_id")
	requestID, _ := uuid.Parse(c.Param("id"))
	offerID, _ := uuid.Parse(c.Param("offerId"))

	var req struct {
		CounterPrice float64 `json:"counter_price" binding:"required,gt=0"`
		Message      string  `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var r ReverseAuctionRequest
	if err := h.db.Where("id = ? AND buyer_id = ? AND status = ?", requestID, uid, RequestOpen).First(&r).Error; err != nil {
		response.NotFound(c, "open request owned by you")
		return
	}

	var offer ReverseAuctionOffer
	if err := h.db.Where("id = ? AND request_id = ? AND status = ?", offerID, requestID, OfferPending).First(&offer).Error; err != nil {
		response.NotFound(c, "pending offer")
		return
	}

	if offer.ExpiresAt != nil && time.Now().After(*offer.ExpiresAt) {
		h.db.Model(&offer).Update("status", OfferExpired)
		response.BadRequest(c, "offer has expired")
		return
	}

	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)
	h.db.Model(&offer).Updates(map[string]any{
		"status":        OfferCountered,
		"counter_price": req.CounterPrice,
		"message":       req.Message,
		"expires_at":    expiresAt,
		"responded_at":  now,
	})

	response.OK(c, gin.H{"message": "counter offer sent", "offer": offer})
}

// PUT /api/v1/reverse-auctions/:id/offers/:offerId/respond — seller accepts or declines a counter
func (h *Handler) RespondToCounter(c *gin.Context) {
	uid := c.GetString("user_id")
	offerID, _ := uuid.Parse(c.Param("offerId"))
	requestID, _ := uuid.Parse(c.Param("id"))

	var req struct {
		Accept bool `json:"accept"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var offer ReverseAuctionOffer
	if err := h.db.Where("id = ? AND request_id = ? AND seller_id = ? AND status = ?",
		offerID, requestID, uid, OfferCountered).First(&offer).Error; err != nil {
		response.NotFound(c, "countered offer owned by you")
		return
	}

	if offer.ExpiresAt != nil && time.Now().After(*offer.ExpiresAt) {
		h.db.Model(&offer).Update("status", OfferExpired)
		response.BadRequest(c, "counter offer has expired")
		return
	}

	now := time.Now()

	if req.Accept {
		// Seller accepts the counter — same as buyer accepting the offer at counter price
		var r ReverseAuctionRequest
		h.db.Where("id = ?", requestID).First(&r)

		// ── Atomic: accept + hold funds in ONE transaction (M-02 fix) ─────────
		if txErr := h.db.Transaction(func(tx *gorm.DB) error {
			tx.Model(&offer).Updates(map[string]any{
				"status":       OfferAccepted,
				"price":        *offer.CounterPrice,
				"responded_at": now,
			})
			tx.Model(&ReverseAuctionOffer{}).
				Where("request_id = ? AND id != ? AND status IN ?", requestID, offerID, []OfferStatus{OfferPending, OfferCountered}).
				Updates(map[string]any{"status": OfferRejected, "responded_at": now})
			tx.Model(&r).Update("status", RequestFulfilled)
			_, err := wallet.HoldFunds(tx, r.BuyerID, offer.SellerID, *offer.CounterPrice, "USD", "REVERSE_AUCTION", requestID.String())
			if err != nil {
				return fmt.Errorf("escrow_hold_failed: %w", err)
			}
			return nil
		}); txErr != nil {
			if len(txErr.Error()) > 19 && txErr.Error()[:19] == "escrow_hold_failed:" {
				response.BadRequest(c, "Could not secure buyer funds for this deal")
			} else {
				response.InternalError(c, txErr)
			}
			return
		}

		response.OK(c, gin.H{"message": "counter accepted, deal closed", "final_price": *offer.CounterPrice})
	} else {
		h.db.Model(&offer).Updates(map[string]any{
			"status":       OfferRejected,
			"responded_at": now,
		})
		response.OK(c, gin.H{"message": "counter declined"})
	}
}

// GET /api/v1/reverse-auctions/my/sent — offers I made as a seller
func (h *Handler) MySentOffers(c *gin.Context) {
	uid := c.GetString("user_id")
	status := c.Query("status")

	var offers []ReverseAuctionOffer
	q := h.db.Where("seller_id = ?", uid)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Order("created_at DESC").Limit(50).Find(&offers)
	response.OK(c, offers)
}

// GET /api/v1/reverse-auctions/my/received — offers I received as a buyer
func (h *Handler) MyReceivedOffers(c *gin.Context) {
	uid := c.GetString("user_id")
	status := c.Query("status")

	var requests []ReverseAuctionRequest
	h.db.Where("buyer_id = ?", uid).Select("id").Find(&requests)
	if len(requests) == 0 {
		response.OK(c, []any{})
		return
	}

	requestIDs := make([]uuid.UUID, len(requests))
	for i, r := range requests {
		requestIDs[i] = r.ID
	}

	var offers []ReverseAuctionOffer
	q := h.db.Where("request_id IN ?", requestIDs)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Order("created_at DESC").Limit(100).Find(&offers)
	response.OK(c, offers)
}
