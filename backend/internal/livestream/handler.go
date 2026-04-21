package livestream

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/freeze"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Handler struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewHandler(db *gorm.DB, rdb ...*redis.Client) *Handler {
	h := &Handler{db: db}
	if len(rdb) > 0 {
		h.rdb = rdb[0]
	}
	return h
}

// ── Request types ──────────────────────────────────────────────────────────────

type CreateSessionReq struct {
	AuctionID    *string `json:"auction_id"`
	Title        string  `json:"title" binding:"required,min=3,max=255"`
	Description  string  `json:"description"`
	ThumbnailURL string  `json:"thumbnail_url"`
}

type JoinTokenReq struct {
	DisplayName string `json:"display_name"`
}

// ── POST /api/v1/livestream — create a session (host only) ────────────────────

func (h *Handler) CreateSession(c *gin.Context) {
	var req CreateSessionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	hostIDStr := c.GetString("user_id")
	hostID, err := uuid.Parse(hostIDStr)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	var auctionID *uuid.UUID
	if req.AuctionID != nil && *req.AuctionID != "" {
		id, e := uuid.Parse(*req.AuctionID)
		if e != nil {
			response.BadRequest(c, "invalid auction_id")
			return
		}
		auctionID = &id
	}

	roomName := fmt.Sprintf("room-%s", uuid.New().String()[:8])

	sess := Session{
		HostID:       hostID,
		AuctionID:    auctionID,
		Title:        req.Title,
		Description:  req.Description,
		ThumbnailURL: req.ThumbnailURL,
		Status:       StatusScheduled,
		RoomName:     roomName,
	}
	if err := h.db.Create(&sess).Error; err != nil {
		slog.Error("livestream: failed to create session", "error", err.Error())
		response.InternalError(c, err)
		return
	}

	slog.Info("livestream session created", "session_id", sess.ID, "host", hostIDStr)
	response.Created(c, sess)
}

// ── POST /api/v1/livestream/:id/start ─────────────────────────────────────────

func (h *Handler) StartSession(c *gin.Context) {
	sess, ok := h.getSessionAndVerifyHost(c)
	if !ok {
		return
	}
	if sess.Status != StatusScheduled {
		response.BadRequest(c, "session is not in scheduled state")
		return
	}

	now := time.Now()
	if err := h.db.Model(sess).Updates(map[string]any{
		"status":     StatusLive,
		"started_at": now,
	}).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	token, err := GenerateToken(sess.RoomName, c.GetString("user_id"), "Host", true)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, gin.H{
		"session":     sess,
		"token":       token,
		"livekit_url": LiveKitURL(),
		"simulated":   os.Getenv("LIVEKIT_API_KEY") == "",
	})
}

// ── POST /api/v1/livestream/:id/end ───────────────────────────────────────────

func (h *Handler) EndSession(c *gin.Context) {
	sess, ok := h.getSessionAndVerifyHost(c)
	if !ok {
		return
	}
	if sess.Status == StatusEnded || sess.Status == StatusCancelled {
		response.BadRequest(c, "session already ended")
		return
	}

	now := time.Now()
	if err := h.db.Model(sess).Updates(map[string]any{
		"status":   StatusEnded,
		"ended_at": now,
	}).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	slog.Info("livestream session ended", "session_id", sess.ID)
	response.OK(c, sess)
}

// ── POST /api/v1/livestream/:id/join — viewer gets token ─────────────────────

func (h *Handler) JoinSession(c *gin.Context) {
	// ── Global panic check ──────────────────────────────────────────────────
	if IsLiveSystemDisabled() {
		response.BadRequest(c, "Live system temporarily disabled")
		return
	}

	// ── Freeze check: frozen users cannot join sessions ─────────────────────
	if userID := c.GetString("user_id"); userID != "" {
		if uid, err := uuid.Parse(userID); err == nil {
			if freeze.IsUserFrozen(h.db, uid) {
				response.Forbidden(c)
				return
			}
		}
	}

	sessionID := c.Param("id")
	var sess Session
	if err := h.db.Where("id = ? AND status = ?", sessionID, StatusLive).First(&sess).Error; err != nil {
		response.NotFound(c, "session")
		return
	}

	// ── Sprint 12: Pay-to-enter VIP gate ───────────────────────────────────
	if IsEntryFeeEnabled() && sess.EntryFeeCents > 0 {
		if uidStr := c.GetString("user_id"); uidStr != "" {
			if uid, err := uuid.Parse(uidStr); err == nil {
				if !HasUserPaidEntry(h.db, uid, sess.ID) {
					response.BadRequest(c, fmt.Sprintf("Entry fee required (%d cents). POST /livestream/%s/entry first.", sess.EntryFeeCents, sess.ID))
					return
				}
			}
		}
	}

	var req JoinTokenReq
	c.ShouldBindJSON(&req)
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = "Viewer"
	}

	viewerID := c.GetString("user_id")
	if viewerID == "" {
		viewerID = "anon-" + uuid.New().String()[:8]
	}

	token, err := GenerateToken(sess.RoomName, viewerID, displayName, false)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	h.db.Model(&sess).UpdateColumn("viewer_count", gorm.Expr("viewer_count + 1"))

	// Broadcast viewer join event
	if h.rdb != nil {
		la := NewLiveAuctionHandler(h.db, h.rdb)
		la.broadcastViewerEvent(sess.ID, EventViewerJoin, viewerID, displayName)
		// Sprint 14: AI drop-off tracking — snapshot peak viewer count
		la.TrackViewerSnapshot(sess.ID, int(sess.ViewerCount)+1)
	}

	// Sprint 11: funnel tracking — record 'view' stage
	var userUUID *uuid.UUID
	if uid, err := uuid.Parse(viewerID); err == nil {
		userUUID = &uid
	}
	RecordConversionEvent(h.db, sess.ID, nil, userUUID, StageView, 0, "")

	// Sprint 15: daily live-join streak (authenticated viewers only)
	if userUUID != nil {
		go UpdateStreak(h.db, *userUUID, "live_join")
	}

	response.OK(c, gin.H{
		"session":     sess,
		"token":       token,
		"livekit_url": LiveKitURL(),
		"simulated":   os.Getenv("LIVEKIT_API_KEY") == "",
	})
}

// ── GET /api/v1/livestream — list live sessions ───────────────────────────────

func (h *Handler) ListSessions(c *gin.Context) {
	statusFilter := c.DefaultQuery("status", "live")

	var sessions []Session
	q := h.db.Order("created_at DESC").Limit(50)
	if statusFilter != "all" {
		q = q.Where("status = ?", statusFilter)
	}
	if err := q.Find(&sessions).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, sessions)
}

// ── GET /api/v1/livestream/:id ────────────────────────────────────────────────

func (h *Handler) GetSession(c *gin.Context) {
	var sess Session
	if err := h.db.Where("id = ? AND deleted_at IS NULL", c.Param("id")).First(&sess).Error; err != nil {
		response.NotFound(c, "session")
		return
	}
	response.OK(c, sess)
}

// ── POST /api/v1/livestream/:id/leave — viewer leaves session ─────────────────

func (h *Handler) LeaveSession(c *gin.Context) {
	sessionID := c.Param("id")
	var sess Session
	if err := h.db.Where("id = ?", sessionID).First(&sess).Error; err != nil {
		response.NotFound(c, "session")
		return
	}

	viewerID := c.GetString("user_id")
	if viewerID == "" {
		viewerID = c.Query("viewer_id")
	}

	// Decrement viewer count (floor at 0)
	h.db.Model(&sess).
		Where("viewer_count > 0").
		UpdateColumn("viewer_count", gorm.Expr("viewer_count - 1"))

	// Broadcast viewer leave event
	if h.rdb != nil && viewerID != "" {
		la := NewLiveAuctionHandler(h.db, h.rdb)
		la.broadcastViewerEvent(sess.ID, EventViewerLeave, viewerID, "")

		// Sprint 14: AI drop-off prevention — if viewers dropped >30% from peak,
		// nudge the seller + broadcast toast to remaining viewers.
		var freshSess Session
		h.db.Select("viewer_count").Where("id = ?", sess.ID).First(&freshSess)
		if la.DetectDropoff(sess.ID, int(freshSess.ViewerCount)) {
			la.BroadcastToast(sess.ID, "👀 Keep watching — things are heating up!", "👀")
			// Also trigger AI evaluation (dropoff remedy)
			go func() {
				// Find the active item in this session and evaluate
				var active LiveItem
				if err := h.db.Where("session_id = ? AND status = ?", sess.ID, ItemActive).
					First(&active).Error; err == nil {
					la.EvaluateAndSuggest(sess.ID, active.ID)
				}
			}()
		}
	}

	response.OK(c, gin.H{"message": "left session"})
}

// ── DELETE /api/v1/livestream/:id — cancel scheduled session ─────────────────

func (h *Handler) CancelSession(c *gin.Context) {
	sess, ok := h.getSessionAndVerifyHost(c)
	if !ok {
		return
	}
	if sess.Status == StatusLive {
		response.BadRequest(c, "stop the stream before cancelling")
		return
	}

	if err := h.db.Model(sess).Update("status", StatusCancelled).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "session cancelled"})
}

// ── helper ────────────────────────────────────────────────────────────────────

func (h *Handler) getSessionAndVerifyHost(c *gin.Context) (*Session, bool) {
	var sess Session
	if err := h.db.Where("id = ? AND deleted_at IS NULL", c.Param("id")).First(&sess).Error; err != nil {
		response.NotFound(c, "session")
		return nil, false
	}
	if sess.HostID.String() != c.GetString("user_id") {
		c.JSON(http.StatusForbidden, gin.H{"error": "not the session host"})
		return nil, false
	}
	return &sess, true
}
