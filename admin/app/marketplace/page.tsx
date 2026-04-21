"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { addonsApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import { Download, Power, PowerOff, Trash2, Star, Search, Package, Settings } from "lucide-react";

interface Addon {
  id: string;
  slug: string;
  name: string;
  description: string;
  category: string;
  tags: string;
  author: string;
  version: string;
  download_count: number;
  avg_rating: number;
  rating_count: number;
  is_free: boolean;
  price: number;
  currency: string;
  is_verified: boolean;
  is_official: boolean;
  status: string;
  permissions: string;
  hooks: string;
  installed_at: string | null;
}

interface MarketplaceStats {
  total_addons: number;
  installed_count: number;
  enabled_count: number;
  total_downloads: number;
  categories: Array<{ Category: string; Count: number }>;
}

const STATUS_VARIANTS: Record<string, "success" | "warning" | "info" | "neutral"> = {
  available: "neutral",
  installed: "warning",
  enabled: "success",
  error: "danger" as unknown as "neutral",
};

const CATEGORY_COLORS: Record<string, string> = {
  payments: "bg-green-100 text-green-700",
  analytics: "bg-blue-100 text-blue-700",
  marketing: "bg-purple-100 text-purple-700",
  notifications: "bg-yellow-100 text-yellow-700",
  security: "bg-red-100 text-red-700",
  experience: "bg-indigo-100 text-indigo-700",
  engagement: "bg-pink-100 text-pink-700",
};

export default function MarketplacePage() {
  const qc = useQueryClient();
  const [search, setSearch] = useState("");
  const [filterCategory, setFilterCategory] = useState("");
  const [filterStatus, setFilterStatus] = useState("");

  const params: Record<string, string> = {};
  if (search) params.q = search;
  if (filterCategory) params.category = filterCategory;
  if (filterStatus) params.status = filterStatus;

  const { data: statsData } = useQuery({ queryKey: ["addon-stats"], queryFn: addonsApi.stats });
  const stats: MarketplaceStats | null = statsData ?? null;

  const { data, isLoading } = useQuery({
    queryKey: ["addons", params],
    queryFn: () => addonsApi.list(params),
  });

  const addons: Addon[] = (data?.data ?? data ?? []) as Addon[];

  const installMut = useMutation({
    mutationFn: addonsApi.install,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["addons"] }),
  });
  const uninstallMut = useMutation({
    mutationFn: addonsApi.uninstall,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["addons"] }),
  });
  const enableMut = useMutation({
    mutationFn: addonsApi.enable,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["addons"] }),
  });
  const disableMut = useMutation({
    mutationFn: addonsApi.disable,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["addons"] }),
  });

  const renderStars = (rating: number) => {
    const stars = [];
    for (let i = 1; i <= 5; i++) {
      stars.push(
        <Star key={i} className={`w-3 h-3 ${i <= Math.round(rating) ? "text-yellow-400 fill-yellow-400" : "text-slate-200"}`} />
      );
    }
    return <span className="flex items-center gap-0.5">{stars}</span>;
  };

  const renderActions = (addon: Addon) => {
    switch (addon.status) {
      case "available":
        return (
          <button onClick={() => installMut.mutate(addon.id)} disabled={installMut.isPending}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-white disabled:opacity-50"
            style={{ background: "var(--color-brand)" }}>
            <Download className="w-3.5 h-3.5" /> Install
          </button>
        );
      case "installed":
        return (
          <div className="flex gap-2">
            <button onClick={() => enableMut.mutate(addon.id)} disabled={enableMut.isPending}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-white disabled:opacity-50"
              style={{ background: "var(--color-success)" }}>
              <Power className="w-3.5 h-3.5" /> Enable
            </button>
            <button onClick={() => uninstallMut.mutate(addon.id)} disabled={uninstallMut.isPending}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border border-slate-200 text-slate-600 hover:bg-slate-50 disabled:opacity-50">
              <Trash2 className="w-3.5 h-3.5" /> Remove
            </button>
          </div>
        );
      case "enabled":
        return (
          <div className="flex gap-2">
            <button onClick={() => disableMut.mutate(addon.id)} disabled={disableMut.isPending}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border border-slate-200 text-slate-600 hover:bg-slate-50 disabled:opacity-50">
              <PowerOff className="w-3.5 h-3.5" /> Disable
            </button>
            <button onClick={() => uninstallMut.mutate(addon.id)} disabled={uninstallMut.isPending}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border border-red-200 text-red-600 hover:bg-red-50 disabled:opacity-50">
              <Trash2 className="w-3.5 h-3.5" /> Remove
            </button>
          </div>
        );
      default:
        return null;
    }
  };

  return (
    <div>
      <PageHeader title="Addon Marketplace" description="Browse, install, and manage platform addons & integrations" />

      {/* Stats Bar */}
      {stats && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
          <div className="p-4 rounded-xl border border-slate-200 bg-white">
            <div className="text-xs text-slate-500">Total Addons</div>
            <div className="text-xl font-bold text-slate-800">{stats.total_addons}</div>
          </div>
          <div className="p-4 rounded-xl border border-slate-200 bg-white">
            <div className="text-xs text-slate-500">Installed</div>
            <div className="text-xl font-bold text-slate-800">{stats.installed_count}</div>
          </div>
          <div className="p-4 rounded-xl border border-slate-200 bg-white">
            <div className="text-xs text-slate-500">Active</div>
            <div className="text-xl font-bold text-green-600">{stats.enabled_count}</div>
          </div>
          <div className="p-4 rounded-xl border border-slate-200 bg-white">
            <div className="text-xs text-slate-500">Total Downloads</div>
            <div className="text-xl font-bold text-slate-800">{stats.total_downloads.toLocaleString()}</div>
          </div>
        </div>
      )}

      {/* Filters */}
      <div className="flex flex-wrap gap-3 mb-4">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-2.5 w-4 h-4 text-slate-400" />
          <input placeholder="Search addons..." value={search} onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-9 border border-slate-200 rounded-lg px-3 py-2 text-sm bg-white" />
        </div>
        <select value={filterCategory} onChange={(e) => setFilterCategory(e.target.value)}
          className="border border-slate-200 rounded-lg px-3 py-2 text-sm bg-white">
          <option value="">All Categories</option>
          <option value="payments">Payments</option>
          <option value="analytics">Analytics</option>
          <option value="marketing">Marketing</option>
          <option value="notifications">Notifications</option>
          <option value="security">Security</option>
          <option value="experience">Experience</option>
          <option value="engagement">Engagement</option>
        </select>
        <select value={filterStatus} onChange={(e) => setFilterStatus(e.target.value)}
          className="border border-slate-200 rounded-lg px-3 py-2 text-sm bg-white">
          <option value="">All Status</option>
          <option value="available">Available</option>
          <option value="installed">Installed</option>
          <option value="enabled">Enabled</option>
        </select>
      </div>

      {/* Addon Grid */}
      {isLoading ? (
        <div className="text-center py-12 text-slate-400">Loading marketplace...</div>
      ) : addons.length === 0 ? (
        <div className="text-center py-12">
          <Package className="w-12 h-12 text-slate-300 mx-auto mb-3" />
          <p className="text-slate-400">No addons found</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {addons.map((addon) => (
            <div key={addon.id} className="p-5 rounded-xl border border-slate-200 bg-white hover:border-slate-300 transition-colors flex flex-col">
              {/* Header */}
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center gap-3">
                  <div className={`w-10 h-10 rounded-lg flex items-center justify-center text-lg ${CATEGORY_COLORS[addon.category] ?? "bg-slate-100 text-slate-600"}`}>
                    <Package className="w-5 h-5" />
                  </div>
                  <div>
                    <h3 className="text-sm font-semibold text-slate-800">{addon.name}</h3>
                    <p className="text-xs text-slate-400">by {addon.author} · v{addon.version}</p>
                  </div>
                </div>
                <StatusBadge status={addon.status} variant={STATUS_VARIANTS[addon.status] ?? "neutral"} />
              </div>

              {/* Description */}
              <p className="text-xs text-slate-500 mb-3 line-clamp-2 flex-1">{addon.description}</p>

              {/* Meta */}
              <div className="flex items-center gap-3 mb-3 text-xs text-slate-400">
                <span className={`px-2 py-0.5 rounded-full ${CATEGORY_COLORS[addon.category] ?? "bg-slate-100 text-slate-600"}`}>{addon.category}</span>
                {renderStars(addon.avg_rating)}
                <span>{addon.rating_count} reviews</span>
                <span>{addon.download_count} downloads</span>
              </div>

              {/* Badges */}
              <div className="flex items-center gap-2 mb-3">
                {addon.is_official && <span className="px-2 py-0.5 rounded-full bg-brand-50 text-brand-700 text-xs font-medium">Official</span>}
                {addon.is_verified && <span className="px-2 py-0.5 rounded-full bg-blue-50 text-blue-700 text-xs font-medium">Verified</span>}
                {addon.is_free ? (
                  <span className="px-2 py-0.5 rounded-full bg-green-50 text-green-700 text-xs font-medium">Free</span>
                ) : (
                  <span className="px-2 py-0.5 rounded-full bg-amber-50 text-amber-700 text-xs font-medium">{addon.currency} {addon.price.toFixed(2)}</span>
                )}
              </div>

              {/* Actions */}
              <div className="pt-3 border-t border-slate-100">
                {renderActions(addon)}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
