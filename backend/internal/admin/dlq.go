package admin

import (
	"strconv"

	"github.com/geocore-next/backend/pkg/jobs"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// DLQHandler provides admin endpoints for the Dead Letter Queue.
type DLQHandler struct {
	queue *jobs.JobQueue
}

// NewDLQHandler creates a DLQ handler backed by the given job queue.
func NewDLQHandler(q *jobs.JobQueue) *DLQHandler {
	return &DLQHandler{queue: q}
}

// ListFailedJobs returns failed jobs from the DLQ.
// GET /admin/dlq?limit=50
func (h *DLQHandler) ListFailedJobs(c *gin.Context) {
	if h.queue == nil {
		response.OK(c, gin.H{"jobs": []interface{}{}, "total": 0})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	jobs := h.queue.ListFailedJobs(limit)
	stats := h.queue.GetStats()
	response.OK(c, gin.H{
		"jobs":  jobs,
		"total": len(jobs),
		"stats": stats,
	})
}

// RetryAllFailed re-enqueues all failed jobs.
// POST /admin/dlq/retry
func (h *DLQHandler) RetryAllFailed(c *gin.Context) {
	if h.queue == nil {
		response.OK(c, gin.H{"retried": 0})
		return
	}
	count := h.queue.RetryFailed()
	response.OK(c, gin.H{"retried": count})
}

// RetryOneFailed re-enqueues a specific failed job by ID.
// POST /admin/dlq/:id/retry
func (h *DLQHandler) RetryOneFailed(c *gin.Context) {
	if h.queue == nil {
		response.BadRequest(c, "Job queue not available")
		return
	}
	jobID := c.Param("id")
	if h.queue.RetryOneFailedJob(jobID) {
		response.OK(c, gin.H{"retried": jobID})
	} else {
		response.NotFound(c, "Failed job")
	}
}

// PurgeDLQ removes all failed jobs from the DLQ.
// DELETE /admin/dlq
func (h *DLQHandler) PurgeDLQ(c *gin.Context) {
	if h.queue == nil {
		response.OK(c, gin.H{"purged": 0})
		return
	}
	removed := h.queue.PurgeFailedJobs()
	response.OK(c, gin.H{"purged": removed})
}
