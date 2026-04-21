package p2p

import (
	"log/slog"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct{ db *gorm.DB }

func NewHandler(db *gorm.DB) *Handler { return &Handler{db: db} }

// ── Create ──────────────────────────────────────────────────────────────────

type CreateReq struct {
	FromCurrency string  `json:"from_currency" binding:"required"`
	ToCurrency   string  `json:"to_currency" binding:"required"`
	FromAmount   float64 `json:"from_amount" binding:"required,gt=0"`
	ToAmount     float64 `json:"to_amount" binding:"required,gt=0"`
	UseEscrow    bool    `json:"use_escrow"`
	Notes        string  `json:"notes"`
}

// POST /api/v1/p2p/requests
func (h *Handler) Create(c *gin.Context) {
	var req CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if req.FromCurrency == req.ToCurrency {
		response.BadRequest(c, "currencies must be different")
		return
	}
	uid, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Unauthorized(c)
		return
	}

	rate := req.ToAmount / req.FromAmount

	er := ExchangeRequest{
		UserID:       uid,
		FromCurrency: req.FromCurrency,
		ToCurrency:   req.ToCurrency,
		FromAmount:   req.FromAmount,
		ToAmount:     req.ToAmount,
		DesiredRate:  rate,
		UseEscrow:    req.UseEscrow,
		Notes:        req.Notes,
		Status:       StatusOpen,
	}
	if err := h.db.Create(&er).Error; err != nil {
		slog.Error("p2p: create failed", "error", err.Error())
		response.InternalError(c, err)
		return
	}
	response.Created(c, er)
}

// ── List / Browse ───────────────────────────────────────────────────────────

// GET /api/v1/p2p/requests
func (h *Handler) List(c *gin.Context) {
	var items []ExchangeRequest
	q := h.db.Order("created_at DESC").Limit(50)

	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	} else {
		q = q.Where("status = ?", StatusOpen)
	}
	if fc := c.Query("from_currency"); fc != "" {
		q = q.Where("from_currency = ?", fc)
	}
	if tc := c.Query("to_currency"); tc != "" {
		q = q.Where("to_currency = ?", tc)
	}
	if mine := c.Query("mine"); mine == "true" {
		q = q.Where("user_id = ?", c.GetString("user_id"))
	}

	if err := q.Find(&items).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, items)
}

// GET /api/v1/p2p/requests/:id
func (h *Handler) Get(c *gin.Context) {
	var er ExchangeRequest
	if err := h.db.Where("id = ?", c.Param("id")).First(&er).Error; err != nil {
		response.NotFound(c, "exchange request")
		return
	}
	response.OK(c, er)
}

// ── Accept (match) ──────────────────────────────────────────────────────────

// POST /api/v1/p2p/requests/:id/accept
func (h *Handler) Accept(c *gin.Context) {
	uid, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Unauthorized(c)
		return
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		var er ExchangeRequest
		if err := tx.Where("id = ? AND status = ?", c.Param("id"), StatusOpen).First(&er).Error; err != nil {
			return err
		}
		if er.UserID == uid {
			return gorm.ErrInvalidData // can't accept own request
		}
		now := time.Now()
		return tx.Model(&er).Updates(map[string]any{
			"status":          StatusMatched,
			"matched_user_id": uid,
			"matched_at":      now,
		}).Error
	})

	if err != nil {
		response.BadRequest(c, "accept failed: "+err.Error())
		return
	}
	response.OK(c, gin.H{"message": "request accepted, you are now matched"})
}

// ── Complete ────────────────────────────────────────────────────────────────

// POST /api/v1/p2p/requests/:id/complete
func (h *Handler) Complete(c *gin.Context) {
	uid := c.GetString("user_id")

	var er ExchangeRequest
	if err := h.db.Where("id = ? AND status = ? AND (user_id = ? OR matched_user_id = ?)",
		c.Param("id"), StatusMatched, uid, uid).First(&er).Error; err != nil {
		response.NotFound(c, "matched exchange request")
		return
	}

	now := time.Now()
	h.db.Model(&er).Updates(map[string]any{"status": StatusCompleted, "completed_at": now})
	response.OK(c, gin.H{"message": "exchange completed"})
}

// ── Cancel ──────────────────────────────────────────────────────────────────

// POST /api/v1/p2p/requests/:id/cancel
func (h *Handler) Cancel(c *gin.Context) {
	uid := c.GetString("user_id")

	var er ExchangeRequest
	if err := h.db.Where("id = ? AND user_id = ? AND status IN ?",
		c.Param("id"), uid, []ExchangeStatus{StatusOpen, StatusMatched}).First(&er).Error; err != nil {
		response.NotFound(c, "exchange request")
		return
	}
	h.db.Model(&er).Update("status", StatusCancelled)
	response.OK(c, gin.H{"message": "exchange cancelled"})
}

// ── Messages ────────────────────────────────────────────────────────────────

// GET /api/v1/p2p/requests/:id/messages
func (h *Handler) ListMessages(c *gin.Context) {
	var msgs []ExchangeMessage
	if err := h.db.Where("request_id = ?", c.Param("id")).Order("created_at ASC").Limit(200).Find(&msgs).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, msgs)
}

// POST /api/v1/p2p/requests/:id/messages
func (h *Handler) SendMessage(c *gin.Context) {
	var body struct {
		Body string `json:"body" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	uid, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Unauthorized(c)
		return
	}
	reqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid request id")
		return
	}

	msg := ExchangeMessage{RequestID: reqID, SenderID: uid, Body: body.Body}
	if err := h.db.Create(&msg).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.Created(c, msg)
}
