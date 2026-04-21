package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/auctions"
	"github.com/geocore-next/backend/internal/listings"
	"github.com/geocore-next/backend/internal/order"
	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ════════════════════════════════════════════════════════════════════════════
// SECTION 1 — Enhanced Dashboard
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) GetDashboardFull(c *gin.Context) {
	today := time.Now().Truncate(24 * time.Hour)
	weekAgo := today.AddDate(0, 0, -7)
	monthAgo := today.AddDate(0, -1, 0)

	// Stats
	var stats DashboardStats
	h.db.Model(&users.User{}).Count(&stats.TotalUsers)
	h.db.Model(&users.User{}).Where("created_at >= ?", today).Count(&stats.ActiveUsersToday)
	h.db.Model(&users.User{}).Where("created_at >= ?", weekAgo).Count(&stats.NewUsersThisWeek)
	h.db.Model(&listings.Listing{}).Count(&stats.TotalListings)
	h.db.Model(&listings.Listing{}).Where("status = ?", "active").Count(&stats.ActiveListings)
	h.db.Model(&listings.Listing{}).Where("status = ?", "pending").Count(&stats.PendingModeration)
	h.db.Model(&listings.Listing{}).Where("created_at >= ?", today).Count(&stats.NewListingsToday)
	h.db.Model(&auctions.Auction{}).Count(&stats.TotalAuctions)
	h.db.Model(&auctions.Auction{}).Where("status = ? AND ends_at > NOW()", "active").Count(&stats.LiveAuctions)

	// Charts: daily signups last 30d
	type dateCount struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	var dailySignups []dateCount
	h.db.Model(&users.User{}).
		Select("TO_CHAR(created_at, 'YYYY-MM-DD') as date, COUNT(*) as count").
		Where("created_at >= ?", monthAgo).
		Group("date").Order("date ASC").Scan(&dailySignups)

	// Charts: daily revenue last 30d
	type dateAmount struct {
		Date   string  `json:"date"`
		Amount float64 `json:"amount"`
	}
	var dailyRevenue []dateAmount
	h.db.Table("payments").
		Select("TO_CHAR(created_at, 'YYYY-MM-DD') as date, COALESCE(SUM(amount), 0) as amount").
		Where("status = ? AND created_at >= ?", "succeeded", monthAgo).
		Group("date").Order("date ASC").Scan(&dailyRevenue)

	// Charts: listings by category
	type catCount struct {
		Category string `json:"category"`
		Count    int64  `json:"count"`
	}
	var listingsByCat []catCount
	h.db.Table("listings").
		Select("categories.name_en as category, COUNT(listings.id) as count").
		Joins("LEFT JOIN categories ON categories.id = listings.category_id").
		Where("listings.status != 'deleted'").
		Group("categories.name_en").Order("count DESC").Limit(10).
		Scan(&listingsByCat)

	// Charts: listings by status
	type statusCount struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var listingsByStatus []statusCount
	h.db.Model(&listings.Listing{}).
		Select("status, COUNT(*) as count").
		Group("status").Scan(&listingsByStatus)

	statusMap := map[string]int64{}
	for _, s := range listingsByStatus {
		statusMap[s.Status] = s.Count
	}

	// Recent activity from audit log
	var recentActivity []AdminLog
	h.db.Order("created_at DESC").Limit(20).Find(&recentActivity)

	response.OK(c, gin.H{
		"stats": stats,
		"charts": gin.H{
			"daily_signups":        dailySignups,
			"daily_revenue":        dailyRevenue,
			"listings_by_category": listingsByCat,
			"listings_by_status":   statusMap,
		},
		"recent_activity": recentActivity,
	})
}

// ════════════════════════════════════════════════════════════════════════════
// SECTION 2 — Users: suspend, verify, role, group, listings, orders
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) SuspendUser(c *gin.Context) {
	result := h.db.Model(&users.User{}).Where("id = ?", c.Param("id")).
		Update("is_active", false)
	if result.RowsAffected == 0 {
		response.NotFound(c, "user")
		return
	}
	h.logAction(c, "suspend_user", "user", c.Param("id"), nil)
	response.OK(c, gin.H{"message": "User suspended."})
}

func (h *Handler) VerifyUser(c *gin.Context) {
	result := h.db.Model(&users.User{}).Where("id = ?", c.Param("id")).
		Update("is_verified", true)
	if result.RowsAffected == 0 {
		response.NotFound(c, "user")
		return
	}
	h.logAction(c, "verify_user", "user", c.Param("id"), nil)
	response.OK(c, gin.H{"message": "User verified."})
}

func (h *Handler) ChangeUserRole(c *gin.Context) {
	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result := h.db.Model(&users.User{}).Where("id = ?", c.Param("id")).
		Update("role", req.Role)
	if result.RowsAffected == 0 {
		response.NotFound(c, "user")
		return
	}
	h.logAction(c, "change_role", "user", c.Param("id"), gin.H{"role": req.Role})
	response.OK(c, gin.H{"message": "Role updated."})
}

func (h *Handler) ChangeUserGroup(c *gin.Context) {
	var req struct {
		GroupID int `json:"group_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result := h.db.Model(&users.User{}).Where("id = ?", c.Param("id")).
		Update("group_id", req.GroupID)
	if result.RowsAffected == 0 {
		response.NotFound(c, "user")
		return
	}
	h.logAction(c, "change_group", "user", c.Param("id"), gin.H{"group_id": req.GroupID})
	response.OK(c, gin.H{"message": "Group updated."})
}

func (h *Handler) GetUserListings(c *gin.Context) {
	page, perPage := paginationParams(c)
	uid := c.Param("id")
	var total int64
	h.db.Model(&listings.Listing{}).Where("user_id = ?", uid).Count(&total)
	var list []listings.Listing
	h.db.Where("user_id = ?", uid).
		Preload("Category").Preload("Images").
		Offset((page - 1) * perPage).Limit(perPage).
		Order("created_at DESC").Find(&list)
	response.OKMeta(c, list, response.Meta{Total: total, Page: page, PerPage: perPage, Pages: (total + int64(perPage) - 1) / int64(perPage)})
}

func (h *Handler) GetUserOrders(c *gin.Context) {
	page, perPage := paginationParams(c)
	uid := c.Param("id")
	var total int64
	h.db.Model(&order.Order{}).Where("buyer_id = ? OR seller_id = ?", uid, uid).Count(&total)
	var list []order.Order
	h.db.Where("buyer_id = ? OR seller_id = ?", uid, uid).
		Offset((page - 1) * perPage).Limit(perPage).
		Order("created_at DESC").Find(&list)
	response.OKMeta(c, list, response.Meta{Total: total, Page: page, PerPage: perPage, Pages: (total + int64(perPage) - 1) / int64(perPage)})
}

func (h *Handler) ImpersonateUser(c *gin.Context) {
	var user users.User
	if err := h.db.First(&user, "id = ?", c.Param("id")).Error; err != nil {
		response.NotFound(c, "user")
		return
	}
	h.logAction(c, "impersonate_user", "user", c.Param("id"), nil)
	response.OK(c, gin.H{
		"message":    "Impersonation session created. Use this in the frontend to switch context.",
		"user_id":    user.ID,
		"user_email": user.Email,
		"user_name":  user.Name,
	})
}

// ── User Groups CRUD ────────────────────────────────────────────────────────

func (h *Handler) ListUserGroups(c *gin.Context) {
	var groups []UserGroup
	h.db.Order("sort_order ASC").Find(&groups)
	response.OK(c, groups)
}

func (h *Handler) CreateUserGroup(c *gin.Context) {
	var g UserGroup
	if err := c.ShouldBindJSON(&g); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.db.Create(&g).Error; err != nil {
		response.BadRequest(c, "slug already exists or invalid input")
		return
	}
	h.logAction(c, "create_user_group", "user_group", fmt.Sprint(g.ID), gin.H{"name": g.Name})
	response.Created(c, g)
}

func (h *Handler) UpdateUserGroup(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var g UserGroup
	if err := h.db.First(&g, id).Error; err != nil {
		response.NotFound(c, "user_group")
		return
	}
	if err := c.ShouldBindJSON(&g); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.db.Save(&g)
	h.logAction(c, "update_user_group", "user_group", c.Param("id"), gin.H{"name": g.Name})
	response.OK(c, g)
}

func (h *Handler) DeleteUserGroup(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	result := h.db.Delete(&UserGroup{}, id)
	if result.RowsAffected == 0 {
		response.NotFound(c, "user_group")
		return
	}
	h.logAction(c, "delete_user_group", "user_group", c.Param("id"), nil)
	response.OK(c, gin.H{"message": "User group deleted."})
}

// ════════════════════════════════════════════════════════════════════════════
// SECTION 3 — Listings: pending, get, edit, feature, extend, extras, bulk
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) ListPendingListings(c *gin.Context) {
	page, perPage := paginationParams(c)
	q := h.db.Model(&listings.Listing{}).Where("status = ?", "pending").
		Preload("Category").Preload("Images")
	var total int64
	q.Count(&total)
	var list []listings.Listing
	q.Offset((page - 1) * perPage).Limit(perPage).Order("created_at ASC").Find(&list)
	response.OKMeta(c, list, response.Meta{Total: total, Page: page, PerPage: perPage, Pages: (total + int64(perPage) - 1) / int64(perPage)})
}

func (h *Handler) GetListing(c *gin.Context) {
	var listing listings.Listing
	if err := h.db.Preload("Category").Preload("Images").First(&listing, "id = ?", c.Param("id")).Error; err != nil {
		response.NotFound(c, "listing")
		return
	}
	// Extras attached
	var extras []ListingExtraPurchase
	h.db.Preload("Extra").Where("listing_id = ?", listing.ID).Find(&extras)

	response.OK(c, gin.H{"listing": listing, "extras": extras})
}

func (h *Handler) UpdateListing(c *gin.Context) {
	var listing listings.Listing
	if err := h.db.First(&listing, "id = ?", c.Param("id")).Error; err != nil {
		response.NotFound(c, "listing")
		return
	}
	var req struct {
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Status      string  `json:"status"`
		IsFeatured  *bool   `json:"is_featured"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	updates := map[string]any{}
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Price > 0 {
		updates["price"] = req.Price
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.IsFeatured != nil {
		updates["is_featured"] = *req.IsFeatured
	}
	if len(updates) == 0 {
		response.BadRequest(c, "no fields to update")
		return
	}
	h.db.Model(&listing).Updates(updates)
	h.logAction(c, "update_listing", "listing", c.Param("id"), updates)
	response.OK(c, gin.H{"message": "Listing updated."})
}

func (h *Handler) FeatureListing(c *gin.Context) {
	result := h.db.Model(&listings.Listing{}).Where("id = ?", c.Param("id")).
		Update("is_featured", true)
	if result.RowsAffected == 0 {
		response.NotFound(c, "listing")
		return
	}
	h.logAction(c, "feature_listing", "listing", c.Param("id"), nil)
	response.OK(c, gin.H{"message": "Listing featured."})
}

func (h *Handler) ExtendListing(c *gin.Context) {
	var req struct {
		Days int `json:"days" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	var listing listings.Listing
	if err := h.db.First(&listing, "id = ?", c.Param("id")).Error; err != nil {
		response.NotFound(c, "listing")
		return
	}
	newExpiry := time.Now().AddDate(0, 0, req.Days)
	if listing.ExpiresAt != nil && listing.ExpiresAt.After(time.Now()) {
		newExpiry = listing.ExpiresAt.AddDate(0, 0, req.Days)
	}
	h.db.Model(&listing).Update("expires_at", newExpiry)
	h.logAction(c, "extend_listing", "listing", c.Param("id"), gin.H{"days": req.Days})
	response.OK(c, gin.H{"message": fmt.Sprintf("Listing extended by %d days.", req.Days), "new_expires_at": newExpiry})
}

func (h *Handler) AddListingExtra(c *gin.Context) {
	listingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid listing id")
		return
	}
	var req struct {
		ExtraID int `json:"extra_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	var extra ListingExtra
	if err := h.db.First(&extra, req.ExtraID).Error; err != nil {
		response.NotFound(c, "extra")
		return
	}
	purchase := ListingExtraPurchase{
		ListingID:   listingID,
		ExtraID:     req.ExtraID,
		PurchasedAt: time.Now(),
	}
	if extra.DurationDays != nil && *extra.DurationDays > 0 {
		exp := time.Now().AddDate(0, 0, *extra.DurationDays)
		purchase.ExpiresAt = &exp
	}
	h.db.Create(&purchase)
	h.logAction(c, "add_listing_extra", "listing", c.Param("id"), gin.H{"extra_id": req.ExtraID})
	response.Created(c, purchase)
}

func (h *Handler) RemoveListingExtra(c *gin.Context) {
	extraPurchaseID, _ := strconv.Atoi(c.Param("extraId"))
	result := h.db.Delete(&ListingExtraPurchase{}, extraPurchaseID)
	if result.RowsAffected == 0 {
		response.NotFound(c, "extra purchase")
		return
	}
	h.logAction(c, "remove_listing_extra", "listing", c.Param("id"), gin.H{"purchase_id": extraPurchaseID})
	response.OK(c, gin.H{"message": "Extra removed."})
}

func (h *Handler) BulkApproveListings(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result := h.db.Model(&listings.Listing{}).Where("id IN ? AND status = ?", req.IDs, "pending").
		Update("status", "active")
	h.logAction(c, "bulk_approve", "listing", "", gin.H{"count": result.RowsAffected})
	response.OK(c, gin.H{"approved": result.RowsAffected})
}

func (h *Handler) BulkRejectListings(c *gin.Context) {
	var req struct {
		IDs    []string `json:"ids" binding:"required"`
		Reason string   `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result := h.db.Model(&listings.Listing{}).Where("id IN ?", req.IDs).
		Update("status", "rejected")
	h.logAction(c, "bulk_reject", "listing", "", gin.H{"count": result.RowsAffected, "reason": req.Reason})
	response.OK(c, gin.H{"rejected": result.RowsAffected})
}

func (h *Handler) BulkDeleteListings(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result := h.db.Where("id IN ?", req.IDs).Delete(&listings.Listing{})
	h.logAction(c, "bulk_delete", "listing", "", gin.H{"count": result.RowsAffected})
	response.OK(c, gin.H{"deleted": result.RowsAffected})
}

// ════════════════════════════════════════════════════════════════════════════
// SECTION 4 — Auctions admin
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) ListAuctions(c *gin.Context) {
	page, perPage := paginationParams(c)
	q := h.db.Model(&auctions.Auction{})
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if aType := c.Query("type"); aType != "" {
		q = q.Where("type = ?", aType)
	}
	var total int64
	q.Count(&total)
	var list []auctions.Auction
	q.Offset((page - 1) * perPage).Limit(perPage).Order("created_at DESC").Find(&list)
	response.OKMeta(c, list, response.Meta{Total: total, Page: page, PerPage: perPage, Pages: (total + int64(perPage) - 1) / int64(perPage)})
}

func (h *Handler) ListPendingAuctions(c *gin.Context) {
	page, perPage := paginationParams(c)
	q := h.db.Model(&auctions.Auction{}).Where("status = ?", "scheduled")
	var total int64
	q.Count(&total)
	var list []auctions.Auction
	q.Offset((page - 1) * perPage).Limit(perPage).Order("created_at ASC").Find(&list)
	response.OKMeta(c, list, response.Meta{Total: total, Page: page, PerPage: perPage, Pages: (total + int64(perPage) - 1) / int64(perPage)})
}

func (h *Handler) GetAuction(c *gin.Context) {
	var auction auctions.Auction
	if err := h.db.Preload("Bids").First(&auction, "id = ?", c.Param("id")).Error; err != nil {
		response.NotFound(c, "auction")
		return
	}
	response.OK(c, auction)
}

func (h *Handler) ApproveAuction(c *gin.Context) {
	result := h.db.Model(&auctions.Auction{}).Where("id = ? AND status = ?", c.Param("id"), "scheduled").
		Update("status", "active")
	if result.RowsAffected == 0 {
		response.BadRequest(c, "auction not found or not pending")
		return
	}
	h.logAction(c, "approve_auction", "auction", c.Param("id"), nil)
	response.OK(c, gin.H{"message": "Auction approved."})
}

func (h *Handler) RejectAuction(c *gin.Context) {
	result := h.db.Model(&auctions.Auction{}).Where("id = ?", c.Param("id")).
		Update("status", "cancelled")
	if result.RowsAffected == 0 {
		response.NotFound(c, "auction")
		return
	}
	h.logAction(c, "reject_auction", "auction", c.Param("id"), nil)
	response.OK(c, gin.H{"message": "Auction rejected."})
}

func (h *Handler) CancelAuction(c *gin.Context) {
	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req) //nolint:errcheck
	result := h.db.Model(&auctions.Auction{}).Where("id = ?", c.Param("id")).
		Update("status", "cancelled")
	if result.RowsAffected == 0 {
		response.NotFound(c, "auction")
		return
	}
	h.logAction(c, "cancel_auction", "auction", c.Param("id"), gin.H{"reason": req.Reason})
	response.OK(c, gin.H{"message": "Auction cancelled."})
}

func (h *Handler) ExtendAuction(c *gin.Context) {
	var req struct {
		Hours int `json:"hours" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	var auction auctions.Auction
	if err := h.db.First(&auction, "id = ?", c.Param("id")).Error; err != nil {
		response.NotFound(c, "auction")
		return
	}
	newEnd := auction.EndsAt.Add(time.Duration(req.Hours) * time.Hour)
	h.db.Model(&auction).Update("ends_at", newEnd)
	h.logAction(c, "extend_auction", "auction", c.Param("id"), gin.H{"hours": req.Hours})
	response.OK(c, gin.H{"message": fmt.Sprintf("Auction extended by %d hours.", req.Hours), "new_ends_at": newEnd})
}

func (h *Handler) GetAuctionBids(c *gin.Context) {
	var bids []auctions.Bid
	h.db.Where("auction_id = ?", c.Param("id")).Order("amount DESC").Find(&bids)
	response.OK(c, bids)
}

func (h *Handler) DeleteAuctionBid(c *gin.Context) {
	result := h.db.Where("id = ? AND auction_id = ?", c.Param("bidId"), c.Param("id")).
		Delete(&auctions.Bid{})
	if result.RowsAffected == 0 {
		response.NotFound(c, "bid")
		return
	}
	h.logAction(c, "delete_bid", "bid", c.Param("bidId"), gin.H{"auction_id": c.Param("id")})
	response.OK(c, gin.H{"message": "Bid deleted."})
}

// ════════════════════════════════════════════════════════════════════════════
// SECTION 5 — Categories: reorder + fields CRUD
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) ReorderCategory(c *gin.Context) {
	var req struct {
		SortOrder int `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result := h.db.Model(&listings.Category{}).Where("id = ?", c.Param("id")).
		Update("sort_order", req.SortOrder)
	if result.RowsAffected == 0 {
		response.NotFound(c, "category")
		return
	}
	response.OK(c, gin.H{"message": "Category reordered."})
}

// Category fields use the category_fields table from migration 022
type CategoryField struct {
	ID          int    `gorm:"primaryKey;autoIncrement" json:"id"`
	CategoryID  string `gorm:"not null;index" json:"category_id"`
	Name        string `gorm:"size:100;not null" json:"name"`
	Label       string `gorm:"size:100;not null" json:"label"`
	LabelEn     string `gorm:"size:100" json:"label_en,omitempty"`
	FieldType   string `gorm:"size:20;not null" json:"field_type"`
	Options     string `gorm:"type:jsonb;default:'[]'" json:"options"`
	IsRequired  bool   `gorm:"default:false" json:"is_required"`
	Placeholder string `gorm:"size:200" json:"placeholder,omitempty"`
	Unit        string `gorm:"size:20" json:"unit,omitempty"`
	SortOrder   int    `gorm:"default:0" json:"sort_order"`
	IsActive    bool   `gorm:"default:true" json:"is_active"`
}

func (CategoryField) TableName() string { return "category_fields" }

func (h *Handler) ListCategoryFields(c *gin.Context) {
	var fields []CategoryField
	h.db.Where("category_id = ?", c.Param("id")).Order("sort_order ASC").Find(&fields)
	response.OK(c, fields)
}

func (h *Handler) CreateCategoryField(c *gin.Context) {
	var f CategoryField
	if err := c.ShouldBindJSON(&f); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	f.CategoryID = c.Param("id")
	h.db.Create(&f)
	h.logAction(c, "create_category_field", "category_field", fmt.Sprint(f.ID), gin.H{"category_id": f.CategoryID, "name": f.Name})
	response.Created(c, f)
}

func (h *Handler) UpdateCategoryField(c *gin.Context) {
	fieldID, _ := strconv.Atoi(c.Param("fieldId"))
	var f CategoryField
	if err := h.db.First(&f, fieldID).Error; err != nil {
		response.NotFound(c, "category_field")
		return
	}
	if err := c.ShouldBindJSON(&f); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	f.ID = fieldID
	f.CategoryID = c.Param("id")
	h.db.Save(&f)
	response.OK(c, f)
}

func (h *Handler) DeleteCategoryField(c *gin.Context) {
	fieldID, _ := strconv.Atoi(c.Param("fieldId"))
	result := h.db.Delete(&CategoryField{}, fieldID)
	if result.RowsAffected == 0 {
		response.NotFound(c, "category_field")
		return
	}
	h.logAction(c, "delete_category_field", "category_field", c.Param("fieldId"), nil)
	response.OK(c, gin.H{"message": "Field deleted."})
}

// ════════════════════════════════════════════════════════════════════════════
// SECTION 6 — Pricing: plans CRUD, gateways, invoices, discount codes
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) CreatePlan(c *gin.Context) {
	var plan Plan
	if err := c.ShouldBindJSON(&plan); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.db.Create(&plan).Error; err != nil {
		response.BadRequest(c, "plan name already exists or invalid input")
		return
	}
	h.logAction(c, "create_plan", "plan", plan.ID.String(), gin.H{"name": plan.Name})
	response.Created(c, plan)
}

func (h *Handler) DeletePlan(c *gin.Context) {
	result := h.db.Where("id = ?", c.Param("id")).Delete(&Plan{})
	if result.RowsAffected == 0 {
		response.NotFound(c, "plan")
		return
	}
	h.logAction(c, "delete_plan", "plan", c.Param("id"), nil)
	response.OK(c, gin.H{"message": "Plan deleted."})
}

// Payment Gateways
func (h *Handler) ListPaymentGateways(c *gin.Context) {
	var gws []PaymentGateway
	h.db.Order("sort_order ASC").Find(&gws)
	response.OK(c, gws)
}

func (h *Handler) UpdatePaymentGateway(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var gw PaymentGateway
	if err := h.db.First(&gw, id).Error; err != nil {
		response.NotFound(c, "payment_gateway")
		return
	}
	if err := c.ShouldBindJSON(&gw); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	gw.ID = id
	h.db.Save(&gw)
	h.logAction(c, "update_gateway", "payment_gateway", c.Param("id"), gin.H{"name": gw.Name})
	response.OK(c, gw)
}

// Invoices
func (h *Handler) ListInvoices(c *gin.Context) {
	page, perPage := paginationParams(c)
	q := h.db.Model(&Invoice{})
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	var list []Invoice
	q.Offset((page - 1) * perPage).Limit(perPage).Order("created_at DESC").Find(&list)
	response.OKMeta(c, list, response.Meta{Total: total, Page: page, PerPage: perPage, Pages: (total + int64(perPage) - 1) / int64(perPage)})
}

func (h *Handler) GetInvoice(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var inv Invoice
	if err := h.db.First(&inv, id).Error; err != nil {
		response.NotFound(c, "invoice")
		return
	}
	response.OK(c, inv)
}

// Discount Codes
func (h *Handler) ListDiscountCodes(c *gin.Context) {
	var codes []DiscountCode
	h.db.Order("created_at DESC").Find(&codes)
	response.OK(c, codes)
}

func (h *Handler) CreateDiscountCode(c *gin.Context) {
	var code DiscountCode
	if err := c.ShouldBindJSON(&code); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.db.Create(&code).Error; err != nil {
		response.BadRequest(c, "code already exists")
		return
	}
	h.logAction(c, "create_discount_code", "discount_code", fmt.Sprint(code.ID), gin.H{"code": code.Code})
	response.Created(c, code)
}

func (h *Handler) UpdateDiscountCode(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var code DiscountCode
	if err := h.db.First(&code, id).Error; err != nil {
		response.NotFound(c, "discount_code")
		return
	}
	if err := c.ShouldBindJSON(&code); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	code.ID = id
	h.db.Save(&code)
	response.OK(c, code)
}

func (h *Handler) DeleteDiscountCode(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	result := h.db.Delete(&DiscountCode{}, id)
	if result.RowsAffected == 0 {
		response.NotFound(c, "discount_code")
		return
	}
	h.logAction(c, "delete_discount_code", "discount_code", c.Param("id"), nil)
	response.OK(c, gin.H{"message": "Discount code deleted."})
}

// Listing Extras management
func (h *Handler) ListListingExtras(c *gin.Context) {
	var extras []ListingExtra
	h.db.Find(&extras)
	response.OK(c, extras)
}

func (h *Handler) CreateListingExtra(c *gin.Context) {
	var e ListingExtra
	if err := c.ShouldBindJSON(&e); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.db.Create(&e)
	response.Created(c, e)
}

func (h *Handler) UpdateListingExtra(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var e ListingExtra
	if err := h.db.First(&e, id).Error; err != nil {
		response.NotFound(c, "listing_extra")
		return
	}
	if err := c.ShouldBindJSON(&e); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	e.ID = id
	h.db.Save(&e)
	response.OK(c, e)
}

func (h *Handler) DeleteListingExtra(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	result := h.db.Delete(&ListingExtra{}, id)
	if result.RowsAffected == 0 {
		response.NotFound(c, "listing_extra")
		return
	}
	response.OK(c, gin.H{"message": "Listing extra deleted."})
}

// ════════════════════════════════════════════════════════════════════════════
// SECTION 7 — Content: email templates, static pages, announcements
// ════════════════════════════════════════════════════════════════════════════

// Email Templates
func (h *Handler) ListEmailTemplates(c *gin.Context) {
	var tpls []EmailTemplate
	h.db.Find(&tpls)
	response.OK(c, tpls)
}

func (h *Handler) GetEmailTemplate(c *gin.Context) {
	slug := c.Param("slug")
	var tpl EmailTemplate
	if err := h.db.Where("slug = ?", slug).First(&tpl).Error; err != nil {
		response.NotFound(c, "email_template")
		return
	}
	response.OK(c, tpl)
}

func (h *Handler) UpdateEmailTemplate(c *gin.Context) {
	slug := c.Param("slug")
	var tpl EmailTemplate
	if err := h.db.Where("slug = ?", slug).First(&tpl).Error; err != nil {
		response.NotFound(c, "email_template")
		return
	}
	var req struct {
		Subject  string `json:"subject"`
		BodyHTML string `json:"body_html"`
		BodyText string `json:"body_text"`
		IsActive *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	updates := map[string]any{"updated_at": time.Now()}
	if req.Subject != "" {
		updates["subject"] = req.Subject
	}
	if req.BodyHTML != "" {
		updates["body_html"] = req.BodyHTML
	}
	if req.BodyText != "" {
		updates["body_text"] = req.BodyText
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if adminID := c.GetString("user_id"); adminID != "" {
		updates["updated_by"] = adminID
	}
	h.db.Model(&tpl).Updates(updates)
	h.logAction(c, "update_email_template", "email_template", slug, nil)
	h.db.Where("slug = ?", slug).First(&tpl) // reload
	response.OK(c, tpl)
}

func (h *Handler) PreviewEmailTemplate(c *gin.Context) {
	slug := c.Param("slug")
	var tpl EmailTemplate
	if err := h.db.Where("slug = ?", slug).First(&tpl).Error; err != nil {
		response.NotFound(c, "email_template")
		return
	}

	// Build sample data from template variables
	var vars []string
	_ = json.Unmarshal([]byte(tpl.Variables), &vars)
	sampleData := make(map[string]string)
	for _, v := range vars {
		sampleData[v] = "sample_" + v
	}

	// Simple template replacement: {{variable}} → sample_value
	rendered := tpl.BodyHTML
	for k, v := range sampleData {
		rendered = strings.ReplaceAll(rendered, "{{"+k+"}}", v)
	}
	renderedSubject := tpl.Subject
	for k, v := range sampleData {
		renderedSubject = strings.ReplaceAll(renderedSubject, "{{"+k+"}}", v)
	}

	response.OK(c, gin.H{
		"subject":   renderedSubject,
		"body_html": rendered,
		"variables": sampleData,
	})
}

func (h *Handler) TestEmailTemplate(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	// In a real implementation, send the email. For now, log it.
	h.logAction(c, "test_email_template", "email_template", c.Param("slug"), gin.H{"to": req.Email})
	response.OK(c, gin.H{"message": fmt.Sprintf("Test email queued to %s", req.Email)})
}

// Static Pages
func (h *Handler) ListStaticPages(c *gin.Context) {
	var pages []StaticPage
	h.db.Order("created_at ASC").Find(&pages)
	response.OK(c, pages)
}

func (h *Handler) GetStaticPage(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p StaticPage
	if err := h.db.First(&p, id).Error; err != nil {
		response.NotFound(c, "static_page")
		return
	}
	response.OK(c, p)
}

func (h *Handler) CreateStaticPage(c *gin.Context) {
	var p StaticPage
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.db.Create(&p).Error; err != nil {
		response.BadRequest(c, "slug already exists")
		return
	}
	h.logAction(c, "create_static_page", "static_page", fmt.Sprint(p.ID), gin.H{"slug": p.Slug})
	response.Created(c, p)
}

func (h *Handler) UpdateStaticPage(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p StaticPage
	if err := h.db.First(&p, id).Error; err != nil {
		response.NotFound(c, "static_page")
		return
	}
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	p.ID = id
	p.UpdatedAt = time.Now()
	h.db.Save(&p)
	response.OK(c, p)
}

func (h *Handler) DeleteStaticPage(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	result := h.db.Delete(&StaticPage{}, id)
	if result.RowsAffected == 0 {
		response.NotFound(c, "static_page")
		return
	}
	h.logAction(c, "delete_static_page", "static_page", c.Param("id"), nil)
	response.OK(c, gin.H{"message": "Page deleted."})
}

// Announcements
func (h *Handler) ListAnnouncements(c *gin.Context) {
	var anns []Announcement
	h.db.Order("created_at DESC").Find(&anns)
	response.OK(c, anns)
}

func (h *Handler) CreateAnnouncement(c *gin.Context) {
	var a Announcement
	if err := c.ShouldBindJSON(&a); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.db.Create(&a)
	h.logAction(c, "create_announcement", "announcement", fmt.Sprint(a.ID), gin.H{"title": a.Title})
	response.Created(c, a)
}

func (h *Handler) UpdateAnnouncement(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var a Announcement
	if err := h.db.First(&a, id).Error; err != nil {
		response.NotFound(c, "announcement")
		return
	}
	if err := c.ShouldBindJSON(&a); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	a.ID = id
	h.db.Save(&a)
	response.OK(c, a)
}

func (h *Handler) DeleteAnnouncement(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	result := h.db.Delete(&Announcement{}, id)
	if result.RowsAffected == 0 {
		response.NotFound(c, "announcement")
		return
	}
	h.logAction(c, "delete_announcement", "announcement", c.Param("id"), nil)
	response.OK(c, gin.H{"message": "Announcement deleted."})
}

// ════════════════════════════════════════════════════════════════════════════
// SECTION 8 — Geography
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) ListGeoRegions(c *gin.Context) {
	var regions []GeoRegion
	h.db.Where("parent_id IS NULL").Preload("Children").
		Order("sort_order ASC, name ASC").Find(&regions)
	response.OK(c, regions)
}

func (h *Handler) GetGeoChildren(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var children []GeoRegion
	h.db.Where("parent_id = ?", id).Order("sort_order ASC, name ASC").Find(&children)
	response.OK(c, children)
}

func (h *Handler) CreateGeoRegion(c *gin.Context) {
	var r GeoRegion
	if err := c.ShouldBindJSON(&r); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.db.Create(&r)
	h.logAction(c, "create_geo_region", "geo_region", fmt.Sprint(r.ID), gin.H{"name": r.Name})
	response.Created(c, r)
}

func (h *Handler) UpdateGeoRegion(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var r GeoRegion
	if err := h.db.First(&r, id).Error; err != nil {
		response.NotFound(c, "geo_region")
		return
	}
	if err := c.ShouldBindJSON(&r); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	r.ID = id
	h.db.Save(&r)
	response.OK(c, r)
}

func (h *Handler) DeleteGeoRegion(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	result := h.db.Delete(&GeoRegion{}, id)
	if result.RowsAffected == 0 {
		response.NotFound(c, "geo_region")
		return
	}
	h.logAction(c, "delete_geo_region", "geo_region", c.Param("id"), nil)
	response.OK(c, gin.H{"message": "Region deleted."})
}

// User Custom Fields CRUD
func (h *Handler) ListUserCustomFields(c *gin.Context) {
	var fields []UserCustomField
	h.db.Where("is_active = ?", true).Order("sort_order ASC").Find(&fields)
	response.OK(c, fields)
}

func (h *Handler) CreateUserCustomField(c *gin.Context) {
	var f UserCustomField
	if err := c.ShouldBindJSON(&f); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.db.Create(&f)
	response.Created(c, f)
}

func (h *Handler) UpdateUserCustomField(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var f UserCustomField
	if err := h.db.First(&f, id).Error; err != nil {
		response.NotFound(c, "user_custom_field")
		return
	}
	if err := c.ShouldBindJSON(&f); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	f.ID = id
	h.db.Save(&f)
	response.OK(c, f)
}

func (h *Handler) DeleteUserCustomField(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	result := h.db.Delete(&UserCustomField{}, id)
	if result.RowsAffected == 0 {
		response.NotFound(c, "user_custom_field")
		return
	}
	response.OK(c, gin.H{"message": "Custom field deleted."})
}

// ════════════════════════════════════════════════════════════════════════════
// SECTION 6 — Public: listing extras for listing pages
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) GetPublicListingExtras(c *gin.Context) {
	var extras []ListingExtra
	h.db.Where("is_active = ?", true).Order("price ASC").Find(&extras)
	c.JSON(http.StatusOK, gin.H{"data": extras})
}

// ════════════════════════════════════════════════════════════════════════════
// SECTION 7 — Storefronts Admin
// ════════════════════════════════════════════════════════════════════════════

type storefrontRow struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	LogoURL       string    `json:"logo_url"`
	BannerURL     string    `json:"banner_url"`
	Views         int       `json:"views"`
	IsActive      bool      `json:"is_active"`
	IsFeatured    bool      `json:"is_featured"`
	ListingsCount int64     `json:"listings_count"`
	OwnerName     string    `json:"owner_name"`
	CreatedAt     time.Time `json:"created_at"`
}

func (h *Handler) ListStorefronts(c *gin.Context) {
	var rows []struct {
		ID          string    `json:"id"`
		UserID      string    `json:"user_id"`
		Slug        string    `json:"slug"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		LogoURL     string    `json:"logo_url"`
		BannerURL   string    `json:"banner_url"`
		Views       int       `json:"views"`
		IsActive    bool      `json:"is_active"`
		IsFeatured  bool      `json:"is_featured"`
		CreatedAt   time.Time `json:"created_at"`
	}
	h.db.Table("storefronts").Where("deleted_at IS NULL").Order("created_at DESC").Find(&rows)

	result := make([]storefrontRow, len(rows))
	for i, r := range rows {
		result[i] = storefrontRow{
			ID: r.ID, UserID: r.UserID, Slug: r.Slug, Name: r.Name,
			Description: r.Description, LogoURL: r.LogoURL, BannerURL: r.BannerURL,
			Views: r.Views, IsActive: r.IsActive, IsFeatured: r.IsFeatured, CreatedAt: r.CreatedAt,
		}
		// count listings
		h.db.Table("listings").Where("user_id = ? AND status = 'active'", r.UserID).Count(&result[i].ListingsCount)
		// owner name
		var u users.User
		if h.db.Select("name").Where("id = ?", r.UserID).First(&u).Error == nil {
			result[i].OwnerName = u.Name
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) GetStorefront(c *gin.Context) {
	id := c.Param("id")
	var store map[string]interface{}
	if err := h.db.Table("storefronts").Where("id = ? AND deleted_at IS NULL", id).First(&store).Error; err != nil {
		response.NotFound(c, "storefront")
		return
	}
	response.OK(c, store)
}

func (h *Handler) ApproveStorefront(c *gin.Context) {
	id := c.Param("id")
	result := h.db.Table("storefronts").Where("id = ?", id).Update("is_active", true)
	if result.RowsAffected == 0 {
		response.NotFound(c, "storefront")
		return
	}
	h.logAction(c, "storefront.approved", "storefront", id, nil)
	response.OK(c, gin.H{"message": "Storefront approved."})
}

func (h *Handler) SuspendStorefront(c *gin.Context) {
	id := c.Param("id")
	result := h.db.Table("storefronts").Where("id = ?", id).Update("is_active", false)
	if result.RowsAffected == 0 {
		response.NotFound(c, "storefront")
		return
	}
	h.logAction(c, "storefront.suspended", "storefront", id, nil)
	response.OK(c, gin.H{"message": "Storefront suspended."})
}

func (h *Handler) FeatureStorefront(c *gin.Context) {
	id := c.Param("id")
	// toggle is_featured
	h.db.Table("storefronts").Where("id = ?", id).Update("is_featured", h.db.Raw("NOT is_featured"))
	h.logAction(c, "storefront.feature_toggled", "storefront", id, nil)
	response.OK(c, gin.H{"message": "Featured status toggled."})
}

func (h *Handler) DeleteStorefront(c *gin.Context) {
	id := c.Param("id")
	now := time.Now()
	result := h.db.Table("storefronts").Where("id = ?", id).Update("deleted_at", now)
	if result.RowsAffected == 0 {
		response.NotFound(c, "storefront")
		return
	}
	h.logAction(c, "storefront.deleted", "storefront", id, nil)
	response.OK(c, gin.H{"message": "Storefront deleted."})
}
