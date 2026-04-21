package country

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ── Repository ─────────────────────────────────────────────────────────────────

type Repository struct {
	db  *gorm.DB
	rdb *redis.Client
	mu  sync.RWMutex
	cache map[string]*CountryConfig // in-memory fallback
}

func NewRepository(db *gorm.DB, rdb *redis.Client) *Repository {
	return &Repository{
		db:    db,
		rdb:   rdb,
		cache: make(map[string]*CountryConfig),
	}
}

// GetConfig retrieves a country's configuration, checking Redis cache first,
// then DB, then in-memory fallback.
func (r *Repository) GetConfig(ctx context.Context, code string) (*CountryConfig, error) {
	key := "country:config:" + code

	// 1. Try Redis
	if r.rdb != nil {
		val, err := r.rdb.Get(ctx, key).Bytes()
		if err == nil && val != nil {
			var cfg CountryConfig
			if json.Unmarshal(val, &cfg) == nil {
				return &cfg, nil
			}
		}
	}

	// 2. Try in-memory cache
	r.mu.RLock()
	if cached, ok := r.cache[code]; ok {
		r.mu.RUnlock()
		return cached, nil
	}
	r.mu.RUnlock()

	// 3. Query DB
	var cfg CountryConfig
	if err := r.db.Where("code = ? AND is_active = ?", code, true).First(&cfg).Error; err != nil {
		return nil, fmt.Errorf("country config not found: %s", code)
	}

	// Cache it
	r.cacheSet(ctx, key, &cfg)
	return &cfg, nil
}

// GetAllConfigs returns all active country configurations.
func (r *Repository) GetAllConfigs(ctx context.Context) ([]CountryConfig, error) {
	var configs []CountryConfig
	if err := r.db.Where("is_active = ?", true).Order("code").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// UpsertConfig creates or updates a country configuration.
func (r *Repository) UpsertConfig(ctx context.Context, cfg *CountryConfig) error {
	if err := r.db.Save(cfg).Error; err != nil {
		return err
	}
	r.cacheSet(ctx, "country:config:"+cfg.Code, cfg)
	return nil
}

// DeleteConfig soft-deletes by setting is_active = false.
func (r *Repository) DeleteConfig(ctx context.Context, code string) error {
	if err := r.db.Model(&CountryConfig{}).Where("code = ?", code).Update("is_active", false).Error; err != nil {
		return err
	}
	if r.rdb != nil {
		r.rdb.Del(ctx, "country:config:"+code)
	}
	r.mu.Lock()
	delete(r.cache, code)
	r.mu.Unlock()
	return nil
}

// ── Overrides ──────────────────────────────────────────────────────────────────

func (r *Repository) GetOverrides(ctx context.Context, countryCode, targetType, targetID string) ([]CountryOverride, error) {
	var overrides []CountryOverride
	q := r.db.Where("country_code = ?", countryCode)
	if targetType != "" {
		q = q.Where("target_type = ?", targetType)
	}
	if targetID != "" {
		q = q.Where("target_id = ?", targetID)
	}
	if err := q.Order("field").Find(&overrides).Error; err != nil {
		return nil, err
	}
	return overrides, nil
}

func (r *Repository) CreateOverride(ctx context.Context, o *CountryOverride) error {
	return r.db.Create(o).Error
}

func (r *Repository) DeleteOverride(ctx context.Context, id string) error {
	return r.db.Where("id = ?", id).Delete(&CountryOverride{}).Error
}

// ── Resolve ────────────────────────────────────────────────────────────────────

// ResolveConfig computes the final config for a user in a country,
// applying any user-level or category-level overrides.
func (r *Repository) ResolveConfig(ctx context.Context, countryCode, userID string) (*ResolvedConfig, error) {
	cfg, err := r.GetConfig(ctx, countryCode)
	if err != nil {
		return nil, err
	}

	resolved := &ResolvedConfig{
		CountryCode:                cfg.Code,
		Currency:                   cfg.Currency,
		CurrencyAr:                 cfg.CurrencyAr,
		CurrencySymbol:             cfg.CurrencySymbol,
		TaxRate:                    cfg.TaxRate,
		TaxLabel:                   cfg.TaxLabel,
		TaxInclusive:               cfg.TaxInclusive,
		ServiceFeeRate:             cfg.ServiceFeeRate,
		WithholdingRate:            cfg.WithholdingRate,
		KYCTier1LimitCents:         cfg.KYCTier1LimitCents,
		KYCTier2LimitCents:         cfg.KYCTier2LimitCents,
		KYCTier3LimitCents:         cfg.KYCTier3LimitCents,
		MaxListingPriceCents:       cfg.MaxListingPriceCents,
		PaymentMethods:             cfg.PaymentMethods,
		EnableAuctions:             cfg.EnableAuctions,
		EnableLive:                 cfg.EnableLive,
		EnableBNPL:                 cfg.EnableBNPL,
		EnableP2P:                  cfg.EnableP2P,
		EnableCrypto:               cfg.EnableCrypto,
		EnableCrowdship:            cfg.EnableCrowdship,
		EnableWholesale:            cfg.EnableWholesale,
		EnableRealEstate:           cfg.EnableRealEstate,
		DefaultShippingCents:       cfg.DefaultShippingCents,
		FreeShippingThresholdCents: cfg.FreeShippingThresholdCents,
		Locale:                     cfg.Locale,
		RequireNationalID:          cfg.RequireNationalID,
		RequireAddressProof:        cfg.RequireAddressProof,
		MinAge:                     cfg.MinAge,
		MaxReturnDays:              cfg.MaxReturnDays,
	}

	// Apply user-level overrides
	if userID != "" {
		overrides, err := r.GetOverrides(ctx, countryCode, "user", userID)
		if err == nil {
			applyOverrides(resolved, overrides)
		}
	}

	return resolved, nil
}

// ── Helpers ────────────────────────────────────────────────────────────────────

func (r *Repository) cacheSet(ctx context.Context, key string, cfg *CountryConfig) {
	if r.rdb != nil {
		data, err := json.Marshal(cfg)
		if err == nil {
			r.rdb.Set(ctx, key, data, 10*time.Minute)
		}
	}
	r.mu.Lock()
	r.cache[cfg.Code] = cfg
	r.mu.Unlock()
}

// applyOverrides patches the resolved config with override values.
func applyOverrides(rc *ResolvedConfig, overrides []CountryOverride) {
	for _, o := range overrides {
		switch o.Field {
		case "tax_rate":
			if v, err := strconv.ParseFloat(o.Value, 64); err == nil {
				rc.TaxRate = v
			}
		case "service_fee_rate":
			if v, err := strconv.ParseFloat(o.Value, 64); err == nil {
				rc.ServiceFeeRate = v
			}
		case "withholding_rate":
			if v, err := strconv.ParseFloat(o.Value, 64); err == nil {
				rc.WithholdingRate = v
			}
		case "kyc_tier1_limit_cents":
			if v, err := strconv.ParseInt(o.Value, 10, 64); err == nil {
				rc.KYCTier1LimitCents = v
			}
		case "kyc_tier2_limit_cents":
			if v, err := strconv.ParseInt(o.Value, 10, 64); err == nil {
				rc.KYCTier2LimitCents = v
			}
		case "kyc_tier3_limit_cents":
			if v, err := strconv.ParseInt(o.Value, 10, 64); err == nil {
				rc.KYCTier3LimitCents = v
			}
		case "max_listing_price_cents":
			if v, err := strconv.ParseInt(o.Value, 10, 64); err == nil {
				rc.MaxListingPriceCents = v
			}
		case "payment_methods":
			var methods []string
			if json.Unmarshal([]byte(o.Value), &methods) == nil {
				rc.PaymentMethods = methods
			}
		case "enable_auctions":
			rc.EnableAuctions = o.Value == "true"
		case "enable_live":
			rc.EnableLive = o.Value == "true"
		case "enable_bnpl":
			rc.EnableBNPL = o.Value == "true"
		case "enable_p2p":
			rc.EnableP2P = o.Value == "true"
		case "enable_crypto":
			rc.EnableCrypto = o.Value == "true"
		case "enable_crowdship":
			rc.EnableCrowdship = o.Value == "true"
		case "enable_wholesale":
			rc.EnableWholesale = o.Value == "true"
		case "enable_real_estate":
			rc.EnableRealEstate = o.Value == "true"
		case "default_shipping_cents":
			if v, err := strconv.ParseInt(o.Value, 10, 64); err == nil {
				rc.DefaultShippingCents = v
			}
		case "free_shipping_threshold_cents":
			if v, err := strconv.ParseInt(o.Value, 10, 64); err == nil {
				rc.FreeShippingThresholdCents = v
			}
		default:
			slog.Warn("country: unknown override field", "field", o.Field)
		}
	}
}
