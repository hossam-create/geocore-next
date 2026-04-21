"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { formatPrice } from "@/lib/utils";
import {
  Package, Clock, CheckCircle, Truck, XCircle, AlertCircle,
  Search, ChevronRight, ExternalLink, ShieldAlert,
} from "lucide-react";
import { formatDistanceToNow, parseISO } from "date-fns";

interface Order {
  id: string;
  item_title: string;
  seller_name?: string;
  amount: number;
  currency: string;
  status: string;
  created_at: string;
  tracking_number?: string;
  carrier?: string;
  image?: string;
}

const MOCK: Order[] = [
  { id: "ord-001", item_title: "Sony WH-1000XM5 Headphones",     seller_name: "TechHub UAE",      amount: 1299, currency: "AED", status: "shipped",   created_at: new Date(Date.now() - 2 * 86400000).toISOString(),  tracking_number: "DHL9234567", carrier: "DHL",    image: "https://picsum.photos/seed/headphones/56/56" },
  { id: "ord-002", item_title: "Nike Air Max 270 — Size 43",      seller_name: "SportsZone",       amount: 620,  currency: "AED", status: "pending",   created_at: new Date(Date.now() - 86400000).toISOString(),      image: "https://picsum.photos/seed/nike1/56/56" },
  { id: "ord-003", item_title: "Dyson V15 Detect Vacuum",         seller_name: "HomeAppliances",   amount: 2800, currency: "AED", status: "delivered", created_at: new Date(Date.now() - 8 * 86400000).toISOString(),  image: "https://picsum.photos/seed/dyson/56/56" },
  { id: "ord-004", item_title: "Apple Watch Series 9 GPS 45mm",   seller_name: "iStore",           amount: 1890, currency: "AED", status: "confirmed", created_at: new Date(Date.now() - 3 * 86400000).toISOString(),  image: "https://picsum.photos/seed/applewatch/56/56" },
  { id: "ord-005", item_title: "LEGO Technic Set 42143",          seller_name: "ToyWorld",         amount: 480,  currency: "AED", status: "cancelled", created_at: new Date(Date.now() - 15 * 86400000).toISOString(), image: "https://picsum.photos/seed/lego/56/56" },
  { id: "ord-006", item_title: "Bose QuietComfort Ultra Earbuds", seller_name: "AudioPro",         amount: 1100, currency: "AED", status: "disputed",  created_at: new Date(Date.now() - 10 * 86400000).toISOString(), image: "https://picsum.photos/seed/bose/56/56" },
];

const STATUS_CFG: Record<string, { label: string; cls: string; icon: React.ElementType }> = {
  pending:   { label: "Pending",   cls: "bg-amber-50 text-amber-700 border-amber-200",    icon: Clock },
  confirmed: { label: "Confirmed", cls: "bg-violet-50 text-violet-700 border-violet-200", icon: CheckCircle },
  shipped:   { label: "Shipped",   cls: "bg-blue-50 text-blue-700 border-blue-200",       icon: Truck },
  delivered: { label: "Delivered", cls: "bg-emerald-50 text-emerald-700 border-emerald-200", icon: CheckCircle },
  cancelled: { label: "Cancelled", cls: "bg-red-50 text-red-600 border-red-200",          icon: XCircle },
  disputed:  { label: "Disputed",  cls: "bg-orange-50 text-orange-700 border-orange-200", icon: AlertCircle },
};

type FilterStatus = "all" | "active" | "delivered" | "cancelled" | "disputed";

function timeAgo(d: string) {
  try { return formatDistanceToNow(parseISO(d), { addSuffix: true }); }
  catch { return d; }
}

export default function BuyerOrdersPage() {
  const { isAuthenticated } = useAuthStore();
  const [filter, setFilter] = useState<FilterStatus>("all");
  const [search, setSearch] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const { data: orders = MOCK, isLoading } = useQuery<Order[]>({
    queryKey: ["buyer-orders-full"],
    queryFn: async () => {
      try {
        const d = (await api.get("/orders?page=1&limit=100")).data?.data ?? [];
        return Array.isArray(d) && d.length ? d : MOCK;
      } catch { return MOCK; }
    },
    enabled: isAuthenticated,
  });

  const filtered = orders.filter((o) => {
    const matchStatus =
      filter === "all" ? true :
      filter === "active" ? ["pending", "confirmed", "shipped"].includes(o.status) :
      o.status === filter;
    const matchSearch = !search || o.item_title.toLowerCase().includes(search.toLowerCase()) || o.id.includes(search);
    return matchStatus && matchSearch;
  });

  const counts: Record<string, number> = { all: orders.length };
  orders.forEach((o) => {
    if (["pending", "confirmed", "shipped"].includes(o.status)) counts.active = (counts.active ?? 0) + 1;
    counts[o.status] = (counts[o.status] ?? 0) + 1;
  });

  const selected = useMemo(() => orders.find((o) => o.id === selectedId) ?? null, [orders, selectedId]);

  const FILTERS: { key: FilterStatus; label: string }[] = [
    { key: "all",       label: `All (${counts.all ?? 0})` },
    { key: "active",    label: `Active (${counts.active ?? 0})` },
    { key: "delivered", label: `Delivered (${counts.delivered ?? 0})` },
    { key: "cancelled", label: `Cancelled (${counts.cancelled ?? 0})` },
    { key: "disputed",  label: `Disputed (${counts.disputed ?? 0})` },
  ];

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div>
          <h1 className="text-xl font-bold text-gray-900">My Orders</h1>
          <p className="text-sm text-gray-400">{orders.length} total purchases</p>
        </div>
        <Link
          href="/listings"
          className="flex items-center gap-1.5 px-4 py-2 bg-indigo-600 text-white rounded-xl text-sm font-semibold hover:bg-indigo-700 transition-colors"
        >
          Browse More
        </Link>
      </div>

      {/* Filters + Search */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex gap-1.5 flex-wrap">
          {FILTERS.map(({ key, label }) => (
            <button
              key={key}
              onClick={() => setFilter(key)}
              className={`px-3 py-1.5 rounded-full text-xs font-semibold transition-colors ${
                filter === key
                  ? "bg-indigo-600 text-white"
                  : "bg-white text-gray-500 border border-gray-200 hover:bg-gray-50"
              }`}
            >
              {label}
            </button>
          ))}
        </div>
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search orders..."
            className="pl-8 pr-3 py-1.5 text-sm border border-gray-200 rounded-xl bg-white outline-none focus:ring-2 focus:ring-indigo-300 focus:border-indigo-400 w-52"
          />
        </div>
      </div>

      {/* Main grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
        {/* Orders list */}
        <div className="lg:col-span-2 bg-white rounded-2xl border border-gray-100 overflow-hidden">
          <div className="hidden md:grid grid-cols-[44px_1fr_120px_90px_32px] gap-3 px-4 py-2.5 bg-gray-50 border-b border-gray-100">
            {["", "Item", "Amount", "Status", ""].map((h) => (
              <p key={h} className="text-[10px] font-bold uppercase tracking-wider text-gray-400">{h}</p>
            ))}
          </div>

          {isLoading ? (
            <div className="space-y-2 p-4">
              {[1, 2, 3].map((i) => <div key={i} className="h-14 animate-pulse rounded-xl bg-gray-100" />)}
            </div>
          ) : filtered.length === 0 ? (
            <div className="py-14 text-center">
              <Package className="w-10 h-10 mx-auto mb-3 text-gray-200" />
              <p className="text-sm text-gray-400">No orders found.</p>
            </div>
          ) : (
            <div className="divide-y divide-gray-50">
              {filtered.map((order) => {
                const cfg = STATUS_CFG[order.status] ?? STATUS_CFG.pending;
                const StatusIcon = cfg.icon;
                const isSelected = selectedId === order.id;
                return (
                  <button
                    key={order.id}
                    onClick={() => setSelectedId(order.id)}
                    className={`w-full grid md:grid-cols-[44px_1fr_120px_90px_32px] gap-3 items-center px-4 py-3.5 text-left transition-colors ${
                      isSelected ? "bg-indigo-50/60" : "hover:bg-gray-50/60"
                    }`}
                  >
                    <div className="w-11 h-11 rounded-xl overflow-hidden bg-gray-100 shrink-0">
                      {order.image ? (
                        <img src={order.image} alt="" className="w-full h-full object-cover" />
                      ) : (
                        <div className="w-full h-full flex items-center justify-center">
                          <Package className="w-4 h-4 text-gray-300" />
                        </div>
                      )}
                    </div>
                    <div className="min-w-0">
                      <p className="text-sm font-semibold text-gray-900 truncate">{order.item_title}</p>
                      <p className="text-[11px] text-gray-400 mt-0.5">
                        {order.seller_name && <span>{order.seller_name} · </span>}
                        {timeAgo(order.created_at)}
                      </p>
                    </div>
                    <p className="text-sm font-bold text-gray-800 tabular-nums hidden md:block">
                      {formatPrice(order.amount, order.currency)}
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

        {/* Detail panel */}
        <div className="bg-white rounded-2xl border border-gray-100 p-5 self-start sticky top-24">
          <h2 className="text-sm font-bold text-gray-800 mb-4">Order Details</h2>

          {!selected ? (
            <div className="py-6 text-center">
              <Package className="w-8 h-8 mx-auto mb-2 text-gray-200" />
              <p className="text-sm text-gray-400">Select an order to see details.</p>
            </div>
          ) : (
            <div className="space-y-4">
              {/* Summary */}
              <div className="rounded-xl bg-gray-50 border border-gray-100 p-3.5 space-y-2.5">
                {selected.image && (
                  <img src={selected.image} alt="" className="w-full h-32 object-cover rounded-lg" />
                )}
                <p className="text-sm font-semibold text-gray-900">{selected.item_title}</p>
                {selected.seller_name && (
                  <p className="text-xs text-gray-500">Sold by <span className="font-medium">{selected.seller_name}</span></p>
                )}
                <div className="flex items-center justify-between">
                  <p className="text-sm font-bold text-gray-800">{formatPrice(selected.amount, selected.currency)}</p>
                  {(() => {
                    const cfg = STATUS_CFG[selected.status] ?? STATUS_CFG.pending;
                    const StatusIcon = cfg.icon;
                    return (
                      <span className={`inline-flex items-center gap-1 text-[10px] font-bold px-2 py-0.5 rounded-full border ${cfg.cls}`}>
                        <StatusIcon className="w-3 h-3" /> {cfg.label}
                      </span>
                    );
                  })()}
                </div>
                <p className="text-[11px] text-gray-400">{timeAgo(selected.created_at)}</p>
              </div>

              {/* Tracking */}
              {selected.tracking_number && (
                <div className="rounded-xl border border-blue-100 bg-blue-50 p-3.5">
                  <div className="flex items-center gap-2 mb-1">
                    <Truck className="w-4 h-4 text-blue-600" />
                    <p className="text-xs font-bold text-blue-700">Shipment Tracking</p>
                  </div>
                  <p className="text-xs text-blue-700">
                    <span className="font-medium">{selected.carrier}</span> — {selected.tracking_number}
                  </p>
                </div>
              )}

              {/* Actions */}
              <div className="space-y-2">
                <Link
                  href={`/orders/${selected.id}`}
                  className="flex items-center justify-center gap-2 w-full py-2.5 rounded-xl border border-gray-200 bg-white text-sm font-semibold text-gray-700 hover:bg-gray-50 transition-colors"
                >
                  <ExternalLink className="w-3.5 h-3.5" /> Full Order Details
                </Link>

                {selected.status === "delivered" && (
                  <Link
                    href={`/disputes/new?order_id=${selected.id}`}
                    className="flex items-center justify-center gap-2 w-full py-2.5 rounded-xl border border-orange-200 bg-orange-50 text-sm font-semibold text-orange-700 hover:bg-orange-100 transition-colors"
                  >
                    <ShieldAlert className="w-3.5 h-3.5" /> Open Dispute
                  </Link>
                )}

                {selected.status === "disputed" && (
                  <Link
                    href={`/buyer/disputes`}
                    className="flex items-center justify-center gap-2 w-full py-2.5 rounded-xl border border-orange-200 bg-orange-50 text-sm font-semibold text-orange-700 hover:bg-orange-100 transition-colors"
                  >
                    <AlertCircle className="w-3.5 h-3.5" /> View Dispute
                  </Link>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
