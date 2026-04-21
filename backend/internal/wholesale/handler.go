package wholesale

import (
	"fmt"
	"math"
	"net/http"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Handler ────────────────────────────────────────────────────────────────────

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// ── Seller Endpoints ───────────────────────────────────────────────────────────

// POST /api/v1/wholesale/sellers — register as wholesale seller
func (h *Handler) RegisterSeller(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, err := uuid.Parse(fmt.Sprintf("%v", userID))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req struct {
		CompanyName        string   `json:"company_name" binding:"required"`
		TaxID              string   `json:"tax_id"`
		BusinessType       string   `json:"business_type"`
		Categories         []string `json:"categories"`
		MinOrderValueCents int64    `json:"min_order_value_cents"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	seller := WholesaleSeller{
		UserID:             uid,
		CompanyName:        req.CompanyName,
		TaxID:              req.TaxID,
		BusinessType:       req.BusinessType,
		Categories:         req.Categories,
		MinOrderValueCents: req.MinOrderValueCents,
		Status:             "pending",
	}

	if err := h.db.Create(&seller).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	c.JSON(http.StatusCreated, seller)
}

// GET /api/v1/wholesale/sellers/me — get my seller profile
func (h *Handler) GetMySellerProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := uuid.Parse(fmt.Sprintf("%v", userID))

	var seller WholesaleSeller
	if err := h.db.Where("user_id = ?", uid).First(&seller).Error; err != nil {
		response.NotFound(c, "wholesale seller profile not found")
		return
	}

	response.OK(c, seller)
}

// ── Listing Endpoints ─────────────────────────────────────────────────────────

// POST /api/v1/wholesale/listings — create a wholesale listing
func (h *Handler) CreateListing(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := uuid.Parse(fmt.Sprintf("%v", userID))

	// Verify seller is active
	var seller WholesaleSeller
	if err := h.db.Where("user_id = ? AND status = ?", uid, "active").First(&seller).Error; err != nil {
		response.BadRequest(c, "active wholesale seller profile required")
		return
	}

	var listing WholesaleListing
	if err := c.ShouldBindJSON(&listing); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	listing.ID = uuid.Nil
	listing.SellerID = uid
	listing.Status = "active"
	listing.IsVerified = seller.IsVerified

	if listing.MOQ < 1 {
		listing.MOQ = 1
	}
	if listing.Currency == "" {
		listing.Currency = "EGP"
	}

	if err := h.db.Create(&listing).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Update seller stats
	h.db.Model(&WholesaleSeller{}).Where("id = ?", seller.ID).
		UpdateColumn("total_listings", gorm.Expr("total_listings + 1"))

	c.JSON(http.StatusCreated, listing)
}

// GET /api/v1/wholesale/listings — browse wholesale listings
func (h *Handler) ListListings(c *gin.Context) {
	page, pageSize := pagination(c)

	query := h.db.Where("status = ?", "active")
	if cat := c.Query("category"); cat != "" {
		query = query.Where("category_slug = ?", cat)
	}
	if verified := c.Query("verified"); verified == "true" {
		query = query.Where("is_verified = ?", true)
	}
	if minMOQ := c.Query("min_moq"); minMOQ != "" {
		query = query.Where("moq >= ?", minMOQ)
	}

	var total int64
	query.Model(&WholesaleListing{}).Count(&total)

	var listings []WholesaleListing
	query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&listings)

	response.OK(c, gin.H{
		"items":     listings,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GET /api/v1/wholesale/listings/:id — get a single wholesale listing
func (h *Handler) GetListing(c *gin.Context) {
	id := c.Param("id")

	var listing WholesaleListing
	if err := h.db.Where("id = ?", id).First(&listing).Error; err != nil {
		response.NotFound(c, "wholesale listing not found")
		return
	}

	// Increment views
	h.db.Model(&listing).UpdateColumn("views_count", gorm.Expr("views_count + 1"))

	response.OK(c, listing)
}

// ── Order Endpoints ────────────────────────────────────────────────────────────

// POST /api/v1/wholesale/orders — place a wholesale order
func (h *Handler) CreateOrder(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := uuid.Parse(fmt.Sprintf("%v", userID))

	var req struct {
		ListingID string `json:"listing_id" binding:"required"`
		Quantity  int    `json:"quantity" binding:"required,min=1"`
		Notes     string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Fetch listing
	listingID, _ := uuid.Parse(req.ListingID)
	var listing WholesaleListing
	if err := h.db.Where("id = ? AND status = ?", listingID, "active").First(&listing).Error; err != nil {
		response.NotFound(c, "wholesale listing not found or inactive")
		return
	}

	// Validate MOQ
	if req.Quantity < listing.MOQ {
		response.BadRequest(c, fmt.Sprintf("quantity must be at least %d (MOQ)", listing.MOQ))
		return
	}

	// Validate max order quantity
	if listing.MaxOrderQuantity > 0 && req.Quantity > listing.MaxOrderQuantity {
		response.BadRequest(c, fmt.Sprintf("quantity cannot exceed %d", listing.MaxOrderQuantity))
		return
	}

	// Validate availability
	if listing.AvailableUnits > 0 && req.Quantity > listing.AvailableUnits {
		response.BadRequest(c, fmt.Sprintf("only %d units available", listing.AvailableUnits))
		return
	}

	// Calculate tier price
	unitPrice := listing.UnitPriceCents
	for _, tier := range listing.TierPricing {
		if req.Quantity >= tier.MinQuantity && (tier.MaxQuantity == 0 || req.Quantity <= tier.MaxQuantity) {
			unitPrice = tier.UnitPriceCents
		}
	}

	totalPrice := unitPrice * int64(req.Quantity)

	// Calculate shipping
	shipping := listing.ShippingPerUnitCents * int64(req.Quantity)
	if listing.FreeShippingMOQ > 0 && req.Quantity >= listing.FreeShippingMOQ {
		shipping = 0
	}

	order := WholesaleOrder{
		BuyerID:         uid,
		SellerID:        listing.SellerID,
		ListingID:       listingID,
		Quantity:        req.Quantity,
		UnitPriceCents:  unitPrice,
		TotalPriceCents: totalPrice,
		Currency:        listing.Currency,
		ShippingCents:   shipping,
		Status:          "pending",
		Notes:           req.Notes,
	}

	if err := h.db.Create(&order).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Update listing stats
	h.db.Model(&listing).UpdateColumn("orders_count", gorm.Expr("orders_count + 1"))

	c.JSON(http.StatusCreated, order)
}

// GET /api/v1/wholesale/orders — list my wholesale orders
func (h *Handler) ListMyOrders(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := uuid.Parse(fmt.Sprintf("%v", userID))
	page, pageSize := pagination(c)

	role := c.Query("role") // "buyer" or "seller"

	query := h.db.Model(&WholesaleOrder{})
	if role == "seller" {
		query = query.Where("seller_id = ?", uid)
	} else {
		query = query.Where("buyer_id = ?", uid)
	}

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var orders []WholesaleOrder
	query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&orders)

	response.OK(c, gin.H{
		"items":     orders,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// PATCH /api/v1/wholesale/orders/:id/respond — seller responds to an order
func (h *Handler) RespondToOrder(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := uuid.Parse(fmt.Sprintf("%v", userID))
	orderID := c.Param("id")

	var req struct {
		Response          string `json:"response" binding:"required"` // accepted, rejected, counter_offer
		CounterOfferCents int64  `json:"counter_offer_cents"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var order WholesaleOrder
	if err := h.db.Where("id = ? AND seller_id = ?", orderID, uid).First(&order).Error; err != nil {
		response.NotFound(c, "order not found")
		return
	}

	if order.Status != "pending" {
		response.BadRequest(c, "order is not in pending state")
		return
	}

	updates := map[string]interface{}{
		"seller_response":     req.Response,
		"counter_offer_cents": req.CounterOfferCents,
	}

	switch req.Response {
	case "accepted":
		updates["status"] = "confirmed"
	case "rejected":
		updates["status"] = "cancelled"
	case "counter_offer":
		updates["status"] = "negotiating"
	default:
		response.BadRequest(c, "response must be accepted, rejected, or counter_offer")
		return
	}

	h.db.Model(&order).Updates(updates)
	response.OK(c, order)
}

// ── Admin Endpoints ────────────────────────────────────────────────────────────

// GET /admin/wholesale/sellers — list all wholesale sellers
func (h *Handler) AdminListSellers(c *gin.Context) {
	var sellers []WholesaleSeller
	status := c.Query("status")
	query := h.db.Order("created_at DESC")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Find(&sellers)
	response.OK(c, sellers)
}

// PATCH /admin/wholesale/sellers/:id/verify — verify or suspend a seller
func (h *Handler) AdminVerifySeller(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Action string `json:"action" binding:"required"` // verify, suspend, activate
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	adminID, _ := c.Get("user_id")
	adminUID, _ := uuid.Parse(fmt.Sprintf("%v", adminID))

	updates := map[string]interface{}{}
	switch req.Action {
	case "verify":
		updates["is_verified"] = true
		updates["status"] = "active"
		updates["verified_by"] = adminUID
		updates["verified_at"] = gorm.Expr("NOW()")
	case "suspend":
		updates["status"] = "suspended"
		updates["is_verified"] = false
	case "activate":
		updates["status"] = "active"
	default:
		response.BadRequest(c, "action must be verify, suspend, or activate")
		return
	}

	h.db.Model(&WholesaleSeller{}).Where("id = ?", id).Updates(updates)
	response.OK(c, gin.H{"updated": true})
}

// GET /admin/wholesale/listings — admin browse all wholesale listings
func (h *Handler) AdminListListings(c *gin.Context) {
	var listings []WholesaleListing
	h.db.Order("created_at DESC").Find(&listings)
	response.OK(c, listings)
}

// ── Helpers ────────────────────────────────────────────────────────────────────

func pagination(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 20
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
		if page < 1 {
			page = 1
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}
	}
	return
}

// CalculateTierPrice returns the unit price for a given quantity based on tier pricing.
func CalculateTierPrice(basePriceCents int64, tiers []PriceTier, quantity int) int64 {
	price := basePriceCents
	for _, tier := range tiers {
		if quantity >= tier.MinQuantity && (tier.MaxQuantity == 0 || quantity <= tier.MaxQuantity) {
			price = tier.UnitPriceCents
		}
	}
	return price
}

// CalculateTotal returns the total price for a wholesale order.
func CalculateTotal(unitPriceCents int64, quantity int) int64 {
	return unitPriceCents * int64(quantity)
}

// CalculateDiscount returns the discount percentage from base price.
func CalculateDiscount(basePriceCents, tierPriceCents int64) float64 {
	if basePriceCents == 0 {
		return 0
	}
	return math.Round(float64(basePriceCents-tierPriceCents)/float64(basePriceCents)*10000) / 100
}
