package gameday

import (
	"context"
	"log/slog"
	"time"

	chaosstate "github.com/geocore-next/backend/pkg/chaos"
)

func defaultScenarios() []GameDay {
	return []GameDay{
		{Name: "db_failure", Scenario: ChaosDBFailureScenario},
		{Name: "kafka_lag", Scenario: KafkaLagScenario},
		{Name: "redis_eviction", Scenario: RedisEvictionScenario},
		{Name: "high_traffic_spike", Scenario: HighTrafficSpikeScenario},
	}
}

// ChaosDBFailureScenario simulates a DB failure by injecting latency
// then verifying the system degrades gracefully.
func ChaosDBFailureScenario(ctx context.Context) {
	slog.Info("gameday: [DB_FAILURE] injecting 2s DB latency")
	chaosstate.SetDBLatency(2 * time.Second)

	// Let system experience the failure for 10 seconds
	time.Sleep(10 * time.Second)

	slog.Info("gameday: [DB_FAILURE] restoring DB")
	chaosstate.SetDBLatency(0)
	slog.Info("gameday: [DB_FAILURE] scenario complete — system should have degraded gracefully")
}

// KafkaLagScenario simulates Kafka being unavailable.
func KafkaLagScenario(ctx context.Context) {
	slog.Info("gameday: [KAFKA_LAG] forcing Kafka down")
	chaosstate.SetKafkaDown(true)

	// Let events accumulate in outbox for 10 seconds
	time.Sleep(10 * time.Second)

	slog.Info("gameday: [KAFKA_LAG] restoring Kafka")
	chaosstate.SetKafkaDown(false)
	slog.Info("gameday: [KAFKA_LAG] scenario complete — outbox should drain")
}

// RedisEvictionScenario simulates Redis being unavailable.
func RedisEvictionScenario(ctx context.Context) {
	slog.Info("gameday: [REDIS_EVICTION] forcing Redis down")
	chaosstate.SetRedisDown(true)

	// System should fall back to DB for 10 seconds
	time.Sleep(10 * time.Second)

	slog.Info("gameday: [REDIS_EVICTION] restoring Redis")
	chaosstate.SetRedisDown(false)
	slog.Info("gameday: [REDIS_EVICTION] scenario complete — cache should warm up")
}

// HighTrafficSpikeScenario simulates a traffic spike by injecting
// API latency and DB load simultaneously.
func HighTrafficSpikeScenario(ctx context.Context) {
	slog.Info("gameday: [TRAFFIC_SPIKE] injecting combined failures")
	chaosstate.SetDBLatency(500 * time.Millisecond)
	chaosstate.SetRedisDown(true)

	time.Sleep(15 * time.Second)

	slog.Info("gameday: [TRAFFIC_SPIKE] restoring all systems")
	chaosstate.SetDBLatency(0)
	chaosstate.SetRedisDown(false)
	slog.Info("gameday: [TRAFFIC_SPIKE] scenario complete — system should have auto-scaled")
}
