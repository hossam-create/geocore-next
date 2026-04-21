package controltower

import (
	"time"

	"gorm.io/gorm"
)

// RevenueDashboard aggregates all monetisation streams.
type RevenueDashboard struct {
	TodayTotal       int64          `json:"today_total_cents"`
	WeekTotal        int64          `json:"week_total_cents"`
	MonthTotal       int64          `json:"month_total_cents"`
	Commissions      RevenueStream  `json:"commissions"`
	Boosts           RevenueStream  `json:"boosts"`
	EntryFees        RevenueStream  `json:"entry_fees"`
	CreatorPayouts   RevenueStream  `json:"creator_payouts"`
	VIPSubscriptions RevenueStream  `json:"vip_subscriptions"`
	ExchangeFees     RevenueStream  `json:"exchange_fees"`
	DailyTrend       []DailyRevenue `json:"daily_trend_7d"`
	CapturedAt       time.Time      `json:"captured_at"`
}

type RevenueStream struct {
	TodayCents int64   `json:"today_cents"`
	WeekCents  int64   `json:"week_cents"`
	Count      int64   `json:"count_today"`
	AvgCents   float64 `json:"avg_cents"`
}

type DailyRevenue struct {
	Date       string `json:"date"`
	TotalCents int64  `json:"total_cents"`
}

// GetRevenueDashboard queries all revenue tables read-only.
func GetRevenueDashboard(db *gorm.DB) RevenueDashboard {
	r := RevenueDashboard{CapturedAt: time.Now().UTC()}

	// Live commissions.
	db.Raw(`SELECT COALESCE(SUM(commission_cents),0) FROM live_commissions WHERE created_at >= CURRENT_DATE`).Scan(&r.Commissions.TodayCents)
	db.Raw(`SELECT COALESCE(SUM(commission_cents),0) FROM live_commissions WHERE created_at >= NOW() - INTERVAL '7 days'`).Scan(&r.Commissions.WeekCents)
	db.Raw(`SELECT COUNT(*), COALESCE(AVG(commission_cents),0) FROM live_commissions WHERE created_at >= CURRENT_DATE`).
		Row().Scan(&r.Commissions.Count, &r.Commissions.AvgCents)

	// Boost revenue.
	db.Raw(`SELECT COALESCE(SUM(price_cents),0) FROM live_boosts WHERE created_at >= CURRENT_DATE`).Scan(&r.Boosts.TodayCents)
	db.Raw(`SELECT COALESCE(SUM(price_cents),0) FROM live_boosts WHERE created_at >= NOW() - INTERVAL '7 days'`).Scan(&r.Boosts.WeekCents)
	db.Raw(`SELECT COUNT(*) FROM live_boosts WHERE created_at >= CURRENT_DATE`).Scan(&r.Boosts.Count)

	// Entry fees.
	db.Raw(`SELECT COALESCE(SUM(fee_cents),0) FROM live_paid_entries WHERE created_at >= CURRENT_DATE`).Scan(&r.EntryFees.TodayCents)
	db.Raw(`SELECT COALESCE(SUM(fee_cents),0) FROM live_paid_entries WHERE created_at >= NOW() - INTERVAL '7 days'`).Scan(&r.EntryFees.WeekCents)
	db.Raw(`SELECT COUNT(*) FROM live_paid_entries WHERE created_at >= CURRENT_DATE`).Scan(&r.EntryFees.Count)

	// Creator payouts (outgoing — tracked separately from revenue).
	db.Raw(`SELECT COALESCE(SUM(amount_cents),0) FROM live_creator_earnings WHERE created_at >= CURRENT_DATE`).Scan(&r.CreatorPayouts.TodayCents)
	db.Raw(`SELECT COALESCE(SUM(amount_cents),0) FROM live_creator_earnings WHERE created_at >= NOW() - INTERVAL '7 days'`).Scan(&r.CreatorPayouts.WeekCents)
	db.Raw(`SELECT COUNT(*) FROM live_creator_earnings WHERE created_at >= CURRENT_DATE`).Scan(&r.CreatorPayouts.Count)

	// VIP subscriptions.
	db.Raw(`SELECT COUNT(*) FROM vip_users WHERE created_at >= CURRENT_DATE`).Scan(&r.VIPSubscriptions.Count)
	db.Raw(`SELECT COUNT(*) FROM vip_users WHERE created_at >= NOW() - INTERVAL '7 days'`).Scan(&r.VIPSubscriptions.WeekCents) // reuse field as count

	// Exchange fees.
	db.Raw(`SELECT COALESCE(SUM(fee_amount_cents),0) FROM exchange_fees WHERE created_at >= CURRENT_DATE`).Scan(&r.ExchangeFees.TodayCents)
	db.Raw(`SELECT COALESCE(SUM(fee_amount_cents),0) FROM exchange_fees WHERE created_at >= NOW() - INTERVAL '7 days'`).Scan(&r.ExchangeFees.WeekCents)
	db.Raw(`SELECT COUNT(*) FROM exchange_fees WHERE created_at >= CURRENT_DATE`).Scan(&r.ExchangeFees.Count)

	// Totals.
	r.TodayTotal = r.Commissions.TodayCents + r.Boosts.TodayCents + r.EntryFees.TodayCents + r.ExchangeFees.TodayCents
	r.WeekTotal = r.Commissions.WeekCents + r.Boosts.WeekCents + r.EntryFees.WeekCents + r.ExchangeFees.WeekCents

	db.Raw(`SELECT COALESCE(SUM(commission_cents),0) FROM live_commissions WHERE created_at >= NOW() - INTERVAL '30 days'`).Scan(&r.MonthTotal)

	// 7-day daily trend.
	db.Raw(`
		SELECT
			TO_CHAR(DATE_TRUNC('day', created_at), 'YYYY-MM-DD') AS date,
			COALESCE(SUM(commission_cents), 0)                   AS total_cents
		FROM live_commissions
		WHERE created_at >= NOW() - INTERVAL '7 days'
		GROUP BY DATE_TRUNC('day', created_at)
		ORDER BY DATE_TRUNC('day', created_at)
	`).Scan(&r.DailyTrend)

	return r
}
