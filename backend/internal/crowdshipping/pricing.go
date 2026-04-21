package crowdshipping

import (
	"log/slog"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// ── Pricing types ─────────────────────────────────────────────────────────────

type ItemType string

const (
	ItemTypeElectronics ItemType = "electronics"
	ItemTypeClothing    ItemType = "clothing"
	ItemTypeFood        ItemType = "food"
	ItemTypeFragile     ItemType = "fragile"
	ItemTypeOther       ItemType = "other"
)

type Urgency string

const (
	UrgencyStandard Urgency = "standard"
	UrgencyExpress  Urgency = "express"
	UrgencySameDay  Urgency = "same_day"
)

type PricingParams struct {
	WeightKg    float64  `json:"weight_kg"`
	DistanceKm  float64  `json:"distance_km"`
	Urgency     Urgency  `json:"urgency"`
	ItemType    ItemType `json:"item_type"`
	ItemValue   float64  `json:"item_value"`
	Origin      string   `json:"origin"`
	Destination string   `json:"destination"`
}

type PricingBreakdown struct {
	BaseFee               float64 `json:"base_fee"`
	WeightCost            float64 `json:"weight_cost"`
	DistanceCost          float64 `json:"distance_cost"`
	ItemTypeFee           float64 `json:"item_type_fee"`
	UrgencyMultiplier     float64 `json:"urgency_multiplier"`
	CorridorCustoms       float64 `json:"corridor_customs_multiplier"`
	ValueBandLabel        string  `json:"value_band_label"`
	ValueBandMult         float64 `json:"value_band_multiplier"`
	Subtotal              float64 `json:"subtotal"`
	Total                 float64 `json:"total"`
	TotalCents            int64   `json:"total_cents"`
	PlatformFee           float64 `json:"platform_fee"`
	PlatformFeeCents      int64   `json:"platform_fee_cents"`
	TravelerEarnings      float64 `json:"traveler_earnings"`
	TravelerEarningsCents int64   `json:"traveler_earnings_cents"`
	Currency              string  `json:"currency"`
}

// ── Constants ─────────────────────────────────────────────────────────────────

const (
	baseFee         = 5.0
	perKgRate       = 2.0
	perKmRate       = 0.5
	platformPct     = 0.15
	travelerPct     = 0.85
	pricingCurrency = "USD"
)

var itemTypeFees = map[ItemType]float64{
	ItemTypeElectronics: 3.0,
	ItemTypeClothing:    0.0,
	ItemTypeFood:        2.0,
	ItemTypeFragile:     5.0,
	ItemTypeOther:       1.0,
}

var urgencyMultipliers = map[Urgency]float64{
	UrgencyStandard: 1.0,
	UrgencyExpress:  1.5,
	UrgencySameDay:  2.0,
}

// ── Core calculation ──────────────────────────────────────────────────────────

func toCents(usd float64) int64     { return int64(usd*100 + 0.5) }
func fromCents(cents int64) float64 { return float64(cents) / 100.0 }

func CalculateDeliveryPrice(p PricingParams) PricingBreakdown {
	var bd PricingBreakdown

	bd.BaseFee = baseFee
	bd.WeightCost = p.WeightKg * perKgRate
	bd.DistanceCost = p.DistanceKm * perKmRate

	itFee, ok := itemTypeFees[p.ItemType]
	if !ok {
		itFee = itemTypeFees[ItemTypeOther]
	}
	bd.ItemTypeFee = itFee

	urgMult, ok := urgencyMultipliers[p.Urgency]
	if !ok {
		urgMult = urgencyMultipliers[UrgencyStandard]
	}
	bd.UrgencyMultiplier = urgMult

	corridorMult := 1.0
	valueBandMult := 1.0
	valueBandLabel := "Default"
	if cfg := GetCorridorConfig(p.Origin, p.Destination); cfg != nil {
		corridorMult = cfg.Risk.CustomsMultiplier
		vb := GetValueBandMultiplier(cfg, p.ItemValue)
		valueBandMult = vb.Multiplier
		valueBandLabel = vb.Label
	}
	bd.CorridorCustoms = corridorMult
	bd.ValueBandMult = valueBandMult
	bd.ValueBandLabel = valueBandLabel

	bd.Subtotal = bd.BaseFee + bd.WeightCost + bd.DistanceCost + bd.ItemTypeFee

	// Integer-based money: compute total in cents then derive split deterministically
	totalCents := toCents(bd.Subtotal * urgMult * corridorMult * valueBandMult)
	bd.TotalCents = totalCents
	bd.Total = fromCents(totalCents)

	// Single source of truth: PlatformFee derived first, TravelerEarnings = Total - PlatformFee
	platformCents := totalCents * 15 / 100
	bd.PlatformFeeCents = platformCents
	bd.PlatformFee = fromCents(platformCents)

	travelerCents := totalCents - platformCents
	bd.TravelerEarningsCents = travelerCents
	bd.TravelerEarnings = fromCents(travelerCents)

	bd.Currency = pricingCurrency

	slog.Info("crowdshipping: pricing calculated",
		"corridor", p.Origin+"_"+p.Destination,
		"weight_kg", p.WeightKg,
		"distance_km", p.DistanceKm,
		"urgency", string(p.Urgency),
		"item_type", string(p.ItemType),
		"item_value", p.ItemValue,
		"total_cents", totalCents,
		"platform_cents", platformCents,
		"traveler_cents", travelerCents,
		"currency", pricingCurrency,
	)

	return bd
}

// ── HTTP handler ──────────────────────────────────────────────────────────────

type pricingRequest struct {
	WeightKg    float64 `json:"weight_kg" binding:"required,gt=0"`
	DistanceKm  float64 `json:"distance_km" binding:"required,gte=0"`
	Urgency     string  `json:"urgency" binding:"required"`
	ItemType    string  `json:"item_type" binding:"required"`
	ItemValue   float64 `json:"item_value" binding:"required,gt=0"`
	Origin      string  `json:"origin" binding:"required"`
	Destination string  `json:"destination" binding:"required"`
}

func (h *Handler) CalculatePrice(c *gin.Context) {
	var req pricingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	params := PricingParams{
		WeightKg:    req.WeightKg,
		DistanceKm:  req.DistanceKm,
		Urgency:     Urgency(req.Urgency),
		ItemType:    ItemType(req.ItemType),
		ItemValue:   req.ItemValue,
		Origin:      req.Origin,
		Destination: req.Destination,
	}

	bd := CalculateDeliveryPrice(params)
	response.OK(c, bd)
}
