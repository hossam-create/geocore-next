package aiops

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// Rule defines an anomaly detection rule evaluated against a Prometheus metric.
type Rule struct {
	Name      string
	Query     string
	Baseline  float64  // expected normal value
	Threshold float64  // trigger when value > Baseline * Threshold (or > Threshold when Baseline == 0)
	Severity  Severity
	Service   string
	Title     string
}

var defaultRules = []Rule{
	{
		Name:      "api_error_rate",
		Query:     `sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))`,
		Baseline:  0.01,
		Threshold: 3.0, // fires at >3%
		Severity:  SeverityP0,
		Service:   "api",
		Title:     "High 5xx error rate (>3%)",
	},
	{
		Name:      "kafka_consumer_lag",
		Query:     `max(kafka_consumer_group_lag)`,
		Baseline:  100,
		Threshold: 50.0, // fires at >5000
		Severity:  SeverityP0,
		Service:   "kafka",
		Title:     "Kafka consumer lag critical (>5000)",
	},
	{
		Name:      "db_pool_saturation",
		Query:     `db_connections_in_use / db_connections_open`,
		Baseline:  0.5,
		Threshold: 1.7, // fires at >85%
		Severity:  SeverityP0,
		Service:   "database",
		Title:     "DB connection pool near exhaustion (>85%)",
	},
	{
		Name:      "wallet_invariant_violation",
		Query:     `wallet_invariant_violation_total`,
		Baseline:  0,
		Threshold: 0.5, // fires at any value > 0
		Severity:  SeverityP0,
		Service:   "wallet",
		Title:     "Wallet invariant violation detected",
	},
	{
		Name:      "api_latency_p95",
		Query:     `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))`,
		Baseline:  0.3,
		Threshold: 2.5, // fires at >750ms
		Severity:  SeverityP1,
		Service:   "api",
		Title:     "API p95 latency spike (>750ms)",
	},
	{
		Name:      "redis_memory_high",
		Query:     `redis_memory_used_bytes / redis_memory_max_bytes`,
		Baseline:  0.5,
		Threshold: 1.8, // fires at >90%
		Severity:  SeverityP1,
		Service:   "redis",
		Title:     "Redis memory utilization high (>90%)",
	},
	{
		Name:      "kafka_dlq_growing",
		Query:     `sum(rate(kafka_messages_in_total{topic=~".*\\.dlq"}[5m]))`,
		Baseline:  0,
		Threshold: 0.5, // fires at any DLQ ingestion
		Severity:  SeverityP1,
		Service:   "kafka",
		Title:     "Dead Letter Queue receiving messages",
	},
	{
		Name:      "outbox_backlog",
		Query:     `kafka_outbox_pending`,
		Baseline:  10,
		Threshold: 10.0, // fires at >100
		Severity:  SeverityP2,
		Service:   "outbox",
		Title:     "Kafka outbox backlog growing (>100)",
	},
}

type Detector struct {
	prometheusURL string
	client        *http.Client
	rules         []Rule
}

func NewDetector(rules []Rule) *Detector {
	if len(rules) == 0 {
		rules = defaultRules
	}
	return &Detector{
		prometheusURL: os.Getenv("PROMETHEUS_URL"),
		client:        &http.Client{Timeout: 5 * time.Second},
		rules:         rules,
	}
}

type promQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// queryMetric executes a Prometheus instant query and returns the scalar value.
func (d *Detector) queryMetric(ctx context.Context, query string) (float64, error) {
	if d.prometheusURL == "" {
		return 0, fmt.Errorf("PROMETHEUS_URL not set")
	}
	endpoint := fmt.Sprintf("%s/api/v1/query?query=%s", d.prometheusURL, url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result promQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}
	if result.Status != "success" || len(result.Data.Result) == 0 {
		return 0, nil
	}
	vals := result.Data.Result[0].Value
	if len(vals) < 2 {
		return 0, nil
	}
	s, _ := vals[1].(string)
	return strconv.ParseFloat(s, 64)
}

// Scan evaluates all rules against live Prometheus data and returns triggered incidents.
func (d *Detector) Scan(ctx context.Context) []*Incident {
	if d.prometheusURL == "" {
		return nil
	}
	var triggered []*Incident
	for _, rule := range d.rules {
		value, err := d.queryMetric(ctx, rule.Query)
		if err != nil {
			continue
		}
		// Compute effective threshold
		threshold := rule.Baseline * rule.Threshold
		if rule.Baseline == 0 {
			threshold = rule.Threshold
		}
		if value > threshold {
			triggered = append(triggered, &Incident{
				ID:         newIncidentID(),
				Severity:   rule.Severity,
				Service:    rule.Service,
				Metric:     rule.Name,
				Value:      value,
				Baseline:   rule.Baseline,
				Title:      rule.Title,
				Status:     StatusOpen,
				DetectedAt: time.Now(),
			})
		}
	}
	return triggered
}
