package intelligence

import (
	"math"
)

// WorkloadModel predicts resource needs based on traffic patterns.
type WorkloadModel struct {
	history []WorkloadSample
}

// WorkloadSample is a point-in-time workload measurement.
type WorkloadSample struct {
	RPS    float64
	CPU    float64 // CPU utilization %
	Memory float64 // Memory utilization %
	Pods   int
}

// NewWorkloadModel creates a workload model.
func NewWorkloadModel() *WorkloadModel {
	return &WorkloadModel{}
}

// AddSample adds a workload sample to the model.
func (w *WorkloadModel) AddSample(s WorkloadSample) {
	w.history = append(w.history, s)
	if len(w.history) > 1000 {
		w.history = w.history[1:]
	}
}

// PredictCPU estimates CPU utilization for a given RPS.
func (w *WorkloadModel) PredictCPU(rps float64) float64 {
	if len(w.history) < 2 {
		return rps * 0.1 // default: 10% CPU per 100 RPS
	}

	// Linear regression: CPU = a*RPS + b
	a, b := linearRegression(w.history, func(s WorkloadSample) (float64, float64) {
		return s.RPS, s.CPU
	})

	predicted := a*rps + b
	return clamp(predicted, 0, 100)
}

// PredictMemory estimates memory utilization for a given RPS.
func (w *WorkloadModel) PredictMemory(rps float64) float64 {
	if len(w.history) < 2 {
		return 30.0 // default baseline
	}

	a, b := linearRegression(w.history, func(s WorkloadSample) (float64, float64) {
		return s.RPS, s.Memory
	})

	predicted := a*rps + b
	return clamp(predicted, 0, 100)
}

// RecommendedPods calculates the optimal pod count for a given RPS.
func (w *WorkloadModel) RecommendedPods(rps float64) int {
	cpu := w.PredictCPU(rps)
	// Target 70% CPU utilization per pod
	pods := int(math.Ceil(cpu / 70.0))
	if pods < 2 {
		return 2
	}
	if pods > 20 {
		return 20
	}
	return pods
}

func linearRegression(samples []WorkloadSample, extract func(WorkloadSample) (float64, float64)) (slope, intercept float64) {
	n := float64(len(samples))
	var sumX, sumY, sumXY, sumX2 float64

	for _, s := range samples {
		x, y := extract(s)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return 0, sumY / n
	}

	slope = (n*sumXY - sumX*sumY) / denom
	intercept = (sumY - slope*sumX) / n
	return slope, intercept
}

func clamp(v, min_, max_ float64) float64 {
	if v < min_ {
		return min_
	}
	if v > max_ {
		return max_
	}
	return v
}
