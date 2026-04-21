"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { formatPrice } from "@/lib/utils";
import {
  ShoppingCart, Package, Heart, Wallet, TrendingDown,
  ArrowUpRight, Clock, CheckCircle, XCircle, Truck,
  AlertCircle, Bell, Search, Star, MapPin, Tag,
  ChevronRight, Gift, Gavel, RotateCcw, Plus,
} from "lucide-react";

// ── Types ─────────────────────────────────────────────────────────────────────

interface BuyerStats {
  total_orders: number;
  pending_orders: number;
  completed_orders: number;
  total_spent: number;
  this_month_spent: number;
  wallet_balance: number;
  watchlist_count: number;
  active_bids: number;
  loyalty_points: number;
}

interface Order {
  id: string;
  item_title: string;
  seller_name?: string;
  amount: number;
  currency: string;
  status: "pending" | "confirmed" | "shipped" | "delivered" | "cancelled" | "disputed";
  created_at: string;
  tracking_number?: string;
  image?: string;
}

interface WatchlistItem {
  id: string;
  title: string;
  price: number;
  currency: string;
  type: string;
  ends_at?: string;
  image?: string;
  price_drop?: boolean;
}

// ── Mock data ─────────────────────────────────────────────────────────────────

const MOCK_STATS: BuyerStats = {
  total_orders: 18,
  pending_orders: 2,
  completed_orders: 14,
  total_spent: 23400,
  this_month_spent: 4200,
  wallet_balance: 3750,
  watchlist_count: 7,
  active_bids: 3,
  loyalty_points: 1280,
};

const MOCK_ORDERS: Order[] = [
  { id: "ord-001", item_title: "Sony WH-1000XM5 Headphones", seller_name: "TechHub UAE", amount: 1299, currency: "AED", status: "shipped", created_at: new Date(Date.now() - 2 * 86400000).toISOString(), tracking_number: "DHL9234567", image: "https://picsum.photos/seed/headphones/56/56" },
  { id: "ord-002", item_title: "Nike Air Max 270 — Size 43", seller_name: "SportsZone", amount: 620, currency: "AED", status: "pending", created_at: new Date(Date.now() - 86400000).toISOString(), image: "https://picsum.photos/seed/nike1/56/56" },
  { id: "ord-003", item_title: "Dyson V15 Detect Vacuum", seller_name: "HomeAppliances", amount: 2800, currency: "AED", status: "delivered", created_at: new Date(Date.now() - 8 * 86400000).toISOString(), image: "https://picsum.photos/seed/dyson/56/56" },
  { id: "ord-004", item_title: "Apple Watch Series 9 GPS 45mm", seller_name: "iStore", amount: 1890, currency: "AED", status: "confirmed", created_at: new Date(Date.now() - 3 * 86400000).toISOString(), image: "https://picsum.photos/seed/applewatch/56/56" },
  { id: "ord-005", item_title: "LEGO Technic Set 42143", seller_name: "ToyWorld", amount: 480, currency: "AED", status: "cancelled", created_at: new Date(Date.now() - 15 * 86400000).toISOString(), image: "https://picsum.photos/seed/lego/56/56" },
];

const MOCK_WATCHLIST: WatchlistItem[] = [
  { id: "wl-001", title: "MacBook Pro M3 14\" — Space Black", price: 8200, currency: "AED", type: "buy_now", image: "https://picsum.photos/seed/mbp/56/56", price_drop: true },
  { id: "wl-002", title: "Canon EOS R6 Mark II Body", price: 11500, currency: "AED", type: "standard_auction", ends_at: new Date(Date.now() + 5 * 3600000).toISOString(), image: "https://picsum.photos/seed/canon/56/56" },
  { id: "wl-003", title: "PlayStation 5 Slim + Extra Controller", price: 2300, currency: "AED", type: "buy_now", image: "https://picsum.photos/seed/ps5s/56/56" },
  { id: "wl-004", title: "Vintage Rolex Datejust 36mm", price: 24000, currency: "AED", type: "sealed_auction", ends_at: new Date(Date.now() + 2 * 86400000).toISOString(), image: "https://picsum.photos/seed/rolex2/56/56" },
];

// ── Helpers ───────────────────────────────────────────────────────────────────

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const days = Math.floor(diff / 86400000);
  const hrs = Math.floor(diff / 3600000);
  if (hrs < 24) return `${hrs}h ago`;
  if (days === 1) return "Yesterday";
  return `${days}d ago`;
}

function timeLeft(dateStr: string): string {
  const diff = new Date(dateStr).getTime() - Date.now();
  if (diff <= 0) return "Ended";
  const h = Math.floor(diff / 3600000);
  const d = Math.floor(diff / 86400000);
  if (d >= 1) return `${d}d left`;
  return `${h}h left`;
}

const ORDER_STATUS: Record<string, { label: string; cls: string; icon: React.ElementType }> = {
  pending:   { label: "Pending",   cls: "bg-amber-50 text-amber-700 border border-amber-200",   icon: Clock },
  confirmed: { label: "Confirmed", cls: "bg-violet-50 text-violet-700 border border-violet-200", icon: CheckCircle },
  shipped:   { label: "Shipped",   cls: "bg-blue-50 text-blue-700 border border-blue-200",       icon: Truck },
  delivered: { label: "Delivered", cls: "bg-emerald-50 text-emerald-700 border border-emerald-200", icon: CheckCircle },
  cancelled: { label: "Cancelled", cls: "bg-red-50 text-red-600 border border-red-200",          icon: XCircle },
  disputed:  { label: "Disputed",  cls: "bg-orange-50 text-orange-700 border border-orange-200", icon: AlertCircle },
};

// ── KPI Card ──────────────────────────────────────────────────────────────────

function KPI({ label, value, sub, icon: Icon, accent, href }: {
  label: string; value: string; sub?: string;
  icon: React.ElementType; accent: string; href?: string;
}) {
  const inner = (
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
  return href ? <Link href={href}>{inner}</Link> : inner;
}

// ── Main Page ─────────────────────────────────────────────────────────────────

export default function BuyerDashboardPage() {
  const { user, isAuthenticated } = useAuthStore();
  const [orderFilter, setOrderFilter] = useState<"all" | "active" | "delivered" | "cancelled">("all");

  const { data: stats = MOCK_STATS } = useQuery<BuyerStats>({
    queryKey: ["buyer-stats"],
    queryFn: async () => {
      try { return (await api.get("/users/me/stats")).data?.data ?? MOCK_STATS; }
      catch { return MOCK_STATS; }
    },
    enabled: isAuthenticated,
  });

  const { data: orders = MOCK_ORDERS } = useQuery<Order[]>({
    queryKey: ["buyer-orders"],
    queryFn: async () => {
      try {
        const d = (await api.get("/orders")).data?.data ?? [];
        return Array.isArray(d) && d.length ? d : MOCK_ORDERS;
      } catch { return MOCK_ORDERS; }
    },
    enabled: isAuthenticated,
  });

  const { data: watchlist = MOCK_WATCHLIST } = useQuery<WatchlistItem[]>({
    queryKey: ["buyer-watchlist"],
    queryFn: async () => {
      try {
        const d = (await api.get("/watchlist")).data?.data ?? [];
        return Array.isArray(d) && d.length ? d : MOCK_WATCHLIST;
      } catch { return MOCK_WATCHLIST; }
    },
    enabled: isAuthenticated,
  });

  const shippedOrders = orders.filter((o) => o.status === "shipped");
  const pendingOrders = orders.filter((o) => o.status === "pending" || o.status === "confirmed");

  const filteredOrders = orders.filter((o) => {
    if (orderFilter === "active") return ["pending", "confirmed", "shipped"].includes(o.status);
    if (orderFilter === "delivered") return o.status === "delivered";
    if (orderFilter === "cancelled") return o.status === "cancelled";
    return true;
  });

  return (
    <div className="space-y-6">

      {/* Welcome */}
      <p className="text-sm text-gray-500">
        Welcome back, <span className="font-semibold text-gray-700">{user?.name?.split(" ")[0] ?? "Buyer"}</span>. Here&apos;s your shopping activity.
      </p>

      {/* Alert: in-transit orders */}
      {shippedOrders.length > 0 && (
          <div className="flex items-start gap-3 bg-blue-50 border border-blue-200 text-blue-800 rounded-xl px-4 py-3">
            <Truck className="w-4 h-4 mt-0.5 flex-shrink-0" />
            <div className="flex-1 min-w-0">
              <p className="text-sm font-semibold">
                {shippedOrders.length} order{shippedOrders.length > 1 ? "s are" : " is"} on the way
              </p>
              <p className="text-xs text-blue-600 mt-0.5">
                {shippedOrders[0].item_title}
                {shippedOrders[0].tracking_number && ` · Tracking: ${shippedOrders[0].tracking_number}`}
              </p>
            </div>
            <Link href="/buyer/orders" className="text-xs font-medium underline flex-shrink-0">Track</Link>
          </div>
      )}

      {/* Alert: pending confirmation */}
      {pendingOrders.length > 0 && (
          <div className="flex items-start gap-3 bg-amber-50 border border-amber-200 text-amber-800 rounded-xl px-4 py-3">
            <Bell className="w-4 h-4 mt-0.5 flex-shrink-0" />
            <p className="text-sm font-semibold flex-1">
              {pendingOrders.length} order{pendingOrders.length > 1 ? "s" : ""} awaiting seller confirmation
            </p>
            <Link href="/buyer/orders" className="text-xs font-medium underline flex-shrink-0">View</Link>
          </div>
      )}

      {/* KPI Strip */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
        <KPI label="Total Orders" value={String(stats.total_orders)} sub={`${stats.completed_orders} completed`} icon={Package} accent="bg-indigo-500" href="/buyer/orders" />
        <KPI label="Active Orders" value={String(stats.pending_orders)} sub="In progress" icon={ShoppingCart} accent="bg-amber-400" href="/buyer/orders" />
        <KPI label="Total Spent" value={formatPrice(stats.total_spent)} sub={`AED ${stats.this_month_spent.toLocaleString()} this month`} icon={TrendingDown} accent="bg-rose-400" />
        <KPI label="Wallet Balance" value={formatPrice(stats.wallet_balance)} sub="Available funds" icon={Wallet} accent="bg-emerald-500" href="/wallet" />
        <KPI label="Active Bids" value={String(stats.active_bids)} sub={`${stats.watchlist_count} in watchlist`} icon={Gavel} accent="bg-purple-500" href="/auctions" />
      </div>

      {/* Main Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">

        {/* Orders — 2/3 width */}
        <div className="lg:col-span-2 bg-white rounded-2xl border border-gray-100 overflow-hidden">
          <div className="flex items-center justify-between px-5 py-4 border-b border-gray-50">
            <h2 className="font-semibold text-gray-800">My Orders</h2>
              <div className="flex items-center gap-1.5">
                {(["all", "active", "delivered", "cancelled"] as const).map((f) => (
                  <button
                    key={f}
                    onClick={() => setOrderFilter(f)}
                    className={`text-xs px-2.5 py-1 rounded-full font-medium capitalize transition ${
                      orderFilter === f ? "bg-indigo-600 text-white" : "text-gray-400 hover:text-gray-700 hover:bg-gray-100"
                    }`}
                  >
                    {f}
                  </button>
                ))}
              </div>
            </div>

            <div className="divide-y divide-gray-50">
              {filteredOrders.length === 0 && (
                <div className="py-12 text-center text-gray-400 text-sm">No orders found.</div>
              )}
              {filteredOrders.map((order) => {
                const sc = ORDER_STATUS[order.status] ?? ORDER_STATUS.pending;
                const StatusIcon = sc.icon;
                return (
                  <div key={order.id} className="flex items-center gap-3 px-5 py-3.5 hover:bg-gray-50 transition group">
                    {order.image ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={order.image} alt="" className="w-10 h-10 rounded-lg object-cover flex-shrink-0 border border-gray-100" />
                    ) : (
                      <div className="w-10 h-10 rounded-lg bg-gray-100 flex-shrink-0 flex items-center justify-center">
                        <Package className="w-4 h-4 text-gray-300" />
                      </div>
                    )}
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-gray-800 truncate">{order.item_title}</p>
                      <div className="flex items-center gap-2 mt-0.5">
                        {order.seller_name && (
                          <span className="text-xs text-gray-400 flex items-center gap-0.5">
                            <MapPin className="w-3 h-3" />{order.seller_name}
                          </span>
                        )}
                        <span className="text-xs text-gray-300">{timeAgo(order.created_at)}</span>
                      </div>
                    </div>
                    <div className="flex items-center gap-3 flex-shrink-0">
                      <span className={`inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full font-medium ${sc.cls}`}>
                        <StatusIcon className="w-3 h-3" />{sc.label}
                      </span>
                      <span className="text-sm font-semibold text-gray-700 tabular-nums">
                        {formatPrice(order.amount)}
                      </span>
                      <Link href={`/orders/${order.id}`} className="opacity-0 group-hover:opacity-100 transition">
                        <ChevronRight className="w-4 h-4 text-gray-400" />
                      </Link>
                    </div>
                  </div>
                );
              })}
            </div>

          <div className="px-5 py-3 border-t border-gray-50">
            <Link href="/buyer/orders" className="text-xs text-indigo-600 hover:underline font-medium flex items-center gap-1">
              View all orders <ArrowUpRight className="w-3 h-3" />
            </Link>
          </div>
        </div>

        {/* Right column */}
        <div className="space-y-5">

            {/* Loyalty Points */}
            <div className="bg-gradient-to-br from-indigo-600 to-purple-600 rounded-2xl p-5 text-white relative overflow-hidden">
              <div className="absolute -top-6 -right-6 w-24 h-24 rounded-full bg-white/10" />
              <div className="absolute -bottom-4 -left-4 w-16 h-16 rounded-full bg-white/10" />
              <div className="relative">
                <div className="flex items-center gap-2 mb-3">
                  <Gift className="w-4 h-4 text-indigo-200" />
                  <span className="text-xs font-semibold text-indigo-200 uppercase tracking-wide">Loyalty Points</span>
                </div>
                <p className="text-3xl font-bold tabular-nums">{stats.loyalty_points.toLocaleString()}</p>
                <p className="text-xs text-indigo-200 mt-1">≈ AED {(stats.loyalty_points * 0.05).toFixed(0)} value</p>
                <Link href="/wallet" className="mt-3 inline-flex items-center gap-1 bg-white/20 hover:bg-white/30 text-white text-xs font-medium px-3 py-1.5 rounded-lg transition">
                  Redeem <ArrowUpRight className="w-3 h-3" />
                </Link>
              </div>
            </div>

            {/* Quick Actions */}
            <div className="bg-white rounded-2xl border border-gray-100 p-5">
              <h2 className="text-sm font-semibold text-gray-800 mb-3">Quick Actions</h2>
              <div className="grid grid-cols-2 gap-2">
                {[
                  { href: "/listings",         icon: Search,    label: "Browse",       cls: "bg-indigo-50 text-indigo-700" },
                  { href: "/auctions",         icon: Gavel,     label: "Auctions",     cls: "bg-purple-50 text-purple-700" },
                  { href: "/reverse-auctions/new", icon: Plus,  label: "Request Item", cls: "bg-emerald-50 text-emerald-700" },
                  { href: "/buyer/orders",      icon: RotateCcw, label: "My Orders",    cls: "bg-amber-50 text-amber-700" },
                  { href: "/wallet",           icon: Wallet,    label: "Wallet",       cls: "bg-rose-50 text-rose-700" },
                  { href: "/profile",          icon: Star,      label: "Reviews",      cls: "bg-sky-50 text-sky-700" },
                ].map(({ href, icon: Icon, label, cls }) => (
                  <Link key={href} href={href} className={`flex flex-col items-center gap-1.5 py-3 rounded-xl text-xs font-medium transition hover:opacity-80 ${cls}`}>
                    <Icon className="w-4 h-4" />{label}
                  </Link>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* Watchlist */}
        {watchlist.length > 0 && (
          <div className="bg-white rounded-2xl border border-gray-100 overflow-hidden">
            <div className="flex items-center justify-between px-5 py-4 border-b border-gray-50">
              <div className="flex items-center gap-2">
                <Heart className="w-4 h-4 text-rose-500" />
                <h2 className="font-semibold text-gray-800">Watchlist</h2>
                <span className="text-xs bg-rose-100 text-rose-600 font-medium px-1.5 py-0.5 rounded-full">{watchlist.length}</span>
              </div>
              <Link href="/buyer/watchlist" className="text-xs text-indigo-600 hover:underline font-medium">View all</Link>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-0 divide-y sm:divide-y-0 sm:divide-x divide-gray-50">
              {watchlist.map((item) => (
                <Link key={item.id} href={`/listings/${item.id}`} className="flex items-center gap-3 px-4 py-3.5 hover:bg-gray-50 transition group">
                  {item.image ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={item.image} alt="" className="w-10 h-10 rounded-lg object-cover border border-gray-100 flex-shrink-0" />
                  ) : (
                    <div className="w-10 h-10 rounded-lg bg-gray-100 flex-shrink-0" />
                  )}
                  <div className="min-w-0 flex-1">
                    <p className="text-xs font-medium text-gray-800 truncate leading-snug">{item.title}</p>
                    <div className="flex items-center gap-1.5 mt-0.5 flex-wrap">
                      <span className="text-xs font-bold text-gray-900 tabular-nums">{formatPrice(item.price)}</span>
                      {item.price_drop && (
                        <span className="text-[10px] bg-rose-100 text-rose-600 px-1 py-0.5 rounded font-medium flex items-center gap-0.5">
                          <Tag className="w-2.5 h-2.5" />Drop
                        </span>
                      )}
                      {item.ends_at && (
                        <span className="text-[10px] text-amber-600 font-medium">{timeLeft(item.ends_at)}</span>
                      )}
                    </div>
                  </div>
                </Link>
              ))}
            </div>
        </div>
      )}

    </div>
  );
}
