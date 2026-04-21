"use client";

import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import { mockAuditLogs } from "@/lib/mockData";

export default function AuditLogsPage() {
  return (
    <div>
      <PageHeader title="Audit Logs" description="Immutable decision and action history" />
      <div className="surface overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr style={{ borderBottom: "1px solid var(--border-default)" }}>
              {["ID", "Action", "Actor", "Target", "Details", "Time"].map((h) => (
                <th key={h} className="text-left px-4 py-3 text-xs font-semibold uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {mockAuditLogs.map((log) => (
              <tr key={log.id} style={{ borderBottom: "1px solid var(--border-default)" }}>
                <td className="px-4 py-3 font-mono text-xs" style={{ color: "var(--text-tertiary)" }}>{log.id}</td>
                <td className="px-4 py-3"><StatusBadge status={log.action} variant="info" /></td>
                <td className="px-4 py-3" style={{ color: "var(--text-primary)" }}>{log.actor}</td>
                <td className="px-4 py-3 font-mono text-xs" style={{ color: "var(--text-secondary)" }}>{log.target}</td>
                <td className="px-4 py-3" style={{ color: "var(--text-secondary)" }}>{log.details}</td>
                <td className="px-4 py-3 text-xs" style={{ color: "var(--text-tertiary)" }}>{new Date(log.timestamp).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
