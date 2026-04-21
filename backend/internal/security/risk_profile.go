package security

import (
	"time"

	"github.com/geocore-next/backend/internal/config"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRiskProfile is the per-user risk aggregate used for auto-freeze decisions.
type UserRiskProfile struct {
	UserID       uuid.UUID  `gorm:"type:uuid;primaryKey"               json:"user_id"`
	RiskScore    int        `gorm:"not null;default:0;index"           json:"risk_score"`
	LastIP       string     `gorm:"size:64"                            json:"last_ip"`
	DeviceCount  int        `gorm:"default:0"                          json:"device_count"`
	FlagsCount   int        `gorm:"default:0"                          json:"flags_count"`
	Frozen       bool       `gorm:"default:false;index"                json:"frozen"`
	FrozenReason string     `gorm:"size:255"                           json:"frozen_reason,omitempty"`
	FrozenAt     *time.Time `                                         json:"frozen_at,omitempty"`
	LastEventAt  time.Time  `                                          json:"last_event_at"`
	UpdatedAt    time.Time  `                                          json:"updated_at"`
}

func (UserRiskProfile) TableName() string { return "user_security_profiles" }

// Risk scoring deltas.
const (
	DeltaLoginFailBurst    = 10 // triggered when 5+ failures hit in window
	DeltaMultipleIPs       = 15
	DeltaRapidActions      = 20
	DeltaFraudFlag         = 40
	DeltaRateLimit         = 5
	DeltaSuspiciousGeneric = 20
	DeltaSessionRevoke     = 25
	DeltaPasswordReset     = 10

	AutoFreezeThreshold   = 70 // score at which user is auto-frozen
	MultipleIPsThreshold  = 3  // distinct IPs in 24h considered suspicious
	LoginFailBurstWindow  = 10 * time.Minute
	LoginFailBurstTrigger = 5
	RapidActionsWindow    = 60 * time.Second
	RapidActionsTrigger   = 20
)

// Risk level labels for API visual tagging (Part 11).
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// LevelFromScore maps an integer score to a categorical RiskLevel.
func LevelFromScore(score int) RiskLevel {
	switch {
	case score >= 86:
		return RiskCritical
	case score >= 61:
		return RiskHigh
	case score >= 31:
		return RiskMedium
	default:
		return RiskLow
	}
}

// GetRiskProfile fetches or creates a user's risk profile.
func GetRiskProfile(db *gorm.DB, userID uuid.UUID) UserRiskProfile {
	var p UserRiskProfile
	db.Where("user_id = ?", userID).First(&p)
	if p.UserID == uuid.Nil {
		p = UserRiskProfile{UserID: userID, UpdatedAt: time.Now().UTC()}
		db.Create(&p)
	}
	return p
}

// AddRisk atomically bumps the risk score and persists. Returns the new score.
func AddRisk(db *gorm.DB, userID uuid.UUID, delta int, ip string) int {
	if userID == uuid.Nil || delta <= 0 {
		return 0
	}
	now := time.Now().UTC()
	p := GetRiskProfile(db, userID)
	newScore := p.RiskScore + delta
	if newScore > 100 {
		newScore = 100
	}

	updates := map[string]any{
		"risk_score":    newScore,
		"flags_count":   p.FlagsCount + 1,
		"last_event_at": now,
		"updated_at":    now,
	}
	if ip != "" && ip != p.LastIP {
		updates["last_ip"] = ip
	}
	db.Model(&UserRiskProfile{}).Where("user_id = ?", userID).Updates(updates)

	// Auto-freeze when threshold is crossed.
	if !p.Frozen && newScore >= AutoFreezeThreshold && config.GetFlags().EnableAutoFreeze {
		FreezeUser(db, userID, "auto_freeze: risk_score>="+itoa(AutoFreezeThreshold))
	}
	return newScore
}

// FreezeUser flags a user as frozen, preventing all privileged actions.
func FreezeUser(db *gorm.DB, userID uuid.UUID, reason string) {
	now := time.Now().UTC()
	db.Model(&UserRiskProfile{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"frozen":        true,
			"frozen_reason": reason,
			"frozen_at":     now,
			"updated_at":    now,
		})
	LogEventDirect(db, &userID, "user_frozen", "", "", map[string]any{"reason": reason})
}

// UnfreezeUser clears the frozen state.
func UnfreezeUser(db *gorm.DB, userID uuid.UUID, reason string) {
	db.Model(&UserRiskProfile{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"frozen":        false,
			"frozen_reason": "",
			"frozen_at":     nil,
			"updated_at":    time.Now().UTC(),
		})
	LogEventDirect(db, &userID, "user_unfrozen", "", "", map[string]any{"reason": reason})
}

// IsUserFrozen returns true when the given user is currently frozen.
func IsUserFrozen(db *gorm.DB, userID uuid.UUID) bool {
	if userID == uuid.Nil {
		return false
	}
	var frozen bool
	db.Raw(`SELECT COALESCE(frozen, false) FROM user_security_profiles WHERE user_id = ?`, userID).Scan(&frozen)
	return frozen
}

// small local itoa to avoid importing strconv twice across this file.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
