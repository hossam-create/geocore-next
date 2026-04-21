"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import RightPanel from "@/components/shared/RightPanel";
import DataTable from "@/components/shared/DataTable";
import { mockEscrow } from "@/lib/mockData";
import { escrowApi } from "@/lib/api";
import { getErrorMessage } from "@/lib/errorMessage";
import { useToastStore } from "@/lib/toast";
import { Lock } from "lucide-react";

export default function EscrowTrackingPage() {
  const [selected, setSelected] = useState<typeof mockEscrow[0] | null>(null);
  const showToast = useToastStore((s) => s.showToast);

  const releaseMutation = useMutation({
    mutationFn: (id: string) => escrowApi.release(id, "Released by admin control center"),
    onSuccess: (_, id) => {
      showToast({ type: "success", title: "Funds released", message: `Escrow ${id} has been released.` });
    },
    onError: (error) => {
      showToast({ type: "error", title: "Release failed", message: getErrorMessage(error, "Could not release funds.") });
    },
  });

  const isReleasing = releaseMutation.isPending;

  const handleRelease = async () => {
    if (!selected || selected.decision !== "approved") return;
    releaseMutation.mutate(selected.id);
  };

  return (
    <div>
      <PageHeader title="Escrow Tracking" description="Monitor held funds — release requires Custodii approval" />

      <DataTable
        columns={[
          { key: "id", label: "Escrow ID", render: (e: (typeof mockEscrow)[number]) => <span className="font-mono text-xs">{e.id}</span> },
          { key: "orderId", label: "Order", render: (e: (typeof mockEscrow)[number]) => <span className="font-mono text-xs">{e.orderId}</span> },
          { key: "buyer", label: "Buyer" },
          { key: "seller", label: "Seller" },
          { key: "amount", label: "Amount", render: (e: (typeof mockEscrow)[number]) => `$${e.amount.toLocaleString()}` },
          { key: "decision", label: "Decision", render: (e: (typeof mockEscrow)[number]) => <StatusBadge status={e.decision} dot /> },
          { key: "status", label: "Status", render: (e: (typeof mockEscrow)[number]) => <StatusBadge status={e.status} /> },
        ]}
        data={mockEscrow}
        rowKey={(e) => e.id}
        onRowClick={setSelected}
        emptyMessage="No escrow records found."
      />

      <RightPanel open={!!selected} onClose={() => setSelected(null)} title="Escrow Details">
        {selected && (
          <div className="space-y-4">
            <div><p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Amount</p><p className="text-xl font-bold" style={{ color: "var(--text-primary)" }}>${selected.amount.toLocaleString()}</p></div>
            <div><p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Buyer</p><p className="text-sm" style={{ color: "var(--text-primary)" }}>{selected.buyer}</p></div>
            <div><p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Seller</p><p className="text-sm" style={{ color: "var(--text-secondary)" }}>{selected.seller}</p></div>
            <div><p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Custodii Decision</p><StatusBadge status={selected.decision} dot /></div>

            <div className="pt-4" style={{ borderTop: "1px solid var(--border-default)" }}>
              <div className="flex items-center gap-2 mb-3 p-3 rounded-lg" style={{ background: "var(--color-warning-light)" }}>
                <Lock className="w-4 h-4" style={{ color: "var(--color-warning)" }} />
                <p className="text-xs font-medium" style={{ color: "var(--color-warning)" }}>Release requires Custodii APPROVED decision</p>
              </div>
              <button
                disabled={selected.decision !== "approved" || isReleasing}
                onClick={handleRelease}
                className="w-full py-2.5 rounded-lg text-sm font-medium text-white transition-opacity"
                style={{
                  background: selected.decision === "approved" ? "var(--color-success)" : "var(--bg-inset)",
                  color: selected.decision === "approved" ? "white" : "var(--text-tertiary)",
                  opacity: selected.decision === "approved" ? 1 : 0.5,
                  cursor: selected.decision === "approved" ? "pointer" : "not-allowed",
                }}
              >
                {selected.decision === "approved" ? (isReleasing ? "Releasing..." : "Release Funds") : "Awaiting Decision"}
              </button>
            </div>
          </div>
        )}
      </RightPanel>
    </div>
  );
}
