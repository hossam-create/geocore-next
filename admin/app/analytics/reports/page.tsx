"use client";

import PageHeader from "@/components/shared/PageHeader";

const REPORTS = [
  { id: "R-001", name: "Marketplace Weekly", status: "ready" },
  { id: "R-002", name: "Fraud Trend", status: "processing" },
  { id: "R-003", name: "Escrow Exposure", status: "ready" },
];

export default function ReportsPage() {
  return (
    <div>
      <PageHeader title="Reports" description="Generated and scheduled analytics reports" />
      <div className="surface overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr style={{ borderBottom: "1px solid var(--border-default)" }}>
            {["ID", "Name", "Status", "Action"].map((h) => <th key={h} className="text-left px-4 py-3 text-xs uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>{h}</th>)}
          </tr></thead>
          <tbody>
            {REPORTS.map((r) => (
              <tr key={r.id} style={{ borderBottom: "1px solid var(--border-default)" }}>
                <td className="px-4 py-3 font-mono text-xs" style={{ color: "var(--text-tertiary)" }}>{r.id}</td>
                <td className="px-4 py-3" style={{ color: "var(--text-primary)" }}>{r.name}</td>
                <td className="px-4 py-3" style={{ color: "var(--text-secondary)" }}>{r.status}</td>
                <td className="px-4 py-3"><button className="text-xs" style={{ color: "var(--color-brand)" }}>Download</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
