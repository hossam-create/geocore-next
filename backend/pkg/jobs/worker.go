package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// JobType defines the type of background job
type JobType string

const (
	JobTypeEmail           JobType = "email"
	JobTypeSMS             JobType = "sms"
	JobTypePushNotification JobType = "push_notification"
	JobTypeAuctionEnd      JobType = "auction_end"
	JobTypeAuctionReminder JobType = "auction_reminder"
	JobTypeImageProcess    JobType = "image_process"
	JobTypeEscrowRelease   JobType = "escrow_release"
	JobTypeKYCVerify       JobType = "kyc_verify"
	JobTypeAnalytics       JobType = "analytics"
	JobTypeCleanup         JobType = "cleanup"
)

// JobStatus defines job lifecycle states
type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
	StatusRetrying  JobStatus = "retrying"
)

// Job represents a background job
type Job struct {
	ID          string                 `json:"id"`
	Type        JobType                `json:"type"`
	Payload     map[string]interface{} `json:"payload"`
	Status      JobStatus              `json:"status"`
	Priority    int                    `json:"priority"` // 1 = highest, 10 = lowest
	Attempts    int                    `json:"attempts"`
	MaxAttempts int                    `json:"max_attempts"`
	Delay       time.Duration          `json:"delay"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// JobQueue manages background jobs using Redis
type JobQueue struct {
	rdb      *redis.Client
	handlers map[JobType]JobHandler
	ctx      context.Context
	cancel   context.CancelFunc
}

// JobHandler is a function that processes a job
type JobHandler func(ctx context.Context, job *Job) error

// NewJobQueue creates a new job queue
func NewJobQueue(rdb *redis.Client) *JobQueue {
	ctx, cancel := context.WithCancel(context.Background())
	return &JobQueue{
		rdb:      rdb,
		handlers: make(map[JobType]JobHandler),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// RegisterHandler registers a handler for a job type
func (q *JobQueue) RegisterHandler(jobType JobType, handler JobHandler) {
	q.handlers[jobType] = handler
}

// Enqueue adds a job to the queue
func (q *JobQueue) Enqueue(job *Job) error {
	if job.ID == "" {
		job.ID = fmt.Sprintf("%s-%d", job.Type, time.Now().UnixNano())
	}
	if job.MaxAttempts == 0 {
		job.MaxAttempts = 3
	}
	if job.Priority == 0 {
		job.Priority = 5
	}
	job.Status = StatusPending
	job.CreatedAt = time.Now()

	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	queueKey := fmt.Sprintf("jobs:queue:%d", job.Priority)
	
	if job.Delay > 0 {
		// Delayed job - use sorted set with score as execution time
		score := float64(time.Now().Add(job.Delay).Unix())
		return q.rdb.ZAdd(q.ctx, "jobs:delayed", redis.Z{
			Score:  score,
			Member: string(data),
		}).Err()
	}

	return q.rdb.LPush(q.ctx, queueKey, data).Err()
}

// EnqueueAt schedules a job for a specific time
func (q *JobQueue) EnqueueAt(job *Job, at time.Time) error {
	job.Delay = time.Until(at)
	return q.Enqueue(job)
}

// EnqueueIn schedules a job after a duration
func (q *JobQueue) EnqueueIn(job *Job, delay time.Duration) error {
	job.Delay = delay
	return q.Enqueue(job)
}

// Start begins processing jobs
func (q *JobQueue) Start(workers int) {
	// Process delayed jobs
	go q.processDelayedJobs()

	// Start workers
	for i := 0; i < workers; i++ {
		go q.worker(i)
	}

	slog.Info("Job queue started", "workers", workers)
}

// Stop gracefully stops the job queue
func (q *JobQueue) Stop() {
	q.cancel()
	slog.Info("Job queue stopped")
}

// worker processes jobs from the queue
func (q *JobQueue) worker(id int) {
	queues := []string{
		"jobs:queue:1", "jobs:queue:2", "jobs:queue:3",
		"jobs:queue:4", "jobs:queue:5", "jobs:queue:6",
		"jobs:queue:7", "jobs:queue:8", "jobs:queue:9", "jobs:queue:10",
	}

	for {
		select {
		case <-q.ctx.Done():
			return
		default:
			// BRPOP from multiple queues (priority order)
			result, err := q.rdb.BRPop(q.ctx, 5*time.Second, queues...).Result()
			if err != nil {
				if err != redis.Nil {
					slog.Error("Worker error", "worker", id, "error", err)
				}
				continue
			}

			var job Job
			if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
				slog.Error("Failed to unmarshal job", "error", err)
				continue
			}

			q.processJob(&job)
		}
	}
}

// processJob executes a single job
func (q *JobQueue) processJob(job *Job) {
	handler, ok := q.handlers[job.Type]
	if !ok {
		slog.Warn("No handler for job type", "type", job.Type)
		return
	}

	now := time.Now()
	job.Status = StatusRunning
	job.StartedAt = &now
	job.Attempts++

	slog.Info("Processing job", "id", job.ID, "type", job.Type, "attempt", job.Attempts)

	ctx, cancel := context.WithTimeout(q.ctx, 5*time.Minute)
	defer cancel()

	err := handler(ctx, job)
	if err != nil {
		job.Error = err.Error()
		
		if job.Attempts < job.MaxAttempts {
			job.Status = StatusRetrying
			// Exponential backoff
			delay := time.Duration(job.Attempts*job.Attempts) * time.Second
			q.EnqueueIn(job, delay)
			slog.Warn("Job failed, retrying", "id", job.ID, "attempt", job.Attempts, "delay", delay)
		} else {
			job.Status = StatusFailed
			q.saveFailedJob(job)
			slog.Error("Job failed permanently", "id", job.ID, "error", err)
		}
		return
	}

	completed := time.Now()
	job.Status = StatusCompleted
	job.CompletedAt = &completed
	slog.Info("Job completed", "id", job.ID, "duration", completed.Sub(*job.StartedAt))
}

// processDelayedJobs moves delayed jobs to the main queue when ready
func (q *JobQueue) processDelayedJobs() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
			now := float64(time.Now().Unix())
			
			// Get jobs that are ready to run
			jobs, err := q.rdb.ZRangeByScore(q.ctx, "jobs:delayed", &redis.ZRangeBy{
				Min: "-inf",
				Max: fmt.Sprintf("%f", now),
			}).Result()
			
			if err != nil {
				continue
			}

			for _, jobData := range jobs {
				// Remove from delayed set
				q.rdb.ZRem(q.ctx, "jobs:delayed", jobData)
				
				// Add to main queue
				var job Job
				if err := json.Unmarshal([]byte(jobData), &job); err != nil {
					continue
				}
				job.Delay = 0
				q.Enqueue(&job)
			}
		}
	}
}

// saveFailedJob saves a failed job for later inspection
func (q *JobQueue) saveFailedJob(job *Job) {
	data, _ := json.Marshal(job)
	q.rdb.LPush(q.ctx, "jobs:failed", data)
	// Keep only last 1000 failed jobs
	q.rdb.LTrim(q.ctx, "jobs:failed", 0, 999)
}

// GetStats returns queue statistics
func (q *JobQueue) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	for i := 1; i <= 10; i++ {
		key := fmt.Sprintf("jobs:queue:%d", i)
		count, _ := q.rdb.LLen(q.ctx, key).Result()
		stats[fmt.Sprintf("priority_%d", i)] = count
	}
	
	delayed, _ := q.rdb.ZCard(q.ctx, "jobs:delayed").Result()
	stats["delayed"] = delayed
	
	failed, _ := q.rdb.LLen(q.ctx, "jobs:failed").Result()
	stats["failed"] = failed
	
	return stats
}

// RetryFailed retries all failed jobs
func (q *JobQueue) RetryFailed() int {
	count := 0
	for {
		data, err := q.rdb.RPop(q.ctx, "jobs:failed").Result()
		if err != nil {
			break
		}
		
		var job Job
		if err := json.Unmarshal([]byte(data), &job); err != nil {
			continue
		}
		
		job.Attempts = 0
		job.Error = ""
		q.Enqueue(&job)
		count++
	}
	return count
}
