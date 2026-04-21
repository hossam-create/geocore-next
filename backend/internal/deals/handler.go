package deals

import (
	"strconv"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// CreateDeal creates a new promotional deal for a listing
func (h *Handler) CreateDeal(c *gin.Context) {
	sellerID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Unauthorized(c)
		return
	}

	var req DealCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	listingID, err := uuid.Parse(req.ListingID)
	if err != nil {
		response.BadRequest(c, "Invalid listing ID")
		return
	}

	startAt, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		response.BadRequest(c, "Invalid start_at format (use RFC3339)")
		return
	}

	endAt, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		response.BadRequest(c, "Invalid end_at format (use RFC3339)")
		return
	}

	// Validate times
	if endAt.Before(startAt) {
		response.BadRequest(c, "end_at must be after start_at")
		return
	}

	// Get listing and verify ownership
	var listing struct {
		ID       uuid.UUID
		Title    string
		Price    float64
		Currency string
		UserID   uuid.UUID
	}
	if err := h.db.Table("listings").
		Select("id, title, price, currency, user_id").
		Where("id = ?", listingID).
		First(&listing).Error; err != nil {
		response.NotFound(c, "Listing")
		return
	}

	if listing.UserID != sellerID {
		response.Forbidden(c)
		return
	}

	// Validate deal price
	if req.DealPrice >= listing.Price {
		response.BadRequest(c, "Deal price must be less than original price")
		return
	}

	// Check for existing active deal
	var existingCount int64
	h.db.Model(&Deal{}).Where("listing_id = ? AND status IN ?", listingID, []DealStatus{DealStatusActive, DealStatusScheduled}).Count(&existingCount)
	if existingCount > 0 {
		response.BadRequest(c, "Listing already has an active or scheduled deal")
		return
	}

	deal := Deal{
		ID:            uuid.New(),
		ListingID:     listingID,
		SellerID:      sellerID,
		OriginalPrice: listing.Price,
		DealPrice:     req.DealPrice,
		DiscountPct:   CalculateDiscount(listing.Price, req.DealPrice),
		StartAt:       startAt,
		EndAt:         endAt,
		Status:        DealStatusScheduled,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := h.db.Create(&deal).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	response.Created(c, deal)
}

// GetDeals returns all active deals, sorted by discount percentage
func (h *Handler) GetDeals(c *gin.Context) {
	now := time.Now()

	var deals []Deal
	err := h.db.Table("deals d").
		Select(`d.id, d.listing_id, d.seller_id, d.original_price, d.deal_price, d.discount_pct, 
			d.start_at, d.end_at, d.status,
			COALESCE(l.title, '') as listing_title, COALESCE(l.currency, '') as currency,
			COALESCE((SELECT li.url FROM listing_images li WHERE li.listing_id = l.id ORDER BY li.sort_order ASC LIMIT 1), '') as listing_image,
			COALESCE(u.name, '') as seller_name`).
		Joins("LEFT JOIN listings l ON l.id = d.listing_id").
		Joins("LEFT JOIN users u ON u.id = d.seller_id").
		Where("d.start_at <= ? AND d.end_at > ?", now, now).
		Where("d.status != ?", DealStatusCancelled).
		Order("d.discount_pct DESC").
		Limit(50).
		Find(&deals).Error

	if err != nil {
		response.InternalError(c, err)
		return
	}

	// Transform to public response
	var resp []DealPublicResponse
	for _, d := range deals {
		timeRemaining := formatTimeRemaining(d.EndAt)
		resp = append(resp, DealPublicResponse{
			ID:            d.ID.String(),
			ListingID:     d.ListingID.String(),
			ListingTitle:  d.ListingTitle,
			ListingImage:  d.ListingImage,
			SellerID:      d.SellerID.String(),
			SellerName:    d.SellerName,
			OriginalPrice: d.OriginalPrice,
			DealPrice:     d.DealPrice,
			DiscountPct:   d.DiscountPct,
			Currency:      d.Currency,
			StartAt:       d.StartAt,
			EndAt:         d.EndAt,
			Status:        string(DealStatusActive),
			TimeRemaining: timeRemaining,
		})
	}

	response.OK(c, resp)
}

// GetDeal returns a single deal by ID
func (h *Handler) GetDeal(c *gin.Context) {
	dealID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid deal ID")
		return
	}

	var deal Deal
	err = h.db.Table("deals d").
		Select(`d.id, d.listing_id, d.seller_id, d.original_price, d.deal_price, d.discount_pct, 
			d.start_at, d.end_at, d.status,
			COALESCE(l.title, '') as listing_title, COALESCE(l.currency, '') as currency, l.condition,
			COALESCE((SELECT li.url FROM listing_images li WHERE li.listing_id = l.id ORDER BY li.sort_order ASC LIMIT 1), '') as listing_image,
			COALESCE(u.name, '') as seller_name`).
		Joins("LEFT JOIN listings l ON l.id = d.listing_id").
		Joins("LEFT JOIN users u ON u.id = d.seller_id").
		Where("d.id = ?", dealID).
		First(&deal).Error

	if err != nil {
		response.NotFound(c, "Deal")
		return
	}

	timeRemaining := formatTimeRemaining(deal.EndAt)
	resp := DealPublicResponse{
		ID:            deal.ID.String(),
		ListingID:     deal.ListingID.String(),
		ListingTitle:  deal.ListingTitle,
		ListingImage:  deal.ListingImage,
		SellerID:      deal.SellerID.String(),
		SellerName:    deal.SellerName,
		OriginalPrice: deal.OriginalPrice,
		DealPrice:     deal.DealPrice,
		DiscountPct:   deal.DiscountPct,
		Currency:      deal.Currency,
		StartAt:       deal.StartAt,
		EndAt:         deal.EndAt,
		Status:        string(deal.GetStatus()),
		TimeRemaining: timeRemaining,
	}

	response.OK(c, resp)
}

// GetMyDeals returns deals for the authenticated seller
func (h *Handler) GetMyDeals(c *gin.Context) {
	sellerID := c.GetString("user_id")

	var deals []Deal
	err := h.db.Table("deals d").
		Select(`d.id, d.listing_id, d.seller_id, d.original_price, d.deal_price, d.discount_pct, 
			d.start_at, d.end_at, d.status,
			COALESCE(l.title, '') as listing_title,
			COALESCE((SELECT li.url FROM listing_images li WHERE li.listing_id = l.id ORDER BY li.sort_order ASC LIMIT 1), '') as listing_image`).
		Joins("LEFT JOIN listings l ON l.id = d.listing_id").
		Where("d.seller_id = ?", sellerID).
		Order("d.created_at DESC").
		Find(&deals).Error

	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, deals)
}

// CancelDeal cancels a scheduled or active deal
func (h *Handler) CancelDeal(c *gin.Context) {
	sellerID := c.GetString("user_id")
	dealID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid deal ID")
		return
	}

	result := h.db.Model(&Deal{}).
		Where("id = ? AND seller_id = ?", dealID, sellerID).
		Update("status", DealStatusCancelled)

	if result.Error != nil {
		response.InternalError(c, result.Error)
		return
	}

	if result.RowsAffected == 0 {
		response.NotFound(c, "Deal")
		return
	}

	response.OK(c, gin.H{"message": "Deal cancelled"})
}

// UpdateExpiredDeals marks expired deals (called by cron)
func UpdateExpiredDeals(db *gorm.DB) error {
	now := time.Now()
	return db.Model(&Deal{}).
		Where("end_at < ? AND status IN ?", now, []DealStatus{DealStatusActive, DealStatusScheduled}).
		Update("status", DealStatusExpired).Error
}

// ActivateScheduledDeals marks scheduled deals as active (called by cron)
func ActivateScheduledDeals(db *gorm.DB) error {
	now := time.Now()
	return db.Model(&Deal{}).
		Where("start_at <= ? AND end_at > ? AND status = ?", now, now, DealStatusScheduled).
		Update("status", DealStatusActive).Error
}

// formatTimeRemaining returns a human-readable time remaining string
func formatTimeRemaining(endAt time.Time) string {
	if time.Now().After(endAt) {
		return "Expired"
	}

	remaining := time.Until(endAt)
	if remaining < 0 {
		return "Expired"
	}

	days := int(remaining.Hours()) / 24
	hours := int(remaining.Hours()) % 24
	minutes := int(remaining.Minutes()) % 60

	if days > 0 {
		return formatPlural(days, "day", "days") + " left"
	}
	if hours > 0 {
		return formatPlural(hours, "hour", "hours") + " left"
	}
	return formatPlural(minutes, "minute", "minutes") + " left"
}

func formatPlural(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return strconv.Itoa(n) + " " + plural
}
