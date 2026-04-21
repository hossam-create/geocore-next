package users

import (
	"strings"

	"github.com/geocore-next/backend/internal/security"
	"github.com/geocore-next/backend/pkg/reputation"
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
	return &Handler{db, rdb}
}

func (h *Handler) GetProfile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	var user User
	if err := h.db.First(&user, "id = ?", id).Error; err != nil {
		response.NotFound(c, "User")
		return
	}
	response.OK(c, user.ToPublic())
}

func (h *Handler) UpdateMe(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	var req struct {
		Name     string `json:"name" binding:"omitempty,min=2,max=100"`
		Bio      string `json:"bio" binding:"omitempty,max=2000"`
		Location string `json:"location" binding:"omitempty,max=120"`
		Language string `json:"language" binding:"omitempty,min=2,max=10"`
		Currency string `json:"currency" binding:"omitempty,len=3"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Sanitize user-controlled strings before persistence
	req.Name = security.SanitizeText(req.Name)
	req.Bio = security.SanitizeHTML(req.Bio)
	req.Location = security.SanitizeText(req.Location)
	req.Language = security.SanitizeText(req.Language)
	req.Currency = strings.ToUpper(security.SanitizeText(req.Currency))
	var user User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		response.NotFound(c, "User")
		return
	}
	if req.Name != "" {
		user.Name = req.Name
	}
	user.Bio = req.Bio
	user.Location = req.Location
	if req.Language != "" {
		user.Language = req.Language
	}
	if req.Currency != "" {
		user.Currency = req.Currency
	}
	h.db.Save(&user)
	response.OK(c, user)
}

func (h *Handler) GetMe(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	var user User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		response.NotFound(c, "User")
		return
	}
	response.OK(c, user)
}

// NotificationPreferences represents user notification settings
type NotificationPreferences struct {
	// Email
	EmailNewMessage    bool `json:"email_new_message"`
	EmailAuctionOutbid bool `json:"email_auction_outbid"`
	EmailOrderUpdate   bool `json:"email_order_update"`
	EmailPriceDrop     bool `json:"email_price_drop"`
	EmailPromoOffers   bool `json:"email_promo_offers"`

	// Push
	PushNewMessage    bool `json:"push_new_message"`
	PushAuctionOutbid bool `json:"push_auction_outbid"`
	PushOrderUpdate   bool `json:"push_order_update"`
	PushPriceDrop     bool `json:"push_price_drop"`
	PushPromoOffers   bool `json:"push_promo_offers"`

	// SMS
	SMSNewMessage    bool `json:"sms_new_message"`
	SMSAuctionOutbid bool `json:"sms_auction_outbid"`
	SMSOrderUpdate   bool `json:"sms_order_update"`
	SMSPriceDrop     bool `json:"sms_price_drop"`
	SMSPromoOffers   bool `json:"sms_promo_offers"`
}

// GetNotificationPreferences returns user's notification settings
func (h *Handler) GetNotificationPreferences(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var prefs NotificationPreferences
	// Default all to true if not set
	err := h.db.Table("notification_preferences").
		Where("user_id = ?", userID).
		First(&prefs).Error
	if err != nil {
		// Return defaults
		prefs = NotificationPreferences{
			EmailNewMessage: true, EmailAuctionOutbid: true, EmailOrderUpdate: true, EmailPriceDrop: true, EmailPromoOffers: true,
			PushNewMessage: true, PushAuctionOutbid: true, PushOrderUpdate: true, PushPriceDrop: true, PushPromoOffers: true,
			SMSNewMessage: false, SMSAuctionOutbid: false, SMSOrderUpdate: true, SMSPriceDrop: false, SMSPromoOffers: false,
		}
	}
	response.OK(c, prefs)
}

// UpdateNotificationPreferences updates user's notification settings
func (h *Handler) UpdateNotificationPreferences(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var req NotificationPreferences
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Upsert preferences
	err := h.db.Exec(`
		INSERT INTO notification_preferences (user_id, email_new_message, email_auction_outbid, email_order_update, email_price_drop, email_promo_offers,
			push_new_message, push_auction_outbid, push_order_update, push_price_drop, push_promo_offers,
			sms_new_message, sms_auction_outbid, sms_order_update, sms_price_drop, sms_promo_offers, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			email_new_message = EXCLUDED.email_new_message,
			email_auction_outbid = EXCLUDED.email_auction_outbid,
			email_order_update = EXCLUDED.email_order_update,
			email_price_drop = EXCLUDED.email_price_drop,
			email_promo_offers = EXCLUDED.email_promo_offers,
			push_new_message = EXCLUDED.push_new_message,
			push_auction_outbid = EXCLUDED.push_auction_outbid,
			push_order_update = EXCLUDED.push_order_update,
			push_price_drop = EXCLUDED.push_price_drop,
			push_promo_offers = EXCLUDED.push_promo_offers,
			sms_new_message = EXCLUDED.sms_new_message,
			sms_auction_outbid = EXCLUDED.sms_auction_outbid,
			sms_order_update = EXCLUDED.sms_order_update,
			sms_price_drop = EXCLUDED.sms_price_drop,
			sms_promo_offers = EXCLUDED.sms_promo_offers,
			updated_at = NOW()
	`, userID, req.EmailNewMessage, req.EmailAuctionOutbid, req.EmailOrderUpdate, req.EmailPriceDrop, req.EmailPromoOffers,
		req.PushNewMessage, req.PushAuctionOutbid, req.PushOrderUpdate, req.PushPriceDrop, req.PushPromoOffers,
		req.SMSNewMessage, req.SMSAuctionOutbid, req.SMSOrderUpdate, req.SMSPriceDrop, req.SMSPromoOffers).Error

	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "Notification preferences updated"})
}

// GetReputation returns the public reputation profile for any user.
// GET /users/:id/reputation
func (h *Handler) GetReputation(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	profile, err := reputation.Get(h.db, id)
	if err != nil {
		response.NotFound(c, "Reputation")
		return
	}
	response.OK(c, profile)
}

// GetMyReputation returns the authenticated user's reputation and triggers a refresh.
// GET /users/me/reputation
func (h *Handler) GetMyReputation(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	profile, err := reputation.Refresh(h.db, userID)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, profile)
}
