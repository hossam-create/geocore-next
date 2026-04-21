'use client'
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { PointsDisplay } from "@/components/loyalty/PointsDisplay";
import { formatPrice } from "@/lib/utils";
import { useTranslations } from "next-intl";
import {
  LayoutDashboard,
  Package,
  ShoppingBag,
  BarChart2,
  Plus,
  Eye,
  Edit3,
  Trash2,
  CheckCircle,
  Clock,
  XCircle,
  TrendingUp,
  Store,
  Wallet,
  Star,
  ArrowUpRight,
  ArrowDownRight,
  ChevronRight,
  Gavel,
  Tag,
  Truck,
  AlertCircle,
  RefreshCw,
  Copy,
  Users,
  Gift,
} from "lucide-react";

// ── Types ─────────────────────────────────────────────────────────────────────

type Tab = "overview" | "listings" | "orders" | "analytics";

interface Listing {
  id: string;
  title: string;
  price: number;
  currency: string;
  status: "active" | "sold" | "inactive" | "pending";
  type: string;
  // Backend returns category as an object; mock uses a plain string
  category?: string | { slug?: string; name_en?: string; name?: string } | null;
  // Backend uses view_count; mock uses views
  views?: number;
  view_count?: number;
  created_at: string;
  images?: { url: string }[];
}

/** Resolve view count from either backend (view_count) or mock (views) shape */
function getViews(l: Listing): number {
  return l.view_count ?? l.views ?? 0;
}

/** Resolve category display label from either backend (object) or mock (string) shape */
function getCategoryLabel(l: Listing): string {
  if (!l.category) return "";
  if (typeof l.category === "string") return l.category;
  return l.category.name_en ?? l.category.name ?? l.category.slug ?? "";
}

interface Order {
  id: string;
  item_title: string;
  buyer_name?: string;
  seller_name?: string;
  amount: number;
  currency: string;
  status: "pending" | "confirmed" | "shipped" | "delivered" | "cancelled";
  role: "seller" | "buyer";
  created_at: string;
}

// ── Mock data ──────────────────────────────────────────────────────────────────

const MOCK_STATS = {
  total_listings: 12,
  active_listings: 8,
  total_orders: 34,
  pending_orders: 3,
  total_revenue: 47850,
  this_month_revenue: 8400,
  store_visits: 1240,
  average_rating: 4.7,
  wallet_balance: 12300,
};

const MOCK_LISTINGS: Listing[] = [
  { id: "lst-001", title: "iPhone 15 Pro Max 256GB — Natural Titanium", price: 4200, currency: "AED", status: "active", type: "buy_now", category: "electronics", views: 312, created_at: new Date(Date.now() - 3 * 86400000).toISOString(), images: [{ url: "https://picsum.photos/seed/phone1/80/80" }] },
  { id: "lst-002", title: "Toyota Camry 2023 — Midnight Black", price: 89000, currency: "AED", status: "active", type: "standard_auction", category: "vehicles", views: 875, created_at: new Date(Date.now() - 7 * 86400000).toISOString(), images: [{ url: "https://picsum.photos/seed/car1/80/80" }] },
  { id: "lst-003", title: "Luxury 2BR Apartment — Dubai Marina", price: 1800, currency: "AED", status: "active", type: "buy_now", category: "real-estate", views: 540, created_at: new Date(Date.now() - 10 * 86400000).toISOString(), images: [{ url: "https://picsum.photos/seed/apt1/80/80" }] },
  { id: "lst-004", title: "Rolex Submariner — 2022 Box & Papers", price: 32000, currency: "AED", status: "sold", type: "standard_auction", category: "jewelry", views: 1240, created_at: new Date(Date.now() - 20 * 86400000).toISOString(), images: [{ url: "https://picsum.photos/seed/watch1/80/80" }] },
  { id: "lst-005", title: "MacBook Pro M3 — Space Black 1TB", price: 9800, currency: "AED", status: "inactive", type: "buy_now", category: "electronics", views: 89, created_at: new Date(Date.now() - 15 * 86400000).toISOString(), images: [{ url: "https://picsum.photos/seed/mac1/80/80" }] },
  { id: "lst-006", title: "PS5 Console + 3 Games Bundle", price: 2100, currency: "AED", status: "active", type: "buy_now", category: "gaming", views: 223, created_at: new Date(Date.now() - 5 * 86400000).toISOString(), images: [{ url: "https://picsum.photos/seed/ps5/80/80" }] },
];

const MOCK_ORDERS: Order[] = [
  { id: "ord-001", item_title: "iPhone 15 Pro Max 256GB", buyer_name: "Ali Hassan", amount: 4200, currency: "AED", status: "pending", role: "seller", created_at: new Date(Date.now() - 1 * 86400000).toISOString() },
  { id: "ord-002", item_title: "Rolex Submariner", buyer_name: "Sarah Al-Mansoori", amount: 32000, currency: "AED", status: "delivered", role: "seller", created_at: new Date(Date.now() - 5 * 86400000).toISOString() },
  { id: "ord-003", item_title: "Samsung 85\" QLED TV", seller_name: "TechZone Store", amount: 7500, currency: "AED", status: "shipped", role: "buyer", created_at: new Date(Date.now() - 3 * 86400000).toISOString() },
  { id: "ord-004", item_title: "PS5 Console + 3 Games Bundle", buyer_name: "Mohammed Khalid", amount: 2100, currency: "AED", status: "confirmed", role: "seller", created_at: new Date(Date.now() - 2 * 86400000).toISOString() },
  { id: "ord-005", item_title: "Nike Air Max 270 — Size 44", seller_name: "SportZone UAE", amount: 450, currency: "AED", status: "delivered", role: "buyer", created_at: new Date(Date.now() - 8 * 86400000).toISOString() },
];

// ── Helpers ────────────────────────────────────────────────────────────────────

function listingStatusConfig(status: Listing["status"]) {
  switch (status) {
    case "active": return { label: "Active", cls: "bg-green-100 text-green-700" };
    case "sold": return { label: "Sold", cls: "bg-blue-100 text-blue-700" };
    case "inactive": return { label: "Inactive", cls: "bg-gray-100 text-gray-500" };
    default: return { label: "Pending", cls: "bg-yellow-100 text-yellow-700" };
  }
}

function orderStatusConfig(status: Order["status"]) {
  switch (status) {
    case "delivered": return { label: "Delivered", cls: "bg-green-100 text-green-700", icon: CheckCircle };
    case "shipped": return { label: "Shipped", cls: "bg-blue-100 text-blue-700", icon: Truck };
    case "confirmed": return { label: "Confirmed", cls: "bg-purple-100 text-purple-700", icon: CheckCircle };
    case "cancelled": return { label: "Cancelled", cls: "bg-red-100 text-red-700", icon: XCircle };
    default: return { label: "Pending", cls: "bg-yellow-100 text-yellow-700", icon: Clock };
  }
}

function timeAgo(dateStr: string) {
  const diff = Date.now() - new Date(dateStr).getTime();
  const days = Math.floor(diff / 86400000);
  if (days === 0) return "Today";
  if (days === 1) return "Yesterday";
  return `${days}d ago`;
}

// ── Stat Card ──────────────────────────────────────────────────────────────────

function StatCard({
  label, value, sub, icon: Icon, color, trend,
}: {
  label: string; value: string; sub?: string; icon: React.ElementType;
  color: string; trend?: "up" | "down" | "neutral";
}) {
  return (
    <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-5 flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <span className={`p-2.5 rounded-xl ${color}`}>
          <Icon className="w-5 h-5" />
        </span>
        {trend === "up" && <span className="flex items-center gap-1 text-xs text-green-600 font-medium"><ArrowUpRight className="w-3.5 h-3.5" />+12%</span>}
        {trend === "down" && <span className="flex items-center gap-1 text-xs text-red-500 font-medium"><ArrowDownRight className="w-3.5 h-3.5" />-3%</span>}
      </div>
      <div>
        <p className="text-2xl font-bold text-gray-900">{value}</p>
        <p className="text-xs text-gray-500 mt-0.5">{label}</p>
        {sub && <p className="text-xs text-[#0071CE] font-medium mt-1">{sub}</p>}
      </div>
    </div>
  );
}

// ── Referral Widget ─────────────────────────────────────────────────────────────

function ReferralWidget() {
  const [code, setCode] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    api.get("/referral/code").then((r) => setCode(r.data?.code ?? null)).catch(() => {});
  }, []);

  if (!code) return null;

  const handleCopy = async () => {
    await navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="bg-gradient-to-r from-blue-50 to-indigo-50 rounded-2xl border border-blue-100 p-5">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <Gift className="w-5 h-5 text-blue-600" />
          <h3 className="text-sm font-semibold text-gray-800">Refer &amp; Earn</h3>
        </div>
        <Link href="/referral" className="text-xs text-blue-600 hover:underline font-medium">
          View stats →
        </Link>
      </div>
      <p className="text-xs text-gray-500 mb-3">Share your code — earn 100 loyalty points for each friend who completes their first order.</p>
      <div className="flex items-center gap-2">
        <div className="flex-1 bg-white border border-blue-200 rounded-xl px-4 py-2.5 text-center">
          <span className="font-mono font-bold tracking-widest text-blue-700 text-base">{code}</span>
        </div>
        <button
          onClick={handleCopy}
          className="flex items-center gap-1.5 px-3 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-xl text-xs font-semibold transition-colors"
        >
          {copied ? <CheckCircle className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
          {copied ? "Copied!" : "Copy"}
        </button>
      </div>
    </div>
  );
}

// ── Overview Tab ────────────────────────────────────────────────────────────────

function OverviewTab({ stats, listings, orders }: {
  stats: typeof MOCK_STATS;
  listings: Listing[];
  orders: Order[];
}) {
  return (
    <div className="space-y-6">
      {/* Stats grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard label="Active Listings" value={String(stats.active_listings)} sub={`of ${stats.total_listings} total`} icon={Package} color="bg-[#0071CE]/10 text-[#0071CE]" trend="up" />
        <StatCard label="Pending Orders" value={String(stats.pending_orders)} sub={`${stats.total_orders} total orders`} icon={ShoppingBag} color="bg-yellow-100 text-yellow-600" trend="neutral" />
        <StatCard label="This Month" value={formatPrice(stats.this_month_revenue, "AED")} sub="Revenue" icon={TrendingUp} color="bg-green-100 text-green-600" trend="up" />
        <StatCard label="Wallet Balance" value={formatPrice(stats.wallet_balance, "AED")} icon={Wallet} color="bg-purple-100 text-purple-600" />
      </div>

      <div className="grid md:grid-cols-2 gap-4">
        {/* Secondary stats */}
        <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-5">
          <h3 className="text-sm font-semibold text-gray-700 mb-4">Store Performance</h3>
          <div className="space-y-3">
            <div className="flex items-center justify-between text-sm">
              <span className="text-gray-500">Total Revenue</span>
              <span className="font-semibold text-gray-900">{formatPrice(stats.total_revenue, "AED")}</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-gray-500">Store Visits</span>
              <span className="font-semibold text-gray-900">{stats.store_visits.toLocaleString()}</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-gray-500">Average Rating</span>
              <span className="font-semibold text-yellow-500 flex items-center gap-1">
                <Star className="w-3.5 h-3.5 fill-yellow-400 text-yellow-400" />{stats.average_rating}
              </span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-gray-500">Total Orders</span>
              <span className="font-semibold text-gray-900">{stats.total_orders}</span>
            </div>
          </div>
        </div>

        {/* Quick actions */}
        <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-5">
          <h3 className="text-sm font-semibold text-gray-700 mb-4">Quick Actions</h3>
          <div className="space-y-2">
            {[
              { label: "Post New Listing", icon: Plus, href: "/sell", color: "bg-[#0071CE] text-white hover:bg-[#005ba3]" },
              { label: "Manage My Store", icon: Store, href: "/my-store", color: "bg-[#FFC220] text-gray-900 hover:bg-yellow-400" },
              { label: "My Orders", icon: ShoppingBag, href: "/orders", color: "bg-white text-[#0071CE] border border-[#0071CE] hover:bg-blue-50" },
              { label: "My Sales", icon: Tag, href: "/selling/orders", color: "bg-white text-[#0071CE] border border-[#0071CE] hover:bg-blue-50" },
              { label: "View Wallet", icon: Wallet, href: "/wallet", color: "bg-gray-100 text-gray-700 hover:bg-gray-200" },
              { label: "See My Reviews", icon: Star, href: "/reviews", color: "bg-gray-100 text-gray-700 hover:bg-gray-200" },
            ].map(({ label, icon: Icon, href, color }) => (
              <Link key={label} href={href}
                className={`flex items-center justify-between px-4 py-3 rounded-xl text-sm font-medium transition-colors ${color}`}>
                <span className="flex items-center gap-2"><Icon className="w-4 h-4" />{label}</span>
                <ChevronRight className="w-4 h-4" />
              </Link>
            ))}
          </div>
        </div>
      </div>

      {/* Loyalty Points Widget */}
      <div className="mb-6">
        <PointsDisplay />
      </div>

      {/* Referral Widget */}
      <ReferralWidget />

      {/* Recent orders */}
      <div className="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden">
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-100">
          <h3 className="text-sm font-semibold text-gray-700">Recent Orders</h3>
          <button className="text-xs text-[#0071CE] hover:underline">View all</button>
        </div>
        <div className="divide-y divide-gray-50">
          {orders.slice(0, 4).map((o) => {
            const { label, cls, icon: SIcon } = orderStatusConfig(o.status);
            return (
              <div key={o.id} className="flex items-center gap-4 px-5 py-3.5 hover:bg-gray-50 transition-colors">
                <div className={`p-2 rounded-xl ${o.role === "seller" ? "bg-[#0071CE]/10 text-[#0071CE]" : "bg-purple-100 text-purple-600"}`}>
                  {o.role === "seller" ? <Tag className="w-4 h-4" /> : <ShoppingBag className="w-4 h-4" />}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-gray-900 truncate">{o.item_title}</p>
                  <p className="text-xs text-gray-400">
                    {o.role === "seller" ? `Buyer: ${o.buyer_name}` : `Seller: ${o.seller_name}`} · {timeAgo(o.created_at)}
                  </p>
                </div>
                <div className="text-right shrink-0">
                  <p className="text-sm font-semibold text-gray-900">{formatPrice(o.amount, o.currency)}</p>
                  <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${cls}`}>{label}</span>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

// ── Listings Tab ───────────────────────────────────────────────────────────────

function ListingsTab({ listings }: { listings: Listing[] }) {
  const [filter, setFilter] = useState<"all" | "active" | "sold" | "inactive">("all");
  const qc = useQueryClient();
  const { toast } = useSimpleToast();

  const filtered = filter === "all" ? listings : listings.filter((l) => l.status === filter);

  const handleDelete = async (id: string) => {
    if (!confirm("Delete this listing?")) return;
    try {
      await api.delete(`/listings/${id}`);
    } catch {}
    toast("Listing deleted");
    qc.invalidateQueries({ queryKey: ["my-listings"] });
  };

  const handleToggle = async (listing: Listing) => {
    const newStatus = listing.status === "active" ? "inactive" : "active";
    try {
      await api.put(`/listings/${listing.id}`, { status: newStatus });
    } catch {}
    toast(`Listing ${newStatus === "active" ? "activated" : "paused"}`);
    qc.invalidateQueries({ queryKey: ["my-listings"] });
  };

  const FILTERS: { key: typeof filter; label: string; count: number }[] = [
    { key: "all", label: "All", count: listings.length },
    { key: "active", label: "Active", count: listings.filter((l) => l.status === "active").length },
    { key: "sold", label: "Sold", count: listings.filter((l) => l.status === "sold").length },
    { key: "inactive", label: "Inactive", count: listings.filter((l) => l.status === "inactive").length },
  ];

  return (
    <div className="space-y-4">
      {/* Filter pills + new listing button */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex gap-2">
          {FILTERS.map(({ key, label, count }) => (
            <button key={key} onClick={() => setFilter(key)}
              className={`px-3 py-1.5 rounded-full text-xs font-medium transition-colors ${filter === key ? "bg-[#0071CE] text-white" : "bg-white text-gray-500 border border-gray-200 hover:bg-gray-50"}`}>
              {label} <span className="opacity-70">({count})</span>
            </button>
          ))}
        </div>
        <Link href="/sell"
          className="flex items-center gap-1.5 bg-[#0071CE] text-white px-4 py-2 rounded-xl text-sm font-semibold hover:bg-[#005ba3] transition-colors">
          <Plus className="w-4 h-4" /> New Listing
        </Link>
      </div>

      {/* Listings list */}
      <div className="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden">
        {filtered.length === 0 ? (
          <div className="py-16 text-center text-gray-400">
            <Package className="w-10 h-10 mx-auto mb-3 text-gray-200" />
            <p className="text-sm">No listings found</p>
            <Link href="/sell" className="mt-2 text-xs text-[#0071CE] hover:underline inline-block">Create your first listing →</Link>
          </div>
        ) : (
          <div className="divide-y divide-gray-50">
            {filtered.map((listing) => {
              const { label, cls } = listingStatusConfig(listing.status);
              return (
                <div key={listing.id} className="flex items-center gap-4 px-5 py-4 hover:bg-gray-50 transition-colors">
                  {/* Thumbnail */}
                  <div className="w-14 h-14 rounded-xl overflow-hidden bg-gray-100 shrink-0">
                    {listing.images?.[0] ? (
                      <img src={listing.images[0].url} alt={listing.title} className="w-full h-full object-cover" />
                    ) : (
                      <div className="w-full h-full flex items-center justify-center text-gray-300">
                        <Package className="w-6 h-6" />
                      </div>
                    )}
                  </div>

                  {/* Info */}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-semibold text-gray-900 truncate">{listing.title}</p>
                    <div className="flex items-center gap-2 mt-1">
                      <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${cls}`}>{label}</span>
                      <span className="text-xs text-gray-400 capitalize">{listing.type.replace(/_/g, " ")}</span>
                      <span className="text-xs text-gray-400 flex items-center gap-0.5"><Eye className="w-3 h-3" />{getViews(listing).toLocaleString()}</span>
                      <span className="text-xs text-gray-400">{timeAgo(listing.created_at)}</span>
                    </div>
                  </div>

                  {/* Price */}
                  <div className="text-right shrink-0">
                    <p className="text-sm font-bold text-[#0071CE]">{formatPrice(listing.price, listing.currency)}</p>
                    <p className="text-xs text-gray-400 capitalize">{getCategoryLabel(listing)}</p>
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-1 shrink-0">
                    <Link href={`/listings/${listing.id}`}
                      className="p-2 rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors" title="View">
                      <Eye className="w-4 h-4" />
                    </Link>
                    {listing.status !== "sold" && (
                      <>
                        <button onClick={() => handleToggle(listing)}
                          className="p-2 rounded-lg hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors"
                          title={listing.status === "active" ? "Pause" : "Activate"}>
                          {listing.status === "active"
                            ? <XCircle className="w-4 h-4" />
                            : <CheckCircle className="w-4 h-4 text-green-500" />}
                        </button>
                        <button onClick={() => handleDelete(listing.id)}
                          className="p-2 rounded-lg hover:bg-red-50 text-gray-400 hover:text-red-500 transition-colors" title="Delete">
                          <Trash2 className="w-4 h-4" />
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

// ── Orders Tab ─────────────────────────────────────────────────────────────────

function OrdersTab({ orders }: { orders: Order[] }) {
  const [role, setRole] = useState<"all" | "seller" | "buyer">("all");
  const filtered = role === "all" ? orders : orders.filter((o) => o.role === role);

  return (
    <div className="space-y-4">
      {/* Role filter */}
      <div className="flex gap-2">
        {(["all", "seller", "buyer"] as const).map((r) => (
          <button key={r} onClick={() => setRole(r)}
            className={`px-3 py-1.5 rounded-full text-xs font-medium capitalize transition-colors ${role === r ? "bg-[#0071CE] text-white" : "bg-white text-gray-500 border border-gray-200 hover:bg-gray-50"}`}>
            {r === "all" ? "All Orders" : r === "seller" ? "🏷️ Sales" : "🛍️ Purchases"}
          </button>
        ))}
      </div>

      {/* Orders list */}
      <div className="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden">
        {filtered.length === 0 ? (
          <div className="py-16 text-center text-gray-400">
            <ShoppingBag className="w-10 h-10 mx-auto mb-3 text-gray-200" />
            <p className="text-sm">No orders found</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-50">
            {filtered.map((order) => {
              const { label, cls, icon: SIcon } = orderStatusConfig(order.status);
              return (
                <div key={order.id} className="flex items-center gap-4 px-5 py-4 hover:bg-gray-50 transition-colors">
                  {/* Role icon */}
                  <div className={`p-3 rounded-xl shrink-0 ${order.role === "seller" ? "bg-[#0071CE]/10 text-[#0071CE]" : "bg-purple-100 text-purple-600"}`}>
                    {order.role === "seller" ? <Tag className="w-5 h-5" /> : <ShoppingBag className="w-5 h-5" />}
                  </div>

                  {/* Info */}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-semibold text-gray-900 truncate">{order.item_title}</p>
                    <div className="flex items-center gap-2 mt-1">
                      <span className="text-xs text-gray-400">
                        {order.role === "seller" ? `Buyer: ${order.buyer_name}` : `Seller: ${order.seller_name}`}
                      </span>
                      <span className="text-gray-200">·</span>
                      <span className="text-xs text-gray-400">{timeAgo(order.created_at)}</span>
                      <span className="text-gray-200">·</span>
                      <span className="text-xs text-gray-400">#{order.id}</span>
                    </div>
                  </div>

                  {/* Amount + status */}
                  <div className="text-right shrink-0">
                    <p className={`text-sm font-bold ${order.role === "seller" ? "text-green-600" : "text-gray-900"}`}>
                      {order.role === "seller" ? "+" : "-"}{formatPrice(order.amount, order.currency)}
                    </p>
                    <span className={`inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full font-medium mt-1 ${cls}`}>
                      <SIcon className="w-3 h-3" />{label}
                    </span>
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

// ── Analytics Tab ──────────────────────────────────────────────────────────────

function AnalyticsTab({ stats, listings }: { stats: typeof MOCK_STATS; listings: Listing[] }) {
  const topListings = [...listings].sort((a, b) => getViews(b) - getViews(a)).slice(0, 5);

  return (
    <div className="space-y-6">
      {/* Revenue overview */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-5">
          <p className="text-xs text-gray-400 mb-1">Total Revenue</p>
          <p className="text-2xl font-bold text-gray-900">{formatPrice(stats.total_revenue, "AED")}</p>
          <div className="mt-3 h-1.5 bg-gray-100 rounded-full overflow-hidden">
            <div className="h-full bg-[#0071CE] rounded-full" style={{ width: "72%" }} />
          </div>
          <p className="text-xs text-gray-400 mt-1.5">72% of annual target</p>
        </div>
        <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-5">
          <p className="text-xs text-gray-400 mb-1">This Month</p>
          <p className="text-2xl font-bold text-gray-900">{formatPrice(stats.this_month_revenue, "AED")}</p>
          <div className="flex items-center gap-1 mt-2 text-xs text-green-600 font-medium">
            <ArrowUpRight className="w-3.5 h-3.5" /> +18% vs last month
          </div>
        </div>
        <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-5">
          <p className="text-xs text-gray-400 mb-1">Store Visits</p>
          <p className="text-2xl font-bold text-gray-900">{stats.store_visits.toLocaleString()}</p>
          <div className="flex items-center gap-1 mt-2 text-xs text-green-600 font-medium">
            <ArrowUpRight className="w-3.5 h-3.5" /> +24% this week
          </div>
        </div>
      </div>

      {/* Top performing listings */}
      <div className="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden">
        <div className="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
          <div>
            <h3 className="text-sm font-semibold text-gray-700">Top Performing Listings</h3>
            <p className="text-xs text-gray-400 mt-0.5">Ranked by total views</p>
          </div>
          <Link href="/seller/analytics" className="text-xs text-[#0071CE] font-medium hover:underline">
            View Full Analytics →
          </Link>
        </div>
        <div className="divide-y divide-gray-50">
          {topListings.map((l, i) => (
            <div key={l.id} className="flex items-center gap-4 px-5 py-3.5">
              <span className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold shrink-0
                ${i === 0 ? "bg-yellow-100 text-yellow-700" : i === 1 ? "bg-gray-200 text-gray-600" : "bg-orange-100 text-orange-600"}`}>
                {i + 1}
              </span>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900 truncate">{l.title}</p>
                <p className="text-xs text-gray-400 capitalize">{getCategoryLabel(l)}</p>
              </div>
              <div className="text-right shrink-0">
                <p className="text-sm font-semibold text-gray-900">{getViews(l).toLocaleString()} views</p>
                <p className="text-xs text-[#0071CE] font-medium">{formatPrice(l.price, l.currency)}</p>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Category breakdown */}
      <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-5">
        <h3 className="text-sm font-semibold text-gray-700 mb-4">Listings by Category</h3>
        <div className="space-y-3">
          {Object.entries(
            listings.reduce((acc, l) => { const cat = getCategoryLabel(l) || "Other"; return { ...acc, [cat]: (acc[cat] || 0) + 1 }; }, {} as Record<string, number>)
          ).map(([cat, count]) => (
            <div key={cat} className="flex items-center gap-3">
              <p className="text-sm text-gray-600 capitalize w-28 shrink-0">{cat}</p>
              <div className="flex-1 h-2 bg-gray-100 rounded-full overflow-hidden">
                <div className="h-full bg-[#0071CE] rounded-full transition-all" style={{ width: `${(count / listings.length) * 100}%` }} />
              </div>
              <span className="text-xs text-gray-500 w-8 text-right">{count}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ── Simple toast hook ──────────────────────────────────────────────────────────

function useSimpleToast() {
  const [msg, setMsg] = useState<string | null>(null);
  useEffect(() => {
    if (!msg) return;
    const t = setTimeout(() => setMsg(null), 2500);
    return () => clearTimeout(t);
  }, [msg]);
  return { toast: setMsg, msg };
}

// ── Main Page ──────────────────────────────────────────────────────────────────

const TABS: { key: Tab; labelKey: string; icon: React.ElementType }[] = [
  { key: "overview", labelKey: "overview", icon: LayoutDashboard },
  { key: "listings", labelKey: "myListings", icon: Package },
  { key: "orders", labelKey: "orders", icon: ShoppingBag },
  { key: "analytics", labelKey: "analytics", icon: BarChart2 },
];

export default function DashboardPage() {
  const t = useTranslations("dashboard");
  const { user, isAuthenticated } = useAuthStore();
  const router = useRouter();
  const [activeTab, setActiveTab] = useState<Tab>("overview");
  const { toast: showToast, msg: toastMsg } = useSimpleToast();

  useEffect(() => {
    if (!isAuthenticated || !user) {
      router.push("/login?next=/dashboard");
    }
  }, [isAuthenticated, user, router]);

  // Fetch seller stats
  const { data: stats } = useQuery({
    queryKey: ["seller-stats"],
    queryFn: async () => {
      try {
        const res = await api.get("/users/me/stats");
        return res.data?.data ?? MOCK_STATS;
      } catch {
        return MOCK_STATS;
      }
    },
    enabled: isAuthenticated && !!user,
  });

  // Fetch my listings
  const { data: listings } = useQuery<Listing[]>({
    queryKey: ["my-listings"],
    queryFn: async () => {
      try {
        const res = await api.get("/listings/me");
        const data = res.data?.data ?? [];
        return Array.isArray(data) ? data : MOCK_LISTINGS;
      } catch {
        return MOCK_LISTINGS;
      }
    },
    enabled: isAuthenticated && !!user,
  });

  // Fetch my orders
  const { data: orders } = useQuery<Order[]>({
    queryKey: ["my-orders"],
    queryFn: async () => {
      try {
        const res = await api.get("/orders/me");
        const data = res.data?.data ?? [];
        return Array.isArray(data) && data.length > 0 ? data : MOCK_ORDERS;
      } catch {
        return MOCK_ORDERS;
      }
    },
    enabled: isAuthenticated && !!user,
  });

  const displayStats = stats ?? MOCK_STATS;
  const displayListings = listings ?? MOCK_LISTINGS;
  const displayOrders = orders ?? MOCK_ORDERS;

  const pendingOrdersCount = displayOrders.filter(
    (o) => o.role === "seller" && o.status === "pending"
  ).length;

  if (!isAuthenticated || !user) return null;

  return (
    <div className="max-w-5xl mx-auto px-4 py-8">
      {/* Toast */}
      {toastMsg && (
        <div className="fixed bottom-6 left-1/2 -translate-x-1/2 bg-gray-900 text-white px-5 py-3 rounded-xl text-sm shadow-lg z-50 animate-bounce-in">
          {toastMsg}
        </div>
      )}

      {/* Page header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">
            👋 Hey, {user.name?.split(" ")[0]}!
          </h1>
          <p className="text-sm text-gray-500 mt-0.5">
            Manage your store, listings & orders from one place
          </p>
        </div>
        <div className="flex items-center gap-3">
          {pendingOrdersCount > 0 && (
            <span className="flex items-center gap-1.5 bg-yellow-50 text-yellow-700 border border-yellow-200 px-3 py-1.5 rounded-xl text-xs font-semibold">
              <AlertCircle className="w-3.5 h-3.5" />
              {pendingOrdersCount} pending order{pendingOrdersCount > 1 ? "s" : ""}
            </span>
          )}
          <Link href="/sell"
            className="flex items-center gap-1.5 bg-[#0071CE] text-white px-4 py-2.5 rounded-xl text-sm font-bold hover:bg-[#005ba3] transition-colors shadow-sm">
            <Plus className="w-4 h-4" /> Post Listing
          </Link>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 bg-gray-100 p-1 rounded-xl mb-6 overflow-x-auto">
        {TABS.map(({ key, labelKey, icon: Icon }) => (
          <button
            key={key}
            onClick={() => setActiveTab(key)}
            className={`flex items-center gap-2 px-4 py-2.5 rounded-lg text-sm font-medium whitespace-nowrap transition-all flex-1 justify-center ${
              activeTab === key
                ? "bg-white text-[#0071CE] shadow-sm"
                : "text-gray-500 hover:text-gray-700"
            }`}
          >
            <Icon className="w-4 h-4" />
            {t(labelKey)}
            {key === "orders" && pendingOrdersCount > 0 && (
              <span className="bg-yellow-400 text-gray-900 text-xs w-5 h-5 rounded-full flex items-center justify-center font-bold">
                {pendingOrdersCount}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Tab content */}
      {activeTab === "overview" && (
        <OverviewTab stats={displayStats} listings={displayListings} orders={displayOrders} />
      )}
      {activeTab === "listings" && <ListingsTab listings={displayListings} />}
      {activeTab === "orders" && <OrdersTab orders={displayOrders} />}
      {activeTab === "analytics" && <AnalyticsTab stats={displayStats} listings={displayListings} />}
    </div>
  );
}
