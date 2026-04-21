"use client";

import { useState, useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import { settingsApi } from "@/lib/api";
import { useToastStore } from "@/lib/toast";
import { Truck, Eye, EyeOff, TestTube, Save } from "lucide-react";

type Carrier = { key: string; name: string; enabled: boolean; credentials: Record<string, string> };

const DEFAULT_CARRIERS: Carrier[] = [
  { key: "shipping.dhl", name: "DHL Express", enabled: true, credentials: { api_key: "••••••dhl1", account: "DHL-ACC-001" } },
  { key: "shipping.aramex", name: "Aramex", enabled: true, credentials: { api_key: "••••••arax", account: "ARAMEX-ACC-002" } },
  { key: "shipping.fedex", name: "FedEx", enabled: false, credentials: { api_key: "", account: "" } },
];

export default function ShippingSettingsPage() {
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [carriers, setCarriers] = useState<Carrier[]>(DEFAULT_CARRIERS);
  const [revealedKeys, setRevealedKeys] = useState<Record<string, boolean>>({});
  const [testOrigin, setTestOrigin] = useState("EG");
  const [testDest, setTestDest] = useState("US");
  const [testWeight, setTestWeight] = useState(1.5);
  const [saving, setSaving] = useState(false);

  const { data: liveSettings } = useQuery({
    queryKey: ["admin", "settings", "shipping"],
    queryFn: () => settingsApi.getByCategory("shipping"),
    retry: 1,
  });

  useEffect(() => {
    if (liveSettings && Array.isArray(liveSettings)) {
      const settings = liveSettings as Record<string, unknown>[];
      const carriersSetting = settings.find((s) => s.key === "shipping.carriers");
      if (carriersSetting) {
        try {
          const parsed = JSON.parse(String(carriersSetting.value));
          if (Array.isArray(parsed)) setCarriers(parsed);
        } catch { /* ignore */ }
      }
    }
  }, [liveSettings]);

  const toggleCarrier = (idx: number) => {
    const updated = [...carriers];
    updated[idx] = { ...updated[idx], enabled: !updated[idx].enabled };
    setCarriers(updated);
  };

  const toggleReveal = (carrierKey: string) => {
    setRevealedKeys((prev) => ({ ...prev, [carrierKey]: !prev[carrierKey] }));
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await settingsApi.update("shipping.carriers", JSON.stringify(carriers));
      qc.invalidateQueries({ queryKey: ["admin", "settings", "shipping"] });
      showToast({ type: "success", title: "Shipping settings saved", message: "Carrier configuration updated." });
    } catch (error: unknown) {
      showToast({ type: "error", title: "Save failed", message: (error as { message?: string })?.message ?? "Could not save shipping settings." });
    } finally {
      setSaving(false);
    }
  };

  const handleTestRate = () => {
    const enabled = carriers.filter((c) => c.enabled);
    if (enabled.length === 0) {
      showToast({ type: "error", title: "No carriers enabled", message: "Enable at least one carrier to test rates." });
      return;
    }
    showToast({ type: "success", title: "Rate test simulated", message: `${testOrigin} → ${testDest}, ${testWeight}kg via ${enabled.map((c) => c.name).join(", ")}` });
  };

  return (
    <div>
      <PageHeader title="Shipping Manager" description="Carrier on/off, credentials, and rate testing" />

      <div className="surface p-5 rounded-lg mb-4">
        <h3 className="text-sm font-semibold mb-4 flex items-center gap-2" style={{ color: "var(--text-primary)" }}>
          <Truck className="w-4 h-4" />Carriers
        </h3>
        <div className="space-y-3">
          {carriers.map((carrier, idx) => (
            <div key={carrier.key} className="p-4 rounded-lg" style={{ background: "var(--bg-inset)" }}>
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => toggleCarrier(idx)}
                    className={`relative w-11 h-6 rounded-full transition-colors ${carrier.enabled ? "bg-blue-600" : "bg-slate-300"}`}
                  >
                    <span className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform ${carrier.enabled ? "translate-x-5" : ""}`} />
                  </button>
                  <span className="text-sm font-medium" style={{ color: "var(--text-primary)" }}>{carrier.name}</span>
                  <span className={`text-[10px] font-bold uppercase px-1.5 py-0.5 rounded ${carrier.enabled ? "bg-green-100 text-green-700" : "bg-slate-100 text-slate-500"}`}>
                    {carrier.enabled ? "Active" : "Disabled"}
                  </span>
                </div>
              </div>
              {carrier.enabled && (
                <div className="grid grid-cols-2 gap-2">
                  {Object.entries(carrier.credentials).map(([credKey, credVal]) => (
                    <div key={credKey}>
                      <label className="text-xs block mb-0.5" style={{ color: "var(--text-tertiary)" }}>{credKey}</label>
                      <div className="relative">
                        <input
                          type={revealedKeys[carrier.key] ? "text" : "password"}
                          value={revealedKeys[carrier.key] ? credVal : (credVal.startsWith("••") ? credVal : `••••••${credVal.slice(-4)}`)}
                          readOnly
                          className="w-full px-2 py-1 border rounded text-xs font-mono"
                          style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-secondary)" }}
                        />
                        <button
                          onClick={() => toggleReveal(carrier.key)}
                          className="absolute right-1.5 top-1/2 -translate-y-1/2"
                        >
                          {revealedKeys[carrier.key] ? <EyeOff className="w-3 h-3 text-slate-400" /> : <Eye className="w-3 h-3 text-slate-400" />}
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
        <div className="mt-4">
          <button onClick={handleSave} disabled={saving} className="px-4 py-2 rounded-lg text-sm font-medium text-white flex items-center gap-1.5" style={{ background: "var(--color-brand)" }}>
            <Save className="w-4 h-4" />{saving ? "Saving..." : "Save Carriers"}
          </button>
        </div>
      </div>

      <div className="surface p-5 rounded-lg">
        <h3 className="text-sm font-semibold mb-4 flex items-center gap-2" style={{ color: "var(--text-primary)" }}>
          <TestTube className="w-4 h-4" />Rate Test Tool
        </h3>
        <div className="grid grid-cols-3 gap-3 mb-4">
          <div>
            <label className="text-xs block mb-1" style={{ color: "var(--text-tertiary)" }}>Origin Country</label>
            <input type="text" value={testOrigin} onChange={(e) => setTestOrigin(e.target.value)}
              className="w-full px-3 py-1.5 border rounded-lg text-sm" style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }} />
          </div>
          <div>
            <label className="text-xs block mb-1" style={{ color: "var(--text-tertiary)" }}>Destination Country</label>
            <input type="text" value={testDest} onChange={(e) => setTestDest(e.target.value)}
              className="w-full px-3 py-1.5 border rounded-lg text-sm" style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }} />
          </div>
          <div>
            <label className="text-xs block mb-1" style={{ color: "var(--text-tertiary)" }}>Weight (kg)</label>
            <input type="number" step="0.1" value={testWeight} onChange={(e) => setTestWeight(parseFloat(e.target.value) || 0)}
              className="w-full px-3 py-1.5 border rounded-lg text-sm" style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }} />
          </div>
        </div>
        <button onClick={handleTestRate} className="px-4 py-2 rounded-lg text-sm font-medium text-white flex items-center gap-1.5" style={{ background: "var(--color-brand)" }}>
          <TestTube className="w-4 h-4" />Test Rates
        </button>
      </div>

      {/* Delivery Time Estimates */}
      <div className="surface p-5 rounded-lg">
        <h3 className="text-sm font-semibold mb-1" style={{ color: "var(--text-primary)" }}>Delivery Time Estimates by Region</h3>
        <p className="text-xs mb-4" style={{ color: "var(--text-tertiary)" }}>Displayed to buyers at checkout. Editable per region.</p>
        <div className="space-y-2">
          <div className="grid grid-cols-3 gap-3 text-xs font-semibold uppercase tracking-wider pb-2" style={{ color: "var(--text-tertiary)", borderBottom: "1px solid var(--border-default)" }}>
            <span>Region</span><span>Min (days)</span><span>Max (days)</span>
          </div>
          {[
            { region: "Domestic", min: 2, max: 5 },
            { region: "Middle East", min: 5, max: 10 },
            { region: "Europe", min: 7, max: 14 },
            { region: "North America", min: 10, max: 18 },
            { region: "Asia Pacific", min: 10, max: 21 },
            { region: "Africa", min: 14, max: 28 },
          ].map((row) => (
            <div key={row.region} className="grid grid-cols-3 gap-3 items-center">
              <span className="text-sm" style={{ color: "var(--text-secondary)" }}>{row.region}</span>
              <input type="number" defaultValue={row.min} className="px-2 py-1.5 border rounded text-sm font-mono" style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }} />
              <input type="number" defaultValue={row.max} className="px-2 py-1.5 border rounded text-sm font-mono" style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }} />
            </div>
          ))}
        </div>
        <button className="mt-4 px-4 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}>Save Estimates</button>
      </div>
    </div>
  );
}
