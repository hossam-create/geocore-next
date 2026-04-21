package billing

// PlanID identifies a subscription tier.
type PlanID string

const (
	PlanStarter    PlanID = "starter"    // $29/mo
	PlanPro        PlanID = "pro"        // $199/mo
	PlanEnterprise PlanID = "enterprise" // $2000+/mo
)

// Plan defines the feature set and quotas for a subscription tier.
type Plan struct {
	ID            PlanID `json:"id"`
	Name          string `json:"name"`
	MonthlyPrice  int    `json:"monthly_price_cents"` // USD cents
	MaxRequests   int64  `json:"max_requests_per_day"` // 0 = unlimited
	MaxAPIKeys    int    `json:"max_api_keys"`          // 0 = unlimited
	MaxServices   int    `json:"max_services"`          // 0 = unlimited
	AIOpsEnabled  bool   `json:"aiops_enabled"`
	ChaosEnabled  bool   `json:"chaos_enabled"`
	ResLabEnabled bool   `json:"reslab_enabled"`
	MultiRegion   bool   `json:"multi_region"`
	SupportSLA    string `json:"support_sla"`
}

// Catalog is the authoritative plan registry.
var Catalog = map[PlanID]Plan{
	PlanStarter: {
		ID:            PlanStarter,
		Name:          "Starter",
		MonthlyPrice:  2900,
		MaxRequests:   100_000,
		MaxAPIKeys:    3,
		MaxServices:   1,
		AIOpsEnabled:  false,
		ChaosEnabled:  false,
		ResLabEnabled: false,
		MultiRegion:   false,
		SupportSLA:    "best-effort",
	},
	PlanPro: {
		ID:            PlanPro,
		Name:          "Pro",
		MonthlyPrice:  19900,
		MaxRequests:   10_000_000,
		MaxAPIKeys:    20,
		MaxServices:   10,
		AIOpsEnabled:  true,
		ChaosEnabled:  true,
		ResLabEnabled: true,
		MultiRegion:   true,
		SupportSLA:    "99.9%",
	},
	PlanEnterprise: {
		ID:            PlanEnterprise,
		Name:          "Enterprise",
		MonthlyPrice:  200000,
		MaxRequests:   0,
		MaxAPIKeys:    0,
		MaxServices:   0,
		AIOpsEnabled:  true,
		ChaosEnabled:  true,
		ResLabEnabled: true,
		MultiRegion:   true,
		SupportSLA:    "99.99%",
	},
}

// Get returns the plan for the given ID, defaulting to Starter if unknown.
func Get(id PlanID) Plan {
	if p, ok := Catalog[id]; ok {
		return p
	}
	return Catalog[PlanStarter]
}
