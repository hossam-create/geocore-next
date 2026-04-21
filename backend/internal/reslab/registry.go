package reslab

import "github.com/geocore-next/backend/internal/stress"

// BuiltinExperiments maps experiment IDs to their definitions.
// Each experiment targets a different failure mode and stress profile.
var BuiltinExperiments = map[string]Experiment{
	string(ExperimentBaseline): {
		ID:          string(ExperimentBaseline),
		Type:        ExperimentBaseline,
		Name:        "Baseline Load Test",
		Description: "Clean load test with zero chaos — establishes the performance baseline for all other experiments",
		Scenario:    stress.BuiltinScenarios["black_friday"],
	},
	string(ExperimentDBPressure): {
		ID:          string(ExperimentDBPressure),
		Type:        ExperimentDBPressure,
		Name:        "DB Pressure Test",
		Description: "Heavy concurrent DB reads to expose connection pool limits and slow query bottlenecks",
		Scenario:    stress.BuiltinScenarios["db_collapse"],
	},
	string(ExperimentKafkaOverload): {
		ID:          string(ExperimentKafkaOverload),
		Type:        ExperimentKafkaOverload,
		Name:        "Kafka Overload Test",
		Description: "Burst event production to expose consumer lag, DLQ growth, and retry storm conditions",
		Scenario:    stress.BuiltinScenarios["kafka_breakdown"],
	},
	string(ExperimentRedisStorm): {
		ID:          string(ExperimentRedisStorm),
		Type:        ExperimentRedisStorm,
		Name:        "Redis Eviction Storm",
		Description: "Cache miss explosion forcing full DB fallback — reveals DB capacity without cache",
		Scenario:    stress.BuiltinScenarios["redis_storm"],
	},
	string(ExperimentMixedFailure): {
		ID:          string(ExperimentMixedFailure),
		Type:        ExperimentMixedFailure,
		Name:        "Mixed Failure Scenario",
		Description: "Full cascading failure chain: Redis → DB overload → Kafka lag → API degradation → AIOps response",
		Scenario:    stress.BuiltinScenarios["cascading_failure"],
	},
}
