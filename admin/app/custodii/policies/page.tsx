"use client";

import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import { Scale, Plus } from "lucide-react";

const POLICIES = [
  { id: "POL-001", name: "Auto-approve low-risk listings", condition: "risk_score < 20", action: "approve", enabled: true },
  { id: "POL-002", name: "Flag high-value escrow", condition: "amount > 5000", action: "require_manual_review", enabled: true },
  { id: "POL-003", name: "Block banned-country IPs", condition: "country IN blocked_list", action: "reject", enabled: false },
  { id: "POL-004", name: "Auto-extend ending auctions", condition: "bid_in_last_5min AND bids > 10", action: "extend_5min", enabled: true },
  { id: "POL-005", name: "Velocity check new accounts", condition: "account_age < 7d AND listings > 5", action: "flag_review", enabled: true },
];

export default function PoliciesPage() {
  return (
    <div>
      <PageHeader
        title="Policy Engine"
        description="Define rules that govern automated decisions"
        actions={
          <button className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}>
            <Plus className="w-4 h-4" /> New Policy
          </button>
        }
      />

      <div className="space-y-3">
        {POLICIES.map((p) => (
          <div key={p.id} className="surface p-4 flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="w-10 h-10 rounded-lg flex items-center justify-center" style={{ background: p.enabled ? "var(--color-brand-light)" : "var(--bg-inset)" }}>
                <Scale className="w-5 h-5" style={{ color: p.enabled ? "var(--color-brand)" : "var(--text-tertiary)" }} />
              </div>
              <div>
                <p className="text-sm font-medium" style={{ color: "var(--text-primary)" }}>{p.name}</p>
                <p className="text-xs font-mono mt-0.5" style={{ color: "var(--text-tertiary)" }}>{p.condition}</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <StatusBadge status={p.action.replace(/_/g, " ")} variant={p.action === "reject" ? "danger" : p.action === "approve" ? "success" : "info"} />
              <button
                className="relative w-10 h-5 rounded-full transition-colors"
                style={{ background: p.enabled ? "var(--color-brand)" : "var(--bg-inset)" }}
              >
                <span className="absolute top-0.5 w-4 h-4 rounded-full bg-white shadow-sm transition-transform" style={{ left: p.enabled ? "22px" : "2px" }} />
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
