package intelligence

// LatencyModel predicts latency impact of infrastructure changes.
type LatencyModel struct {
	history []LatencySample
}

// LatencySample is a point-in-time latency measurement.
type LatencySample struct {
	Pods      int
	RPS       float64
	P50Ms     float64
	P95Ms     float64
	P99Ms     float64
}

// NewLatencyModel creates a latency prediction model.
func NewLatencyModel() *LatencyModel {
	return &LatencyModel{}
}

// AddSample adds a latency sample to the model.
func (l *LatencyModel) AddSample(s LatencySample) {
	l.history = append(l.history, s)
	if len(l.history) > 1000 {
		l.history = l.history[1:]
	}
}

// PredictP95 estimates P95 latency for a given pod count and RPS.
func (l *LatencyModel) PredictP95(pods int, rps float64) float64 {
	if len(l.history) < 2 {
		// Default: more pods = lower latency, more RPS = higher latency
		baseLatency := 200.0
		loadFactor := rps / float64(pods) / 50.0 // 50 RPS per pod is baseline
		return baseLatency * loadFactor
	}

	// Find closest historical samples
	var closest *LatencySample
	minDist := 1e18
	for _, s := range l.history {
		dist := abs(float64(s.Pods)-float64(pods)) + abs(s.RPS-rps)
		if dist < minDist {
			minDist = dist
			closest = &s
		}
	}

	if closest != nil {
		// Scale by pod ratio
		podRatio := float64(closest.Pods) / float64(pods)
		return closest.P95Ms / podRatio
	}

	return 200.0
}

// EstimateLatencyDelta predicts the latency change from scaling.
func (l *LatencyModel) EstimateLatencyDelta(currentPods, proposedPods int, rps float64) float64 {
	currentP95 := l.PredictP95(currentPods, rps)
	proposedP95 := l.PredictP95(proposedPods, rps)
	return proposedP95 - currentP95 // negative = improvement
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
