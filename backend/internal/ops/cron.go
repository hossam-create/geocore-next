package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/geocore-next/backend/pkg/jobs"
	"gorm.io/gorm"
)

// CronScheduler runs registered cron schedules by ticking every minute.
type CronScheduler struct {
	db       *gorm.DB
	jobQueue *jobs.JobQueue
	mu       sync.Mutex
	cancel   context.CancelFunc
	// internal action registry
	actions map[string]func(payload map[string]interface{}) error
}

// NewCronScheduler creates a scheduler backed by the DB and the existing job queue.
func NewCronScheduler(db *gorm.DB, jq *jobs.JobQueue) *CronScheduler {
	s := &CronScheduler{
		db:       db,
		jobQueue: jq,
		actions:  make(map[string]func(payload map[string]interface{}) error),
	}
	s.registerBuiltins()
	return s
}

// RegisterAction registers an internal action callable from a cron schedule.
func (s *CronScheduler) RegisterAction(name string, fn func(payload map[string]interface{}) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.actions[name] = fn
}

// Start begins the scheduler goroutine.
func (s *CronScheduler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.loop(ctx)
	slog.Info("ops: cron scheduler started")
}

// Stop halts the scheduler.
func (s *CronScheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	slog.Info("ops: cron scheduler stopped")
}

func (s *CronScheduler) loop(ctx context.Context) {
	// align to the next whole minute
	now := time.Now()
	wait := time.Until(now.Truncate(time.Minute).Add(time.Minute))
	select {
	case <-ctx.Done():
		return
	case <-time.After(wait):
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	s.tick()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

func (s *CronScheduler) tick() {
	now := time.Now().UTC()
	var schedules []CronSchedule
	if err := s.db.Where("enabled = true").Find(&schedules).Error; err != nil {
		slog.Error("ops: failed loading cron schedules", "error", err)
		return
	}
	for _, sched := range schedules {
		if matchesCron(sched.Schedule, now) {
			go s.run(sched)
		}
	}
}

func (s *CronScheduler) run(sched CronSchedule) {
	start := time.Now()
	var payload map[string]interface{}
	if sched.Payload != "" {
		_ = json.Unmarshal([]byte(sched.Payload), &payload)
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}

	var runErr error

	s.mu.Lock()
	fn, isInternal := s.actions[sched.Action]
	s.mu.Unlock()

	if isInternal {
		runErr = fn(payload)
	} else {
		// treat Action as a jobs.JobType and enqueue it
		job := &jobs.Job{
			Type:        jobs.JobType(sched.Action),
			Payload:     payload,
			MaxAttempts: 3,
		}
		runErr = s.jobQueue.Enqueue(job)
	}

	ok := runErr == nil
	errMsg := ""
	if runErr != nil {
		errMsg = runErr.Error()
		slog.Error("ops: cron job failed", "name", sched.Name, "error", errMsg)
	} else {
		slog.Info("ops: cron job ran", "name", sched.Name, "duration", time.Since(start))
	}

	now := time.Now()
	next := nextRunAfter(sched.Schedule, now)
	s.db.Model(&CronSchedule{}).Where("id = ?", sched.ID).Updates(map[string]interface{}{
		"last_run_at":  now,
		"last_run_ok":  ok,
		"last_run_err": errMsg,
		"next_run_at":  next,
	})
}

// registerBuiltins wires the existing deal expiry and escrow helpers as named actions.
func (s *CronScheduler) registerBuiltins() {
	s.actions["expire_deals"] = func(_ map[string]interface{}) error {
		return expireDealsAction(s.db)
	}
	s.actions["activate_deals"] = func(_ map[string]interface{}) error {
		return activateDealsAction(s.db)
	}
	s.actions["cleanup_sessions"] = func(_ map[string]interface{}) error {
		return cleanupSessionsAction(s.db)
	}
}

// ─────────────────────────────────────────────
// Builtin actions (call existing logic)
// ─────────────────────────────────────────────

func expireDealsAction(db *gorm.DB) error {
	now := time.Now()
	return db.Exec(
		"UPDATE deals SET status='expired' WHERE end_at < ? AND status IN ('active','scheduled')", now,
	).Error
}

func activateDealsAction(db *gorm.DB) error {
	now := time.Now()
	return db.Exec(
		"UPDATE deals SET status='active' WHERE start_at <= ? AND end_at > ? AND status='scheduled'", now, now,
	).Error
}

func cleanupSessionsAction(db *gorm.DB) error {
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	return db.Exec("DELETE FROM revoked_tokens WHERE revoked_at < ?", cutoff).Error
}

// SeedDefaultSchedules inserts built-in cron schedules if they don't exist.
func SeedDefaultSchedules(db *gorm.DB) {
	defaults := []CronSchedule{
		{
			Name:        "expire-deals",
			Description: "Mark expired deals every 5 minutes",
			Schedule:    "*/5 * * * *",
			Action:      "expire_deals",
			Enabled:     true,
		},
		{
			Name:        "activate-deals",
			Description: "Activate scheduled deals every 5 minutes",
			Schedule:    "*/5 * * * *",
			Action:      "activate_deals",
			Enabled:     true,
		},
		{
			Name:        "cleanup-sessions",
			Description: "Remove expired revoked tokens daily at 3 AM",
			Schedule:    "0 3 * * *",
			Action:      "cleanup_sessions",
			Enabled:     true,
		},
	}
	for _, d := range defaults {
		db.Where("name = ?", d.Name).FirstOrCreate(&d)
	}
}

// ─────────────────────────────────────────────
// Minimal 5-field cron expression matcher
// Fields: minute hour day month weekday  (all 0-based, weekday 0=Sunday)
// Supports: * , - /
// ─────────────────────────────────────────────

func matchesCron(expr string, t time.Time) bool {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return false
	}
	vals := [5]int{t.Minute(), t.Hour(), t.Day(), int(t.Month()), int(t.Weekday())}
	ranges := [5][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 6}}
	for i, f := range fields {
		if !matchField(f, vals[i], ranges[i][0], ranges[i][1]) {
			return false
		}
	}
	return true
}

func matchField(expr string, val, lo, hi int) bool {
	for _, part := range strings.Split(expr, ",") {
		if matchPart(part, val, lo, hi) {
			return true
		}
	}
	return false
}

func matchPart(part string, val, lo, hi int) bool {
	if part == "*" {
		return true
	}
	if strings.Contains(part, "/") {
		sp := strings.SplitN(part, "/", 2)
		step, err := strconv.Atoi(sp[1])
		if err != nil || step <= 0 {
			return false
		}
		start := lo
		if sp[0] != "*" {
			start, _ = strconv.Atoi(sp[0])
		}
		for v := start; v <= hi; v += step {
			if v == val {
				return true
			}
		}
		return false
	}
	if strings.Contains(part, "-") {
		sp := strings.SplitN(part, "-", 2)
		a, _ := strconv.Atoi(sp[0])
		b, _ := strconv.Atoi(sp[1])
		return val >= a && val <= b
	}
	n, err := strconv.Atoi(part)
	return err == nil && n == val
}

func nextRunAfter(expr string, after time.Time) *time.Time {
	t := after.UTC().Truncate(time.Minute).Add(time.Minute)
	for i := 0; i < 366*24*60; i++ {
		if matchesCron(expr, t) {
			return &t
		}
		t = t.Add(time.Minute)
	}
	return nil
}

// ─────────────────────────────────────────────
// API helpers
// ─────────────────────────────────────────────

func validateCronExpr(expr string) error {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return fmt.Errorf("cron expression must have 5 fields (minute hour day month weekday), got: %q", expr)
	}
	return nil
}
