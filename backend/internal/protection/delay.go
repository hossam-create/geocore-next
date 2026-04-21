package protection

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Delay Detection + Auto Compensation ─────────────────────────────────────────

const (
	// DelaySLAHours is the threshold beyond which a delay is considered significant.
	DelaySLAHours = 24

	// DelayCompensationRate is the % of order value compensated per 24h of delay.
	DelayCompensationRate = 5.0 // 5% per 24h late

	// MaxDelayCompensationPercent caps total delay compensation.
	MaxDelayCompensationPercent = 25.0

	// AutoCompensationThreshold triggers automatic credit without claim.
	AutoCompensationThreshold = 48 // hours late
)

// DelayStatus represents the delay state of an order.
type DelayStatus struct {
	OrderID        uuid.UUID `json:"order_id"`
	IsDelayed      bool      `json:"is_delayed"`
	HoursLate      float64   `json:"hours_late"`
	Deadline       *time.Time `json:"deadline,omitempty"`
	EstimatedArrival *time.Time `json:"estimated_arrival,omitempty"`
	CompensationCents int64   `json:"compensation_cents,omitempty"`
}

// CheckDelayStatus checks if an order is delayed and calculates compensation.
func CheckDelayStatus(db *gorm.DB, orderID uuid.UUID) (*DelayStatus, error) {
	status := &DelayStatus{OrderID: orderID}

	// 1. Load order
	var ord struct {
		BuyerID     uuid.UUID
		Status      string
		ShippedAt   *time.Time
		DeliveredAt *time.Time
		CreatedAt   time.Time
	}
	if err := db.Table("orders").
		Select("buyer_id, status, shipped_at, delivered_at, created_at").
		Where("id = ?", orderID).First(&ord).Error; err != nil {
		return nil, fmt.Errorf("order not found")
	}

	// Only check delay for shipped/in-transit orders
	if ord.Status != "shipped" && ord.Status != "processing" {
		status.IsDelayed = false
		return status, nil
	}

	// 2. Load delivery request deadline
	var dr struct {
		Deadline *time.Time
	}
	db.Table("delivery_requests").
		Select("deadline").
		Where("buyer_id = ?", ord.BuyerID).
		Order("created_at DESC").
		Limit(1).
		Scan(&dr)

	status.Deadline = dr.Deadline

	if dr.Deadline == nil {
		// No deadline set — can't determine delay
		status.IsDelayed = false
		return status, nil
	}

	// 3. Calculate delay
	now := time.Now()
	if ord.DeliveredAt != nil {
		// Already delivered — was it late?
		hoursLate := ord.DeliveredAt.Sub(*dr.Deadline).Hours()
		status.HoursLate = hoursLate
		status.IsDelayed = hoursLate > 0
	} else {
		// Still in transit — is it past deadline?
		hoursLate := now.Sub(*dr.Deadline).Hours()
		status.HoursLate = hoursLate
		status.IsDelayed = hoursLate > 0
	}

	// 4. Calculate compensation if delayed
	if status.IsDelayed && status.HoursLate > 0 {
		var ordTotal struct{ Total float64 }
		db.Table("orders").Select("total").Where("id = ?", orderID).Scan(&ordTotal)

		compensationPct := DelayCompensationRate * (status.HoursLate / 24.0)
		if compensationPct > MaxDelayCompensationPercent {
			compensationPct = MaxDelayCompensationPercent
		}
		totalCents := int64(ordTotal.Total * 100)
		status.CompensationCents = int64(float64(totalCents) * compensationPct / 100.0)
	}

	return status, nil
}

// AutoCompensateDelay automatically credits the buyer if delay exceeds threshold.
// This runs without requiring the buyer to file a claim.
func AutoCompensateDelay(db *gorm.DB, orderID uuid.UUID) error {
	status, err := CheckDelayStatus(db, orderID)
	if err != nil {
		return err
	}

	if !status.IsDelayed || status.HoursLate < AutoCompensationThreshold {
		return nil // not delayed enough for auto-compensation
	}

	// Check if order has delay protection
	var protection OrderProtection
	if err := db.Where("order_id = ? AND (has_delay = ? OR has_full = ?) AND is_used = ?",
		orderID, true, true, false).First(&protection).Error; err != nil {
		return nil // no delay protection
	}

	// Check if already auto-compensated
	var existingCount int64
	db.Model(&GuaranteeClaim{}).
		Where("order_id = ? AND type = ? AND auto_evaluated = ?",
			orderID, ClaimDelay, true).Count(&existingCount)
	if existingCount > 0 {
		return nil // already compensated
	}

	// Create auto-approved claim
	claim := GuaranteeClaim{
		OrderID:           orderID,
		UserID:            protection.UserID,
		TravelerID:        getSellerID(db, orderID),
		Type:              ClaimDelay,
		EvidenceJSON:      fmt.Sprintf(`{"auto_detected":true,"hours_late":%.1f}`, status.HoursLate),
		Status:            ClaimAutoApproved,
		RefundCents:       0, // no refund, just compensation
		CompensationCents: status.CompensationCents,
		TravelerPenalty:   status.HoursLate > 72,
		AutoEvaluated:     true,
	}
	now := time.Now()
	claim.ResolvedAt = &now

	if err := db.Create(&claim).Error; err != nil {
		return fmt.Errorf("auto-claim creation failed: %w", err)
	}

	// Credit buyer wallet
	processClaimRefund(db, &claim, protection.UserID)

	// Mark protection as used (for delay portion only — not full use)
	if protection.HasFull {
		// Full protection can still be used for cancellation
		// Don't mark as fully used
	} else {
		db.Model(&protection).Update("is_used", true)
	}

	return nil
}

// getSellerID retrieves the seller ID for an order.
func getSellerID(db *gorm.DB, orderID uuid.UUID) uuid.UUID {
	var sellerID uuid.UUID
	db.Table("orders").Select("seller_id").Where("id = ?", orderID).Scan(&sellerID)
	return sellerID
}

// ScanDelayedOrders scans all in-transit orders and auto-compensates where needed.
// Should be called by a periodic job (e.g., every hour).
func ScanDelayedOrders(db *gorm.DB) error {
	var orderIDs []uuid.UUID
	db.Table("orders").
		Select("id").
		Where("status IN ? AND shipped_at IS NOT NULL", []string{"shipped", "processing"}).
		Scan(&orderIDs)

	for _, id := range orderIDs {
		_ = AutoCompensateDelay(db, id) // errors are non-critical
	}
	return nil
}
