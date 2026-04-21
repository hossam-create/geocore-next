"use client";

import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import { mockRiskAlerts } from "@/lib/mockData";

export default function RiskAlertsPage() {
  return (
    <div>
      <PageHeader title="Risk Alerts" description="High-priority anomalies detected by Custodii" />
      <div className="space-y-3">
        {mockRiskAlerts.map((a) => (
          <div key={a.id} className="surface p-4">
            <div className="flex items-center justify-between mb-2">
              <p className="font-medium" style={{ color: "var(--text-primary)" }}>{a.message}</p>
              <StatusBadge status={a.severity} dot />
            </div>
            <div className="text-xs" style={{ color: "var(--text-tertiary)" }}>
              {a.id} · {a.type.replace(/_/g, " ")} · user: {a.user} · {new Date(a.createdAt).toLocaleString()}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
