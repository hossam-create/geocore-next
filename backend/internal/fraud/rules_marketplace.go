package fraud

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MarketplaceRiskResult is the output of marketplace-specific fraud checks.
type MarketplaceRiskResult struct {
	Score  int      `json:"score"`  // 0–100
	Flags  []string `json:"flags"`  // detected flags
	Action string   `json:"action"` // allow, review, block
}

// MarketplaceCheckInput contains data for marketplace-specific fraud rules.
type MarketplaceCheckInput struct {
	UserID    uuid.UUID
	IP        string
	DeviceID  string
	WalletID  string
	EventType string // "offer", "listing", "withdraw", "boost"
	Amount    float64
	Price     float64 // listing/offer price
	MarketAvg float64 // market average price for category
}

// CheckMarketplaceRules evaluates marketplace-specific fraud patterns.
// This extends the existing fraud engine with marketplace-specific rules.
func CheckMarketplaceRules(ctx context.Context, db *gorm.DB, input MarketplaceCheckInput) MarketplaceRiskResult {
	result := MarketplaceRiskResult{Action: "allow"}

	// ── 1. Rapid behavior: > 5 offers in 1 minute ──────────────────────────
	offerCount := countRecentActions(db, input.UserID, "offer", 1*time.Minute)
	if offerCount > 5 {
		result.Flags = append(result.Flags, "rapid_offers")
		result.Score += 30
	}

	// ── 2. Suspicious pricing: price < 30% market avg ───────────────────────
	if input.MarketAvg > 0 && input.Price > 0 {
		ratio := input.Price / input.MarketAvg
		if ratio < 0.3 {
			result.Flags = append(result.Flags, "suspicious_pricing")
			result.Score += 25
		} else if ratio < 0.5 {
			result.Flags = append(result.Flags, "below_market")
			result.Score += 10
		}
	}

	// ── 3. Multi-account detection: same IP / device / wallet ──────────────
	if input.IP != "" {
		sameIP := countUsersWithIP(db, input.IP, input.UserID)
		if sameIP > 2 {
			result.Flags = append(result.Flags, "multi_account_ip")
			result.Score += 35
		}
	}
	if input.DeviceID != "" {
		sameDevice := countUsersWithDevice(db, input.DeviceID, input.UserID)
		if sameDevice > 2 {
			result.Flags = append(result.Flags, "multi_account_device")
			result.Score += 35
		}
	}
	if input.WalletID != "" {
		sameWallet := countUsersWithWallet(db, input.WalletID, input.UserID)
		if sameWallet > 1 {
			result.Flags = append(result.Flags, "shared_wallet")
			result.Score += 40
		}
	}

	// ── 4. Escrow abuse: frequent cancel after funds hold ──────────────────
	cancelAfterHold := countCancelAfterHold(db, input.UserID)
	if cancelAfterHold >= 3 {
		result.Flags = append(result.Flags, "escrow_abuse")
		result.Score += 30
	}

	// ── 5. New account + high value ─────────────────────────────────────────
	if isNewAccount(db, input.UserID) && input.Amount > 500 {
		result.Flags = append(result.Flags, "new_account_high_value")
		result.Score += 20
	}

	// Clamp score
	result.Score = int(math.Max(0, math.Min(100, float64(result.Score))))

	// Determine action
	switch {
	case result.Score >= 70:
		result.Action = "block"
	case result.Score >= 40:
		result.Action = "review"
	default:
		result.Action = "allow"
	}

	return result
}

// ── Helper queries ────────────────────────────────────────────────────────────

func countRecentActions(db *gorm.DB, userID uuid.UUID, actionType string, window time.Duration) int64 {
	var cnt int64
	db.Table("user_activity_log").
		Where("user_id=? AND action_type=? AND created_at>?", userID, actionType, time.Now().Add(-window)).
		Count(&cnt)
	return cnt
}

func countUsersWithIP(db *gorm.DB, ip string, excludeUser uuid.UUID) int64 {
	var cnt int64
	db.Table("user_sessions").
		Where("ip_address=? AND user_id!=?", ip, excludeUser).
		Distinct("user_id").Count(&cnt)
	return cnt
}

func countUsersWithDevice(db *gorm.DB, deviceID string, excludeUser uuid.UUID) int64 {
	var cnt int64
	db.Table("user_devices").
		Where("device_id=? AND user_id!=?", deviceID, excludeUser).
		Distinct("user_id").Count(&cnt)
	return cnt
}

func countUsersWithWallet(db *gorm.DB, walletID string, excludeUser uuid.UUID) int64 {
	var cnt int64
	db.Table("wallets").
		Where("external_id=? AND user_id!=?", walletID, excludeUser).
		Count(&cnt)
	return cnt
}

func countCancelAfterHold(db *gorm.DB, userID uuid.UUID) int64 {
	var cnt int64
	db.Table("orders").
		Where("buyer_id=? AND status='cancelled' AND escrow_held_at IS NOT NULL", userID).
		Count(&cnt)
	return cnt
}

func isNewAccount(db *gorm.DB, userID uuid.UUID) bool {
	var createdAt time.Time
	db.Table("users").Where("id=?", userID).Select("created_at").Scan(&createdAt)
	return time.Since(createdAt) < 7*24*time.Hour
}

// FlagUser creates a fraud alert for a user.
func FlagUser(db *gorm.DB, userID uuid.UUID, alertType string, severity Severity, indicators string) error {
	alert := FraudAlert{
		TargetType: TargetUser,
		TargetID:   userID,
		AlertType:  alertType,
		Severity:   severity,
		DetectedBy: "marketplace_rules",
		Indicators: indicators,
		Status:     AlertPending,
	}
	return db.Create(&alert).Error
}

// FormatRiskResult creates a human-readable summary.
func FormatMarketplaceRiskResult(r MarketplaceRiskResult) string {
	return fmt.Sprintf("score=%d action=%s flags=%v", r.Score, r.Action, r.Flags)
}
