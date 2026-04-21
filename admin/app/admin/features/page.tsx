"use client";

import { useMemo } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { featuresApi } from "@/lib/api";
import FeatureFlagRow from "@/components/features/FeatureFlagRow";
import PageHeader from "@/components/shared/PageHeader";
import { useToastStore } from "@/lib/toast";
import type { FeatureFlag } from "@/lib/types";

// Display titles for DB-provided categories. New categories added to the DB
// will appear automatically under a title-cased heading — no frontend deploy needed.
const CATEGORY_LABELS: Record<string, string> = {
  commerce: "Commerce",
  growth: "Growth",
  auctions: "Auctions",
  payments: "Payments",
  future: "Future / Experimental",
};

// Stable display order. Unlisted categories sort alphabetically after these.
const CATEGORY_ORDER = ["commerce", "growth", "auctions", "payments", "future"];

function titleCase(s: string): string {
  return s.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

export default function AdminFeaturesPage() {
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const { data: flags = [], isLoading } = useQuery<FeatureFlag[]>({
    queryKey: ["admin-features"],
    queryFn: featuresApi.getAll,
  });

  const handleUpdate = async (key: string, data: { enabled?: boolean; rollout_pct?: number; allowed_groups?: string[] }) => {
    try {
      await featuresApi.update(key, data);
      qc.invalidateQueries({ queryKey: ["admin-features"] });
      showToast({ type: "success", title: "Feature updated", message: `${key} has been updated.` });
    } catch (error) {
      const message = (error as { message?: string } | null)?.message ?? "Could not update feature flag.";
      showToast({ type: "error", title: "Update failed", message });
      throw error;
    }
  };

  // Dynamically group flags by their DB-provided category
  const { enabledFlags, groupedCategories } = useMemo(() => {
    const enabled = flags.filter((f) => f.enabled);
    const catMap: Record<string, FeatureFlag[]> = {};
    for (const f of flags) {
      const cat = f.category || "other";
      (catMap[cat] ??= []).push(f);
    }
    // Sort categories: known order first, then alphabetical
    const orderedCats = Object.keys(catMap).sort((a, b) => {
      const ai = CATEGORY_ORDER.indexOf(a);
      const bi = CATEGORY_ORDER.indexOf(b);
      if (ai !== -1 && bi !== -1) return ai - bi;
      if (ai !== -1) return -1;
      if (bi !== -1) return 1;
      return a.localeCompare(b);
    });
    return {
      enabledFlags: enabled,
      groupedCategories: orderedCats.map((cat) => ({
        key: cat,
        title: CATEGORY_LABELS[cat] || titleCase(cat),
        flags: catMap[cat],
      })),
    };
  }, [flags]);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Feature Flags"
        description="Toggle features on/off without code deploys. Adjust rollout % for gradual releases."
      />

      {isLoading ? (
        <div className="surface p-6 text-sm" style={{ color: "var(--text-tertiary)" }}>
          Loading feature flags...
        </div>
      ) : (
        <>
          {enabledFlags.length > 0 && (
            <div className="surface">
              <div className="px-4 py-3 border-b" style={{ borderColor: "var(--border-subtle)" }}>
                <h2 className="text-sm font-semibold" style={{ color: "var(--color-success)" }}>
                  Live Now ({enabledFlags.length})
                </h2>
              </div>
              {enabledFlags.map((f) => (
                <FeatureFlagRow key={f.key} flag={f} onUpdate={handleUpdate} />
              ))}
            </div>
          )}

          {groupedCategories.map(({ key, title, flags: catFlags }) => (
            <div key={key} className="surface">
              <div className="px-4 py-3 border-b" style={{ borderColor: "var(--border-subtle)" }}>
                <h2 className="text-sm font-semibold" style={{ color: "var(--text-secondary)" }}>
                  {title} ({catFlags.length})
                </h2>
              </div>
              {catFlags.map((f) => (
                <FeatureFlagRow key={f.key} flag={f} onUpdate={handleUpdate} />
              ))}
            </div>
          ))}
        </>
      )}
    </div>
  );
}
