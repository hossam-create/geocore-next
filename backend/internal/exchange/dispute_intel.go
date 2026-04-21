package exchange

// dispute_intel.go — Part 5: Dispute Intelligence.
//
// AutoResolveDispute applies heuristic rules to close disputes without manual review:
//   1. One side uploaded proof, other didn't → uploader wins immediately.
//   2. Both uploaded → compare upload timestamps and claimed amounts.
//   3. Loser receives a reputation penalty and a risk flag.
//   4. If inconclusive → left OPEN for human review.

import (
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/reputation"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AutoResolveResult is the output of AutoResolveDispute.
type AutoResolveResult struct {
	Resolved       bool   `json:"resolved"`
	Winner         string `json:"winner,omitempty"` // "a" | "b" | "inconclusive"
	Reason         string `json:"reason"`
	PenaltyApplied bool   `json:"penalty_applied"`
}

// AutoResolveDispute attempts to resolve a match dispute using proof metadata.
func AutoResolveDispute(db *gorm.DB, matchID uuid.UUID) AutoResolveResult {
	var settlement ExchangeSettlement
	if err := db.First(&settlement, "match_id=?", matchID).Error; err != nil {
		return AutoResolveResult{Reason: "settlement not found"}
	}
	var match ExchangeMatch
	if err := db.First(&match, "id=?", matchID).Error; err != nil {
		return AutoResolveResult{Reason: "match not found"}
	}
	var reqA, reqB ExchangeRequest
	db.First(&reqA, "id=?", match.RequestAID)
	db.First(&reqB, "id=?", match.RequestBID)

	hasA := settlement.UserAProof != ""
	hasB := settlement.UserBProof != ""

	var result AutoResolveResult

	switch {
	case hasA && !hasB:
		result = AutoResolveResult{Resolved: true, Winner: "a",
			Reason: "Party A submitted proof; Party B did not."}
		penalise(db, reqB.UserID, matchID)
		result.PenaltyApplied = true
		closeDispute(db, matchID, result.Reason)

	case !hasA && hasB:
		result = AutoResolveResult{Resolved: true, Winner: "b",
			Reason: "Party B submitted proof; Party A did not."}
		penalise(db, reqA.UserID, matchID)
		result.PenaltyApplied = true
		closeDispute(db, matchID, result.Reason)

	case hasA && hasB:
		result = resolveByMetadata(db, &settlement, &reqA, &reqB, matchID)

	default:
		result = AutoResolveResult{
			Resolved: false,
			Winner:   "inconclusive",
			Reason:   "Neither party submitted proof. Escalating to manual review.",
		}
	}

	slog.Info("exchange: auto-resolve",
		"match_id", matchID, "resolved", result.Resolved, "winner", result.Winner)
	return result
}

// resolveByMetadata compares timestamps and amounts when both parties uploaded.
func resolveByMetadata(db *gorm.DB, s *ExchangeSettlement, reqA, reqB *ExchangeRequest, matchID uuid.UUID) AutoResolveResult {
	var winnerID, loserID uuid.UUID
	var reason string

	// Rule 1 — earlier timestamp is more credible (>5 min gap required)
	if s.ProofAAt != nil && s.ProofBAt != nil {
		diff := s.ProofAAt.Sub(*s.ProofBAt)
		switch {
		case diff < -5*time.Minute:
			winnerID, loserID = reqA.UserID, reqB.UserID
			reason = "Party A submitted proof significantly earlier."
		case diff > 5*time.Minute:
			winnerID, loserID = reqB.UserID, reqA.UserID
			reason = "Party B submitted proof significantly earlier."
		}
	}

	// Rule 2 — claimed amount cross-check (within 5% tolerance)
	if winnerID == uuid.Nil && s.UserAAmount != nil && s.UserBAmount != nil {
		aOK := approxEqual(*s.UserAAmount, reqB.Amount, 0.05)
		bOK := approxEqual(*s.UserBAmount, reqA.Amount, 0.05)
		switch {
		case aOK && !bOK:
			winnerID, loserID = reqA.UserID, reqB.UserID
			reason = "Party A's claimed amount matches the agreed exchange amount."
		case bOK && !aOK:
			winnerID, loserID = reqB.UserID, reqA.UserID
			reason = "Party B's claimed amount matches the agreed exchange amount."
		}
	}

	if winnerID == uuid.Nil {
		return AutoResolveResult{
			Resolved: false,
			Winner:   "inconclusive",
			Reason:   "Both parties submitted conflicting proofs. Escalating to manual review.",
		}
	}

	penalise(db, loserID, matchID)
	closeDispute(db, matchID, reason)
	side := "a"
	if winnerID == reqB.UserID {
		side = "b"
	}
	return AutoResolveResult{Resolved: true, Winner: side, Reason: reason, PenaltyApplied: true}
}

func penalise(db *gorm.DB, userID, matchID uuid.UUID) {
	_ = reputation.ApplyScoreDelta(db, userID, "buyer", -10, "exchange dispute loss")
	_ = reputation.ApplyScoreDelta(db, userID, "seller", -10, "exchange dispute loss")
	FlagFakeProof(db, matchID, userID, "auto-resolved: dispute loss")
}

func closeDispute(db *gorm.DB, matchID uuid.UUID, resolution string) {
	db.Model(&ExchangeDispute{}).
		Where("match_id=? AND status=?", matchID, DisputeOpen).
		Updates(map[string]interface{}{"status": DisputeResolved, "resolution": resolution})
}

func approxEqual(a, b, tolerance float64) bool {
	if b == 0 {
		return a == 0
	}
	diff := (a - b) / b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
