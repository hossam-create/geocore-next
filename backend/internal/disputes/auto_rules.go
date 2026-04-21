package disputes

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/internal/reputation"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AutoResolve attempts automatic dispute resolution based on evidence.
func AutoResolve(db *gorm.DB, notifSvc *notifications.Service, disputeID uuid.UUID) (*ResolutionType, error) {
	var d Dispute
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("Evidence").
		Where("id=? AND status=?", disputeID, StatusUnderReview).
		First(&d).Error; err != nil {
		return nil, fmt.Errorf("dispute not eligible for auto-resolve")
	}

	// Count evidence by type and submitter
	buyerEvidence := 0
	sellerEvidence := 0
	deliveryProof := 0
	for _, e := range d.Evidence {
		if e.SubmittedBy == d.BuyerID {
			buyerEvidence++
		} else {
			sellerEvidence++
		}
		if e.Type == "tracking" || e.Type == "delivery_confirmation" {
			deliveryProof++
		}
	}

	var resolution ResolutionType
	var resolutionNotes string

	switch {
	// No delivery proof + buyer has evidence → refund
	case deliveryProof == 0 && buyerEvidence > 0:
		resolution = ResolutionFullRefund
		resolutionNotes = "Auto-resolved: no delivery proof provided, buyer evidence present"

	// Delivery confirmed + no buyer evidence → release to seller
	case deliveryProof > 0 && buyerEvidence == 0:
		resolution = ResolutionNoRefund
		resolutionNotes = "Auto-resolved: delivery confirmed, no buyer evidence"

	// Both sides have evidence → needs manual review
	default:
		slog.Info("disputes: auto-resolve inconclusive, requires manual review", "dispute_id", disputeID)
		return nil, nil // no auto-resolution possible
	}

	// Apply resolution
	now := time.Now()
	d.Status = StatusResolved
	d.Resolution = &resolution
	d.ResolutionNotes = resolutionNotes
	d.ResolvedAt = &now
	db.Save(&d)

	// Apply reputation penalties
	applyDisputeResolutionPenalties(db, d, resolution)

	// Notify
	if notifSvc != nil {
		go notifSvc.Notify(notifications.NotifyInput{
			UserID: d.BuyerID, Type: "dispute_resolved",
			Title: "Dispute Resolved",
			Body:  fmt.Sprintf("Your dispute has been resolved: %s", string(resolution)),
			Data:  map[string]string{"dispute_id": d.ID.String(), "resolution": string(resolution)},
		})
		go notifSvc.Notify(notifications.NotifyInput{
			UserID: d.SellerID, Type: "dispute_resolved",
			Title: "Dispute Resolved",
			Body:  fmt.Sprintf("Dispute on your order has been resolved: %s", string(resolution)),
			Data:  map[string]string{"dispute_id": d.ID.String(), "resolution": string(resolution)},
		})
	}

	slog.Info("disputes: auto-resolved", "dispute_id", disputeID, "resolution", resolution)
	return &resolution, nil
}

// FreezeEscrowOnDispute freezes the escrow associated with a dispute.
func FreezeEscrowOnDispute(db *gorm.DB, disputeID uuid.UUID) error {
	var d Dispute
	if err := db.Where("id=?", disputeID).First(&d).Error; err != nil {
		return err
	}
	if d.EscrowID == nil {
		return nil
	}
	// Mark escrow as frozen (dispute hold)
	return db.Table("escrows").Where("id=?", *d.EscrowID).
		Update("status", "disputed").Error
}

// CanOpenDispute checks if a buyer is eligible to open a dispute.
// Requires delivery confirmation photo or tracking evidence.
func CanOpenDispute(db *gorm.DB, orderID uuid.UUID, buyerID uuid.UUID) error {
	// Check order exists and belongs to buyer
	var order struct {
		Status      string
		DeliveredAt *time.Time
		SellerID    uuid.UUID
	}
	if err := db.Table("orders").
		Where("id=? AND buyer_id=?", orderID, buyerID).
		First(&order).Error; err != nil {
		return fmt.Errorf("order not found or not yours")
	}

	// Must be delivered to dispute
	if order.Status != "delivered" && order.Status != "completed" {
		return fmt.Errorf("can only dispute delivered orders")
	}

	// Must have delivery confirmation (photo or tracking)
	var evidenceCount int64
	db.Table("order_evidence").
		Where("order_id=? AND type IN ?", orderID, []string{"delivery_photo", "tracking_confirmation"}).
		Count(&evidenceCount)
	if evidenceCount == 0 {
		return fmt.Errorf("delivery confirmation required before opening dispute")
	}

	return nil
}

// AutoReleaseEscrowAfter24h checks if escrow should be auto-released.
// Rule: if delivery confirmed AND no dispute within 24h → auto-release to seller.
func AutoReleaseEscrowAfter24h(db *gorm.DB) int {
	released := 0

	// Find delivered orders with escrow still held, older than 24h, no open dispute
	type eligibleOrder struct {
		OrderID  uuid.UUID
		EscrowID uuid.UUID
		SellerID uuid.UUID
	}
	var orders []eligibleOrder

	db.Table("orders o").
		Select("o.id as order_id, e.id as escrow_id, o.seller_id").
		Joins("JOIN escrows e ON e.order_id = o.id").
		Where("o.status IN ?", []string{"delivered", "completed"}).
		Where("e.status = ?", "held").
		Where("o.delivered_at < ?", time.Now().Add(-24*time.Hour)).
		Where("NOT EXISTS (SELECT 1 FROM disputes d WHERE d.order_id = o.id AND d.status IN ?)", []string{"open", "under_review", "awaiting_response"}).
		Scan(&orders)

	for _, o := range orders {
		// Release escrow
		if err := db.Table("escrows").Where("id=?", o.EscrowID).Update("status", "released").Error; err != nil {
			slog.Error("disputes: auto-release failed", "escrow_id", o.EscrowID, "error", err)
			continue
		}
		released++
		slog.Info("disputes: auto-released escrow after 24h", "order_id", o.OrderID, "escrow_id", o.EscrowID)

		// Apply delivery bonus to seller reputation
		_ = reputation.ApplyScoreDelta(db, o.SellerID, "seller", 5, "successful_delivery_auto_release")
	}

	return released
}

// applyDisputeResolutionPenalties applies reputation changes based on resolution.
func applyDisputeResolutionPenalties(db *gorm.DB, d Dispute, resolution ResolutionType) {
	switch resolution {
	case ResolutionFullRefund:
		// Seller lost dispute
		_ = reputation.ApplyScoreDelta(db, d.SellerID, "seller", -20, "dispute_lost")
	case ResolutionNoRefund:
		// Buyer lost dispute (frivolous)
		_ = reputation.ApplyScoreDelta(db, d.BuyerID, "buyer", -10, "dispute_lost_buyer")
	case ResolutionPartialRefund:
		// Split penalty
		_ = reputation.ApplyScoreDelta(db, d.SellerID, "seller", -10, "dispute_partial")
		_ = reputation.ApplyScoreDelta(db, d.BuyerID, "buyer", -5, "dispute_partial")
	}
}
