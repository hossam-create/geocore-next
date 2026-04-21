"use client";

import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import StatCard from "@/components/shared/StatCard";
import { analyticsApi } from "@/lib/api";
import { mockDashboardStats } from "@/lib/mockData";
import { Users, UserPlus, ShoppingBag, Gavel, DollarSign, TrendingUp, BarChart3, ArrowRight } from "lucide-react";

// ── Mock fallbacks for timeseries, categories, funnel ──────────────────────
const MOCK_TIMESERIES = Array.from({ length: 30 }, (_, i) => ({
  date: new Date(Date.now() - (29 - i) * 86400000).toISOString().slice(0, 10),
  dau: Math.round(800 + Math.random() * 400 + i * 12),
  revenue: Math.round(6000 + Math.random() * 3000 + i * 200),
  gmv: Math.round(30000 + Math.random() * 15000 + i * 800),
}));

const MOCK_CATEGORIES = [
  { name: "Electronics", revenue: 124500 },
  { name: "Jewelry", revenue: 98200 },
  { name: "Automotive", revenue: 76800 },
  { name: "Fashion", revenue: 62400 },
  { name: "Home & Garden", revenue: 54100 },
  { name: "Collectibles", revenue: 41200 },
  { name: "Sports", revenue: 33800 },
  { name: "Art", revenue: 28700 },
  { name: "Books", revenue: 19500 },
  { name: "Toys", revenue: 14200 },
];

const MOCK_FUNNEL = [
  { stage: "Registered", count: 12453 },
  { stage: "Listed", count: 4821 },
  { stage: "Sold", count: 2156 },
  { stage: "Reviewed", count: 1420 },
];

type TimeseriesPoint = { date: string; dau: number; revenue: number; gmv: number };
type CategoryRow = { name: string; revenue: number };
type FunnelStep = { stage: string; count: number };

// ── Sparkline: pure CSS bar chart ──────────────────────────────────────────
function MiniBarChart({ data, dataKey, color }: { data: TimeseriesPoint[]; dataKey: keyof TimeseriesPoint; color: string }) {
  const values = data.map((d) => Number(d[dataKey]));
  const max = Math.max(...values, 1);
  return (
    <div className="flex items-end gap-[2px] h-20">
      {values.map((v, i) => (
        <div
          key={i}
          className="flex-1 rounded-t-sm transition-all"
          style={{ height: `${(v / max) * 100}%`, background: color, opacity: 0.7 + (i / values.length) * 0.3 }}
          title={`${data[i].date}: ${v.toLocaleString()}`}
        />
      ))}
    </div>
  );
}

export default function AnalyticsOverviewPage() {
  const [period, setPeriod] = useState<"30" | "90">("30");

  const { data: overview } = useQuery({
    queryKey: ["analytics", "overview"],
    queryFn: () => analyticsApi.overview(),
    retry: 1,
  });

  const { data: rawTimeseries } = useQuery({
    queryKey: ["analytics", "timeseries", period],
    queryFn: () => analyticsApi.timeseries({ days: period }),
    retry: 1,
  });

  const { data: rawCategories } = useQuery({
    queryKey: ["analytics", "top-categories"],
    queryFn: () => analyticsApi.topCategories({ limit: "10" }),
    retry: 1,
  });

  const { data: rawFunnel } = useQuery({
    queryKey: ["analytics", "funnel"],
    queryFn: () => analyticsApi.funnel(),
    retry: 1,
  });

  // Normalize with fallbacks
  const stats = overview ?? mockDashboardStats;
  const timeseries: TimeseriesPoint[] = Array.isArray(rawTimeseries) && rawTimeseries.length > 0 ? rawTimeseries : MOCK_TIMESERIES;
  const categories: CategoryRow[] = Array.isArray(rawCategories) && rawCategories.length > 0 ? rawCategories : MOCK_CATEGORIES;
  const funnel: FunnelStep[] = Array.isArray(rawFunnel) && rawFunnel.length > 0 ? rawFunnel : MOCK_FUNNEL;

  const maxCatRevenue = Math.max(...categories.map((c) => c.revenue), 1);
  const funnelMax = Math.max(...funnel.map((f) => f.count), 1);

  const dau = (stats as Record<string, unknown>).daily_active_users ?? (stats as Record<string, unknown>).totalUsers ?? 0;
  const newRegs = (stats as Record<string, unknown>).new_registrations ?? 0;
  const listingsCreated = (stats as Record<string, unknown>).listings_created ?? (stats as Record<string, unknown>).activeListings ?? 0;
  const auctionsStarted = (stats as Record<string, unknown>).auctions_started ?? (stats as Record<string, unknown>).liveAuctions ?? 0;
  const gmv = (stats as Record<string, unknown>).gmv ?? 0;
  const takeRate = (stats as Record<string, unknown>).take_rate ?? 0;

  return (
    <div className="space-y-6">
      <PageHeader title="Analytics Overview" description="Daily active users, revenue, GMV, and marketplace health" />

      {/* KPI Cards */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
        <StatCard label="DAU" value={Number(dau).toLocaleString()} icon={<Users className="w-4 h-4" />} />
        <StatCard label="New Registrations" value={Number(newRegs).toLocaleString()} icon={<UserPlus className="w-4 h-4" />} />
        <StatCard label="Listings Created" value={Number(listingsCreated).toLocaleString()} icon={<ShoppingBag className="w-4 h-4" />} />
        <StatCard label="Auctions Started" value={Number(auctionsStarted).toLocaleString()} icon={<Gavel className="w-4 h-4" />} />
        <StatCard label="GMV" value={`$${Number(gmv).toLocaleString()}`} icon={<DollarSign className="w-4 h-4" />} trend="up" />
        <StatCard label="Take Rate" value={`${Number(takeRate)}%`} icon={<TrendingUp className="w-4 h-4" />} />
      </div>

      {/* Period Toggle */}
      <div className="flex items-center gap-2">
        <span className="text-xs font-medium" style={{ color: "var(--text-tertiary)" }}>Period:</span>
        {(["30", "90"] as const).map((p) => (
          <button
            key={p}
            onClick={() => setPeriod(p)}
            className="px-3 py-1 rounded-lg text-xs font-medium transition-colors"
            style={{
              background: period === p ? "var(--color-brand)" : "var(--bg-surface)",
              color: period === p ? "#fff" : "var(--text-secondary)",
              border: period === p ? "none" : "1px solid var(--border-default)",
            }}
          >
            {p}d
          </button>
        ))}
      </div>

      {/* Line Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div className="surface p-4 rounded-lg">
          <p className="text-xs font-semibold mb-3 flex items-center gap-1.5" style={{ color: "var(--text-secondary)" }}>
            <Users className="w-3.5 h-3.5" /> DAU ({period}d)
          </p>
          <MiniBarChart data={timeseries} dataKey="dau" color="#3b82f6" />
        </div>
        <div className="surface p-4 rounded-lg">
          <p className="text-xs font-semibold mb-3 flex items-center gap-1.5" style={{ color: "var(--text-secondary)" }}>
            <DollarSign className="w-3.5 h-3.5" /> Revenue ({period}d)
          </p>
          <MiniBarChart data={timeseries} dataKey="revenue" color="#10b981" />
        </div>
        <div className="surface p-4 rounded-lg">
          <p className="text-xs font-semibold mb-3 flex items-center gap-1.5" style={{ color: "var(--text-secondary)" }}>
            <TrendingUp className="w-3.5 h-3.5" /> GMV ({period}d)
          </p>
          <MiniBarChart data={timeseries} dataKey="gmv" color="#8b5cf6" />
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Top 10 Categories */}
        <div className="surface p-5 rounded-lg">
          <h3 className="text-sm font-semibold mb-4 flex items-center gap-2" style={{ color: "var(--text-primary)" }}>
            <BarChart3 className="w-4 h-4" /> Top 10 Categories by Revenue
          </h3>
          <div className="space-y-2">
            {categories.slice(0, 10).map((cat, i) => (
              <div key={cat.name} className="flex items-center gap-3">
                <span className="w-5 text-xs font-mono text-right" style={{ color: "var(--text-tertiary)" }}>{i + 1}</span>
                <span className="w-28 text-sm truncate" style={{ color: "var(--text-secondary)" }}>{cat.name}</span>
                <div className="flex-1 h-2.5 rounded-full" style={{ background: "var(--bg-inset)" }}>
                  <div className="h-full rounded-full" style={{ width: `${(cat.revenue / maxCatRevenue) * 100}%`, background: "var(--color-brand)" }} />
                </div>
                <span className="w-20 text-right text-xs font-medium" style={{ color: "var(--text-primary)" }}>${cat.revenue.toLocaleString()}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Conversion Funnel */}
        <div className="surface p-5 rounded-lg">
          <h3 className="text-sm font-semibold mb-4" style={{ color: "var(--text-primary)" }}>
            Conversion Funnel
          </h3>
          <div className="space-y-3">
            {funnel.map((step, i) => {
              const pct = ((step.count / funnelMax) * 100).toFixed(0);
              const dropoff = i > 0 ? ((1 - step.count / funnel[i - 1].count) * 100).toFixed(1) : null;
              return (
                <div key={step.stage}>
                  <div className="flex items-center justify-between mb-1">
                    <div className="flex items-center gap-2">
                      {i > 0 && <ArrowRight className="w-3 h-3" style={{ color: "var(--text-tertiary)" }} />}
                      <span className="text-sm font-medium" style={{ color: "var(--text-primary)" }}>{step.stage}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-bold" style={{ color: "var(--text-primary)" }}>{step.count.toLocaleString()}</span>
                      {dropoff && (
                        <span className="text-[10px] px-1.5 py-0.5 rounded-full font-medium" style={{ background: "rgba(239,68,68,0.1)", color: "var(--color-danger)" }}>
                          -{dropoff}%
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="h-3 rounded-full" style={{ background: "var(--bg-inset)" }}>
                    <div
                      className="h-full rounded-full transition-all"
                      style={{
                        width: `${pct}%`,
                        background: `hsl(${220 - i * 40}, 70%, 55%)`,
                      }}
                    />
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}
