package config

import (
	"os"
	"sync"
	"testing"
)

// ════════════════════════════════════════════════════════════════════════════
// STEP 3: Feature Flags Tests
// ════════════════════════════════════════════════════════════════════════════

func TestFeatureFlags_Defaults(t *testing.T) {
	flagsOnce = sync.Once{} // reset
	flags = FeatureFlags{}
	f := GetFlags()
	if !f.EnableAutoOffers {
		t.Error("auto offers should be enabled by default")
	}
	if !f.EnableDealCloser {
		t.Error("deal closer should be enabled by default")
	}
	if !f.EnableP2PMatching {
		t.Error("p2p matching should be enabled by default")
	}
	if !f.EnableDynamicFees {
		t.Error("dynamic fees should be enabled by default")
	}
	if !f.EnableRetention {
		t.Error("retention should be enabled by default")
	}
	if !f.EnableBootstrap {
		t.Error("bootstrap should be enabled by default")
	}
}

func TestFeatureFlags_DisableViaEnv(t *testing.T) {
	os.Setenv("ENABLE_AUTO_OFFERS", "false")
	defer os.Unsetenv("ENABLE_AUTO_OFFERS")
	flagsOnce = sync.Once{}
	flags = FeatureFlags{}

	f := GetFlags()
	if f.EnableAutoOffers {
		t.Error("auto offers should be disabled when ENABLE_AUTO_OFFERS=false")
	}
}

func TestFeatureFlags_DisableAll(t *testing.T) {
	os.Setenv("ENABLE_AUTO_OFFERS", "false")
	os.Setenv("ENABLE_DEAL_CLOSER", "false")
	os.Setenv("ENABLE_P2P", "false")
	os.Setenv("ENABLE_DYNAMIC_FEES", "false")
	os.Setenv("ENABLE_RETENTION", "false")
	os.Setenv("ENABLE_BOOTSTRAP", "false")
	defer func() {
		os.Unsetenv("ENABLE_AUTO_OFFERS")
		os.Unsetenv("ENABLE_DEAL_CLOSER")
		os.Unsetenv("ENABLE_P2P")
		os.Unsetenv("ENABLE_DYNAMIC_FEES")
		os.Unsetenv("ENABLE_RETENTION")
		os.Unsetenv("ENABLE_BOOTSTRAP")
	}()

	flagsOnce = sync.Once{}
	flags = FeatureFlags{}
	f := GetFlags()

	if f.EnableAutoOffers || f.EnableDealCloser || f.EnableP2PMatching || f.EnableDynamicFees || f.EnableRetention || f.EnableBootstrap {
		t.Error("all flags should be disabled")
	}
}

func TestFeatureFlags_SetFlag(t *testing.T) {
	flagsOnce = sync.Once{}
	flags = FeatureFlags{}
	_ = GetFlags() // initialize

	SetFlag("auto_offers", false)
	if flags.EnableAutoOffers {
		t.Error("SetFlag should disable auto_offers")
	}
	SetFlag("deal_closer", false)
	if flags.EnableDealCloser {
		t.Error("SetFlag should disable deal_closer")
	}
	SetFlag("p2p_matching", false)
	if flags.EnableP2PMatching {
		t.Error("SetFlag should disable p2p_matching")
	}
	SetFlag("dynamic_fees", false)
	if flags.EnableDynamicFees {
		t.Error("SetFlag should disable dynamic_fees")
	}
}

func TestFeatureFlags_ReloadFlags(t *testing.T) {
	os.Setenv("ENABLE_P2P", "false")
	defer os.Unsetenv("ENABLE_P2P")

	flagsOnce = sync.Once{}
	flags = FeatureFlags{}
	f := ReloadFlags()
	if f.EnableP2PMatching {
		t.Error("ReloadFlags should re-read ENV and disable P2P")
	}
}

func TestEnvBool(t *testing.T) {
	tests := []struct {
		key    string
		val    string
		def    bool
		expect bool
	}{
		{"TEST_BOOL_TRUE", "true", false, true},
		{"TEST_BOOL_FALSE", "false", true, false},
		{"TEST_BOOL_EMPTY", "", true, true},
		{"TEST_BOOL_1", "1", false, true},
		{"TEST_BOOL_0", "0", true, false},
	}
	for _, tt := range tests {
		if tt.val != "" {
			os.Setenv(tt.key, tt.val)
			defer os.Unsetenv(tt.key)
		}
		got := envBool(tt.key, tt.def)
		if got != tt.expect {
			t.Errorf("envBool(%s) = %v, want %v", tt.key, got, tt.expect)
		}
	}
}
