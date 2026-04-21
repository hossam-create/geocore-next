package analytics

import (
	"fmt"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Metrics Dashboard
// GMV, conversion rate, time to first offer, offers per request, liquidity ratio.
// ════════════════════════════════════════════════════════════════════════════

// PlatformMetrics holds all key marketplace metrics.
type PlatformMetrics struct {
	GMV                decimal.Decimal `json:"gmv"`
	ConversionRate     decimal.Decimal `json:"conversion_rate"`
	TimeToFirstOffer   float64         `json:"time_to_first_offer_hours"`
	OffersPerRequest   decimal.Decimal `json:"offers_per_request"`
	LiquidityRatio     decimal.Decimal `json:"liquidity_ratio"`
	ActiveUsers        int64           `json:"active_users"`
	ActiveListings     int64           `json:"active_listings"`
	ActiveTrips        int64           `json:"active_trips"`
	PendingRequests    int64           `json:"pending_requests"`
	CompletedDeliveries int64          `json:"completed_deliveries"`
	TotalReferrals     int64           `json:"total_referrals"`
	RewardedReferrals  int64           `json:"rewarded_referrals"`
	Period             string          `json:"period"`
}

// GetPlatformMetrics computes all key marketplace metrics.
func GetPlatformMetrics(db *gorm.DB, period string) PlatformMetrics {
	since := getSinceDate(period)
	m := PlatformMetrics{Period: period}

	// GMV: total value of completed orders
	var gmv float64
	db.Table("orders").
		Where("status IN ? AND created_at >= ?", []string{"completed", "delivered"}, since).
		Select("COALESCE(SUM(total),0)").Scan(&gmv)
	m.GMV = decimal.NewFromFloat(gmv)

	// Conversion rate: requests completed / requests created
	var totalRequests int64
	var completedRequests int64
	db.Table("delivery_requests").Where("created_at >= ? AND deleted_at IS NULL", since).Count(&totalRequests)
	db.Table("delivery_requests").Where("status = ? AND created_at >= ? AND deleted_at IS NULL", "delivered", since).Count(&completedRequests)
	if totalRequests > 0 {
		m.ConversionRate = decimal.NewFromInt(completedRequests).Div(decimal.NewFromInt(totalRequests)).Mul(decimal.NewFromInt(100))
	}

	// Time to first offer: average hours between request creation and first offer
	var avgHours float64
	db.Table("delivery_requests dr").
		Joins("JOIN traveler_offers o ON o.delivery_request_id = dr.id AND o.deleted_at IS NULL").
		Where("dr.created_at >= ?", since).
		Select("EXTRACT(EPOCH FROM (MIN(o.created_at) - dr.created_at))/3600").
		Group("dr.id").
		Having("COUNT(o.id) > 0").
		Scan(&avgHours)
	m.TimeToFirstOffer = avgHours

	// Offers per request
	var totalOffers int64
	db.Table("traveler_offers o").
		Joins("JOIN delivery_requests dr ON dr.id = o.delivery_request_id").
		Where("dr.created_at >= ? AND o.deleted_at IS NULL", since).
		Count(&totalOffers)
	if totalRequests > 0 {
		m.OffersPerRequest = decimal.NewFromInt(totalOffers).Div(decimal.NewFromInt(totalRequests))
	}

	// Liquidity ratio: active travelers / pending requests
	var activeTravelers int64
	db.Table("trips").Where("status = ? AND created_at >= ?", "active", since).Count(&activeTravelers)
	db.Table("delivery_requests").Where("status = ? AND created_at >= ? AND deleted_at IS NULL", "pending", since).Count(&m.PendingRequests)
	if m.PendingRequests > 0 {
		m.LiquidityRatio = decimal.NewFromInt(activeTravelers).Div(decimal.NewFromInt(m.PendingRequests))
	}

	// Active users
	db.Table("users").Where("is_active = ? AND updated_at >= ?", true, since).Count(&m.ActiveUsers)

	// Active listings
	db.Table("listings").Where("status = ? AND deleted_at IS NULL", "active").Count(&m.ActiveListings)

	// Active trips
	db.Table("trips").Where("status = ?", "active").Count(&m.ActiveTrips)

	// Completed deliveries
	db.Table("delivery_requests").Where("status = ? AND deleted_at IS NULL", "delivered").Count(&m.CompletedDeliveries)

	// Referral metrics
	db.Table("referrals").Count(&m.TotalReferrals)
	db.Table("referrals").Where("status = ?", "rewarded").Count(&m.RewardedReferrals)

	return m
}

// GetMetricsHandler — GET /admin/metrics
func (h *Handler) GetMetricsHandler(c *gin.Context) {
	period := c.DefaultQuery("period", "30d")
	metrics := GetPlatformMetrics(h.db, period)
	response.OK(c, metrics)
}

// GetMetricsTimeSeries returns GMV over time.
func GetMetricsTimeSeries(db *gorm.DB, period string) []map[string]interface{} {
	since := getSinceDate(period)
	var rows []struct {
		Date   time.Time `json:"date"`
		Amount float64   `json:"amount"`
	}
	db.Table("orders").
		Select("DATE(created_at) as date, COALESCE(SUM(total),0) as amount").
		Where("status IN ? AND created_at >= ?", []string{"completed", "delivered"}, since).
		Group("DATE(created_at)").
		Order("DATE(created_at) ASC").
		Find(&rows)

	result := make([]map[string]interface{}, len(rows))
	for i, r := range rows {
		result[i] = map[string]interface{}{
			"date":   r.Date.Format("2006-01-02"),
			"amount": fmt.Sprintf("%.2f", r.Amount),
		}
	}
	return result
}

// GetMetricsTimeSeriesHandler — GET /admin/metrics/timeseries
func (h *Handler) GetMetricsTimeSeriesHandler(c *gin.Context) {
	period := c.DefaultQuery("period", "30d")
	series := GetMetricsTimeSeries(h.db, period)
	response.OK(c, series)
}

func getSinceDate(period string) time.Time {
	switch period {
	case "7d":
		return time.Now().AddDate(0, 0, -7)
	case "90d":
		return time.Now().AddDate(0, 0, -90)
	case "1y":
		return time.Now().AddDate(-1, 0, 0)
	default: // 30d
		return time.Now().AddDate(0, 0, -30)
	}
}
