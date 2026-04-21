package intelligence

// CostModel estimates infrastructure cost impact of optimization proposals.
type CostModel struct {
	CostPerPodHour   float64 // cost per pod per hour
	CostPerCPUHour   float64 // cost per vCPU hour
	CostPerGBHour    float64 // cost per GB memory hour
	MonthlyBudgetCap float64 // maximum monthly spend allowed
	CurrentSpend     float64 // current monthly spend
}

// NewCostModel creates a cost model with default pricing.
func NewCostModel() *CostModel {
	return &CostModel{
		CostPerPodHour:   0.10,  // ~$0.10/pod/hour
		CostPerCPUHour:   0.05,  // ~$0.05/vCPU/hour
		CostPerGBHour:    0.01,  // ~$0.01/GB/hour
		MonthlyBudgetCap: 5000.0,
		CurrentSpend:     2000.0,
	}
}

// EstimatePodCostChange calculates monthly cost delta from pod count changes.
func (c *CostModel) EstimatePodCostChange(currentPods, proposedPods int) float64 {
	delta := float64(proposedPods-currentPods) * c.CostPerPodHour * 24 * 30
	return delta
}

// EstimateScaleUpCost returns the monthly cost increase for scaling up.
func (c *CostModel) EstimateScaleUpCost(additionalPods int) float64 {
	return float64(additionalPods) * c.CostPerPodHour * 24 * 30
}

// EstimateScaleDownSavings returns the monthly savings from scaling down.
func (c *CostModel) EstimateScaleDownSavings(removedPods int) float64 {
	return float64(removedPods) * c.CostPerPodHour * 24 * 30
}

// IsWithinBudget checks if a proposed cost change stays within budget.
func (c *CostModel) IsWithinBudget(projectedMonthlyCost float64) bool {
	return projectedMonthlyCost <= c.MonthlyBudgetCap
}

// ProjectedMonthlySpend estimates total monthly spend after a change.
func (c *CostModel) ProjectedMonthlySpend(delta float64) float64 {
	return c.CurrentSpend + delta
}

// BudgetUtilization returns current budget utilization as a percentage.
func (c *CostModel) BudgetUtilization() float64 {
	if c.MonthlyBudgetCap == 0 {
		return 0
	}
	return (c.CurrentSpend / c.MonthlyBudgetCap) * 100
}
