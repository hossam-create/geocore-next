"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import axios from "axios";
import { useAuthStore } from "@/store/auth";
import {
  DollarSign, TrendingUp, Package, Calendar,
  CheckCircle, Clock, ArrowUpRight, Wallet,
} from "lucide-react";
import { formatDistanceToNow, parseISO, format } from "date-fns";
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer,
} from "recharts";
import Link from "next/link";

interface EarningEntry {
  id: string;
  item_name: string;
  pickup_city: string;
  delivery_city: string;
  reward: number;
  currency: string;
  status: "delivered" | "in_transit" | "accepted";
  completed_at: string;
}

interface EarningsSummary {
  total_earnings: number;
  this_month: number;
  last_month: number;
  pending_payout: number;
  total_deliveries: number;
  avg_reward: number;
  currency: string;
}

const MOCK_SUMMARY: EarningsSummary = {
  total_earnings:    2840,
  this_month:        680,
  last_month:        920,
  pending_payout:    180,
  total_deliveries:  24,
  avg_reward:        118,
  currency:          "AED",
};

const MOCK_ENTRIES: EarningEntry[] = [
  { id: "e1", item_name: "Italian Leather Bag",      pickup_city: "Milan",    delivery_city: "Dubai",     reward: 180, currency: "AED", status: "delivered",  completed_at: new Date(Date.now() - 2 * 86400000).toISOString() },
  { id: "e2", item_name: "Swiss Chocolate Set",      pickup_city: "Zurich",   delivery_city: "Riyadh",    reward: 55,  currency: "AED", status: "in_transit", completed_at: new Date(Date.now() - 5 * 86400000).toISOString() },
  { id: "e3", item_name: "Sneakers Nike Dunk Low",   pickup_city: "New York", delivery_city: "Abu Dhabi", reward: 120, currency: "AED", status: "delivered",  completed_at: new Date(Date.now() - 8 * 86400000).toISOString() },
  { id: "e4", item_name: "Camera Lens EF 50mm",      pickup_city: "London",   delivery_city: "Dubai",     reward: 200, currency: "AED", status: "delivered",  completed_at: new Date(Date.now() - 12 * 86400000).toISOString() },
  { id: "e5", item_name: "Book Set Arabic Edition",  pickup_city: "Paris",    delivery_city: "Cairo",     reward: 70,  currency: "AED", status: "accepted",   completed_at: new Date(Date.now() - 15 * 86400000).toISOString() },
  { id: "e6", item_name: "PlayStation 5 Controller", pickup_city: "Tokyo",    delivery_city: "Dubai",     reward: 95,  currency: "AED", status: "delivered",  completed_at: new Date(Date.now() - 20 * 86400000).toISOString() },
];

const MOCK_CHART = [
  { month: "Nov", earnings: 540 },
  { month: "Dec", earnings: 780 },
  { month: "Jan", earnings: 920 },
  { month: "Feb", earnings: 610 },
  { month: "Mar", earnings: 920 },
  { month: "Apr", earnings: 680 },
];

const STATUS_CFG = {
  delivered:  { label: "Paid",        cls: "bg-emerald-50 text-emerald-700 border-emerald-200", icon: CheckCircle },
  in_transit: { label: "In Transit",  cls: "bg-amber-50 text-amber-700 border-amber-200",       icon: Clock },
  accepted:   { label: "Pending",     cls: "bg-blue-50 text-blue-700 border-blue-200",          icon: Clock },
};

type Period = "3m" | "6m" | "1y";

function fmtDate(d: string) {
  try { return format(parseISO(d), "MMM d, yyyy"); }
  catch { return d; }
}

function timeAgo(d: string) {
  try { return formatDistanceToNow(parseISO(d), { addSuffix: true }); }
  catch { return d; }
}

export default function TravelerEarningsPage() {
  const { isAuthenticated } = useAuthStore();
  const [period, setPeriod] = useState<Period>("6m");

  const { data: summary = MOCK_SUMMARY } = useQuery<EarningsSummary>({
    queryKey: ["traveler-earnings-summary"],
    queryFn: async () => {
      try {
        const { data } = await axios.get("/api/v1/traveler/earnings/summary");
        return data.data ?? MOCK_SUMMARY;
      } catch { return MOCK_SUMMARY; }
    },
    enabled: isAuthenticated,
  });

  const { data: entries = MOCK_ENTRIES } = useQuery<EarningEntry[]>({
    queryKey: ["traveler-earnings-entries"],
    queryFn: async () => {
      try {
        const { data } = await axios.get("/api/v1/traveler/earnings?limit=50");
        const d = data.data ?? [];
        return Array.isArray(d) && d.length ? d : MOCK_ENTRIES;
      } catch { return MOCK_ENTRIES; }
    },
    enabled: isAuthenticated,
  });

  const growthPct = MOCK_SUMMARY.last_month > 0
    ? (((MOCK_SUMMARY.this_month - MOCK_SUMMARY.last_month) / MOCK_SUMMARY.last_month) * 100).toFixed(1)
    : "0";
  const isGrowthPositive = parseFloat(growthPct) >= 0;

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div>
          <h1 className="text-xl font-bold text-gray-900">Earnings</h1>
          <p className="text-sm text-gray-400">Track your delivery income and payouts</p>
        </div>
        <Link
          href="/wallet"
          className="flex items-center gap-1.5 px-4 py-2 bg-[#0071CE] text-white rounded-xl text-sm font-semibold hover:bg-[#005ba3] transition-colors"
        >
          <Wallet className="w-4 h-4" /> View Wallet
        </Link>
      </div>

      {/* KPI row */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
        {[
          { label: "Total Earned",     value: `${summary.currency} ${summary.total_earnings.toLocaleString()}`, icon: DollarSign,  accent: "bg-emerald-500" },
          { label: "This Month",       value: `${summary.currency} ${summary.this_month.toLocaleString()}`,     icon: Calendar,    accent: "bg-[#0071CE]" },
          { label: "Growth",           value: `${isGrowthPositive ? "+" : ""}${growthPct}%`,                    icon: TrendingUp,  accent: isGrowthPositive ? "bg-emerald-400" : "bg-red-400" },
          { label: "Pending Payout",   value: `${summary.currency} ${summary.pending_payout}`,                  icon: Clock,       accent: "bg-amber-400" },
          { label: "Deliveries Done",  value: String(summary.total_deliveries),                                  icon: Package,     accent: "bg-violet-500" },
          { label: "Avg. per Delivery",value: `${summary.currency} ${summary.avg_reward}`,                      icon: ArrowUpRight,accent: "bg-sky-400" },
        ].map(({ label, value, icon: Icon, accent }) => (
          <div key={label} className="relative bg-white rounded-2xl border border-gray-100 p-4 overflow-hidden">
            <div className={`absolute top-0 left-0 right-0 h-[3px] ${accent}`} />
            <p className="text-[10px] font-bold text-gray-400 uppercase tracking-wider mb-2">{label}</p>
            <p className="text-lg font-bold text-gray-900 tabular-nums leading-tight">{value}</p>
            <div className="absolute bottom-3 right-3 w-7 h-7 rounded-xl bg-gray-50 flex items-center justify-center">
              <Icon className="w-3.5 h-3.5 text-gray-400" />
            </div>
          </div>
        ))}
      </div>

      {/* Chart + History grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
        {/* Chart */}
        <div className="lg:col-span-2 bg-white rounded-2xl border border-gray-100 p-5">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-sm font-semibold text-gray-800">Monthly Earnings</h2>
              <p className="text-xs text-gray-400">Revenue from completed deliveries</p>
            </div>
            <div className="flex gap-1 bg-gray-100 rounded-lg p-0.5">
              {(["3m", "6m", "1y"] as Period[]).map((p) => (
                <button
                  key={p}
                  onClick={() => setPeriod(p)}
                  className={`px-2.5 py-1 rounded-md text-xs font-semibold transition-all ${period === p ? "bg-white text-[#0071CE] shadow-sm" : "text-gray-500"}`}
                >
                  {p}
                </button>
              ))}
            </div>
          </div>
          <div className="h-52">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={MOCK_CHART} barSize={28}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" vertical={false} />
                <XAxis dataKey="month" tick={{ fontSize: 11 }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 11 }} axisLine={false} tickLine={false} tickFormatter={(v) => `${v}`} />
                <Tooltip
                  formatter={(v: number) => [`AED ${v}`, "Earnings"]}
                  contentStyle={{ borderRadius: 12, border: "1px solid #e2e8f0", fontSize: 12 }}
                />
                <Bar dataKey="earnings" fill="#0071CE" radius={[6, 6, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Payout summary */}
        <div className="bg-white rounded-2xl border border-gray-100 p-5 space-y-4">
          <h2 className="text-sm font-semibold text-gray-800">Payout Status</h2>

          <div className="bg-gradient-to-br from-[#0071CE] to-[#005ba1] rounded-xl p-4 text-white">
            <p className="text-xs opacity-70 mb-1">Available for Payout</p>
            <p className="text-2xl font-bold tabular-nums">AED {summary.pending_payout}</p>
            <button className="mt-3 text-xs font-semibold bg-white/20 hover:bg-white/30 px-3 py-1.5 rounded-lg transition-colors">
              Request Payout
            </button>
          </div>

          <div className="space-y-2.5">
            {[
              { label: "Total Paid Out",    value: `AED ${(summary.total_earnings - summary.pending_payout).toLocaleString()}`, cls: "text-emerald-600" },
              { label: "This Month",        value: `AED ${summary.this_month}`,   cls: "text-[#0071CE]" },
              { label: "Last Month",        value: `AED ${summary.last_month}`,   cls: "text-gray-600" },
              { label: "Total Deliveries",  value: String(summary.total_deliveries), cls: "text-gray-600" },
            ].map(({ label, value, cls }) => (
              <div key={label} className="flex items-center justify-between py-2 border-b border-gray-50 last:border-0">
                <p className="text-xs text-gray-500">{label}</p>
                <p className={`text-sm font-bold tabular-nums ${cls}`}>{value}</p>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Transaction history */}
      <div className="bg-white rounded-2xl border border-gray-100 overflow-hidden">
        <div className="px-5 py-4 border-b border-gray-100">
          <h2 className="text-sm font-semibold text-gray-800">Delivery History</h2>
          <p className="text-xs text-gray-400">{entries.length} deliveries</p>
        </div>

        <div className="hidden md:grid grid-cols-[1fr_160px_80px_90px_100px] gap-4 px-5 py-2.5 bg-gray-50 border-b border-gray-100">
          {["Item", "Route", "Weight", "Status", "Reward"].map((h) => (
            <p key={h} className="text-[10px] font-bold uppercase tracking-wider text-gray-400">{h}</p>
          ))}
        </div>

        <div className="divide-y divide-gray-50">
          {entries.map((entry) => {
            const cfg = STATUS_CFG[entry.status] ?? STATUS_CFG.delivered;
            const StatusIcon = cfg.icon;
            return (
              <div key={entry.id} className="grid md:grid-cols-[1fr_160px_80px_90px_100px] gap-4 items-center px-5 py-3.5 hover:bg-gray-50/50 transition-colors">
                <div className="min-w-0">
                  <p className="text-sm font-medium text-gray-900 truncate">{entry.item_name}</p>
                  <p className="text-[11px] text-gray-400 mt-0.5">{timeAgo(entry.completed_at)}</p>
                </div>
                <p className="text-xs text-gray-500 hidden md:block">
                  {entry.pickup_city} → {entry.delivery_city}
                </p>
                <p className="text-xs text-gray-400 hidden md:block">—</p>
                <div className="hidden md:block">
                  <span className={`inline-flex items-center gap-1 text-[10px] font-bold px-2 py-0.5 rounded-full border ${cfg.cls}`}>
                    <StatusIcon className="w-2.5 h-2.5" /> {cfg.label}
                  </span>
                </div>
                <p className={`text-sm font-bold tabular-nums ${entry.status === "delivered" ? "text-emerald-600" : "text-gray-400"}`}>
                  {entry.status === "delivered" ? "+" : ""}{entry.currency} {entry.reward}
                </p>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
