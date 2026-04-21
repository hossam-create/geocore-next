package chaos

import (
	"context"
	"errors"
	"math/rand"
	"time"

	chaosstate "github.com/geocore-next/backend/pkg/chaos"
)

// InjectDBLatency sleeps for the given duration to simulate slow DB queries.
func InjectDBLatency(ms int) {
	if ms > 0 {
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
}

// InjectKafkaFailure randomly returns an error based on the error rate (0-100).
func InjectKafkaFailure(errRate int) error {
	if errRate <= 0 {
		return nil
	}
	if rand.Intn(100) < errRate {
		return errors.New("chaos: kafka injected failure")
	}
	return nil
}

// InjectAPILatency sleeps to simulate API latency.
func InjectAPILatency(ms int) {
	if ms > 0 {
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
}

// InjectRedisFailure returns an error simulating Redis unavailability.
func InjectRedisFailure(errRate int) error {
	if errRate <= 0 {
		return nil
	}
	if rand.Intn(100) < errRate {
		return errors.New("chaos: redis injected failure")
	}
	return nil
}

// ShouldInjectDB checks both the engine rate and the deterministic chaos state.
func ShouldInjectDB(ctx context.Context, engine *ChaosEngine) bool {
	if chaosstate.DBLatency() > 0 {
		return true
	}
	if engine != nil && engine.ShouldInject(ctx, "db_fail") {
		return true
	}
	return false
}

// ShouldInjectKafka checks engine rate and deterministic state.
func ShouldInjectKafka(ctx context.Context, engine *ChaosEngine) bool {
	if chaosstate.IsKafkaDown() {
		return true
	}
	if engine != nil && engine.ShouldInject(ctx, "kafka_fail") {
		return true
	}
	return false
}
