package protection

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Claim Filing ────────────────────────────────────────────────────────────────

type FileClaimRequest struct {
	Type         ClaimType `json:"type" binding:"required"`
	EvidenceJSON string    `json:"evidence_json"`
}

// FileClaim creates a guarantee claim and runs auto-evaluation.
func FileClaim(db *gorm.DB, orderID, userID uuid.UUID, req FileClaimRequest) (*GuaranteeClaim, error) {
	// 1. Load order
	var ord struct {
		ID         uuid.UUID
		BuyerID    uuid.UUID
		SellerID   uuid.UUID
		Status     string
		Total      float64
		ShippedAt  *time.Time
		DeliveredAt *time.Time
		CreatedAt  time.Time
	}
	if err := db.Table("orders").
		Select("id, buyer_id, seller_id, status, total, shipped_at, delivered_at, created_at").
		Where("id = ?", orderID).First(&ord).Error; err != nil {
		return nil, fmt.Errorf("order not found")
	}
	if ord.BuyerID != userID {
		return nil, fmt.Errorf("only the buyer can file a claim")
	}

	// 2. Check if order has protection
	var protection OrderProtection
	if err := db.Where("order_id = ? AND is_used = ?", orderID, false).First(&protection).Error; err != nil {
		return nil, fmt.Errorf("no active protection found for this order")
	}

	// 3. Validate claim type against protection
	switch req.Type {
	case ClaimNoShow, ClaimMismatch:
		if !protection.HasFull && !protection.HasCancellation {
			return nil, fmt.Errorf("your protection does not cover %s claims", req.Type)
		}
	case ClaimDelay:
		if !protection.HasDelay && !protection.HasFull {
			return nil, fmt.Errorf("your protection does not cover delay claims")
		}
	}

	// 4. Check for duplicate claims
	var existingCount int64
	db.Model(&GuaranteeClaim{}).Where("order_id = ? AND type = ? AND status IN ?",
		orderID, req.Type, []ClaimStatus{ClaimPending, ClaimAutoApproved, ClaimApproved}).Count(&existingCount)
	if existingCount > 0 {
		return nil, fmt.Errorf("a %s claim already exists for this order", req.Type)
	}

	// 5. Create the claim
	claim := GuaranteeClaim{
		OrderID:       orderID,
		UserID:        userID,
		TravelerID:    ord.SellerID,
		Type:          req.Type,
		EvidenceJSON:  req.EvidenceJSON,
		Status:        ClaimPending,
		AutoEvaluated: false,
	}
	if claim.EvidenceJSON == "" {
		claim.EvidenceJSON = "{}"
	}
	if err := db.Create(&claim).Error; err != nil {
		return nil, fmt.Errorf("claim creation failed: %w", err)
	}

	// 6. Run auto-evaluation
	evaluation := AutoEvaluateClaim(db, &claim, &ord)

	// 7. Apply evaluation result
	if evaluation.Decision == ClaimAutoApproved {
		claim.Status = ClaimAutoApproved
		claim.RefundCents = int64(float64(int64(ord.Total*100)) * evaluation.RefundPercent / 100.0)
		claim.CompensationCents = evaluation.CompensationCents
		claim.TravelerPenalty = evaluation.TravelerPenalty
		claim.AutoEvaluated = true
		now := time.Now()
		claim.ResolvedAt = &now

		db.Save(&claim)

		// Mark protection as used
		db.Model(&protection).Update("is_used", true)

		// Process refund
		processClaimRefund(db, &claim, userID)
	}
	// If not auto-approved, stays as pending for manual review

	return &claim, nil
}

// ── Auto Evaluation Engine ──────────────────────────────────────────────────────

// AutoEvaluateClaim applies rules-based evaluation with fallback to manual review.
func AutoEvaluateClaim(db *gorm.DB, claim *GuaranteeClaim, ord interface{}) *ClaimEvaluation {
	eval := &ClaimEvaluation{
		Decision:      ClaimPending, // default: needs manual review
		AutoEvaluated: true,
	}

	// Parse evidence
	var evidence map[string]interface{}
	_ = json.Unmarshal([]byte(claim.EvidenceJSON), &evidence)

	switch claim.Type {
	case ClaimNoShow:
		eval = evaluateNoShow(db, claim, evidence)
	case ClaimDelay:
		eval = evaluateDelay(db, claim, evidence)
	case ClaimMismatch:
		eval = evaluateMismatch(db, claim, evidence)
	}

	return eval
}

// evaluateNoShow checks if traveler didn't show up.
func evaluateNoShow(db *gorm.DB, claim *GuaranteeClaim, evidence map[string]interface{}) *ClaimEvaluation {
	eval := &ClaimEvaluation{AutoEvaluated: true}

	// Rule 1: If order was never shipped → likely no-show
	type orderInfo struct {
		ShippedAt   *time.Time
		DeliveredAt *time.Time
		Status      string
	}
	var ord orderInfo
	db.Table("orders").Select("shipped_at, delivered_at, status").Where("id = ?", claim.OrderID).Scan(&ord)

	if ord.ShippedAt == nil && ord.Status != "shipped" && ord.Status != "delivered" {
		// Order was never shipped — strong no-show signal
		eval.Decision = ClaimAutoApproved
		eval.RefundPercent = 100
		eval.CompensationCents = 0
		eval.TravelerPenalty = true
		eval.Reason = "Order never shipped — traveler no-show"
		return eval
	}

	// Rule 2: Check if delivery request has tracking data
	var trackingCount int64
	db.Table("tracking_updates").Where("order_id = ?", claim.OrderID).Count(&trackingCount)
	if trackingCount == 0 && ord.ShippedAt == nil {
		eval.Decision = ClaimAutoApproved
		eval.RefundPercent = 100
		eval.TravelerPenalty = true
		eval.Reason = "No tracking data — traveler no-show"
		return eval
	}

	// Cannot auto-approve — needs manual review with evidence
	eval.Decision = ClaimPending
	eval.Reason = "Requires manual review — evidence needed"
	return eval
}

// evaluateDelay checks if delivery was late beyond SLA.
func evaluateDelay(db *gorm.DB, claim *GuaranteeClaim, evidence map[string]interface{}) *ClaimEvaluation {
	eval := &ClaimEvaluation{AutoEvaluated: true}

	// Load order and delivery request timelines
	var ord struct {
		ShippedAt    *time.Time
		DeliveredAt  *time.Time
		CreatedAt    time.Time
	}
	db.Table("orders").Select("shipped_at, delivered_at, created_at").
		Where("id = ?", claim.OrderID).Scan(&ord)

	var dr struct {
		Deadline *time.Time
	}
	db.Table("delivery_requests").
		Select("deadline").
		Where("buyer_id = (SELECT buyer_id FROM orders WHERE id = ?)", claim.OrderID).
		Scan(&dr)

	if dr.Deadline != nil && ord.DeliveredAt != nil {
		delayDuration := ord.DeliveredAt.Sub(*dr.Deadline)
		if delayDuration > 0 {
			// Delivery was late
			hoursLate := delayDuration.Hours()
			switch {
			case hoursLate > 48:
				eval.Decision = ClaimAutoApproved
				eval.RefundPercent = 50
				eval.CompensationCents = 500 // 5 EGP compensation
				eval.TravelerPenalty = true
				eval.Reason = fmt.Sprintf("Delivery %.0f hours late — auto-approved", hoursLate)
			case hoursLate > 24:
				eval.Decision = ClaimAutoApproved
				eval.RefundPercent = 25
				eval.CompensationCents = 200 // 2 EGP compensation
				eval.TravelerPenalty = false
				eval.Reason = fmt.Sprintf("Delivery %.0f hours late — partial refund", hoursLate)
			default:
				eval.Decision = ClaimAutoApproved
				eval.RefundPercent = 10
				eval.CompensationCents = 0
				eval.TravelerPenalty = false
				eval.Reason = fmt.Sprintf("Delivery %.0f hours late — minor delay compensation", hoursLate)
			}
			return eval
		}
	}

	// No deadline or not yet delivered — needs manual review
	eval.Decision = ClaimPending
	eval.Reason = "Cannot auto-evaluate — deadline or delivery time missing"
	return eval
}

// evaluateMismatch checks if delivered item doesn't match description.
func evaluateMismatch(db *gorm.DB, claim *GuaranteeClaim, evidence map[string]interface{}) *ClaimEvaluation {
	eval := &ClaimEvaluation{AutoEvaluated: true}

	// Mismatch claims always require manual review (needs photo evidence)
	// But we can pre-score based on evidence strength
	photoCount := 0
	if urls, ok := evidence["photo_urls"].([]interface{}); ok {
		photoCount = len(urls)
	}

	if photoCount >= 3 {
		// Strong evidence — still needs manual review but flagged as high-confidence
		eval.Decision = ClaimPending
		eval.Reason = "Strong evidence — priority manual review recommended"
	} else if photoCount >= 1 {
		eval.Decision = ClaimPending
		eval.Reason = "Some evidence — standard manual review"
	} else {
		eval.Decision = ClaimPending
		eval.Reason = "No photo evidence — request evidence from buyer"
	}

	return eval
}

// ── Manual Review ──────────────────────────────────────────────────────────────

type ReviewClaimRequest struct {
	Decision          ClaimStatus `json:"decision" binding:"required"`
	RefundPercent     float64     `json:"refund_percent"`
	CompensationCents int64       `json:"compensation_cents"`
	TravelerPenalty   bool        `json:"traveler_penalty"`
}

// ReviewClaim allows an admin to manually review and resolve a claim.
func ReviewClaim(db *gorm.DB, claimID, reviewerID uuid.UUID, req ReviewClaimRequest) (*GuaranteeClaim, error) {
	var claim GuaranteeClaim
	if err := db.First(&claim, "id = ?", claimID).Error; err != nil {
		return nil, fmt.Errorf("claim not found")
	}
	if claim.Status != ClaimPending {
		return nil, fmt.Errorf("claim is not pending review")
	}

	// Load order for refund calculation
	var ord struct{ Total float64 }
	db.Table("orders").Select("total").Where("id = ?", claim.OrderID).Scan(&ord)

	claim.Status = req.Decision
	claim.RefundCents = int64(float64(int64(ord.Total*100)) * req.RefundPercent / 100.0)
	claim.CompensationCents = req.CompensationCents
	claim.TravelerPenalty = req.TravelerPenalty
	claim.ReviewerID = &reviewerID
	now := time.Now()
	claim.ResolvedAt = &now

	if err := db.Save(&claim).Error; err != nil {
		return nil, fmt.Errorf("failed to update claim")
	}

	// If approved, process refund and mark protection used
	if req.Decision == ClaimApproved {
		db.Model(&OrderProtection{}).Where("order_id = ?", claim.OrderID).Update("is_used", true)
		processClaimRefund(db, &claim, claim.UserID)

		// Apply traveler penalty if flagged
		if req.TravelerPenalty {
			applyTravelerPenalty(db, claim.TravelerID)
		}
	}

	return &claim, nil
}

// processClaimRefund credits the buyer's wallet with the refund amount.
func processClaimRefund(db *gorm.DB, claim *GuaranteeClaim, buyerID uuid.UUID) {
	totalAmount := float64(claim.RefundCents+claim.CompensationCents) / 100.0
	if totalAmount <= 0 {
		return
	}

	var walletID string
	db.Table("wallets").Select("id").Where("user_id = ?", buyerID).Scan(&walletID)
	if walletID == "" {
		return
	}

	db.Table("wallet_balances").
		Where("wallet_id = ? AND currency = ?", walletID, "AED").
		Updates(map[string]interface{}{
			"available_balance": gorm.Expr("available_balance + ?", totalAmount),
			"balance":          gorm.Expr("balance + ?", totalAmount),
		})

	db.Table("wallet_transactions").Create(map[string]interface{}{
		"wallet_id":      walletID,
		"type":           "refund",
		"amount":         totalAmount,
		"currency":       "AED",
		"status":         "completed",
		"reference_id":   fmt.Sprintf("claim:%s", claim.ID),
		"reference_type": "guarantee_claim",
		"description":    fmt.Sprintf("Guarantee claim refund for order %s", claim.OrderID),
		"completed_at":   time.Now(),
	})
}

// applyTravelerPenalty reduces the traveler's reputation score.
func applyTravelerPenalty(db *gorm.DB, travelerID uuid.UUID) {
	// Reduce geoscore by 10 points
	db.Table("geo_scores").
		Where("user_id = ?", travelerID).
		Updates(map[string]interface{}{
			"score":     gorm.Expr("GREATEST(score - 10, 0)"),
			"updated_at": time.Now(),
		})
}
