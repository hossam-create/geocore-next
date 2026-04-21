package crowdshipping

import (
	"testing"
)

// ── 1. CORRIDOR TESTS ─────────────────────────────────────────────────────────

func TestGetCorridorConfig_ValidUSEG(t *testing.T) {
	cfg := GetCorridorConfig("US", "EG")
	if cfg == nil {
		t.Fatal("US→EG corridor not found")
	}
	if cfg.Risk.CustomsMultiplier != 1.3 {
		t.Errorf("US→EG customs multiplier = %v, want 1.3", cfg.Risk.CustomsMultiplier)
	}
	if len(cfg.Risk.ValueBands) != 5 {
		t.Errorf("US→EG value bands = %d, want 5", len(cfg.Risk.ValueBands))
	}
	if len(cfg.Risk.DeliveryWindows) != 3 {
		t.Errorf("US→EG delivery windows = %d, want 3", len(cfg.Risk.DeliveryWindows))
	}
	if len(cfg.Restrictions) == 0 {
		t.Error("US→EG has no restrictions")
	}
	if cfg.EscrowPolicy != EscrowAlwaysRecommended {
		t.Errorf("US→EG escrow = %v, want ALWAYS_RECOMMENDED", cfg.EscrowPolicy)
	}
	if cfg.Version != 1 {
		t.Errorf("US→EG version = %d, want 1", cfg.Version)
	}
}

func TestGetCorridorConfig_ValidUSAE(t *testing.T) {
	cfg := GetCorridorConfig("US", "AE")
	if cfg == nil {
		t.Fatal("US→AE corridor not found")
	}
	if cfg.Risk.CustomsMultiplier != 1.1 {
		t.Errorf("US→AE customs multiplier = %v, want 1.1", cfg.Risk.CustomsMultiplier)
	}
}

func TestGetCorridorConfig_ValidUSSA(t *testing.T) {
	cfg := GetCorridorConfig("US", "SA")
	if cfg == nil {
		t.Fatal("US→SA corridor not found")
	}
	if cfg.Risk.CustomsMultiplier != 1.4 {
		t.Errorf("US→SA customs multiplier = %v, want 1.4", cfg.Risk.CustomsMultiplier)
	}
}

func TestGetCorridorConfig_InvalidEGBR(t *testing.T) {
	cfg := GetCorridorConfig("EG", "BR")
	if cfg != nil {
		t.Error("EG→BR should not exist, got non-nil config")
	}
}

func TestGetCorridorConfig_CaseInsensitive(t *testing.T) {
	cfg := GetCorridorConfig("us", "eg")
	if cfg == nil {
		t.Error("case-insensitive lookup failed for us→eg")
	}
}

func TestIsCorridorSupported(t *testing.T) {
	if !IsCorridorSupported("US", "EG") {
		t.Error("US→EG should be supported")
	}
	if IsCorridorSupported("XX", "YY") {
		t.Error("XX→YY should not be supported")
	}
}

func TestGetValueBandMultiplier(t *testing.T) {
	cfg := GetCorridorConfig("US", "EG")
	if cfg == nil {
		t.Fatal("no config")
	}

	vb := GetValueBandMultiplier(cfg, 50)
	if vb.Multiplier != 1.0 || vb.Label != "Low Value" {
		t.Errorf("50 USD: got mult=%v label=%q", vb.Multiplier, vb.Label)
	}
	vb = GetValueBandMultiplier(cfg, 150)
	if vb.Multiplier != 1.1 || vb.Label != "Standard" {
		t.Errorf("150 USD: got mult=%v label=%q", vb.Multiplier, vb.Label)
	}
	vb = GetValueBandMultiplier(cfg, 300)
	if vb.Multiplier != 1.3 || vb.Label != "Elevated" {
		t.Errorf("300 USD: got mult=%v label=%q", vb.Multiplier, vb.Label)
	}
	vb = GetValueBandMultiplier(cfg, 1000)
	if vb.Multiplier != 1.5 || vb.Label != "High Value" {
		t.Errorf("1000 USD: got mult=%v label=%q", vb.Multiplier, vb.Label)
	}
	vb = GetValueBandMultiplier(cfg, 5000)
	if vb.Multiplier != 2.0 || vb.Label != "Very High" {
		t.Errorf("5000 USD: got mult=%v label=%q", vb.Multiplier, vb.Label)
	}
}

func TestGetValueBandMultiplier_BoundaryAt100(t *testing.T) {
	cfg := GetCorridorConfig("US", "EG")
	vb := GetValueBandMultiplier(cfg, 100)
	if vb.Label != "Standard" {
		t.Errorf("exactly 100 should be 'Standard', got %q", vb.Label)
	}
}

func TestGetDeliveryWindowMultiplier(t *testing.T) {
	cfg := GetCorridorConfig("US", "EG")
	dw := GetDeliveryWindowMultiplier(cfg, 3)
	if dw.Multiplier != 1.3 || dw.Label != "Express" {
		t.Errorf("3 days: got mult=%v label=%q", dw.Multiplier, dw.Label)
	}
	dw = GetDeliveryWindowMultiplier(cfg, 10)
	if dw.Multiplier != 1.0 || dw.Label != "Standard" {
		t.Errorf("10 days: got mult=%v label=%q", dw.Multiplier, dw.Label)
	}
	dw = GetDeliveryWindowMultiplier(cfg, 20)
	if dw.Multiplier != 0.9 || dw.Label != "Economy" {
		t.Errorf("20 days: got mult=%v label=%q", dw.Multiplier, dw.Label)
	}
}

func TestIsRestricted(t *testing.T) {
	cfg := GetCorridorConfig("US", "AE")
	if !IsRestricted(cfg, "alcohol") {
		t.Error("alcohol should be restricted for US→AE")
	}
	if !IsRestricted(cfg, "Alcohol") {
		t.Error("Alcohol (capitalized) should be restricted for US→AE")
	}
	if IsRestricted(cfg, "clothing") {
		t.Error("clothing should NOT be restricted for US→AE")
	}
	if IsRestricted(nil, "anything") {
		t.Error("nil config should not restrict anything")
	}
}

// ── 2. PRICING TESTS ─────────────────────────────────────────────────────────

func TestPricing_CaseA_LowValue(t *testing.T) {
	bd := CalculateDeliveryPrice(PricingParams{
		WeightKg: 1, DistanceKm: 100, Urgency: UrgencyStandard,
		ItemType: ItemTypeClothing, ItemValue: 50, Origin: "US", Destination: "EG",
	})
	if bd.BaseFee != 5.0 {
		t.Errorf("base_fee = %v, want 5.0", bd.BaseFee)
	}
	if bd.WeightCost != 2.0 {
		t.Errorf("weight_cost = %v, want 2.0", bd.WeightCost)
	}
	if bd.DistanceCost != 50.0 {
		t.Errorf("distance_cost = %v, want 50.0", bd.DistanceCost)
	}
	if bd.ItemTypeFee != 0.0 {
		t.Errorf("item_type_fee = %v, want 0.0", bd.ItemTypeFee)
	}
	if bd.UrgencyMultiplier != 1.0 {
		t.Errorf("urgency_multiplier = %v, want 1.0", bd.UrgencyMultiplier)
	}
	if bd.CorridorCustoms != 1.3 {
		t.Errorf("corridor_customs = %v, want 1.3", bd.CorridorCustoms)
	}
	if bd.ValueBandMult != 1.0 {
		t.Errorf("value_band_mult = %v, want 1.0", bd.ValueBandMult)
	}
	expectedSubtotal := 5.0 + 2.0 + 50.0 + 0.0
	if bd.Subtotal != expectedSubtotal {
		t.Errorf("subtotal = %v, want %v", bd.Subtotal, expectedSubtotal)
	}
	// Total = subtotal * 1.0 * 1.3 * 1.0 = 74.10
	expectedTotalCents := int64(7410)
	if bd.TotalCents != expectedTotalCents {
		t.Errorf("total_cents = %d, want %d", bd.TotalCents, expectedTotalCents)
	}
	if bd.Total <= 0 {
		t.Error("total should be positive")
	}
	if bd.Currency != "USD" {
		t.Errorf("currency = %q, want USD", bd.Currency)
	}
}

func TestPricing_CaseB_HighValue(t *testing.T) {
	bd := CalculateDeliveryPrice(PricingParams{
		WeightKg: 10, DistanceKm: 5000, Urgency: UrgencyExpress,
		ItemType: ItemTypeElectronics, ItemValue: 800, Origin: "US", Destination: "AE",
	})
	if bd.WeightCost != 20.0 {
		t.Errorf("weight_cost = %v, want 20.0", bd.WeightCost)
	}
	if bd.DistanceCost != 2500.0 {
		t.Errorf("distance_cost = %v, want 2500.0", bd.DistanceCost)
	}
	if bd.ItemTypeFee != 3.0 {
		t.Errorf("item_type_fee = %v, want 3.0", bd.ItemTypeFee)
	}
	if bd.UrgencyMultiplier != 1.5 {
		t.Errorf("urgency_multiplier = %v, want 1.5", bd.UrgencyMultiplier)
	}
	if bd.CorridorCustoms != 1.1 {
		t.Errorf("corridor_customs = %v, want 1.1", bd.CorridorCustoms)
	}
	if bd.ValueBandMult != 1.4 {
		t.Errorf("value_band_mult = %v, want 1.4", bd.ValueBandMult)
	}
	if bd.Total <= 0 {
		t.Error("total should be positive")
	}
	if bd.Total > 10000 {
		t.Logf("total = %v (high but plausible for 5000km express)", bd.Total)
	}
}

func TestPricing_CaseC_Extreme(t *testing.T) {
	bd := CalculateDeliveryPrice(PricingParams{
		WeightKg: 50, DistanceKm: 10000, Urgency: UrgencySameDay,
		ItemType: ItemTypeFragile, ItemValue: 3000, Origin: "US", Destination: "SA",
	})
	if bd.UrgencyMultiplier != 2.0 {
		t.Errorf("urgency_multiplier = %v, want 2.0", bd.UrgencyMultiplier)
	}
	if bd.CorridorCustoms != 1.4 {
		t.Errorf("corridor_customs = %v, want 1.4", bd.CorridorCustoms)
	}
	if bd.ValueBandMult != 2.2 {
		t.Errorf("value_band_mult = %v, want 2.2", bd.ValueBandMult)
	}
	if bd.Total <= 0 {
		t.Error("total should be positive")
	}
}

func TestPricing_PlatformTravelerSplit(t *testing.T) {
	bd := CalculateDeliveryPrice(PricingParams{
		WeightKg: 5, DistanceKm: 500, Urgency: UrgencyStandard,
		ItemType: ItemTypeClothing, ItemValue: 100, Origin: "US", Destination: "EG",
	})
	// EXACT invariant: PlatformFeeCents + TravelerEarningsCents == TotalCents
	if bd.PlatformFeeCents+bd.TravelerEarningsCents != bd.TotalCents {
		t.Errorf("platform_cents(%d) + traveler_cents(%d) = %d, want total_cents %d",
			bd.PlatformFeeCents, bd.TravelerEarningsCents,
			bd.PlatformFeeCents+bd.TravelerEarningsCents, bd.TotalCents)
	}
	// Float representation within ±0.01 (IEEE 754 addition drift)
	diff := bd.Total - (bd.PlatformFee + bd.TravelerEarnings)
	if diff > 0.01 || diff < -0.01 {
		t.Errorf("platform(%v) + traveler(%v) = %v, want total %v (drift=%v)",
			bd.PlatformFee, bd.TravelerEarnings, bd.PlatformFee+bd.TravelerEarnings, bd.Total, diff)
	}
}

func TestPricing_UnknownItemType(t *testing.T) {
	bd := CalculateDeliveryPrice(PricingParams{
		WeightKg: 1, DistanceKm: 100, Urgency: UrgencyStandard,
		ItemType: "unknown_thing", ItemValue: 50, Origin: "US", Destination: "EG",
	})
	if bd.ItemTypeFee != 1.0 {
		t.Errorf("unknown item type fee = %v, want 1.0 (Other default)", bd.ItemTypeFee)
	}
}

func TestPricing_UnknownUrgency(t *testing.T) {
	bd := CalculateDeliveryPrice(PricingParams{
		WeightKg: 1, DistanceKm: 100, Urgency: "hyper_speed",
		ItemType: ItemTypeClothing, ItemValue: 50, Origin: "US", Destination: "EG",
	})
	if bd.UrgencyMultiplier != 1.0 {
		t.Errorf("unknown urgency multiplier = %v, want 1.0 (Standard default)", bd.UrgencyMultiplier)
	}
}

func TestPricing_UnsupportedCorridor(t *testing.T) {
	bd := CalculateDeliveryPrice(PricingParams{
		WeightKg: 1, DistanceKm: 100, Urgency: UrgencyStandard,
		ItemType: ItemTypeClothing, ItemValue: 50, Origin: "XX", Destination: "YY",
	})
	if bd.CorridorCustoms != 1.0 {
		t.Errorf("unsupported corridor customs = %v, want 1.0", bd.CorridorCustoms)
	}
	if bd.ValueBandLabel != "Default" {
		t.Errorf("unsupported corridor value band label = %q, want Default", bd.ValueBandLabel)
	}
}

// ── 3. COMPLIANCE TESTS ──────────────────────────────────────────────────────

func TestCompliance_CaseA_Amount500(t *testing.T) {
	r := CheckCompliance("US", "EG", 500)
	if !r.Allowed {
		t.Error("500 USD should be allowed")
	}
	if r.KYCRequired {
		t.Error("500 USD should NOT require KYC")
	}
	if !r.EscrowRecommended {
		t.Error("US→EG should recommend escrow (ALWAYS_RECOMMENDED)")
	}
	if r.MinBuyerTrust != TrustTrusted {
		t.Errorf("min_buyer_trust = %v, want TRUSTED", r.MinBuyerTrust)
	}
}

func TestCompliance_CaseB_Amount2000(t *testing.T) {
	r := CheckCompliance("US", "EG", 2000)
	if !r.Allowed {
		t.Error("2000 USD should be allowed (with KYC)")
	}
	if !r.KYCRequired {
		t.Error("2000 USD SHOULD require KYC")
	}
	kycFound := false
	for _, b := range r.Boundaries {
		if b.Type == "KYC_REQUIREMENT" {
			kycFound = true
		}
	}
	if !kycFound {
		t.Error("KYC_REQUIREMENT boundary missing for $2000")
	}
}

func TestCompliance_CaseC_Amount15000(t *testing.T) {
	r := CheckCompliance("US", "EG", 15000)
	if !r.Allowed {
		t.Error("15000 USD should be allowed (enhanced review is soft by default)")
	}
	if !r.KYCRequired {
		t.Error("15000 USD SHOULD require KYC")
	}
	crossBorderFound := false
	for _, b := range r.Boundaries {
		if b.Type == "CROSS_BORDER_LIMIT" {
			crossBorderFound = true
			if b.EnforcementLevel != EnforcementSoft {
				t.Errorf("cross border enforcement = %v, want SOFT_LIMIT", b.EnforcementLevel)
			}
		}
	}
	if !crossBorderFound {
		t.Error("CROSS_BORDER_LIMIT boundary missing for $15000")
	}
}

func TestCompliance_CaseD_RestrictedItems(t *testing.T) {
	r := CheckCompliance("US", "AE", 300, "alcohol", "pork_products", "clothing")
	if len(r.ItemRestrictions) != 2 {
		t.Errorf("expected 2 restricted items, got %d: %v", len(r.ItemRestrictions), r.ItemRestrictions)
	}
	restrictedCount := 0
	for _, w := range r.Warnings {
		if len(w) >= 14 && w[:14] == "Item category " {
			restrictedCount++
		}
	}
	if restrictedCount != 2 {
		t.Errorf("expected 2 restriction warnings, got %d", restrictedCount)
	}
}

func TestCompliance_SanctionedCountry(t *testing.T) {
	r := CheckCompliance("US", "IR", 500)
	if r.Allowed {
		t.Error("IR (Iran) should be BLOCKED by sanctions")
	}
	sanctionsFound := false
	for _, b := range r.Boundaries {
		if b.Type == "SANCTIONS" {
			sanctionsFound = true
			if b.Status != "BLOCKED" {
				t.Errorf("sanctions status = %q, want BLOCKED", b.Status)
			}
		}
	}
	if !sanctionsFound {
		t.Error("SANCTIONS boundary missing for Iran")
	}
}

func TestCompliance_SanctionedCountryCaseSensitive(t *testing.T) {
	r := CheckCompliance("US", "ir", 500)
	if r.Allowed {
		t.Error("lowercase 'ir' should still be blocked — SANCTIONS CHECK IS CASE-SENSITIVE (BUG)")
	}
}

func TestCompliance_EgyptCurrencyRestriction(t *testing.T) {
	r := CheckCompliance("US", "EG", 100)
	found := false
	for _, b := range r.Boundaries {
		if b.Type == "CURRENCY_RESTRICTION" {
			found = true
		}
	}
	if !found {
		t.Error("EG destination should have CURRENCY_RESTRICTION boundary")
	}
}

func TestCompliance_UnsupportedCorridor(t *testing.T) {
	r := CheckCompliance("XX", "YY", 300)
	if !r.Allowed {
		t.Error("unsupported corridor should still be allowed (with defaults)")
	}
	if r.MinBuyerTrust != TrustStandard {
		t.Errorf("unsupported corridor min_buyer_trust = %v, want STANDARD", r.MinBuyerTrust)
	}
	if !r.EscrowRecommended {
		t.Error("300 USD on unsupported corridor should recommend escrow (>= 200 default)")
	}
}

func TestCompliance_NoCategories(t *testing.T) {
	r := CheckCompliance("US", "EG", 100)
	if len(r.ItemRestrictions) != 0 {
		t.Errorf("no categories passed, but got restrictions: %v", r.ItemRestrictions)
	}
}

// ── 4. EDGE CASES ────────────────────────────────────────────────────────────

func TestGetValueBandMultiplier_NilConfig(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("GetValueBandMultiplier with nil config panicked: %v", r)
		}
	}()
	// This WILL panic because it dereferences nil CorridorConfig
	_ = GetValueBandMultiplier(nil, 100)
}

func TestGetDeliveryWindowMultiplier_NilConfig(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("GetDeliveryWindowMultiplier with nil config panicked: %v", r)
		}
	}()
	_ = GetDeliveryWindowMultiplier(nil, 5)
}

func TestPricing_ZeroDistance(t *testing.T) {
	bd := CalculateDeliveryPrice(PricingParams{
		WeightKg: 1, DistanceKm: 0, Urgency: UrgencyStandard,
		ItemType: ItemTypeClothing, ItemValue: 50, Origin: "US", Destination: "EG",
	})
	if bd.DistanceCost != 0 {
		t.Errorf("zero distance cost = %v, want 0", bd.DistanceCost)
	}
	if bd.Total <= 0 {
		t.Error("total should still be positive with zero distance")
	}
}

func TestValueBand_MaxValueJSONSafe(t *testing.T) {
	cfg := GetCorridorConfig("US", "EG")
	lastBand := cfg.Risk.ValueBands[len(cfg.Risk.ValueBands)-1]
	if lastBand.MaxValue != 999999999 {
		t.Errorf("last band max = %v, want 999999999 (JSON-safe sentinel)", lastBand.MaxValue)
	}
}

// ── 5. PRODUCTION HARDENING TESTS ─────────────────────────────────────────────

func TestCompliance_HardBlock_Over50K(t *testing.T) {
	r := CheckCompliance("US", "EG", 60000)
	if r.Allowed {
		t.Error("60000 USD should be HARD BLOCKED (>$50,000)")
	}
	hardBlockFound := false
	for _, b := range r.Boundaries {
		if b.Type == "HARD_BLOCK" {
			hardBlockFound = true
			if b.EnforcementLevel != EnforcementHard {
				t.Errorf("hard block enforcement = %v, want HARD_LIMIT", b.EnforcementLevel)
			}
			if b.Status != "BLOCKED" {
				t.Errorf("hard block status = %q, want BLOCKED", b.Status)
			}
		}
	}
	if !hardBlockFound {
		t.Error("HARD_BLOCK boundary missing for $60000")
	}
}

func TestCompliance_HardBlock_Exactly50K(t *testing.T) {
	// Exactly 50000 is NOT > 50000, so should still be allowed
	r := CheckCompliance("US", "EG", 50000)
	if !r.Allowed {
		t.Error("exactly $50,000 should be allowed (threshold is > not >=)")
	}
}

func TestCompliance_EnhancedReviewHardEnforcement(t *testing.T) {
	// Override thresholds to make enhanced review hard
	origThresholds := GetComplianceThresholds()
	customThresholds := origThresholds
	customThresholds.EnhancedReviewEnforcement = EnforcementHard
	SetComplianceThresholds(customThresholds)
	defer SetComplianceThresholds(origThresholds)

	r := CheckCompliance("US", "EG", 15000)
	if r.Allowed {
		t.Error("15000 USD should be BLOCKED when enhanced review enforcement is HARD")
	}
	for _, b := range r.Boundaries {
		if b.Type == "CROSS_BORDER_LIMIT" && b.EnforcementLevel != EnforcementHard {
			t.Errorf("cross border enforcement = %v, want HARD_LIMIT", b.EnforcementLevel)
		}
	}
}

func TestCompliance_EnforcementLevelOnBoundaries(t *testing.T) {
	r := CheckCompliance("US", "EG", 2000)
	for _, b := range r.Boundaries {
		if b.EnforcementLevel == "" {
			t.Errorf("boundary %q has empty enforcement_level", b.Type)
		}
	}
}

func TestPricing_ExactCentsInvariant(t *testing.T) {
	// Test multiple scenarios to ensure cents never drift
	scenarios := []PricingParams{
		{WeightKg: 1, DistanceKm: 100, Urgency: UrgencyStandard, ItemType: ItemTypeClothing, ItemValue: 50, Origin: "US", Destination: "EG"},
		{WeightKg: 10, DistanceKm: 5000, Urgency: UrgencyExpress, ItemType: ItemTypeElectronics, ItemValue: 800, Origin: "US", Destination: "AE"},
		{WeightKg: 50, DistanceKm: 10000, Urgency: UrgencySameDay, ItemType: ItemTypeFragile, ItemValue: 3000, Origin: "US", Destination: "SA"},
		{WeightKg: 0.5, DistanceKm: 0, Urgency: UrgencyStandard, ItemType: ItemTypeFood, ItemValue: 10, Origin: "US", Destination: "EG"},
	}
	for i, p := range scenarios {
		bd := CalculateDeliveryPrice(p)
		// Cents invariant is the financial guarantee — always exact
		if bd.PlatformFeeCents+bd.TravelerEarningsCents != bd.TotalCents {
			t.Errorf("scenario %d: cents drift: platform(%d)+traveler(%d)=%d, want total(%d)",
				i, bd.PlatformFeeCents, bd.TravelerEarningsCents,
				bd.PlatformFeeCents+bd.TravelerEarningsCents, bd.TotalCents)
		}
		// Float values derived from cents — allow ±0.01 tolerance for IEEE 754 addition drift
		diff := bd.Total - (bd.PlatformFee + bd.TravelerEarnings)
		if diff > 0.01 || diff < -0.01 {
			t.Errorf("scenario %d: float drift exceeds 1 cent: platform(%v)+traveler(%v)=%v, want total(%v)",
				i, bd.PlatformFee, bd.TravelerEarnings,
				bd.PlatformFee+bd.TravelerEarnings, bd.Total)
		}
	}
}

func TestCorridorVersion(t *testing.T) {
	for _, route := range []struct{ o, d string }{{"US", "EG"}, {"US", "AE"}, {"US", "SA"}} {
		cfg := GetCorridorConfig(route.o, route.d)
		if cfg == nil {
			t.Fatalf("%s→%s corridor not found", route.o, route.d)
		}
		if cfg.Version < 1 {
			t.Errorf("%s→%s version = %d, want >= 1", route.o, route.d, cfg.Version)
		}
	}
}

func TestCorridorRepository_Interface(t *testing.T) {
	repo := GetCorridorRepository()
	if repo == nil {
		t.Fatal("global repo is nil")
	}
	configs := repo.FindAll()
	if len(configs) == 0 {
		t.Error("FindAll returned no corridors")
	}
	cfg := repo.FindByRoute("US", "EG")
	if cfg == nil {
		t.Error("FindByRoute US→EG returned nil")
	}
	cfg = repo.FindByRoute("XX", "YY")
	if cfg != nil {
		t.Error("FindByRoute XX→YY should return nil")
	}
}

func TestComplianceThresholds_Defaults(t *testing.T) {
	th := DefaultComplianceThresholds()
	if th.KYCThresholdUSD != 1000 {
		t.Errorf("KYC threshold = %v, want 1000", th.KYCThresholdUSD)
	}
	if th.EnhancedReviewUSD != 10000 {
		t.Errorf("enhanced review = %v, want 10000", th.EnhancedReviewUSD)
	}
	if th.HardBlockUSD != 50000 {
		t.Errorf("hard block = %v, want 50000", th.HardBlockUSD)
	}
	if th.DefaultEscrowUSD != 200 {
		t.Errorf("default escrow = %v, want 200", th.DefaultEscrowUSD)
	}
	if th.EnhancedReviewEnforcement != EnforcementSoft {
		t.Errorf("enhanced review enforcement = %v, want SOFT_LIMIT", th.EnhancedReviewEnforcement)
	}
}

func TestCheckComplianceWithConfig_CustomThresholds(t *testing.T) {
	customThresholds := ComplianceThresholds{
		KYCThresholdUSD:           500,
		EnhancedReviewUSD:         5000,
		HardBlockUSD:              25000,
		CrossBorderReportUSD:      5000,
		DefaultEscrowUSD:          100,
		EnhancedReviewEnforcement: EnforcementSoft,
	}
	r := CheckComplianceWithConfig("US", "EG", 600, customThresholds)
	if !r.KYCRequired {
		t.Error("600 USD should require KYC with custom threshold 500")
	}
	r = CheckComplianceWithConfig("US", "EG", 30000, customThresholds)
	if r.Allowed {
		t.Error("30000 USD should be hard blocked with custom threshold 25000")
	}
}

func TestCompliance_SanctionsBoundaryEnforcement(t *testing.T) {
	r := CheckCompliance("US", "IR", 500)
	for _, b := range r.Boundaries {
		if b.Type == "SANCTIONS" && b.EnforcementLevel != EnforcementHard {
			t.Errorf("sanctions enforcement = %v, want HARD_LIMIT", b.EnforcementLevel)
		}
	}
}
