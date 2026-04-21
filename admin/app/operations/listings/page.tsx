"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import FiltersBar from "@/components/shared/FiltersBar";
import RightPanel from "@/components/shared/RightPanel";
import DataTable from "@/components/shared/DataTable";
import { mockListings } from "@/lib/mockData";
import { listingsApi } from "@/lib/api";
import { useToastStore } from "@/lib/toast";
import { Eye, Check, X } from "lucide-react";

type ListingRow = {
  id: string;
  title: string;
  user: string;
  price: number;
  status: string;
};

function normalizeListings(payload: unknown): ListingRow[] {
  const box = payload as
    | { data?: Array<Record<string, unknown>>; meta?: unknown }
    | Array<Record<string, unknown>>
    | null
    | undefined;

  const rows = Array.isArray(box) ? box : Array.isArray(box?.data) ? box.data : [];
  return rows
    .map((item) => {
      const price = Number(item.price ?? item.start_price ?? 0);
      return {
        id: String(item.id ?? ""),
        title: String(item.title ?? "Untitled"),
        user: String(item.seller_name ?? item.user_name ?? item.seller_id ?? "Unknown"),
        price: Number.isFinite(price) ? price : 0,
        status: String(item.status ?? "pending"),
      };
    })
    .filter((x) => x.id);
}

export default function ListingsQueuePage() {
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [selected, setSelected] = useState<ListingRow | null>(null);

  const { data: liveListings, isLoading } = useQuery({
    queryKey: ["operations", "listings"],
    queryFn: async () => {
      const res = await listingsApi.list();
      return normalizeListings(res);
    },
    retry: 1,
  });

  const source: ListingRow[] = liveListings?.length
    ? liveListings
    : mockListings.map((x) => ({
        id: x.id,
        title: x.title,
        user: x.user,
        price: x.price,
        status: x.status,
      }));

  const approveMutation = useMutation({
    mutationFn: (id: string) => listingsApi.approve(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["operations", "listings"] });
      showToast({ type: "success", title: "Listing approved", message: "The listing is now live." });
    },
    onError: (error: { message?: string }) => {
      showToast({ type: "error", title: "Approval failed", message: error?.message ?? "Could not approve listing." });
    },
  });

  const rejectMutation = useMutation({
    mutationFn: (id: string) => listingsApi.reject(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["operations", "listings"] });
      showToast({ type: "success", title: "Listing rejected", message: "The listing has been rejected." });
    },
    onError: (error: { message?: string }) => {
      showToast({ type: "error", title: "Rejection failed", message: error?.message ?? "Could not reject listing." });
    },
  });

  const mutationBusy = approveMutation.isPending || rejectMutation.isPending;

  const filtered = source.filter((l) => {
    if (search && !l.title.toLowerCase().includes(search.toLowerCase())) return false;
    if (statusFilter && l.status !== statusFilter) return false;
    return true;
  });

  return (
    <div>
      <PageHeader title="Listings Queue" description="Review, approve, or reject pending listings" />

      <FiltersBar
        search={search}
        onSearchChange={setSearch}
        searchPlaceholder="Search listings..."
        filters={[{
          key: "status", label: "All Status", value: statusFilter, onChange: setStatusFilter,
          options: [
            { label: "Pending", value: "pending" },
            { label: "Approved", value: "approved" },
            { label: "Flagged", value: "flagged" },
            { label: "Rejected", value: "rejected" },
          ],
        }]}
      />

      <DataTable
        columns={[
          { key: "id", label: "ID", render: (l: ListingRow) => <span className="font-mono text-xs">{l.id}</span> },
          { key: "title", label: "Title" },
          { key: "user", label: "User" },
          { key: "price", label: "Price", render: (l: ListingRow) => `$${l.price.toLocaleString()}` },
          { key: "status", label: "Status", render: (l: ListingRow) => <StatusBadge status={l.status} dot /> },
          {
            key: "actions",
            label: "Actions",
            render: (l: ListingRow) => (
              <div className="flex items-center gap-1">
                <button className="p-1.5 rounded-md transition-colors" style={{ color: "var(--text-tertiary)" }} title="View">
                  <Eye className="w-4 h-4" />
                </button>
                <button
                  className="p-1.5 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  style={{ color: "var(--color-success)" }}
                  title="Approve"
                  disabled={mutationBusy}
                  onClick={(e) => {
                    e.stopPropagation();
                    approveMutation.mutate(l.id);
                  }}
                >
                  <Check className="w-4 h-4" />
                </button>
                <button
                  className="p-1.5 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  style={{ color: "var(--color-danger)" }}
                  title="Reject"
                  disabled={mutationBusy}
                  onClick={(e) => {
                    e.stopPropagation();
                    rejectMutation.mutate(l.id);
                  }}
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            ),
          },
        ]}
        data={filtered}
        isLoading={isLoading}
        loadingMessage="Loading listings queue..."
        emptyMessage="No listings match your current filters."
        rowKey={(l) => l.id}
        onRowClick={setSelected}
      />

      <RightPanel open={!!selected} onClose={() => setSelected(null)} title="Listing Details">
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
              <p className="text-sm font-bold" style={{ color: "var(--text-primary)" }}>${selected.price.toLocaleString()}</p>
            </div>
            <div>
              <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Status</p>
              <StatusBadge status={selected.status} dot />
            </div>
            <div className="flex gap-2 pt-4" style={{ borderTop: "1px solid var(--border-default)" }}>
              <button
                className="flex-1 py-2 rounded-lg text-sm font-medium text-white"
                style={{ background: "var(--color-success)" }}
                onClick={() => approveMutation.mutate(selected.id)}
                disabled={mutationBusy}
              >
                {approveMutation.isPending ? "Approving..." : "Approve"}
              </button>
              <button
                className="flex-1 py-2 rounded-lg text-sm font-medium text-white"
                style={{ background: "var(--color-danger)" }}
                onClick={() => rejectMutation.mutate(selected.id)}
                disabled={mutationBusy}
              >
                {rejectMutation.isPending ? "Rejecting..." : "Reject"}
              </button>
            </div>
          </div>
        )}
      </RightPanel>
    </div>
  );
}
