package intelligence

import (
	"log/slog"
	"math"
)

// TrafficAI predicts load spikes, pre-scales services, and moves traffic across regions.
type TrafficAI struct {
	history     []TrafficSample
	PrimaryRPS  float64
	CurrentRPS  float64
}

// TrafficSample is a point-in-time traffic measurement.
type TrafficSample struct {
	RPS      float64
	Hour     int // 0-23
	DayOfWeek int // 0=Sunday, 6=Saturday
}

// NewTrafficAI creates a traffic intelligence engine.
func NewTrafficAI() *TrafficAI {
	return &TrafficAI{}
}

// Analyze predicts upcoming traffic patterns and returns scaling suggestions.
func (t *TrafficAI) Analyze(currentRPS, cpuPct float64, hour, dayOfWeek int) []TrafficSuggestion {
	t.CurrentRPS = currentRPS
	t.AddSample(TrafficSample{RPS: currentRPS, Hour: hour, DayOfWeek: dayOfWeek})

	var suggestions []TrafficSuggestion

	// Predict next-hour traffic
	predictedRPS := t.PredictNextHour(hour, dayOfWeek)

	// Pre-scale if predicted traffic will exceed capacity
	currentCapacity := currentRPS / (cpuPct / 100)
	if currentCapacity == 0 {
		currentCapacity = 100
	}

	if predictedRPS > currentCapacity*0.8 {
		suggestions = append(suggestions, TrafficSuggestion{
			Action:       "pre_scale",
			Target:       "api",
			AdditionalPods: int(math.Ceil((predictedRPS - currentCapacity) / 50)),
			Reason:       "Predicted traffic spike in next hour",
			PredictedRPS: predictedRPS,
		})
	}

	// Suggest region shift if primary is overloaded
	if cpuPct > 80 {
		suggestions = append(suggestions, TrafficSuggestion{
			Action:       "shift_traffic",
			Target:       "region",
			AdditionalPods: 0,
			Reason:       "Primary region overloaded, shift traffic to secondary",
			PredictedRPS: predictedRPS,
		})
	}

	if len(suggestions) > 0 {
		slog.Info("cloudos: traffic AI suggestions", "count", len(suggestions))
	}

	return suggestions
}

// PredictNextHour predicts RPS for the next hour based on historical patterns.
func (t *TrafficAI) PredictNextHour(hour, dayOfWeek int) float64 {
	if len(t.history) < 10 {
		return t.CurrentRPS * 1.1 // default: 10% increase
	}

	// Average RPS for this hour + day from history
	var sum float64
	var count float64
	for _, s := range t.history {
		if s.Hour == hour && s.DayOfWeek == dayOfWeek {
			sum += s.RPS
			count++
		}
	}

	if count > 0 {
		return sum / count
	}
	return t.CurrentRPS * 1.1
}

// AddSample adds a traffic sample to the model.
func (t *TrafficAI) AddSample(s TrafficSample) {
	t.history = append(t.history, s)
	if len(t.history) > 1000 {
		t.history = t.history[1:]
	}
}

// TrafficSuggestion is a traffic optimization suggestion.
type TrafficSuggestion struct {
	Action        string  `json:"action"`
	Target        string  `json:"target"`
	AdditionalPods int    `json:"additional_pods"`
	Reason        string  `json:"reason"`
	PredictedRPS  float64 `json:"predicted_rps"`
}
