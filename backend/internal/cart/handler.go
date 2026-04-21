package cart

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Handler struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
	return &Handler{db: db, rdb: rdb}
}

type addItemRequest struct {
	ListingID string `json:"listing_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
}

type listingSnapshot struct {
	ID        uuid.UUID  `gorm:"column:id"`
	Title     string     `gorm:"column:title"`
	Status    string     `gorm:"column:status"`
	Price     *float64   `gorm:"column:price"`
	Currency  string     `gorm:"column:currency"`
	ExpiresAt *time.Time `gorm:"column:expires_at"`
}

func (h *Handler) AddItem(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req addItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	listingID, err := uuid.Parse(req.ListingID)
	if err != nil {
		response.BadRequest(c, "invalid listing_id")
		return
	}

	var listing listingSnapshot
	err = h.db.WithContext(c.Request.Context()).
		Table("listings").
		Select("id, title, status, price, currency, expires_at").
		Where("id = ?", listingID).
		First(&listing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "Listing")
			return
		}
		response.InternalError(c, err)
		return
	}

	if listing.Status == "sold" || listing.Status == "expired" {
		response.BadRequest(c, "listing is not available")
		return
	}
	if listing.ExpiresAt != nil && listing.ExpiresAt.Before(time.Now()) {
		response.BadRequest(c, "listing has expired")
		return
	}
	if listing.Price == nil || *listing.Price <= 0 {
		response.BadRequest(c, "listing is not purchasable")
		return
	}

	ctx := c.Request.Context()
	cart, err := h.loadCart(ctx, userID)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	updated := false
	for i := range cart.Items {
		if cart.Items[i].ListingID == listingID.String() {
			cart.Items[i].Quantity += req.Quantity
			cart.Items[i].UnitPrice = *listing.Price
			cart.Items[i].Currency = listing.Currency
			cart.Items[i].Title = listing.Title
			cart.Items[i].Subtotal = float64(cart.Items[i].Quantity) * cart.Items[i].UnitPrice
			updated = true
			break
		}
	}

	if !updated {
		item := CartItem{
			ListingID: listingID.String(),
			Title:     listing.Title,
			Currency:  listing.Currency,
			UnitPrice: *listing.Price,
			Quantity:  req.Quantity,
			Subtotal:  float64(req.Quantity) * *listing.Price,
		}
		cart.Items = append(cart.Items, item)
	}

	h.recalculate(cart)
	if err := h.saveCart(ctx, userID, cart); err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, cart)
}

func (h *Handler) GetCart(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	cart, err := h.loadCart(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	h.recalculate(cart)
	response.OK(c, cart)
}

func (h *Handler) RemoveItem(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	listingID := c.Param("listing_id")
	if _, err := uuid.Parse(listingID); err != nil {
		response.BadRequest(c, "invalid listing_id")
		return
	}

	ctx := c.Request.Context()
	cart, err := h.loadCart(ctx, userID)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	filtered := make([]CartItem, 0, len(cart.Items))
	for _, item := range cart.Items {
		if item.ListingID != listingID {
			filtered = append(filtered, item)
		}
	}
	cart.Items = filtered
	h.recalculate(cart)

	if len(cart.Items) == 0 {
		if err := h.rdb.Del(ctx, h.cartKey(userID)).Err(); err != nil {
			response.InternalError(c, err)
			return
		}
		response.OK(c, gin.H{"message": "item removed", "item_count": 0})
		return
	}

	if err := h.saveCart(ctx, userID, cart); err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, cart)
}

func (h *Handler) ClearCart(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	if err := h.rdb.Del(c.Request.Context(), h.cartKey(userID)).Err(); err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, gin.H{"message": "cart cleared", "item_count": 0})
}

func (h *Handler) cartKey(userID string) string {
	return "cart:" + userID
}

func (h *Handler) loadCart(ctx context.Context, userID string) (*Cart, error) {
	key := h.cartKey(userID)
	val, err := h.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return &Cart{Items: []CartItem{}, UpdatedAt: time.Now()}, nil
		}
		return nil, err
	}

	var cart Cart
	if err := json.Unmarshal([]byte(val), &cart); err != nil {
		return &Cart{Items: []CartItem{}, UpdatedAt: time.Now()}, nil
	}
	if cart.Items == nil {
		cart.Items = []CartItem{}
	}
	return &cart, nil
}

func (h *Handler) saveCart(ctx context.Context, userID string, cart *Cart) error {
	cart.UpdatedAt = time.Now()
	payload, err := json.Marshal(cart)
	if err != nil {
		return err
	}
	return h.rdb.Set(ctx, h.cartKey(userID), payload, cartTTL).Err()
}

func (h *Handler) recalculate(cart *Cart) {
	total := 0.0
	itemCount := 0
	currency := ""
	for i := range cart.Items {
		if cart.Items[i].Quantity < 1 {
			cart.Items[i].Quantity = 1
		}
		cart.Items[i].Subtotal = cart.Items[i].UnitPrice * float64(cart.Items[i].Quantity)
		total += cart.Items[i].Subtotal
		itemCount += cart.Items[i].Quantity
		if currency == "" {
			currency = cart.Items[i].Currency
		}
	}
	cart.Total = total
	cart.ItemCount = itemCount
	cart.Currency = currency
}
