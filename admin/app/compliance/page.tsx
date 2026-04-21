"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import DataTable from "@/components/shared/DataTable";
import FiltersBar from "@/components/shared/FiltersBar";
import { mockComplianceAudit } from "@/lib/mockData";
import { complianceApi, usersApi } from "@/lib/api";
import { useToastStore } from "@/lib/toast";
import { FileDown, Trash2, Shield, Search, Download } from "lucide-react";

type AuditRow = {
  id: string;
  admin_user_id: string;
  admin_name?: string;
  action: string;
  target_type: string;
  target_id: string;
  old_value?: string;
  new_value?: string;
  ip_address: string;
  created_at: string;
};

function normalizeAudit(payload: unknown): AuditRow[] {
  const box = payload as { data?: unknown[] } | unknown[] | null | undefined;
  const rows = Array.isArray(box) ? box : Array.isArray((box as { data?: unknown[] })?.data) ? (box as { data?: unknown[] }).data : [];
  return (rows as Record<string, unknown>[]).map((item) => ({
    id: String(item.id ?? ""),
    admin_user_id: String(item.admin_user_id ?? item.user_id ?? ""),
    admin_name: item.admin_name ? String(item.admin_name) : undefined,
    action: String(item.action ?? ""),
    target_type: String(item.target_type ?? ""),
    target_id: String(item.target_id ?? ""),
    old_value: item.old_value ? String(item.old_value) : undefined,
    new_value: item.new_value ? String(item.new_value) : undefined,
    ip_address: String(item.ip_address ?? ""),
    created_at: String(item.created_at ?? new Date().toISOString()),
  })).filter((x) => x.id);
}

export default function CompliancePage() {
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [actionFilter, setActionFilter] = useState("");
  const [gdprUserId, setGdprUserId] = useState("");
  const [gdprAction, setGdprAction] = useState<"export" | "delete" | "">("");

  const { data: liveAudit, isLoading } = useQuery({
    queryKey: ["compliance", "audit"],
    queryFn: async () => {
      try {
        const res = await complianceApi.auditLogs();
        return normalizeAudit(res);
      } catch { return []; }
    },
    retry: 1,
  });

  const source: AuditRow[] = liveAudit?.length ? liveAudit : (mockComplianceAudit as unknown as AuditRow[]);

  const filtered = source.filter((a) => {
    if (actionFilter && !a.action.startsWith(actionFilter)) return false;
    return true;
  });

  const gdprExportMutation = useMutation({
    mutationFn: (userId: string) => complianceApi.gdprExport(userId),
    onSuccess: (data) => {
      showToast({ type: "success", title: "GDPR Export Complete", message: `Data ZIP generated for user ${gdprUserId}` });
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `gdpr-export-${gdprUserId}.json`;
      a.click();
      URL.revokeObjectURL(url);
      setGdprAction("");
    },
    onError: (error: { message?: string }) => {
      showToast({ type: "error", title: "Export failed", message: error?.message ?? "Could not export user data." });
    },
  });

  const gdprDeleteMutation = useMutation({
    mutationFn: (userId: string) => complianceApi.gdprDelete(userId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["compliance", "audit"] });
      showToast({ type: "success", title: "Right to Erasure Executed", message: `All personal data for user ${gdprUserId} has been deleted.` });
      setGdprAction("");
    },
    onError: (error: { message?: string }) => {
      showToast({ type: "error", title: "Deletion failed", message: error?.message ?? "Could not delete user data." });
    },
  });

  const handleGdprAction = () => {
    if (!gdprUserId.trim()) {
      showToast({ type: "error", title: "User ID required", message: "Enter a user ID to perform GDPR action." });
      return;
    }
    if (gdprAction === "export") gdprExportMutation.mutate(gdprUserId);
    if (gdprAction === "delete") gdprDeleteMutation.mutate(gdprUserId);
  };

  const handleExportCsv = () => {
    const header = "ID,Admin,Action,Target Type,Target ID,Old Value,New Value,IP,Created At";
    const rows = filtered.map((a) => `${a.id},${a.admin_name ?? a.admin_user_id},${a.action},${a.target_type},${a.target_id},${a.old_value ?? ""},${a.new_value ?? ""},${a.ip_address},${a.created_at}`);
    const csv = [header, ...rows].join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "audit-log-export.csv";
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div>
      <PageHeader title="GDPR & Compliance" description="Data export, right to erasure, and audit log viewer" />

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-4">
        <div className="surface p-5 rounded-lg lg:col-span-1">
          <h3 className="text-sm font-semibold mb-4 flex items-center gap-2" style={{ color: "var(--text-primary)" }}>
            <Shield className="w-4 h-4" />GDPR Actions
          </h3>
          <div className="space-y-3">
            <div>
              <label className="text-xs block mb-1" style={{ color: "var(--text-tertiary)" }}>User ID</label>
              <input
                type="text"
                value={gdprUserId}
                onChange={(e) => setGdprUserId(e.target.value)}
                placeholder="Enter user ID"
                className="w-full px-3 py-1.5 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }}
              />
            </div>
            <div className="flex gap-2">
              <button
                className={`flex-1 py-2 rounded-lg text-sm font-medium text-white flex items-center justify-center gap-1.5 ${gdprAction === "export" ? "ring-2 ring-blue-300" : ""}`}
                style={{ background: gdprAction === "export" ? "var(--color-brand)" : "#6b7280" }}
                onClick={() => setGdprAction(gdprAction === "export" ? "" : "export")}
              >
                <FileDown className="w-4 h-4" />Export ZIP
              </button>
              <button
                className={`flex-1 py-2 rounded-lg text-sm font-medium text-white flex items-center justify-center gap-1.5 ${gdprAction === "delete" ? "ring-2 ring-red-300" : ""}`}
                style={{ background: gdprAction === "delete" ? "var(--color-danger)" : "#6b7280" }}
                onClick={() => setGdprAction(gdprAction === "delete" ? "" : "delete")}
              >
                <Trash2 className="w-4 h-4" />Right to Erasure
              </button>
            </div>
            {gdprAction && (
              <button
                onClick={handleGdprAction}
                disabled={gdprExportMutation.isPending || gdprDeleteMutation.isPending}
                className="w-full py-2 rounded-lg text-sm font-medium text-white"
                style={{ background: gdprAction === "delete" ? "var(--color-danger)" : "var(--color-brand)" }}
              >
                {gdprAction === "export" ? (gdprExportMutation.isPending ? "Exporting..." : "Confirm Export") : (gdprDeleteMutation.isPending ? "Deleting..." : "Confirm Deletion")}
              </button>
            )}
            {gdprAction === "delete" && (
              <p className="text-xs" style={{ color: "var(--color-danger)" }}>⚠ This action is irreversible. All personal data will be permanently deleted.</p>
            )}
          </div>
        </div>

        <div className="surface p-5 rounded-lg lg:col-span-2">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold flex items-center gap-2" style={{ color: "var(--text-primary)" }}>
              <Search className="w-4 h-4" />Audit Log
            </h3>
            <button
              onClick={handleExportCsv}
              className="px-3 py-1.5 rounded-lg text-sm font-medium flex items-center gap-1.5"
              style={{ background: "var(--bg-surface)", border: "1px solid var(--border-default)", color: "var(--text-secondary)" }}
            >
              <Download className="w-3.5 h-3.5" />Export CSV
            </button>
          </div>

          <FiltersBar
            filters={[
              { key: "action", label: "Action", value: actionFilter, onChange: setActionFilter, options: [
                { label: "All", value: "" }, { label: "User Actions", value: "user." }, { label: "Listing Actions", value: "listing." },
                { label: "Settings", value: "settings." }, { label: "Feature Flags", value: "feature." }, { label: "Trust Flags", value: "trust_" },
              ]},
            ]}
          />

          <DataTable
            columns={[
              { key: "id", label: "ID", render: (a: AuditRow) => <span className="font-mono text-xs">{a.id}</span> },
              { key: "admin_name", label: "Actor", render: (a: AuditRow) => a.admin_name ?? a.admin_user_id },
              { key: "action", label: "Action" },
              { key: "target_type", label: "Target" },
              { key: "target_id", label: "Target ID" },
              { key: "old_value", label: "Old", render: (a: AuditRow) => a.old_value ? <span className="text-xs font-mono" style={{ color: "var(--color-danger)" }}>{a.old_value}</span> : "—" },
              { key: "new_value", label: "New", render: (a: AuditRow) => a.new_value ? <span className="text-xs font-mono" style={{ color: "var(--color-success)" }}>{a.new_value}</span> : "—" },
              { key: "ip_address", label: "IP", render: (a: AuditRow) => <span className="text-xs font-mono">{a.ip_address}</span> },
              { key: "created_at", label: "Time", render: (a: AuditRow) => <span className="text-xs">{new Date(a.created_at).toLocaleString()}</span> },
            ]}
            data={filtered}
            isLoading={isLoading}
            loadingMessage="Loading audit log..."
            rowKey={(a) => a.id}
            emptyMessage="No audit entries found."
          />
        </div>
      </div>
    </div>
  );
}
