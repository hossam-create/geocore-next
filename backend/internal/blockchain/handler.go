package blockchain

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

type CreateEscrowReq struct {
	OrderID  string  `json:"order_id" binding:"required"`
	SellerID string  `json:"seller_id" binding:"required"`
	Amount   float64 `json:"amount" binding:"required,gt=0"`
	Currency string  `json:"currency"`
	Chain    string  `json:"chain"`
}

// POST /api/v1/escrow
func (h *Handler) CreateEscrow(c *gin.Context) {
	var req CreateEscrowReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	buyerID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Unauthorized(c)
		return
	}
	orderID, _ := uuid.Parse(req.OrderID)
	sellerID, _ := uuid.Parse(req.SellerID)

	currency := req.Currency
	if currency == "" {
		currency = "AED"
	}
	chain := req.Chain
	if chain == "" {
		chain = "ethereum"
	}

	expires := time.Now().Add(72 * time.Hour)

	ec := EscrowContract{
		OrderID:   orderID,
		BuyerID:   buyerID,
		SellerID:  sellerID,
		Amount:    req.Amount,
		Currency:  currency,
		Chain:     chain,
		Status:    EscrowPending,
		ExpiresAt: &expires,
	}

	if err := h.db.Create(&ec).Error; err != nil {
		slog.Error("blockchain: create escrow failed", "error", err.Error())
		response.InternalError(c, err)
		return
	}
	response.Created(c, ec)
}

// GET /api/v1/escrow
func (h *Handler) List(c *gin.Context) {
	uid := c.GetString("user_id")
	var items []EscrowContract
	q := h.db.Where("buyer_id = ? OR seller_id = ?", uid, uid).Order("created_at DESC").Limit(50)
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	if err := q.Find(&items).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, items)
}

// GET /api/v1/escrow/:id
func (h *Handler) Get(c *gin.Context) {
	var ec EscrowContract
	if err := h.db.Where("id = ?", c.Param("id")).First(&ec).Error; err != nil {
		response.NotFound(c, "escrow contract")
		return
	}
	response.OK(c, ec)
}

// POST /api/v1/escrow/:id/fund
func (h *Handler) Fund(c *gin.Context) {
	var body struct {
		TxHash          string `json:"tx_hash" binding:"required"`
		ContractAddress string `json:"contract_address"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	uid := c.GetString("user_id")
	var ec EscrowContract
	if err := h.db.Where("id = ? AND buyer_id = ? AND status = ?", c.Param("id"), uid, EscrowPending).First(&ec).Error; err != nil {
		response.NotFound(c, "pending escrow")
		return
	}

	now := time.Now()
	updates := map[string]any{
		"status":      EscrowFunded,
		"tx_hash_fund": body.TxHash,
		"funded_at":   now,
	}
	if body.ContractAddress != "" {
		updates["contract_address"] = body.ContractAddress
	}
	h.db.Model(&ec).Updates(updates)
	response.OK(c, gin.H{"message": "escrow funded"})
}

// POST /api/v1/escrow/:id/release
func (h *Handler) Release(c *gin.Context) {
	var body struct {
		TxHash string `json:"tx_hash"`
	}
	c.ShouldBindJSON(&body)

	uid := c.GetString("user_id")
	var ec EscrowContract
	if err := h.db.Where("id = ? AND buyer_id = ? AND status = ?", c.Param("id"), uid, EscrowFunded).First(&ec).Error; err != nil {
		response.NotFound(c, "funded escrow")
		return
	}

	now := time.Now()
	updates := map[string]any{"status": EscrowReleased, "released_at": now}
	if body.TxHash != "" {
		updates["tx_hash_release"] = body.TxHash
	}
	h.db.Model(&ec).Updates(updates)
	response.OK(c, gin.H{"message": "funds released to seller"})
}

// POST /api/v1/escrow/:id/refund
func (h *Handler) Refund(c *gin.Context) {
	uid := c.GetString("user_id")
	var ec EscrowContract
	if err := h.db.Where("id = ? AND (buyer_id = ? OR seller_id = ?) AND status = ?",
		c.Param("id"), uid, uid, EscrowFunded).First(&ec).Error; err != nil {
		response.NotFound(c, "funded escrow")
		return
	}
	h.db.Model(&ec).Update("status", EscrowRefunded)
	response.OK(c, gin.H{"message": "escrow refunded"})
}

// POST /api/v1/escrow/:id/dispute
func (h *Handler) Dispute(c *gin.Context) {
	uid := c.GetString("user_id")
	var ec EscrowContract
	if err := h.db.Where("id = ? AND (buyer_id = ? OR seller_id = ?) AND status = ?",
		c.Param("id"), uid, uid, EscrowFunded).First(&ec).Error; err != nil {
		response.NotFound(c, "funded escrow")
		return
	}
	h.db.Model(&ec).Update("status", EscrowDisputed)
	response.OK(c, gin.H{"message": "escrow disputed — admin will review"})
}
