package reslab

import (
	"time"

	"github.com/geocore-next/backend/internal/stress"
)

// ExperimentType identifies the class of resilience experiment.
type ExperimentType string

const (
	ExperimentBaseline      ExperimentType = "baseline_load_test"
	ExperimentDBPressure    ExperimentType = "chaos_db_pressure"
	ExperimentKafkaOverload ExperimentType = "kafka_overload"
	ExperimentRedisStorm    ExperimentType = "redis_eviction_storm"
	ExperimentMixedFailure  ExperimentType = "mixed_failure_scenario"
)

// Experiment wraps a stress scenario with lab metadata.
type Experiment struct {
	ID          string         `json:"id"`
	Type        ExperimentType `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Scenario    stress.Scenario `json:"scenario"`
}

// RunResult is the complete outcome of one experiment execution.
type RunResult struct {
	ID           string                 `json:"id"`
	ExperimentID string                 `json:"experiment_id"`
	RunAt        time.Time              `json:"run_at"`
	Score        ExperimentScore        `json:"score"`
	Metrics      stress.MetricsSummary  `json:"metrics"`
	Validation   stress.ValidationResult `json:"validation"`
	Suggestions  []Suggestion           `json:"suggestions"`
	Report       stress.StressReport    `json:"report"`
}
