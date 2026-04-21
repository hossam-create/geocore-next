'use client';
import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { useAuthStore } from '@/store/auth';
import api from '@/lib/api';
import { 
  DollarSign, Package, Eye, Star, TrendingUp, ArrowLeft,
  Calendar, BarChart3, Loader2
} from 'lucide-react';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  BarChart, Bar,
} from 'recharts';

interface SellerSummary {
  total_revenue: number;
  total_orders: number;
  active_listings: number;
  total_views: number;
  avg_rating: number;
}

interface RevenuePoint {
  date: string;
  amount: number;
}

interface ListingBreakdown {
  id: string;
  title: string;
  views: number;
  favorites: number;
  orders: number;
  conversion_rate: number;
}

const PERIODS = [
  { value: '7d', label: '7 Days' },
  { value: '30d', label: '30 Days' },
  { value: '90d', label: '90 Days' },
  { value: '1y', label: '1 Year' },
];

export default function SellerAnalyticsPage() {
  const router = useRouter();
  const { user, isAuthenticated } = useAuthStore();
  const [period, setPeriod] = useState('30d');
  const [summary, setSummary] = useState<SellerSummary | null>(null);
  const [revenueData, setRevenueData] = useState<RevenuePoint[]>([]);
  const [listings, setListings] = useState<ListingBreakdown[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (mounted && !isAuthenticated) {
      router.push('/login?redirect=/seller/analytics');
    }
  }, [mounted, isAuthenticated, router]);

  useEffect(() => {
    if (!isAuthenticated) return;
    
    const fetchData = async () => {
      setLoading(true);
      setError(null);
      try {
        const [summaryRes, revenueRes, listingsRes] = await Promise.all([
          api.get('/analytics/seller/summary'),
          api.get(`/analytics/seller/revenue?period=${period}`),
          api.get('/analytics/seller/listings'),
        ]);
        setSummary(summaryRes.data);
        setRevenueData(revenueRes.data.series || []);
        setListings(listingsRes.data || []);
      } catch (err: any) {
        setError(err.response?.data?.error || 'Failed to load analytics');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [isAuthenticated, period]);

  if (!mounted || !isAuthenticated) {
    return null;
  }

  const formatCurrency = (val: number) => `AED ${val.toLocaleString()}`;
  const formatNumber = (val: number) => val.toLocaleString();

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      {/* Header */}
      <div className="mb-6 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div className="flex items-center gap-3">
          <Link href="/dashboard" className="rounded-lg p-2 hover:bg-gray-100">
            <ArrowLeft size={20} className="text-gray-600" />
          </Link>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Seller Analytics</h1>
            <p className="text-sm text-gray-500">Track your performance and sales</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Calendar size={16} className="text-gray-400" />
          <select
            value={period}
            onChange={(e) => setPeriod(e.target.value)}
            className="rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-[#0071CE] focus:outline-none focus:ring-1 focus:ring-[#0071CE]"
          >
            {PERIODS.map((p) => (
              <option key={p.value} value={p.value}>{p.label}</option>
            ))}
          </select>
        </div>
      </div>

      {error && (
        <div className="mb-6 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {error}
        </div>
      )}

      {loading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 size={32} className="animate-spin text-[#0071CE]" />
        </div>
      ) : (
        <>
          {/* Metric Cards */}
          <div className="mb-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
            <div className="rounded-xl border border-gray-200 bg-white p-4">
              <div className="mb-2 flex items-center gap-2">
                <DollarSign size={16} className="text-emerald-600" />
                <span className="text-xs font-medium text-gray-500">Total Revenue</span>
              </div>
              <p className="text-xl font-bold text-gray-900">{formatCurrency(summary?.total_revenue || 0)}</p>
            </div>
            <div className="rounded-xl border border-gray-200 bg-white p-4">
              <div className="mb-2 flex items-center gap-2">
                <Package size={16} className="text-[#0071CE]" />
                <span className="text-xs font-medium text-gray-500">Total Orders</span>
              </div>
              <p className="text-xl font-bold text-gray-900">{formatNumber(summary?.total_orders || 0)}</p>
            </div>
            <div className="rounded-xl border border-gray-200 bg-white p-4">
              <div className="mb-2 flex items-center gap-2">
                <BarChart3 size={16} className="text-violet-600" />
                <span className="text-xs font-medium text-gray-500">Active Listings</span>
              </div>
              <p className="text-xl font-bold text-gray-900">{formatNumber(summary?.active_listings || 0)}</p>
            </div>
            <div className="rounded-xl border border-gray-200 bg-white p-4">
              <div className="mb-2 flex items-center gap-2">
                <Eye size={16} className="text-amber-600" />
                <span className="text-xs font-medium text-gray-500">Total Views</span>
              </div>
              <p className="text-xl font-bold text-gray-900">{formatNumber(summary?.total_views || 0)}</p>
            </div>
            <div className="rounded-xl border border-gray-200 bg-white p-4">
              <div className="mb-2 flex items-center gap-2">
                <Star size={16} className="text-yellow-500" />
                <span className="text-xs font-medium text-gray-500">Avg Rating</span>
              </div>
              <p className="text-xl font-bold text-gray-900">{(summary?.avg_rating || 0).toFixed(1)}</p>
            </div>
          </div>

          {/* Revenue Chart */}
          <div className="mb-8 rounded-xl border border-gray-200 bg-white p-5">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-base font-semibold text-gray-900">Revenue Trend</h2>
              <div className="flex items-center gap-2 text-xs text-gray-500">
                <TrendingUp size={14} className="text-emerald-600" />
                {period}
              </div>
            </div>
            <div className="h-64">
              {revenueData.length > 0 ? (
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={revenueData}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
                    <XAxis 
                      dataKey="date" 
                      tick={{ fontSize: 10 }} 
                      tickFormatter={(v) => v.slice(5)}
                      stroke="#9ca3af"
                    />
                    <YAxis 
                      tick={{ fontSize: 10 }} 
                      tickFormatter={(v) => `${v}`}
                      stroke="#9ca3af"
                    />
                    <Tooltip 
                      formatter={(value: number) => [formatCurrency(value), 'Revenue']}
                      labelFormatter={(label) => `Date: ${label}`}
                    />
                    <Line 
                      type="monotone" 
                      dataKey="amount" 
                      stroke="#0071CE" 
                      strokeWidth={2}
                      dot={false}
                    />
                  </LineChart>
                </ResponsiveContainer>
              ) : (
                <div className="flex h-full items-center justify-center text-sm text-gray-400">
                  No revenue data for this period
                </div>
              )}
            </div>
          </div>

          {/* Listings Breakdown */}
          <div className="rounded-xl border border-gray-200 bg-white p-5">
            <h2 className="mb-4 text-base font-semibold text-gray-900">Listings Performance</h2>
            {listings.length > 0 ? (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-200 text-left">
                      <th className="pb-3 font-medium text-gray-500">Listing</th>
                      <th className="pb-3 font-medium text-gray-500 text-right">Views</th>
                      <th className="pb-3 font-medium text-gray-500 text-right">Favorites</th>
                      <th className="pb-3 font-medium text-gray-500 text-right">Orders</th>
                      <th className="pb-3 font-medium text-gray-500 text-right">Conv. Rate</th>
                    </tr>
                  </thead>
                  <tbody>
                    {listings.slice(0, 10).map((listing) => (
                      <tr key={listing.id} className="border-b border-gray-100">
                        <td className="py-3">
                          <Link 
                            href={`/listings/${listing.id}`}
                            className="text-gray-900 hover:text-[#0071CE] hover:underline"
                          >
                            {listing.title}
                          </Link>
                        </td>
                        <td className="py-3 text-right text-gray-600">{formatNumber(listing.views)}</td>
                        <td className="py-3 text-right text-gray-600">{formatNumber(listing.favorites)}</td>
                        <td className="py-3 text-right text-gray-600">{formatNumber(listing.orders)}</td>
                        <td className="py-3 text-right text-gray-600">{listing.conversion_rate.toFixed(1)}%</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="py-8 text-center text-sm text-gray-400">
                No listings data yet
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
