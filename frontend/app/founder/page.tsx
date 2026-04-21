'use client';
import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/store/auth';
import { PERMISSIONS, hasPermission, isInternalRole } from '@/lib/permissions';
import api from '@/lib/api';
import {
  Users, Package, ShoppingCart, DollarSign, AlertTriangle,
  TrendingUp, Calendar, Loader2, BarChart3, Shield
} from 'lucide-react';

interface PlatformMetrics {
  total_users: number;
  new_users_7d: number;
  new_users_30d: number;
  total_listings: number;
  active_listings: number;
  total_orders: number;
  gmv_30d: number;
  total_revenue: number;
  open_disputes: number;
  resolved_disputes: number;
  top_categories: Array<{
    category_id: string;
    category_name: string;
    revenue: number;
    listings: number;
  }>;
  daily_signups: Array<{ date: string; count: number }>;
  daily_orders: Array<{ date: string; count: number }>;
}

export default function FounderDashboard() {
  const router = useRouter();
  const { user, isAuthenticated } = useAuthStore();
  const [mounted, setMounted] = useState(false);
  const [metrics, setMetrics] = useState<PlatformMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [forbidden, setForbidden] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (mounted && !isAuthenticated) {
      router.push('/login?redirect=/founder');
    }
  }, [mounted, isAuthenticated, router]);

  useEffect(() => {
    if (!isAuthenticated || !user) return;

    const role = user.role;
    if (!isInternalRole(role) || !hasPermission(role, PERMISSIONS.ADMIN_DASHBOARD_READ)) {
      setForbidden(true);
      setLoading(false);
      return;
    }

    const fetchMetrics = async () => {
      setLoading(true);
      try {
        const res = await api.get('/analytics/platform');
        setMetrics(res.data);
      } catch {
        setForbidden(true);
      } finally {
        setLoading(false);
      }
    };

    fetchMetrics();
  }, [isAuthenticated, user]);

  if (!mounted || !isAuthenticated) return null;

  if (forbidden) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center px-4">
        <div className="max-w-md w-full bg-white rounded-2xl shadow-sm p-8 text-center">
          <div className="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <Shield size={32} className="text-red-600" />
          </div>
          <h1 className="text-2xl font-bold text-gray-900 mb-2">Access Denied</h1>
          <p className="text-gray-500 mb-6">
            You don't have permission to view this page. This dashboard is only available to administrators.
          </p>
          <button
            onClick={() => router.push('/')}
            className="inline-flex items-center gap-2 bg-[#0071CE] text-white px-6 py-3 rounded-xl font-semibold hover:bg-[#005ba3] transition-colors"
          >
            Go to Home
          </button>
        </div>
      </div>
    );
  }

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-AE', {
      style: 'currency',
      currency: 'AED',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value);
  };

  const formatNumber = (value: number) => {
    return new Intl.NumberFormat('en-US').format(value);
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-gradient-to-r from-[#1e3a5f] to-[#0f1f33] text-white py-8 px-4">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center gap-2 text-blue-200 text-sm mb-2">
            <BarChart3 size={16} />
            <span>Admin Dashboard</span>
          </div>
          <h1 className="text-3xl font-bold">Platform Overview</h1>
          <p className="text-blue-200 mt-1">Business metrics and performance insights</p>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 py-8">
        {loading ? (
          <div className="flex items-center justify-center py-20">
            <Loader2 size={32} className="animate-spin text-[#0071CE]" />
          </div>
        ) : metrics ? (
          <>
            {/* Key Metrics Grid */}
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4 mb-8">
              <MetricCard
                title="Total Users"
                value={formatNumber(metrics.total_users)}
                subtitle={`+${metrics.new_users_7d} this week`}
                icon={<Users size={20} />}
                trend={metrics.new_users_7d > 0 ? 'up' : 'neutral'}
              />
              <MetricCard
                title="Active Listings"
                value={formatNumber(metrics.active_listings)}
                subtitle={`${formatNumber(metrics.total_listings)} total`}
                icon={<Package size={20} />}
              />
              <MetricCard
                title="Total Orders"
                value={formatNumber(metrics.total_orders)}
                icon={<ShoppingCart size={20} />}
              />
              <MetricCard
                title="Total Revenue"
                value={formatCurrency(metrics.total_revenue)}
                subtitle="Platform fees collected"
                icon={<DollarSign size={20} />}
                highlight
              />
            </div>

            {/* Secondary Metrics */}
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4 mb-8">
              <MetricCard
                title="GMV (30 days)"
                value={formatCurrency(metrics.gmv_30d)}
                subtitle="Gross Merchandise Value"
                icon={<TrendingUp size={20} />}
              />
              <MetricCard
                title="New Users (30d)"
                value={formatNumber(metrics.new_users_30d)}
                icon={<Calendar size={20} />}
              />
              <MetricCard
                title="Open Disputes"
                value={formatNumber(metrics.open_disputes)}
                subtitle={`${formatNumber(metrics.resolved_disputes)} resolved`}
                icon={<AlertTriangle size={20} />}
                alert={metrics.open_disputes > 5}
              />
              <MetricCard
                title="Listing Activity"
                value={`${Math.round((metrics.active_listings / Math.max(metrics.total_listings, 1)) * 100)}%`}
                subtitle="Active rate"
                icon={<Package size={20} />}
              />
            </div>

            {/* Charts Row */}
            <div className="grid gap-6 lg:grid-cols-2 mb-8">
              {/* Daily Signups Chart */}
              <div className="bg-white rounded-xl border border-gray-200 p-6">
                <h3 className="text-sm font-semibold text-gray-900 mb-4">Daily Signups (Last 7 Days)</h3>
                <div className="h-40 flex items-end gap-2">
                  {metrics.daily_signups.map((d, i) => {
                    const maxCount = Math.max(...metrics.daily_signups.map(x => x.count), 1);
                    const height = (d.count / maxCount) * 100;
                    return (
                      <div key={i} className="flex-1 flex flex-col items-center gap-1">
                        <div
                          className="w-full bg-[#0071CE] rounded-t transition-all"
                          style={{ height: `${Math.max(height, 4)}%` }}
                        />
                        <span className="text-xs text-gray-400">
                          {new Date(d.date).toLocaleDateString('en-US', { weekday: 'short' })}
                        </span>
                      </div>
                    );
                  })}
                </div>
              </div>

              {/* Daily Orders Chart */}
              <div className="bg-white rounded-xl border border-gray-200 p-6">
                <h3 className="text-sm font-semibold text-gray-900 mb-4">Daily Orders (Last 7 Days)</h3>
                <div className="h-40 flex items-end gap-2">
                  {metrics.daily_orders.map((d, i) => {
                    const maxCount = Math.max(...metrics.daily_orders.map(x => x.count), 1);
                    const height = (d.count / maxCount) * 100;
                    return (
                      <div key={i} className="flex-1 flex flex-col items-center gap-1">
                        <div
                          className="w-full bg-green-500 rounded-t transition-all"
                          style={{ height: `${Math.max(height, 4)}%` }}
                        />
                        <span className="text-xs text-gray-400">
                          {new Date(d.date).toLocaleDateString('en-US', { weekday: 'short' })}
                        </span>
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>

            {/* Top Categories Table */}
            <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
              <div className="px-6 py-4 border-b border-gray-100">
                <h3 className="text-sm font-semibold text-gray-900">Top Categories by Revenue</h3>
              </div>
              {metrics.top_categories.length > 0 ? (
                <table className="w-full">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="text-left text-xs font-medium text-gray-500 px-6 py-3">Category</th>
                      <th className="text-right text-xs font-medium text-gray-500 px-6 py-3">Listings</th>
                      <th className="text-right text-xs font-medium text-gray-500 px-6 py-3">Revenue</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {metrics.top_categories.map((cat, i) => (
                      <tr key={i} className="hover:bg-gray-50">
                        <td className="px-6 py-4 text-sm font-medium text-gray-900">{cat.category_name}</td>
                        <td className="px-6 py-4 text-sm text-gray-500 text-right">{formatNumber(cat.listings)}</td>
                        <td className="px-6 py-4 text-sm text-gray-900 text-right font-semibold">{formatCurrency(cat.revenue)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ) : (
                <div className="px-6 py-8 text-center text-gray-500 text-sm">
                  No category data available yet
                </div>
              )}
            </div>
          </>
        ) : null}
      </div>
    </div>
  );
}

function MetricCard({
  title,
  value,
  subtitle,
  icon,
  trend,
  highlight,
  alert,
}: {
  title: string;
  value: string;
  subtitle?: string;
  icon: React.ReactNode;
  trend?: 'up' | 'down' | 'neutral';
  highlight?: boolean;
  alert?: boolean;
}) {
  return (
    <div className={`bg-white rounded-xl border p-5 ${highlight ? 'border-green-200 bg-green-50' : alert ? 'border-red-200 bg-red-50' : 'border-gray-200'}`}>
      <div className="flex items-start justify-between mb-3">
        <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${highlight ? 'bg-green-100 text-green-600' : alert ? 'bg-red-100 text-red-600' : 'bg-gray-100 text-gray-600'}`}>
          {icon}
        </div>
        {trend === 'up' && (
          <span className="text-xs text-green-600 font-medium flex items-center gap-1">
            <TrendingUp size={12} /> Up
          </span>
        )}
      </div>
      <p className="text-2xl font-bold text-gray-900">{value}</p>
      {subtitle && <p className="text-xs text-gray-500 mt-1">{subtitle}</p>}
      <p className="text-xs text-gray-400 mt-2">{title}</p>
    </div>
  );
}
