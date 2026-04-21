"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import FiltersBar from "@/components/shared/FiltersBar";
import RightPanel from "@/components/shared/RightPanel";
import DataTable from "@/components/shared/DataTable";
import { mockTrustFlags, mockTrustStats } from "@/lib/mockData";
import { trustApi } from "@/lib/api";
import { useToastStore } from "@/lib/toast";
import { Shield, AlertTriangle, CheckCircle, DollarSign, Eye } from "lucide-react";
import Link from "next/link";

type FlagRow = {
  id: string;
  target_type: string;
  target_id: string;
  flag_type: string;
  severity: string;
  source: string;
  status: string;
  notes?: string;
  risk_score?: number;
  created_at: string;
};

function normalizeFlags(payload: unknown): FlagRow[] {
  const box = payload as { data?: unknown[] } | unknown[] | null | undefined;
  const rows = Array.isArray(box) ? box : Array.isArray((box as { data?: unknown[] })?.data) ? (box as { data?: unknown[] }).data : [];
  return (rows as Record<string, unknown>[]).map((item) => ({
    id: String(item.id ?? ""),
    target_type: String(item.target_type ?? "user"),
    target_id: String(item.target_id ?? ""),
    flag_type: String(item.flag_type ?? "unknown"),
    severity: String(item.severity ?? "medium"),
    source: String(item.source ?? "manual"),
    status: String(item.status ?? "open"),
    notes: item.notes ? String(item.notes) : undefined,
    risk_score: item.risk_score ? Number(item.risk_score) : undefined,
    created_at: String(item.created_at ?? new Date().toISOString()),
  })).filter((x) => x.id);
}

function severityColor(s: string) {
  switch (s) {
    case "critical": return { bg: "rgba(239,68,68,0.15)", text: "#ef4444" };
    case "high": return { bg: "rgba(245,158,11,0.15)", text: "#f59e0b" };
    case "medium": return { bg: "rgba(59,130,246,0.15)", text: "#3b82f6" };
    default: return { bg: "rgba(100,116,139,0.15)", text: "#94a3b8" };
  }
}

export default function TrustSafetyPage() {
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [severityFilter, setSeverityFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [selected, setSelected] = useState<FlagRow | null>(null);
  const [checked, setChecked] = useState<Set<string>>(new Set());

  const { data: liveFlags, isLoading } = useQuery({
    queryKey: ["trust", "flags"],
    queryFn: async () => {
      const res = await trustApi.listFlags();
      return normalizeFlags(res);
    },
    retry: 1,
  });

  const { data: liveStats } = useQuery({
    queryKey: ["trust", "stats"],
    queryFn: () => trustApi.getStats(),
    retry: 1,
  });

  const source: FlagRow[] = liveFlags?.length ? liveFlags : mockTrustFlags as unknown as FlagRow[];
  const stats = liveStats ?? mockTrustStats;

  const resolveMutation = useMutation({
    mutationFn: ({ id, status, notes }: { id: string; status: string; notes?: string }) =>
      trustApi.resolveFlag(id, { status, notes }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["trust", "flags"] });
      qc.invalidateQueries({ queryKey: ["trust", "stats"] });
      showToast({ type: "success", title: "Flag resolved", message: "The trust flag has been updated." });
    },
    onError: (error: { message?: string }) => {
      showToast({ type: "error", title: "Resolution failed", message: error?.message ?? "Could not resolve flag." });
    },
  });

  const bulkResolveMutation = useMutation({
    mutationFn: (status: string) => trustApi.bulkResolve(Array.from(checked), status),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["trust", "flags"] });
      qc.invalidateQueries({ queryKey: ["trust", "stats"] });
      setChecked(new Set());
      showToast({ type: "success", title: "Bulk action complete", message: `${checked.size} flags updated.` });
    },
    onError: () => showToast({ type: "error", title: "Bulk action failed" }),
  });

  const filtered = source.filter((f) => {
    if (severityFilter && f.severity !== severityFilter) return false;
    if (statusFilter && f.status !== statusFilter) return false;
    return true;
  });

  const statCards = [
    { label: "Open Flags", value: stats.open_flags, icon: AlertTriangle, color: "#f59e0b" },
    { label: "Critical", value: stats.critical_flags, icon: Shield, color: "#ef4444" },
    { label: "Auto-Resolved Today", value: stats.auto_resolved_today, icon: CheckCircle, color: "#22c55e" },
    { label: "Fraud Prevented", value: `$${stats.fraud_prevented_usd.toLocaleString()}`, icon: DollarSign, color: "#3b82f6" },
  ];

  return (
    <div>
      <PageHeader title="Trust & Safety" description="Fraud flags, risk alerts, and moderation queue" />

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
        {statCards.map((card) => (
          <div key={card.label} className="surface p-3 rounded-lg">
            <div className="flex items-center gap-2 mb-1">
              <card.icon className="w-4 h-4" style={{ color: card.color }} />
              <span className="text-xs font-medium" style={{ color: "var(--text-tertiary)" }}>{card.label}</span>
            </div>
            <p className="text-lg font-bold" style={{ color: "var(--text-primary)" }}>{card.value}</p>
          </div>
        ))}
      </div>

      <FiltersBar
        filters={[
          { key: "severity", label: "Severity", value: severityFilter, onChange: setSeverityFilter, options: [{ label: "All", value: "" }, { label: "Low", value: "low" }, { label: "Medium", value: "medium" }, { label: "High", value: "high" }, { label: "Critical", value: "critical" }] },
          { key: "status", label: "Status", value: statusFilter, onChange: setStatusFilter, options: [{ label: "All", value: "" }, { label: "Open", value: "open" }, { label: "Investigating", value: "investigating" }, { label: "Resolved", value: "resolved" }, { label: "False Positive", value: "false_positive" }] },
        ]}
      />

      {/* Bulk Actions */}
      {checked.size > 0 && (
        <div className="flex items-center gap-3 mb-3 p-3 rounded-lg" style={{ background: "var(--bg-inset)", border: "1px solid var(--border-default)" }}>
          <span className="text-xs font-medium" style={{ color: "var(--text-secondary)" }}>{checked.size} selected</span>
          <button
            onClick={() => bulkResolveMutation.mutate("resolved")}
            disabled={bulkResolveMutation.isPending}
            className="px-3 py-1 rounded-lg text-xs font-medium text-white"
            style={{ background: "var(--color-success)" }}
          >
            Bulk Resolve
          </button>
          <button
            onClick={() => bulkResolveMutation.mutate("investigating")}
            disabled={bulkResolveMutation.isPending}
            className="px-3 py-1 rounded-lg text-xs font-medium text-white"
            style={{ background: "var(--color-warning)" }}
          >
            Bulk Escalate
          </button>
          <button
            onClick={() => setChecked(new Set())}
            className="px-3 py-1 rounded-lg text-xs font-medium"
            style={{ color: "var(--text-tertiary)" }}
          >
            Clear
          </button>
        </div>
      )}

      <DataTable
        columns={[
          { key: "select", label: "", render: (f: FlagRow) => (
            <input type="checkbox" checked={checked.has(f.id)} onChange={() => {
              const next = new Set(checked);
              if (next.has(f.id)) next.delete(f.id); else next.add(f.id);
              setChecked(next);
            }} className="rounded" onClick={(e) => e.stopPropagation()} />
          )},
          { key: "id", label: "ID", render: (f: FlagRow) => <span className="font-mono text-xs">{f.id}</span> },
          { key: "flag_type", label: "Type" },
          { key: "severity", label: "Severity", render: (f: FlagRow) => {
            const c = severityColor(f.severity);
            return <span className="text-xs font-semibold px-2 py-0.5 rounded-full" style={{ background: c.bg, color: c.text }}>{f.severity}</span>;
          }},
          { key: "target_type", label: "Target" },
          { key: "source", label: "Source" },
          { key: "risk_score", label: "Risk", render: (f: FlagRow) => f.risk_score ? <span className="font-mono text-xs">{f.risk_score}</span> : "—" },
          { key: "status", label: "Status", render: (f: FlagRow) => <StatusBadge status={f.status} dot /> },
          {
            key: "actions", label: "", render: (f: FlagRow) => (
              <Link href={`/trust/${f.id}`} className="p-1.5 rounded-md hover:bg-slate-100" title="View details">
                <Eye className="w-4 h-4" style={{ color: "var(--color-brand)" }} />
              </Link>
            ),
          },
        ]}
        data={filtered}
        isLoading={isLoading}
        loadingMessage="Loading trust flags..."
        rowKey={(f) => f.id}
        onRowClick={setSelected}
        emptyMessage="No trust flags found."
      />

      <RightPanel open={!!selected} onClose={() => setSelected(null)} title="Flag Details">
        {selected && (
          <div className="space-y-4">
            <div>
              <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Flag Type</p>
              <p className="text-sm font-medium" style={{ color: "var(--text-primary)" }}>{selected.flag_type}</p>
            </div>
            <div>
              <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Target</p>
              <p className="text-sm" style={{ color: "var(--text-secondary)" }}>{selected.target_type} {selected.target_id}</p>
            </div>
            <div>
              <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Severity</p>
              <span className="text-xs font-semibold px-2 py-0.5 rounded-full" style={{ background: severityColor(selected.severity).bg, color: severityColor(selected.severity).text }}>{selected.severity}</span>
            </div>
            {selected.notes && (
              <div>
                <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Notes</p>
                <p className="text-sm" style={{ color: "var(--text-secondary)" }}>{selected.notes}</p>
              </div>
            )}
            <div className="flex gap-2 pt-4" style={{ borderTop: "1px solid var(--border-default)" }}>
              <button
                className="flex-1 py-2 rounded-lg text-sm font-medium text-white"
                style={{ background: "var(--color-success)" }}
                disabled={resolveMutation.isPending}
                onClick={() => resolveMutation.mutate({ id: selected.id, status: "resolved", notes: "Resolved by admin" })}
              >
                Resolve
              </button>
              <button
                className="flex-1 py-2 rounded-lg text-sm font-medium text-white"
                style={{ background: "var(--color-warning)" }}
                disabled={resolveMutation.isPending}
                onClick={() => resolveMutation.mutate({ id: selected.id, status: "false_positive", notes: "False positive" })}
              >
                False Positive
              </button>
            </div>
          </div>
        )}
      </RightPanel>
    </div>
  );
}
