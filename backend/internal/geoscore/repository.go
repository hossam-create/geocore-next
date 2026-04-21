package geoscore

import (
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository gathers signals from existing tables and persists GeoScore records.
// It uses raw SQL to avoid importing other internal packages (prevents import cycles).
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a geoscore repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// GatherSignals collects the raw metrics needed to compute a GeoScore.
func (r *Repository) GatherSignals(userID uuid.UUID) Input {
	uid := userID.String()

	// ── Success rate (delivered / total non-cancelled orders as seller) ──────
	var orderStats struct {
		Total     int64
		Delivered int64
	}
	r.db.Raw(`
		SELECT
			COUNT(*) FILTER (WHERE status NOT IN ('cancelled','pending')) AS total,
			COUNT(*) FILTER (WHERE status IN ('delivered','completed'))  AS delivered
		FROM orders
		WHERE (seller_id = ? OR buyer_id = ?)
		  AND deleted_at IS NULL`, uid, uid).Scan(&orderStats)

	successRate := 0.0
	if orderStats.Total > 0 {
		successRate = float64(orderStats.Delivered) / float64(orderStats.Total)
	}

	// ── Dispute rate ──────────────────────────────────────────────────────────
	var disputeCount int64
	r.db.Raw(`
		SELECT COUNT(*) FROM disputes
		WHERE (complainant_id = ? OR respondent_id = ?)
		  AND status NOT IN ('cancelled','closed_no_action')`, uid, uid).Scan(&disputeCount)

	disputeRate := 0.0
	if orderStats.Total > 0 {
		disputeRate = math.Min(1.0, float64(disputeCount)/float64(orderStats.Total))
	}

	// ── KYC score ─────────────────────────────────────────────────────────────
	var kycLevel string
	r.db.Raw(`SELECT COALESCE(verification_level,'none') FROM kyc_profiles WHERE user_id = ? LIMIT 1`, uid).
		Scan(&kycLevel)
	kycScore := map[string]float64{"none": 0.0, "basic": 0.5, "full": 1.0}[kycLevel]

	// ── Delivery score (crowdshipping) ────────────────────────────────────────
	var deliveryStats struct {
		Total     int64
		Completed int64
	}
	r.db.Raw(`
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'completed') AS completed
		FROM delivery_requests
		WHERE requester_id = ? AND status NOT IN ('pending','cancelled')`, uid).
		Scan(&deliveryStats)

	deliveryScore := 1.0 // default to 1 if no history
	if deliveryStats.Total > 0 {
		deliveryScore = float64(deliveryStats.Completed) / float64(deliveryStats.Total)
	}

	// ── Fraud score (from fraud_events if table exists, default 0) ────────────
	var fraudEvents int64
	r.db.Raw(`
		SELECT COUNT(*) FROM fraud_events
		WHERE user_id = ? AND created_at > ?`, uid, time.Now().AddDate(0, -3, 0)).
		Scan(&fraudEvents)
	fraudScore := math.Min(1.0, float64(fraudEvents)/10.0) // cap at 10 events

	return Input{
		SuccessRate:   successRate,
		DisputeRate:   disputeRate,
		KYCScore:      kycScore,
		DeliveryScore: deliveryScore,
		FraudScore:    fraudScore,
	}
}

// Save upserts a GeoScore record.
func (r *Repository) Save(gs *GeoScore) error {
	gs.UpdatedAt = time.Now()
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"score", "success_rate", "dispute_rate", "kyc_score", "delivery_score", "fraud_score", "updated_at"}),
	}).Create(gs).Error
}

// Get returns the cached GeoScore for a user or nil if not found.
func (r *Repository) Get(userID uuid.UUID) (*GeoScore, error) {
	var gs GeoScore
	if err := r.db.Where("user_id = ?", userID).First(&gs).Error; err != nil {
		return nil, err
	}
	return &gs, nil
}

// SaveBehaviorEvent inserts a behavior event.
func (r *Repository) SaveBehaviorEvent(evt *BehaviorEvent) error {
	if evt.ID == uuid.Nil {
		evt.ID = uuid.New()
	}
	if evt.CreatedAt.IsZero() {
		evt.CreatedAt = time.Now()
	}
	if err := r.db.Create(evt).Error; err != nil {
		slog.Warn("geoscore: behavior event insert failed", "error", err)
		return err
	}
	return nil
}
