package payments

import (
	"fmt"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// VIPUser Model
// ════════════════════════════════════════════════════════════════════════════

type VIPUser struct {
	ID              uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID          uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	Tier            string          `gorm:"size:20;not null;default:'silver'" json:"tier"`
	DailyLimit      decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"daily_limit"`
	MonthlyLimit    decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"monthly_limit"`
	TransferFeePct  decimal.Decimal `gorm:"type:decimal(5,4);default:0.005" json:"transfer_fee_pct"`
	PriorityMatching bool           `gorm:"default:true" json:"priority_matching"`
	FastTrackKYC    bool            `gorm:"default:false" json:"fast_track_kyc"`
	DedicatedAgentID *uuid.UUID     `gorm:"type:uuid" json:"dedicated_agent_id,omitempty"`
	ActivatedAt     time.Time       `json:"activated_at"`
	CreatedAt       time.Time       `json:"created_at"`
}

func (VIPUser) TableName() string { return "vip_users" }

// ════════════════════════════════════════════════════════════════════════════
// VIP Tier Configs
// ════════════════════════════════════════════════════════════════════════════

type VIPTierConfig struct {
	DailyLimit       decimal.Decimal
	MonthlyLimit     decimal.Decimal
	TransferFeePct   decimal.Decimal
	PriorityMatching bool
	FastTrackKYC     bool
	WithdrawSpeedH   int
	DedicatedAgent   bool
	ZeroFeeTransfers int
}

var VIPTiers = map[string]VIPTierConfig{
	"silver": {
		DailyLimit:       decimal.NewFromFloat(1000),
		MonthlyLimit:     decimal.NewFromFloat(10000),
		TransferFeePct:   decimal.NewFromFloat(0.008),
		PriorityMatching: true,
		WithdrawSpeedH:   4,
	},
	"gold": {
		DailyLimit:       decimal.NewFromFloat(5000),
		MonthlyLimit:     decimal.NewFromFloat(50000),
		TransferFeePct:   decimal.NewFromFloat(0.005),
		PriorityMatching: true,
		FastTrackKYC:     true,
		WithdrawSpeedH:   2,
		DedicatedAgent:   true,
	},
	"platinum": {
		DailyLimit:       decimal.NewFromFloat(20000),
		MonthlyLimit:     decimal.NewFromFloat(200000),
		TransferFeePct:   decimal.NewFromFloat(0.003),
		PriorityMatching: true,
		FastTrackKYC:     true,
		WithdrawSpeedH:   1,
		DedicatedAgent:   true,
		ZeroFeeTransfers: 3,
	},
}

// ════════════════════════════════════════════════════════════════════════════
// VIP Handlers
// ════════════════════════════════════════════════════════════════════════════

// UpgradeToVIP — POST /api/v1/admin/payments/users/:id/vip
func (h *Handler) UpgradeToVIP(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user ID")
		return
	}

	var req struct {
		Tier            string `json:"tier" binding:"required"`
		DedicatedAgentID *uuid.UUID `json:"dedicated_agent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	config, ok := VIPTiers[req.Tier]
	if !ok {
		response.BadRequest(c, "invalid tier, must be silver/gold/platinum")
		return
	}

	vip := VIPUser{
		ID:              uuid.New(),
		UserID:          userID,
		Tier:            req.Tier,
		DailyLimit:      config.DailyLimit,
		MonthlyLimit:    config.MonthlyLimit,
		TransferFeePct:  config.TransferFeePct,
		PriorityMatching: config.PriorityMatching,
		FastTrackKYC:    config.FastTrackKYC,
		DedicatedAgentID: req.DedicatedAgentID,
		ActivatedAt:     time.Now(),
	}

	if err := h.db.Where("user_id = ?", userID).Assign(vip).FirstOrCreate(&vip).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, vip)
}

// UpdateVIPTier — PUT /api/v1/admin/payments/users/:id/vip/tier
func (h *Handler) UpdateVIPTier(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user ID")
		return
	}

	var req struct {
		Tier            string `json:"tier" binding:"required"`
		DedicatedAgentID *uuid.UUID `json:"dedicated_agent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	config, ok := VIPTiers[req.Tier]
	if !ok {
		response.BadRequest(c, "invalid tier, must be silver/gold/platinum")
		return
	}

	var vip VIPUser
	if h.db.Where("user_id = ?", userID).First(&vip).Error != nil {
		response.NotFound(c, "VIP user")
		return
	}

	h.db.Model(&vip).Updates(map[string]interface{}{
		"tier":              req.Tier,
		"daily_limit":       config.DailyLimit,
		"monthly_limit":     config.MonthlyLimit,
		"transfer_fee_pct":  config.TransferFeePct,
		"priority_matching": config.PriorityMatching,
		"fast_track_kyc":   config.FastTrackKYC,
		"dedicated_agent_id": req.DedicatedAgentID,
	})

	response.OK(c, vip)
}

// ════════════════════════════════════════════════════════════════════════════
// VIP Limit Checks
// ════════════════════════════════════════════════════════════════════════════

// checkUserLimits verifies the user's VIP daily/monthly limits for an operation.
func checkUserLimits(db *gorm.DB, userID uuid.UUID, usdAmount decimal.Decimal, operation string) error {
	var vip VIPUser
	isVIP := db.Where("user_id = ?", userID).First(&vip).Error == nil

	if !isVIP {
		// Default limits for non-VIP users
		vip.DailyLimit = decimal.NewFromFloat(500)
		vip.MonthlyLimit = decimal.NewFromFloat(5000)
	}

	// Check daily limit
	today := time.Now().Truncate(24 * time.Hour)
	var dailyTotal decimal.Decimal
	switch operation {
	case "deposit":
		var total float64
		db.Model(&DepositRequest{}).
			Where("user_id = ? AND status IN ? AND created_at > ?",
				userID, []string{"confirmed", "pending", "paid"}, today).
			Select("COALESCE(SUM(usd_amount),0)").Scan(&total)
		dailyTotal = decimal.NewFromFloat(total)
	case "withdraw":
		var total float64
		db.Model(&WithdrawRequest{}).
			Where("user_id = ? AND status IN ? AND created_at > ?",
				userID, []string{"completed", "pending", "assigned", "processing"}, today).
			Select("COALESCE(SUM(usd_amount),0)").Scan(&total)
		dailyTotal = decimal.NewFromFloat(total)
	}

	if dailyTotal.Add(usdAmount).GreaterThan(vip.DailyLimit) {
		return fmt.Errorf("daily limit exceeded (%s/%s)", dailyTotal.String(), vip.DailyLimit.String())
	}

	// Check monthly limit
	monthStart := time.Now().AddDate(0, 0, -30)
	var monthlyTotal decimal.Decimal
	switch operation {
	case "deposit":
		var total float64
		db.Model(&DepositRequest{}).
			Where("user_id = ? AND status IN ? AND created_at > ?",
				userID, []string{"confirmed", "pending", "paid"}, monthStart).
			Select("COALESCE(SUM(usd_amount),0)").Scan(&total)
		monthlyTotal = decimal.NewFromFloat(total)
	case "withdraw":
		var total float64
		db.Model(&WithdrawRequest{}).
			Where("user_id = ? AND status IN ? AND created_at > ?",
				userID, []string{"completed", "pending", "assigned", "processing"}, monthStart).
			Select("COALESCE(SUM(usd_amount),0)").Scan(&total)
		monthlyTotal = decimal.NewFromFloat(total)
	}

	if monthlyTotal.Add(usdAmount).GreaterThan(vip.MonthlyLimit) {
		return fmt.Errorf("monthly limit exceeded (%s/%s)", monthlyTotal.String(), vip.MonthlyLimit.String())
	}

	return nil
}

// getVIPUser returns the VIP profile for a user, or nil if not VIP.
func getVIPUser(db *gorm.DB, userID uuid.UUID) *VIPUser {
	var vip VIPUser
	if db.Where("user_id = ?", userID).First(&vip).Error != nil {
		return nil
	}
	return &vip
}
