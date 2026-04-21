"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import { listingsApi } from "@/lib/api";
import { useToastStore } from "@/lib/toast";
import { CheckCircle, XCircle, Star, Clock, DollarSign, Tag, User } from "lucide-react";

type ListingDetail = {
  id: string;
  title: string;
  description: string;
  price: number;
  category: string;
  seller_id: string;
  seller_name: string;
  status: string;
  images: string[];
  views: number;
  watchlist_count: number;
  created_at: string;
  updated_at: string;
};

const MOCK_LISTING: ListingDetail = {
  id: "L-001",
  title: "iPhone 15 Pro Max 256GB",
  description: "Brand new, sealed box. International warranty. Space Black color. Includes screen protector and case.",
  price: 1200,
  category: "Electronics",
  seller_id: "S-001",
  seller_name: "TechStore EG",
  status: "active",
  images: [],
  views: 342,
  watchlist_count: 28,
  created_at: "2026-04-15T08:00:00Z",
  updated_at: "2026-04-19T14:30:00Z",
};

export default function ListingDetailPage() {
  const params = useParams();
  const id = params.id as string;
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [rejectReason, setRejectReason] = useState("");
  const [showReject, setShowReject] = useState(false);

  const { data: liveListing, isLoading } = useQuery({
    queryKey: ["listings", id],
    queryFn: async () => {
      try {
        return await listingsApi.get(id) as ListingDetail;
      } catch { return null; }
    },
    retry: 1,
  });

  const listing: ListingDetail = liveListing ?? { ...MOCK_LISTING, id };

  const approveMutation = useMutation({
    mutationFn: () => listingsApi.approve(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["listings", id] });
      showToast({ type: "success", title: "Listing approved" });
    },
    onError: () => showToast({ type: "error", title: "Approve failed" }),
  });

  const rejectMutation = useMutation({
    mutationFn: (reason: string) => listingsApi.reject(id, reason),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["listings", id] });
      setShowReject(false);
      showToast({ type: "success", title: "Listing rejected" });
    },
    onError: () => showToast({ type: "error", title: "Reject failed" }),
  });

  if (isLoading) return <div className="p-6 text-sm" style={{ color: "var(--text-tertiary)" }}>Loading listing...</div>;

  return (
    <div>
      <PageHeader title={listing.title} description={`Listing ${listing.id}`} />

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Main content */}
        <div className="lg:col-span-2 space-y-4">
          <div className="surface p-5 rounded-lg">
            <div className="flex items-center justify-between mb-4">
              <StatusBadge status={listing.status} dot />
              <span className="text-lg font-bold" style={{ color: "var(--color-brand)" }}>${listing.price.toLocaleString()}</span>
            </div>
            <div className="space-y-3">
              <div>
                <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Description</p>
                <p className="text-sm" style={{ color: "var(--text-secondary)" }}>{listing.description}</p>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="flex items-center gap-2">
                  <Tag className="w-3.5 h-3.5" style={{ color: "var(--text-tertiary)" }} />
                  <span className="text-sm" style={{ color: "var(--text-secondary)" }}>{listing.category}</span>
                </div>
                <div className="flex items-center gap-2">
                  <User className="w-3.5 h-3.5" style={{ color: "var(--text-tertiary)" }} />
                  <span className="text-sm" style={{ color: "var(--text-secondary)" }}>{listing.seller_name}</span>
                </div>
                <div className="flex items-center gap-2">
                  <Clock className="w-3.5 h-3.5" style={{ color: "var(--text-tertiary)" }} />
                  <span className="text-sm" style={{ color: "var(--text-secondary)" }}>{new Date(listing.created_at).toLocaleDateString()}</span>
                </div>
                <div className="flex items-center gap-2">
                  <Star className="w-3.5 h-3.5" style={{ color: "var(--text-tertiary)" }} />
                  <span className="text-sm" style={{ color: "var(--text-secondary)" }}>{listing.watchlist_count} watching</span>
                </div>
              </div>
            </div>
          </div>

          {/* Actions */}
          {listing.status === "pending_review" && (
            <div className="surface p-4 rounded-lg">
              <h3 className="text-sm font-semibold mb-3" style={{ color: "var(--text-primary)" }}>Moderation Actions</h3>
              <div className="flex gap-2">
                <button
                  onClick={() => approveMutation.mutate()}
                  disabled={approveMutation.isPending}
                  className="flex-1 py-2 rounded-lg text-sm font-medium text-white flex items-center justify-center gap-1.5"
                  style={{ background: "var(--color-success)" }}
                >
                  <CheckCircle className="w-4 h-4" />Approve
                </button>
                <button
                  onClick={() => setShowReject(true)}
                  className="flex-1 py-2 rounded-lg text-sm font-medium text-white flex items-center justify-center gap-1.5"
                  style={{ background: "var(--color-danger)" }}
                >
                  <XCircle className="w-4 h-4" />Reject
                </button>
              </div>
              {showReject && (
                <div className="mt-3 flex gap-2">
                  <input
                    type="text"
                    value={rejectReason}
                    onChange={(e) => setRejectReason(e.target.value)}
                    placeholder="Reason for rejection..."
                    className="flex-1 px-3 py-1.5 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-red-400"
                    style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }}
                  />
                  <button
                    onClick={() => rejectMutation.mutate(rejectReason || "Does not meet listing standards")}
                    disabled={rejectMutation.isPending}
                    className="px-3 py-1.5 rounded-lg text-xs font-medium text-white"
                    style={{ background: "var(--color-danger)" }}
                  >
                    Confirm
                  </button>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Sidebar stats */}
        <div className="space-y-4">
          <div className="surface p-4 rounded-lg">
            <h3 className="text-sm font-semibold mb-3" style={{ color: "var(--text-primary)" }}>Performance</h3>
            <div className="space-y-3">
              <div className="flex justify-between text-sm">
                <span style={{ color: "var(--text-tertiary)" }}>Views</span>
                <span className="font-medium" style={{ color: "var(--text-primary)" }}>{listing.views}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span style={{ color: "var(--text-tertiary)" }}>Watchlist</span>
                <span className="font-medium" style={{ color: "var(--text-primary)" }}>{listing.watchlist_count}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span style={{ color: "var(--text-tertiary)" }}>Updated</span>
                <span style={{ color: "var(--text-secondary)" }}>{new Date(listing.updated_at).toLocaleDateString()}</span>
              </div>
            </div>
          </div>

          <div className="surface p-4 rounded-lg">
            <h3 className="text-sm font-semibold mb-3" style={{ color: "var(--text-primary)" }}>Seller</h3>
            <p className="text-sm font-medium" style={{ color: "var(--text-primary)" }}>{listing.seller_name}</p>
            <p className="text-xs font-mono mt-1" style={{ color: "var(--text-tertiary)" }}>{listing.seller_id}</p>
          </div>
        </div>
      </div>
    </div>
  );
}
