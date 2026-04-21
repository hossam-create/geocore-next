package exchange

// settlement.go — Sprint 19 Settlement Orchestrator.
//
// Flow:
//   1. Each party uploads a payment receipt (image URL / text reference).
//   2. Platform admin or auto-verification marks each proof as verified.
//   3. When BOTH are verified → settlement moves to RELEASED, match to SETTLED.
//
// HARD CONSTRAINTS:
//   - NO wallet.HoldFunds
//   - NO escrow
//   - NO balance updates
//   - Platform only records and checks proofs

import (
	"errors"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/invite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrSettlementNotFound = errors.New("settlement: record not found")
	ErrAlreadyVerified    = errors.New("settlement: proof already verified")
	ErrMatchNotFound      = errors.New("settlement: match not found")
	ErrProofMissing       = errors.New("settlement: proof URL is required")
)

// UploadProof records a payment proof for the given user on the settlement.
// userID must be either User A or User B in the matched pair.
func UploadProof(db *gorm.DB, matchID, userID uuid.UUID, proofURL string, amount *float64) (*ExchangeSettlement, error) {
	if proofURL == "" {
		return nil, ErrProofMissing
	}

	// Load the match to identify which side the user is on.
	var match ExchangeMatch
	if err := db.First(&match, "id = ?", matchID).Error; err != nil {
		return nil, ErrMatchNotFound
	}

	// Load request owners to determine A vs B.
	var reqA, reqB ExchangeRequest
	if err := db.First(&reqA, "id = ?", match.RequestAID).Error; err != nil {
		return nil, err
	}
	if err := db.First(&reqB, "id = ?", match.RequestBID).Error; err != nil {
		return nil, err
	}

	var settlement ExchangeSettlement
	if err := db.First(&settlement, "match_id = ?", matchID).Error; err != nil {
		return nil, ErrSettlementNotFound
	}

	now := time.Now()
	switch {
	case userID == reqA.UserID:
		if settlement.UserAProof != "" {
			return nil, ErrAlreadyVerified
		}
		settlement.UserAProof = proofURL
		settlement.UserAAmount = amount
		settlement.ProofAAt = &now
	case userID == reqB.UserID:
		if settlement.UserBProof != "" {
			return nil, ErrAlreadyVerified
		}
		settlement.UserBProof = proofURL
		settlement.UserBAmount = amount
		settlement.ProofBAt = &now
	default:
		return nil, errors.New("settlement: user is not a participant in this match")
	}

	if err := db.Save(&settlement).Error; err != nil {
		return nil, err
	}

	slog.Info("exchange: proof uploaded", "match_id", matchID, "user_id", userID)
	return &settlement, nil
}

// VerifyProof marks one side's proof as verified (by admin or automated check).
// side must be "a" or "b".
func VerifyProof(db *gorm.DB, matchID uuid.UUID, side string) (*ExchangeSettlement, error) {
	var settlement ExchangeSettlement
	if err := db.First(&settlement, "match_id = ?", matchID).Error; err != nil {
		return nil, ErrSettlementNotFound
	}

	switch side {
	case "a":
		settlement.VerifiedA = true
	case "b":
		settlement.VerifiedB = true
	default:
		return nil, errors.New("settlement: side must be 'a' or 'b'")
	}

	// If both sides verified → release settlement and complete match.
	if settlement.VerifiedA && settlement.VerifiedB {
		settlement.Status = SettlementReleased
		if err := db.Save(&settlement).Error; err != nil {
			return nil, err
		}
		if err := completeMatch(db, matchID); err != nil {
			return nil, err
		}
		// Grant delayed referral rewards for both participants.
		var match ExchangeMatch
		if db.First(&match, "id = ?", matchID).Error == nil {
			var reqA, reqB ExchangeRequest
			db.First(&reqA, "id = ?", match.RequestAID)
			db.First(&reqB, "id = ?", match.RequestBID)
			go grantReferralReward(db, reqA.UserID)
			go grantReferralReward(db, reqB.UserID)
		}
		slog.Info("exchange: settlement released", "match_id", matchID)
		return &settlement, nil
	}

	// One side still pending.
	settlement.Status = SettlementVerified
	if err := db.Save(&settlement).Error; err != nil {
		return nil, err
	}
	return &settlement, nil
}

// completeMatch marks the ExchangeMatch and both ExchangeRequests as completed.
func completeMatch(db *gorm.DB, matchID uuid.UUID) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&ExchangeMatch{}).
			Where("id = ?", matchID).
			Update("status", MatchSettled).Error; err != nil {
			return err
		}
		// Mark both underlying requests as COMPLETED.
		var match ExchangeMatch
		if err := tx.First(&match, "id = ?", matchID).Error; err != nil {
			return err
		}
		return tx.Model(&ExchangeRequest{}).
			Where("id IN ?", []uuid.UUID{match.RequestAID, match.RequestBID}).
			Update("status", StatusCompleted).Error
	})
}

// OpenDispute raises a dispute for a match.
func OpenDispute(db *gorm.DB, matchID, raisedBy uuid.UUID, reason string) (*ExchangeDispute, error) {
	dispute := ExchangeDispute{
		ID:       uuid.New(),
		MatchID:  matchID,
		RaisedBy: raisedBy,
		Reason:   reason,
		Status:   DisputeOpen,
	}
	if err := db.Create(&dispute).Error; err != nil {
		return nil, err
	}
	// Move match to DISPUTED.
	db.Model(&ExchangeMatch{}).Where("id = ?", matchID).Update("status", MatchDisputed)
	slog.Warn("exchange: dispute opened", "match_id", matchID, "raised_by", raisedBy)
	return &dispute, nil
}

// grantReferralReward fires invite.GrantReward for a user if a pending reward exists.
func grantReferralReward(db *gorm.DB, userID uuid.UUID) {
	if err := invite.GrantReward(db, userID); err != nil {
		slog.Warn("exchange: referral reward grant failed", "user_id", userID, "error", err)
	}
}

// ResolveDispute sets the resolution and closes the dispute.
func ResolveDispute(db *gorm.DB, disputeID uuid.UUID, resolution string) (*ExchangeDispute, error) {
	var dispute ExchangeDispute
	if err := db.First(&dispute, "id = ?", disputeID).Error; err != nil {
		return nil, err
	}
	dispute.Resolution = resolution
	dispute.Status = DisputeResolved
	if err := db.Save(&dispute).Error; err != nil {
		return nil, err
	}
	return &dispute, nil
}
