"use client";

import { useState, useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import { settingsApi } from "@/lib/api";
import { useToastStore } from "@/lib/toast";
import { Calculator, DollarSign, Percent } from "lucide-react";

type FeeTier = { min: number; max: number; rate: number };

type CategoryOverride = { category: string; rate_adjustment: number };

const DEFAULT_OVERRIDES: CategoryOverride[] = [
  { category: "Electronics", rate_adjustment: 0 },
  { category: "Jewelry", rate_adjustment: -0.5 },
  { category: "Automotive", rate_adjustment: +1.0 },
  { category: "Fashion", rate_adjustment: 0 },
  { category: "Collectibles", rate_adjustment: -1.0 },
];

const DEFAULT_TIERS: FeeTier[] = [
  { min: 0, max: 100, rate: 5.0 },
  { min: 100, max: 500, rate: 4.0 },
  { min: 500, max: 2000, rate: 3.0 },
  { min: 2000, max: 0, rate: 2.5 },
];

export default function PricingSettingsPage() {
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [tiers, setTiers] = useState<FeeTier[]>(DEFAULT_TIERS);
  const [calcSale, setCalcSale] = useState(500);
  const [saving, setSaving] = useState(false);
  const [overrides, setOverrides] = useState<CategoryOverride[]>(DEFAULT_OVERRIDES);

  const { data: liveSettings } = useQuery({
    queryKey: ["admin", "settings", "pricing"],
    queryFn: () => settingsApi.getByCategory("pricing"),
    retry: 1,
  });

  useEffect(() => {
    if (liveSettings && Array.isArray(liveSettings)) {
      const settings = liveSettings as Record<string, unknown>[];
      const feeSchedule = settings.find((s) => s.key === "pricing.fee_schedule");
      if (feeSchedule) {
        try {
          const parsed = JSON.parse(String(feeSchedule.value));
          if (Array.isArray(parsed)) setTiers(parsed.map((t: Record<string, unknown>) => ({ min: Number(t.min), max: Number(t.max), rate: Number(t.rate) })));
        } catch { /* ignore */ }
      }
      const overrideSetting = settings.find((s) => s.key === "pricing.category_overrides");
      if (overrideSetting) {
        try {
          const parsed = JSON.parse(String(overrideSetting.value));
          if (Array.isArray(parsed)) setOverrides(parsed.map((o: Record<string, unknown>) => ({ category: String(o.category), rate_adjustment: Number(o.rate_adjustment) })));
        } catch { /* ignore */ }
      }
    }
  }, [liveSettings]);

  const calcFee = (sale: number): { tier: FeeTier; fee: number; sellerReceives: number } => {
    const tier = tiers.find((t) => sale >= t.min && (t.max === 0 || sale < t.max)) ?? tiers[tiers.length - 1];
    const fee = sale * (tier.rate / 100);
    return { tier, fee, sellerReceives: sale - fee };
  };

  const result = calcFee(calcSale);

  const handleSave = async () => {
    setSaving(true);
    try {
      await settingsApi.update("pricing.fee_schedule", JSON.stringify(tiers));
      qc.invalidateQueries({ queryKey: ["admin", "settings", "pricing"] });
      showToast({ type: "success", title: "Pricing saved", message: "Fee schedule updated." });
    } catch (error: unknown) {
      showToast({ type: "error", title: "Save failed", message: (error as { message?: string })?.message ?? "Could not save pricing." });
    } finally {
      setSaving(false);
    }
  };

  const updateTier = (idx: number, field: keyof FeeTier, val: number) => {
    const updated = [...tiers];
    updated[idx] = { ...updated[idx], [field]: val };
    setTiers(updated);
  };

  const addTier = () => {
    const lastMax = tiers[tiers.length - 1]?.max ?? 0;
    setTiers([...tiers, { min: lastMax, max: 0, rate: 2.0 }]);
  };

  const removeTier = (idx: number) => {
    if (tiers.length <= 1) return;
    setTiers(tiers.filter((_, i) => i !== idx));
  };

  return (
    <div>
      <PageHeader title="Dynamic Pricing Controls" description="Fee schedule, live calculator, and category overrides" />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="surface p-5 rounded-lg">
          <h3 className="text-sm font-semibold mb-4 flex items-center gap-2" style={{ color: "var(--text-primary)" }}>
            <Percent className="w-4 h-4" />Fee Schedule
          </h3>
          <div className="space-y-2">
            <div className="grid grid-cols-4 gap-2 text-xs font-semibold uppercase tracking-wider pb-2" style={{ color: "var(--text-tertiary)", borderBottom: "1px solid var(--border-default)" }}>
              <span>Min ($)</span><span>Max ($)</span><span>Rate (%)</span><span></span>
            </div>
            {tiers.map((t, idx) => (
              <div key={idx} className="grid grid-cols-4 gap-2 items-center">
                <input type="number" value={t.min} onChange={(e) => updateTier(idx, "min", parseFloat(e.target.value) || 0)}
                  className="px-2 py-1.5 border rounded text-sm" style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }} />
                <input type="number" value={t.max} onChange={(e) => updateTier(idx, "max", parseFloat(e.target.value) || 0)}
                  className="px-2 py-1.5 border rounded text-sm" style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }} />
                <input type="number" step="0.1" value={t.rate} onChange={(e) => updateTier(idx, "rate", parseFloat(e.target.value) || 0)}
                  className="px-2 py-1.5 border rounded text-sm font-mono" style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }} />
                <button onClick={() => removeTier(idx)} className="text-xs px-2 py-1 rounded" style={{ color: "var(--color-danger)" }}>Remove</button>
              </div>
            ))}
          </div>
          <div className="flex gap-2 mt-4">
            <button onClick={addTier} className="px-3 py-1.5 rounded-lg text-sm font-medium" style={{ background: "var(--bg-surface)", border: "1px solid var(--border-default)", color: "var(--text-secondary)" }}>
              Add Tier
            </button>
            <button onClick={handleSave} disabled={saving} className="px-4 py-1.5 rounded-lg text-sm font-medium text-white ml-auto" style={{ background: "var(--color-brand)" }}>
              {saving ? "Saving..." : "Save Schedule"}
            </button>
          </div>
        </div>

        <div className="surface p-5 rounded-lg">
          <h3 className="text-sm font-semibold mb-4 flex items-center gap-2" style={{ color: "var(--text-primary)" }}>
            <Calculator className="w-4 h-4" />Live Calculator
          </h3>
          <p className="text-xs mb-3" style={{ color: "var(--text-tertiary)" }}>Enter a sale price to see the fee breakdown in real-time.</p>
          <div className="mb-4">
            <label className="text-xs font-medium block mb-1" style={{ color: "var(--text-tertiary)" }}>Sale Price ($)</label>
            <input
              type="number"
              value={calcSale}
              onChange={(e) => setCalcSale(parseFloat(e.target.value) || 0)}
              className="w-full px-3 py-2 border rounded-lg text-lg font-bold focus:outline-none focus:ring-2 focus:ring-blue-500"
              style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }}
            />
          </div>
          <div className="space-y-3 p-4 rounded-lg" style={{ background: "var(--bg-inset)" }}>
            <div className="flex justify-between text-sm">
              <span style={{ color: "var(--text-tertiary)" }}>Applicable Tier</span>
              <span className="font-medium" style={{ color: "var(--text-primary)" }}>
                ${result.tier.min}{result.tier.max > 0 ? ` – $${result.tier.max}` : "+"} @ {result.tier.rate}%
              </span>
            </div>
            <div className="flex justify-between text-sm">
              <span style={{ color: "var(--text-tertiary)" }}>Platform Fee</span>
              <span className="font-medium" style={{ color: "var(--color-danger)" }}>
                <DollarSign className="w-3.5 h-3.5 inline" />{result.fee.toFixed(2)}
              </span>
            </div>
            <div className="flex justify-between text-sm pt-2" style={{ borderTop: "1px solid var(--border-default)" }}>
              <span className="font-medium" style={{ color: "var(--text-primary)" }}>Seller Receives</span>
              <span className="text-lg font-bold" style={{ color: "var(--color-success)" }}>
                ${result.sellerReceives.toFixed(2)}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Category-Specific Overrides */}
      <div className="surface p-5 rounded-lg mt-4">
        <h3 className="text-sm font-semibold mb-1" style={{ color: "var(--text-primary)" }}>Category-Specific Fee Overrides</h3>
        <p className="text-xs mb-4" style={{ color: "var(--text-tertiary)" }}>
          Adjust the base fee rate per category. A positive value increases the fee, negative decreases it.
        </p>
        <div className="space-y-2">
          <div className="grid grid-cols-3 gap-3 text-xs font-semibold uppercase tracking-wider pb-2" style={{ color: "var(--text-tertiary)", borderBottom: "1px solid var(--border-default)" }}>
            <span>Category</span><span>Rate Adjustment (%)</span><span>Effective Rate</span>
          </div>
          {overrides.map((o, idx) => {
            const baseTier = tiers.find((t) => 500 >= t.min && (t.max === 0 || 500 < t.max)) ?? tiers[0];
            const effective = (baseTier?.rate ?? 3) + o.rate_adjustment;
            return (
              <div key={o.category} className="grid grid-cols-3 gap-3 items-center">
                <span className="text-sm" style={{ color: "var(--text-secondary)" }}>{o.category}</span>
                <input
                  type="number"
                  step="0.1"
                  value={o.rate_adjustment}
                  onChange={(e) => {
                    const updated = [...overrides];
                    updated[idx] = { ...updated[idx], rate_adjustment: parseFloat(e.target.value) || 0 };
                    setOverrides(updated);
                  }}
                  className="px-2 py-1.5 border rounded text-sm font-mono"
                  style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: o.rate_adjustment > 0 ? "var(--color-danger)" : o.rate_adjustment < 0 ? "var(--color-success)" : "var(--text-primary)" }}
                />
                <span className="text-sm font-mono" style={{ color: "var(--text-secondary)" }}>{effective.toFixed(1)}%</span>
              </div>
            );
          })}
        </div>
        <button
          onClick={async () => {
            setSaving(true);
            try {
              await settingsApi.update("pricing.category_overrides", JSON.stringify(overrides));
              qc.invalidateQueries({ queryKey: ["admin", "settings", "pricing"] });
              showToast({ type: "success", title: "Overrides saved", message: "Category fee overrides updated." });
            } catch {
              showToast({ type: "error", title: "Save failed", message: "Could not save overrides." });
            } finally {
              setSaving(false);
            }
          }}
          disabled={saving}
          className="mt-4 px-4 py-1.5 rounded-lg text-sm font-medium text-white"
          style={{ background: "var(--color-brand)" }}
        >
          {saving ? "Saving..." : "Save Overrides"}
        </button>
      </div>
    </div>
  );
}
