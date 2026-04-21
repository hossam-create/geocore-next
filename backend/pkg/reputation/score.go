package reputation

import (
	"log/slog"
	"math"
	"time"

	"gorm.io/gorm"
)

// TrustLevel represents a user's trust tier.
type TrustLevel string

const (
	TrustBasic   TrustLevel = "basic"
	TrustVerified TrustLevel = "verified"
	TrustTrusted  TrustLevel = "trusted"
)

// Profile is a computed reputation snapshot for a user.
// Stored in user_reputations and refreshed on demand or periodically.
type Profile struct {
	UserID          string     `gorm:"type:uuid;primaryKey" json:"user_id"`
	Score           float64    `gorm:"not null;default:0" json:"score"`          // 0–100
	TrustLevel      TrustLevel `gorm:"size:20;default:'basic'" json:"trust_level"`
	AvgRating       float64    `gorm:"default:0" json:"avg_rating"`               // 1–5
	ReviewCount     int        `gorm:"default:0" json:"review_count"`
	CompletedOrders int        `gorm:"default:0" json:"completed_orders"`
	DisputeCount    int        `gorm:"default:0" json:"dispute_count"`
	DisputeRatio    float64    `gorm:"default:0" json:"dispute_ratio"`            // 0–1
	KYCLevel        int        `gorm:"default:0" json:"kyc_level"`               // 0=none,1=basic,2=full
	EmailVerified   bool       `gorm:"default:false" json:"email_verified"`
	ComputedAt      time.Time  `json:"computed_at"`
}

func (Profile) TableName() string { return "user_reputations" }

// Compute calculates the reputation score from raw signals.
//
// Formula (max 100):
//
//	Rating component  = (avg_rating / 5) * 40         → 0–40
//	Orders component  = min(completed_orders, 30)     → 0–30
//	KYC component     = kyc_level * 10                → 0–20
//	Dispute component = (1 - dispute_ratio) * 10      → 0–10
func Compute(
	avgRating float64,
	reviewCount int,
	completedOrders int,
	disputeCount int,
	kycLevel int,
	emailVerified bool,
) Profile {
	// Rating component (only meaningful when there are reviews)
	ratingComp := 0.0
	if reviewCount > 0 {
		ratingComp = (avgRating / 5.0) * 40.0
	}

	// Orders component — capped at 30 to prevent gaming
	orderComp := math.Min(float64(completedOrders), 30.0)

	// KYC component
	kycComp := math.Min(float64(kycLevel)*10.0, 20.0)

	// Dispute ratio (lower is better)
	disputeRatio := 0.0
	if completedOrders > 0 {
		disputeRatio = math.Min(1.0, float64(disputeCount)/float64(completedOrders))
	}
	disputeComp := (1.0 - disputeRatio) * 10.0

	score := math.Min(100.0, ratingComp+orderComp+kycComp+disputeComp)

	trustLevel := TrustBasic
	switch {
	case score >= 70 && kycLevel >= 1:
		trustLevel = TrustTrusted
	case score >= 40 || emailVerified:
		trustLevel = TrustVerified
	}

	return Profile{
		Score:           score,
		TrustLevel:      trustLevel,
		AvgRating:       avgRating,
		ReviewCount:     reviewCount,
		CompletedOrders: completedOrders,
		DisputeCount:    disputeCount,
		DisputeRatio:    disputeRatio,
		KYCLevel:        kycLevel,
		EmailVerified:   emailVerified,
		ComputedAt:      time.Now(),
	}
}

// Refresh re-computes and upserts the reputation profile for the given user.
func Refresh(db *gorm.DB, userID string) (*Profile, error) {
	// Gather signals from DB
	var signals struct {
		AvgRating       float64
		ReviewCount     int
		CompletedOrders int
		DisputeCount    int
		KYCLevel        int
		EmailVerified   bool
	}

	// Reviews
	db.Raw(`SELECT COALESCE(AVG(rating),0) as avg_rating, COUNT(*) as review_count
		FROM reviews WHERE seller_id = ? AND deleted_at IS NULL`, userID).
		Scan(&signals)

	// Completed orders (as buyer or seller)
	db.Raw(`SELECT COUNT(*) as completed_orders FROM orders
		WHERE (buyer_id = ? OR seller_id = ?) AND status = 'delivered'`, userID, userID).
		Scan(&signals)

	// Disputes
	db.Raw(`SELECT COUNT(*) as dispute_count FROM disputes
		WHERE (complainant_id = ? OR respondent_id = ?) AND status NOT IN ('cancelled','withdrawn')`,
		userID, userID).Scan(&signals)

	// KYC
	db.Raw(`SELECT COALESCE(
		CASE verification_level WHEN 'full' THEN 2 WHEN 'basic' THEN 1 ELSE 0 END, 0
	) as kyc_level FROM kyc_profiles WHERE user_id = ? LIMIT 1`, userID).Scan(&signals)

	// Email verified
	db.Raw(`SELECT email_verified FROM users WHERE id = ?`, userID).
		Scan(&signals)

	profile := Compute(
		signals.AvgRating,
		signals.ReviewCount,
		signals.CompletedOrders,
		signals.DisputeCount,
		signals.KYCLevel,
		signals.EmailVerified,
	)
	profile.UserID = userID

	// Upsert
	if err := db.Save(&profile).Error; err != nil {
		slog.Warn("reputation: upsert failed", "user_id", userID, "error", err)
		return &profile, err
	}
	return &profile, nil
}

// Get returns the cached reputation profile, refreshing if stale (> 1h).
func Get(db *gorm.DB, userID string) (*Profile, error) {
	var profile Profile
	if err := db.Where("user_id = ?", userID).First(&profile).Error; err != nil || time.Since(profile.ComputedAt) > time.Hour {
		return Refresh(db, userID)
	}
	return &profile, nil
}
