package exchange

// rate_hint.go — Part 3 + Final Patch: Rate Guidance with action suggestions.
//
// Shows users a recommended rate derived from open requests + external anchor.
// Platform NEVER enforces any price — users agree freely.

import (
	"fmt"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ActionSuggestion tells the user how to improve their match probability.
type ActionSuggestion struct {
	AdjustRatePercent float64 `json:"adjust_rate_percent"` // e.g. -1.5 means "lower by 1.5%"
	ExpectedMatchTime string  `json:"expected_match_time"` // e.g. "1-3 min"
	MatchProbability  float64 `json:"match_probability"`   // 0–100%
	Reason            string  `json:"reason"`
}

// RateHint provides advisory rate information for a currency pair.
type RateHint struct {
	Pair             string            `json:"pair"`            // "EGP/USD"
	MarketRate       float64           `json:"market_rate"`     // blended internal + external
	InternalRate     float64           `json:"internal_rate"`   // median of user preferred_rates
	ExternalRate     float64           `json:"external_rate"`   // from forex DB / API
	ExternalSource   string            `json:"external_source"` // "forex_db" | "redis_cache" | "none"
	BestMatchRate    float64           `json:"best_match_rate"` // best available counterparty rate
	Spread           float64           `json:"spread"`
	SampleSize       int               `json:"sample_size"`
	Quality          string            `json:"quality"`       // good_deal | fair | below_market
	SpreadStatus     string            `json:"spread_status"` // tight | normal | wide
	Warning          string            `json:"warning,omitempty"`
	ActionSuggestion *ActionSuggestion `json:"action_suggestion,omitempty"`
	ManipulationFlag bool              `json:"manipulation_flag"`
	Disclaimer       string            `json:"disclaimer"`
}

const rateDisclaimer = "Rate hints are advisory only. The platform does not set or enforce any exchange rate."

// GetRateHint computes an advisory rate hint for a currency pair from open requests.
func GetRateHint(db *gorm.DB, from, to string) RateHint {
	return GetRateHintWithAnchor(db, nil, from, to)
}

// GetRateHintWithAnchor computes a rate hint using an external market anchor.
func GetRateHintWithAnchor(db *gorm.DB, rdb *redis.Client, from, to string) RateHint {
	hint := RateHint{
		Pair:       from + "/" + to,
		Disclaimer: rateDisclaimer,
	}

	// 1. Get external anchor (forex DB + Redis cache)
	anchor := GetMarketAnchor(db, rdb, from, to)
	hint.ExternalRate = anchor.ExternalRate
	hint.ExternalSource = anchor.ExternalSource
	hint.MarketRate = anchor.BlendedRate
	hint.InternalRate = anchor.InternalRate

	// 2. Best match rate: best counterparty offer (includes influencer seeds)
	var counterRates []float64
	db.Model(&ExchangeRequest{}).
		Where("from_currency=? AND to_currency=? AND status=? AND preferred_rate IS NOT NULL AND preferred_rate>0",
			to, from, StatusOpen).
		Pluck("preferred_rate", &counterRates)

	hint.SampleSize = 0
	var userRates []float64
	db.Model(&ExchangeRequest{}).
		Where("from_currency=? AND to_currency=? AND status=? AND preferred_rate IS NOT NULL AND preferred_rate>0 AND is_system_generated=?",
			from, to, StatusOpen, false).
		Pluck("preferred_rate", &userRates)
	hint.SampleSize = len(userRates)

	if len(counterRates) > 0 {
		best := 0.0
		for _, r := range counterRates {
			if r > best {
				best = r
			}
		}
		if best > 0 {
			hint.BestMatchRate = round2(1.0 / best)
		}
	} else if hint.MarketRate > 0 {
		hint.BestMatchRate = hint.MarketRate
	}

	// 3. Spread analysis
	if hint.MarketRate > 0 && hint.BestMatchRate > 0 {
		hint.Spread = round2(hint.BestMatchRate - hint.MarketRate)
		spreadPct := hint.Spread / hint.MarketRate * 100
		switch {
		case spreadPct > 2:
			hint.SpreadStatus = "wide"
		case spreadPct < -2:
			hint.SpreadStatus = "wide"
		case abs(spreadPct) < 0.5:
			hint.SpreadStatus = "tight"
		default:
			hint.SpreadStatus = "normal"
		}
	}

	// 4. Quality label
	switch {
	case hint.MarketRate > 0 && hint.Spread > hint.MarketRate*0.02:
		hint.Quality = "good_deal"
	case hint.MarketRate > 0 && hint.Spread < -hint.MarketRate*0.02:
		hint.Quality = "below_market"
		hint.Warning = "This rate is below the market average. You may find a better deal by waiting."
	default:
		hint.Quality = "fair"
	}

	// 5. Rate manipulation detection
	manipUsers := DetectRateManipulation(db, from, to)
	hint.ManipulationFlag = len(manipUsers) > 0

	// 6. Action suggestion — simulate matching against current order book
	hint.ActionSuggestion = computeActionSuggestion(db, from, to, hint.MarketRate)

	return hint
}

// computeActionSuggestion simulates the order book to estimate match probability.
func computeActionSuggestion(db *gorm.DB, from, to string, marketRate float64) *ActionSuggestion {
	if marketRate <= 0 {
		return nil
	}

	// Count counterparties (opposite direction)
	var counterCount int64
	db.Model(&ExchangeRequest{}).
		Where("from_currency=? AND to_currency=? AND status=? AND is_system_generated=?",
			to, from, StatusOpen, false).
		Count(&counterCount)

	// Count same-direction requests
	var sameCount int64
	db.Model(&ExchangeRequest{}).
		Where("from_currency=? AND to_currency=? AND status=? AND is_system_generated=?",
			from, to, StatusOpen, false).
		Count(&sameCount)

	// Match probability: based on counterparty availability and competition
	prob := 0.0
	if counterCount > 0 && sameCount > 0 {
		prob = float64(counterCount) / float64(counterCount+sameCount) * 100
	} else if counterCount > 0 {
		prob = 90 // no competition
	} else {
		prob = 5 // no counterparties
	}

	// Expected match time estimate
	expectedTime := "10+ min"
	switch {
	case prob > 70:
		expectedTime = "1-3 min"
	case prob > 40:
		expectedTime = "3-7 min"
	case prob > 15:
		expectedTime = "7-15 min"
	}

	// Suggest rate adjustment if probability is low
	adjustPct := 0.0
	reason := ""
	if prob < 50 && counterCount > 0 {
		// Get counterparty rates to find what rate would match
		var counterRates []float64
		db.Model(&ExchangeRequest{}).
			Where("from_currency=? AND to_currency=? AND status=? AND preferred_rate IS NOT NULL AND preferred_rate>0 AND is_system_generated=?",
				to, from, StatusOpen, false).
			Pluck("preferred_rate", &counterRates)
		if len(counterRates) > 0 {
			bestCounter := 0.0
			for _, r := range counterRates {
				if r > bestCounter {
					bestCounter = r
				}
			}
			if bestCounter > 0 {
				impliedRate := 1.0 / bestCounter
				adjustPct = round2((impliedRate - marketRate) / marketRate * 100)
				if adjustPct < 0 {
					reason = "Lower your rate to align with the best available counterparty offer."
				} else {
					reason = "Your rate is competitive. Consider raising slightly for better terms."
				}
			}
		}
	} else if prob < 20 {
		adjustPct = -1.5
		reason = "No counterparties available. Lowering your rate may attract matches when they appear."
	} else {
		reason = "Your rate is well-positioned for a quick match."
	}

	return &ActionSuggestion{
		AdjustRatePercent: round2(adjustPct),
		ExpectedMatchTime: expectedTime,
		MatchProbability:  round2(prob),
		Reason:            reason,
	}
}

// NoMatchFeedback returns actionable feedback when a match attempt fails.
func NoMatchFeedback(db *gorm.DB, rdb *redis.Client, from, to string, userRate float64) map[string]interface{} {
	hint := GetRateHintWithAnchor(db, rdb, from, to)
	reason := "no counterparties"
	suggestion := ""
	expectedTime := ""

	if hint.BestMatchRate > 0 && userRate > 0 {
		gap := (userRate - hint.BestMatchRate) / hint.BestMatchRate * 100
		switch {
		case gap > 5:
			reason = "rate too high"
			suggestion = fmt.Sprintf("lower rate by %.1f%%", gap)
			expectedTime = "1-3 min"
		case gap > 2:
			reason = "rate slightly above market"
			suggestion = fmt.Sprintf("lower rate by %.1f%%", gap)
			expectedTime = "3-7 min"
		default:
			reason = "no available counterparty right now"
			suggestion = "your rate is fair — wait for a counterparty"
			expectedTime = "5-15 min"
		}
	}

	return map[string]interface{}{
		"reason":        reason,
		"suggestion":    suggestion,
		"expected_time": expectedTime,
		"rate_hint":     hint,
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// median returns the median of a float64 slice.
func median(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	s := make([]float64, len(data))
	copy(s, data)
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
	n := len(s)
	if n%2 == 0 {
		return (s[n/2-1] + s[n/2]) / 2
	}
	return s[n/2]
}
