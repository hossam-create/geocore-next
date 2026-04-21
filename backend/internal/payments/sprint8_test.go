package payments

import (
	"testing"

	"github.com/shopspring/decimal"
)

// ════════════════════════════════════════════════════════════════════════════
// Sprint 8: Dynamic Fees Tests
// ════════════════════════════════════════════════════════════════════════════

func TestDynamicFeeConfig_Defaults(t *testing.T) {
	cfg := DefaultDynamicFeeConfig
	if !cfg.LowSupplyFeePct.Equal(decimal.NewFromFloat(0.10)) {
		t.Errorf("low supply fee should be 10%%, got %s", cfg.LowSupplyFeePct.String())
	}
	if !cfg.BalancedFeePct.Equal(decimal.NewFromFloat(0.12)) {
		t.Errorf("balanced fee should be 12%%, got %s", cfg.BalancedFeePct.String())
	}
	if !cfg.HighDemandFeePct.Equal(decimal.NewFromFloat(0.15)) {
		t.Errorf("high demand fee should be 15%%, got %s", cfg.HighDemandFeePct.String())
	}
	if !cfg.VeryHighDemandFeePct.Equal(decimal.NewFromFloat(0.18)) {
		t.Errorf("very high demand fee should be 18%%, got %s", cfg.VeryHighDemandFeePct.String())
	}
	if !cfg.VIPDiscountPct.Equal(decimal.NewFromFloat(0.30)) {
		t.Errorf("VIP discount should be 30%%, got %s", cfg.VIPDiscountPct.String())
	}
}

func TestLiquidityLevels(t *testing.T) {
	levels := map[string]LiquidityLevel{
		"low_supply":      LiquidityLowSupply,
		"balanced":        LiquidityBalanced,
		"high_demand":     LiquidityHighDemand,
		"very_high_demand": LiquidityVeryHigh,
	}
	for expected, actual := range levels {
		if string(actual) != expected {
			t.Errorf("expected '%s', got '%s'", expected, actual)
		}
	}
}

func TestDynamicFeeResult_Fields(t *testing.T) {
	result := DynamicFeeResult{
		LiquidityLevel: LiquidityBalanced,
		BaseFeePct:     decimal.NewFromFloat(0.12),
		FinalFeePct:    decimal.NewFromFloat(0.12),
		FeeAmount:      decimal.NewFromFloat(12.00),
		IsVIP:          false,
	}
	if result.IsVIP {
		t.Error("non-VIP user should not have VIP flag")
	}
	if !result.FeeAmount.Equal(decimal.NewFromFloat(12.00)) {
		t.Errorf("fee amount should be 12.00, got %s", result.FeeAmount.String())
	}
}

func TestDynamicFee_VIPDiscount(t *testing.T) {
	cfg := DefaultDynamicFeeConfig
	baseFee := cfg.BalancedFeePct
	vipDiscount := baseFee.Mul(cfg.VIPDiscountPct)
	finalFee := baseFee.Sub(vipDiscount)

	expectedFinal := decimal.NewFromFloat(0.084) // 12% - (12% * 30%) = 8.4%
	if !finalFee.Equal(expectedFinal) {
		t.Errorf("VIP final fee should be %s, got %s", expectedFinal.String(), finalFee.String())
	}
}

func TestDynamicFee_FeeOrdering(t *testing.T) {
	cfg := DefaultDynamicFeeConfig
	if !cfg.LowSupplyFeePct.LessThan(cfg.BalancedFeePct) {
		t.Error("low supply fee should be less than balanced")
	}
	if !cfg.BalancedFeePct.LessThan(cfg.HighDemandFeePct) {
		t.Error("balanced fee should be less than high demand")
	}
	if !cfg.HighDemandFeePct.LessThan(cfg.VeryHighDemandFeePct) {
		t.Error("high demand fee should be less than very high")
	}
}
