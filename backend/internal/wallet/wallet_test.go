package wallet

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func d(f float64) decimal.Decimal { return decimal.NewFromFloat(f) }

// makeBalance creates a WalletBalance with explicit fields for test setup.
func makeBalance(balance, available, pending float64) WalletBalance {
	return WalletBalance{
		Balance:          d(balance),
		AvailableBalance: d(available),
		PendingBalance:   d(pending),
	}
}

// ── FIX-1: Deposit invariant ─────────────────────────────────────────────────

// TestDepositPreservingPending verifies that depositing into a wallet that has
// an existing PendingBalance (escrow hold) does NOT clobber the pending amount.
//
// Scenario: Balance=100, Available=60, Pending=40, Deposit=50
// Expected: Balance=150, Available=110, Pending=40 (unchanged)
//
// Bug (pre-fix): balance.AvailableBalance = newBalance set Available=150,
// violating Balance==Available+Pending (150 != 150+40).
func TestDepositPreservingPending(t *testing.T) {
	b := makeBalance(100, 60, 40)

	_, _ = applyDeposit(&b, d(50))

	assert.True(t, b.Balance.Equal(d(150)), "balance should be 150, got %s", b.Balance)
	assert.True(t, b.AvailableBalance.Equal(d(110)), "available should be 110, got %s (bug: was 150)", b.AvailableBalance)
	assert.True(t, b.PendingBalance.Equal(d(40)), "pending must be unchanged at 40, got %s", b.PendingBalance)
	require.NoError(t, checkInvariant(b), "invariant Balance==Available+Pending must hold after deposit")
}

// TestDepositNoPendingBalance tests the simple case: no escrow holds present.
// Expected: Available and Balance both increase by the deposit amount.
func TestDepositNoPendingBalance(t *testing.T) {
	b := makeBalance(100, 100, 0)

	_, _ = applyDeposit(&b, d(50))

	assert.True(t, b.Balance.Equal(d(150)))
	assert.True(t, b.AvailableBalance.Equal(d(150)))
	assert.True(t, b.PendingBalance.Equal(d(0)))
	require.NoError(t, checkInvariant(b))
}

// TestDepositReturnsCorrectSnapshots verifies the returned balanceBefore and
// availableBefore match the pre-mutation state (used in WalletTransaction records).
func TestDepositReturnsCorrectSnapshots(t *testing.T) {
	b := makeBalance(100, 60, 40)

	balBefore, availBefore := applyDeposit(&b, d(50))

	assert.True(t, balBefore.Equal(d(100)), "balanceBefore must capture pre-mutation Balance")
	assert.True(t, availBefore.Equal(d(60)), "availableBefore must capture pre-mutation Available")
}

// ── FIX-2: Withdrawal invariant ───────────────────────────────────────────────

// TestWithdrawPreservingPending verifies a withdrawal only decrements Available,
// leaving PendingBalance untouched.
//
// Scenario: Balance=100, Available=60, Pending=40, Withdraw=40
// Expected: Balance=60, Available=20, Pending=40
//
// Bug (pre-fix): balance.AvailableBalance = newBalance set Available=60,
// meaning user appeared to have 60 available when they should have 20.
func TestWithdrawPreservingPending(t *testing.T) {
	b := makeBalance(100, 60, 40)

	_, _ = applyWithdrawal(&b, d(40))

	assert.True(t, b.Balance.Equal(d(60)), "balance should be 60, got %s", b.Balance)
	assert.True(t, b.AvailableBalance.Equal(d(20)), "available should be 20, got %s (bug: was 60)", b.AvailableBalance)
	assert.True(t, b.PendingBalance.Equal(d(40)), "pending must be unchanged at 40, got %s", b.PendingBalance)
	require.NoError(t, checkInvariant(b), "invariant must hold after withdrawal")
}

// TestWithdrawReturnsCorrectSnapshots verifies the returned snapshots match
// pre-mutation state — critical for accurate WalletTransaction records.
func TestWithdrawReturnsCorrectSnapshots(t *testing.T) {
	b := makeBalance(100, 60, 40)

	balBefore, availBefore := applyWithdrawal(&b, d(40))

	assert.True(t, balBefore.Equal(d(100)), "balanceBefore must capture pre-mutation Balance")
	assert.True(t, availBefore.Equal(d(60)), "availableBefore must capture pre-mutation Available")
}

// ── FIX-3: Escrow hold ───────────────────────────────────────────────────────

// TestEscrowHoldPreservesTotalBalance verifies that an escrow hold moves funds
// from Available to Pending without changing the total Balance.
//
// Scenario: Balance=100, Available=100, Pending=0, Hold=40
// Expected: Balance=100, Available=60, Pending=40
func TestEscrowHoldPreservesTotalBalance(t *testing.T) {
	b := makeBalance(100, 100, 0)

	availBefore := applyEscrowHold(&b, d(40))

	assert.True(t, b.Balance.Equal(d(100)), "total balance must not change on hold")
	assert.True(t, b.AvailableBalance.Equal(d(60)), "available should decrease by hold amount")
	assert.True(t, b.PendingBalance.Equal(d(40)), "pending should increase by hold amount")
	assert.True(t, availBefore.Equal(d(100)), "availableBefore must capture pre-hold value")
	require.NoError(t, checkInvariant(b), "invariant must hold after escrow hold")
}

// TestEscrowHoldReturnsAvailableBefore verifies the returned availableBefore is
// correct (used as BalanceBefore in WalletTransaction for escrow operations).
//
// Bug (pre-fix in escrow_service.go): BalanceBefore was not set at all,
// and BalanceAfter used balance.Balance instead of post-hold AvailableBalance.
func TestEscrowHoldReturnsAvailableBefore(t *testing.T) {
	b := makeBalance(200, 150, 50)

	availBefore := applyEscrowHold(&b, d(30))

	assert.True(t, availBefore.Equal(d(150)), "availableBefore = 150 (pre-hold)")
	assert.True(t, b.AvailableBalance.Equal(d(120)), "availableAfter = 120")
	require.NoError(t, checkInvariant(b))
}

// ── Invariant checker ─────────────────────────────────────────────────────────

// TestCheckInvariantValid verifies no error returned for a consistent balance.
func TestCheckInvariantValid(t *testing.T) {
	b := makeBalance(100, 60, 40)
	require.NoError(t, checkInvariant(b))
}

// TestCheckInvariantViolation verifies that a drifted balance returns an error.
// This is the exact state produced by the buggy Deposit code.
func TestCheckInvariantViolation(t *testing.T) {
	// Simulate what the buggy code produces:
	// Balance=150, Available=150 (clobbered), Pending=40
	// 150 != 150 + 40 → invariant violated
	b := makeBalance(150, 150, 40)
	err := checkInvariant(b)
	require.Error(t, err, "must detect invariant violation: 150 != 150+40")
}

// ── Escrow cancel/release ─────────────────────────────────────────────────────

// TestEscrowCancelRestoresFunds verifies cancel moves Pending→Available.
func TestEscrowCancelRestoresFunds(t *testing.T) {
	b := makeBalance(100, 60, 40)

	applyEscrowCancel(&b, d(40))

	assert.True(t, b.Balance.Equal(d(100)), "total balance unchanged on cancel")
	assert.True(t, b.AvailableBalance.Equal(d(100)), "all funds restored to available")
	assert.True(t, b.PendingBalance.Equal(d(0)))
	require.NoError(t, checkInvariant(b))
}

// TestEscrowReleaseBuyerSide verifies buyer-side release deducts from Pending+Balance.
func TestEscrowReleaseBuyerSide(t *testing.T) {
	b := makeBalance(100, 60, 40)

	applyEscrowRelease(&b, d(40))

	assert.True(t, b.Balance.Equal(d(60)), "balance decreases by released amount")
	assert.True(t, b.AvailableBalance.Equal(d(60)), "available unchanged (was not holding)")
	assert.True(t, b.PendingBalance.Equal(d(0)), "pending clears")
	require.NoError(t, checkInvariant(b))
}

// ── Multi-operation sequence ──────────────────────────────────────────────────

// TestSequence_DepositHoldWithdraw verifies a full sequence:
// Deposit → Hold → Withdraw maintains invariant at each step.
func TestSequence_DepositHoldWithdraw(t *testing.T) {
	b := makeBalance(0, 0, 0)

	// Deposit 200
	applyDeposit(&b, d(200))
	require.NoError(t, checkInvariant(b))
	assert.True(t, b.Balance.Equal(d(200)))
	assert.True(t, b.AvailableBalance.Equal(d(200)))

	// Hold 80 in escrow
	applyEscrowHold(&b, d(80))
	require.NoError(t, checkInvariant(b))
	assert.True(t, b.Balance.Equal(d(200)))
	assert.True(t, b.AvailableBalance.Equal(d(120)))
	assert.True(t, b.PendingBalance.Equal(d(80)))

	// Deposit another 50
	applyDeposit(&b, d(50))
	require.NoError(t, checkInvariant(b))
	assert.True(t, b.Balance.Equal(d(250)))
	assert.True(t, b.AvailableBalance.Equal(d(170)), "deposit must not clobber pending hold")
	assert.True(t, b.PendingBalance.Equal(d(80)), "pending must remain 80 after deposit")

	// Withdraw 100 from available
	applyWithdrawal(&b, d(100))
	require.NoError(t, checkInvariant(b))
	assert.True(t, b.Balance.Equal(d(150)))
	assert.True(t, b.AvailableBalance.Equal(d(70)))
	assert.True(t, b.PendingBalance.Equal(d(80)), "pending unchanged after withdrawal")
}

// TestEscrowReleaseApproval_FirstAdmin verifies first approval is recorded
// and escrow remains pending second approval.
func TestEscrowReleaseApproval_FirstAdmin(t *testing.T) {
	esc := Escrow{}
	admin1 := uuid.New()
	now := time.Now()

	ready, err := applyEscrowReleaseApproval(&esc, admin1, now)
	require.NoError(t, err)
	assert.False(t, ready)
	require.NotNil(t, esc.Approval1By)
	assert.Equal(t, admin1, *esc.Approval1By)
	require.NotNil(t, esc.Approval1At)
	assert.Nil(t, esc.Approval2By)
}

// TestEscrowReleaseApproval_SameAdminRejected verifies second approval must be
// from a distinct admin.
func TestEscrowReleaseApproval_SameAdminRejected(t *testing.T) {
	esc := Escrow{}
	admin1 := uuid.New()
	now := time.Now()

	_, err := applyEscrowReleaseApproval(&esc, admin1, now)
	require.NoError(t, err)

	ready, err := applyEscrowReleaseApproval(&esc, admin1, now.Add(time.Minute))
	require.Error(t, err)
	assert.EqualError(t, err, "second_approval_must_be_distinct_admin")
	assert.False(t, ready)
	assert.Nil(t, esc.Approval2By)
}

// TestEscrowReleaseApproval_SecondDistinctAdmin verifies distinct second admin
// marks escrow ready for release.
func TestEscrowReleaseApproval_SecondDistinctAdmin(t *testing.T) {
	esc := Escrow{}
	admin1 := uuid.New()
	admin2 := uuid.New()
	now := time.Now()

	ready, err := applyEscrowReleaseApproval(&esc, admin1, now)
	require.NoError(t, err)
	assert.False(t, ready)

	ready, err = applyEscrowReleaseApproval(&esc, admin2, now.Add(time.Minute))
	require.NoError(t, err)
	assert.True(t, ready)
	require.NotNil(t, esc.Approval2By)
	assert.Equal(t, admin2, *esc.Approval2By)
	require.NotNil(t, esc.Approval2At)
}

func TestEscrowRelease_SystemBalanceInvariant(t *testing.T) {
	buyer := makeBalance(200, 120, 80)
	seller := makeBalance(100, 100, 0)

	beforeSystem := buyer.Balance.Add(seller.Balance)

	applyEscrowRelease(&buyer, d(80))
	_, _ = applyDeposit(&seller, d(80))

	require.NoError(t, checkInvariant(buyer))
	require.NoError(t, checkInvariant(seller))

	afterSystem := buyer.Balance.Add(seller.Balance)
	assert.True(t, beforeSystem.Equal(afterSystem), "system total balance must be conserved across escrow release")
}
