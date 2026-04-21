"use client";

import { useState } from "react";
import { Users } from "lucide-react";
import type { FeatureFlag } from "@/lib/types";

const AVAILABLE_GROUPS = ["admin", "beta_testers", "premium", "sellers", "buyers", "all"];

interface FeatureFlagRowProps {
  flag: FeatureFlag;
  onUpdate: (key: string, data: { enabled?: boolean; rollout_pct?: number; allowed_groups?: string[] }) => Promise<void>;
}

export default function FeatureFlagRow({ flag, onUpdate }: FeatureFlagRowProps) {
  const [rollout, setRollout] = useState(flag.rollout_pct);
  const [saving, setSaving] = useState(false);
  const [showGroups, setShowGroups] = useState(false);

  const toggle = async () => {
    setSaving(true);
    try {
      await onUpdate(flag.key, { enabled: !flag.enabled });
    } finally {
      setSaving(false);
    }
  };

  const updateRollout = async () => {
    setSaving(true);
    try {
      await onUpdate(flag.key, { rollout_pct: rollout });
    } finally {
      setSaving(false);
    }
  };

  const toggleGroup = async (group: string) => {
    const current = flag.allowed_groups ?? [];
    const next = current.includes(group)
      ? current.filter((g) => g !== group)
      : [...current, group];
    setSaving(true);
    try {
      await onUpdate(flag.key, { allowed_groups: next });
    } finally {
      setSaving(false);
    }
  };

  const name = flag.key.replace("feature.", "").replace(/_/g, " ");

  return (
    <div className="py-3.5 px-4 border-b last:border-0" style={{ borderColor: "var(--border-subtle)" }}>
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium capitalize" style={{ color: "var(--text-primary)" }}>{name}</p>
          {flag.description && (
            <p className="text-xs mt-0.5" style={{ color: "var(--text-tertiary)" }}>{flag.description}</p>
          )}
        </div>

        <div className="flex items-center gap-4 flex-shrink-0">
          <span className="text-[11px] font-medium" style={{ color: flag.enabled ? "var(--color-success)" : "var(--text-tertiary)" }}>
            {flag.enabled ? "Enabled" : "Disabled"}
          </span>
          {flag.enabled && (
            <>
              <div className="flex items-center gap-2">
                <input
                  type="range"
                  min={0}
                  max={100}
                  value={rollout}
                  onChange={(e) => setRollout(parseInt(e.target.value))}
                  onMouseUp={updateRollout}
                  onTouchEnd={updateRollout}
                  className="w-24 accent-blue-600"
                />
                <span className="text-xs w-8 text-right font-mono" style={{ color: "var(--text-secondary)" }}>{rollout}%</span>
              </div>
              <button
                onClick={() => setShowGroups(!showGroups)}
                className="p-1 rounded-md transition-colors"
                title="Allowed groups"
                style={{ color: (flag.allowed_groups?.length ?? 0) > 0 ? "var(--color-brand)" : "var(--text-tertiary)" }}
              >
                <Users className="w-3.5 h-3.5" />
              </button>
            </>
          )}

          <button
            onClick={toggle}
            disabled={saving}
            title={flag.enabled ? "Disable feature" : "Enable feature"}
            aria-label={flag.enabled ? "Disable feature" : "Enable feature"}
            className={`relative w-11 h-6 rounded-full transition-colors ${
              flag.enabled ? "bg-blue-600" : "bg-slate-300"
            }`}
          >
            <span
              className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform ${
                flag.enabled ? "translate-x-5" : ""
              }`}
            />
          </button>
        </div>
      </div>

      {flag.enabled && showGroups && (
        <div className="mt-2 pt-2" style={{ borderTop: "1px solid var(--border-subtle)" }}>
          <p className="text-[10px] font-semibold uppercase tracking-wider mb-1.5" style={{ color: "var(--text-tertiary)" }}>
            Allowed Groups {(flag.allowed_groups?.length ?? 0) === 0 ? "(all users)" : ""}
          </p>
          <div className="flex flex-wrap gap-1.5">
            {AVAILABLE_GROUPS.map((g) => {
              const active = flag.allowed_groups?.includes(g);
              return (
                <button
                  key={g}
                  onClick={() => toggleGroup(g)}
                  disabled={saving}
                  className="px-2 py-0.5 rounded-full text-[11px] font-medium transition-colors"
                  style={{
                    background: active ? "var(--color-brand)" : "var(--bg-inset)",
                    color: active ? "#fff" : "var(--text-secondary)",
                  }}
                >
                  {g.replace(/_/g, " ")}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
