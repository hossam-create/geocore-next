package payments

import (
	"context"
	"fmt"
	"time"

	"github.com/geocore-next/backend/internal/forex"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// FXService provides currency conversion using Redis cache + DB fallback.
type FXService struct {
	db  *gorm.DB
	rdb *redis.Client
}

// NewFXService creates a new FX service instance.
func NewFXService(db *gorm.DB, rdb *redis.Client) *FXService {
	return &FXService{db: db, rdb: rdb}
}

// GetRate returns the exchange rate from `from` currency to `to` currency.
// Uses Redis cache (5min TTL) with DB fallback. Returns 1.0 if same currency or rate not found.
func (fx *FXService) GetRate(from, to string) decimal.Decimal {
	if from == to {
		return decimal.NewFromInt(1)
	}

	// Try Redis cache
	if fx.rdb != nil {
		cacheKey := fmt.Sprintf("fx_rate:%s:%s", from, to)
		cached, err := fx.rdb.Get(context.Background(), cacheKey).Result()
		if err == nil {
			rate, _ := decimal.NewFromString(cached)
			if !rate.IsZero() {
				return rate
			}
		}
	}

	// Fallback to DB (exchange_rates table)
	var rate forex.ExchangeRate
	if fx.db != nil {
		fx.db.Where("from_currency = ? AND to_currency = ? AND valid_to IS NULL",
			from, to).Order("created_at DESC").First(&rate)
		if rate.EffectiveRate > 0 {
			dRate := decimal.NewFromFloat(rate.EffectiveRate)
			// Cache for 5 minutes
			if fx.rdb != nil {
				cacheKey := fmt.Sprintf("fx_rate:%s:%s", from, to)
				fx.rdb.Set(context.Background(), cacheKey, dRate.String(), 5*time.Minute)
			}
			return dRate
		}
	}

	// Final fallback: return 1.0 (no conversion)
	return decimal.NewFromInt(1)
}
