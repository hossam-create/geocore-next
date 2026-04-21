package exchange

// market_anchor.go — Part 1 (Final Patch): External Market Rate Anchor.
//
// Prevents self-referential rate manipulation by blending internal
// preferred-rate median with an external FX anchor (from forex DB or API).
// Cached in Redis with 60s TTL. Falls back gracefully when no external source.

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/forex"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	// Redis key pattern: exchange:anchor:{from}:{to}
	anchorKeyPattern = "exchange:anchor:%s:%s"
	anchorTTL        = 60 * time.Second

	// Weight of external rate in weighted average (0–1).
	// 0.7 = 70% external, 30% internal → strong anti-manipulation bias.
	externalWeight = 0.7
)

// MarketAnchor holds the blended rate for a pair.
type MarketAnchor struct {
	Pair           string    `json:"pair"`
	InternalRate   float64   `json:"internal_rate"`   // median of user preferred_rates
	ExternalRate   float64   `json:"external_rate"`   // from forex DB / API
	BlendedRate    float64   `json:"blended_rate"`    // weighted average
	ExternalSource string    `json:"external_source"` // "forex_db" | "redis_cache" | "none"
	CachedAt       time.Time `json:"cached_at"`
}

// GetMarketAnchor returns the blended market rate for a pair.
// Uses Redis cache (60s TTL), falls back to forex DB, then internal-only.
func GetMarketAnchor(db *gorm.DB, rdb *redis.Client, from, to string) MarketAnchor {
	pair := from + "/" + to
	anchor := MarketAnchor{Pair: pair}

	// 1. Compute internal rate (median of current open preferred_rates)
	anchor.InternalRate = computeInternalRate(db, from, to)

	// 2. Try Redis cache for external rate
	if rdb != nil {
		key := fmt.Sprintf(anchorKeyPattern, from, to)
		cached, err := rdb.Get(context.Background(), key).Float64()
		if err == nil && cached > 0 {
			anchor.ExternalRate = round2(cached)
			anchor.ExternalSource = "redis_cache"
			anchor.BlendedRate = round2(blend(anchor.InternalRate, anchor.ExternalRate))
			return anchor
		}
	}

	// 3. Fallback: forex DB (exchange_rates table)
	if db != nil {
		var rate forex.ExchangeRate
		db.Where("from_currency = ? AND to_currency = ? AND valid_to IS NULL",
			from, to).Order("created_at DESC").First(&rate)
		if rate.EffectiveRate > 0 {
			anchor.ExternalRate = round2(rate.EffectiveRate)
			anchor.ExternalSource = "forex_db"
			// Cache in Redis
			if rdb != nil {
				key := fmt.Sprintf(anchorKeyPattern, from, to)
				rdb.Set(context.Background(), key, anchor.ExternalRate, anchorTTL)
			}
		}
	}

	// 4. Blend or fallback
	if anchor.ExternalRate > 0 && anchor.InternalRate > 0 {
		anchor.BlendedRate = round2(blend(anchor.InternalRate, anchor.ExternalRate))
	} else if anchor.ExternalRate > 0 {
		anchor.BlendedRate = anchor.ExternalRate
	} else if anchor.InternalRate > 0 {
		anchor.BlendedRate = anchor.InternalRate
		anchor.ExternalSource = "none"
	} else {
		anchor.ExternalSource = "none"
	}

	anchor.CachedAt = time.Now()
	return anchor
}

// blend returns the weighted average of internal and external rates.
func blend(internal, external float64) float64 {
	return internal*(1-externalWeight) + external*externalWeight
}

// computeInternalRate returns the median preferred rate from open requests.
func computeInternalRate(db *gorm.DB, from, to string) float64 {
	var rates []float64
	db.Model(&ExchangeRequest{}).
		Where("from_currency=? AND to_currency=? AND status=? AND preferred_rate IS NOT NULL AND preferred_rate>0 AND is_system_generated=?",
			from, to, StatusOpen, false).
		Pluck("preferred_rate", &rates)
	if len(rates) == 0 {
		return 0
	}
	return round2(median(rates))
}

// DetectRateManipulation checks for suspicious clustering of similar rates.
// Returns a list of user IDs whose rates appear manipulated.
func DetectRateManipulation(db *gorm.DB, from, to string) []uuid.UUID {
	var rates []struct {
		UserID        uuid.UUID
		PreferredRate float64
	}
	db.Model(&ExchangeRequest{}).
		Where("from_currency=? AND to_currency=? AND status=? AND preferred_rate IS NOT NULL AND preferred_rate>0 AND is_system_generated=?",
			from, to, StatusOpen, false).
		Select("user_id, preferred_rate").
		Find(&rates)

	if len(rates) < 3 {
		return nil
	}

	// Compute median and detect clusters within 0.5% band
	medianRate := median(mapToFloats(rates))
	var flagged []uuid.UUID
	clusterCount := 0
	tightBand := medianRate * 0.005 // 0.5% band

	for _, r := range rates {
		diff := r.PreferredRate - medianRate
		if diff < 0 {
			diff = -diff
		}
		if diff <= tightBand {
			clusterCount++
		}
	}

	// If >60% of rates are within a 0.5% band, flag those users
	threshold := float64(len(rates)) * 0.6
	if float64(clusterCount) > threshold && len(rates) >= 5 {
		for _, r := range rates {
			diff := r.PreferredRate - medianRate
			if diff < 0 {
				diff = -diff
			}
			if diff <= tightBand {
				flagged = append(flagged, r.UserID)
			}
		}
		slog.Warn("exchange: rate manipulation cluster detected",
			"pair", from+"/"+to, "cluster_size", clusterCount, "total", len(rates))
	}
	return flagged
}

func mapToFloats(rates []struct {
	UserID        uuid.UUID
	PreferredRate float64
}) []float64 {
	out := make([]float64, len(rates))
	for i, r := range rates {
		out[i] = r.PreferredRate
	}
	return out
}
