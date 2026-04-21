'use client';
import { useState, useEffect, useCallback } from 'react';
import Link from 'next/link';
import api from '@/lib/api';
import { Search, Plus, MessageSquare, DollarSign, Clock, ChevronRight, Loader2, PackageSearch } from 'lucide-react';

interface ProductRequest {
  id: string;
  user_id: string;
  user_name: string;
  title: string;
  description?: string;
  category_name?: string;
  budget?: number;
  currency: string;
  status: string;
  response_count: number;
  created_at: string;
  expires_at?: string;
}

interface Pagination {
  total: number;
  page: number;
  per_page: number;
  pages: number;
}

export default function ProductRequestsPage() {
  const [requests, setRequests] = useState<ProductRequest[]>([]);
  const [pagination, setPagination] = useState<Pagination | null>(null);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [searchInput, setSearchInput] = useState('');

  const fetchRequests = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page: String(page), per_page: '20' });
      if (search) params.set('q', search);
      const res = await api.get(`/api/v1/requests?${params}`);
      setRequests(res.data.data ?? []);
      setPagination(res.data.pagination);
    } catch (err) {
      console.error('Failed to load requests', err);
    } finally {
      setLoading(false);
    }
  }, [page, search]);

  useEffect(() => { fetchRequests(); }, [fetchRequests]);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setSearch(searchInput);
    setPage(1);
  };

  const formatBudget = (budget: number, currency: string) =>
    new Intl.NumberFormat('en-US', { style: 'currency', currency }).format(budget);

  const formatDate = (dateStr: string) =>
    new Date(dateStr).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-4xl mx-auto px-4 py-8">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 flex items-center gap-2">
              <PackageSearch className="w-6 h-6 text-blue-600" />
              Product Requests
            </h1>
            <p className="text-sm text-gray-500 mt-1">
              Buyers looking for specific items — fulfill a request to connect directly.
            </p>
          </div>
          <Link
            href="/requests/new"
            className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-xl font-medium text-sm transition-colors"
          >
            <Plus className="w-4 h-4" />
            Post a Request
          </Link>
        </div>

        {/* Search */}
        <form onSubmit={handleSearch} className="flex gap-2 mb-6">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input
              value={searchInput}
              onChange={e => setSearchInput(e.target.value)}
              placeholder="Search requests…"
              className="w-full pl-9 pr-4 py-2.5 border border-gray-200 rounded-xl bg-white text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <button
            type="submit"
            className="px-4 py-2.5 bg-gray-900 hover:bg-gray-700 text-white rounded-xl text-sm font-medium"
          >
            Search
          </button>
        </form>

        {/* List */}
        {loading ? (
          <div className="flex justify-center py-12">
            <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
          </div>
        ) : requests.length === 0 ? (
          <div className="text-center py-16 text-gray-500">
            <PackageSearch className="w-12 h-12 mx-auto mb-3 text-gray-300" />
            <p className="font-medium text-gray-700">No open requests yet</p>
            <p className="text-sm mt-1">Be the first to post what you're looking for.</p>
            <Link href="/requests/new" className="inline-block mt-4 text-blue-600 hover:underline text-sm font-medium">
              Post a request →
            </Link>
          </div>
        ) : (
          <div className="space-y-3">
            {requests.map(req => (
              <Link
                key={req.id}
                href={`/requests/${req.id}`}
                className="block bg-white rounded-2xl border border-gray-200 p-5 hover:border-blue-300 hover:shadow-sm transition-all group"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex-1 min-w-0">
                    <h3 className="font-semibold text-gray-900 group-hover:text-blue-600 transition-colors truncate">
                      {req.title}
                    </h3>
                    {req.description && (
                      <p className="text-sm text-gray-500 mt-1 line-clamp-2">{req.description}</p>
                    )}
                    <div className="flex flex-wrap items-center gap-3 mt-3 text-xs text-gray-500">
                      <span className="font-medium text-gray-700">{req.user_name}</span>
                      {req.category_name && (
                        <span className="bg-gray-100 px-2 py-0.5 rounded-full">{req.category_name}</span>
                      )}
                      {req.budget && (
                        <span className="flex items-center gap-1 text-green-700 font-medium">
                          <DollarSign className="w-3 h-3" />
                          Budget: {formatBudget(req.budget, req.currency)}
                        </span>
                      )}
                      <span className="flex items-center gap-1">
                        <Clock className="w-3 h-3" />
                        {formatDate(req.created_at)}
                      </span>
                      <span className="flex items-center gap-1">
                        <MessageSquare className="w-3 h-3" />
                        {req.response_count} {req.response_count === 1 ? 'response' : 'responses'}
                      </span>
                    </div>
                  </div>
                  <ChevronRight className="w-5 h-5 text-gray-400 group-hover:text-blue-500 flex-shrink-0 mt-1" />
                </div>
              </Link>
            ))}
          </div>
        )}

        {/* Pagination */}
        {pagination && pagination.pages > 1 && (
          <div className="flex justify-center gap-2 mt-8">
            <button
              onClick={() => setPage(p => Math.max(1, p - 1))}
              disabled={page === 1}
              className="px-4 py-2 text-sm border border-gray-200 rounded-xl bg-white disabled:opacity-40 hover:bg-gray-50"
            >
              Previous
            </button>
            <span className="px-4 py-2 text-sm text-gray-600">
              Page {page} of {pagination.pages}
            </span>
            <button
              onClick={() => setPage(p => Math.min(pagination.pages, p + 1))}
              disabled={page === pagination.pages}
              className="px-4 py-2 text-sm border border-gray-200 rounded-xl bg-white disabled:opacity-40 hover:bg-gray-50"
            >
              Next
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
