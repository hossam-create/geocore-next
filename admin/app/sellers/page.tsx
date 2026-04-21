"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import DataTable from "@/components/shared/DataTable";
import RightPanel from "@/components/shared/RightPanel";
import { mockSellers } from "@/lib/mockData";
import { sellersApi, usersApi } from "@/lib/api";
import { Trophy, AlertTriangle, Star, Download, ShieldBan, ShieldCheck, TrendingUp } from "lucide-react";
import { useToastStore } from "@/lib/toast";

type SellerRow = {
  id: string;
  username: string;
  gmv: number;
  avg_rating: number;
  dispute_rate: number;
  refund_rate: number;
  flag_count: number;
  total_sales: number;
  joined: string;
  status: string;
};

function normalizeSellers(payload: unknown): SellerRow[] {
  const box = payload as { data?: unknown[] } | unknown[] | null | undefined;
  const rows = Array.isArray(box) ? box : Array.isArray((box as { data?: unknown[] })?.data) ? (box as { data?: unknown[] }).data : [];
  return (rows as Record<string, unknown>[]).map((item) => ({
    id: String(item.id ?? ""),
    username: String(item.username ?? "Unknown"),
    gmv: Number(item.gmv ?? 0),
    avg_rating: Number(item.avg_rating ?? 0),
    dispute_rate: Number(item.dispute_rate ?? 0),
    refund_rate: Number(item.refund_rate ?? 0),
    flag_count: Number(item.flag_count ?? 0),
    total_sales: Number(item.total_sales ?? 0),
    joined: String(item.joined ?? new Date().toISOString()),
    status: String(item.status ?? "active"),
  })).filter((x) => x.id);
}

export default function SellersPage() {
  const [view, setView] = useState<"leaderboard" | "risk">("leaderboard");
  const [selected, setSelected] = useState<SellerRow | null>(null);
  const showToast = useToastStore((s) => s.showToast);
  const [actioning, setActioning] = useState(false);

  const { data: liveSellers, isLoading } = useQuery({
    queryKey: ["admin", "sellers", "top"],
    queryFn: async () => {
      try {
        const res = await sellersApi.top();
        return normalizeSellers(res);
      } catch { return []; }
    },
    retry: 1,
  });

  const sellers: SellerRow[] = liveSellers?.length ? liveSellers : (mockSellers as unknown as SellerRow[]);

  const leaderboard = [...sellers].sort((a, b) => b.gmv - a.gmv);
  const riskTable = [...sellers].sort((a, b) => b.flag_count - a.flag_count || b.dispute_rate - a.dispute_rate);
  const data = view === "leaderboard" ? leaderboard : riskTable;

  const handleExportCsv = () => {
    const header = "ID,Username,GMV,Avg Rating,Dispute Rate,Refund Rate,Flags,Sales,Status";
    const rows = sellers.map((s) => `${s.id},${s.username},${s.gmv},${s.avg_rating},${s.dispute_rate},${s.refund_rate},${s.flag_count},${s.total_sales},${s.status}`);
    const csv = [header, ...rows].join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "sellers-export.csv";
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div>
      <PageHeader title="Seller Hub" description="Leaderboard, risk table, and seller scorecards" />

      <div className="flex items-center gap-3 mb-4">
        <div className="flex rounded-lg overflow-hidden" style={{ border: "1px solid var(--border-default)" }}>
          <button
            className="px-4 py-2 text-sm font-medium transition-colors"
            style={{ background: view === "leaderboard" ? "var(--color-brand)" : "var(--bg-surface)", color: view === "leaderboard" ? "#fff" : "var(--text-secondary)" }}
            onClick={() => setView("leaderboard")}
          >
            <Trophy className="w-3.5 h-3.5 inline mr-1.5" />Leaderboard
          </button>
          <button
            className="px-4 py-2 text-sm font-medium transition-colors"
            style={{ background: view === "risk" ? "var(--color-danger)" : "var(--bg-surface)", color: view === "risk" ? "#fff" : "var(--text-secondary)" }}
            onClick={() => setView("risk")}
          >
            <AlertTriangle className="w-3.5 h-3.5 inline mr-1.5" />Risk Table
          </button>
        </div>
        <button
          onClick={handleExportCsv}
          className="ml-auto px-3 py-1.5 rounded-lg text-sm font-medium flex items-center gap-1.5"
          style={{ background: "var(--bg-surface)", border: "1px solid var(--border-default)", color: "var(--text-secondary)" }}
        >
          <Download className="w-3.5 h-3.5" />Export CSV
        </button>
      </div>

      <DataTable
        columns={
          view === "leaderboard"
            ? [
                { key: "username", label: "Seller" },
                { key: "gmv", label: "GMV", render: (s: SellerRow) => <span className="font-medium">${s.gmv.toLocaleString()}</span> },
                { key: "avg_rating", label: "Rating", render: (s: SellerRow) => (
                  <span className="flex items-center gap-1"><Star className="w-3 h-3" style={{ color: "#f59e0b" }} />{s.avg_rating}</span>
                )},
                { key: "total_sales", label: "Sales" },
                { key: "status", label: "Status", render: (s: SellerRow) => <StatusBadge status={s.status} dot /> },
              ]
            : [
                { key: "username", label: "Seller" },
                { key: "dispute_rate", label: "Dispute %", render: (s: SellerRow) => (
                  <span className="text-xs font-medium" style={{ color: s.dispute_rate > 0.1 ? "var(--color-danger)" : "var(--text-secondary)" }}>
                    {(s.dispute_rate * 100).toFixed(1)}%
                  </span>
                )},
                { key: "refund_rate", label: "Refund %", render: (s: SellerRow) => (
                  <span className="text-xs font-medium" style={{ color: s.refund_rate > 0.1 ? "var(--color-warning)" : "var(--text-secondary)" }}>
                    {(s.refund_rate * 100).toFixed(1)}%
                  </span>
                )},
                { key: "flag_count", label: "Flags", render: (s: SellerRow) => (
                  <span className="text-xs font-semibold" style={{ color: s.flag_count > 3 ? "var(--color-danger)" : "var(--text-secondary)" }}>{s.flag_count}</span>
                )},
                { key: "status", label: "Status", render: (s: SellerRow) => <StatusBadge status={s.status} dot /> },
              ]
        }
        data={data}
        isLoading={isLoading}
        loadingMessage="Loading sellers..."
        rowKey={(s) => s.id}
        onRowClick={setSelected}
        emptyMessage="No sellers found."
      />

      <RightPanel open={!!selected} onClose={() => setSelected(null)} title="Seller Scorecard">
        {selected && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Seller</p>
                <p className="text-sm font-medium" style={{ color: "var(--text-primary)" }}>{selected.username}</p>
              </div>
              <StatusBadge status={selected.status} dot />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="surface p-3 rounded-lg">
                <p className="text-xs" style={{ color: "var(--text-tertiary)" }}>GMV</p>
                <p className="text-lg font-bold" style={{ color: "var(--color-brand)" }}>${selected.gmv.toLocaleString()}</p>
              </div>
              <div className="surface p-3 rounded-lg">
                <p className="text-xs" style={{ color: "var(--text-tertiary)" }}>Rating</p>
                <p className="text-lg font-bold flex items-center gap-1" style={{ color: "var(--text-primary)" }}><Star className="w-4 h-4" style={{ color: "#f59e0b" }} />{selected.avg_rating}</p>
              </div>
              <div className="surface p-3 rounded-lg">
                <p className="text-xs" style={{ color: "var(--text-tertiary)" }}>Disputes</p>
                <p className="text-sm font-medium" style={{ color: selected.dispute_rate > 0.1 ? "var(--color-danger)" : "var(--text-primary)" }}>{(selected.dispute_rate * 100).toFixed(1)}%</p>
              </div>
              <div className="surface p-3 rounded-lg">
                <p className="text-xs" style={{ color: "var(--text-tertiary)" }}>Flags</p>
                <p className="text-sm font-medium" style={{ color: selected.flag_count > 3 ? "var(--color-danger)" : "var(--text-primary)" }}>{selected.flag_count}</p>
              </div>
            </div>

            {/* Performance Trend */}
            <div>
              <p className="text-xs font-medium uppercase tracking-wider mb-2 flex items-center gap-1" style={{ color: "var(--text-tertiary)" }}>
                <TrendingUp className="w-3 h-3" />Monthly Sales Trend
              </p>
              <div className="flex items-end gap-1 h-10">
                {[0.3, 0.5, 0.4, 0.7, 0.6, 0.8, 0.9, 0.75, 0.85, 1.0, 0.7, 0.9].map((v, i) => (
                  <div key={i} className="flex-1 rounded-sm" style={{ height: `${v * 100}%`, background: i >= 10 ? "var(--color-brand)" : "var(--border-default)", minWidth: 4 }} />
                ))}
              </div>
              <div className="flex justify-between mt-1">
                <span className="text-[9px]" style={{ color: "var(--text-tertiary)" }}>Jan</span>
                <span className="text-[9px]" style={{ color: "var(--text-tertiary)" }}>Dec</span>
              </div>
            </div>

            <div>
              <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Total Sales</p>
              <p className="text-sm" style={{ color: "var(--text-secondary)" }}>{selected.total_sales} transactions</p>
            </div>
            <div>
              <p className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--text-tertiary)" }}>Joined</p>
              <p className="text-sm" style={{ color: "var(--text-secondary)" }}>{new Date(selected.joined).toLocaleDateString()}</p>
            </div>

            {/* Quick Actions */}
            <div className="pt-3 space-y-2" style={{ borderTop: "1px solid var(--border-default)" }}>
              <p className="text-xs font-semibold uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>Actions</p>
              {selected.status !== "suspended" ? (
                <button
                  disabled={actioning}
                  onClick={async () => {
                    if (!confirm(`Suspend ${selected.username}?`)) return;
                    setActioning(true);
                    try {
                      await usersApi.suspend(selected.id, new Date(Date.now() + 30 * 86400000).toISOString());
                      showToast({ type: "success", title: "Seller suspended", message: `${selected.username} has been suspended for 30 days.` });
                      setSelected(null);
                    } catch {
                      showToast({ type: "error", title: "Action failed", message: "Could not suspend seller." });
                    } finally { setActioning(false); }
                  }}
                  className="w-full flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-xs font-medium"
                  style={{ background: "rgba(239,68,68,0.1)", color: "var(--color-danger)" }}
                >
                  <ShieldBan className="w-3.5 h-3.5" />{actioning ? "Processing..." : "Suspend Seller"}
                </button>
              ) : (
                <button
                  disabled={actioning}
                  onClick={async () => {
                    setActioning(true);
                    try {
                      await usersApi.update(selected.id, { status: "active" });
                      showToast({ type: "success", title: "Seller reinstated", message: `${selected.username} is now active.` });
                      setSelected(null);
                    } catch {
                      showToast({ type: "error", title: "Action failed", message: "Could not reinstate seller." });
                    } finally { setActioning(false); }
                  }}
                  className="w-full flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-xs font-medium"
                  style={{ background: "rgba(34,197,94,0.1)", color: "var(--color-success)" }}
                >
                  <ShieldCheck className="w-3.5 h-3.5" />{actioning ? "Processing..." : "Reinstate Seller"}
                </button>
              )}
            </div>
          </div>
        )}
      </RightPanel>
    </div>
  );
}
