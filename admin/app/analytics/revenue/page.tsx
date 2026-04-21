"use client";

import PageHeader from "@/components/shared/PageHeader";

const MONTHS: Array<[string, number]> = [
  ["Jan", 120000], ["Feb", 132000], ["Mar", 145000], ["Apr", 168000], ["May", 177000], ["Jun", 191000],
];

export default function RevenuePage() {
  const max = Math.max(...MONTHS.map((m) => m[1]));
  return (
    <div>
      <PageHeader title="Revenue" description="Monthly performance trend (skeleton)" />
      <div className="surface p-5">
        <div className="space-y-3">
          {MONTHS.map(([m, v]) => (
            <div key={m} className="flex items-center gap-3">
              <span className="w-10 text-xs" style={{ color: "var(--text-tertiary)" }}>{m}</span>
              <div className="flex-1 h-3 rounded-full" style={{ background: "var(--bg-inset)" }}>
                <div className="h-full rounded-full" style={{ width: `${(Number(v) / max) * 100}%`, background: "var(--color-brand)" }} />
              </div>
              <span className="w-24 text-right text-sm font-medium" style={{ color: "var(--text-primary)" }}>${Number(v).toLocaleString()}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
