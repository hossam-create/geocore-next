package simulator

// ImpactModel predicts the impact of infrastructure changes.
type ImpactModel struct {
	latencyPerPod float64 // ms latency reduction per additional pod
	costPerPod    float64 // hourly cost per pod
}

// NewImpactModel creates an impact model with default parameters.
func NewImpactModel() *ImpactModel {
	return &ImpactModel{
		latencyPerPod: 15,  // ~15ms reduction per pod
		costPerPod:    0.10, // ~$0.10/hour per pod
	}
}

// PredictScaleUpImpact estimates the impact of adding pods.
func (m *ImpactModel) PredictScaleUpImpact(currentPods, additionalPods int) Impact {
	return Impact{
		Safe:         additionalPods <= 4, // max 4 pods at once
		LatencyDelta: -float64(additionalPods) * m.latencyPerPod,
		CostDelta:    float64(additionalPods) * m.costPerPod,
		ErrorDelta:   -0.1 * float64(additionalPods),
	}
}

// PredictScaleDownImpact estimates the impact of removing pods.
func (m *ImpactModel) PredictScaleDownImpact(currentPods, removedPods int) Impact {
	remaining := currentPods - removedPods
	safe := remaining >= 2 // never go below 2

	return Impact{
		Safe:         safe,
		LatencyDelta: float64(removedPods) * m.latencyPerPod,
		CostDelta:    -float64(removedPods) * m.costPerPod,
		ErrorDelta:   0.05 * float64(removedPods),
	}
}
