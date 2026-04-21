"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import FiltersBar from "@/components/shared/FiltersBar";
import DataTable from "@/components/shared/DataTable";
import RightPanel from "@/components/shared/RightPanel";
import { mockListings } from "@/lib/mockData";
import { listingsApi } from "@/lib/api";
import { useToastStore } from "@/lib/toast";
import { Check, X, Star, Eye } from "lucide-react";

type ListingRow = {
  id: string;
  title: string;
  user: string;
  price: number;
  status: string;
  created: string;
};

function normalizeListings(payload: unknown): ListingRow[] {
  const box = payload as { data?: unknown[] } | unknown[] | null | undefined;
  const rows = Array.isArray(box) ? box : Array.isArray((box as { data?: unknown[] })?.data) ? (box as { data?: unknown[] }).data : [];
  return (rows as Record<string, unknown>[]).map((item) => ({
    id: String(item.id ?? ""),
    title: String(item.title ?? "Untitled"),
    user: String(item.seller_name ?? item.user_name ?? item.seller_id ?? "Unknown"),
    price: Number(item.price ?? item.start_price ?? 0),
    status: String(item.status ?? "pending"),
    created: String(item.created_at ?? item.created ?? new Date().toISOString()),
  })).filter((x) => x.id);
}

export default function ListingModerationPage() {
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [statusFilter, setStatusFilter] = useState("");
  const [selected, setSelected] = useState<ListingRow | null>(null);
  const [rejectReason, setRejectReason] = useState("");

  const { data: liveListings, isLoading } = useQuery({
    queryKey: ["operations", "listings", "moderation"],
    queryFn: async () => {
      try {
        const res = await listingsApi.pending();
        return normalizeListings(res);
      } catch {
        try {
          const res = await listingsApi.list({ status: "pending" });
          return normalizeListings(res);
        } catch { return []; }
      }
    },
    retry: 1,
  });

  const source: ListingRow[] = liveListings?.length
    ? liveListings
    : mockListings.filter((l) => l.status === "pending" || l.status === "flagged").map((x) => ({
        id: x.id, title: x.title, user: x.user, price: x.price, status: x.status, created: x.created,
      }));

  const approveMutation = useMutation({
    mutationFn: (id: string) => listingsApi.approve(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["operations", "listings", "moderation"] });
      showToast({ type: "success", title: "Listing approved", message: "The listing is now live." });
    },
    onError: (error: { message?: string }) => {
      showToast({ type: "error", title: "Approval failed", message: error?.message ?? "Could not approve listing." });
    },
  });

  const rejectMutation = useMutation({
    mutationFn: (id: string) => listingsApi.reject(id, rejectReason || "Rejected by admin"),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["operations", "listings", "moderation"] });
      setRejectReason("");
      showToast({ type: "success", title: "Listing rejected", message: "The listing has been rejected." });
    },
    onError: (error: { message?: string }) => {
      showToast({ type: "error", title: "Rejection failed", message: error?.message ?? "Could not reject listing." });
    },
  });

  const featureMutation = useMutation({
    mutationFn: (id: string) => listingsApi.feature(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["operations", "listings", "moderation"] });
      showToast({ type: "success", title: "Listing featured", message: "Listing is now featured." });
    },
    onError: (error: { message?: string }) => {
      showToast({ type: "error", title: "Feature failed", message: error?.message ?? "Could not feature listing." });
    },
  });

  const filtered = source.filter((l) => {
    if (statusFilter && l.status !== statusFilter) return false;
    return true;
  });

  const pendingCount = source.filter((l) => l.status === "pending").length;
  const flaggedCount = source.filter((l) => l.status === "flagged").length;

  return (
    <div>
      <PageHeader title="Listing Moderation Queue" description="Approve, reject, or feature listings before they go live" />

      <div className="grid grid-cols-2 gap-3 mb-4">
        <div className="surface p-3 rounded-lg">
          <p className="text-xs font-medium" style={{ color: "var(--text-tertiary)" }}>Pending Review</p>
          <p className="text-lg font-bold" style={{ color: "var(--color-warning)" }}>{pendingCount}</p>
        </div>
        <div className="surface p-3 rounded-lg">
          <p className="text-xs font-medium" style={{ color: "var(--text-tertiary)" }}>Flagged</p>
          <p className="text-lg font-bold" style={{ color: "var(--color-danger)" }}>{flaggedCount}</p>
        </div>
      </div>

      <FiltersBar
        filters={[
          { key: "status", label: "Status", value: statusFilter, onChange: setStatusFilter, options: [
            { label: "All", value: "" }, { label: "Pending", value: "pending" }, { label: "Flagged", value: "flagged" },
          ]},
        ]}
      />

      <DataTable
        columns={[
          { key: "id", label: "ID", render: (l: ListingRow) => <span className="font-mono text-xs">{l.id}</span> },
          { key: "title", label: "Title" },
          { key: "user", label: "Seller" },
          { key: "price", label: "Price", render: (l: ListingRow) => `$${l.price.toLocaleString()}` },
          { key: "status", label: "Status", render: (l: ListingRow) => <StatusBadge status={l.status} dot /> },
          {
            key: "actions", label: "Quick Actions", render: (l: ListingRow) => (
              <div className="flex items-center gap-1">
                <button className="p-1.5 rounded-md" style={{ color: "var(--color-success)" }} title="Approve"
                  disabled={approveMutation.isPending}
                  onClick={(e) => { e.stopPropagation(); approveMutation.mutate(l.id); }}>
                  <Check className="w-4 h-4" />
                </button>
                <button className="p-1.5 rounded-md" style={{ color: "var(--color-danger)" }} title="Reject"
                  disabled={rejectMutation.isPending}
                  onClick={(e) => { e.stopPropagation(); rejectMutation.mutate(l.id); }}>
                  <X className="w-4 h-4" />
                </button>
                <button className="p-1.5 rounded-md" style={{ color: "var(--color-brand)" }} title="Feature"
                  disabled={featureMutation.isPending}
                  onClick={(e) => { e.stopPropagation(); featureMutation.mutate(l.id); }}>
                  <Star className="w-4 h-4" />
                </button>
              </div>
            ),
          },
        ]}
        data={filtered}
        isLoading={isLoading}
        loadingMessage="Loading moderation queue..."
        rowKey={(l) => l.id}
        onRowClick={setSelected}
        emptyMessage="No listings in moderation queue."
      />

      <RightPanel open={!!selected} onClose={() => setSelected(null)} title="Listing Review">
        {selected && (
          <div className="space-y-4">
            <div>
              <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Title</p>
              <p className="text-sm font-medium" style={{ color: "var(--text-primary)" }}>{selected.title}</p>
            </div>
            <div>
              <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Seller</p>
              <p className="text-sm" style={{ color: "var(--text-secondary)" }}>{selected.user}</p>
            </div>
            <div>
              <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Price</p>
              <p className="text-lg font-bold" style={{ color: "var(--color-brand)" }}>${selected.price.toLocaleString()}</p>
            </div>
            <div>
              <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Status</p>
              <StatusBadge status={selected.status} dot />
            </div>
            <textarea
              value={rejectReason}
              onChange={(e) => setRejectReason(e.target.value)}
              placeholder="Rejection reason (optional for approve, required for reject)"
              rows={2}
              className="w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }}
            />
            <div className="flex gap-2 pt-4" style={{ borderTop: "1px solid var(--border-default)" }}>
              <button
                className="flex-1 py-2 rounded-lg text-sm font-medium text-white flex items-center justify-center gap-1.5"
                style={{ background: "var(--color-success)" }}
                disabled={approveMutation.isPending}
                onClick={() => approveMutation.mutate(selected.id)}
              >
                <Check className="w-4 h-4" />Approve
              </button>
              <button
                className="flex-1 py-2 rounded-lg text-sm font-medium text-white flex items-center justify-center gap-1.5"
                style={{ background: "var(--color-danger)" }}
                disabled={rejectMutation.isPending}
                onClick={() => rejectMutation.mutate(selected.id)}
              >
                <X className="w-4 h-4" />Reject
              </button>
              <button
                className="py-2 px-3 rounded-lg text-sm font-medium text-white flex items-center justify-center gap-1.5"
                style={{ background: "var(--color-brand)" }}
                disabled={featureMutation.isPending}
                onClick={() => featureMutation.mutate(selected.id)}
              >
                <Star className="w-4 h-4" />Feature
              </button>
            </div>
          </div>
        )}
      </RightPanel>
    </div>
  );
}
