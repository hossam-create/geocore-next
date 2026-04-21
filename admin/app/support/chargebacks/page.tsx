"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { chargebacksApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import { X, Upload } from "lucide-react";

interface Chargeback {
  id: string;
  payment_id: string;
  order_id: string | null;
  stripe_dispute_id: string;
  amount: number;
  currency: string;
  reason: string;
  status: string;
  evidence_due_by: string | null;
  created_at: string;
  updated_at: string;
}

const STATUS_VARIANTS: Record<string, "danger" | "warning" | "success" | "info" | "neutral"> = {
  open: "danger",
  under_review: "warning",
  won: "success",
  lost: "neutral",
};

export default function ChargebacksPage() {
  const qc = useQueryClient();
  const [filterStatus, setFilterStatus] = useState("");
  const [evidenceCb, setEvidenceCb] = useState<Chargeback | null>(null);
  const [evidenceType, setEvidenceType] = useState("refund_policy");
  const [evidenceDesc, setEvidenceDesc] = useState("");
  const [evidenceFile, setEvidenceFile] = useState("");

  const params: Record<string, string> = {};
  if (filterStatus) params.status = filterStatus;

  const { data, isLoading } = useQuery({
    queryKey: ["chargebacks", params],
    queryFn: () => chargebacksApi.list(params),
  });

  const chargebacks: Chargeback[] = (data?.data ?? data ?? []) as Chargeback[];

  const evidenceMut = useMutation({
    mutationFn: ({ id, ...d }: { id: string; evidence_type: string; description?: string; file_url?: string }) =>
      chargebacksApi.submitEvidence(id, d),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["chargebacks"] });
      setEvidenceCb(null);
      setEvidenceDesc("");
      setEvidenceFile("");
    },
  });

  const submitEvidence = () => {
    if (!evidenceCb) return;
    evidenceMut.mutate({
      id: evidenceCb.id,
      evidence_type: evidenceType,
      description: evidenceDesc || undefined,
      file_url: evidenceFile || undefined,
    });
  };

  const fmtDate = (d: string | null) => {
    if (!d) return "—";
    return new Date(d).toLocaleDateString();
  };

  const fmtAmount = (a: number, c: string) => `${c.toUpperCase()} ${a.toFixed(2)}`;

  const isOverdue = (due: string | null) => {
    if (!due) return false;
    return new Date(due) < new Date();
  };

  return (
    <div>
      <PageHeader title="Chargebacks" description="Bank-initiated dispute management & evidence submission" />

      {/* Filters */}
      <div className="flex gap-3 mb-4">
        <select value={filterStatus} onChange={(e) => setFilterStatus(e.target.value)}
          className="border border-slate-200 rounded-lg px-3 py-1.5 text-sm bg-white">
          <option value="">All Status</option>
          <option value="open">Open</option>
          <option value="under_review">Under Review</option>
          <option value="won">Won</option>
          <option value="lost">Lost</option>
        </select>
      </div>

      {/* Evidence Modal */}
      {evidenceCb && (
        <div className="p-4 mb-4 rounded-xl border border-slate-200 bg-white">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold text-sm">Submit Evidence — {evidenceCb.stripe_dispute_id || evidenceCb.id.slice(0, 8)}</h3>
            <button onClick={() => setEvidenceCb(null)}><X className="w-4 h-4 text-slate-400" /></button>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <select value={evidenceType} onChange={(e) => setEvidenceType(e.target.value)} className="border rounded-lg px-3 py-2 text-sm">
              <option value="refund_policy">Refund Policy</option>
              <option value="customer_communication">Customer Communication</option>
              <option value="delivery_confirmation">Delivery Confirmation</option>
              <option value="service_documentation">Service Documentation</option>
              <option value="uncategorized">Other</option>
            </select>
            <input placeholder="File URL (optional)" value={evidenceFile} onChange={(e) => setEvidenceFile(e.target.value)} className="border rounded-lg px-3 py-2 text-sm" />
            <textarea placeholder="Evidence description" value={evidenceDesc} onChange={(e) => setEvidenceDesc(e.target.value)} className="border rounded-lg px-3 py-2 text-sm md:col-span-2" rows={3} />
          </div>
          <div className="flex justify-end gap-2 mt-3">
            <button onClick={() => setEvidenceCb(null)} className="px-3 py-1.5 text-sm rounded-lg border">Cancel</button>
            <button onClick={submitEvidence} disabled={evidenceMut.isPending} className="px-3 py-1.5 text-sm rounded-lg text-white disabled:opacity-50" style={{ background: "var(--color-brand)" }}>
              {evidenceMut.isPending ? "Submitting..." : "Submit Evidence"}
            </button>
          </div>
        </div>
      )}

      {/* Table */}
      {isLoading ? (
        <div className="text-center py-12 text-slate-400">Loading...</div>
      ) : chargebacks.length === 0 ? (
        <div className="text-center py-12">
          <p className="text-slate-400 text-lg">No chargebacks found</p>
        </div>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-slate-200 bg-white">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-100 text-left text-xs text-slate-500">
                <th className="px-4 py-3">Dispute ID</th>
                <th className="px-4 py-3">Amount</th>
                <th className="px-4 py-3">Reason</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3">Evidence Due</th>
                <th className="px-4 py-3">Created</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody>
              {chargebacks.map((cb) => (
                <tr key={cb.id} className="border-b border-slate-50 hover:bg-slate-25">
                  <td className="px-4 py-3 font-mono text-xs">{cb.stripe_dispute_id || cb.id.slice(0, 8)}</td>
                  <td className="px-4 py-3">{fmtAmount(cb.amount, cb.currency)}</td>
                  <td className="px-4 py-3 text-xs text-slate-600 max-w-[200px] truncate">{cb.reason || "—"}</td>
                  <td className="px-4 py-3"><StatusBadge status={cb.status} variant={STATUS_VARIANTS[cb.status] ?? "neutral"} /></td>
                  <td className="px-4 py-3 text-xs">
                    <span className={isOverdue(cb.evidence_due_by) ? "text-red-500 font-medium" : "text-slate-500"}>
                      {fmtDate(cb.evidence_due_by)}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-xs text-slate-500">{fmtDate(cb.created_at)}</td>
                  <td className="px-4 py-3">
                    {cb.status === "open" && (
                      <button onClick={() => setEvidenceCb(cb)} className="flex items-center gap-1 px-2 py-1 rounded text-xs font-medium text-white" style={{ background: "var(--color-brand)" }}>
                        <Upload className="w-3 h-3" /> Evidence
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
