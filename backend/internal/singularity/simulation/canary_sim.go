package simulation

import (
	"log/slog"
	"time"
)

// CanarySimulation models a gradual rollout: 1% → 10% → 50% → 100%.
type CanarySimulation struct {
	Steps     []CanaryStep
	Current   int
	StartTime time.Time
}

// CanaryStep is a single stage in the canary rollout.
type CanaryStep struct {
	TrafficPercent int     `json:"traffic_percent"`
	Duration       time.Duration `json:"duration"`
	PassThreshold  float64 `json:"pass_threshold"` // max error rate allowed
	Passed         bool    `json:"passed"`
}

// NewCanarySimulation creates a standard 4-stage canary rollout.
func NewCanarySimulation() *CanarySimulation {
	return &CanarySimulation{
		Steps: []CanaryStep{
			{TrafficPercent: 1, Duration: 5 * time.Minute, PassThreshold: 2.0},
			{TrafficPercent: 10, Duration: 10 * time.Minute, PassThreshold: 1.5},
			{TrafficPercent: 50, Duration: 15 * time.Minute, PassThreshold: 1.0},
			{TrafficPercent: 100, Duration: 0, PassThreshold: 0.5},
		},
		Current:   0,
		StartTime: time.Now().UTC(),
	}
}

// CurrentStep returns the current canary step.
func (c *CanarySimulation) CurrentStep() *CanaryStep {
	if c.Current >= len(c.Steps) {
		return nil
	}
	return &c.Steps[c.Current]
}

// Advance moves to the next canary step if the current one passed.
func (c *CanarySimulation) Advance(errorRate float64) bool {
	step := c.CurrentStep()
	if step == nil {
		return false
	}

	if errorRate <= step.PassThreshold {
		step.Passed = true
		c.Current++
		slog.Info("singularity: canary step passed",
			"traffic_pct", step.TrafficPercent,
			"error_rate", errorRate,
			"next_step", c.Current,
		)
		return true
	}

	// Canary failed — stop rollout
	step.Passed = false
	slog.Warn("singularity: canary step FAILED — halting rollout",
		"traffic_pct", step.TrafficPercent,
		"error_rate", errorRate,
		"threshold", step.PassThreshold,
	)
	return false
}

// IsComplete returns true if all canary steps passed.
func (c *CanarySimulation) IsComplete() bool {
	return c.Current >= len(c.Steps)
}

// ShouldRollback returns true if the canary should be rolled back.
func (c *CanarySimulation) ShouldRollback() bool {
	if c.Current >= len(c.Steps) {
		return false
	}
	return !c.Steps[c.Current].Passed && c.Current > 0
}

// RollbackTriggers defines conditions that trigger an immediate rollback.
type RollbackTriggers struct {
	P95LatencySpike float64 // ms above baseline
	ErrorRateAbove  float64 // percent
	KafkaLagAbove   int     // messages
	DBSaturation    float64 // connection pool %
}

// DefaultRollbackTriggers returns standard rollback trigger thresholds.
func DefaultRollbackTriggers() RollbackTriggers {
	return RollbackTriggers{
		P95LatencySpike: 200,
		ErrorRateAbove:  2.0,
		KafkaLagAbove:   10000,
		DBSaturation:    90.0,
	}
}

// ShouldTriggerRollback checks if any rollback condition is met.
func (r RollbackTriggers) ShouldTriggerRollback(p95, errorRate float64, kafkaLag int, dbSat float64) bool {
	return p95 > r.P95LatencySpike ||
		errorRate > r.ErrorRateAbove ||
		kafkaLag > r.KafkaLagAbove ||
		dbSat > r.DBSaturation
}
