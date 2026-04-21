package livestream

import (
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ════════════════════════════════════════════════════════════════════════════
// Sprint 17: Marketplace Brain — HTTP Handlers
// ════════════════════════════════════════════════════════════════════════════

// ── GET /livestream/feed ────────────────────────────────────────────────────

func (h *LiveAuctionHandler) GetFeedHandler(c *gin.Context) {
	if !IsSmartRankingEnabled() {
		// Fallback: return empty (smart ranking disabled)
		response.OK(c, []FeedEntry{})
		return
	}
	limit := 50
	if l, err := parseIntQuery(c, "limit"); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	entries, err := GetRankedFeed(h.db, h.rdb, limit)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, entries)
}

// ── GET /livestream/:id/score ──────────────────────────────────────────────

func (h *LiveAuctionHandler) GetSessionScoreHandler(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid session id")
		return
	}
	score, err := ComputeSessionScore(h.db, h.rdb, id)
	if err != nil {
		response.NotFound(c, "session score")
		return
	}
	response.OK(c, score)
}

// ── GET /livestream/:id/traffic ─────────────────────────────────────────────

func (h *LiveAuctionHandler) GetTrafficAllocationHandler(c *gin.Context) {
	if !IsTrafficAllocationEnabled() {
		response.BadRequest(c, "traffic allocation disabled")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid session id")
		return
	}
	alloc, err := AllocateTrafficForSession(h.db, h.rdb, id)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, alloc)
}

// ── GET /creators/:id/exposure ──────────────────────────────────────────────

func (h *LiveAuctionHandler) GetCreatorExposureHandler(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid creator id")
		return
	}
	exposure, err := AdjustCreatorExposure(h.db, h.rdb, id)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, exposure)
}

// ── POST /livestream/:id/recover ────────────────────────────────────────────

func (h *LiveAuctionHandler) RecoverSessionHandler(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid session id")
		return
	}
	RecoverSession(h.db, h.rdb, id)
	response.OK(c, gin.H{"recovery_triggered": true})
}

// ── GET /admin/marketplace/brain ────────────────────────────────────────────

func (h *LiveAuctionHandler) AdminMarketplaceBrainHandler(c *gin.Context) {
	metrics, err := GetMarketplaceBrainMetrics(h.db, h.rdb)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, metrics)
}

// ── POST /admin/marketplace/revenue-priority ────────────────────────────────

func (h *LiveAuctionHandler) AdminRevenuePriorityHandler(c *gin.Context) {
	feed, err := ApplyRevenuePriority(h.db, h.rdb)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, feed)
}

// ── POST /admin/marketplace/snapshot ────────────────────────────────────────

func (h *LiveAuctionHandler) AdminRecordSnapshotHandler(c *gin.Context) {
	// Score all live sessions and record snapshots
	var sessions []Session
	h.db.Where("status = ?", StatusLive).Find(&sessions)
	count := 0
	for _, sess := range sessions {
		score, err := ComputeSessionScore(h.db, h.rdb, sess.ID)
		if err != nil {
			continue
		}
		RecordScoreSnapshot(h.db, score)
		count++
	}
	response.OK(c, gin.H{"snapshots_recorded": count})
}
