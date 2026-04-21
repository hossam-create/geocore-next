"use client";

import { useQuery } from "@tanstack/react-query";
import { dashboardFullApi } from "@/lib/api";
import { mockDashboardStats } from "@/lib/mockData";
import PageHeader from "@/components/shared/PageHeader";
import { timeAgo } from "@/lib/format";
import StatusBadge from "@/components/shared/StatusBadge";
import {
  Users, List, Gavel, DollarSign, AlertTriangle, Clock,
  TrendingUp, ArrowUpRight, ArrowDownRight, ShieldCheck, FileWarning, UserCheck,
} from "lucide-react";
import {
  LineChart, Line, BarChart, Bar, PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend,
} from "recharts";

/* ── mock chart data (used when backend unavailable) ───────────────── */
const mockSignups = Array.from({ length: 30 }, (_, i) => {
  const d = new Date(); d.setDate(d.getDate() - (29 - i));
  return { date: d.toISOString().slice(5, 10), count: Math.floor(Math.random() * 40 + 10) };
});
const mockRevenue = Array.from({ length: 30 }, (_, i) => {
  const d = new Date(); d.setDate(d.getDate() - (29 - i));
  return { date: d.toISOString().slice(5, 10), amount: Math.floor(Math.random() * 800 + 200) };
});
const mockByCategory = [
  { category: "سيارات", count: 234 }, { category: "إلكترونيات", count: 189 },
  { category: "عقارات", count: 156 }, { category: "أزياء", count: 98 },
  { category: "أخرى", count: 67 },
];
const mockActivity = [
  { type: "new_user", description: "New user registered: ali@example.com", created_at: new Date().toISOString() },
  { type: "new_listing", description: "New listing: iPhone 15 Pro Max", created_at: new Date(Date.now() - 300000).toISOString() },
  { type: "auction_ended", description: "Auction ended: Rare Painting — $5,200", created_at: new Date(Date.now() - 600000).toISOString() },
  { type: "payment", description: "Payment received: $1,200 from Ali Ahmed", created_at: new Date(Date.now() - 900000).toISOString() },
  { type: "report", description: "New report: spam listing flagged", created_at: new Date(Date.now() - 1200000).toISOString() },
];

const PIE_COLORS = ["#6366f1", "#06b6d4", "#f59e0b", "#10b981", "#8b5cf6"];

interface DashboardData {
  stats?: Record<string, number>;
  charts?: {
    daily_signups?: { date: string; count: number }[];
    daily_revenue?: { date: string; amount: number }[];
    listings_by_category?: { category: string; count: number }[];
  };
  recent_activity?: { type?: string; description?: string; created_at?: string }[];
}

export default function DashboardPage() {
  const { data } = useQuery<DashboardData>({
    queryKey: ["dashboard-full"],
    queryFn: dashboardFullApi.get,
    retry: 1,
  });

  const s = data?.stats ?? {
    total_users: mockDashboardStats.totalUsers,
    new_users_today: 24,
    new_users_week: mockDashboardStats.totalUsers * 0.08,
    total_listings: mockDashboardStats.activeListings + 400,
    active_listings: mockDashboardStats.activeListings,
    pending_listings: 12,
    total_auctions: mockDashboardStats.liveAuctions + 80,
    active_auctions: mockDashboardStats.liveAuctions,
    total_revenue: mockDashboardStats.revenue,
    revenue_today: 3200,
    revenue_month: mockDashboardStats.revenue,
    pending_reports: 5,
    pending_kyc: 8,
  };

  const signups = data?.charts?.daily_signups ?? mockSignups;
  const revenue = data?.charts?.daily_revenue ?? mockRevenue;
  const byCategory = data?.charts?.listings_by_category ?? mockByCategory;
  const activity = data?.recent_activity ?? mockActivity;

  const kpis = [
    { label: "Total Users", value: (s.total_users ?? 0).toLocaleString(), icon: Users, trend: 8.2 },
    { label: "Active Listings", value: (s.active_listings ?? 0).toLocaleString(), icon: List, trend: 4.1 },
    { label: "Live Auctions", value: (s.active_auctions ?? 0).toString(), icon: Gavel, trend: -2.3 },
    { label: "Revenue (30d)", value: `$${(s.revenue_month ?? 0).toLocaleString()}`, icon: DollarSign, trend: 12.5 },
  ];

  const alerts = [
    { label: "Pending Listings", count: s.pending_listings ?? 0, href: "/operations/listings", color: "var(--color-warning)" },
    { label: "Pending KYC", count: s.pending_kyc ?? 0, href: "/custodii/decisions", color: "var(--color-info)" },
    { label: "Open Reports", count: s.pending_reports ?? 0, href: "/analytics/reports", color: "var(--color-danger)" },
  ];

  const activityIcon = (type: string) => {
    if (type === "new_user") return <UserCheck className="w-3.5 h-3.5 text-blue-500" />;
    if (type === "new_listing") return <List className="w-3.5 h-3.5 text-green-500" />;
    if (type === "payment") return <DollarSign className="w-3.5 h-3.5 text-emerald-500" />;
    if (type === "report") return <FileWarning className="w-3.5 h-3.5 text-red-500" />;
    return <Clock className="w-3.5 h-3.5 text-slate-400" />;
  };

  return (
    <div className="space-y-6">
      <PageHeader title="Dashboard" description="Platform overview and key metrics" />

      {/* KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {kpis.map((k) => {
          const Icon = k.icon;
          return (
            <div key={k.label} className="rounded-xl border border-slate-200 bg-white p-4">
              <div className="flex items-center justify-between mb-3">
                <div className="w-9 h-9 rounded-lg flex items-center justify-center bg-slate-100">
                  <Icon className="w-4 h-4 text-slate-500" />
                </div>
                {k.trend !== undefined && (
                  <span className={`flex items-center gap-0.5 text-xs font-medium ${k.trend > 0 ? "text-green-600" : "text-red-500"}`}>
                    {k.trend > 0 ? <ArrowUpRight className="w-3 h-3" /> : <ArrowDownRight className="w-3 h-3" />}
                    {Math.abs(k.trend)}%
                  </span>
                )}
              </div>
              <p className="text-2xl font-bold text-slate-800">{k.value}</p>
              <p className="text-xs mt-0.5 text-slate-400">{k.label}</p>
            </div>
          );
        })}
      </div>

      {/* Alert Badges */}
      {alerts.some((a) => a.count > 0) && (
        <div className="flex flex-wrap gap-3">
          {alerts.filter((a) => a.count > 0).map((a) => (
            <a key={a.label} href={a.href} className="flex items-center gap-2 px-3 py-2 rounded-lg border border-slate-200 bg-white hover:shadow-sm transition-shadow">
              <span className="w-2 h-2 rounded-full" style={{ background: a.color }} />
              <span className="text-sm font-medium text-slate-700">{a.label}</span>
              <span className="text-xs font-bold px-1.5 py-0.5 rounded-full text-white" style={{ background: a.color }}>{a.count}</span>
            </a>
          ))}
        </div>
      )}

      {/* Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Daily Signups Line Chart */}
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="text-sm font-semibold text-slate-700 mb-4">Daily Signups (30 days)</h3>
          <ResponsiveContainer width="100%" height={240}>
            <LineChart data={signups}>
              <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
              <XAxis dataKey="date" tick={{ fontSize: 10 }} stroke="#94a3b8" />
              <YAxis tick={{ fontSize: 10 }} stroke="#94a3b8" />
              <Tooltip contentStyle={{ fontSize: 12, borderRadius: 8 }} />
              <Line type="monotone" dataKey="count" stroke="#6366f1" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* Daily Revenue Bar Chart */}
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="text-sm font-semibold text-slate-700 mb-4">Daily Revenue (30 days)</h3>
          <ResponsiveContainer width="100%" height={240}>
            <BarChart data={revenue}>
              <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
              <XAxis dataKey="date" tick={{ fontSize: 10 }} stroke="#94a3b8" />
              <YAxis tick={{ fontSize: 10 }} stroke="#94a3b8" />
              <Tooltip contentStyle={{ fontSize: 12, borderRadius: 8 }} formatter={(v: number) => [`$${v}`, "Revenue"]} />
              <Bar dataKey="amount" fill="#06b6d4" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Pie + Activity Row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Listings by Category Pie */}
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="text-sm font-semibold text-slate-700 mb-4">Listings by Category</h3>
          <ResponsiveContainer width="100%" height={240}>
            <PieChart>
              <Pie data={byCategory} dataKey="count" nameKey="category" cx="50%" cy="50%" outerRadius={80} label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`} labelLine={false} fontSize={10}>
                {byCategory.map((_, i) => <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />)}
              </Pie>
              <Tooltip contentStyle={{ fontSize: 12, borderRadius: 8 }} />
            </PieChart>
          </ResponsiveContainer>
        </div>

        {/* Recent Activity Feed */}
        <div className="lg:col-span-2 rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="text-sm font-semibold text-slate-700 mb-4">Recent Activity</h3>
          <div className="space-y-3 max-h-[260px] overflow-y-auto">
            {activity.map((a, i) => {
              const type = a?.type ?? "activity";
              const description = a?.description ?? "Activity event";
              const createdAt = a?.created_at ?? new Date().toISOString();

              return (
                <div key={i} className="flex items-start gap-3 pb-3 border-b border-slate-50 last:border-0">
                  <div className="mt-0.5">{activityIcon(type)}</div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-slate-700 truncate">{description}</p>
                    <p className="text-[11px] text-slate-400">{timeAgo(createdAt)}</p>
                  </div>
                  <StatusBadge status={type.replace(/_/g, " ")} variant="neutral" />
                </div>
              );
            })}
            {activity.length === 0 && <p className="text-sm text-slate-400 text-center py-4">No recent activity</p>}
          </div>
        </div>
      </div>
    </div>
  );
}
