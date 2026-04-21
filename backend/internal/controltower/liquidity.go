package controltower

import (
	"time"

	"gorm.io/gorm"
)

// LiquidityDashboard shows exchange depth and match health.
type LiquidityDashboard struct {
	Pairs           []PairLiquidity `json:"pairs"`
	OverallBalance  float64         `json:"overall_balance_ratio"` // 0=perfect imbalance, 1=perfect balance
	MatchSuccessRate float64        `json:"match_success_rate_pct"`
	AvgMatchSeconds float64         `json:"avg_match_time_seconds"`
	OpenRequests    int64           `json:"open_requests"`
	MatchedToday    int64           `json:"matched_today"`
	CapturedAt      time.Time       `json:"captured_at"`
}

type PairLiquidity struct {
	CurrencyFrom    string  `json:"currency_from"`
	CurrencyTo      string  `json:"currency_to"`
	BuyVolume       float64 `json:"buy_volume"`
	SellVolume      float64 `json:"sell_volume"`
	ImbalanceRatio  float64 `json:"imbalance_ratio"` // |buy-sell|/(buy+sell), 0=balanced
	AvailableBuyers int64   `json:"available_buyers"`
	AvailableSellers int64  `json:"available_sellers"`
}

// GetLiquidityDashboard aggregates liquidity data read-only from exchange tables.
func GetLiquidityDashboard(db *gorm.DB) LiquidityDashboard {
	d := LiquidityDashboard{CapturedAt: time.Now().UTC()}

	// Per-pair liquidity from the liquidity profiles table.
	db.Raw(`
		SELECT
			currency_from,
			currency_to,
			buy_volume,
			sell_volume,
			CASE
				WHEN (buy_volume + sell_volume) = 0 THEN 0
				ELSE ABS(buy_volume - sell_volume) / (buy_volume + sell_volume)
			END AS imbalance_ratio,
			available_buyers,
			available_sellers
		FROM exchange_liquidity_profiles
		ORDER BY (buy_volume + sell_volume) DESC
		LIMIT 20
	`).Scan(&d.Pairs)

	// Overall balance ratio (average across pairs).
	if len(d.Pairs) > 0 {
		total := 0.0
		for _, p := range d.Pairs {
			total += 1.0 - p.ImbalanceRatio
		}
		d.OverallBalance = total / float64(len(d.Pairs))
	}

	// Match success rate: matched / (matched + unmatched expired) in last 24h.
	var matched, expired int64
	db.Raw(`SELECT COUNT(*) FROM exchange_matches WHERE created_at > NOW() - INTERVAL '24 hours'`).Scan(&matched)
	db.Raw(`SELECT COUNT(*) FROM exchange_requests WHERE status = 'expired' AND updated_at > NOW() - INTERVAL '24 hours'`).Scan(&expired)
	if total := matched + expired; total > 0 {
		d.MatchSuccessRate = float64(matched) / float64(total) * 100
	}

	// Average match time: time between request creation and match creation.
	db.Raw(`
		SELECT COALESCE(
			EXTRACT(EPOCH FROM AVG(m.created_at - r.created_at)), 0
		)
		FROM exchange_matches m
		JOIN exchange_requests r ON r.id = m.buy_request_id OR r.id = m.sell_request_id
		WHERE m.created_at > NOW() - INTERVAL '24 hours'
	`).Scan(&d.AvgMatchSeconds)

	// Open requests count.
	db.Raw(`SELECT COUNT(*) FROM exchange_requests WHERE status = 'open'`).Scan(&d.OpenRequests)

	// Matched today.
	db.Raw(`SELECT COUNT(*) FROM exchange_matches WHERE created_at >= CURRENT_DATE`).Scan(&d.MatchedToday)

	return d
}
