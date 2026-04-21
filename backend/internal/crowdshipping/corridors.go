package crowdshipping

import (
	"strings"
	"sync"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// ── Corridor types ────────────────────────────────────────────────────────────

type EscrowPolicy string

const (
	EscrowAlwaysRecommended EscrowPolicy = "ALWAYS_RECOMMENDED"
	EscrowHighValueOnly     EscrowPolicy = "HIGH_VALUE_ONLY"
	EscrowOptional          EscrowPolicy = "OPTIONAL"
)

type TrustRequirement string

const (
	TrustVerified TrustRequirement = "VERIFIED"
	TrustTrusted  TrustRequirement = "TRUSTED"
	TrustStandard TrustRequirement = "STANDARD"
	TrustNew      TrustRequirement = "NEW"
	TrustAny      TrustRequirement = "ANY"
)

type ValueBand struct {
	MinValue   float64 `json:"min_value"`
	MaxValue   float64 `json:"max_value"`
	Multiplier float64 `json:"multiplier"`
	Label      string  `json:"label"`
}

type DeliveryWindow struct {
	MinDays    int     `json:"min_days"`
	MaxDays    int     `json:"max_days"`
	Multiplier float64 `json:"multiplier"`
	Label      string  `json:"label"`
}

type RiskConfig struct {
	CustomsMultiplier float64          `json:"customs_multiplier"`
	ValueBands        []ValueBand      `json:"value_bands"`
	DeliveryWindows   []DeliveryWindow `json:"delivery_windows"`
}

type TrustConfig struct {
	HighValueThreshold float64          `json:"high_value_threshold"`
	MinBuyerTrust      TrustRequirement `json:"min_buyer_trust"`
	MinTravelerTrust   TrustRequirement `json:"min_traveler_trust"`
}

type CorridorConfig struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Origin       string       `json:"origin"`
	Destinations []string     `json:"destinations"`
	Enabled      bool         `json:"enabled"`
	Version      int          `json:"version"`
	Risk         RiskConfig   `json:"risk"`
	Trust        TrustConfig  `json:"trust"`
	EscrowPolicy EscrowPolicy `json:"escrow_policy"`
	Restrictions []string     `json:"restrictions"`
}

// ── Corridor Repository ───────────────────────────────────────────────────────

type CorridorRepository interface {
	FindAll() []CorridorConfig
	FindByRoute(origin, dest string) *CorridorConfig
}

type memoryCorridorRepo struct {
	mu      sync.RWMutex
	configs []CorridorConfig
}

func newMemoryCorridorRepo(configs []CorridorConfig) *memoryCorridorRepo {
	return &memoryCorridorRepo{configs: configs}
}

func (r *memoryCorridorRepo) FindAll() []CorridorConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]CorridorConfig, 0, len(r.configs))
	for _, c := range r.configs {
		if c.Enabled {
			result = append(result, c)
		}
	}
	return result
}

func (r *memoryCorridorRepo) FindByRoute(origin, dest string) *CorridorConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := range r.configs {
		c := &r.configs[i]
		if !c.Enabled {
			continue
		}
		if !strings.EqualFold(c.Origin, origin) {
			continue
		}
		for _, d := range c.Destinations {
			if strings.EqualFold(d, dest) {
				return c
			}
		}
	}
	return nil
}

var (
	globalRepoMu sync.RWMutex
	globalRepo   CorridorRepository
)

func init() {
	globalRepo = newMemoryCorridorRepo(initCorridors())
}

func GetCorridorRepository() CorridorRepository {
	globalRepoMu.RLock()
	defer globalRepoMu.RUnlock()
	return globalRepo
}

func SetCorridorRepository(repo CorridorRepository) {
	globalRepoMu.Lock()
	defer globalRepoMu.Unlock()
	globalRepo = repo
}

func initCorridors() []CorridorConfig {
	var usEG CorridorConfig
	usEG.ID = "US_EG"
	usEG.Name = "United States → Egypt"
	usEG.Origin = "US"
	usEG.Destinations = []string{"EG"}
	usEG.Enabled = true
	usEG.Version = 1
	usEG.Risk.CustomsMultiplier = 1.3
	usEG.Risk.ValueBands = []ValueBand{
		{0, 100, 1.0, "Low Value"},
		{100, 200, 1.1, "Standard"},
		{200, 500, 1.3, "Elevated"},
		{500, 2000, 1.5, "High Value"},
		{2000, 999999999, 2.0, "Very High"},
	}
	usEG.Risk.DeliveryWindows = []DeliveryWindow{
		{1, 7, 1.3, "Express"},
		{7, 14, 1.0, "Standard"},
		{14, 30, 0.9, "Economy"},
	}
	usEG.Trust.HighValueThreshold = 200
	usEG.Trust.MinBuyerTrust = TrustTrusted
	usEG.Trust.MinTravelerTrust = TrustTrusted
	usEG.EscrowPolicy = EscrowAlwaysRecommended
	usEG.Restrictions = []string{"electronics_batteries", "liquids_over_100ml", "restricted_medications"}

	var usAE CorridorConfig
	usAE.ID = "US_AE"
	usAE.Name = "United States → UAE"
	usAE.Origin = "US"
	usAE.Destinations = []string{"AE"}
	usAE.Enabled = true
	usAE.Version = 1
	usAE.Risk.CustomsMultiplier = 1.1
	usAE.Risk.ValueBands = []ValueBand{
		{0, 100, 1.0, "Low Value"},
		{100, 200, 1.05, "Standard"},
		{200, 500, 1.2, "Elevated"},
		{500, 2000, 1.4, "High Value"},
		{2000, 999999999, 1.8, "Very High"},
	}
	usAE.Risk.DeliveryWindows = []DeliveryWindow{
		{1, 5, 1.2, "Express"},
		{5, 10, 1.0, "Standard"},
		{10, 21, 0.9, "Economy"},
	}
	usAE.Trust.HighValueThreshold = 200
	usAE.Trust.MinBuyerTrust = TrustTrusted
	usAE.Trust.MinTravelerTrust = TrustTrusted
	usAE.EscrowPolicy = EscrowAlwaysRecommended
	usAE.Restrictions = []string{"alcohol", "pork_products", "restricted_medications"}

	var usSA CorridorConfig
	usSA.ID = "US_SA"
	usSA.Name = "United States → Saudi Arabia"
	usSA.Origin = "US"
	usSA.Destinations = []string{"SA"}
	usSA.Enabled = true
	usSA.Version = 1
	usSA.Risk.CustomsMultiplier = 1.4
	usSA.Risk.ValueBands = []ValueBand{
		{0, 100, 1.0, "Low Value"},
		{100, 200, 1.15, "Standard"},
		{200, 500, 1.35, "Elevated"},
		{500, 2000, 1.6, "High Value"},
		{2000, 999999999, 2.2, "Very High"},
	}
	usSA.Risk.DeliveryWindows = []DeliveryWindow{
		{1, 7, 1.3, "Express"},
		{7, 14, 1.0, "Standard"},
		{14, 28, 0.85, "Economy"},
	}
	usSA.Trust.HighValueThreshold = 200
	usSA.Trust.MinBuyerTrust = TrustTrusted
	usSA.Trust.MinTravelerTrust = TrustTrusted
	usSA.EscrowPolicy = EscrowAlwaysRecommended
	usSA.Restrictions = []string{"alcohol", "pork_products", "religious_items", "restricted_medications"}

	return []CorridorConfig{usEG, usAE, usSA}
}

// ── Lookup helpers ────────────────────────────────────────────────────────────

func GetCorridorConfig(origin, dest string) *CorridorConfig {
	return GetCorridorRepository().FindByRoute(origin, dest)
}

func IsCorridorSupported(origin, dest string) bool {
	return GetCorridorConfig(origin, dest) != nil
}

func GetValueBandMultiplier(c *CorridorConfig, valueUSD float64) ValueBand {
	if c == nil {
		return ValueBand{Multiplier: 1.0, Label: "Default"}
	}
	for _, b := range c.Risk.ValueBands {
		if valueUSD >= b.MinValue && valueUSD < b.MaxValue {
			return b
		}
	}
	if len(c.Risk.ValueBands) > 0 {
		return c.Risk.ValueBands[len(c.Risk.ValueBands)-1]
	}
	return ValueBand{Multiplier: 1.0}
}

func GetDeliveryWindowMultiplier(c *CorridorConfig, days int) DeliveryWindow {
	if c == nil {
		return DeliveryWindow{Multiplier: 1.0, Label: "Default"}
	}
	for _, w := range c.Risk.DeliveryWindows {
		if days >= w.MinDays && days <= w.MaxDays {
			return w
		}
	}
	if len(c.Risk.DeliveryWindows) > 0 {
		return c.Risk.DeliveryWindows[len(c.Risk.DeliveryWindows)-1]
	}
	return DeliveryWindow{Multiplier: 1.0}
}

func IsRestricted(c *CorridorConfig, itemCategory string) bool {
	if c == nil {
		return false
	}
	cat := strings.ToLower(strings.TrimSpace(itemCategory))
	for _, r := range c.Restrictions {
		if strings.EqualFold(r, cat) {
			return true
		}
	}
	return false
}

// ── HTTP handlers ─────────────────────────────────────────────────────────────

func (h *Handler) ListCorridors(c *gin.Context) {
	repo := GetCorridorRepository()
	configs := repo.FindAll()

	type corridorSummary struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		Origin       string   `json:"origin"`
		Destinations []string `json:"destinations"`
		Enabled      bool     `json:"enabled"`
		EscrowPolicy string   `json:"escrow_policy"`
	}
	summaries := make([]corridorSummary, 0, len(configs))
	for _, cc := range configs {
		summaries = append(summaries, corridorSummary{
			ID:           cc.ID,
			Name:         cc.Name,
			Origin:       cc.Origin,
			Destinations: cc.Destinations,
			Enabled:      cc.Enabled,
			EscrowPolicy: string(cc.EscrowPolicy),
		})
	}
	response.OK(c, summaries)
}

func (h *Handler) GetCorridor(c *gin.Context) {
	origin := c.Param("origin")
	dest := c.Param("dest")
	cfg := GetCorridorConfig(origin, dest)
	if cfg == nil {
		response.NotFound(c, "corridor")
		return
	}
	response.OK(c, cfg)
}
