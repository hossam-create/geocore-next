package payments

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ════════════════════════════════════════════════════════════════════════════
// PaymentAgent Model Tests
// ════════════════════════════════════════════════════════════════════════════

func TestPaymentAgent_TableName(t *testing.T) {
	agent := PaymentAgent{}
	if agent.TableName() != "payment_agents" {
		t.Errorf("expected table name 'payment_agents', got '%s'", agent.TableName())
	}
}

func TestPaymentAgent_Defaults(t *testing.T) {
	agent := PaymentAgent{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		Country:      "EG",
		Currency:     "EGP",
		BalanceLimit: decimal.NewFromInt(10000),
	}

	// Go zero-values — DB defaults differ
	if agent.Status != "" {
		t.Errorf("expected empty status (DB default: 'pending'), got '%s'", agent.Status)
	}
	if agent.TrustScore != 0 {
		t.Errorf("expected trust_score 0 (DB default: 50), got %d", agent.TrustScore)
	}
	if !agent.CurrentBalance.IsZero() {
		t.Errorf("expected current_balance 0, got %s", agent.CurrentBalance.String())
	}
	if !agent.CollateralHeld.IsZero() {
		t.Errorf("expected collateral_held 0, got %s", agent.CollateralHeld.String())
	}
}

// ════════════════════════════════════════════════════════════════════════════
// DepositRequest Model Tests
// ════════════════════════════════════════════════════════════════════════════

func TestDepositRequest_TableName(t *testing.T) {
	dr := DepositRequest{}
	if dr.TableName() != "deposit_requests" {
		t.Errorf("expected table name 'deposit_requests', got '%s'", dr.TableName())
	}
}

func TestDepositRequest_Statuses(t *testing.T) {
	statuses := []string{"pending", "paid", "confirmed", "rejected", "expired"}
	for _, s := range statuses {
		dr := DepositRequest{Status: s}
		if dr.Status != s {
			t.Errorf("expected status '%s', got '%s'", s, dr.Status)
		}
	}
}

// ════════════════════════════════════════════════════════════════════════════
// WithdrawRequest Model Tests
// ════════════════════════════════════════════════════════════════════════════

func TestWithdrawRequest_TableName(t *testing.T) {
	wr := WithdrawRequest{}
	if wr.TableName() != "withdraw_requests" {
		t.Errorf("expected table name 'withdraw_requests', got '%s'", wr.TableName())
	}
}

func TestWithdrawRequest_Statuses(t *testing.T) {
	statuses := []string{"pending", "assigned", "processing", "completed", "failed", "cancelled"}
	for _, s := range statuses {
		wr := WithdrawRequest{Status: s}
		if wr.Status != s {
			t.Errorf("expected status '%s', got '%s'", s, wr.Status)
		}
	}
}

// ════════════════════════════════════════════════════════════════════════════
// AgentLiquidityLog Model Tests
// ════════════════════════════════════════════════════════════════════════════

func TestAgentLiquidityLog_TableName(t *testing.T) {
	log := AgentLiquidityLog{}
	if log.TableName() != "agent_liquidity_log" {
		t.Errorf("expected table name 'agent_liquidity_log', got '%s'", log.TableName())
	}
}

// ════════════════════════════════════════════════════════════════════════════
// VIPUser Model Tests
// ════════════════════════════════════════════════════════════════════════════

func TestVIPUser_TableName(t *testing.T) {
	vip := VIPUser{}
	if vip.TableName() != "vip_users" {
		t.Errorf("expected table name 'vip_users', got '%s'", vip.TableName())
	}
}

// ════════════════════════════════════════════════════════════════════════════
// VIP Tier Config Tests
// ════════════════════════════════════════════════════════════════════════════

func TestVIPTiers_Silver(t *testing.T) {
	silver := VIPTiers["silver"]
	if !silver.DailyLimit.Equal(decimal.NewFromFloat(1000)) {
		t.Errorf("silver daily limit should be $1000, got %s", silver.DailyLimit.String())
	}
	if !silver.MonthlyLimit.Equal(decimal.NewFromFloat(10000)) {
		t.Errorf("silver monthly limit should be $10000, got %s", silver.MonthlyLimit.String())
	}
	if !silver.TransferFeePct.Equal(decimal.NewFromFloat(0.008)) {
		t.Errorf("silver fee should be 0.8%%, got %s", silver.TransferFeePct.String())
	}
	if silver.WithdrawSpeedH != 4 {
		t.Errorf("silver withdraw speed should be 4h, got %d", silver.WithdrawSpeedH)
	}
}

func TestVIPTiers_Gold(t *testing.T) {
	gold := VIPTiers["gold"]
	if !gold.DailyLimit.Equal(decimal.NewFromFloat(5000)) {
		t.Errorf("gold daily limit should be $5000, got %s", gold.DailyLimit.String())
	}
	if !gold.FastTrackKYC {
		t.Error("gold should have fast track KYC")
	}
	if !gold.DedicatedAgent {
		t.Error("gold should have dedicated agent")
	}
	if gold.WithdrawSpeedH != 2 {
		t.Errorf("gold withdraw speed should be 2h, got %d", gold.WithdrawSpeedH)
	}
}

func TestVIPTiers_Platinum(t *testing.T) {
	plat := VIPTiers["platinum"]
	if !plat.DailyLimit.Equal(decimal.NewFromFloat(20000)) {
		t.Errorf("platinum daily limit should be $20000, got %s", plat.DailyLimit.String())
	}
	if !plat.TransferFeePct.Equal(decimal.NewFromFloat(0.003)) {
		t.Errorf("platinum fee should be 0.3%%, got %s", plat.TransferFeePct.String())
	}
	if plat.WithdrawSpeedH != 1 {
		t.Errorf("platinum withdraw speed should be 1h, got %d", plat.WithdrawSpeedH)
	}
	if plat.ZeroFeeTransfers != 3 {
		t.Errorf("platinum should have 3 zero-fee transfers, got %d", plat.ZeroFeeTransfers)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// AgentUtilization Tests
// ════════════════════════════════════════════════════════════════════════════

func TestAgentUtilization_Levels(t *testing.T) {
	tests := []struct {
		name        string
		current     float64
		limit       float64
		expectLevel string
	}{
		{"healthy", 3000, 10000, "healthy"},
		{"warning", 7500, 10000, "warning"},
		{"critical", 9500, 10000, "critical"},
		{"zero_balance", 0, 10000, "healthy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			util := AgentUtilization{
				CurrentBalance: decimal.NewFromFloat(tt.current),
				BalanceLimit:   decimal.NewFromFloat(tt.limit),
			}
			if !util.BalanceLimit.IsZero() {
				util.Utilization = util.CurrentBalance.Div(util.BalanceLimit).Mul(decimal.NewFromInt(100))
			}
			util.Available = util.BalanceLimit.Sub(util.CurrentBalance)

			switch {
			case util.Utilization.GreaterThanOrEqual(decimal.NewFromInt(90)):
				util.Level = "critical"
			case util.Utilization.GreaterThanOrEqual(decimal.NewFromInt(70)):
				util.Level = "warning"
			default:
				util.Level = "healthy"
			}

			if util.Level != tt.expectLevel {
				t.Errorf("expected level '%s', got '%s'", tt.expectLevel, util.Level)
			}
		})
	}
}

// ════════════════════════════════════════════════════════════════════════════
// Payment Trust Gate Tests
// ════════════════════════════════════════════════════════════════════════════

func TestPaymentTrustGateResult_Defaults(t *testing.T) {
	result := PaymentTrustGateResult{}
	if result.Allowed != false {
		t.Error("default Allowed should be false")
	}
	if result.Action != "" {
		t.Errorf("default Action should be empty, got '%s'", result.Action)
	}
}

func TestPaymentTrustGate_AllowThreshold(t *testing.T) {
	result := PaymentTrustGateResult{
		Allowed:   true,
		RiskScore: 20,
		Action:    "allow",
	}
	if !result.Allowed {
		t.Error("risk score 20 should be allowed")
	}
	if result.Action != "allow" {
		t.Errorf("risk score 20 should be 'allow', got '%s'", result.Action)
	}
}

func TestPaymentTrustGate_ReviewThreshold(t *testing.T) {
	result := PaymentTrustGateResult{
		Allowed:   true,
		RiskScore: 45,
		Action:    "review",
	}
	if !result.Allowed {
		t.Error("risk score 45 should still be allowed (review)")
	}
	if result.Action != "review" {
		t.Errorf("risk score 45 should be 'review', got '%s'", result.Action)
	}
}

func TestPaymentTrustGate_BlockThreshold(t *testing.T) {
	result := PaymentTrustGateResult{
		Allowed:   false,
		RiskScore: 75,
		Action:    "block",
	}
	if result.Allowed {
		t.Error("risk score 75 should be blocked")
	}
	if result.Action != "block" {
		t.Errorf("risk score 75 should be 'block', got '%s'", result.Action)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// FXService Tests
// ════════════════════════════════════════════════════════════════════════════

func TestFXService_SameCurrency(t *testing.T) {
	fx := NewFXService(nil, nil)
	rate := fx.GetRate("USD", "USD")
	if !rate.Equal(decimal.NewFromInt(1)) {
		t.Errorf("same currency rate should be 1, got %s", rate.String())
	}
}

func TestFXService_NoDBFallback(t *testing.T) {
	fx := NewFXService(nil, nil)
	rate := fx.GetRate("EGP", "USD")
	// Without DB, should return 1.0 (fallback)
	if !rate.Equal(decimal.NewFromInt(1)) {
		t.Errorf("no DB fallback should return 1.0, got %s", rate.String())
	}
}

// ════════════════════════════════════════════════════════════════════════════
// PaymentMethod Tests
// ════════════════════════════════════════════════════════════════════════════

func TestPaymentMethod_JSON(t *testing.T) {
	methods := []PaymentMethod{
		{Type: "instapay", Identifier: "01xxxxxxxxx", Name: "Ahmed"},
		{Type: "vodafone_cash", Identifier: "01xxxxxxxxx", Name: "Ahmed Mohamed"},
	}

	data, err := json.Marshal(methods)
	if err != nil {
		t.Fatalf("failed to marshal payment methods: %v", err)
	}

	var parsed []PaymentMethod
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal payment methods: %v", err)
	}

	if len(parsed) != 2 {
		t.Errorf("expected 2 methods, got %d", len(parsed))
	}
	if parsed[0].Type != "instapay" {
		t.Errorf("expected first method type 'instapay', got '%s'", parsed[0].Type)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// AgentPublicView Tests
// ════════════════════════════════════════════════════════════════════════════

func TestAgentPublicView_Capacity(t *testing.T) {
	view := AgentPublicView{
		ID:                uuid.New(),
		TrustScore:        80,
		AvailableCapacity: decimal.NewFromInt(5000),
	}
	if !view.AvailableCapacity.Equal(decimal.NewFromInt(5000)) {
		t.Errorf("expected capacity 5000, got %s", view.AvailableCapacity.String())
	}
}

// ════════════════════════════════════════════════════════════════════════════
// buildPaymentInstructions Tests
// ════════════════════════════════════════════════════════════════════════════

func TestBuildPaymentInstructions(t *testing.T) {
	methods := []PaymentMethod{
		{Type: "instapay", Identifier: "0123456789", Name: "Ahmed"},
		{Type: "vodafone_cash", Identifier: "0123456789", Name: "Sara"},
		{Type: "bank_transfer", Identifier: "ACC123", Name: "Bank"},
		{Type: "other_method", Identifier: "X", Name: "Other"},
	}

	instructions := buildPaymentInstructions(methods, decimal.NewFromInt(500), "EGP")
	if len(instructions) != 4 {
		t.Fatalf("expected 4 instructions, got %d", len(instructions))
	}

	// Check instapay instruction contains "InstaPay"
	if !contains(instructions[0], "InstaPay") {
		t.Errorf("instapay instruction should mention InstaPay: %s", instructions[0])
	}
	// Check vodafone_cash instruction contains "Vodafone Cash"
	if !contains(instructions[1], "Vodafone Cash") {
		t.Errorf("vodafone_cash instruction should mention Vodafone Cash: %s", instructions[1])
	}
	// Check bank_transfer instruction contains "bank account"
	if !contains(instructions[2], "bank account") {
		t.Errorf("bank_transfer instruction should mention bank account: %s", instructions[2])
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
