"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { formatPrice } from "@/lib/utils";
import {
  ShoppingBag, Clock, CheckCircle, Truck, XCircle,
  ChevronRight, Search, Package,
} from "lucide-react";
import { formatDistanceToNow, parseISO } from "date-fns";

interface OrderItem { id: string; title: string; quantity: number; }
interface Order {
  id: string;
  status: string;
  total: number;
  currency: string;
  buyer_id: string;
  buyer_name?: string;
  created_at: string;
  tracking_number?: string;
  carrier?: string;
  items?: OrderItem[];
}

const MOCK_ORDERS: Order[] = [
  { id: "ord-001", status: "pending",   total: 4200,  currency: "AED", buyer_id: "u1", buyer_name: "Ali Hassan",       created_at: new Date(Date.now() - 86400000).toISOString(),     items: [{ id: "i1", title: "iPhone 15 Pro Max", quantity: 1 }] },
  { id: "ord-002", status: "delivered", total: 32000, currency: "AED", buyer_id: "u2", buyer_name: "Sarah Al-Mansoori", created_at: new Date(Date.now() - 5 * 86400000).toISOString(), items: [{ id: "i2", title: "Rolex Submariner", quantity: 1 }] },
  { id: "ord-003", status: "confirmed", total: 2100,  currency: "AED", buyer_id: "u3", buyer_name: "Mohammed Khalid",  created_at: new Date(Date.now() - 2 * 86400000).toISOString(), items: [{ id: "i3", title: "PS5 Console + 3 Games", quantity: 1 }] },
  { id: "ord-004", status: "shipped",   total: 1200,  currency: "AED", buyer_id: "u4", buyer_name: "Layla Mansoor",    created_at: new Date(Date.now() - 3 * 86400000).toISOString(), items: [{ id: "i4", title: "AirPods Pro Max", quantity: 1 }], tracking_number: "DHL123456", carrier: "DHL" },
  { id: "ord-005", status: "cancelled", total: 850,   currency: "AED", buyer_id: "u5", buyer_name: "Omar Farouq",      created_at: new Date(Date.now() - 8 * 86400000).toISOString(), items: [{ id: "i5", title: "Keyboard MX Keys", quantity: 1 }] },
];

const STATUS_CONFIG: Record<string, { label: string; cls: string; icon: React.ElementType }> = {
  pending:   { label: "Pending",   cls: "bg-amber-50 text-amber-700 border-amber-200",    icon: Clock },
  confirmed: { label: "Confirmed", cls: "bg-violet-50 text-violet-700 border-violet-200", icon: CheckCircle },
  processing:{ label: "Processing",cls: "bg-blue-50 text-blue-700 border-blue-200",       icon: Package },
  shipped:   { label: "Shipped",   cls: "bg-sky-50 text-sky-700 border-sky-200",          icon: Truck },
  delivered: { label: "Delivered", cls: "bg-emerald-50 text-emerald-700 border-emerald-200", icon: CheckCircle },
  cancelled: { label: "Cancelled", cls: "bg-red-50 text-red-600 border-red-200",          icon: XCircle },
};

type FilterStatus = "all" | "pending" | "confirmed" | "shipped" | "delivered" | "cancelled";

function timeAgo(d: string) {
  try { return formatDistanceToNow(parseISO(d), { addSuffix: true }); }
  catch { return d; }
}

export default function SellerOrdersPage() {
  const qc = useQueryClient();
  const { isAuthenticated } = useAuthStore();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [filter, setFilter] = useState<FilterStatus>("all");
  const [search, setSearch] = useState("");
  const [tracking, setTracking] = useState("");
  const [carrier, setCarrier] = useState("");
  const [notice, setNotice] = useState("");

  const { data: orders = MOCK_ORDERS, isLoading } = useQuery<Order[]>({
    queryKey: ["seller-orders-full"],
    queryFn: async () => {
      try {
        const d = (await api.get("/orders/selling?page=1&limit=100")).data?.data ?? [];
        return Array.isArray(d) && d.length ? d : MOCK_ORDERS;
      } catch { return MOCK_ORDERS; }
    },
    enabled: isAuthenticated,
  });

  const confirmMutation = useMutation({
    mutationFn: (id: string) => api.patch(`/orders/${id}/confirm`, {}),
    onSuccess: () => {
      setNotice("Order confirmed successfully.");
      qc.invalidateQueries({ queryKey: ["seller-orders-full"] });
    },
    onError: () => setNotice("Failed to confirm order."),
  });

  const shipMutation = useMutation({
    mutationFn: ({ id, tracking_number, carrier: c }: { id: string; tracking_number: string; carrier: string }) =>
      api.patch(`/orders/${id}/ship`, { tracking_number, carrier: c }),
    onSuccess: () => {
      setNotice("Order marked as shipped.");
      setTracking("");
      setCarrier("");
      qc.invalidateQueries({ queryKey: ["seller-orders-full"] });
    },
    onError: () => setNotice("Failed to mark as shipped."),
  });

  const filtered = orders.filter((o) => {
    const matchStatus = filter === "all" || o.status === filter;
    const matchSearch = !search || (o.items?.[0]?.title ?? "").toLowerCase().includes(search.toLowerCase()) || o.id.includes(search);
    return matchStatus && matchSearch;
  });

  const selected = useMemo(() => orders.find((o) => o.id === selectedId) ?? null, [orders, selectedId]);

  const counts: Record<string, number> = { all: orders.length };
  orders.forEach((o) => { counts[o.status] = (counts[o.status] ?? 0) + 1; });

  const FILTERS: FilterStatus[] = ["all", "pending", "confirmed", "shipped", "delivered", "cancelled"];

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div>
          <h1 className="text-xl font-bold text-gray-900">Sales Orders</h1>
          <p className="text-sm text-gray-400">{orders.length} total orders</p>
        </div>
        {counts.pending > 0 && (
          <div className="flex items-center gap-2 bg-amber-50 border border-amber-200 rounded-xl px-3 py-2 text-xs font-medium text-amber-800">
            <Clock className="w-3.5 h-3.5 text-amber-500" />
            {counts.pending} order{counts.pending > 1 ? "s" : ""} awaiting confirmation
          </div>
        )}
      </div>

      {/* Filters + Search */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex gap-1.5 flex-wrap">
          {FILTERS.map((f) => (
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
            placeholder="Search orders..."
            className="pl-8 pr-3 py-1.5 text-sm border border-gray-200 rounded-xl bg-white outline-none focus:ring-2 focus:ring-[#0071CE]/20 focus:border-[#0071CE] w-52"
          />
        </div>
      </div>

      {/* Main grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
        {/* Orders list */}
        <div className="lg:col-span-2 bg-white rounded-2xl border border-gray-100 overflow-hidden">
          {/* List header */}
          <div className="hidden md:grid grid-cols-[1fr_130px_90px_80px_32px] gap-3 px-4 py-2.5 bg-gray-50 border-b border-gray-100">
            {["Item", "Buyer", "Amount", "Status", ""].map((h) => (
              <p key={h} className="text-[10px] font-bold uppercase tracking-wider text-gray-400">{h}</p>
            ))}
          </div>

          {isLoading ? (
            <div className="space-y-2 p-4">
              {[1, 2, 3].map((i) => <div key={i} className="h-14 animate-pulse rounded-xl bg-gray-100" />)}
            </div>
          ) : filtered.length === 0 ? (
            <div className="py-14 text-center">
              <ShoppingBag className="w-10 h-10 mx-auto mb-3 text-gray-200" />
              <p className="text-sm text-gray-400">No orders found.</p>
            </div>
          ) : (
            <div className="divide-y divide-gray-50">
              {filtered.map((order) => {
                const cfg = STATUS_CONFIG[order.status] ?? STATUS_CONFIG.pending;
                const StatusIcon = cfg.icon;
                const isSelected = selectedId === order.id;
                return (
                  <button
                    key={order.id}
                    onClick={() => { setSelectedId(order.id); setNotice(""); }}
                    className={`w-full grid md:grid-cols-[1fr_130px_90px_80px_32px] gap-3 items-center px-4 py-3.5 text-left transition-colors ${
                      isSelected ? "bg-blue-50/50" : "hover:bg-gray-50/50"
                    }`}
                  >
                    <div className="min-w-0">
                      <p className="text-sm font-semibold text-gray-900 truncate">
                        {order.items?.[0]?.title ?? "Order item"}
                        {order.items && order.items.length > 1 && <span className="text-gray-400 font-normal"> +{order.items.length - 1} more</span>}
                      </p>
                      <p className="text-[11px] text-gray-400 mt-0.5">#{order.id.slice(0, 8)} · {timeAgo(order.created_at)}</p>
                    </div>
                    <p className="text-xs text-gray-500 hidden md:block truncate">{order.buyer_name ?? "—"}</p>
                    <p className="text-sm font-bold text-emerald-600 tabular-nums hidden md:block">
                      +{formatPrice(order.total, order.currency)}
                    </p>
                    <div className="hidden md:block">
                      <span className={`inline-flex items-center gap-1 text-[10px] font-bold px-2 py-0.5 rounded-full border ${cfg.cls}`}>
                        <StatusIcon className="w-3 h-3" /> {cfg.label}
                      </span>
                    </div>
                    <ChevronRight className="w-4 h-4 text-gray-300 hidden md:block" />
                  </button>
                );
              })}
            </div>
          )}
        </div>

        {/* Action panel */}
        <div className="bg-white rounded-2xl border border-gray-100 p-5 self-start sticky top-24">
          <h2 className="text-sm font-bold text-gray-800 mb-4">Order Actions</h2>

          {!selected ? (
            <div className="py-6 text-center">
              <ShoppingBag className="w-8 h-8 mx-auto mb-2 text-gray-200" />
              <p className="text-sm text-gray-400">Select an order to view details and take action.</p>
            </div>
          ) : (
            <div className="space-y-4">
              {/* Order summary */}
              <div className="rounded-xl bg-gray-50 border border-gray-100 p-3.5 space-y-2">
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className="text-sm font-semibold text-gray-900 truncate">
                      {selected.items?.[0]?.title ?? "Order item"}
                    </p>
                    <p className="text-xs text-gray-400 mt-0.5">Order #{selected.id.slice(0, 8)}</p>
                  </div>
                  <p className="text-sm font-bold text-emerald-600 tabular-nums shrink-0">
                    +{formatPrice(selected.total, selected.currency)}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  {(() => {
                    const cfg = STATUS_CONFIG[selected.status] ?? STATUS_CONFIG.pending;
                    const StatusIcon = cfg.icon;
                    return (
                      <span className={`inline-flex items-center gap-1 text-[10px] font-bold px-2 py-0.5 rounded-full border ${cfg.cls}`}>
                        <StatusIcon className="w-3 h-3" /> {cfg.label}
                      </span>
                    );
                  })()}
                  <span className="text-xs text-gray-400">{timeAgo(selected.created_at)}</span>
                </div>
                {selected.tracking_number && (
                  <div className="text-xs text-gray-500 pt-1 border-t border-gray-100">
                    <span className="font-medium">Tracking:</span> {selected.carrier} — {selected.tracking_number}
                  </div>
                )}
                <Link
                  href={`/orders/${selected.id}`}
                  className="inline-flex items-center gap-1 text-xs font-semibold text-[#0071CE] hover:underline mt-1"
                >
                  Full order details <ChevronRight className="w-3 h-3" />
                </Link>
              </div>

              {/* Confirm action */}
              {selected.status === "pending" && (
                <button
                  onClick={() => confirmMutation.mutate(selected.id)}
                  disabled={confirmMutation.isPending}
                  className="w-full py-2.5 rounded-xl bg-[#0071CE] text-white text-sm font-semibold hover:bg-[#005ba3] transition-colors disabled:opacity-60"
                >
                  {confirmMutation.isPending ? "Confirming…" : "✓ Confirm Order"}
                </button>
              )}

              {/* Ship action */}
              {["confirmed", "processing"].includes(selected.status) && (
                <div className="space-y-2 rounded-xl border border-gray-100 p-3.5">
                  <p className="text-xs font-bold text-gray-700 mb-2">Mark as Shipped</p>
                  <input
                    value={carrier}
                    onChange={(e) => setCarrier(e.target.value)}
                    placeholder="Carrier (DHL, Aramex, SMSA…)"
                    className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 outline-none focus:ring-2 focus:ring-[#0071CE]/20 focus:border-[#0071CE]"
                  />
                  <input
                    value={tracking}
                    onChange={(e) => setTracking(e.target.value)}
                    placeholder="Tracking number"
                    className="w-full text-sm border border-gray-200 rounded-lg px-3 py-2 outline-none focus:ring-2 focus:ring-[#0071CE]/20 focus:border-[#0071CE]"
                  />
                  <button
                    disabled={shipMutation.isPending || !carrier.trim() || !tracking.trim()}
                    onClick={() => shipMutation.mutate({ id: selected.id, tracking_number: tracking.trim(), carrier: carrier.trim() })}
                    className="w-full py-2.5 rounded-xl bg-[#0071CE] text-white text-sm font-semibold hover:bg-[#005ba3] transition-colors disabled:opacity-60"
                  >
                    {shipMutation.isPending ? "Saving…" : "📦 Mark as Shipped"}
                  </button>
                </div>
              )}

              {/* Status notice */}
              {notice && (
                <p className="text-sm font-medium text-[#0071CE] bg-blue-50 rounded-lg px-3 py-2">{notice}</p>
              )}

              {/* Terminal states */}
              {["delivered", "cancelled"].includes(selected.status) && !notice && (
                <p className="text-xs text-gray-400 text-center">
                  {selected.status === "delivered" ? "✓ This order was delivered." : "✗ This order was cancelled."}
                </p>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
