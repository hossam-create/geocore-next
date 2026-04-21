'use client';

import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/auth';
import { formatPrice } from '@/lib/utils';
import { OrderStatusBadge } from '@/components/orders/OrderStatusBadge';
import { OrderTimeline } from '@/components/orders/OrderTimeline';
import { ShipmentTimeline, type ShipmentStatus } from '@/components/orders/ShipmentTimeline';
import { PriceBreakdown } from '@/components/listings/PriceBreakdown';
import { FeatureFlags } from '@/lib/featureFlags';

interface OrderItem {
  id: string;
  title: string;
  quantity: number;
  unit_price: number;
  total_price: number;
}

interface Order {
  id: string;
  buyer_id: string;
  seller_id: string;
  status: string;
  total: number;
  subtotal: number;
  currency: string;
  delivery_type?: string;
  platform_fee?: number;
  delivery_price?: number;
  tracking_number?: string;
  carrier?: string;
  notes?: string;
  created_at: string;
  confirmed_at?: string;
  shipped_at?: string;
  delivered_at?: string;
  completed_at?: string;
  cancelled_at?: string;
  items?: OrderItem[];
}

export default function OrderDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const qc = useQueryClient();
  const { user, isAuthenticated } = useAuthStore();
  const [message, setMessage] = useState('');

  useEffect(() => {
    if (!isAuthenticated) router.push(`/login?next=/orders/${params.id}`);
  }, [isAuthenticated, params.id, router]);

  const { data, isLoading, isError } = useQuery<{ data: Order }>({
    queryKey: ['order', params.id],
    queryFn: async () => {
      const res = await api.get(`/orders/${params.id}`);
      return res.data as { data: Order };
    },
    enabled: isAuthenticated && Boolean(params.id),
    retry: false,
  });

  const order = data?.data;
  const isBuyer = Boolean(order && user?.id && order.buyer_id === user.id);
  const canConfirmDelivery = Boolean(order && isBuyer && ['shipped', 'delivered'].includes(order.status));
  const isCrowdshipping = order?.delivery_type === 'crowdshipping';

  // Sprint 9: Fetch tracking events for crowdshipping orders
  const { data: trackingEvents } = useQuery({
    queryKey: ['tracking', params.id],
    queryFn: async () => {
      const res = await api.get(`/tracking/${params.id}`);
      return (res.data?.data ?? res.data) as Array<{
        status: ShipmentStatus;
        location?: string;
        note?: string;
        proof_image_url?: string;
        created_at: string;
      }>;
    },
    enabled: isAuthenticated && Boolean(params.id) && isCrowdshipping,
    retry: false,
  });

  const confirmDelivery = useMutation({
    mutationFn: async () => api.patch(`/orders/${params.id}/deliver`, {}),
    onSuccess: () => {
      setMessage('Delivery confirmed successfully.');
      qc.invalidateQueries({ queryKey: ['order', params.id] });
      qc.invalidateQueries({ queryKey: ['orders', 'buyer'] });
    },
    onError: () => setMessage('Unable to confirm delivery right now.'),
  });

  if (!isAuthenticated) return null;

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <div className="mb-5 flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Order Details</h1>
          <p className="text-sm text-gray-500">Review status, items, and shipping updates.</p>
        </div>
        <Link href="/orders" className="rounded-xl border border-gray-200 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50">
          Back to Orders
        </Link>
      </div>

      {isLoading ? (
        <div className="h-40 animate-pulse rounded-2xl bg-gray-100" />
      ) : isError || !order ? (
        <div className="rounded-2xl border border-red-100 bg-red-50 p-5 text-sm text-red-600">Could not load this order.</div>
      ) : (
        <div className="grid gap-5 lg:grid-cols-3">
          <div className="space-y-5 lg:col-span-2">
            <section className="rounded-2xl border border-gray-200 bg-white p-5">
              <div className="mb-3 flex items-center justify-between">
                <p className="text-sm font-semibold text-gray-900">Order #{order.id.slice(0, 8)}</p>
                <OrderStatusBadge status={order.status} />
              </div>
              <div className="grid gap-2 text-sm text-gray-600 sm:grid-cols-2">
                <p>Placed: <span className="font-medium text-gray-900">{new Date(order.created_at).toLocaleString()}</span></p>
                <p>Total: <span className="font-bold text-gray-900">{formatPrice(order.total, order.currency || 'AED')}</span></p>
                <p>Carrier: <span className="font-medium text-gray-900">{order.carrier || 'Not assigned yet'}</span></p>
                <p>Tracking: <span className="font-medium text-gray-900">{order.tracking_number || 'Not available yet'}</span></p>
              </div>
            </section>

            <section className="overflow-hidden rounded-2xl border border-gray-200 bg-white">
              <div className="border-b border-gray-100 px-5 py-3 text-sm font-bold text-gray-800">Items</div>
              <ul className="divide-y divide-gray-100">
                {(order.items ?? []).map((item) => (
                  <li key={item.id} className="flex items-center justify-between gap-3 px-5 py-3 text-sm">
                    <div>
                      <p className="font-medium text-gray-900">{item.title}</p>
                      <p className="text-xs text-gray-500">Qty: {item.quantity}</p>
                    </div>
                    <p className="font-semibold text-gray-900">{formatPrice(item.total_price, order.currency || 'AED')}</p>
                  </li>
                ))}
              </ul>
            </section>

            {canConfirmDelivery && (
              <section className="rounded-2xl border border-green-100 bg-green-50 p-5">
                <p className="text-sm text-green-800">Have you received the order in good condition?</p>
                <button
                  onClick={() => confirmDelivery.mutate()}
                  disabled={confirmDelivery.isPending}
                  className="mt-3 rounded-xl bg-green-600 px-4 py-2 text-sm font-semibold text-white hover:bg-green-700 disabled:opacity-60"
                >
                  {confirmDelivery.isPending ? 'Confirming…' : 'Confirm Delivery'}
                </button>
              </section>
            )}

            {message && <p className="text-sm text-[#0071CE]">{message}</p>}
          </div>

          <div className="space-y-5">
            {/* Sprint 9: Shipment Timeline for crowdshipping orders */}
            {FeatureFlags.shipmentTimeline && isCrowdshipping && trackingEvents && trackingEvents.length > 0 && (
              <ShipmentTimeline
                events={trackingEvents}
                currentStatus={trackingEvents[trackingEvents.length - 1]?.status}
                isBuyer={isBuyer}
                onConfirmDelivery={canConfirmDelivery ? () => confirmDelivery.mutate() : undefined}
              />
            )}

            {/* Original Order Timeline (always shown) */}
            <OrderTimeline order={order} />

            {/* Sprint 9: Price Breakdown */}
            {FeatureFlags.priceBreakdown && order.subtotal > 0 && (
              <PriceBreakdown
                itemPrice={order.subtotal}
                deliveryPrice={order.delivery_price}
                platformFee={order.platform_fee}
                total={order.total}
                currency={order.currency || 'AED'}
                escrowed
              />
            )}
          </div>
        </div>
      )}
    </div>
  );
}
