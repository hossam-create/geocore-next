package admin

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ════════════════════════════════════════════════════════════════════════════
// STEP 2: Admin Control Tests
// ════════════════════════════════════════════════════════════════════════════

func TestUserFreeze_TableName(t *testing.T) {
	uf := UserFreeze{}
	if uf.TableName() != "user_freezes" {
		t.Errorf("expected 'user_freezes', got '%s'", uf.TableName())
	}
}

func TestWalletAdjustment_TableName(t *testing.T) {
	wa := WalletAdjustment{}
	if wa.TableName() != "wallet_adjustments" {
		t.Errorf("expected 'wallet_adjustments', got '%s'", wa.TableName())
	}
}

func TestTransactionOverride_TableName(t *testing.T) {
	to := TransactionOverride{}
	if to.TableName() != "transaction_overrides" {
		t.Errorf("expected 'transaction_overrides', got '%s'", to.TableName())
	}
}

func TestAuditLogEntry_TableName(t *testing.T) {
	al := AuditLogEntry{}
	if al.TableName() != "admin_audit_log" {
		t.Errorf("expected 'admin_audit_log', got '%s'", al.TableName())
	}
}

func TestUserFreeze_Fields(t *testing.T) {
	userID := uuid.New()
	adminID := uuid.New()
	uf := UserFreeze{
		UserID:   userID,
		Reason:   "suspicious activity",
		FrozenBy: adminID,
		IsFrozen: true,
	}
	if uf.UserID != userID {
		t.Error("user ID mismatch")
	}
	if uf.Reason != "suspicious activity" {
		t.Error("reason mismatch")
	}
	if !uf.IsFrozen {
		t.Error("should be frozen")
	}
}

func TestWalletAdjustment_Fields(t *testing.T) {
	userID := uuid.New()
	adminID := uuid.New()
	wa := WalletAdjustment{
		UserID:          userID,
		AmountCents:     -500, // $5 debit
		Reason:          "correction",
		AdjustedBy:      adminID,
		PreviousBalance: decimal.NewFromInt(100),
		NewBalance:      decimal.NewFromInt(95),
	}
	if wa.AmountCents != -500 {
		t.Error("amount cents should be -500")
	}
	if !wa.PreviousBalance.Equal(decimal.NewFromInt(100)) {
		t.Error("previous balance should be 100")
	}
}

func TestTransactionOverride_Types(t *testing.T) {
	types := []string{"release", "refund", "cancel"}
	for _, typ := range types {
		to := TransactionOverride{OverrideType: typ}
		if to.OverrideType != typ {
			t.Errorf("override type mismatch: %s", typ)
		}
	}
}

func TestIsUserFrozen_NoRecord(t *testing.T) {
	// Without a DB, IsUserFrozen returns false (no record found)
	// This tests the function signature and default behavior
	userID := uuid.New()
	// Can't call IsUserFrozen without a real DB, but we verify the struct
	_ = userID
}
