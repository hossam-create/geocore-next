"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { formatPrice } from "@/lib/utils";
import {
  TrendingUp, Package, ShoppingBag, Star, Plus, Eye,
  XCircle, CheckCircle, ArrowUpRight, Wallet, BarChart2,
  Clock, AlertTriangle, ChevronRight, Store, Truck,
  Edit3, Trash2, Tag, DollarSign,
} from "lucide-react";

// ── Types ────────────────────────────────────────────────────────────────────

interface SellerStats {
  total_revenue: number;
  this_month_revenue: number;
  active_listings: number;
  total_listings: number;
  pending_orders: number;
  total_orders: number;
  average_rating: number;
  total_reviews: number;
  wallet_balance: number;
  store_visits: number;
}

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

interface Order {
  id: string;
  item_title: string;
  buyer_name?: string;
  amount: number;
  currency: string;
  status: "pending" | "confirmed" | "shipped" | "delivered" | "cancelled";
  created_at: string;
}

// ── Mock fallbacks ────────────────────────────────────────────────────────────

const MOCK_STATS: SellerStats = {
  total_revenue: 47850,
  this_month_revenue: 8400,
  active_listings: 8,
  total_listings: 12,
  pending_orders: 3,
  total_orders: 34,
  average_rating: 4.8,
  total_reviews: 127,
  wallet_balance: 12300,
  store_visits: 1240,
};

const MOCK_LISTINGS: Listing[] = [
  { id: "lst-001", title: "iPhone 15 Pro Max 256GB", price: 4200, currency: "AED", status: "active", type: "buy_now", views: 312, created_at: new Date(Date.now() - 3 * 86400000).toISOString(), images: [{ url: "https://picsum.photos/seed/phone1/64/64" }] },
  { id: "lst-002", title: "Toyota Camry 2023 — Midnight Black", price: 89000, currency: "AED", status: "active", type: "standard_auction", views: 875, created_at: new Date(Date.now() - 7 * 86400000).toISOString(), images: [{ url: "https://picsum.photos/seed/car1/64/64" }] },
  { id: "lst-003", title: "Rolex Submariner — 2022 Box & Papers", price: 32000, currency: "AED", status: "sold", type: "standard_auction", views: 1240, created_at: new Date(Date.now() - 20 * 86400000).toISOString(), images: [{ url: "https://picsum.photos/seed/watch1/64/64" }] },
  { id: "lst-004", title: "MacBook Pro M3 — Space Black 1TB", price: 9800, currency: "AED", status: "inactive", type: "buy_now", views: 89, created_at: new Date(Date.now() - 15 * 86400000).toISOString(), images: [{ url: "https://picsum.photos/seed/mac1/64/64" }] },
  { id: "lst-005", title: "PS5 Console + 3 Games Bundle", price: 2100, currency: "AED", status: "active", type: "buy_now", views: 223, created_at: new Date(Date.now() - 5 * 86400000).toISOString(), images: [{ url: "https://picsum.photos/seed/ps5/64/64" }] },
];

const MOCK_ORDERS: Order[] = [
  { id: "ord-001", item_title: "iPhone 15 Pro Max 256GB", buyer_name: "Ali Hassan", amount: 4200, currency: "AED", status: "pending", created_at: new Date(Date.now() - 86400000).toISOString() },
  { id: "ord-002", item_title: "Rolex Submariner", buyer_name: "Sarah Al-Mansoori", amount: 32000, currency: "AED", status: "delivered", created_at: new Date(Date.now() - 5 * 86400000).toISOString() },
  { id: "ord-003", item_title: "PS5 Console + 3 Games Bundle", buyer_name: "Mohammed Khalid", amount: 2100, currency: "AED", status: "confirmed", created_at: new Date(Date.now() - 2 * 86400000).toISOString() },
  { id: "ord-004", item_title: "AirPods Pro Max", buyer_name: "Layla Mansoor", amount: 1200, currency: "AED", status: "shipped", created_at: new Date(Date.now() - 3 * 86400000).toISOString() },
];

// ── Helpers ───────────────────────────────────────────────────────────────────

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const days = Math.floor(diff / 86400000);
  if (days === 0) return "Today";
  if (days === 1) return "Yesterday";
  return `${days}d ago`;
}

function getViews(l: Listing): number {
  return l.view_count ?? l.views ?? 0;
}

const STATUS_CONFIG: Record<string, { label: string; cls: string }> = {
  active: { label: "Active", cls: "bg-emerald-50 text-emerald-700 border border-emerald-200" },
  sold: { label: "Sold", cls: "bg-blue-50 text-blue-700 border border-blue-200" },
  inactive: { label: "Paused", cls: "bg-gray-100 text-gray-500 border border-gray-200" },
  pending: { label: "Pending", cls: "bg-amber-50 text-amber-700 border border-amber-200" },
};

const ORDER_STATUS: Record<string, { label: string; cls: string; icon: React.ElementType }> = {
  pending: { label: "Pending", cls: "bg-amber-50 text-amber-700", icon: Clock },
  confirmed: { label: "Confirmed", cls: "bg-violet-50 text-violet-700", icon: CheckCircle },
  shipped: { label: "Shipped", cls: "bg-blue-50 text-blue-700", icon: Truck },
  delivered: { label: "Delivered", cls: "bg-emerald-50 text-emerald-700", icon: CheckCircle },
  cancelled: { label: "Cancelled", cls: "bg-red-50 text-red-600", icon: XCircle },
};

// ── KPI Card ──────────────────────────────────────────────────────────────────

function KPI({ label, value, sub, icon: Icon, accent, href }: {
  label: string; value: string; sub?: string;
  icon: React.ElementType; accent: string; href?: string;
}) {
  const card = (
    <div className={`relative bg-white rounded-2xl border border-gray-100 p-5 overflow-hidden hover:shadow-sm transition-shadow group ${href ? "cursor-pointer" : ""}`}>
      <div className={`absolute top-0 left-0 right-0 h-[3px] ${accent}`} />
      <div className="flex items-start justify-between mb-3">
        <p className="text-xs font-semibold text-gray-400 uppercase tracking-wider">{label}</p>
        <div className="w-8 h-8 rounded-xl flex items-center justify-center bg-gray-50">
          <Icon className="w-4 h-4 text-gray-400" />
        </div>
      </div>
      <p className="text-2xl font-bold text-gray-900 tabular-nums">{value}</p>
      {sub && <p className="text-xs text-gray-400 mt-1">{sub}</p>}
      {href && <ArrowUpRight className="absolute bottom-4 right-4 w-3.5 h-3.5 text-gray-200 group-hover:text-gray-400 transition-colors" />}
    </div>
  );
  return href ? <Link href={href}>{card}</Link> : card;
}

// ── Main Page ─────────────────────────────────────────────────────────────────

export default function SellerDashboardPage() {
  const { user, isAuthenticated } = useAuthStore();
  const qc = useQueryClient();
  const [listingFilter, setListingFilter] = useState<"all" | "active" | "sold" | "inactive">("all");

  const { data: stats = MOCK_STATS } = useQuery<SellerStats>({
    queryKey: ["seller-stats"],
    queryFn: async () => {
      try { return (await api.get("/users/me/stats")).data?.data ?? MOCK_STATS; }
      catch { return MOCK_STATS; }
    },
    enabled: isAuthenticated,
  });

  const { data: listings = MOCK_LISTINGS } = useQuery<Listing[]>({
    queryKey: ["seller-listings"],
    queryFn: async () => {
      try {
        const d = (await api.get("/listings/me")).data?.data ?? [];
        return Array.isArray(d) && d.length ? d : MOCK_LISTINGS;
      } catch { return MOCK_LISTINGS; }
    },
    enabled: isAuthenticated,
  });

  const { data: orders = MOCK_ORDERS } = useQuery<Order[]>({
    queryKey: ["seller-orders"],
    queryFn: async () => {
      try {
        const d = (await api.get("/orders/selling")).data?.data ?? [];
        return Array.isArray(d) && d.length ? d : MOCK_ORDERS;
      } catch { return MOCK_ORDERS; }
    },
    enabled: isAuthenticated,
  });

  const pendingOrders = orders.filter((o) => o.status === "pending");
  const filteredListings = listingFilter === "all" ? listings : listings.filter((l) => l.status === listingFilter);

  const handleToggleListing = async (l: Listing) => {
    const newStatus = l.status === "active" ? "inactive" : "active";
    try { await api.put(`/listings/${l.id}`, { status: newStatus }); } catch {}
    qc.invalidateQueries({ queryKey: ["seller-listings"] });
  };

  const handleDeleteListing = async (id: string) => {
    if (!confirm("Delete this listing permanently?")) return;
    try { await api.delete(`/listings/${id}`); } catch {}
    qc.invalidateQueries({ queryKey: ["seller-listings"] });
  };

  return (
    <div className="space-y-6">

      {/* ── Alert: Pending Orders ── */}
      {/* ── Welcome ── */}
      <p className="text-sm text-gray-500">
        Welcome back, <span className="font-medium text-gray-700">{user?.name ?? "Seller"}</span>. Here&apos;s your store performance.
      </p>

      {pendingOrders.length > 0 && (
        <div className="flex items-center gap-3 bg-amber-50 border border-amber-200/70 rounded-xl px-4 py-3">
          <AlertTriangle className="w-4 h-4 text-amber-500 shrink-0" />
          <p className="text-sm text-amber-800 font-medium flex-1">
            You have <strong>{pendingOrders.length}</strong> pending {pendingOrders.length === 1 ? "order" : "orders"} awaiting your action.
          </p>
          <Link href="/seller/orders" className="text-xs font-semibold text-amber-700 hover:text-amber-900 whitespace-nowrap">
            Review now →
          </Link>
        </div>
      )}

      {/* ── KPI Strip ── */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <KPI label="This Month" value={formatPrice(stats.this_month_revenue, "AED")} sub={`of ${formatPrice(stats.total_revenue, "AED")} total`} icon={TrendingUp} accent="bg-emerald-500" href="/selling/analytics" />
        <KPI label="Active Listings" value={String(stats.active_listings)} sub={`${stats.total_listings} total`} icon={Package} accent="bg-blue-500" href="/seller/listings" />
        <KPI label="Pending Orders" value={String(stats.pending_orders)} sub={`${stats.total_orders} total orders`} icon={ShoppingBag} accent="bg-amber-500" href="/seller/orders" />
        <KPI label="Rating" value={stats.average_rating.toFixed(1)} sub={`${stats.total_reviews} reviews`} icon={Star} accent="bg-yellow-400" />
      </div>

      {/* ── Main Grid ── */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">

        {/* Left: Listings (2/3) */}
        <div className="lg:col-span-2 space-y-4">

          {/* Filter row */}
          <div className="flex items-center justify-between gap-2 flex-wrap">
            <div className="flex gap-1.5">
              {(["all", "active", "sold", "inactive"] as const).map((f) => (
                <button key={f} onClick={() => setListingFilter(f)}
                  className={`px-3 py-1.5 rounded-full text-xs font-semibold capitalize transition-colors ${listingFilter === f ? "bg-[#0071CE] text-white" : "bg-white text-gray-500 border border-gray-200 hover:bg-gray-50"}`}>
                  {f === "all" ? `All (${listings.length})` : `${f.charAt(0).toUpperCase() + f.slice(1)} (${listings.filter(l => l.status === f).length})`}
                </button>
              ))}
            </div>
            <Link href="/sell" className="text-xs text-[#0071CE] font-semibold hover:underline flex items-center gap-1">
              <Plus className="w-3 h-3" /> Add listing
            </Link>
          </div>

          {/* Listings */}
          <div className="bg-white rounded-2xl border border-gray-100 overflow-hidden">
            {filteredListings.length === 0 ? (
              <div className="py-16 text-center">
                <Package className="w-10 h-10 mx-auto mb-3 text-gray-200" />
                <p className="text-sm text-gray-400">No listings in this filter.</p>
                <Link href="/sell" className="mt-2 text-xs text-[#0071CE] hover:underline inline-block">Create your first listing →</Link>
              </div>
            ) : (
              <div className="divide-y divide-gray-50">
                {filteredListings.map((l) => {
                  const sc = STATUS_CONFIG[l.status] ?? STATUS_CONFIG.pending;
                  return (
                    <div key={l.id} className="flex items-center gap-3 px-4 py-3.5 hover:bg-gray-50/50 transition-colors group">
                      {/* Thumbnail */}
                      <div className="w-11 h-11 rounded-xl overflow-hidden bg-gray-100 shrink-0">
                        {l.images?.[0] ? (
                          <img src={l.images[0].url} alt={l.title} className="w-full h-full object-cover" />
                        ) : (
                          <div className="w-full h-full flex items-center justify-center">
                            <Package className="w-5 h-5 text-gray-300" />
                          </div>
                        )}
                      </div>

                      {/* Info */}
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-semibold text-gray-900 truncate">{l.title}</p>
                        <div className="flex items-center gap-2 mt-0.5">
                          <span className={`text-[10px] px-1.5 py-0.5 rounded-full font-bold ${sc.cls}`}>{sc.label}</span>
                          <span className="text-xs text-gray-400 flex items-center gap-0.5">
                            <Eye className="w-3 h-3" /> {getViews(l).toLocaleString()}
                          </span>
                          <span className="text-xs text-gray-400">{timeAgo(l.created_at)}</span>
                        </div>
                      </div>

                      {/* Price */}
                      <p className="text-sm font-bold text-[#0071CE] shrink-0 tabular-nums">
                        {formatPrice(l.price, l.currency)}
                      </p>

                      {/* Actions */}
                      <div className="flex items-center gap-1 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
                        <Link href={`/listings/${l.id}`} className="p-1.5 rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors" title="View">
                          <Eye className="w-3.5 h-3.5" />
                        </Link>
                        {l.status !== "sold" && (
                          <>
                            <button onClick={() => handleToggleListing(l)}
                              className="p-1.5 rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors"
                              title={l.status === "active" ? "Pause" : "Activate"}>
                              {l.status === "active" ? <XCircle className="w-3.5 h-3.5" /> : <CheckCircle className="w-3.5 h-3.5 text-emerald-500" />}
                            </button>
                            <button onClick={() => handleDeleteListing(l.id)}
                              className="p-1.5 rounded-lg hover:bg-red-50 text-gray-400 hover:text-red-500 transition-colors" title="Delete">
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

        {/* Right: Sidebar (1/3) */}
        <div className="space-y-4">

          {/* Revenue card */}
          <div className="bg-gradient-to-br from-[#0071CE] to-[#005ba1] rounded-2xl p-5 text-white">
            <div className="flex items-center gap-2 mb-3">
              <Wallet className="w-4 h-4 opacity-70" />
              <p className="text-xs font-semibold opacity-70 uppercase tracking-wide">Wallet Balance</p>
            </div>
            <p className="text-3xl font-bold tabular-nums">{formatPrice(stats.wallet_balance, "AED")}</p>
            <p className="text-xs opacity-60 mt-1">{stats.store_visits.toLocaleString()} store visits this month</p>
            <Link href="/wallet" className="inline-flex items-center gap-1 mt-3 text-xs font-semibold bg-white/20 hover:bg-white/30 rounded-lg px-3 py-1.5 transition-colors">
              View Wallet <ArrowUpRight className="w-3 h-3" />
            </Link>
          </div>

          {/* Rating card */}
          <div className="bg-white rounded-2xl border border-gray-100 p-5">
            <div className="flex items-center justify-between mb-3">
              <p className="text-sm font-semibold text-gray-900">Store Rating</p>
              <Link href="/reviews" className="text-xs text-[#0071CE] hover:underline">All reviews</Link>
            </div>
            <div className="flex items-end gap-2">
              <p className="text-4xl font-bold text-gray-900">{stats.average_rating.toFixed(1)}</p>
              <div className="mb-1">
                <div className="flex gap-0.5">
                  {[1, 2, 3, 4, 5].map((i) => (
                    <Star key={i} className={`w-4 h-4 ${i <= Math.round(stats.average_rating) ? "fill-yellow-400 text-yellow-400" : "text-gray-200"}`} />
                  ))}
                </div>
                <p className="text-xs text-gray-400 mt-0.5">{stats.total_reviews} reviews</p>
              </div>
            </div>
          </div>

          {/* Quick actions */}
          <div className="bg-white rounded-2xl border border-gray-100 p-4">
            <p className="text-xs font-bold uppercase tracking-widest text-gray-400 mb-3">Quick Actions</p>
            <div className="space-y-1.5">
              {[
                { label: "Post New Listing", icon: Plus, href: "/sell", cls: "bg-[#0071CE] text-white hover:bg-[#005ba3]" },
                { label: "My Sales Orders", icon: Tag, href: "/seller/orders", cls: "bg-gray-50 text-gray-700 hover:bg-gray-100 border border-gray-100" },
                { label: "Store Analytics", icon: BarChart2, href: "/seller/analytics", cls: "bg-gray-50 text-gray-700 hover:bg-gray-100 border border-gray-100" },
                { label: "Store Settings", icon: Store, href: "/seller/settings", cls: "bg-gray-50 text-gray-700 hover:bg-gray-100 border border-gray-100" },
              ].map(({ label, icon: Icon, href, cls }) => (
                <Link key={label} href={href}
                  className={`flex items-center justify-between px-3 py-2.5 rounded-xl text-xs font-semibold transition-colors ${cls}`}>
                  <span className="flex items-center gap-2"><Icon className="w-3.5 h-3.5" />{label}</span>
                  <ChevronRight className="w-3.5 h-3.5 opacity-50" />
                </Link>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* ── Recent Orders ── */}
      <div className="bg-white rounded-2xl border border-gray-100 overflow-hidden">
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-100">
          <div>
            <h2 className="text-sm font-semibold text-gray-900">Recent Sales Orders</h2>
            <p className="text-xs text-gray-400">{orders.length} total orders</p>
          </div>
          <Link href="/seller/orders" className="inline-flex items-center gap-1 text-xs text-[#0071CE] font-semibold hover:underline">
            View all <ArrowUpRight className="w-3 h-3" />
          </Link>
        </div>

        <div className="hidden md:grid grid-cols-[1fr_140px_100px_90px_80px] gap-4 px-5 py-2.5 bg-gray-50/60 border-b border-gray-100">
          {["Item", "Buyer", "Amount", "Status", "Date"].map((h) => (
            <p key={h} className="text-[10px] font-bold uppercase tracking-wider text-gray-400">{h}</p>
          ))}
        </div>

        <div className="divide-y divide-gray-50">
          {orders.slice(0, 6).map((order) => {
            const os = ORDER_STATUS[order.status] ?? ORDER_STATUS.pending;
            return (
              <div key={order.id} className="grid md:grid-cols-[1fr_140px_100px_90px_80px] gap-4 items-center px-5 py-3.5 hover:bg-gray-50/50 transition-colors">
                <p className="text-sm font-medium text-gray-900 truncate">{order.item_title}</p>
                <p className="text-xs text-gray-500 hidden md:block truncate">{order.buyer_name ?? "—"}</p>
                <p className="text-sm font-semibold text-emerald-600 hidden md:block tabular-nums">
                  +{formatPrice(order.amount, order.currency)}
                </p>
                <div className="hidden md:block">
                  <span className={`inline-flex items-center gap-1 text-[10px] font-bold px-2 py-0.5 rounded-full ${os.cls}`}>
                    <os.icon className="w-3 h-3" /> {os.label}
                  </span>
                </div>
                <p className="text-xs text-gray-400 hidden md:block">{timeAgo(order.created_at)}</p>
              </div>
            );
          })}
        </div>
      </div>

    </div>
  );
}

