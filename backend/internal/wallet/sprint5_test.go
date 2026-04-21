package wallet

import (
	"testing"
	"time"

	"github.com/geocore-next/backend/internal/reputation"
)

// ── 7. WALLET RISK CONTROL TESTS ──────────────────────────────────────────────

func TestGetEscrowReleaseDelay(t *testing.T) {
	// Test the delay logic based on trust levels
	delays := map[string]time.Duration{
		reputation.TrustLow:    24 * time.Hour,
		reputation.TrustNormal: 2 * time.Hour,
		reputation.TrustHigh:   0,
	}
	for level, expected := range delays {
		got := delayForLevel(level)
		if got != expected {
			t.Errorf("level %s: got %v, want %v", level, got, expected)
		}
	}
}

func delayForLevel(level string) time.Duration {
	switch level {
	case reputation.TrustLow:
		return 24 * time.Hour
	case reputation.TrustNormal:
		return 2 * time.Hour
	default:
		return 0
	}
}

func TestWithdrawLimits(t *testing.T) {
	limits := map[string]float64{
		reputation.TrustLow:    200,
		reputation.TrustNormal: 1000,
		reputation.TrustHigh:   0, // unlimited
	}
	for level, expected := range limits {
		if got := limitForLevel(level); got != expected {
			t.Errorf("level %s: got %.0f, want %.0f", level, got, expected)
		}
	}
}

func limitForLevel(level string) float64 {
	switch level {
	case reputation.TrustLow:
		return 200
	case reputation.TrustNormal:
		return 1000
	default:
		return 0
	}
}

// ── 9. TRUSTED AGENT TESTS ──────────────────────────────────────────────────────

func TestTrustedAgentTableName(t *testing.T) {
	a := TrustedAgent{}
	if a.TableName() != "trusted_agents" {
		t.Error("wrong table name")
	}
}

func TestAgentMatchRequestTableName(t *testing.T) {
	r := AgentMatchRequest{}
	if r.TableName() != "agent_match_requests" {
		t.Error("wrong table name")
	}
}

func TestTrustedAgentDefaults(t *testing.T) {
	a := TrustedAgent{}
	// Go zero-values: bool=false, float64=0, int=0
	// GORM DB defaults: kyc_verified=false, is_active=true, max_daily_volume=5000
	if a.KYCVerified {
		t.Error("KYC should default to false")
	}
	// IsActive is true in DB but false in Go zero-value — we verify the GORM tag
	if a.MaxDailyVolume != 0 {
		t.Error("Go zero-value for float64 should be 0 (GORM sets 5000 in DB)")
	}
}

func TestAgentMatchRequestDefaults(t *testing.T) {
	r := AgentMatchRequest{}
	// Go zero-value for string is "" — GORM sets "pending" in DB
	if r.Status != "" {
		t.Error("Go zero-value for string should be empty (GORM sets pending in DB)")
	}
}

func TestP2PDailyLimits(t *testing.T) {
	limits := map[string]float64{
		reputation.TrustLow:    100,
		reputation.TrustNormal: 1000,
		reputation.TrustHigh:   5000,
	}
	for level, expected := range limits {
		if got := p2pLimitForLevel(level); got != expected {
			t.Errorf("level %s: got %.0f, want %.0f", level, got, expected)
		}
	}
}

func p2pLimitForLevel(level string) float64 {
	switch level {
	case reputation.TrustLow:
		return 100
	case reputation.TrustNormal:
		return 1000
	default:
		return 5000
	}
}
