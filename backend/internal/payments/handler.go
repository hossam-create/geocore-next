package payments

import (
	"os"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
	"gorm.io/gorm"
)

type Handler struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	return &Handler{db, rdb}
}

func (h *Handler) CreatePaymentIntent(c *gin.Context) {
	var req struct {
		Amount    int64  `json:"amount" binding:"required,min=50"` // in cents
		Currency  string `json:"currency"`
		ListingID string `json:"listing_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	currency := req.Currency
	if currency == "" {
		currency = "usd"
	}
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(req.Amount),
		Currency: stripe.String(currency),
	}
	if req.ListingID != "" {
		params.AddMetadata("listing_id", req.ListingID)
	}
	pi, err := paymentintent.New(params)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{
		"client_secret": pi.ClientSecret,
		"payment_intent_id": pi.ID,
	})
}

func (h *Handler) GetPublishableKey(c *gin.Context) {
	response.OK(c, gin.H{"key": os.Getenv("STRIPE_PUBLISHABLE_KEY")})
}
