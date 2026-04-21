"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import FiltersBar from "@/components/shared/FiltersBar";
import DataTable from "@/components/shared/DataTable";
import RightPanel from "@/components/shared/RightPanel";
import { mockKYC } from "@/lib/mockData";
import { kycApi } from "@/lib/api";
import { useToastStore } from "@/lib/toast";
import { UserCheck, UserX, Clock, CheckCircle } from "lucide-react";

type KYCRow = {
  id: string;
  user_id: string;
  user_name: string;
  user_email: string;
  kyc_status: string;
  phone_verified: boolean;
  id_document_url?: string;
  submitted_at: string;
  reviewed_by?: string;
  rejection_reason?: string;
};

function normalizeKYC(payload: unknown): KYCRow[] {
  const box = payload as { data?: unknown[] } | unknown[] | null | undefined;
  const rows = Array.isArray(box) ? box : Array.isArray((box as { data?: unknown[] })?.data) ? (box as { data?: unknown[] }).data : [];
  return (rows as Record<string, unknown>[]).map((item) => ({
    id: String(item.id ?? ""),
    user_id: String(item.user_id ?? ""),
    user_name: String(item.user_name ?? item.name ?? "Unknown"),
    user_email: String(item.user_email ?? item.email ?? ""),
    kyc_status: String(item.kyc_status ?? item.status ?? "pending"),
    phone_verified: Boolean(item.phone_verified),
    id_document_url: item.id_document_url ? String(item.id_document_url) : undefined,
    submitted_at: String(item.submitted_at ?? item.created_at ?? new Date().toISOString()),
    reviewed_by: item.reviewed_by ? String(item.reviewed_by) : undefined,
    rejection_reason: item.rejection_reason ? String(item.rejection_reason) : undefined,
  })).filter((x) => x.id);
}

export default function KYCQueuePage() {
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [statusFilter, setStatusFilter] = useState("");
  const [selected, setSelected] = useState<KYCRow | null>(null);
  const [rejectReason, setRejectReason] = useState("");

  const { data: liveKYC, isLoading } = useQuery({
    queryKey: ["admin", "kyc"],
    queryFn: async () => {
      try {
        const res = await kycApi.list();
        return normalizeKYC(res);
      } catch { return []; }
    },
    retry: 1,
  });

  const source: KYCRow[] = liveKYC?.length ? liveKYC : (mockKYC as unknown as KYCRow[]);

  const approveMutation = useMutation({
    mutationFn: (id: string) => kycApi.approve(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "kyc"] });
      showToast({ type: "success", title: "KYC Approved", message: "User identity verified." });
    },
    onError: (error: { message?: string }) => {
      showToast({ type: "error", title: "Approval failed", message: error?.message ?? "Could not approve KYC." });
    },
  });

  const rejectMutation = useMutation({
    mutationFn: (id: string) => kycApi.reject(id, rejectReason || "Does not meet requirements"),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "kyc"] });
      setRejectReason("");
      showToast({ type: "success", title: "KYC Rejected", message: "Rejection reason recorded." });
    },
    onError: (error: { message?: string }) => {
      showToast({ type: "error", title: "Rejection failed", message: error?.message ?? "Could not reject KYC." });
    },
  });

  const filtered = source.filter((k) => {
    if (statusFilter && k.kyc_status !== statusFilter) return false;
    return true;
  });

  const pendingCount = source.filter((k) => k.kyc_status === "pending").length;
  const completionRate = source.length > 0 ? ((source.filter((k) => k.kyc_status === "approved").length / source.length) * 100).toFixed(0) : "0";

  return (
    <div>
      <PageHeader title="KYC & Identity Verification" description="Review and verify user identity submissions" />

      <div className="grid grid-cols-3 gap-3 mb-4">
        <div className="surface p-3 rounded-lg">
          <div className="flex items-center gap-2 mb-1">
            <Clock className="w-4 h-4" style={{ color: "var(--color-warning)" }} />
            <span className="text-xs font-medium" style={{ color: "var(--text-tertiary)" }}>Pending Review</span>
          </div>
          <p className="text-lg font-bold" style={{ color: "var(--text-primary)" }}>{pendingCount}</p>
        </div>
        <div className="surface p-3 rounded-lg">
          <div className="flex items-center gap-2 mb-1">
            <CheckCircle className="w-4 h-4" style={{ color: "var(--color-success)" }} />
            <span className="text-xs font-medium" style={{ color: "var(--text-tertiary)" }}>Completion Rate</span>
          </div>
          <p className="text-lg font-bold" style={{ color: "var(--text-primary)" }}>{completionRate}%</p>
        </div>
        <div className="surface p-3 rounded-lg">
          <div className="flex items-center gap-2 mb-1">
            <UserX className="w-4 h-4" style={{ color: "var(--color-danger)" }} />
            <span className="text-xs font-medium" style={{ color: "var(--text-tertiary)" }}>Rejected</span>
          </div>
          <p className="text-lg font-bold" style={{ color: "var(--text-primary)" }}>{source.filter((k) => k.kyc_status === "rejected").length}</p>
        </div>
      </div>

      <FiltersBar
        filters={[
          { key: "status", label: "Status", value: statusFilter, onChange: setStatusFilter, options: [
            { label: "All", value: "" }, { label: "Pending", value: "pending" }, { label: "Approved", value: "approved" }, { label: "Rejected", value: "rejected" },
          ]},
        ]}
      />

      <DataTable
        columns={[
          { key: "id", label: "ID", render: (k: KYCRow) => <span className="font-mono text-xs">{k.id}</span> },
          { key: "user_name", label: "Name" },
          { key: "user_email", label: "Email" },
          { key: "phone_verified", label: "Phone", render: (k: KYCRow) => k.phone_verified ? <CheckCircle className="w-4 h-4" style={{ color: "var(--color-success)" }} /> : <UserX className="w-4 h-4" style={{ color: "var(--color-danger)" }} /> },
          { key: "kyc_status", label: "Status", render: (k: KYCRow) => <StatusBadge status={k.kyc_status} dot /> },
          { key: "submitted_at", label: "Submitted", render: (k: KYCRow) => new Date(k.submitted_at).toLocaleDateString() },
        ]}
        data={filtered}
        isLoading={isLoading}
        loadingMessage="Loading KYC queue..."
        rowKey={(k) => k.id}
        onRowClick={setSelected}
        emptyMessage="No KYC submissions found."
      />

      <RightPanel open={!!selected} onClose={() => setSelected(null)} title="KYC Review">
        {selected && (
          <div className="space-y-4">
            <div>
              <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>User</p>
              <p className="text-sm font-medium" style={{ color: "var(--text-primary)" }}>{selected.user_name}</p>
              <p className="text-xs" style={{ color: "var(--text-tertiary)" }}>{selected.user_email}</p>
            </div>
            <div>
              <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Phone Verified</p>
              {selected.phone_verified ? <CheckCircle className="w-4 h-4" style={{ color: "var(--color-success)" }} /> : <UserX className="w-4 h-4" style={{ color: "var(--color-danger)" }} />}
            </div>
            {selected.id_document_url && (
              <div>
                <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>ID Document</p>
                <a href={selected.id_document_url} target="_blank" rel="noopener noreferrer" className="text-sm underline" style={{ color: "var(--color-brand)" }}>View Document</a>
              </div>
            )}
            <div>
              <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Status</p>
              <StatusBadge status={selected.kyc_status} dot />
            </div>
            {selected.rejection_reason && (
              <div>
                <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Rejection Reason</p>
                <p className="text-sm" style={{ color: "var(--color-danger)" }}>{selected.rejection_reason}</p>
              </div>
            )}
            {selected.kyc_status === "pending" && (
              <div className="space-y-3 pt-4" style={{ borderTop: "1px solid var(--border-default)" }}>
                <textarea
                  value={rejectReason}
                  onChange={(e) => setRejectReason(e.target.value)}
                  placeholder="Rejection reason (required if rejecting)"
                  rows={2}
                  className="w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }}
                />
                <div className="flex gap-2">
                  <button
                    className="flex-1 py-2 rounded-lg text-sm font-medium text-white flex items-center justify-center gap-1.5"
                    style={{ background: "var(--color-success)" }}
                    disabled={approveMutation.isPending}
                    onClick={() => approveMutation.mutate(selected.id)}
                  >
                    <UserCheck className="w-4 h-4" />Approve
                  </button>
                  <button
                    className="flex-1 py-2 rounded-lg text-sm font-medium text-white flex items-center justify-center gap-1.5"
                    style={{ background: "var(--color-danger)" }}
                    disabled={rejectMutation.isPending || !rejectReason}
                    onClick={() => rejectMutation.mutate(selected.id)}
                  >
                    <UserX className="w-4 h-4" />Reject
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </RightPanel>
    </div>
  );
}
