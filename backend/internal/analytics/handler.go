package analytics

import (
	"fmt"
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

type sellerSummary struct {
	TotalRevenue   float64 `json:"total_revenue"`
	TotalOrders    int64   `json:"total_orders"`
	ActiveListings int64   `json:"active_listings"`
	TotalViews     int64   `json:"total_views"`
	AvgRating      float64 `json:"avg_rating"`
}

type revenuePoint struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
}

type listingBreakdown struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Views          int64   `json:"views"`
	Favorites      int64   `json:"favorites"`
	Orders         int64   `json:"orders"`
	ConversionRate float64 `json:"conversion_rate"`
}

// SellerSummary returns seller metrics for the authenticated user.
func (h *Handler) SellerSummary(c *gin.Context) {
	sellerID := c.GetString("user_id")
	if sellerID == "" {
		response.Unauthorized(c)
		return
	}

	out := sellerSummary{}

	h.db.Table("orders").
		Where("seller_id = ? AND status IN ?", sellerID, []string{"confirmed", "processing", "shipped", "delivered", "completed"}).
		Select("COALESCE(SUM(total), 0)").
		Scan(&out.TotalRevenue)

	h.db.Table("orders").
		Where("seller_id = ? AND status IN ?", sellerID, []string{"confirmed", "processing", "shipped", "delivered", "completed"}).
		Count(&out.TotalOrders)

	h.db.Table("listings").
		Where("user_id = ? AND status = ?", sellerID, "active").
		Count(&out.ActiveListings)

	h.db.Table("listings").
		Where("user_id = ?", sellerID).
		Select("COALESCE(SUM(view_count), 0)").
		Scan(&out.TotalViews)

	h.db.Table("reviews").
		Where("reviewed_id = ?", sellerID).
		Select("COALESCE(AVG(rating), 0)").
		Scan(&out.AvgRating)

	response.OK(c, out)
}

// SellerRevenue returns seller revenue time-series for the requested period.
func (h *Handler) SellerRevenue(c *gin.Context) {
	sellerID := c.GetString("user_id")
	if sellerID == "" {
		response.Unauthorized(c)
		return
	}

	period := c.DefaultQuery("period", "30d")
	days, err := periodToDays(period)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	since := time.Now().AddDate(0, 0, -days)
	var series []revenuePoint
	if err := h.db.Table("orders").
		Select("TO_CHAR(DATE(created_at), 'YYYY-MM-DD') AS date, COALESCE(SUM(total), 0) AS amount").
		Where("seller_id = ? AND status IN ? AND created_at >= ?", sellerID, []string{"completed", "delivered"}, since).
		Group("DATE(created_at)").
		Order("DATE(created_at) ASC").
		Scan(&series).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, gin.H{
		"period": period,
		"series": series,
	})
}

// SellerListings returns seller listing analytics breakdown.
func (h *Handler) SellerListings(c *gin.Context) {
	sellerID := c.GetString("user_id")
	if sellerID == "" {
		response.Unauthorized(c)
		return
	}

	var rows []listingBreakdown
	err := h.db.Table("listings l").
		Select(`
			l.id::text AS id,
			l.title AS title,
			l.view_count AS views,
			l.favorite_count AS favorites,
			COALESCE(COUNT(oi.order_id), 0) AS orders
		`).
		Joins("LEFT JOIN order_items oi ON oi.listing_id = l.id").
		Joins("LEFT JOIN orders o ON o.id = oi.order_id AND o.seller_id::text = ? AND o.status IN ?", sellerID, []string{"confirmed", "processing", "shipped", "delivered", "completed"}).
		Where("l.user_id::text = ?", sellerID).
		Group("l.id, l.title, l.view_count, l.favorite_count, l.created_at").
		Order("orders DESC, l.created_at DESC").
		Scan(&rows).Error
	if err != nil {
		response.InternalError(c, err)
		return
	}

	for i := range rows {
		if rows[i].Views <= 0 {
			rows[i].ConversionRate = 0
			continue
		}
		rows[i].ConversionRate = (float64(rows[i].Orders) / float64(rows[i].Views)) * 100
	}

	response.OK(c, rows)
}

func periodToDays(period string) (int, error) {
	switch period {
	case "7d":
		return 7, nil
	case "30d":
		return 30, nil
	case "90d":
		return 90, nil
	case "1y":
		return 365, nil
	default:
		return 0, fmt.Errorf("invalid period: use one of 7d, 30d, 90d, 1y")
	}
}

// StorefrontAnalytics returns storefront-specific metrics for the authenticated seller.
func (h *Handler) StorefrontAnalytics(c *gin.Context) {
	sellerID := c.GetString("user_id")
	if sellerID == "" {
		response.Unauthorized(c)
		return
	}

	period := c.DefaultQuery("period", "30d")
	days, err := periodToDays(period)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	since := time.Now().AddDate(0, 0, -days)

	// Get storefront info
	var storefront struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Slug          string `json:"slug"`
		ViewCount     int64  `json:"view_count"`
		FollowerCount int64  `json:"follower_count"`
	}

	err = h.db.Table("storefronts").
		Select("id, name, slug, view_count, (SELECT COUNT(*) FROM storefront_followers WHERE storefront_id = storefronts.id) as follower_count").
		Where("user_id = ?", sellerID).
		Scan(&storefront).Error
	if err != nil {
		response.InternalError(c, err)
		return
	}

	// Get storefront views over period
	var viewsSeries []struct {
		Date  string `json:"date"`
		Views int64  `json:"views"`
	}

	// Note: This assumes a storefront_views table exists for tracking daily views
	// If not, we return the total view_count as a single data point
	h.db.Table("storefronts").
		Select("DATE(updated_at) as date, view_count as views").
		Where("user_id = ? AND updated_at >= ?", sellerID, since).
		Order("date ASC").
		Scan(&viewsSeries)

	// Get top performing listings
	var topListings []listingBreakdown
	h.db.Table("listings l").
		Select(`
			l.id::text AS id,
			l.title AS title,
			l.view_count AS views,
			l.favorite_count AS favorites,
			COALESCE(COUNT(oi.order_id), 0) AS orders
		`).
		Joins("LEFT JOIN order_items oi ON oi.listing_id = l.id").
		Joins("LEFT JOIN orders o ON o.id = oi.order_id AND o.seller_id::text = ? AND o.status IN ?", sellerID, []string{"confirmed", "processing", "shipped", "delivered", "completed"}).
		Where("l.user_id::text = ?", sellerID).
		Group("l.id, l.title, l.view_count, l.favorite_count").
		Order("views DESC").
		Limit(5).
		Scan(&topListings)

	for i := range topListings {
		if topListings[i].Views <= 0 {
			topListings[i].ConversionRate = 0
			continue
		}
		topListings[i].ConversionRate = (float64(topListings[i].Orders) / float64(topListings[i].Views)) * 100
	}

	// Calculate conversion rate (orders / views)
	var totalOrders int64
	var totalViews int64
	h.db.Table("orders").
		Where("seller_id = ? AND status IN ?", sellerID, []string{"confirmed", "processing", "shipped", "delivered", "completed"}).
		Count(&totalOrders)
	h.db.Table("listings").
		Where("user_id = ?", sellerID).
		Select("COALESCE(SUM(view_count), 0)").
		Scan(&totalViews)

	conversionRate := 0.0
	if totalViews > 0 {
		conversionRate = (float64(totalOrders) / float64(totalViews)) * 100
	}

	response.OK(c, gin.H{
		"storefront":      storefront,
		"period":          period,
		"views_series":    viewsSeries,
		"top_listings":    topListings,
		"conversion_rate": conversionRate,
		"total_orders":    totalOrders,
		"total_views":     totalViews,
	})
}

// PlatformMetrics returns platform-wide metrics for admin/founder dashboard
func (h *Handler) PlatformMetrics(c *gin.Context) {
	// Get total users
	var totalUsers int64
	h.db.Table("users").Count(&totalUsers)

	// Get new users in last 7 days
	var newUsers7d int64
	h.db.Table("users").
		Where("created_at > ?", time.Now().AddDate(0, 0, -7)).
		Count(&newUsers7d)

	// Get new users in last 30 days
	var newUsers30d int64
	h.db.Table("users").
		Where("created_at > ?", time.Now().AddDate(0, 0, -30)).
		Count(&newUsers30d)

	// Get total listings
	var totalListings int64
	h.db.Table("listings").Count(&totalListings)

	// Get active listings
	var activeListings int64
	h.db.Table("listings").
		Where("status = ?", "active").
		Count(&activeListings)

	// Get total orders
	var totalOrders int64
	h.db.Table("orders").Count(&totalOrders)

	// Get GMV (Gross Merchandise Value) in last 30 days
	var gmv30d float64
	h.db.Table("orders").
		Where("created_at > ? AND status NOT IN ?", time.Now().AddDate(0, 0, -30), []string{"cancelled", "refunded"}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&gmv30d)

	// Get total revenue (platform fees collected)
	var totalRevenue float64
	h.db.Table("orders").
		Where("status NOT IN ?", []string{"cancelled", "refunded"}).
		Select("COALESCE(SUM(platform_fee), 0)").
		Scan(&totalRevenue)

	// Get open disputes
	var openDisputes int64
	h.db.Table("disputes").
		Where("status = ?", "open").
		Count(&openDisputes)

	// Get resolved disputes
	var resolvedDisputes int64
	h.db.Table("disputes").
		Where("status = ?", "resolved").
		Count(&resolvedDisputes)

	// Get top categories by revenue
	type CategoryRevenue struct {
		CategoryID   string  `json:"category_id"`
		CategoryName string  `json:"category_name"`
		Revenue      float64 `json:"revenue"`
		Listings     int64   `json:"listings"`
	}
	var topCategories []CategoryRevenue
	h.db.Table("orders o").
		Select("c.id as category_id, c.name as category_name, SUM(o.platform_fee) as revenue, COUNT(DISTINCT o.listing_id) as listings").
		Joins("JOIN listings l ON l.id = o.listing_id").
		Joins("JOIN categories c ON c.id = l.category_id").
		Where("o.status NOT IN ?", []string{"cancelled", "refunded"}).
		Group("c.id, c.name").
		Order("revenue DESC").
		Limit(10).
		Find(&topCategories)

	// Get daily signups for last 7 days
	type DailyMetric struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	var dailySignups []DailyMetric
	h.db.Table("users").
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at > ?", time.Now().AddDate(0, 0, -7)).
		Group("DATE(created_at)").
		Order("date ASC").
		Find(&dailySignups)

	// Get daily orders for last 7 days
	var dailyOrders []DailyMetric
	h.db.Table("orders").
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at > ?", time.Now().AddDate(0, 0, -7)).
		Group("DATE(created_at)").
		Order("date ASC").
		Find(&dailyOrders)

	response.OK(c, gin.H{
		"total_users":       totalUsers,
		"new_users_7d":      newUsers7d,
		"new_users_30d":     newUsers30d,
		"total_listings":    totalListings,
		"active_listings":   activeListings,
		"total_orders":      totalOrders,
		"gmv_30d":           gmv30d,
		"total_revenue":     totalRevenue,
		"open_disputes":     openDisputes,
		"resolved_disputes": resolvedDisputes,
		"top_categories":    topCategories,
		"daily_signups":     dailySignups,
		"daily_orders":      dailyOrders,
	})
}

// ════════════════════════════════════════════════════════════════════════════
// Layer 1: Admin Traffic Watch & Overview (eBay Seller Analytics)
// ════════════════════════════════════════════════════════════════════════════

// TrafficWatch returns per-listing traffic metrics for admin analytics.
// GET /admin/analytics/traffic
func (h *Handler) TrafficWatch(c *gin.Context) {
	type ListingTraffic struct {
		ListingID      string  `json:"listing_id"`
		Title          string  `json:"title"`
		PageViews      int64   `json:"page_views"`
		UniqueVisitors int64   `json:"unique_visitors"`
		WatchlistAdds  int64   `json:"watchlist_adds"`
		Inquiries      int64   `json:"message_inquiries"`
		ConversionRate float64 `json:"conversion_rate"`
		AvgTimeOnPage  float64 `json:"avg_time_on_page"`
	}
	var results []ListingTraffic
	h.db.Table("listings l").
		Select(`l.id as listing_id, l.title,
			COALESCE((SELECT COUNT(*) FROM route_metrics WHERE path LIKE '/listings/'||l.id::text), 0) as page_views,
			COALESCE((SELECT COUNT(DISTINCT user_id) FROM favorites WHERE listing_id = l.id), 0) as watchlist_adds,
			0 as unique_visitors, 0 as message_inquiries,
			0 as conversion_rate, 0 as avg_time_on_page`).
		Where("l.status = ?", "active").
		Order("page_views DESC").
		Limit(100).
		Scan(&results)
	response.OK(c, results)
}

// AdminOverview returns platform-wide KPIs for admin analytics overview.
// GET /admin/analytics/overview
func (h *Handler) AdminOverview(c *gin.Context) {
	var dau int64
	h.db.Table("route_metrics").
		Where("created_at > ?", time.Now().AddDate(0, 0, -1)).
		Select("COUNT(DISTINCT user_id)").
		Scan(&dau)

	var newRegs int64
	h.db.Table("users").Where("created_at > ?", time.Now().AddDate(0, 0, -1)).Count(&newRegs)

	var listingsCreated int64
	h.db.Table("listings").Where("created_at > ?", time.Now().AddDate(0, 0, -1)).Count(&listingsCreated)

	var auctionsStarted int64
	h.db.Table("auctions").Where("created_at > ?", time.Now().AddDate(0, 0, -1)).Count(&auctionsStarted)

	var gmv float64
	h.db.Table("orders").
		Where("created_at > ? AND status NOT IN ?", time.Now().AddDate(0, 0, -30), []string{"cancelled", "refunded"}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&gmv)

	var takeRate float64 = 0
	var totalAmount float64
	h.db.Table("orders").
		Where("status NOT IN ?", []string{"cancelled", "refunded"}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalAmount)
	if totalAmount > 0 {
		var totalFees float64
		h.db.Table("orders").
			Where("status NOT IN ?", []string{"cancelled", "refunded"}).
			Select("COALESCE(SUM(platform_fee), 0)").
			Scan(&totalFees)
		takeRate = totalFees / totalAmount * 100
	}

	response.OK(c, gin.H{
		"daily_active_users": dau,
		"new_registrations":  newRegs,
		"listings_created":   listingsCreated,
		"auctions_started":   auctionsStarted,
		"gmv":                gmv,
		"take_rate":          takeRate,
	})
}

// ════════════════════════════════════════════════════════════════════════════
// Sprint 8: Funnel Analytics Handlers
// ════════════════════════════════════════════════════════════════════════════

// GetFunnelDropoffsHandler — GET /api/v1/analytics/funnels
func (h *Handler) GetFunnelDropoffsHandler(c *gin.Context) {
	dropoffs := GetFunnelDropoffs(h.db)
	response.OK(c, dropoffs)
}

// GetUserFunnelProgressHandler — GET /api/v1/analytics/funnels/progress
func (h *Handler) GetUserFunnelProgressHandler(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	progress := GetUserFunnelProgress(h.db, userID)
	response.OK(c, progress)
}
