package intelligence

import (
	"log/slog"
)

// FraudAI detects abnormal behavior and dynamically adjusts fraud thresholds.
type FraudAI struct {
	CurrentThreshold float64 `json:"current_threshold"` // 0-100
	CurrentSensitivity string `json:"current_sensitivity"` // low, medium, high, aggressive
	FraudRate        float64 `json:"fraud_rate"`
	FalsePositiveRate float64 `json:"false_positive_rate"`
}

// NewFraudAI creates a fraud intelligence engine.
func NewFraudAI() *FraudAI {
	return &FraudAI{
		CurrentThreshold:  70,
		CurrentSensitivity: "medium",
	}
}

// Analyze evaluates fraud metrics and returns threshold adjustment suggestions.
func (f *FraudAI) Analyze(fraudRate, falsePositiveRate float64) []FraudSuggestion {
	f.FraudRate = fraudRate
	f.FalsePositiveRate = falsePositiveRate

	var suggestions []FraudSuggestion

	// Fraud spike → increase sensitivity
	if fraudRate > 5.0 {
		suggestions = append(suggestions, FraudSuggestion{
			Action:    "increase_sensitivity",
			NewLevel:  f.nextHigherSensitivity(),
			Reason:    "Fraud rate spike detected",
			Threshold: f.thresholdForLevel(f.nextHigherSensitivity()),
		})
	}

	// High false positives → decrease sensitivity
	if falsePositiveRate > 30.0 {
		suggestions = append(suggestions, FraudSuggestion{
			Action:    "decrease_sensitivity",
			NewLevel:  f.nextLowerSensitivity(),
			Reason:    "False positive rate too high",
			Threshold: f.thresholdForLevel(f.nextLowerSensitivity()),
		})
	}

	if len(suggestions) > 0 {
		slog.Info("cloudos: fraud AI suggestions", "count", len(suggestions))
	}

	return suggestions
}

func (f *FraudAI) nextHigherSensitivity() string {
	switch f.CurrentSensitivity {
	case "low":
		return "medium"
	case "medium":
		return "high"
	case "high":
		return "aggressive"
	default:
		return "aggressive"
	}
}

func (f *FraudAI) nextLowerSensitivity() string {
	switch f.CurrentSensitivity {
	case "aggressive":
		return "high"
	case "high":
		return "medium"
	case "medium":
		return "low"
	default:
		return "low"
	}
}

func (f *FraudAI) thresholdForLevel(level string) float64 {
	switch level {
	case "low":
		return 90
	case "medium":
		return 70
	case "high":
		return 50
	case "aggressive":
		return 30
	default:
		return 70
	}
}

// FraudSuggestion is a fraud threshold adjustment suggestion.
type FraudSuggestion struct {
	Action    string  `json:"action"`
	NewLevel  string  `json:"new_level"`
	Reason    string  `json:"reason"`
	Threshold float64 `json:"threshold"`
}
