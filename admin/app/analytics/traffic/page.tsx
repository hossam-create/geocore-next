"use client";

import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import DataTable from "@/components/shared/DataTable";
import { analyticsApi } from "@/lib/api";
import { Eye, Users, Heart, MessageSquare, TrendingUp, Clock } from "lucide-react";

type TrafficRow = {
  listing_id: string;
  title: string;
  page_views: number;
  unique_visitors: number;
  watchlist_adds: number;
  message_inquiries: number;
  conversion_rate: number;
  avg_time_on_page: number;
};

const MOCK_TRAFFIC: TrafficRow[] = [
  { listing_id: "L-001", title: "iPhone 15 Pro Max", page_views: 3421, unique_visitors: 2840, watchlist_adds: 156, message_inquiries: 23, conversion_rate: 4.2, avg_time_on_page: 45 },
  { listing_id: "L-002", title: "MacBook Pro M4", page_views: 2890, unique_visitors: 2410, watchlist_adds: 201, message_inquiries: 18, conversion_rate: 3.8, avg_time_on_page: 62 },
  { listing_id: "L-003", title: "Vintage Rolex Submariner", page_views: 5200, unique_visitors: 4100, watchlist_adds: 342, message_inquiries: 45, conversion_rate: 6.1, avg_time_on_page: 78 },
  { listing_id: "L-004", title: "Antique Persian Vase", page_views: 890, unique_visitors: 720, watchlist_adds: 34, message_inquiries: 8, conversion_rate: 2.1, avg_time_on_page: 33 },
  { listing_id: "L-005", title: "Gaming PC RTX 5090", page_views: 4100, unique_visitors: 3200, watchlist_adds: 280, message_inquiries: 31, conversion_rate: 5.3, avg_time_on_page: 55 },
  { listing_id: "L-006", title: "Gold Necklace 18K", page_views: 1840, unique_visitors: 1520, watchlist_adds: 98, message_inquiries: 14, conversion_rate: 3.2, avg_time_on_page: 41 },
  { listing_id: "L-007", title: "Signed LeBron Jersey", page_views: 6700, unique_visitors: 5400, watchlist_adds: 520, message_inquiries: 67, conversion_rate: 7.8, avg_time_on_page: 92 },
  { listing_id: "L-008", title: "Classic Mercedes 300SL", page_views: 12300, unique_visitors: 9800, watchlist_adds: 890, message_inquiries: 102, conversion_rate: 1.2, avg_time_on_page: 120 },
];

function normalizeTraffic(payload: unknown): TrafficRow[] {
  const box = payload as { data?: unknown[] } | unknown[] | null | undefined;
  const rows = Array.isArray(box) ? box : Array.isArray((box as { data?: unknown[] })?.data) ? (box as { data?: unknown[] }).data : [];
  return (rows as Record<string, unknown>[]).map((item) => ({
    listing_id: String(item.listing_id ?? item.id ?? ""),
    title: String(item.title ?? "Untitled"),
    page_views: Number(item.page_views ?? 0),
    unique_visitors: Number(item.unique_visitors ?? 0),
    watchlist_adds: Number(item.watchlist_adds ?? 0),
    message_inquiries: Number(item.message_inquiries ?? 0),
    conversion_rate: Number(item.conversion_rate ?? 0),
    avg_time_on_page: Number(item.avg_time_on_page ?? 0),
  })).filter((x) => x.listing_id);
}

type SortKey = keyof TrafficRow;

export default function TrafficWatchPage() {
  const [search, setSearch] = useState("");
  const [sortKey, setSortKey] = useState<SortKey>("page_views");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc");

  const { data: liveTraffic, isLoading } = useQuery({
    queryKey: ["analytics", "traffic"],
    queryFn: async () => {
      try {
        const res = await analyticsApi.traffic();
        return normalizeTraffic(res);
      } catch { return []; }
    },
    retry: 1,
  });

  const traffic = liveTraffic?.length ? liveTraffic : MOCK_TRAFFIC;

  const sorted = useMemo(() => {
    let filtered = traffic;
    if (search) {
      const q = search.toLowerCase();
      filtered = filtered.filter((r) => r.title.toLowerCase().includes(q) || r.listing_id.toLowerCase().includes(q));
    }
    return [...filtered].sort((a, b) => {
      const av = a[sortKey];
      const bv = b[sortKey];
      if (typeof av === "number" && typeof bv === "number") return sortDir === "desc" ? bv - av : av - bv;
      return sortDir === "desc" ? String(bv).localeCompare(String(av)) : String(av).localeCompare(String(bv));
    });
  }, [traffic, search, sortKey, sortDir]);

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === "desc" ? "asc" : "desc"));
    } else {
      setSortKey(key);
      setSortDir("desc");
    }
  };

  const SortHeader = ({ label, field, icon }: { label: string; field: SortKey; icon?: React.ReactNode }) => (
    <button onClick={() => handleSort(field)} className="flex items-center gap-1 text-left w-full">
      {icon}
      {label}
      {sortKey === field && <span className="text-[10px]">{sortDir === "desc" ? "▼" : "▲"}</span>}
    </button>
  );

  const totals = useMemo(() => ({
    views: sorted.reduce((s, r) => s + r.page_views, 0),
    visitors: sorted.reduce((s, r) => s + r.unique_visitors, 0),
    watchlist: sorted.reduce((s, r) => s + r.watchlist_adds, 0),
    messages: sorted.reduce((s, r) => s + r.message_inquiries, 0),
  }), [sorted]);

  return (
    <div className="space-y-6">
      <PageHeader title="Traffic Watch" description="Per-listing traffic analytics — page views, engagement, and conversion" />

      {/* Summary KPIs */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div className="surface p-3 rounded-lg">
          <div className="flex items-center gap-2 mb-1"><Eye className="w-4 h-4" style={{ color: "var(--color-brand)" }} /><span className="text-xs" style={{ color: "var(--text-tertiary)" }}>Total Views</span></div>
          <p className="text-lg font-bold" style={{ color: "var(--text-primary)" }}>{totals.views.toLocaleString()}</p>
        </div>
        <div className="surface p-3 rounded-lg">
          <div className="flex items-center gap-2 mb-1"><Users className="w-4 h-4" style={{ color: "#8b5cf6" }} /><span className="text-xs" style={{ color: "var(--text-tertiary)" }}>Unique Visitors</span></div>
          <p className="text-lg font-bold" style={{ color: "var(--text-primary)" }}>{totals.visitors.toLocaleString()}</p>
        </div>
        <div className="surface p-3 rounded-lg">
          <div className="flex items-center gap-2 mb-1"><Heart className="w-4 h-4" style={{ color: "#ef4444" }} /><span className="text-xs" style={{ color: "var(--text-tertiary)" }}>Watchlist Adds</span></div>
          <p className="text-lg font-bold" style={{ color: "var(--text-primary)" }}>{totals.watchlist.toLocaleString()}</p>
        </div>
        <div className="surface p-3 rounded-lg">
          <div className="flex items-center gap-2 mb-1"><MessageSquare className="w-4 h-4" style={{ color: "#f59e0b" }} /><span className="text-xs" style={{ color: "var(--text-tertiary)" }}>Inquiries</span></div>
          <p className="text-lg font-bold" style={{ color: "var(--text-primary)" }}>{totals.messages.toLocaleString()}</p>
        </div>
      </div>

      {/* Search */}
      <input
        type="text"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        placeholder="Search by listing title or ID..."
        className="w-full max-w-md px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
        style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }}
      />

      {/* Table */}
      <div className="surface rounded-lg overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr style={{ borderBottom: "1px solid var(--border-default)" }}>
              <th className="text-left px-4 py-3 font-semibold text-xs uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>Listing</th>
              <th className="text-right px-3 py-3 font-semibold text-xs uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>
                <SortHeader label="Views" field="page_views" icon={<Eye className="w-3 h-3" />} />
              </th>
              <th className="text-right px-3 py-3 font-semibold text-xs uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>
                <SortHeader label="Visitors" field="unique_visitors" icon={<Users className="w-3 h-3" />} />
              </th>
              <th className="text-right px-3 py-3 font-semibold text-xs uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>
                <SortHeader label="Watchlist" field="watchlist_adds" icon={<Heart className="w-3 h-3" />} />
              </th>
              <th className="text-right px-3 py-3 font-semibold text-xs uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>
                <SortHeader label="Messages" field="message_inquiries" icon={<MessageSquare className="w-3 h-3" />} />
              </th>
              <th className="text-right px-3 py-3 font-semibold text-xs uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>
                <SortHeader label="CVR" field="conversion_rate" icon={<TrendingUp className="w-3 h-3" />} />
              </th>
              <th className="text-right px-3 py-3 font-semibold text-xs uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>
                <SortHeader label="Avg Time" field="avg_time_on_page" icon={<Clock className="w-3 h-3" />} />
              </th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <tr><td colSpan={7} className="px-4 py-8 text-center" style={{ color: "var(--text-tertiary)" }}>Loading traffic data...</td></tr>
            ) : sorted.length === 0 ? (
              <tr><td colSpan={7} className="px-4 py-8 text-center" style={{ color: "var(--text-tertiary)" }}>No listings found.</td></tr>
            ) : (
              sorted.map((r) => (
                <tr key={r.listing_id} className="hover:bg-black/[0.02] transition-colors" style={{ borderBottom: "1px solid var(--border-subtle)" }}>
                  <td className="px-4 py-3">
                    <p className="font-medium" style={{ color: "var(--text-primary)" }}>{r.title}</p>
                    <p className="text-xs font-mono" style={{ color: "var(--text-tertiary)" }}>{r.listing_id}</p>
                  </td>
                  <td className="text-right px-3 py-3 font-mono" style={{ color: "var(--text-secondary)" }}>{r.page_views.toLocaleString()}</td>
                  <td className="text-right px-3 py-3 font-mono" style={{ color: "var(--text-secondary)" }}>{r.unique_visitors.toLocaleString()}</td>
                  <td className="text-right px-3 py-3 font-mono" style={{ color: "var(--text-secondary)" }}>{r.watchlist_adds.toLocaleString()}</td>
                  <td className="text-right px-3 py-3 font-mono" style={{ color: "var(--text-secondary)" }}>{r.message_inquiries}</td>
                  <td className="text-right px-3 py-3">
                    <span className="font-medium" style={{ color: r.conversion_rate > 5 ? "var(--color-success)" : "var(--text-secondary)" }}>
                      {r.conversion_rate.toFixed(1)}%
                    </span>
                  </td>
                  <td className="text-right px-3 py-3 font-mono" style={{ color: "var(--text-secondary)" }}>{r.avg_time_on_page}s</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
