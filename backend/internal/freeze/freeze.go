package freeze

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserFreeze mirrors admin.UserFreeze for standalone freeze checks.
type UserFreeze struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	IsFrozen bool      `gorm:"not null;default:false"`
	Reason   string    `gorm:"type:text"`
	FrozenBy uuid.UUID `gorm:"type:uuid"`
}

func (UserFreeze) TableName() string { return "user_freezes" }

// IsUserFrozen checks if a user is currently frozen.
// This is a standalone function with no import cycles.
func IsUserFrozen(db *gorm.DB, userID uuid.UUID) bool {
	var freeze UserFreeze
	if err := db.Where("user_id = ? AND is_frozen = ?", userID, true).First(&freeze).Error; err != nil {
		return false
	}
	return true
}

// AuditLogEntry mirrors admin.AuditLogEntry against the unified
// `admin_audit_log` table (migration 047).
type AuditLogEntry struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Action    string    `gorm:"size:100;not null;index"`
	ActorID   uuid.UUID `gorm:"column:admin_id;type:uuid;not null;index"`
	TargetID  string    `gorm:"column:target_id;size:128;not null;index;default:''"`
	Details   string    `gorm:"type:jsonb"`
	CreatedAt int64     `gorm:"autoCreateTime"`
}

func (AuditLogEntry) TableName() string { return "admin_audit_log" }

// LogAudit records an action in the admin audit log.
func LogAudit(db *gorm.DB, action string, actorID, targetID uuid.UUID, details string) {
	entry := AuditLogEntry{
		ID:       uuid.New(),
		Action:   action,
		ActorID:  actorID,
		TargetID: targetID.String(),
		Details:  details,
	}
	db.Create(&entry)
}

// FreezeUser freezes a user account and cancels their pending offers.
func FreezeUser(db *gorm.DB, userID, adminID uuid.UUID, reason string) error {
	freeze := UserFreeze{
		ID:       uuid.New(),
		UserID:   userID,
		IsFrozen: true,
		Reason:   reason,
		FrozenBy: adminID,
	}
	if err := db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"is_frozen": true, "reason": reason, "frozen_by": adminID})}).
		Create(&freeze).Error; err != nil {
		return err
	}
	db.Exec("UPDATE traveler_offers SET status='cancelled' WHERE traveler_id=? AND status IN ('pending','payment_pending')", userID)
	db.Exec("UPDATE traveler_offers SET status='cancelled' WHERE buyer_id=? AND status IN ('pending','payment_pending')", userID)
	LogAudit(db, "user_freeze", adminID, userID, fmt.Sprintf("reason=%s", reason))
	return nil
}

// UnfreezeUser removes the freeze on a user.
func UnfreezeUser(db *gorm.DB, userID, adminID uuid.UUID) error {
	if err := db.Model(&UserFreeze{}).Where("user_id = ?", userID).Update("is_frozen", false).Error; err != nil {
		return err
	}
	LogAudit(db, "user_unfreeze", adminID, userID, "")
	return nil
}
