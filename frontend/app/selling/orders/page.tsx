'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
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
  buyer_id: string;
  created_at: string;
  tracking_number?: string;
  carrier?: string;
  items?: OrderItem[];
}

interface OrdersResponse {
  data: Order[];
}

export default function SellerOrdersPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const { isAuthenticated } = useAuthStore();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [trackingNumber, setTrackingNumber] = useState('');
  const [carrier, setCarrier] = useState('');
  const [notice, setNotice] = useState('');

  useEffect(() => {
    if (!isAuthenticated) router.push('/login?next=/selling/orders');
  }, [isAuthenticated, router]);

  const { data, isLoading, isError } = useQuery<OrdersResponse>({
    queryKey: ['orders', 'seller'],
    queryFn: async () => {
      const res = await api.get('/orders/selling?page=1&limit=50');
      return res.data as OrdersResponse;
    },
    enabled: isAuthenticated,
    retry: false,
  });

  const orders = data?.data ?? [];
  const selectedOrder = useMemo(
    () => orders.find((order) => order.id === selectedId) ?? null,
    [orders, selectedId]
  );

  const confirmMutation = useMutation({
    mutationFn: async (orderID: string) => api.patch(`/orders/${orderID}/confirm`, {}),
    onSuccess: () => {
      setNotice('Order confirmed.');
      qc.invalidateQueries({ queryKey: ['orders', 'seller'] });
    },
    onError: () => setNotice('Failed to confirm order.'),
  });

  const shipMutation = useMutation({
    mutationFn: async (payload: { orderID: string; tracking_number: string; carrier: string }) =>
      api.patch(`/orders/${payload.orderID}/ship`, {
        tracking_number: payload.tracking_number,
        carrier: payload.carrier,
      }),
    onSuccess: () => {
      setNotice('Order marked as shipped.');
      setTrackingNumber('');
      setCarrier('');
      qc.invalidateQueries({ queryKey: ['orders', 'seller'] });
    },
    onError: () => setNotice('Failed to mark order as shipped.'),
  });

  if (!isAuthenticated) return null;

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">My Sales Orders</h1>
          <p className="text-sm text-gray-500">Confirm and ship incoming orders from buyers.</p>
        </div>
        <Link href="/orders" className="rounded-xl border border-gray-200 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50">
          View My Purchases
        </Link>
      </div>

      <div className="grid gap-5 lg:grid-cols-3">
        <div className="overflow-hidden rounded-2xl border border-gray-100 bg-white shadow-sm lg:col-span-2">
          {isLoading ? (
            <div className="space-y-2 p-5">{[1, 2, 3].map((i) => <div key={i} className="h-14 animate-pulse rounded-xl bg-gray-100" />)}</div>
          ) : isError ? (
            <div className="p-8 text-center text-sm text-red-500">Could not load seller orders.</div>
          ) : orders.length === 0 ? (
            <div className="p-10 text-center">
              <p className="text-base font-semibold text-gray-700">No incoming orders yet</p>
              <p className="mt-1 text-sm text-gray-500">Once buyers purchase your listings, they will appear here.</p>
            </div>
          ) : (
            <ul className="divide-y divide-gray-100">
              {orders.map((order) => (
                <li key={order.id} className="px-5 py-4 hover:bg-gray-50">
                  <button
                    onClick={() => setSelectedId(order.id)}
                    className="flex w-full items-center justify-between gap-4 text-left"
                  >
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-semibold text-[#0071CE]">Order #{order.id.slice(0, 8)}</p>
                      <p className="mt-1 truncate text-sm text-gray-700">
                        {order.items?.[0]?.title ?? 'Order item'}
                        {order.items && order.items.length > 1 ? ` +${order.items.length - 1} more` : ''}
                      </p>
                      <p className="mt-1 text-xs text-gray-400">{new Date(order.created_at).toLocaleString()}</p>
                    </div>
                    <div className="text-right">
                      <p className="text-sm font-bold text-gray-900">{formatPrice(order.total, order.currency || 'AED')}</p>
                      <div className="mt-1"><OrderStatusBadge status={order.status} /></div>
                    </div>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        <aside className="rounded-2xl border border-gray-200 bg-white p-5">
          <h2 className="mb-3 text-sm font-bold text-gray-800">Order Action Panel</h2>
          {!selectedOrder ? (
            <p className="text-sm text-gray-500">Select an order from the list to view details and actions.</p>
          ) : (
            <div className="space-y-4">
              <div className="rounded-xl border border-gray-100 bg-gray-50 p-3 text-sm">
                <p className="font-semibold text-gray-900">Order #{selectedOrder.id.slice(0, 8)}</p>
                <p className="mt-1 text-gray-600">{selectedOrder.items?.[0]?.title ?? 'Order item'}</p>
                <p className="mt-1 text-xs text-gray-500">Buyer ID: {selectedOrder.buyer_id}</p>
                <div className="mt-2"><OrderStatusBadge status={selectedOrder.status} /></div>
                <Link href={`/orders/${selectedOrder.id}`} className="mt-3 inline-block text-xs font-semibold text-[#0071CE] hover:underline">
                  Open full detail →
                </Link>
              </div>

              {selectedOrder.status === 'pending' && (
                <button
                  onClick={() => confirmMutation.mutate(selectedOrder.id)}
                  disabled={confirmMutation.isPending}
                  className="w-full rounded-xl bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-60"
                >
                  {confirmMutation.isPending ? 'Confirming…' : 'Confirm Order'}
                </button>
              )}

              {['confirmed', 'processing'].includes(selectedOrder.status) && (
                <div className="space-y-2 rounded-xl border border-gray-100 p-3">
                  <p className="text-sm font-semibold text-gray-800">Mark as Shipped</p>
                  <input
                    value={carrier}
                    onChange={(e) => setCarrier(e.target.value)}
                    placeholder="Carrier (DHL, Aramex...)"
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
                  />
                  <input
                    value={trackingNumber}
                    onChange={(e) => setTrackingNumber(e.target.value)}
                    placeholder="Tracking Number"
                    className="w-full rounded-lg border border-gray-200 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
                  />
                  <button
                    disabled={shipMutation.isPending || !carrier.trim() || !trackingNumber.trim()}
                    onClick={() =>
                      shipMutation.mutate({
                        orderID: selectedOrder.id,
                        carrier: carrier.trim(),
                        tracking_number: trackingNumber.trim(),
                      })
                    }
                    className="w-full rounded-xl bg-[#0071CE] px-3 py-2 text-sm font-semibold text-white hover:bg-[#005ba3] disabled:opacity-60"
                  >
                    {shipMutation.isPending ? 'Saving…' : 'Mark as Shipped'}
                  </button>
                </div>
              )}

              {notice && <p className="text-sm text-[#0071CE]">{notice}</p>}
            </div>
          )}
        </aside>
      </div>
    </div>
  );
}
