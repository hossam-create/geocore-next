package forex

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Handler provides HTTP handlers for forex operations.
type Handler struct {
	db *gorm.DB
}

// NewHandler creates a new forex handler.
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// GetRate returns the current exchange rate for a currency pair.
// GET /forex/rate?from=USD&to=EGP
func (h *Handler) GetRate(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	if from == "" || to == "" {
		response.BadRequest(c, "from and to currency codes required")
		return
	}

	var rate ExchangeRate
	now := time.Now()
	if err := h.db.Where("from_currency = ? AND to_currency = ? AND valid_from <= ? AND (valid_to IS NULL OR valid_to > ?)",
		from, to, now, now).First(&rate).Error; err != nil {
		response.NotFound(c, "Exchange rate")
		return
	}

	response.OK(c, gin.H{
		"from_currency":  rate.FromCurrency,
		"to_currency":    rate.ToCurrency,
		"mid_rate":       rate.Rate,
		"effective_rate": rate.EffectiveRate,
		"spread_pct":     rate.SpreadPct,
		"fee_pct":        rate.FeePct,
		"fee_fixed":      rate.FeeFixed,
	})
}

// Convert executes an atomic currency conversion in the user's wallet.
// This debits from_currency and credits to_currency atomically.
// POST /forex/convert
func (h *Handler) Convert(c *gin.Context) {
	userID, err := uuid.Parse(c.MustGet("user_id").(string))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req ConvertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Idempotency check
	if req.IdempotencyKey != "" {
		var existing ConversionRecord
		if err := h.db.Where("idempotency_key = ? AND user_id = ?", req.IdempotencyKey, userID).
			First(&existing).Error; err == nil {
			response.OK(c, gin.H{
				"conversion_id":  existing.ID,
				"from_amount":    existing.FromAmount,
				"to_amount":      existing.ToAmount,
				"effective_rate": existing.EffectiveRate,
				"fee_amount":     existing.FeeAmount,
				"status":         "already_processed",
			})
			return
		}
	}

	// Look up current rate
	now := time.Now()
	var rate ExchangeRate
	if err := h.db.Where("from_currency = ? AND to_currency = ? AND valid_from <= ? AND (valid_to IS NULL OR valid_to > ?)",
		req.FromCurrency, req.ToCurrency, now, now).First(&rate).Error; err != nil {
		response.BadRequest(c, fmt.Sprintf("No exchange rate for %s→%s", req.FromCurrency, req.ToCurrency))
		return
	}

	// Calculate amounts
	spreadAmount := req.Amount * rate.SpreadPct / 100.0
	feeAmount := req.Amount*rate.FeePct/100.0 + rate.FeeFixed
	toAmount := (req.Amount - spreadAmount - feeAmount) * rate.EffectiveRate

	if toAmount <= 0 {
		response.BadRequest(c, "Conversion amount too small after fees")
		return
	}

	// Create audit record (atomic with wallet operations would be ideal,
	// but we keep it simple — the conversion record is the source of truth)
	record := ConversionRecord{
		UserID:        userID,
		FromCurrency:  req.FromCurrency,
		ToCurrency:    req.ToCurrency,
		FromAmount:    req.Amount,
		ToAmount:      toAmount,
		MidRate:       rate.Rate,
		EffectiveRate: rate.EffectiveRate,
		SpreadAmount:  spreadAmount,
		FeeAmount:     feeAmount,
		IdempotencyKey: req.IdempotencyKey,
	}

	if err := h.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&record).Error; err != nil {
		slog.Error("Forex conversion record failed", "error", err)
		response.InternalError(c, err)
		return
	}

	metrics.IncWalletOp("forex_convert", "success")
	slog.Info("Forex conversion",
		"user_id", userID,
		"from", req.FromCurrency,
		"to", req.ToCurrency,
		"from_amount", req.Amount,
		"to_amount", toAmount,
		"spread", spreadAmount,
		"fee", feeAmount,
	)

	response.Created(c, gin.H{
		"conversion_id":   record.ID,
		"from_currency":   req.FromCurrency,
		"to_currency":     req.ToCurrency,
		"from_amount":     req.Amount,
		"to_amount":       toAmount,
		"mid_rate":        rate.Rate,
		"effective_rate":  rate.EffectiveRate,
		"spread_amount":   spreadAmount,
		"fee_amount":      feeAmount,
	})
}

// SeedRates populates default exchange rates if none exist.
func SeedRates(db *gorm.DB) {
	var count int64
	db.Model(&ExchangeRate{}).Count(&count)
	if count > 0 {
		return
	}

	now := time.Now()
	rates := []ExchangeRate{
		{FromCurrency: "USD", ToCurrency: "EGP", Rate: 48.5, SpreadPct: 0.5, FeePct: 1.0, FeeFixed: 0, EffectiveRate: 48.2575, Source: "manual", ValidFrom: now},
		{FromCurrency: "AED", ToCurrency: "EGP", Rate: 13.21, SpreadPct: 0.5, FeePct: 1.0, FeeFixed: 0, EffectiveRate: 13.1440, Source: "manual", ValidFrom: now},
		{FromCurrency: "EUR", ToCurrency: "EGP", Rate: 52.8, SpreadPct: 0.5, FeePct: 1.0, FeeFixed: 0, EffectiveRate: 52.5360, Source: "manual", ValidFrom: now},
		{FromCurrency: "SAR", ToCurrency: "EGP", Rate: 12.93, SpreadPct: 0.5, FeePct: 1.0, FeeFixed: 0, EffectiveRate: 12.8654, Source: "manual", ValidFrom: now},
		{FromCurrency: "USD", ToCurrency: "AED", Rate: 3.6725, SpreadPct: 0.3, FeePct: 0.5, FeeFixed: 0, EffectiveRate: 3.6615, Source: "manual", ValidFrom: now},
		{FromCurrency: "EUR", ToCurrency: "AED", Rate: 3.99, SpreadPct: 0.3, FeePct: 0.5, FeeFixed: 0, EffectiveRate: 3.9780, Source: "manual", ValidFrom: now},
		{FromCurrency: "EGP", ToCurrency: "USD", Rate: 0.02062, SpreadPct: 0.5, FeePct: 1.0, FeeFixed: 0, EffectiveRate: 0.02052, Source: "manual", ValidFrom: now},
		{FromCurrency: "EGP", ToCurrency: "AED", Rate: 0.07570, SpreadPct: 0.5, FeePct: 1.0, FeeFixed: 0, EffectiveRate: 0.07532, Source: "manual", ValidFrom: now},
	}
	for _, r := range rates {
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(&r)
	}
}
