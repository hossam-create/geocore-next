package admin

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/freeze"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ════════════════════════════════════════════════════════════════════════════
// Admin Control API — Production Safety Controls
// Freeze users, force resolve disputes, adjust wallets, override transactions.
// ════════════════════════════════════════════════════════════════════════════

// UserFreeze records a user freeze/unfreeze action.
type UserFreeze struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	Reason    string    `gorm:"type:text;not null" json:"reason"`
	FrozenBy  uuid.UUID `gorm:"type:uuid;not null" json:"frozen_by"`
	IsFrozen  bool      `gorm:"not null;default:true" json:"is_frozen"`
	CreatedAt time.Time `json:"created_at"`
}

func (UserFreeze) TableName() string { return "user_freezes" }

// WalletAdjustment records a manual wallet adjustment by admin.
type WalletAdjustment struct {
	ID              uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID          uuid.UUID       `gorm:"type:uuid;not null;index" json:"user_id"`
	AmountCents     int64           `gorm:"not null" json:"amount_cents"`
	Reason          string          `gorm:"type:text;not null" json:"reason"`
	AdjustedBy      uuid.UUID       `gorm:"type:uuid;not null" json:"adjusted_by"`
	PreviousBalance decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"previous_balance"`
	NewBalance      decimal.Decimal `gorm:"type:decimal(15,2);not null" json:"new_balance"`
	CreatedAt       time.Time       `json:"created_at"`
}

func (WalletAdjustment) TableName() string { return "wallet_adjustments" }

// TransactionOverride records a high-risk transaction override.
type TransactionOverride struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TransactionID  uuid.UUID `gorm:"type:uuid;not null;index" json:"transaction_id"`
	OverrideType   string    `gorm:"size:50;not null" json:"override_type"` // release, refund, cancel
	Reason         string    `gorm:"type:text;not null" json:"reason"`
	OverriddenBy   uuid.UUID `gorm:"type:uuid;not null" json:"overridden_by"`
	PreviousStatus string    `gorm:"size:50;not null" json:"previous_status"`
	NewStatus      string    `gorm:"size:50;not null" json:"new_status"`
	CreatedAt      time.Time `json:"created_at"`
}

func (TransactionOverride) TableName() string { return "transaction_overrides" }

// AuditLogEntry records admin actions for compliance.
//
// Column mapping is pinned to the canonical `admin_audit_log` schema (migration
// 047). Legacy field names (`ActorID` / `TargetID`) are preserved for API
// backwards-compat, but columns map to the unified `admin_id` + `target_id`.
type AuditLogEntry struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"       json:"id"`
	Action    string    `gorm:"size:100;not null;index"                               json:"action"`
	ActorID   uuid.UUID `gorm:"column:admin_id;type:uuid;not null;index"              json:"actor_id"`
	TargetID  string    `gorm:"column:target_id;size:128;not null;index;default:''"   json:"target_id"`
	Details   string    `gorm:"type:jsonb"                                            json:"details"`
	CreatedAt time.Time `json:"created_at"`
}

func (AuditLogEntry) TableName() string { return "admin_audit_log" }

// ── Freeze User ──────────────────────────────────────────────────────────────

// FreezeUser blocks a user from withdrawing, creating offers, or deal closer.
func FreezeUser(db *gorm.DB, userID, adminID uuid.UUID, reason string) error {
	return locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		f := UserFreeze{
			ID:       uuid.New(),
			UserID:   userID,
			Reason:   reason,
			FrozenBy: adminID,
			IsFrozen: true,
		}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"is_frozen": true, "reason": reason, "frozen_by": adminID})}).
			Create(&f).Error; err != nil {
			return err
		}

		// Cancel pending offers
		tx.Exec("UPDATE traveler_offers SET status='cancelled' WHERE traveler_id=? AND status IN ('pending','payment_pending')", userID)
		tx.Exec("UPDATE traveler_offers SET status='cancelled' WHERE buyer_id=? AND status IN ('pending','payment_pending')", userID)

		freeze.LogAudit(tx, "user_freeze", adminID, userID, fmt.Sprintf("reason=%s", reason))
		slog.Info("admin: user frozen", "user_id", userID, "admin_id", adminID, "reason", reason)
		return nil
	})
}

// UnfreezeUser removes the freeze on a user.
func UnfreezeUser(db *gorm.DB, userID, adminID uuid.UUID) error {
	return locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		if err := tx.Model(&UserFreeze{}).Where("user_id = ?", userID).Update("is_frozen", false).Error; err != nil {
			return err
		}
		freeze.LogAudit(tx, "user_unfreeze", adminID, userID, "")
		slog.Info("admin: user unfrozen", "user_id", userID, "admin_id", adminID)
		return nil
	})
}

// IsUserFrozen checks if a user is currently frozen.
func IsUserFrozen(db *gorm.DB, userID uuid.UUID) bool {
	return freeze.IsUserFrozen(db, userID)
}

// ── Wallet Adjustment ────────────────────────────────────────────────────────

// AdjustWallet manually adjusts a user's wallet balance with audit trail.
func AdjustWallet(db *gorm.DB, userID, adminID uuid.UUID, amountCents int64, reason string) error {
	return locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		var prevBalance decimal.Decimal
		tx.Table("wallet_balances").
			Joins("JOIN wallets ON wallets.id = wallet_balances.wallet_id").
			Where("wallets.user_id = ?", userID).
			Select("COALESCE(balance,0)").Scan(&prevBalance)

		amount := decimal.NewFromInt(amountCents).Div(decimal.NewFromInt(100))
		newBalance := prevBalance.Add(amount)

		// Apply adjustment
		tx.Exec(`UPDATE wallet_balances SET balance = balance + ?, available_balance = available_balance + ?
			FROM wallets WHERE wallets.id = wallet_balances.wallet_id AND wallets.user_id = ?`,
			amount, amount, userID)

		adj := WalletAdjustment{
			ID:              uuid.New(),
			UserID:          userID,
			AmountCents:     amountCents,
			Reason:          reason,
			AdjustedBy:      adminID,
			PreviousBalance: prevBalance,
			NewBalance:      newBalance,
		}
		tx.Create(&adj)

		freeze.LogAudit(tx, "wallet_adjust", adminID, userID,
			fmt.Sprintf("amount_cents=%d reason=%s prev=%s new=%s", amountCents, reason, prevBalance.String(), newBalance.String()))
		slog.Info("admin: wallet adjusted", "user_id", userID, "amount_cents", amountCents, "admin_id", adminID)
		return nil
	})
}

// ── Transaction Override ─────────────────────────────────────────────────────

// OverrideTransaction allows admin to force a transaction state change.
func OverrideTransaction(db *gorm.DB, transactionID, adminID uuid.UUID, overrideType, reason string) error {
	return locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		// Get current status from escrow
		var prevStatus string
		tx.Table("escrow_accounts").Where("id = ?", transactionID).Select("status").Scan(&prevStatus)

		var newStatus string
		switch overrideType {
		case "release":
			newStatus = "released"
		case "refund":
			newStatus = "refunded"
		case "cancel":
			newStatus = "cancelled"
		default:
			return fmt.Errorf("unknown override type: %s", overrideType)
		}

		tx.Exec("UPDATE escrow_accounts SET status=?, notes=? WHERE id=?", newStatus,
			fmt.Sprintf("Admin override: %s", reason), transactionID)

		ov := TransactionOverride{
			ID:             uuid.New(),
			TransactionID:  transactionID,
			OverrideType:   overrideType,
			Reason:         reason,
			OverriddenBy:   adminID,
			PreviousStatus: prevStatus,
			NewStatus:      newStatus,
		}
		tx.Create(&ov)

		freeze.LogAudit(tx, "transaction_override", adminID, transactionID,
			fmt.Sprintf("type=%s prev=%s new=%s reason=%s", overrideType, prevStatus, newStatus, reason))
		slog.Info("admin: transaction overridden", "transaction_id", transactionID, "type", overrideType, "admin_id", adminID)
		return nil
	})
}

// ── Audit Logging ────────────────────────────────────────────────────────────

// LogAudit delegates to freeze.LogAudit for backward compatibility.
func LogAudit(db *gorm.DB, action string, actorID, targetID uuid.UUID, details string) {
	freeze.LogAudit(db, action, actorID, targetID, details)
}

// ── HTTP Handlers ────────────────────────────────────────────────────────────

// FreezeUserHandler — POST /admin/users/:id/freeze
func (h *Handler) FreezeUserHandler(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	adminID, _ := uuid.Parse(c.GetString("user_id"))
	if err := FreezeUser(h.db, userID, adminID, req.Reason); err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "user frozen"})
}

// UnfreezeUserHandler — POST /admin/users/:id/unfreeze
func (h *Handler) UnfreezeUserHandler(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	adminID, _ := uuid.Parse(c.GetString("user_id"))
	if err := UnfreezeUser(h.db, userID, adminID); err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "user unfrozen"})
}

// AdjustWalletHandler — POST /admin/wallet/adjust
func (h *Handler) AdjustWalletHandler(c *gin.Context) {
	var req struct {
		UserID      string `json:"user_id" binding:"required"`
		AmountCents int64  `json:"amount_cents" binding:"required"`
		Reason      string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		response.BadRequest(c, "invalid user_id")
		return
	}
	adminID, _ := uuid.Parse(c.GetString("user_id"))
	if err := AdjustWallet(h.db, userID, adminID, req.AmountCents, req.Reason); err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "wallet adjusted"})
}

// OverrideTransactionHandler — POST /admin/override/transaction
func (h *Handler) OverrideTransactionHandler(c *gin.Context) {
	var req struct {
		TransactionID string `json:"transaction_id" binding:"required"`
		OverrideType  string `json:"override_type" binding:"required"`
		Reason        string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	txID, err := uuid.Parse(req.TransactionID)
	if err != nil {
		response.BadRequest(c, "invalid transaction_id")
		return
	}
	adminID, _ := uuid.Parse(c.GetString("user_id"))
	if err := OverrideTransaction(h.db, txID, adminID, req.OverrideType, req.Reason); err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "transaction overridden"})
}

// GetAuditLogHandler — GET /admin/audit-log
func (h *Handler) GetAuditLogHandler(c *gin.Context) {
	var entries []AuditLogEntry
	query := h.db.Order("created_at DESC").Limit(100)
	if targetID := c.Query("target_id"); targetID != "" {
		// target_id is VARCHAR in unified admin_audit_log (migration 047);
		// match as string so non-UUID target identifiers also work.
		query = query.Where("target_id = ?", targetID)
	}
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}
	query.Find(&entries)
	response.OK(c, entries)
}
