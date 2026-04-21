"use client";

import { useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import type { AdminSetting } from "@/lib/types";

interface SettingFieldProps {
  setting: AdminSetting;
  onUpdate: (key: string, value: unknown) => void;
}

function maskSecret(val: string): string {
  if (!val || val.length < 4) return "••••";
  return `••••••${val.slice(-4)}`;
}

export default function SettingField({ setting, onUpdate }: SettingFieldProps) {
  const parsed = parseValue(setting.value, setting.type);
  const isSecret = setting.is_secret || setting.type === "secret";
  const [localValue, setLocalValue] = useState(isSecret ? "" : String(parsed ?? ""));
  const [saving, setSaving] = useState(false);
  const [revealed, setRevealed] = useState(false);
  const [edited, setEdited] = useState(false);

  const save = async (val: unknown) => {
    setSaving(true);
    try {
      await onUpdate(setting.key, val);
    } finally {
      setSaving(false);
    }
  };

  const options: { value: string; label: string }[] = setting.options
    ? (typeof setting.options === "string" ? JSON.parse(setting.options) : setting.options)
    : [];

  const maskedDisplay = maskSecret(String(parsed ?? ""));

  return (
    <div className="flex items-start justify-between gap-4 py-3 border-b border-slate-100 last:border-0">
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium text-slate-800">{setting.label}</p>
        {setting.description && (
          <p className="text-xs text-slate-400 mt-0.5">{setting.description}</p>
        )}
        <p className="text-[10px] text-slate-300 font-mono mt-0.5">{setting.key}</p>
      </div>

      <div className="flex-shrink-0 w-64">
        {setting.type === "bool" && (
          <button
            onClick={() => save(parsed === true ? false : true)}
            disabled={saving}
            title={parsed === true ? "Disable setting" : "Enable setting"}
            aria-label={parsed === true ? "Disable setting" : "Enable setting"}
            className={`relative w-11 h-6 rounded-full transition-colors ${
              parsed === true ? "bg-blue-600" : "bg-slate-300"
            }`}
          >
            <span
              className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform ${
                parsed === true ? "translate-x-5" : ""
              }`}
            />
          </button>
        )}

        {setting.type === "string" && (
          <div className="flex gap-1.5">
            <input
              type="text"
              value={localValue}
              onChange={(e) => setLocalValue(e.target.value)}
              className="w-full px-3 py-1.5 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              onClick={() => save(JSON.stringify(localValue))}
              disabled={saving}
              className="px-3 py-1.5 bg-blue-600 text-white text-xs rounded-lg hover:bg-blue-700 disabled:opacity-50 font-medium"
            >
              {saving ? "Saving..." : "Save"}
            </button>
          </div>
        )}

        {setting.type === "number" && (
          <div className="flex gap-1.5">
            <input
              type="number"
              value={localValue}
              onChange={(e) => setLocalValue(e.target.value)}
              className="w-full px-3 py-1.5 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              onClick={() => save(parseFloat(localValue) || 0)}
              disabled={saving}
              className="px-3 py-1.5 bg-blue-600 text-white text-xs rounded-lg hover:bg-blue-700 disabled:opacity-50 font-medium"
            >
              {saving ? "Saving..." : "Save"}
            </button>
          </div>
        )}

        {setting.type === "select" && (
          <select
            value={String(parsed ?? "")}
            onChange={(e) => save(JSON.stringify(e.target.value))}
            className="w-full px-3 py-1.5 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {options.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        )}

        {setting.type === "secret" && (
          <div className="flex gap-1.5">
            <div className="relative flex-1">
              {!edited ? (
                <div className="flex items-center gap-2">
                  <span className="text-sm font-mono px-3 py-1.5" style={{ color: "var(--text-tertiary)" }}>{maskedDisplay}</span>
                  <button
                    type="button"
                    onClick={() => setEdited(true)}
                    className="text-xs font-medium px-2 py-1 rounded-md"
                    style={{ color: "var(--color-brand)", background: "var(--bg-inset)" }}
                  >
                    Change
                  </button>
                </div>
              ) : (
                <>
                  <input
                    type={revealed ? "text" : "password"}
                    value={localValue}
                    onChange={(e) => setLocalValue(e.target.value)}
                    placeholder="Enter new value..."
                    className="w-full px-3 py-1.5 pr-8 border border-slate-200 rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                  <button
                    type="button"
                    onClick={() => setRevealed(!revealed)}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600"
                    title={revealed ? "Hide" : "Reveal"}
                  >
                    {revealed ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                  </button>
                </>
              )}
            </div>
            {edited && (
              <button
                onClick={() => save(JSON.stringify(localValue))}
                disabled={saving || !localValue}
                className="px-3 py-1.5 bg-blue-600 text-white text-xs rounded-lg hover:bg-blue-700 disabled:opacity-50 font-medium"
              >
                {saving ? "Saving..." : "Save"}
              </button>
            )}
          </div>
        )}

        {setting.type === "json" && (
          <div className="flex gap-1.5">
            <textarea
              value={localValue}
              onChange={(e) => setLocalValue(e.target.value)}
              rows={3}
              className="w-full px-3 py-1.5 border border-slate-200 rounded-lg text-xs font-mono focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
            />
            <button
              onClick={() => save(localValue)}
              disabled={saving}
              className="px-3 py-1.5 bg-blue-600 text-white text-xs rounded-lg hover:bg-blue-700 disabled:opacity-50 font-medium self-end"
            >
              {saving ? "Saving..." : "Save"}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

function parseValue(raw: string, type: string): unknown {
  try {
    if (type === "bool") return raw === "true";
    if (type === "number") return parseFloat(raw) || 0;
    return JSON.parse(raw);
  } catch {
    return raw;
  }
}
