package exchange

// liquidity.go — Part 2: Liquidity Engine.
//
// Detects low-liquidity pairs, computes imbalance ratios, and can seed
// system-generated placeholder requests to boost pair visibility.
// Platform NEVER holds funds — seeded requests are informational only.

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	LiquidityLowThreshold = 3   // < 3 open requests = low liquidity
	ImbalanceHighRatio    = 0.7 // > 70% one-sided = imbalanced
)

// ExchangeLiquidityProfile tracks supply/demand data for a currency pair.
type ExchangeLiquidityProfile struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	CurrencyPair    string    `gorm:"size:20;not null;uniqueIndex"                    json:"currency_pair"`
	ActiveRequests  int       `gorm:"not null;default:0"                              json:"active_requests"`
	AvgMatchTimeSec int64     `gorm:"not null;default:0"                              json:"avg_match_time_sec"`
	ImbalanceRatio  float64   `gorm:"not null;default:0"                              json:"imbalance_ratio"`
	IsLowLiquidity  bool      `gorm:"not null;default:false"                          json:"is_low_liquidity"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (ExchangeLiquidityProfile) TableName() string { return "exchange_liquidity_profiles" }

// LiquidityInsight is the full response payload for the liquidity endpoint.
type LiquidityInsight struct {
	Profile    ExchangeLiquidityProfile `json:"profile"`
	Verdict    string                   `json:"verdict"` // liquid | low_liquidity | imbalanced
	Suggestion string                   `json:"suggestion"`
	SeedActive bool                     `json:"seed_active"`
}

// RefreshLiquidityProfile recomputes and upserts the profile for a pair.
func RefreshLiquidityProfile(db *gorm.DB, from, to string) (*ExchangeLiquidityProfile, error) {
	pair := from + "/" + to

	var sideA, sideB int64
	db.Model(&ExchangeRequest{}).Where("from_currency=? AND to_currency=? AND status=?", from, to, StatusOpen).Count(&sideA)
	db.Model(&ExchangeRequest{}).Where("from_currency=? AND to_currency=? AND status=?", to, from, StatusOpen).Count(&sideB)
	total := sideA + sideB

	imbalance := 0.0
	if total > 0 {
		majority := sideA
		if sideB > sideA {
			majority = sideB
		}
		imbalance = float64(majority) / float64(total)
	}

	var avgSec float64
	db.Raw(`SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (em.updated_at - er.created_at))),0)
		FROM exchange_matches em
		JOIN exchange_requests er ON er.id=em.request_a_id
		WHERE er.from_currency=? AND er.to_currency=? AND em.status=?`,
		from, to, MatchSettled).Scan(&avgSec)

	profile := ExchangeLiquidityProfile{
		CurrencyPair:    pair,
		ActiveRequests:  int(total),
		AvgMatchTimeSec: int64(avgSec),
		ImbalanceRatio:  round2(imbalance),
		IsLowLiquidity:  total < LiquidityLowThreshold,
	}

	var existing ExchangeLiquidityProfile
	if err := db.Where("currency_pair=?", pair).First(&existing).Error; err != nil {
		profile.ID = uuid.New()
		if err := db.Create(&profile).Error; err != nil {
			return nil, err
		}
	} else {
		existing.ActiveRequests = profile.ActiveRequests
		existing.AvgMatchTimeSec = profile.AvgMatchTimeSec
		existing.ImbalanceRatio = profile.ImbalanceRatio
		existing.IsLowLiquidity = profile.IsLowLiquidity
		if err := db.Save(&existing).Error; err != nil {
			return nil, err
		}
		profile = existing
	}
	return &profile, nil
}

// GetLiquidityInsight returns a full human-readable insight for a pair.
func GetLiquidityInsight(db *gorm.DB, from, to string) LiquidityInsight {
	profile, _ := RefreshLiquidityProfile(db, from, to)
	if profile == nil {
		profile = &ExchangeLiquidityProfile{CurrencyPair: from + "/" + to, IsLowLiquidity: true}
	}
	var verdict, suggestion string
	switch {
	case profile.IsLowLiquidity:
		verdict = "low_liquidity"
		suggestion = "Few requests available. You may wait longer for a match. Consider listing your amount to attract counterparties."
	case profile.ImbalanceRatio > ImbalanceHighRatio:
		verdict = "imbalanced"
		suggestion = "More users want " + from + "→" + to + ". Counterparties for your direction are scarce — consider adjusting your rate."
	default:
		verdict = "liquid"
		suggestion = "Good liquidity. Matches are likely to happen quickly."
	}
	var seedCount int64
	db.Model(&ExchangeRequest{}).
		Where("from_currency=? AND to_currency=? AND status=? AND is_system_generated=?", from, to, StatusOpen, true).
		Count(&seedCount)
	return LiquidityInsight{
		Profile:    *profile,
		Verdict:    verdict,
		Suggestion: suggestion,
		SeedActive: seedCount > 0,
	}
}

// GetAllLiquidityProfiles returns all pair profiles ordered by active requests.
func GetAllLiquidityProfiles(db *gorm.DB) ([]ExchangeLiquidityProfile, error) {
	var profiles []ExchangeLiquidityProfile
	err := db.Order("active_requests DESC").Find(&profiles).Error
	return profiles, err
}

// SeedSystemRequest inserts a visibility-only placeholder (never matched, never influences rates).
func SeedSystemRequest(db *gorm.DB, from, to string, amount float64) error {
	var count int64
	db.Model(&ExchangeRequest{}).
		Where("from_currency=? AND to_currency=? AND status=? AND is_system_generated=? AND is_influencer_seed=?",
			from, to, StatusOpen, true, false).
		Count(&count)
	if count > 0 {
		return nil
	}
	req := ExchangeRequest{
		ID:                uuid.New(),
		UserID:            uuid.Nil,
		FromCurrency:      from,
		ToCurrency:        to,
		Amount:            amount,
		Status:            StatusOpen,
		IsSystemGenerated: true,
		IsInfluencerSeed:  false,
	}
	if err := db.Create(&req).Error; err != nil {
		return err
	}
	slog.Info("exchange: visibility seed created", "pair", from+"/"+to, "amount", amount)
	return nil
}

// SeedInfluencerRequest inserts a rate-alignment seed that influences rate hints
// and best_match_rate but is NEVER matched with real users.
// This is a "soft-match assist" — it guides the market, not fakes it.
func SeedInfluencerRequest(db *gorm.DB, from, to string, amount, preferredRate float64) error {
	var count int64
	db.Model(&ExchangeRequest{}).
		Where("from_currency=? AND to_currency=? AND status=? AND is_influencer_seed=?",
			from, to, StatusOpen, true).
		Count(&count)
	if count > 0 {
		return nil
	}
	req := ExchangeRequest{
		ID:                uuid.New(),
		UserID:            uuid.Nil,
		FromCurrency:      from,
		ToCurrency:        to,
		Amount:            amount,
		PreferredRate:     &preferredRate,
		Status:            StatusOpen,
		IsSystemGenerated: true,
		IsInfluencerSeed:  true,
	}
	if err := db.Create(&req).Error; err != nil {
		return err
	}
	slog.Info("exchange: influencer seed created", "pair", from+"/"+to, "rate", preferredRate)
	return nil
}
