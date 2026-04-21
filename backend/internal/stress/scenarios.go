package stress

import "time"

// ChaosType names the class of failure to inject during a scenario.
type ChaosType string

const (
	ChaosNone           ChaosType = "none"
	ChaosDBPressure     ChaosType = "db_pressure"
	ChaosKafkaBreakdown ChaosType = "kafka_breakdown"
	ChaosRedisEviction  ChaosType = "redis_eviction"
	ChaosLatencySpike   ChaosType = "latency_spike"
	ChaosCascading      ChaosType = "cascading_failure"
)

// LoadRamp describes one phase of the load ramp: how many concurrent users for how long.
type LoadRamp struct {
	Users    int           `json:"users"`
	Duration time.Duration `json:"duration_ms"`
}

// Scenario describes a full stress test: load shape + chaos types.
type Scenario struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	LoadRamp    []LoadRamp  `json:"load_ramp"`
	ChaosTypes  []ChaosType `json:"chaos_types"`
}

// BuiltinScenarios are ready-to-run production simulation scenarios.
var BuiltinScenarios = map[string]Scenario{
	"black_friday": {
		ID:          "black_friday",
		Name:        "Black Friday Spike",
		Description: "Viral traffic spike: 50 → 200 → 500 → 1000 concurrent users over 60s",
		LoadRamp: []LoadRamp{
			{Users: 50, Duration: 15 * time.Second},
			{Users: 200, Duration: 15 * time.Second},
			{Users: 500, Duration: 15 * time.Second},
			{Users: 1000, Duration: 15 * time.Second},
		},
		ChaosTypes: []ChaosType{ChaosDBPressure},
	},
	"kafka_breakdown": {
		ID:          "kafka_breakdown",
		Name:        "Kafka Breakdown",
		Description: "Consumer slowdown + DLQ overflow + retry storms under sustained load",
		LoadRamp: []LoadRamp{
			{Users: 100, Duration: 20 * time.Second},
			{Users: 300, Duration: 20 * time.Second},
		},
		ChaosTypes: []ChaosType{ChaosKafkaBreakdown},
	},
	"db_collapse": {
		ID:          "db_collapse",
		Name:        "DB Pressure Collapse",
		Description: "Slow queries, lock contention, and connection exhaustion under write load",
		LoadRamp: []LoadRamp{
			{Users: 300, Duration: 30 * time.Second},
		},
		ChaosTypes: []ChaosType{ChaosDBPressure},
	},
	"redis_storm": {
		ID:          "redis_storm",
		Name:        "Redis Eviction Storm",
		Description: "Cache miss explosion forcing full DB fallback",
		LoadRamp: []LoadRamp{
			{Users: 200, Duration: 15 * time.Second},
			{Users: 400, Duration: 15 * time.Second},
		},
		ChaosTypes: []ChaosType{ChaosRedisEviction},
	},
	"cascading_failure": {
		ID:          "cascading_failure",
		Name:        "Cascading Failure",
		Description: "Redis → cache miss → DB overload → Kafka lag → API slow → AIOps triggered",
		LoadRamp: []LoadRamp{
			{Users: 100, Duration: 15 * time.Second},
			{Users: 400, Duration: 15 * time.Second},
			{Users: 800, Duration: 15 * time.Second},
		},
		ChaosTypes: []ChaosType{ChaosRedisEviction, ChaosDBPressure, ChaosKafkaBreakdown},
	},
}
