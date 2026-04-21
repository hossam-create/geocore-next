package resources

// FraudResource represents the fraud detection system as a managed resource.
type FraudResource struct {
	RiskModelVersion string  `json:"risk_model_version"`
	Threshold        float64 `json:"threshold"` // fraud score threshold (0-100)
	Sensitivity      string  `json:"sensitivity"` // low, medium, high, aggressive
	FalsePositiveRate float64 `json:"false_positive_rate"`
	FraudRate        float64 `json:"fraud_rate"`
	BlockedCount     int64   `json:"blocked_count"`
}

// DefaultFraudResource returns production defaults.
func DefaultFraudResource() FraudResource {
	return FraudResource{
		RiskModelVersion: "v1",
		Threshold:        70,
		Sensitivity:      "medium",
	}
}

// IsHealthy returns true if fraud system is operating normally.
func (f *FraudResource) IsHealthy() bool {
	return f.FalsePositiveRate < 30 && f.Threshold > 0
}

// NeedsSensitivityIncrease returns true if fraud rate is spiking.
func (f *FraudResource) NeedsSensitivityIncrease() bool {
	return f.FraudRate > 5.0
}

// NeedsSensitivityDecrease returns true if false positives are too high.
func (f *FraudResource) NeedsSensitivityDecrease() bool {
	return f.FalsePositiveRate > 30
}
