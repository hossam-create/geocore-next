package wallet

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ════════════════════════════════════════════════════════════════════════════
// Wallet Sync Layer Tests (Sprint 7 — Anti Double-Spend)
// ════════════════════════════════════════════════════════════════════════════

func TestWalletSyncSnapshot_Struct(t *testing.T) {
	snap := WalletSyncSnapshot{
		UserID:            uuid.New(),
		Currency:          USD,
		ConfirmedBalance:  decimal.NewFromInt(1000),
		PendingDeposits:   decimal.NewFromInt(200),
		ConfirmedDeposits: decimal.NewFromInt(1200),
		MatchedTransfers:  decimal.NewFromInt(100),
		AvailableBalance:  decimal.NewFromInt(900),
		PendingBalance:    decimal.NewFromInt(100),
		InvariantOK:       true,
	}
	if snap.Currency != USD {
		t.Errorf("expected USD, got %s", snap.Currency)
	}
	if !snap.ConfirmedBalance.Equal(decimal.NewFromInt(1000)) {
		t.Errorf("confirmed balance should be 1000, got %s", snap.ConfirmedBalance.String())
	}
	if !snap.InvariantOK {
		t.Error("invariant should be OK")
	}
}

func TestWalletSyncSnapshot_InvariantCheck(t *testing.T) {
	// Invariant: available + pending ≈ confirmed_balance
	tests := []struct {
		name     string
		avail    float64
		pending  float64
		confirmed float64
		ok       bool
	}{
		{"balanced", 800, 200, 1000, true},
		{"exact match", 1000, 0, 1000, true},
		{"small rounding", 999.99, 0, 1000, true},
		{"violation", 2000, 0, 1000, false},
		{"negative diff", 500, 0, 1000, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			avail := decimal.NewFromFloat(tt.avail)
			pending := decimal.NewFromFloat(tt.pending)
			confirmed := decimal.NewFromFloat(tt.confirmed)
			total := avail.Add(pending)
			diff := total.Sub(confirmed).Abs()
			invariantOK := diff.LessThanOrEqual(decimal.NewFromFloat(0.01))
			if invariantOK != tt.ok {
				t.Errorf("avail=%v pending=%v confirmed=%v: expected ok=%v, got %v",
					tt.avail, tt.pending, tt.confirmed, tt.ok, invariantOK)
			}
		})
	}
}

func TestReplayAttackCheck_EmptyKey(t *testing.T) {
	// Empty key should always pass (no-op)
	err := ReplayAttackCheck(nil, uuid.New(), "")
	if err != nil {
		t.Errorf("empty key should pass, got: %v", err)
	}
}
