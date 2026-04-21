"use client";

import { Search } from "lucide-react";

interface FilterOption {
  label: string;
  value: string;
}

interface Props {
  search?: string;
  onSearchChange?: (v: string) => void;
  searchPlaceholder?: string;
  filters?: {
    key: string;
    label: string;
    options: FilterOption[];
    value: string;
    onChange: (v: string) => void;
  }[];
  actions?: React.ReactNode;
}

export default function FiltersBar({ search, onSearchChange, searchPlaceholder, filters, actions }: Props) {
  return (
    <div
      className="flex items-center gap-3 flex-wrap"
      style={{ marginBottom: "16px" }}
    >
      {onSearchChange && (
        <div className="relative">
          <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2" style={{ color: "var(--text-tertiary)" }} />
          <input
            type="text"
            value={search ?? ""}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder={searchPlaceholder ?? "Search..."}
            className="pl-9 pr-4 py-1.5 rounded-lg text-sm focus:outline-none focus:ring-2 transition-all"
            style={{
              background: "var(--bg-surface)",
              border: "1px solid var(--border-default)",
              color: "var(--text-primary)",
              width: "240px",
            }}
          />
        </div>
      )}

      {filters?.map((f) => (
        <select
          key={f.key}
          value={f.value}
          onChange={(e) => f.onChange(e.target.value)}
          className="px-3 py-1.5 rounded-lg text-sm focus:outline-none focus:ring-2 transition-all cursor-pointer"
          style={{
            background: "var(--bg-surface)",
            border: "1px solid var(--border-default)",
            color: "var(--text-primary)",
          }}
        >
          <option value="">{f.label}</option>
          {f.options.map((o) => (
            <option key={o.value} value={o.value}>{o.label}</option>
          ))}
        </select>
      ))}

      {actions && <div className="ml-auto flex items-center gap-2">{actions}</div>}
    </div>
  );
}
