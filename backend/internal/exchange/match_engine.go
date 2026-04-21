package exchange

// match_engine.go — Sprint 19/20 P2P Exchange Matching Engine.
//
// Rules:
//  1. Match opposite currency pairs (A wants EGP→USD paired with B wants USD→EGP).
//  2. Sort candidates by: tier_boost DESC, trust_score DESC, amount proximity ASC, created_at ASC.
//  3. VIP candidates receive +15 priority boost; PRO candidates receive +30.
//  4. Platform does NOT set FX rate — the agreed_rate is the midpoint of the two
//     preferred_rates (or RequestA's rate if B has none, or 0 meaning "any").
//  5. On match: both requests move to MATCHED, an ExchangeMatch row is inserted,
//     and an ExchangeSettlement is created in WAITING_PROOF state.
//  6. PRO users get auto-matching: TryMatch is called automatically on request creation.

import (
	"errors"
	"log/slog"
	"math"
	"time"

	"github.com/geocore-next/backend/internal/reputation"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MatchResult is returned by TryMatch.
type MatchResult struct {
	Match      ExchangeMatch
	Settlement ExchangeSettlement
	Fees       []ExchangeFee
}

// TryMatch attempts to find the best counter-request for req and creates the match.
// Returns ErrNoCounterparty if no suitable match exists.
var ErrNoCounterparty = errors.New("exchange: no suitable counterparty found")

func TryMatch(db *gorm.DB, req *ExchangeRequest) (*MatchResult, error) {
	// Find open counter-requests: opposite pair, not by the same user, not expired, not seeded.
	var candidates []ExchangeRequest
	q := db.Where(
		"from_currency = ? AND to_currency = ? AND status = ? AND user_id != ? AND is_system_generated = ?",
		req.ToCurrency, req.FromCurrency, StatusOpen, req.UserID, false,
	)
	if err := q.Order("created_at ASC").Limit(50).Find(&candidates).Error; err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, ErrNoCounterparty
	}

	// Score and rank candidates: tier boost + trust DESC + amount proximity ASC.
	type scored struct {
		req   ExchangeRequest
		score float64
	}
	ranked := make([]scored, 0, len(candidates))
	// Check if requester is a private member for liquidity priority.
	var requesterPrivate struct{ IsPrivateMember bool }
	db.Table("users").Select("is_private_member").Where("id = ?", req.UserID).Scan(&requesterPrivate)

	for _, c := range candidates {
		if c.ExpiresAt != nil && c.ExpiresAt.Before(time.Now()) {
			continue
		}
		candidateTier := GetUserTier(db, c.UserID)
		tierBoost := TierPriorityBoost(candidateTier)          // VIP +15, PRO +30, free 0
		trustScore := reputation.GetOverallScore(db, c.UserID) // 0–100
		amtDiff := math.Abs(c.Amount - req.Amount)
		// Higher is better: tier boost first, then trust, then proximity.
		composite := tierBoost + trustScore - (amtDiff/req.Amount)*10
		// Private members get faster matching priority.
		if requesterPrivate.IsPrivateMember {
			composite += 20
		}
		ranked = append(ranked, scored{req: c, score: composite})
	}
	if len(ranked) == 0 {
		return nil, ErrNoCounterparty
	}
	// Sort descending by composite score (insertion sort is fine for ≤50 items).
	for i := 1; i < len(ranked); i++ {
		for j := i; j > 0 && ranked[j].score > ranked[j-1].score; j-- {
			ranked[j], ranked[j-1] = ranked[j-1], ranked[j]
		}
	}
	best := ranked[0].req

	// Determine agreed_rate: midpoint of preferred rates (zero means "any").
	agreedRate := negotiatedRate(req.PreferredRate, best.PreferredRate)

	// Calculate fees for both parties.
	feesA := calculateFees(db, req, &best, req.UserID)
	feesB := calculateFees(db, req, &best, best.UserID)
	allFees := append(feesA, feesB...)

	// Persist inside a transaction.
	var result MatchResult
	err := db.Transaction(func(tx *gorm.DB) error {
		matchID := uuid.New()

		match := ExchangeMatch{
			ID:         matchID,
			RequestAID: req.ID,
			RequestBID: best.ID,
			AgreedRate: agreedRate,
			Status:     MatchPending,
		}
		if err := tx.Create(&match).Error; err != nil {
			return err
		}

		settlement := ExchangeSettlement{
			ID:      uuid.New(),
			MatchID: matchID,
			Status:  SettlementWaitingProof,
		}
		if err := tx.Create(&settlement).Error; err != nil {
			return err
		}

		// Mark both requests as MATCHED.
		if err := tx.Model(&ExchangeRequest{}).
			Where("id IN ?", []uuid.UUID{req.ID, best.ID}).
			Update("status", StatusMatched).Error; err != nil {
			return err
		}

		// Persist fee records (informational; collection is external).
		for i := range allFees {
			allFees[i].MatchID = matchID
		}
		if len(allFees) > 0 {
			if err := tx.Create(&allFees).Error; err != nil {
				return err
			}
		}

		result = MatchResult{
			Match:      match,
			Settlement: settlement,
			Fees:       allFees,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	slog.Info("exchange: match created",
		"match_id", result.Match.ID,
		"request_a", req.ID,
		"request_b", best.ID,
		"agreed_rate", agreedRate)

	return &result, nil
}

// negotiatedRate returns the midpoint of two optional preferred rates.
// If either party has no preference (nil or 0) it uses the other's rate.
// If both have no preference, returns 0 (to be filled in by users).
func negotiatedRate(rateA, rateB *float64) float64 {
	a := 0.0
	b := 0.0
	if rateA != nil {
		a = *rateA
	}
	if rateB != nil {
		b = *rateB
	}
	switch {
	case a == 0 && b == 0:
		return 0
	case a == 0:
		return b
	case b == 0:
		return a
	default:
		return (a + b) / 2
	}
}
