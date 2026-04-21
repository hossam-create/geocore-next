package ops

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// AlertEngine polls alert rules and fires notifications when thresholds are breached.
type AlertEngine struct {
	db     *gorm.DB
	rdb    *redis.Client
	cancel context.CancelFunc
}

// NewAlertEngine creates an alert engine.
func NewAlertEngine(db *gorm.DB, rdb *redis.Client) *AlertEngine {
	return &AlertEngine{db: db, rdb: rdb}
}

// Start begins alert evaluation every minute.
func (e *AlertEngine) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	go e.loop(ctx)
	slog.Info("ops: alert engine started")
}

// Stop halts the alert engine.
func (e *AlertEngine) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
}

func (e *AlertEngine) loop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.evaluate()
		}
	}
}

func (e *AlertEngine) evaluate() {
	var rules []AlertRule
	if err := e.db.Where("enabled = true").Find(&rules).Error; err != nil {
		slog.Error("ops: failed loading alert rules", "error", err)
		return
	}
	for _, rule := range rules {
		val, err := e.collectMetric(rule.Metric, rule.Window)
		if err != nil {
			slog.Warn("ops: metric collection failed", "metric", rule.Metric, "error", err)
			continue
		}
		if e.breaches(rule.Condition, val, rule.Threshold) {
			e.fire(rule, val)
		}
	}
}

func (e *AlertEngine) collectMetric(metric, window string) (float64, error) {
	dur, err := parseDuration(window)
	if err != nil {
		dur = time.Hour
	}
	since := time.Now().Add(-dur)

	switch metric {
	case "job_failures":
		count, _ := e.rdb.LLen(context.Background(), "jobs:failed").Result()
		return float64(count), nil

	case "queue_depth":
		var total int64
		for i := 1; i <= 10; i++ {
			n, _ := e.rdb.LLen(context.Background(), fmt.Sprintf("jobs:queue:%d", i)).Result()
			total += n
		}
		return float64(total), nil

	case "payment_failures":
		var count int64
		e.db.Raw(
			"SELECT COUNT(*) FROM payments WHERE status = 'failed' AND created_at >= ?", since,
		).Scan(&count)
		return float64(count), nil

	case "payment_volume":
		var total float64
		e.db.Raw(
			"SELECT COALESCE(SUM(amount), 0) FROM payments WHERE status = 'succeeded' AND created_at >= ?", since,
		).Scan(&total)
		return total, nil

	case "new_users":
		var count int64
		e.db.Raw("SELECT COUNT(*) FROM users WHERE created_at >= ?", since).Scan(&count)
		return float64(count), nil

	case "active_auctions":
		var count int64
		e.db.Raw("SELECT COUNT(*) FROM auctions WHERE status = 'active' AND ends_at > NOW()").Scan(&count)
		return float64(count), nil
	}

	return 0, fmt.Errorf("unknown metric: %s", metric)
}

func (e *AlertEngine) breaches(condition string, val, threshold float64) bool {
	switch condition {
	case "gt":
		return val > threshold
	case "gte":
		return val >= threshold
	case "lt":
		return val < threshold
	case "lte":
		return val <= threshold
	case "eq":
		return val == threshold
	}
	return false
}

func (e *AlertEngine) fire(rule AlertRule, val float64) {
	// Throttle: don't fire the same rule more than once per window
	throttleKey := fmt.Sprintf("ops:alert:throttle:%s", rule.ID)
	if e.rdb.Exists(context.Background(), throttleKey).Val() > 0 {
		return
	}

	dur, _ := parseDuration(rule.Window)
	if dur == 0 {
		dur = time.Hour
	}
	e.rdb.Set(context.Background(), throttleKey, "1", dur)

	hist := AlertHistory{
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Metric:    rule.Metric,
		Value:     val,
		Threshold: rule.Threshold,
		FiredAt:   time.Now(),
	}
	e.db.Create(&hist)

	now := time.Now()
	e.db.Model(&AlertRule{}).Where("id = ?", rule.ID).Update("last_fired_at", now)

	slog.Warn("ops: alert fired",
		"rule", rule.Name,
		"metric", rule.Metric,
		"value", val,
		"threshold", rule.Threshold,
		"condition", rule.Condition,
	)
}

// AvailableMetrics returns the list of supported metric names.
func AvailableMetrics() []string {
	return []string{
		"job_failures",
		"queue_depth",
		"payment_failures",
		"payment_volume",
		"new_users",
		"active_auctions",
	}
}

func parseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	// Supports: 30m, 1h, 24h, 7d
	if len(s) > 1 && s[len(s)-1] == 'd' {
		days := 0
		fmt.Sscanf(s[:len(s)-1], "%d", &days)
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
