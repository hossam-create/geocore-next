package fees

import (
	"strconv"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler provides HTTP handlers for fee engine admin operations.
type Handler struct {
	db     *gorm.DB
	engine *Engine
}

// NewHandler creates a fee handler.
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db, engine: NewEngine(db)}
}

// ListFees returns all fee configurations.
// GET /admin/fees
func (h *Handler) ListFees(c *gin.Context) {
	var configs []FeeConfig
	h.db.Order("fee_type, country, min_amount").Find(&configs)
	response.OK(c, configs)
}

// CreateFee creates a new fee configuration.
// POST /admin/fees
func (h *Handler) CreateFee(c *gin.Context) {
	var cfg FeeConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if cfg.Country == "" {
		cfg.Country = "*"
	}
	if err := h.db.Create(&cfg).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	h.engine.InvalidateCache()
	response.Created(c, cfg)
}

// UpdateFee updates an existing fee config.
// PUT /admin/fees/:id
func (h *Handler) UpdateFee(c *gin.Context) {
	id := c.Param("id")
	var cfg FeeConfig
	if err := h.db.Where("id = ?", id).First(&cfg).Error; err != nil {
		response.NotFound(c, "FeeConfig")
		return
	}
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.db.Save(&cfg)
	h.engine.InvalidateCache()
	response.OK(c, cfg)
}

// DeleteFee deactivates a fee config (soft delete via is_active=false).
// DELETE /admin/fees/:id
func (h *Handler) DeleteFee(c *gin.Context) {
	id := c.Param("id")
	h.db.Model(&FeeConfig{}).Where("id = ?", id).Update("is_active", false)
	h.engine.InvalidateCache()
	response.OK(c, gin.H{"status": "deactivated"})
}

// CalculateFee previews a fee calculation.
// GET /admin/fees/calculate?type=transaction&country=EGY&amount=100
func (h *Handler) CalculateFee(c *gin.Context) {
	feeType := FeeType(c.Query("type"))
	country := c.DefaultQuery("country", "*")
	if feeType == "" {
		response.BadRequest(c, "type required")
		return
	}
	amount, _ := strconv.ParseFloat(c.DefaultQuery("amount", "0"), 64)
	result := h.engine.Calculate(feeType, country, amount)
	response.OK(c, result)
}
