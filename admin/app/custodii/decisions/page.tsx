"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import FiltersBar from "@/components/shared/FiltersBar";
import RightPanel from "@/components/shared/RightPanel";
import DataTable from "@/components/shared/DataTable";
import { mockDecisions } from "@/lib/mockData";
import { kycApi } from "@/lib/api";
import { useToastStore } from "@/lib/toast";
import { Shield, CheckCircle, XCircle } from "lucide-react";

type DecisionRow = {
  id: string;
  asset: string;
  action: string;
  riskScore: number;
  status: string;
  source: string;
};

function normalizeKycAsDecisions(payload: unknown): DecisionRow[] {
  const rows = Array.isArray(payload)
    ? payload
    : Array.isArray((payload as { data?: unknown[] } | null | undefined)?.data)
    ? ((payload as { data?: unknown[] }).data as unknown[])
    : [];

  return rows
    .map((r) => {
      const item = r as Record<string, unknown>;
      const status = String(item.status ?? "pending");
      const riskScore = status === "rejected" ? 85 : status === "approved" ? 15 : 55;
      return {
        id: String(item.id ?? ""),
        asset: `KYC ${String(item.user_id ?? item.id ?? "Unknown")}`,
        action: status === "pending" ? "verify_identity" : "review_completed",
        riskScore,
        status,
        source: "kyc_admin",
      };
    })
    .filter((x) => x.id);
}

export default function DecisionQueuePage() {
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [statusFilter, setStatusFilter] = useState("");
  const [selected, setSelected] = useState<DecisionRow | null>(null);

  const { data: liveDecisions, isLoading } = useQuery({
    queryKey: ["custodii", "decisions"],
    queryFn: async () => {
      const res = await kycApi.list();
      return normalizeKycAsDecisions(res);
    },
    retry: 1,
  });

  const source: DecisionRow[] = liveDecisions?.length
    ? liveDecisions
    : mockDecisions.map((d) => ({
        id: d.id,
        asset: d.asset,
        action: d.action,
        riskScore: d.riskScore,
        status: d.status,
        source: d.source,
      }));

  const approveMutation = useMutation({
    mutationFn: (id: string) => kycApi.approve(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["custodii", "decisions"] });
      showToast({ type: "success", title: "Decision approved", message: "Action has been authorized by Custodii." });
    },
    onError: (error: { message?: string }) => {
      showToast({ type: "error", title: "Approval failed", message: error?.message ?? "Could not approve decision." });
    },
  });

  const rejectMutation = useMutation({
    mutationFn: (id: string) => kycApi.reject(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["custodii", "decisions"] });
      showToast({ type: "success", title: "Decision rejected", message: "Action has been denied by Custodii." });
    },
    onError: (error: { message?: string }) => {
      showToast({ type: "error", title: "Rejection failed", message: error?.message ?? "Could not reject decision." });
    },
  });

  const mutationBusy = approveMutation.isPending || rejectMutation.isPending;

  const filtered = source.filter((d) => {
    if (statusFilter && d.status !== statusFilter) return false;
    return true;
  });

  return (
    <div>
      <PageHeader title="Decision Queue" description="Custodii — all critical actions require approval here" />

      <FiltersBar
        filters={[{
          key: "status", label: "All Status", value: statusFilter, onChange: setStatusFilter,
          options: [
            { label: "Pending", value: "pending" },
            { label: "Approved", value: "approved" },
            { label: "Rejected", value: "rejected" },
          ],
        }]}
      />

      <DataTable
        columns={[
          { key: "id", label: "ID", render: (d: DecisionRow) => <span className="font-mono text-xs">{d.id}</span> },
          { key: "asset", label: "Asset" },
          { key: "action", label: "Action", render: (d: DecisionRow) => d.action.replace(/_/g, " ") },
          {
            key: "riskScore",
            label: "Risk Score",
            render: (d: DecisionRow) => (
              <div className="flex items-center gap-2">
                <div className="w-16 h-1.5 rounded-full overflow-hidden" style={{ background: "var(--bg-inset)" }}>
                  <div className="h-full rounded-full" style={{ width: `${d.riskScore}%`, background: d.riskScore > 70 ? "var(--color-danger)" : d.riskScore > 40 ? "var(--color-warning)" : "var(--color-success)" }} />
                </div>
                <span className="text-xs font-medium" style={{ color: d.riskScore > 70 ? "var(--color-danger)" : d.riskScore > 40 ? "var(--color-warning)" : "var(--color-success)" }}>{d.riskScore}%</span>
              </div>
            ),
          },
          { key: "source", label: "Source" },
          { key: "status", label: "Status", render: (d: DecisionRow) => <StatusBadge status={d.status} dot /> },
          {
            key: "actions",
            label: "",
            render: (d: DecisionRow) =>
              d.status === "pending" ? (
                <div className="flex items-center gap-1">
                  <button
                    className="p-1.5 rounded-md"
                    style={{ color: "var(--color-success)" }}
                    title="Approve"
                    disabled={mutationBusy}
                    onClick={(e) => {
                      e.stopPropagation();
                      approveMutation.mutate(d.id);
                    }}
                  >
                    <CheckCircle className="w-4 h-4" />
                  </button>
                  <button
                    className="p-1.5 rounded-md"
                    style={{ color: "var(--color-danger)" }}
                    title="Reject"
                    disabled={mutationBusy}
                    onClick={(e) => {
                      e.stopPropagation();
                      rejectMutation.mutate(d.id);
                    }}
                  >
                    <XCircle className="w-4 h-4" />
                  </button>
                </div>
              ) : null,
          },
        ]}
        data={filtered}
        isLoading={isLoading}
        loadingMessage="Loading decision queue..."
        emptyMessage="No decisions match your current filter."
        rowKey={(d) => d.id}
        onRowClick={setSelected}
      />

      <RightPanel open={!!selected} onClose={() => setSelected(null)} title="Decision Details">
        {selected && (
          <div className="space-y-4">
            <div><p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Asset</p><p className="text-sm font-medium" style={{ color: "var(--text-primary)" }}>{selected.asset}</p></div>
            <div><p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Action Requested</p><p className="text-sm" style={{ color: "var(--text-secondary)" }}>{selected.action.replace(/_/g, " ")}</p></div>

            <div>
              <p className="text-xs font-medium uppercase tracking-wider mb-2" style={{ color: "var(--text-tertiary)" }}>Risk Score</p>
              <div className="flex items-center gap-3">
                <div className="flex-1 h-3 rounded-full overflow-hidden" style={{ background: "var(--bg-inset)" }}>
                  <div className="h-full rounded-full transition-all" style={{ width: `${selected.riskScore}%`, background: selected.riskScore > 70 ? "var(--color-danger)" : selected.riskScore > 40 ? "var(--color-warning)" : "var(--color-success)" }} />
                </div>
                <span className="text-lg font-bold" style={{ color: selected.riskScore > 70 ? "var(--color-danger)" : selected.riskScore > 40 ? "var(--color-warning)" : "var(--color-success)" }}>{selected.riskScore}%</span>
              </div>
            </div>

            <div><p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Source</p><p className="text-sm" style={{ color: "var(--text-secondary)" }}>{selected.source}</p></div>
            <div><p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Status</p><StatusBadge status={selected.status} dot /></div>

            {selected.status === "pending" && (
              <div className="flex gap-2 pt-4" style={{ borderTop: "1px solid var(--border-default)" }}>
                <button
                  className="flex-1 py-2.5 rounded-lg text-sm font-medium text-white flex items-center justify-center gap-2"
                  style={{ background: "var(--color-success)" }}
                  onClick={() => approveMutation.mutate(selected.id)}
                  disabled={mutationBusy}
                >
                  <CheckCircle className="w-4 h-4" /> {approveMutation.isPending ? "Approving..." : "Approve"}
                </button>
                <button
                  className="flex-1 py-2.5 rounded-lg text-sm font-medium text-white flex items-center justify-center gap-2"
                  style={{ background: "var(--color-danger)" }}
                  onClick={() => rejectMutation.mutate(selected.id)}
                  disabled={mutationBusy}
                >
                  <XCircle className="w-4 h-4" /> {rejectMutation.isPending ? "Rejecting..." : "Reject"}
                </button>
              </div>
            )}
          </div>
        )}
      </RightPanel>
    </div>
  );
}
