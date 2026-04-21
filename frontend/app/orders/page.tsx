'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/auth';
import { formatPrice } from '@/lib/utils';
import { OrderStatusBadge } from '@/components/orders/OrderStatusBadge';

interface OrderItem {
  id: string;
  title: string;
  quantity: number;
}

interface Order {
  id: string;
  status: string;
  total: number;
  currency: string;
  created_at: string;
  items?: OrderItem[];
}

interface OrdersResponse {
  data: Order[];
  pagination?: {
    total?: number;
    page?: number;
    limit?: number;
    pages?: number;
  };
}

const LIMIT = 10;

export default function OrdersPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [page, setPage] = useState(1);

  useEffect(() => {
    if (!isAuthenticated) router.push('/login?next=/orders');
  }, [isAuthenticated, router]);

  const { data, isLoading, isError } = useQuery<OrdersResponse>({
    queryKey: ['orders', 'buyer', page],
    queryFn: async () => {
      const res = await api.get(`/orders?page=${page}&limit=${LIMIT}`);
      return res.data as OrdersResponse;
    },
    enabled: isAuthenticated,
    retry: false,
  });

  if (!isAuthenticated) return null;

  const orders = data?.data ?? [];
  const pages = data?.pagination?.pages ?? 1;

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">My Orders</h1>
          <p className="text-sm text-gray-500">Track your purchases and delivery progress.</p>
        </div>
        <Link href="/selling/orders" className="rounded-xl border border-gray-200 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50">
          View My Sales
        </Link>
      </div>

      <div className="overflow-hidden rounded-2xl border border-gray-100 bg-white shadow-sm">
        {isLoading ? (
          <div className="space-y-2 p-5">
            {[1, 2, 3, 4].map((i) => <div key={i} className="h-16 animate-pulse rounded-xl bg-gray-100" />)}
          </div>
        ) : isError ? (
          <div className="p-8 text-center text-sm text-red-500">Failed to load your orders. Please try again.</div>
        ) : orders.length === 0 ? (
          <div className="p-10 text-center">
            <p className="text-base font-semibold text-gray-700">No orders yet</p>
            <p className="mt-1 text-sm text-gray-500">Start browsing listings and place your first order.</p>
            <Link href="/listings" className="mt-4 inline-block rounded-xl bg-[#0071CE] px-4 py-2 text-sm font-semibold text-white hover:bg-[#005ba3]">
              Browse Listings
            </Link>
          </div>
        ) : (
          <ul className="divide-y divide-gray-100">
            {orders.map((order) => (
              <li key={order.id} className="px-5 py-4 hover:bg-gray-50">
                <div className="flex items-center justify-between gap-4">
                  <div className="min-w-0 flex-1">
                    <Link href={`/orders/${order.id}`} className="text-sm font-semibold text-[#0071CE] hover:underline">
                      Order #{order.id.slice(0, 8)}
                    </Link>
                    <p className="mt-1 truncate text-sm text-gray-700">
                      {(order.items?.[0]?.title ?? 'Order item')}
                      {order.items && order.items.length > 1 ? ` +${order.items.length - 1} more` : ''}
                    </p>
                    <p className="mt-1 text-xs text-gray-400">Placed: {new Date(order.created_at).toLocaleString()}</p>
                  </div>
                  <div className="text-right">
                    <p className="text-sm font-bold text-gray-900">{formatPrice(order.total, order.currency || 'AED')}</p>
                    <div className="mt-1">
                      <OrderStatusBadge status={order.status} />
                    </div>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>

      {orders.length > 0 && pages > 1 && (
        <div className="mt-5 flex items-center justify-center gap-2">
          <button
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page <= 1}
            className="rounded-lg border border-gray-200 px-3 py-1.5 text-sm disabled:cursor-not-allowed disabled:opacity-50"
          >
            Prev
          </button>
          <span className="text-sm text-gray-500">Page {page} / {pages}</span>
          <button
            onClick={() => setPage((p) => Math.min(pages, p + 1))}
            disabled={page >= pages}
            className="rounded-lg border border-gray-200 px-3 py-1.5 text-sm disabled:cursor-not-allowed disabled:opacity-50"
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
}
