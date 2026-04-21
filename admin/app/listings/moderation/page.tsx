"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import { listingsApi } from "@/lib/api";
import { useToastStore } from "@/lib/toast";
import { CheckCircle, XCircle, Pencil, Image } from "lucide-react";

// NOTE: This page is also accessible via /operations/listings/moderation (sidebar link).
// Both routes provide the same Listing Moderation Queue functionality.

type ModerationItem = {
  id: string;
  title: string;
  price: number;
  category: string;
  seller_name: string;
  seller_id: string;
  image_url?: string;
  status: string;
  created_at: string;
};

const MOCK_QUEUE: ModerationItem[] = [
  { id: "L-201", title: "iPhone 15 Pro Max 256GB", price: 1200, category: "Electronics", seller_name: "TechStore EG", seller_id: "S-001", image_url: "", status: "pending_review", created_at: "2026-04-20T08:00:00Z" },
  { id: "L-202", title: "Vintage Rolex Daytona", price: 8500, category: "Jewelry", seller_name: "LuxWatch", seller_id: "S-006", image_url: "", status: "pending_review", created_at: "2026-04-20T07:30:00Z" },
  { id: "L-203", title: "Gaming PC RTX 5090 Build", price: 3200, category: "Electronics", seller_name: "PCMaster", seller_id: "S-007", image_url: "", status: "pending_review", created_at: "2026-04-20T07:00:00Z" },
  { id: "L-204", title: "Gold Chain 24K 50g", price: 4800, category: "Jewelry", seller_name: "GoldMarket", seller_id: "S-008", image_url: "", status: "pending_review", created_at: "2026-04-19T22:00:00Z" },
  { id: "L-205", title: "Nike Air Jordan 1 Retro", price: 280, category: "Fashion", seller_name: "SneakerHub", seller_id: "S-009", image_url: "", status: "pending_review", created_at: "2026-04-19T20:00:00Z" },
  { id: "L-206", title: "MacBook Pro M4 16\"", price: 2800, category: "Electronics", seller_name: "AppleReseller", seller_id: "S-010", image_url: "", status: "pending_review", created_at: "2026-04-19T18:00:00Z" },
];

function normalizeQueue(payload: unknown): ModerationItem[] {
  const box = payload as { data?: unknown[] } | unknown[] | null | undefined;
  const rows = Array.isArray(box) ? box : Array.isArray((box as { data?: unknown[] })?.data) ? (box as { data?: unknown[] }).data : [];
  return (rows as Record<string, unknown>[]).map((item) => ({
    id: String(item.id ?? ""),
    title: String(item.title ?? "Untitled"),
    price: Number(item.price ?? 0),
    category: String(item.category ?? ""),
    seller_name: String(item.seller_name ?? item.seller ?? "Unknown"),
    seller_id: String(item.seller_id ?? ""),
    image_url: item.image_url ? String(item.image_url) : undefined,
    status: String(item.status ?? "pending_review"),
    created_at: String(item.created_at ?? new Date().toISOString()),
  })).filter((x) => x.id);
}

export default function ListingModerationPage() {
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [rejectReason, setRejectReason] = useState("");
  const [rejectingId, setRejectingId] = useState<string | null>(null);

  const { data: liveQueue, isLoading } = useQuery({
    queryKey: ["listings", "moderation"],
    queryFn: async () => {
      try {
        const res = await listingsApi.moderation();
        return normalizeQueue(res);
      } catch { return []; }
    },
    retry: 1,
  });

  const queue: ModerationItem[] = liveQueue?.length ? liveQueue : MOCK_QUEUE;

  const approveMutation = useMutation({
    mutationFn: (id: string) => listingsApi.approve(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["listings", "moderation"] });
      showToast({ type: "success", title: "Listing approved", message: "The listing is now live." });
    },
    onError: () => showToast({ type: "error", title: "Approve failed", message: "Could not approve listing." }),
  });

  const rejectMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) => listingsApi.reject(id, reason),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["listings", "moderation"] });
      setRejectingId(null);
      setRejectReason("");
      showToast({ type: "success", title: "Listing rejected", message: "Seller has been notified." });
    },
    onError: () => showToast({ type: "error", title: "Reject failed", message: "Could not reject listing." }),
  });

  const bulkMutation = useMutation({
    mutationFn: (action: "approve" | "reject") => listingsApi.bulk(Array.from(selected), action),
    onSuccess: (_, action) => {
      qc.invalidateQueries({ queryKey: ["listings", "moderation"] });
      setSelected(new Set());
      showToast({ type: "success", title: `Bulk ${action} complete`, message: `${selected.size} listings updated.` });
    },
    onError: () => showToast({ type: "error", title: "Bulk action failed", message: "Could not process bulk action." }),
  });

  const toggleSelect = (id: string) => {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id); else next.add(id);
    setSelected(next);
  };

  const toggleAll = () => {
    if (selected.size === queue.length) setSelected(new Set());
    else setSelected(new Set(queue.map((l) => l.id)));
  };

  return (
    <div>
      <PageHeader
        title="Listing Moderation Queue"
        description={`${queue.length} listings pending review`}
        actions={
          selected.size > 0 ? (
            <div className="flex gap-2">
              <button
                onClick={() => bulkMutation.mutate("approve")}
                disabled={bulkMutation.isPending}
                className="px-3 py-1.5 rounded-lg text-xs font-medium text-white flex items-center gap-1.5"
                style={{ background: "var(--color-success)" }}
              >
                <CheckCircle className="w-3.5 h-3.5" />Approve {selected.size}
              </button>
              <button
                onClick={() => bulkMutation.mutate("reject")}
                disabled={bulkMutation.isPending}
                className="px-3 py-1.5 rounded-lg text-xs font-medium text-white flex items-center gap-1.5"
                style={{ background: "var(--color-danger)" }}
              >
                <XCircle className="w-3.5 h-3.5" />Reject {selected.size}
              </button>
            </div>
          ) : undefined
        }
      />

      {isLoading && <div className="text-center py-12 text-sm" style={{ color: "var(--text-tertiary)" }}>Loading moderation queue...</div>}

      {!isLoading && queue.length === 0 && (
        <div className="surface p-12 rounded-lg text-center">
          <CheckCircle className="w-10 h-10 mx-auto mb-3" style={{ color: "var(--color-success)" }} />
          <p className="text-sm font-medium" style={{ color: "var(--text-primary)" }}>All caught up!</p>
          <p className="text-xs mt-1" style={{ color: "var(--text-tertiary)" }}>No listings pending moderation.</p>
        </div>
      )}

      <div className="space-y-3">
        {queue.length > 0 && (
          <label className="flex items-center gap-2 text-xs font-medium cursor-pointer" style={{ color: "var(--text-tertiary)" }}>
            <input type="checkbox" checked={selected.size === queue.length} onChange={toggleAll} className="rounded" />
            Select all ({queue.length})
          </label>
        )}

        {queue.map((item) => (
          <div key={item.id} className="surface rounded-lg p-4 flex gap-4 items-start" style={{ border: selected.has(item.id) ? "2px solid var(--color-brand)" : "1px solid var(--border-default)" }}>
            <input type="checkbox" checked={selected.has(item.id)} onChange={() => toggleSelect(item.id)} className="mt-1 rounded" />

            {/* Image placeholder */}
            <div className="w-20 h-20 rounded-lg flex-shrink-0 flex items-center justify-center" style={{ background: "var(--bg-inset)" }}>
              {item.image_url ? (
                <img src={item.image_url} alt="" className="w-full h-full object-cover rounded-lg" />
              ) : (
                <Image className="w-6 h-6" style={{ color: "var(--text-tertiary)" }} />
              )}
            </div>

            {/* Content */}
            <div className="flex-1 min-w-0">
              <div className="flex items-start justify-between gap-2">
                <div>
                  <h4 className="text-sm font-medium truncate" style={{ color: "var(--text-primary)" }}>{item.title}</h4>
                  <p className="text-xs mt-0.5" style={{ color: "var(--text-tertiary)" }}>
                    {item.seller_name} · {item.category} · {new Date(item.created_at).toLocaleDateString()}
                  </p>
                </div>
                <span className="text-sm font-bold flex-shrink-0" style={{ color: "var(--color-brand)" }}>${item.price.toLocaleString()}</span>
              </div>

              <div className="flex items-center gap-2 mt-3">
                <button
                  onClick={() => approveMutation.mutate(item.id)}
                  disabled={approveMutation.isPending}
                  className="px-3 py-1.5 rounded-lg text-xs font-medium text-white flex items-center gap-1.5"
                  style={{ background: "var(--color-success)" }}
                >
                  <CheckCircle className="w-3.5 h-3.5" />Approve
                </button>
                <button
                  onClick={() => setRejectingId(item.id)}
                  className="px-3 py-1.5 rounded-lg text-xs font-medium text-white flex items-center gap-1.5"
                  style={{ background: "var(--color-danger)" }}
                >
                  <XCircle className="w-3.5 h-3.5" />Reject
                </button>
                <button
                  className="px-3 py-1.5 rounded-lg text-xs font-medium flex items-center gap-1.5"
                  style={{ background: "var(--bg-surface)", border: "1px solid var(--border-default)", color: "var(--text-secondary)" }}
                >
                  <Pencil className="w-3.5 h-3.5" />Request Edit
                </button>
                <StatusBadge status={item.status} />
              </div>

              {/* Reject reason input */}
              {rejectingId === item.id && (
                <div className="mt-3 flex gap-2">
                  <input
                    type="text"
                    value={rejectReason}
                    onChange={(e) => setRejectReason(e.target.value)}
                    placeholder="Rejection reason..."
                    className="flex-1 px-3 py-1.5 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-red-400"
                    style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }}
                  />
                  <button
                    onClick={() => rejectMutation.mutate({ id: item.id, reason: rejectReason || "Does not meet listing standards" })}
                    disabled={rejectMutation.isPending}
                    className="px-3 py-1.5 rounded-lg text-xs font-medium text-white"
                    style={{ background: "var(--color-danger)" }}
                  >
                    Confirm
                  </button>
                  <button
                    onClick={() => { setRejectingId(null); setRejectReason(""); }}
                    className="px-3 py-1.5 rounded-lg text-xs font-medium"
                    style={{ color: "var(--text-tertiary)" }}
                  >
                    Cancel
                  </button>
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
