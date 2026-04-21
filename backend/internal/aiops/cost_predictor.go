package aiops

// CostModel defines the cost parameters for infrastructure.
type CostModel struct {
	CostPerRPS      float64 // cost per request per second
	BaseInfraCost   float64 // fixed monthly infrastructure cost
	CostPerCPUHour  float64 // cost per CPU hour
	CostPerGBMemory float64 // cost per GB memory hour
}

// DefaultCostModel returns a standard cost model.
func DefaultCostModel() CostModel {
	return CostModel{
		CostPerRPS:      0.002,
		BaseInfraCost:   500.0,  // $500/mo base
		CostPerCPUHour:  0.05,   // $0.05/CPU hour
		CostPerGBMemory: 0.01,   // $0.01/GB hour
	}
}

// PredictCost estimates monthly cost based on current RPS and infra usage.
func PredictCost(rps int, infraHours float64) float64 {
	model := DefaultCostModel()
	return model.PredictMonthly(rps, infraHours)
}

// PredictMonthly calculates the full monthly cost prediction.
func (m CostModel) PredictMonthly(rps int, infraHours float64) float64 {
	rpsCost := float64(rps) * m.CostPerRPS * 30 * 24 * 3600 // monthly
	infraCost := infraHours * m.CostPerCPUHour
	return m.BaseInfraCost + rpsCost + infraCost
}

// PredictDaily estimates daily cost.
func (m CostModel) PredictDaily(rps int, infraHours float64) float64 {
	return m.PredictMonthly(rps, infraHours) / 30
}

// BreakevenRPS calculates the minimum RPS needed to cover costs
// given a revenue-per-request model.
func (m CostModel) BreakevenRPS(revenuePerRequest float64) int {
	if revenuePerRequest <= 0 {
		return 0
	}
	// Monthly cost / (revenue per request * seconds in month)
	monthlyCost := m.BaseInfraCost
	return int(monthlyCost / (revenuePerRequest * 30 * 24 * 3600))
}
