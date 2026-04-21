package intelligence

import (
	"log/slog"
	"math"
)

// CostAI detects overprovisioning, predicts monthly bill, and suggests scaling changes.
type CostAI struct {
	CostPerPodHour   float64
	MonthlyBudgetCap float64
	CurrentSpend     float64
	PodCount         int
}

// NewCostAI creates a cost intelligence engine.
func NewCostAI() *CostAI {
	return &CostAI{
		CostPerPodHour:   0.10,
		MonthlyBudgetCap: 5000,
		CurrentSpend:     2000,
		PodCount:         4,
	}
}

// Analyze detects cost inefficiencies and returns proposals.
func (c *CostAI) Analyze(cpuUtilization float64, podCount int) []CostSuggestion {
	var suggestions []CostSuggestion

	projectedMonthly := c.ProjectMonthlySpend(podCount)

	// Overprovisioning detection
	if cpuUtilization < 30 && podCount > 2 {
		suggestions = append(suggestions, CostSuggestion{
			Type:        "overprovisioned",
			Action:      "scale_down",
			SavingsPct:  20,
			Reason:      "CPU < 30% with excess pods",
			ProjectedCost: projectedMonthly,
		})
	}

	// Budget breach prediction
	if projectedMonthly > c.MonthlyBudgetCap*0.9 {
		suggestions = append(suggestions, CostSuggestion{
			Type:        "budget_warning",
			Action:      "reduce_costs",
			SavingsPct:  15,
			Reason:      "Projected spend approaching budget cap",
			ProjectedCost: projectedMonthly,
		})
	}

	if len(suggestions) > 0 {
		slog.Info("cloudos: cost AI suggestions", "count", len(suggestions))
	}

	return suggestions
}

// ProjectMonthlySpend estimates monthly cost for a given pod count.
func (c *CostAI) ProjectMonthlySpend(podCount int) float64 {
	return float64(podCount) * c.CostPerPodHour * 24 * 30
}

// BudgetUtilization returns current budget utilization as a percentage.
func (c *CostAI) BudgetUtilization() float64 {
	if c.MonthlyBudgetCap == 0 {
		return 0
	}
	return math.Round((c.CurrentSpend/c.MonthlyBudgetCap)*10000) / 100
}

// CostSuggestion is a cost optimization suggestion.
type CostSuggestion struct {
	Type          string  `json:"type"`
	Action        string  `json:"action"`
	SavingsPct    float64 `json:"savings_pct"`
	Reason        string  `json:"reason"`
	ProjectedCost float64 `json:"projected_cost"`
}
