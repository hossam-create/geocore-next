"use client";

import { useState, useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import { settingsApi } from "@/lib/api";
import { useToastStore } from "@/lib/toast";
import { Sliders, Plus, X, AlertTriangle, Eye } from "lucide-react";

type WeightSlider = { key: string; label: string; value: number };

// Mock listings for ranking preview
const PREVIEW_LISTINGS = [
  { title: "Vintage Rolex Submariner", featured: 0.9, recency: 0.3, reviews: 0.85, sales: 0.7, price: 0.4 },
  { title: "iPhone 15 Pro Max", featured: 0.2, recency: 0.95, reviews: 0.6, sales: 0.9, price: 0.8 },
  { title: "MacBook Pro M4", featured: 0.5, recency: 0.8, reviews: 0.75, sales: 0.6, price: 0.5 },
  { title: "Gold Necklace 18K", featured: 0.7, recency: 0.5, reviews: 0.9, sales: 0.4, price: 0.6 },
  { title: "Gaming PC RTX 5090", featured: 0.3, recency: 0.7, reviews: 0.5, sales: 0.8, price: 0.9 },
];

const DEFAULT_WEIGHTS: WeightSlider[] = [
  { key: "search.featured_boost_weight", label: "Featured Boost", value: 0.25 },
  { key: "search.recency_weight", label: "Recency", value: 0.25 },
  { key: "search.review_weight", label: "Review Score", value: 0.20 },
  { key: "search.sales_weight", label: "Sales Volume", value: 0.15 },
  { key: "search.price_competitiveness_weight", label: "Price Competitiveness", value: 0.15 },
];

export default function SearchSettingsPage() {
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [weights, setWeights] = useState<WeightSlider[]>(DEFAULT_WEIGHTS);
  const [bannedKeywords, setBannedKeywords] = useState<string[]>([]);
  const [newKeyword, setNewKeyword] = useState("");
  const [saving, setSaving] = useState(false);
  const [showPreview, setShowPreview] = useState(false);

  const { data: liveSettings } = useQuery({
    queryKey: ["admin", "settings", "search"],
    queryFn: () => settingsApi.getByCategory("search"),
    retry: 1,
  });

  useEffect(() => {
    if (liveSettings && Array.isArray(liveSettings)) {
      const loaded = DEFAULT_WEIGHTS.map((w) => {
        const found = (liveSettings as Record<string, unknown>[]).find((s) => s.key === w.key);
        return { ...w, value: found ? Number(found.value) : w.value };
      });
      setWeights(loaded);
      const kwSetting = (liveSettings as Record<string, unknown>[]).find((s) => s.key === "search.banned_keywords");
      if (kwSetting) {
        try {
          const parsed = JSON.parse(String(kwSetting.value));
          if (Array.isArray(parsed)) setBannedKeywords(parsed.map(String));
        } catch { /* ignore */ }
      }
    }
  }, [liveSettings]);

  const total = weights.reduce((sum, w) => sum + w.value, 0);
  const isValid = Math.abs(total - 1.0) < 0.01;

  const handleWeightChange = (idx: number, val: number) => {
    const updated = [...weights];
    updated[idx] = { ...updated[idx], value: val };
    setWeights(updated);
  };

  const handleSave = async () => {
    if (!isValid) {
      showToast({ type: "error", title: "Invalid weights", message: "Weights must sum to 1.0" });
      return;
    }
    setSaving(true);
    try {
      const settings: Record<string, unknown> = {};
      weights.forEach((w) => { settings[w.key] = w.value; });
      settings["search.banned_keywords"] = JSON.stringify(bannedKeywords);
      await settingsApi.bulkUpdate(settings);
      qc.invalidateQueries({ queryKey: ["admin", "settings", "search"] });
      showToast({ type: "success", title: "Search settings saved", message: "Ranking weights and banned keywords updated." });
    } catch (error: unknown) {
      showToast({ type: "error", title: "Save failed", message: (error as { message?: string })?.message ?? "Could not save settings." });
    } finally {
      setSaving(false);
    }
  };

  const addKeyword = () => {
    if (newKeyword.trim() && !bannedKeywords.includes(newKeyword.trim().toLowerCase())) {
      setBannedKeywords([...bannedKeywords, newKeyword.trim().toLowerCase()]);
      setNewKeyword("");
    }
  };

  const removeKeyword = (kw: string) => {
    setBannedKeywords(bannedKeywords.filter((k) => k !== kw));
  };

  return (
    <div>
      <PageHeader title="Search Ranking Control" description="Cassini-style ranking weights and banned keywords manager" />

      <div className="surface p-5 rounded-lg mb-4">
        <h3 className="text-sm font-semibold mb-4 flex items-center gap-2" style={{ color: "var(--text-primary)" }}>
          <Sliders className="w-4 h-4" />Ranking Weights
        </h3>
        <div className="space-y-4">
          {weights.map((w, idx) => (
            <div key={w.key} className="flex items-center gap-4">
              <span className="text-sm w-44 shrink-0" style={{ color: "var(--text-secondary)" }}>{w.label}</span>
              <input
                type="range"
                min={0}
                max={100}
                value={Math.round(w.value * 100)}
                onChange={(e) => handleWeightChange(idx, parseInt(e.target.value) / 100)}
                className="flex-1 accent-blue-600"
              />
              <span className="text-sm font-mono w-14 text-right" style={{ color: "var(--text-primary)" }}>
                {(w.value * 100).toFixed(0)}%
              </span>
            </div>
          ))}
        </div>
        <div className="mt-4 pt-3 flex items-center justify-between" style={{ borderTop: "1px solid var(--border-default)" }}>
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium" style={{ color: "var(--text-secondary)" }}>Total:</span>
            <span className="text-sm font-bold font-mono" style={{ color: isValid ? "var(--color-success)" : "var(--color-danger)" }}>
              {(total * 100).toFixed(0)}%
            </span>
            {!isValid && (
              <span className="flex items-center gap-1 text-xs" style={{ color: "var(--color-danger)" }}>
                <AlertTriangle className="w-3 h-3" />Must equal 100%
              </span>
            )}
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => setShowPreview(!showPreview)}
              disabled={!isValid}
              className="px-4 py-2 rounded-lg text-sm font-medium flex items-center gap-1.5"
              style={{ background: "var(--bg-surface)", border: "1px solid var(--border-default)", color: "var(--text-secondary)" }}
            >
              <Eye className="w-3.5 h-3.5" />{showPreview ? "Hide Preview" : "Preview"}
            </button>
            <button
              onClick={handleSave}
              disabled={saving || !isValid}
              className="px-4 py-2 rounded-lg text-sm font-medium text-white"
              style={{ background: isValid ? "var(--color-brand)" : "#94a3b8" }}
            >
              {saving ? "Saving..." : "Save Weights"}
            </button>
          </div>
        </div>

        {/* Ranking Preview */}
        {showPreview && isValid && (
          <div className="mt-4 pt-4" style={{ borderTop: "1px solid var(--border-default)" }}>
            <h4 className="text-xs font-semibold uppercase tracking-wider mb-3" style={{ color: "var(--text-tertiary)" }}>
              Ranking Preview — Top 5 listings reordered with current weights
            </h4>
            <div className="space-y-2">
              {PREVIEW_LISTINGS
                .map((l) => {
                  const score =
                    l.featured * weights[0].value +
                    l.recency * weights[1].value +
                    l.reviews * weights[2].value +
                    l.sales * weights[3].value +
                    l.price * weights[4].value;
                  return { ...l, score };
                })
                .sort((a, b) => b.score - a.score)
                .map((l, i) => (
                  <div key={l.title} className="flex items-center gap-3 px-3 py-2 rounded-lg" style={{ background: "var(--bg-inset)" }}>
                    <span className="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold" style={{ background: i === 0 ? "var(--color-brand)" : "var(--bg-surface)", color: i === 0 ? "#fff" : "var(--text-secondary)", border: i > 0 ? "1px solid var(--border-default)" : undefined }}>
                      {i + 1}
                    </span>
                    <span className="flex-1 text-sm font-medium" style={{ color: "var(--text-primary)" }}>{l.title}</span>
                    <span className="text-xs font-mono" style={{ color: "var(--text-tertiary)" }}>
                      {(l.score * 100).toFixed(1)}
                    </span>
                  </div>
                ))}
            </div>
          </div>
        )}
      </div>

      <div className="surface p-5 rounded-lg">
        <h3 className="text-sm font-semibold mb-4" style={{ color: "var(--text-primary)" }}>Banned Keywords</h3>
        <p className="text-xs mb-3" style={{ color: "var(--text-tertiary)" }}>
          Listings containing these keywords will be auto-rejected by the moderation engine.
        </p>
        <div className="flex gap-2 mb-3">
          <input
            type="text"
            value={newKeyword}
            onChange={(e) => setNewKeyword(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && addKeyword()}
            placeholder="Add keyword..."
            className="flex-1 px-3 py-1.5 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }}
          />
          <button
            onClick={addKeyword}
            className="px-3 py-1.5 rounded-lg text-sm font-medium text-white"
            style={{ background: "var(--color-brand)" }}
          >
            <Plus className="w-4 h-4" />
          </button>
        </div>
        <div className="flex flex-wrap gap-2">
          {bannedKeywords.map((kw) => (
            <span key={kw} className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-medium" style={{ background: "rgba(239,68,68,0.1)", color: "var(--color-danger)" }}>
              {kw}
              <button onClick={() => removeKeyword(kw)} className="hover:opacity-70"><X className="w-3 h-3" /></button>
            </span>
          ))}
          {bannedKeywords.length === 0 && (
            <span className="text-xs" style={{ color: "var(--text-tertiary)" }}>No banned keywords configured</span>
          )}
        </div>
      </div>
    </div>
  );
}
