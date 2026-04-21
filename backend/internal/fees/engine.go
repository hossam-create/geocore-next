package fees

import (
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Engine applies configurable fees and caches rules in memory.
type Engine struct {
	db        *gorm.DB
	mu        sync.RWMutex
	cache     []FeeConfig
	cacheTime time.Time
	cacheTTL  time.Duration
}

// NewEngine creates a fee engine with a 5-minute rule cache.
func NewEngine(db *gorm.DB) *Engine {
	return &Engine{db: db, cacheTTL: 5 * time.Minute}
}

// rules returns cached fee configs, refreshing if stale.
func (e *Engine) rules() []FeeConfig {
	e.mu.RLock()
	if time.Since(e.cacheTime) < e.cacheTTL && len(e.cache) > 0 {
		defer e.mu.RUnlock()
		return e.cache
	}
	e.mu.RUnlock()

	e.mu.Lock()
	defer e.mu.Unlock()

	var configs []FeeConfig
	e.db.Where("is_active = true").Order("country DESC, min_amount ASC").Find(&configs)
	e.cache = configs
	e.cacheTime = time.Now()
	return configs
}

// Calculate applies the most appropriate fee rule for the given type/country/amount.
// Precedence: country-specific > wildcard; narrower amount range > wider.
func (e *Engine) Calculate(feeType FeeType, country string, amount float64) FeeResult {
	rules := e.rules()
	var best *FeeConfig

	for i := range rules {
		r := &rules[i]
		if r.FeeType != feeType {
			continue
		}
		if r.Country != "*" && r.Country != country {
			continue
		}
		if r.MinAmount > 0 && amount < r.MinAmount {
			continue
		}
		if r.MaxAmount > 0 && amount > r.MaxAmount {
			continue
		}
		// Prefer country-specific over wildcard
		if best == nil || (r.Country != "*" && best.Country == "*") {
			best = r
		}
	}

	if best == nil {
		return FeeResult{GrossAmount: amount, NetAmount: amount, Rule: "no_rule"}
	}

	feeAmount := amount*best.FeePct/100.0 + best.FeeFixed
	if best.MinFee > 0 && feeAmount < best.MinFee {
		feeAmount = best.MinFee
	}
	if best.MaxFee > 0 && feeAmount > best.MaxFee {
		feeAmount = best.MaxFee
	}
	feeAmount = math.Round(feeAmount*100) / 100

	return FeeResult{
		GrossAmount: amount,
		FeeAmount:   feeAmount,
		NetAmount:   math.Round((amount-feeAmount)*100) / 100,
		FeePct:      best.FeePct,
		FeeFixed:    best.FeeFixed,
		Rule:        fmt.Sprintf("%s/%s", best.FeeType, best.Country),
	}
}

// InvalidateCache forces next call to reload rules from DB.
func (e *Engine) InvalidateCache() {
	e.mu.Lock()
	e.cacheTime = time.Time{}
	e.mu.Unlock()
}

// SeedDefaults populates default fee configs if none exist.
func SeedDefaults(db *gorm.DB) {
	var count int64
	db.Model(&FeeConfig{}).Count(&count)
	if count > 0 {
		return
	}

	defaults := []FeeConfig{
		// Transaction fees
		{FeeType: FeeTypeTransaction, Country: "*", FeePct: 3.0, FeeFixed: 0, MinFee: 0.5, MaxFee: 50, IsActive: true},
		{FeeType: FeeTypeTransaction, Country: "EGY", FeePct: 2.5, FeeFixed: 0, MinFee: 1.0, MaxFee: 100, IsActive: true},
		{FeeType: FeeTypeTransaction, Country: "ARE", FeePct: 2.0, FeeFixed: 0, MinFee: 1.0, MaxFee: 100, IsActive: true},

		// Escrow fees
		{FeeType: FeeTypeEscrow, Country: "*", FeePct: 1.0, FeeFixed: 0, MinFee: 0.5, MaxFee: 20, IsActive: true},

		// Forex spread (on top of mid-market)
		{FeeType: FeeTypeForexSpread, Country: "*", FeePct: 0.5, FeeFixed: 0, IsActive: true},

		// Withdrawal fees
		{FeeType: FeeTypeWithdrawal, Country: "*", FeePct: 1.5, FeeFixed: 1.0, MinFee: 1.0, MaxFee: 25, IsActive: true},
		{FeeType: FeeTypeWithdrawal, Country: "EGY", FeePct: 1.0, FeeFixed: 5.0, MinFee: 5.0, MaxFee: 50, IsActive: true},

		// Referral bonus (credited to referrer wallet)
		{FeeType: FeeTypeReferral, Country: "*", FeeFixed: 5.0, IsActive: true},
	}

	for _, d := range defaults {
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(&d)
	}
	slog.Info("fee engine: seeded default configs")
}

// global singleton
var (
	globalEngine *Engine
	globalMu     sync.Mutex
)

// Global returns the singleton fee engine (initialised by Init).
func Global() *Engine {
	globalMu.Lock()
	defer globalMu.Unlock()
	return globalEngine
}

// Init initialises the global fee engine.
func Init(db *gorm.DB) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalEngine = NewEngine(db)
}
