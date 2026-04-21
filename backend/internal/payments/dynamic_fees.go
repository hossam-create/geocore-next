package payments

import (
	"github.com/geocore-next/backend/internal/config"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Dynamic Fees Engine
// Adjusts platform fees based on supply/demand liquidity.
// ════════════════════════════════════════════════════════════════════════════

// DynamicFeeConfig holds fee percentages for different liquidity states.
type DynamicFeeConfig struct {
	LowSupplyFeePct      decimal.Decimal // e.g. 10% — reduce to attract supply
	BalancedFeePct       decimal.Decimal // e.g. 12% — standard
	HighDemandFeePct     decimal.Decimal // e.g. 15%
	VeryHighDemandFeePct decimal.Decimal // e.g. 18%
	VIPDiscountPct       decimal.Decimal // e.g. 30% off standard fee
}

var DefaultDynamicFeeConfig = DynamicFeeConfig{
	LowSupplyFeePct:      decimal.NewFromFloat(0.10),
	BalancedFeePct:       decimal.NewFromFloat(0.12),
	HighDemandFeePct:     decimal.NewFromFloat(0.15),
	VeryHighDemandFeePct: decimal.NewFromFloat(0.18),
	VIPDiscountPct:       decimal.NewFromFloat(0.30),
}

// LiquidityLevel represents the current supply/demand state.
type LiquidityLevel string

const (
	LiquidityLowSupply  LiquidityLevel = "low_supply"
	LiquidityBalanced   LiquidityLevel = "balanced"
	LiquidityHighDemand LiquidityLevel = "high_demand"
	LiquidityVeryHigh   LiquidityLevel = "very_high_demand"
)

// DynamicFeeResult holds the computed fee for a transaction.
type DynamicFeeResult struct {
	LiquidityLevel LiquidityLevel  `json:"liquidity_level"`
	BaseFeePct     decimal.Decimal `json:"base_fee_pct"`
	VIPDiscount    decimal.Decimal `json:"vip_discount"`
	FinalFeePct    decimal.Decimal `json:"final_fee_pct"`
	FeeAmount      decimal.Decimal `json:"fee_amount"`
	IsVIP          bool            `json:"is_vip"`
}

// CalculateLiquidityLevel determines the current liquidity state.
func CalculateLiquidityLevel(db *gorm.DB) LiquidityLevel {
	// Count active travelers (supply)
	var activeTravelers int64
	db.Table("trips").Where("status = ?", "active").Count(&activeTravelers)

	// Count pending delivery requests (demand)
	var pendingRequests int64
	db.Table("delivery_requests").Where("status = ?", "pending").Count(&pendingRequests)

	if activeTravelers == 0 {
		return LiquidityLowSupply
	}

	ratio := float64(pendingRequests) / float64(activeTravelers)

	switch {
	case ratio > 5.0:
		return LiquidityVeryHigh
	case ratio > 2.0:
		return LiquidityHighDemand
	case ratio < 0.5:
		return LiquidityLowSupply
	default:
		return LiquidityBalanced
	}
}

// CalculateDynamicFee computes the platform fee for a given amount.
func CalculateDynamicFee(db *gorm.DB, amount decimal.Decimal, userID string) DynamicFeeResult {
	if !config.GetFlags().EnableDynamicFees {
		// Feature disabled — return standard 12% fee
		return DynamicFeeResult{
			LiquidityLevel: LiquidityBalanced,
			BaseFeePct:     decimal.NewFromFloat(0.12),
			FinalFeePct:    decimal.NewFromFloat(0.12),
			FeeAmount:      amount.Mul(decimal.NewFromFloat(0.12)),
			IsVIP:          false,
		}
	}

	cfg := DefaultDynamicFeeConfig
	level := CalculateLiquidityLevel(db)

	var baseFee decimal.Decimal
	switch level {
	case LiquidityLowSupply:
		baseFee = cfg.LowSupplyFeePct
	case LiquidityHighDemand:
		baseFee = cfg.HighDemandFeePct
	case LiquidityVeryHigh:
		baseFee = cfg.VeryHighDemandFeePct
	default:
		baseFee = cfg.BalancedFeePct
	}

	// Check VIP status
	isVIP := false
	vipDiscount := decimal.Zero
	if userID != "" {
		vip := getVIPUserByID(db, userID)
		if vip != nil {
			isVIP = true
			vipDiscount = baseFee.Mul(cfg.VIPDiscountPct)
		}
	}

	finalFeePct := baseFee.Sub(vipDiscount)
	feeAmount := amount.Mul(finalFeePct)

	return DynamicFeeResult{
		LiquidityLevel: level,
		BaseFeePct:     baseFee,
		VIPDiscount:    vipDiscount,
		FinalFeePct:    finalFeePct,
		FeeAmount:      feeAmount,
		IsVIP:          isVIP,
	}
}

// getVIPUserByID looks up VIP status by string user ID.
func getVIPUserByID(db *gorm.DB, userID string) *VIPUser {
	var vip VIPUser
	if db.Where("user_id = ?", userID).First(&vip).Error != nil {
		return nil
	}
	return &vip
}

// GetDynamicFeeHandler — GET /api/v1/payments/fee?amount=100
func (h *Handler) GetDynamicFeeHandler(c *gin.Context) {
	amountStr := c.Query("amount")
	amount, err := decimal.NewFromString(amountStr)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		response.BadRequest(c, "invalid amount")
		return
	}

	userID := c.GetString("user_id")
	result := CalculateDynamicFee(h.db, amount, userID)
	response.OK(c, result)
}

// GetLiquidityLevelHandler — GET /api/v1/admin/payments/liquidity/level
func (h *Handler) GetLiquidityLevelHandler(c *gin.Context) {
	level := CalculateLiquidityLevel(h.db)
	response.OK(c, gin.H{"liquidity_level": level})
}
