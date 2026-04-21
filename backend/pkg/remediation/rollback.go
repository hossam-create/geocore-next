package remediation

import (
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	// RollbackThresholds — trigger auto-rollback if exceeded.
	MaxErrorRate  = 0.02 // 2% error rate
	MaxP95Latency = 800 * time.Millisecond
	MaxKafkaLag   = 5000 // consumer lag
)

var (
	rollbackMu   sync.Mutex
	lastRollback time.Time
	cooldown     = 5 * time.Minute // prevent rollback flapping
)

// ShouldRollback evaluates deployment health and returns true if rollback is needed.
func ShouldRollback(errorRate float64, p95 time.Duration) bool {
	if errorRate > MaxErrorRate {
		slog.Error("rollback: error rate exceeded threshold",
			"rate", errorRate, "threshold", MaxErrorRate)
		return true
	}
	if p95 > MaxP95Latency {
		slog.Error("rollback: p95 latency exceeded threshold",
			"p95", p95, "threshold", MaxP95Latency)
		return true
	}
	return false
}

// ShouldRollbackKafka checks Kafka consumer lag as a rollback signal.
func ShouldRollbackKafka(lag int64) bool {
	if lag > MaxKafkaLag {
		slog.Error("rollback: Kafka consumer lag exceeded threshold",
			"lag", lag, "threshold", MaxKafkaLag)
		return true
	}
	return false
}

// SignalRollback triggers an immediate kubectl rollout undo for the affected deployment.
// It writes a marker file AND executes kubectl in K8s environments.
// Cooldown of 5 minutes prevents rollback flapping.
func SignalRollback(reason string) {
	rollbackMu.Lock()
	defer rollbackMu.Unlock()

	if time.Since(lastRollback) < cooldown {
		slog.Warn("rollback: cooldown active — skipping", "reason", reason, "since_last", time.Since(lastRollback))
		return
	}
	lastRollback = time.Now()

	slog.Error("rollback: signaling deployment rollback", "reason", reason)

	// Write marker file for CI/CD pipeline detection
	marker := "/tmp/geocore-rollback"
	_ = os.WriteFile(marker, []byte(reason+"\n"+time.Now().Format(time.RFC3339)), 0644)

	// Execute kubectl rollback for all GeoCore deployments
	namespace := os.Getenv("K8S_NAMESPACE")
	if namespace == "" {
		namespace = "geocore-prod"
	}
	deployments := strings.Fields(os.Getenv("ROLLBACK_DEPLOYMENTS"))
	if len(deployments) == 0 {
		deployments = []string{"geocore-backend", "geocore-worker", "geocore-fraud-engine", "geocore-saas-cp"}
	}
	for _, dep := range deployments {
		go func(d string) {
			cmd := exec.Command("kubectl", "rollout", "undo", "deployment/"+d, "-n", namespace)
			if out, err := cmd.CombinedOutput(); err != nil {
				slog.Error("rollback: kubectl failed", "deployment", d, "error", err, "output", string(out))
			} else {
				slog.Info("rollback: kubectl succeeded", "deployment", d)
			}
		}(dep)
	}
}

// DeployHealth holds the current deployment health metrics.
type DeployHealth struct {
	ErrorRate float64       `json:"error_rate"`
	P95       time.Duration `json:"p95_ms"`
	KafkaLag  int64         `json:"kafka_lag"`
	Healthy   bool          `json:"healthy"`
	Reason    string        `json:"reason,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// CheckDeployHealth evaluates all rollback signals at once.
func CheckDeployHealth(errorRate float64, p95 time.Duration, kafkaLag int64) DeployHealth {
	h := DeployHealth{
		ErrorRate: errorRate,
		P95:       p95,
		KafkaLag:  kafkaLag,
		Healthy:   true,
		Timestamp: time.Now(),
	}

	if errorRate > MaxErrorRate {
		h.Healthy = false
		h.Reason = "error_rate_exceeded"
	}
	if p95 > MaxP95Latency {
		h.Healthy = false
		h.Reason = "p95_latency_exceeded"
	}
	if kafkaLag > MaxKafkaLag {
		h.Healthy = false
		h.Reason = "kafka_lag_exceeded"
	}

	return h
}
