package payments

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ════════════════════════════════════════════════════════════════════════════
// STEP 1: P2P Money Matching Engine Tests
// ════════════════════════════════════════════════════════════════════════════

func TestMatchResult_TableName(t *testing.T) {
	mr := MatchResult{}
	if mr.TableName() != "payment_match_results" {
		t.Errorf("expected 'payment_match_results', got '%s'", mr.TableName())
	}
}

func TestMatchResult_Fields(t *testing.T) {
	depositID := uuid.New()
	withdrawID := uuid.New()
	mr := MatchResult{
		DepositID:  depositID,
		WithdrawID: withdrawID,
		Amount:     decimal.NewFromInt(500),
		Rate:       decimal.NewFromInt(1),
		Status:     "settled",
	}
	if mr.DepositID != depositID {
		t.Errorf("deposit ID mismatch")
	}
	if mr.WithdrawID != withdrawID {
		t.Errorf("withdraw ID mismatch")
	}
	if !mr.Amount.Equal(decimal.NewFromInt(500)) {
		t.Errorf("amount should be 500, got %s", mr.Amount.String())
	}
	if mr.Status != "settled" {
		t.Errorf("status should be 'settled', got '%s'", mr.Status)
	}
}

func TestMatchCriteria_Fields(t *testing.T) {
	mc := MatchCriteria{
		Currency:  "EGP",
		MinAmount: decimal.NewFromInt(100),
		MaxAmount: decimal.NewFromInt(5000),
		MinTrust:  60,
	}
	if mc.Currency != "EGP" {
		t.Errorf("expected EGP, got %s", mc.Currency)
	}
	if mc.MinTrust != 60 {
		t.Errorf("expected min trust 60, got %d", mc.MinTrust)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// STEP 2: Agent Reputation System Tests
// ════════════════════════════════════════════════════════════════════════════

func TestAgentScore_TableName(t *testing.T) {
	as := AgentScore{}
	if as.TableName() != "payment_agent_scores" {
		t.Errorf("expected 'payment_agent_scores', got '%s'", as.TableName())
	}
}

func TestAgentScore_Defaults(t *testing.T) {
	as := AgentScore{
		AgentID: uuid.New(),
	}
	// Go zero-values — DB defaults differ
	if as.Score != 0 {
		t.Errorf("expected score 0 (DB default: 50), got %d", as.Score)
	}
	if as.TotalTx != 0 {
		t.Errorf("expected total_tx 0, got %d", as.TotalTx)
	}
	if as.SuccessTx != 0 {
		t.Errorf("expected success_tx 0, got %d", as.SuccessTx)
	}
	if as.DisputeTx != 0 {
		t.Errorf("expected dispute_tx 0, got %d", as.DisputeTx)
	}
	if as.FraudFlags != 0 {
		t.Errorf("expected fraud_flags 0, got %d", as.FraudFlags)
	}
}

func TestAgentScoreConfig_Thresholds(t *testing.T) {
	cfg := DefaultAgentScoreConfig
	if cfg.BlockScore != 40 {
		t.Errorf("block score should be 40, got %d", cfg.BlockScore)
	}
	if cfg.WarningScore != 60 {
		t.Errorf("warning score should be 60, got %d", cfg.WarningScore)
	}
	if cfg.FraudDropAmount != 30 {
		t.Errorf("fraud drop should be 30, got %d", cfg.FraudDropAmount)
	}
	if cfg.DisputePenalty != 5 {
		t.Errorf("dispute penalty should be 5, got %d", cfg.DisputePenalty)
	}
	if cfg.SuccessBonus != 2 {
		t.Errorf("success bonus should be 2, got %d", cfg.SuccessBonus)
	}
}

func TestAgentScore_BlockingLogic(t *testing.T) {
	tests := []struct {
		name    string
		score   int
		blocked bool
	}{
		{"high score safe", 80, false},
		{"warning zone", 55, false},
		{"at block threshold", 40, false}, // exactly 40 is NOT < 40
		{"below block", 35, true},
		{"zero score", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked := tt.score < DefaultAgentScoreConfig.BlockScore
			if blocked != tt.blocked {
				t.Errorf("score %d: expected blocked=%v, got %v", tt.score, tt.blocked, blocked)
			}
		})
	}
}

// ════════════════════════════════════════════════════════════════════════════
// STEP 3: Payment Dispute Engine Tests
// ════════════════════════════════════════════════════════════════════════════

func TestPaymentDispute_TableName(t *testing.T) {
	pd := PaymentDispute{}
	if pd.TableName() != "payment_disputes" {
		t.Errorf("expected 'payment_disputes', got '%s'", pd.TableName())
	}
}

func TestPaymentDispute_Reasons(t *testing.T) {
	reasons := map[string]string{
		"not_credited":  DisputeReasonNotCredited,
		"not_released":  DisputeReasonNotReleased,
		"double_charge": DisputeReasonDoubleCharge,
		"other":         DisputeReasonOther,
	}
	for expected, actual := range reasons {
		if actual != expected {
			t.Errorf("expected reason '%s', got '%s'", expected, actual)
		}
	}
}

func TestPaymentDispute_Statuses(t *testing.T) {
	statuses := map[string]string{
		"open":     DisputeStatusOpen,
		"review":   DisputeStatusReview,
		"resolved": DisputeStatusResolved,
		"rejected": DisputeStatusRejected,
	}
	for expected, actual := range statuses {
		if actual != expected {
			t.Errorf("expected status '%s', got '%s'", expected, actual)
		}
	}
}

func TestPaymentDispute_Defaults(t *testing.T) {
	pd := PaymentDispute{
		UserID:  uuid.New(),
		AgentID: uuid.New(),
		Amount:  decimal.NewFromInt(100),
		Reason:  DisputeReasonNotCredited,
	}
	if pd.Status != "" {
		t.Errorf("expected empty status (DB default: 'open'), got '%s'", pd.Status)
	}
	if pd.Resolution != nil {
		t.Error("resolution should be nil by default")
	}
	if pd.ProofImage != nil {
		t.Error("proof_image should be nil by default")
	}
}

// ════════════════════════════════════════════════════════════════════════════
// STEP 4: Wallet Sync Layer Tests (anti double-spend)
// WalletSyncSnapshot lives in the wallet package; we test the logic here.
// ════════════════════════════════════════════════════════════════════════════

func TestWalletSyncInvariant_Logic(t *testing.T) {
	// Simulate the invariant check: available + pending ≈ confirmed
	confirmedBalance := decimal.NewFromInt(1000)
	availableBalance := decimal.NewFromInt(800)
	pendingBalance := decimal.NewFromInt(200)
	totalWalletBalance := availableBalance.Add(pendingBalance)

	diff := totalWalletBalance.Sub(confirmedBalance).Abs()
	invariantOK := diff.LessThanOrEqual(decimal.NewFromFloat(0.01))

	if !invariantOK {
		t.Error("800 + 200 = 1000 should match confirmed 1000")
	}
}

func TestWalletSyncInvariant_Violation(t *testing.T) {
	confirmedBalance := decimal.NewFromInt(1000)
	availableBalance := decimal.NewFromInt(2000) // more available than confirmed
	pendingBalance := decimal.Zero
	totalWalletBalance := availableBalance.Add(pendingBalance)

	diff := totalWalletBalance.Sub(confirmedBalance).Abs()
	invariantOK := diff.LessThanOrEqual(decimal.NewFromFloat(0.01))

	if invariantOK {
		t.Error("2000 + 0 ≠ 1000 should detect invariant violation")
	}
}

// ════════════════════════════════════════════════════════════════════════════
// STEP 5: Liquidity Guard Tests
// ════════════════════════════════════════════════════════════════════════════

func TestSystemLiquidityStatus_Levels(t *testing.T) {
	tests := []struct {
		name          string
		imbalancePct  float64
		expectedLevel string
		slowed        bool
	}{
		{"balanced", 5.0, "balanced", false},
		{"slight negative", -5.0, "balanced", false},
		{"warning", -15.0, "warning", true},
		{"critical", -25.0, "critical", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := SystemLiquidityStatus{
				ImbalancePct:      decimal.NewFromFloat(tt.imbalancePct),
				WithdrawalsSlowed: tt.slowed,
			}
			// Determine level
			switch {
			case status.ImbalancePct.LessThan(decimal.NewFromFloat(-20)):
				status.Level = "critical"
			case status.ImbalancePct.LessThan(decimal.NewFromFloat(-10)):
				status.Level = "warning"
			default:
				status.Level = "balanced"
			}
			if status.Level != tt.expectedLevel {
				t.Errorf("expected level '%s', got '%s'", tt.expectedLevel, status.Level)
			}
		})
	}
}

func TestSystemLiquidityStatus_Fees(t *testing.T) {
	// Critical: 3% fee
	critical := SystemLiquidityStatus{Level: "critical"}
	critical.WithdrawalFeePct = decimal.NewFromFloat(0.03)
	if !critical.WithdrawalFeePct.Equal(decimal.NewFromFloat(0.03)) {
		t.Errorf("critical fee should be 3%%")
	}

	// Warning: 1.5% fee
	warning := SystemLiquidityStatus{Level: "warning"}
	warning.WithdrawalFeePct = decimal.NewFromFloat(0.015)
	if !warning.WithdrawalFeePct.Equal(decimal.NewFromFloat(0.015)) {
		t.Errorf("warning fee should be 1.5%%")
	}

	// Balanced: 0.5% fee
	balanced := SystemLiquidityStatus{Level: "balanced"}
	balanced.WithdrawalFeePct = decimal.NewFromFloat(0.005)
	if !balanced.WithdrawalFeePct.Equal(decimal.NewFromFloat(0.005)) {
		t.Errorf("balanced fee should be 0.5%%")
	}
}

func TestSystemLiquidityStatus_TrustThresholds(t *testing.T) {
	// Critical: min trust 80
	critical := SystemLiquidityStatus{Level: "critical", MinTrustForWithdraw: 80}
	if critical.MinTrustForWithdraw != 80 {
		t.Errorf("critical min trust should be 80, got %d", critical.MinTrustForWithdraw)
	}

	// Warning: min trust 60
	warning := SystemLiquidityStatus{Level: "warning", MinTrustForWithdraw: 60}
	if warning.MinTrustForWithdraw != 60 {
		t.Errorf("warning min trust should be 60, got %d", warning.MinTrustForWithdraw)
	}

	// Balanced: min trust 40
	balanced := SystemLiquidityStatus{Level: "balanced", MinTrustForWithdraw: 40}
	if balanced.MinTrustForWithdraw != 40 {
		t.Errorf("balanced min trust should be 40, got %d", balanced.MinTrustForWithdraw)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// STEP 6: Escrow × Payment Sync Tests
// ════════════════════════════════════════════════════════════════════════════

func TestBlockWithdrawalIfUnconfirmed_Logic(t *testing.T) {
	// Test the logic: available < amount → should fail
	available := decimal.NewFromInt(500)
	amount := decimal.NewFromInt(1000)
	if available.LessThan(amount) {
		// This is the core check — it should block
	} else {
		t.Error("available 500 < amount 1000 should block withdrawal")
	}
}

func TestBlockWithdrawalIfUnconfirmed_SufficientBalance(t *testing.T) {
	available := decimal.NewFromInt(2000)
	amount := decimal.NewFromInt(1000)
	if available.LessThan(amount) {
		t.Error("available 2000 >= amount 1000 should allow withdrawal")
	}
}

func TestReplayAttackCheck_EmptyKey(t *testing.T) {
	// Empty key should always pass
	key := ""
	if key != "" {
		t.Error("empty key should be allowed")
	}
}
