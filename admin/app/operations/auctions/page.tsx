"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import RightPanel from "@/components/shared/RightPanel";
import DataTable from "@/components/shared/DataTable";
import { mockAuctions } from "@/lib/mockData";
import { auctionsApi } from "@/lib/api";
import { getErrorMessage } from "@/lib/errorMessage";
import { useToastStore } from "@/lib/toast";
import { Pause, XCircle } from "lucide-react";

type AuctionRow = {
  id: string;
  title: string;
  bids: number;
  currentPrice: number;
  status: string;
  endsAt: string;
};

function normalizeAuctions(payload: unknown): AuctionRow[] {
  const box = payload as
    | { data?: Array<Record<string, unknown>>; meta?: unknown }
    | Array<Record<string, unknown>>
    | null
    | undefined;

  const rows = Array.isArray(box) ? box : Array.isArray(box?.data) ? box.data : [];
  return rows
    .map((item) => {
      const bids = Number(item.bid_count ?? item.bids ?? 0);
      const currentPrice = Number(item.current_bid ?? item.currentPrice ?? item.start_price ?? 0);
      return {
        id: String(item.id ?? ""),
        title: String(item.title ?? `Auction ${String(item.id ?? "")}`),
        bids: Number.isFinite(bids) ? bids : 0,
        currentPrice: Number.isFinite(currentPrice) ? currentPrice : 0,
        status: String(item.status ?? "scheduled"),
        endsAt: String(item.ends_at ?? item.endsAt ?? new Date().toISOString()),
      };
    })
    .filter((x) => x.id);
}

export default function AuctionsMonitorPage() {
  const qc = useQueryClient();
  const [selected, setSelected] = useState<AuctionRow | null>(null);
  const [moderationUnavailable, setModerationUnavailable] = useState(false);
  const showToast = useToastStore((s) => s.showToast);

  const { data: canModerate = true } = useQuery({
    queryKey: ["operations", "auctions", "canModerate"],
    queryFn: auctionsApi.canModerate,
    retry: 0,
  });

  const { data: liveAuctions, isLoading } = useQuery({
    queryKey: ["operations", "auctions"],
    queryFn: async () => {
      const res = await auctionsApi.list();
      return normalizeAuctions(res);
    },
    retry: 1,
  });

  const source: AuctionRow[] = liveAuctions?.length
    ? liveAuctions
    : mockAuctions.map((a) => ({
        id: a.id,
        title: a.title,
        bids: a.bids,
        currentPrice: a.currentPrice,
        status: a.status,
        endsAt: a.endsAt,
      }));

  const pauseMutation = useMutation({
    mutationFn: (id: string) => auctionsApi.pause(id),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: ["operations", "auctions"] });
      showToast({ type: "success", title: "Auction paused", message: `Action applied to auction ${id}.` });
    },
    onError: (error) => {
      const message = getErrorMessage(error, "Could not pause auction.");
      if (message.toLowerCase().includes("not available")) {
        setModerationUnavailable(true);
      }
      showToast({ type: "error", title: "Action failed", message });
    },
  });

  const cancelMutation = useMutation({
    mutationFn: (id: string) => auctionsApi.cancel(id),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: ["operations", "auctions"] });
      showToast({ type: "success", title: "Auction canceled", message: `Action applied to auction ${id}.` });
    },
    onError: (error) => {
      const message = getErrorMessage(error, "Could not cancel auction.");
      if (message.toLowerCase().includes("not available")) {
        setModerationUnavailable(true);
      }
      showToast({ type: "error", title: "Action failed", message });
    },
  });

  const actingId =
    (pauseMutation.variables as string | undefined) ||
    (cancelMutation.variables as string | undefined) ||
    null;
  const moderationBlocked = moderationUnavailable || canModerate === false;

  return (
    <div>
      <PageHeader title="Auctions Monitor" description="Live auction tracking and intervention controls" />

      {moderationBlocked ? (
        <div className="mb-3 px-3 py-2 rounded-lg text-xs" style={{ background: "var(--color-warning-light)", color: "var(--color-warning)" }}>
          Auction pause/cancel controls are not available on this backend yet.
        </div>
      ) : null}

      <DataTable
        columns={[
          { key: "id", label: "ID", render: (a: AuctionRow) => <span className="font-mono text-xs">{a.id}</span> },
          { key: "title", label: "Title" },
          { key: "bids", label: "Bids" },
          { key: "currentPrice", label: "Current Price", render: (a: AuctionRow) => `$${a.currentPrice.toLocaleString()}` },
          { key: "status", label: "Status", render: (a: AuctionRow) => <StatusBadge status={a.status} dot /> },
          { key: "endsAt", label: "Ends At", render: (a: AuctionRow) => new Date(a.endsAt).toLocaleString() },
          {
            key: "actions",
            label: "Actions",
            render: (a: AuctionRow) => (
              <div className="flex items-center gap-1">
                <button
                  className="p-1.5 rounded-md"
                  style={{ color: "var(--color-warning)" }}
                  title="Pause"
                  disabled={actingId === a.id || moderationBlocked}
                  onClick={(e) => {
                    e.stopPropagation();
                    pauseMutation.mutate(a.id);
                  }}
                >
                  <Pause className="w-4 h-4" />
                </button>
                <button
                  className="p-1.5 rounded-md"
                  style={{ color: "var(--color-danger)" }}
                  title="Cancel"
                  disabled={actingId === a.id || moderationBlocked}
                  onClick={(e) => {
                    e.stopPropagation();
                    cancelMutation.mutate(a.id);
                  }}
                >
                  <XCircle className="w-4 h-4" />
                </button>
              </div>
            ),
          },
        ]}
        data={source}
        isLoading={isLoading}
        loadingMessage="Loading auctions monitor..."
        rowKey={(a) => a.id}
        onRowClick={setSelected}
        emptyMessage="No active auctions found."
      />

      <RightPanel open={!!selected} onClose={() => setSelected(null)} title="Auction Details">
        {selected && (
          <div className="space-y-4">
            <div><p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Title</p><p className="text-sm font-medium" style={{ color: "var(--text-primary)" }}>{selected.title}</p></div>
            <div><p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Current Price</p><p className="text-lg font-bold" style={{ color: "var(--color-brand)" }}>${selected.currentPrice.toLocaleString()}</p></div>
            <div><p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Bids</p><p className="text-sm" style={{ color: "var(--text-secondary)" }}>{selected.bids} bids</p></div>
            <div><p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Status</p><StatusBadge status={selected.status} dot /></div>
            <div className="flex gap-2 pt-4" style={{ borderTop: "1px solid var(--border-default)" }}>
              <button
                className="flex-1 py-2 rounded-lg text-sm font-medium text-white"
                style={{ background: "var(--color-warning)" }}
                disabled={actingId === selected.id || moderationBlocked}
                onClick={() => pauseMutation.mutate(selected.id)}
              >
                {pauseMutation.isPending && actingId === selected.id ? "Applying..." : "Pause Auction"}
              </button>
              <button
                className="flex-1 py-2 rounded-lg text-sm font-medium text-white"
                style={{ background: "var(--color-danger)" }}
                disabled={actingId === selected.id || moderationBlocked}
                onClick={() => cancelMutation.mutate(selected.id)}
              >
                {cancelMutation.isPending && actingId === selected.id ? "Applying..." : "Cancel Auction"}
              </button>
            </div>
          </div>
        )}
      </RightPanel>
    </div>
  );
}
