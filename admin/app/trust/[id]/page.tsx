"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams } from "next/navigation";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import { trustApi, usersApi } from "@/lib/api";
import { mockTrustFlags } from "@/lib/mockData";
import { useToastStore } from "@/lib/toast";
import { Shield, AlertTriangle, User, FileText, Ban, ShieldCheck } from "lucide-react";

type FlagDetail = {
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
  resolved_at?: string;
  resolved_by?: string;
};

export default function TrustFlagDetailPage() {
  const params = useParams();
  const id = params.id as string;
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [adminNotes, setAdminNotes] = useState("");

  const { data: liveFlag, isLoading } = useQuery({
    queryKey: ["trust", "flags", id],
    queryFn: async () => {
      try {
        const res = await trustApi.getFlag(id);
        return res as FlagDetail;
      } catch { return null; }
    },
    retry: 1,
  });

  const flag: FlagDetail | null = liveFlag ?? (mockTrustFlags as unknown as FlagDetail[]).find((f) => f.id === id) ?? null;

  const resolveMutation = useMutation({
    mutationFn: (data: { status: string; notes?: string }) => trustApi.resolveFlag(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["trust", "flags", id] });
      showToast({ type: "success", title: "Flag resolved", message: "Action recorded in audit log." });
    },
    onError: (error: { message?: string }) => {
      showToast({ type: "error", title: "Action failed", message: error?.message ?? "Could not resolve flag." });
    },
  });

  const banMutation = useMutation({
    mutationFn: (userId: string) => usersApi.ban(userId, "Banned via Trust & Safety flag"),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["trust", "flags", id] });
      showToast({ type: "success", title: "User banned", message: "All JWTs invalidated, listings hidden." });
    },
    onError: (error: { message?: string }) => {
      showToast({ type: "error", title: "Ban failed", message: error?.message ?? "Could not ban user." });
    },
  });

  if (isLoading) return <div className="p-6 text-sm" style={{ color: "var(--text-tertiary)" }}>Loading flag details...</div>;
  if (!flag) return <div className="p-6 text-sm" style={{ color: "var(--text-tertiary)" }}>Flag not found.</div>;

  const isUser = flag.target_type === "user";

  return (
    <div>
      <PageHeader title={`Flag ${flag.id}`} description={`${flag.flag_type} · ${flag.severity} severity`} />

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div className="lg:col-span-2 space-y-4">
          <div className="surface p-4 rounded-lg">
            <h3 className="text-sm font-semibold mb-3" style={{ color: "var(--text-primary)" }}>What Triggered This Flag</h3>
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span style={{ color: "var(--text-tertiary)" }}>Type</span>
                <span className="font-medium" style={{ color: "var(--text-primary)" }}>{flag.flag_type}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span style={{ color: "var(--text-tertiary)" }}>Source</span>
                <span style={{ color: "var(--text-secondary)" }}>{flag.source}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span style={{ color: "var(--text-tertiary)" }}>Risk Score</span>
                <span className="font-mono font-medium" style={{ color: flag.risk_score && flag.risk_score > 70 ? "var(--color-danger)" : "var(--text-primary)" }}>{flag.risk_score ?? "—"}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span style={{ color: "var(--text-tertiary)" }}>Status</span>
                <StatusBadge status={flag.status} dot />
              </div>
              <div className="flex justify-between text-sm">
                <span style={{ color: "var(--text-tertiary)" }}>Created</span>
                <span style={{ color: "var(--text-secondary)" }}>{new Date(flag.created_at).toLocaleString()}</span>
              </div>
              {flag.resolved_at && (
                <div className="flex justify-between text-sm">
                  <span style={{ color: "var(--text-tertiary)" }}>Resolved</span>
                  <span style={{ color: "var(--text-secondary)" }}>{new Date(flag.resolved_at).toLocaleString()}</span>
                </div>
              )}
              {flag.notes && (
                <div className="pt-2 mt-2" style={{ borderTop: "1px solid var(--border-default)" }}>
                  <p className="text-xs font-medium mb-1" style={{ color: "var(--text-tertiary)" }}>Notes</p>
                  <p className="text-sm" style={{ color: "var(--text-secondary)" }}>{flag.notes}</p>
                </div>
              )}
            </div>
          </div>

          {flag.status !== "resolved" && flag.status !== "false_positive" && (
            <div className="surface p-4 rounded-lg">
              <h3 className="text-sm font-semibold mb-3" style={{ color: "var(--text-primary)" }}>Take Action</h3>
              <textarea
                value={adminNotes}
                onChange={(e) => setAdminNotes(e.target.value)}
                placeholder="Internal admin notes (never shown to user)"
                rows={3}
                className="w-full px-3 py-2 border rounded-lg text-sm mb-3 focus:outline-none focus:ring-2 focus:ring-blue-500"
                style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }}
              />
              <div className="flex flex-wrap gap-2">
                <button
                  className="flex-1 py-2 rounded-lg text-sm font-medium text-white"
                  style={{ background: "var(--color-warning)" }}
                  disabled={resolveMutation.isPending}
                  onClick={() => resolveMutation.mutate({ status: "resolved", notes: adminNotes || "Resolved by admin" })}
                >
                  <div className="flex items-center justify-center gap-1.5"><Shield className="w-4 h-4" /> Warn & Resolve</div>
                </button>
                {isUser && (
                  <button
                    className="flex-1 py-2 rounded-lg text-sm font-medium text-white"
                    style={{ background: "#f97316" }}
                    disabled={banMutation.isPending}
                    onClick={() => {
                      if (!confirm("Suspend this user for 7 days?")) return;
                      banMutation.mutate(flag.target_id);
                    }}
                  >
                    <div className="flex items-center justify-center gap-1.5"><Ban className="w-4 h-4" /> Suspend 7 Days</div>
                  </button>
                )}
                {isUser && (
                  <button
                    className="flex-1 py-2 rounded-lg text-sm font-medium text-white"
                    style={{ background: "var(--color-danger)" }}
                    disabled={banMutation.isPending}
                    onClick={() => {
                      if (!confirm("PERMANENT BAN — this will hide all listings and hold all payouts. Continue?")) return;
                      banMutation.mutate(flag.target_id);
                    }}
                  >
                    <div className="flex items-center justify-center gap-1.5"><Ban className="w-4 h-4" /> Permanent Ban</div>
                  </button>
                )}
                <button
                  className="flex-1 py-2 rounded-lg text-sm font-medium text-white"
                  style={{ background: "#6b7280" }}
                  disabled={resolveMutation.isPending}
                  onClick={() => resolveMutation.mutate({ status: "false_positive", notes: adminNotes || "False positive" })}
                >
                  <div className="flex items-center justify-center gap-1.5"><ShieldCheck className="w-4 h-4" /> False Positive</div>
                </button>
              </div>
            </div>
          )}
        </div>

        <div className="space-y-4">
          <div className="surface p-4 rounded-lg">
            <h3 className="text-sm font-semibold mb-3" style={{ color: "var(--text-primary)" }}>
              {isUser ? "User Profile" : "Listing Info"}
            </h3>
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                {isUser ? <User className="w-4 h-4" style={{ color: "var(--text-tertiary)" }} /> : <FileText className="w-4 h-4" style={{ color: "var(--text-tertiary)" }} />}
                <span className="text-sm font-mono" style={{ color: "var(--text-primary)" }}>{flag.target_id}</span>
              </div>
              <div className="text-xs" style={{ color: "var(--text-tertiary)" }}>
                {isUser ? "Click user ID to view full profile in Users page" : "Click listing ID to view in Listings page"}
              </div>
            </div>
          </div>

          <div className="surface p-4 rounded-lg">
            <h3 className="text-sm font-semibold mb-3" style={{ color: "var(--text-primary)" }}>Same Target History</h3>
            <FlagHistory targetId={flag.target_id} currentFlagId={flag.id} />
          </div>
        </div>
      </div>
    </div>
  );
}

function FlagHistory({ targetId, currentFlagId }: { targetId: string; currentFlagId: string }) {
  const { data: flags } = useQuery({
    queryKey: ["trust", "flags", "history", targetId],
    queryFn: async () => {
      try {
        const res = await trustApi.listFlags();
        const box = res as { data?: unknown[] } | unknown[] | null | undefined;
        const rows = Array.isArray(box) ? box : Array.isArray((box as { data?: unknown[] })?.data) ? (box as { data?: unknown[] }).data : [];
        return (rows as Record<string, unknown>[])
          .filter((f) => String(f.target_id) === targetId && String(f.id) !== currentFlagId)
          .map((f) => ({ id: String(f.id), flag_type: String(f.flag_type ?? ""), severity: String(f.severity ?? ""), status: String(f.status ?? ""), created_at: String(f.created_at ?? "") }));
      } catch { return []; }
    },
    retry: 1,
  });

  if (!flags || flags.length === 0) {
    return <p className="text-xs" style={{ color: "var(--text-tertiary)" }}>No previous flags for this target.</p>;
  }

  return (
    <div className="space-y-2">
      {flags.map((f) => (
        <div key={f.id} className="flex items-center justify-between text-xs py-1.5 px-2 rounded" style={{ background: "var(--bg-inset)" }}>
          <span style={{ color: "var(--text-primary)" }}>{f.flag_type}</span>
          <div className="flex items-center gap-2">
            <span style={{ color: f.severity === "critical" ? "var(--color-danger)" : "var(--text-tertiary)" }}>{f.severity}</span>
            <span style={{ color: "var(--text-tertiary)" }}>{new Date(f.created_at).toLocaleDateString()}</span>
          </div>
        </div>
      ))}
    </div>
  );
}
