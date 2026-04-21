"use client";

import { useState } from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { formatPrice } from "@/lib/utils";
import {
  Package, Plus, Eye, Edit3, Trash2, XCircle, CheckCircle,
  Search, Filter, ArrowUpRight,
} from "lucide-react";
import { formatDistanceToNow, parseISO } from "date-fns";

interface Listing {
  id: string;
  title: string;
  price: number;
  currency: string;
  status: "active" | "sold" | "inactive" | "pending";
  type: string;
  view_count?: number;
  views?: number;
  created_at: string;
  images?: { url: string }[];
}

const MOCK: Listing[] = [
  { id: "lst-001", title: "iPhone 15 Pro Max 256GB", price: 4200, currency: "AED", status: "active", type: "buy_now", views: 312, created_at: new Date(Date.now() - 3 * 86400000).toISOString(), images: [{ url: "https://picsum.photos/seed/phone1/64/64" }] },
  { id: "lst-002", title: "Toyota Camry 2023 — Midnight Black", price: 89000, currency: "AED", status: "active", type: "standard_auction", views: 875, created_at: new Date(Date.now() - 7 * 86400000).toISOString(), images: [{ url: "https://picsum.photos/seed/car1/64/64" }] },
  { id: "lst-003", title: "Rolex Submariner — 2022 Box & Papers", price: 32000, currency: "AED", status: "sold", type: "standard_auction", views: 1240, created_at: new Date(Date.now() - 20 * 86400000).toISOString(), images: [{ url: "https://picsum.photos/seed/watch1/64/64" }] },
  { id: "lst-004", title: "MacBook Pro M3 — Space Black 1TB", price: 9800, currency: "AED", status: "inactive", type: "buy_now", views: 89, created_at: new Date(Date.now() - 15 * 86400000).toISOString(), images: [{ url: "https://picsum.photos/seed/mac1/64/64" }] },
  { id: "lst-005", title: "PS5 Console + 3 Games Bundle", price: 2100, currency: "AED", status: "active", type: "buy_now", views: 223, created_at: new Date(Date.now() - 5 * 86400000).toISOString(), images: [{ url: "https://picsum.photos/seed/ps5/64/64" }] },
  { id: "lst-006", title: "Canon EOS R5 Camera Body", price: 15000, currency: "AED", status: "pending", type: "buy_now", views: 44, created_at: new Date(Date.now() - 1 * 86400000).toISOString(), images: [{ url: "https://picsum.photos/seed/cam1/64/64" }] },
];

const STATUS_STYLES: Record<string, { label: string; cls: string }> = {
  active:   { label: "Active",   cls: "bg-emerald-50 text-emerald-700 border border-emerald-200" },
  sold:     { label: "Sold",     cls: "bg-blue-50 text-blue-700 border border-blue-200" },
  inactive: { label: "Paused",   cls: "bg-gray-100 text-gray-500 border border-gray-200" },
  pending:  { label: "Pending",  cls: "bg-amber-50 text-amber-700 border border-amber-200" },
};

type FilterStatus = "all" | "active" | "sold" | "inactive" | "pending";

function timeAgo(d: string) {
  try { return formatDistanceToNow(parseISO(d), { addSuffix: true }); }
  catch { return d; }
}

export default function SellerListingsPage() {
  const qc = useQueryClient();
  const { isAuthenticated } = useAuthStore();
  const [filter, setFilter] = useState<FilterStatus>("all");
  const [search, setSearch] = useState("");

  const { data: listings = MOCK, isLoading } = useQuery<Listing[]>({
    queryKey: ["seller-listings-full"],
    queryFn: async () => {
      try {
        const d = (await api.get("/listings/me")).data?.data ?? [];
        return Array.isArray(d) && d.length ? d : MOCK;
      } catch { return MOCK; }
    },
    enabled: isAuthenticated,
  });

  const toggleMutation = useMutation({
    mutationFn: (l: Listing) =>
      api.put(`/listings/${l.id}`, { status: l.status === "active" ? "inactive" : "active" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["seller-listings-full"] }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/listings/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["seller-listings-full"] }),
  });

  const filtered = listings.filter((l) => {
    const matchStatus = filter === "all" || l.status === filter;
    const matchSearch = !search || l.title.toLowerCase().includes(search.toLowerCase());
    return matchStatus && matchSearch;
  });

  const counts: Record<string, number> = { all: listings.length };
  listings.forEach((l) => { counts[l.status] = (counts[l.status] ?? 0) + 1; });

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div>
          <h1 className="text-xl font-bold text-gray-900">My Listings</h1>
          <p className="text-sm text-gray-400">{listings.length} total listings</p>
        </div>
        <Link
          href="/sell"
          className="flex items-center gap-1.5 px-4 py-2 bg-[#0071CE] text-white rounded-xl text-sm font-semibold hover:bg-[#005ba3] transition-colors"
        >
          <Plus className="w-4 h-4" /> New Listing
        </Link>
      </div>

      {/* Filters + Search */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex gap-1.5 flex-wrap">
          {(["all", "active", "sold", "inactive", "pending"] as FilterStatus[]).map((f) => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={`px-3 py-1.5 rounded-full text-xs font-semibold capitalize transition-colors ${
                filter === f
                  ? "bg-[#0071CE] text-white"
                  : "bg-white text-gray-500 border border-gray-200 hover:bg-gray-50"
              }`}
            >
              {f === "all" ? `All (${counts.all})` : `${f.charAt(0).toUpperCase() + f.slice(1)} (${counts[f] ?? 0})`}
            </button>
          ))}
        </div>

        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search listings..."
            className="pl-8 pr-3 py-1.5 text-sm border border-gray-200 rounded-xl bg-white outline-none focus:ring-2 focus:ring-[#0071CE]/20 focus:border-[#0071CE] w-56"
          />
        </div>
      </div>

      {/* Table */}
      <div className="bg-white rounded-2xl border border-gray-100 overflow-hidden">
        {/* Table header */}
        <div className="hidden md:grid grid-cols-[44px_1fr_120px_100px_80px_90px_80px] gap-3 px-4 py-2.5 bg-gray-50 border-b border-gray-100">
          {["", "Title", "Price", "Type", "Views", "Status", "Actions"].map((h) => (
            <p key={h} className="text-[10px] font-bold uppercase tracking-wider text-gray-400">{h}</p>
          ))}
        </div>

        {isLoading ? (
          <div className="space-y-2 p-4">
            {[1, 2, 3].map((i) => <div key={i} className="h-14 animate-pulse rounded-xl bg-gray-100" />)}
          </div>
        ) : filtered.length === 0 ? (
          <div className="py-16 text-center">
            <Package className="w-10 h-10 mx-auto mb-3 text-gray-200" />
            <p className="text-sm text-gray-400">
              {search ? `No listings matching "${search}"` : "No listings in this filter."}
            </p>
            <Link href="/sell" className="mt-2 text-xs text-[#0071CE] hover:underline inline-block">
              Create your first listing →
            </Link>
          </div>
        ) : (
          <div className="divide-y divide-gray-50">
            {filtered.map((l) => {
              const sc = STATUS_STYLES[l.status] ?? STATUS_STYLES.pending;
              const views = l.view_count ?? l.views ?? 0;
              const busy = toggleMutation.isPending || deleteMutation.isPending;
              return (
                <div
                  key={l.id}
                  className="grid md:grid-cols-[44px_1fr_120px_100px_80px_90px_80px] gap-3 items-center px-4 py-3.5 hover:bg-gray-50/50 transition-colors group"
                >
                  {/* Thumbnail */}
                  <div className="w-11 h-11 rounded-xl overflow-hidden bg-gray-100 shrink-0">
                    {l.images?.[0] ? (
                      <img src={l.images[0].url} alt={l.title} className="w-full h-full object-cover" />
                    ) : (
                      <div className="w-full h-full flex items-center justify-center">
                        <Package className="w-4 h-4 text-gray-300" />
                      </div>
                    )}
                  </div>

                  {/* Title + date */}
                  <div className="min-w-0">
                    <p className="text-sm font-semibold text-gray-900 truncate">{l.title}</p>
                    <p className="text-[11px] text-gray-400 mt-0.5">{timeAgo(l.created_at)}</p>
                  </div>

                  {/* Price */}
                  <p className="text-sm font-bold text-[#0071CE] tabular-nums hidden md:block">
                    {formatPrice(l.price, l.currency)}
                  </p>

                  {/* Type */}
                  <p className="text-xs text-gray-400 capitalize hidden md:block">
                    {l.type.replace(/_/g, " ")}
                  </p>

                  {/* Views */}
                  <div className="hidden md:flex items-center gap-1 text-xs text-gray-400">
                    <Eye className="w-3 h-3" /> {views.toLocaleString()}
                  </div>

                  {/* Status */}
                  <div className="hidden md:block">
                    <span className={`text-[10px] px-1.5 py-0.5 rounded-full font-bold ${sc.cls}`}>
                      {sc.label}
                    </span>
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-1">
                    <Link
                      href={`/listings/${l.id}`}
                      className="p-1.5 rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors"
                      title="View listing"
                    >
                      <ArrowUpRight className="w-3.5 h-3.5" />
                    </Link>
                    {l.status !== "sold" && (
                      <>
                        <button
                          onClick={() => toggleMutation.mutate(l)}
                          disabled={busy}
                          className="p-1.5 rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors"
                          title={l.status === "active" ? "Pause listing" : "Activate listing"}
                        >
                          {l.status === "active"
                            ? <XCircle className="w-3.5 h-3.5" />
                            : <CheckCircle className="w-3.5 h-3.5 text-emerald-500" />}
                        </button>
                        <button
                          onClick={() => {
                            if (confirm("Delete this listing permanently?")) deleteMutation.mutate(l.id);
                          }}
                          disabled={busy}
                          className="p-1.5 rounded-lg hover:bg-red-50 text-gray-400 hover:text-red-500 transition-colors"
                          title="Delete listing"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
