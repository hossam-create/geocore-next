'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/auth';

interface OrderItem {
  title: string;
}

interface Order {
  id: string;
  status: string;
  created_at: string;
  total: number;
  currency: string;
  items?: OrderItem[];
}

interface OrdersResponse {
  data: Order[];
}

const REASON_OPTIONS = [
  { value: 'item_not_received', label: 'Item not received' },
  { value: 'item_not_as_described', label: 'Item not as described' },
  { value: 'item_damaged', label: 'Item damaged' },
  { value: 'wrong_item', label: 'Wrong item received' },
  { value: 'seller_not_responding', label: 'Seller not responding' },
  { value: 'payment_issue', label: 'Payment issue' },
  { value: 'fraud', label: 'Fraud concern' },
  { value: 'other', label: 'Other' },
] as const;

export default function NewDisputePage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();

  const [orderID, setOrderID] = useState('');
  const [reason, setReason] = useState<(typeof REASON_OPTIONS)[number]['value']>('item_not_as_described');
  const [evidence, setEvidence] = useState('');
  const [successId, setSuccessId] = useState('');
  const [errorMsg, setErrorMsg] = useState('');

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login?next=/disputes/new');
    }
  }, [isAuthenticated, router]);

  const { data: ordersData, isLoading: ordersLoading } = useQuery<OrdersResponse>({
    queryKey: ['orders', 'buyer', 'dispute-form'],
    queryFn: async () => {
      const res = await api.get('/orders?page=1&limit=100');
      return res.data as OrdersResponse;
    },
    enabled: isAuthenticated,
    retry: false,
  });

  const orders = useMemo(() => {
    const raw = ordersData?.data ?? [];
    return raw.filter((o) => !['cancelled', 'refunded'].includes((o.status || '').toLowerCase()));
  }, [ordersData]);

  const submitDispute = useMutation({
    mutationFn: async () => {
      const res = await api.post('/disputes', {
        order_id: orderID,
        reason,
        evidence,
      });
      return res.data?.data?.id as string | undefined;
    },
    onSuccess: (id) => {
      setSuccessId(id || 'created');
      setErrorMsg('');
    },
    onError: (err: any) => {
      setErrorMsg(err?.response?.data?.message || 'Could not open dispute.');
    },
  });

  if (!isAuthenticated) return null;

  if (successId) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-10">
        <div className="rounded-2xl border border-green-200 bg-green-50 p-6">
          <h1 className="text-xl font-bold text-green-800">Dispute Submitted</h1>
          <p className="mt-2 text-sm text-green-700">Dispute ID: {successId}</p>
          <p className="mt-1 text-sm text-green-700">Our team will review your evidence. Keep notifications enabled for updates.</p>
          <div className="mt-4 flex gap-3">
            <Link href="/disputes" className="rounded-xl bg-green-600 px-4 py-2 text-sm font-semibold text-white hover:bg-green-700">
              View My Disputes
            </Link>
            <Link href="/orders" className="rounded-xl border border-green-300 px-4 py-2 text-sm font-semibold text-green-700 hover:bg-green-100">
              Back to Orders
            </Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl px-4 py-10">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Open a New Dispute</h1>
        <p className="text-sm text-gray-500">Submit your issue with supporting details so we can review it quickly.</p>
      </div>

      <div className="rounded-2xl border border-gray-200 bg-white p-5 space-y-4">
        <div>
          <label className="mb-1 block text-sm font-semibold text-gray-700">Order</label>
          <select
            value={orderID}
            onChange={(e) => setOrderID(e.target.value)}
            className="w-full rounded-xl border border-gray-200 px-3 py-2.5 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
          >
            <option value="">Select order</option>
            {orders.map((order) => (
              <option key={order.id} value={order.id}>
                #{order.id.slice(0, 8)} · {order.items?.[0]?.title || 'Order item'} · {new Date(order.created_at).toLocaleDateString()}
              </option>
            ))}
          </select>
          {ordersLoading && <p className="mt-1 text-xs text-gray-400">Loading your orders…</p>}
        </div>

        <div>
          <label className="mb-1 block text-sm font-semibold text-gray-700">Reason</label>
          <select
            value={reason}
            onChange={(e) => setReason(e.target.value as (typeof REASON_OPTIONS)[number]['value'])}
            className="w-full rounded-xl border border-gray-200 px-3 py-2.5 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
          >
            {REASON_OPTIONS.map((r) => (
              <option key={r.value} value={r.value}>{r.label}</option>
            ))}
          </select>
        </div>

        <div>
          <label className="mb-1 block text-sm font-semibold text-gray-700">Evidence</label>
          <textarea
            value={evidence}
            onChange={(e) => setEvidence(e.target.value)}
            rows={6}
            placeholder="Describe what happened and include evidence details (minimum 20 characters)."
            className="w-full rounded-xl border border-gray-200 px-3 py-2.5 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
          />
        </div>

        {errorMsg && <p className="text-sm text-red-500">{errorMsg}</p>}

        <button
          onClick={() => submitDispute.mutate()}
          disabled={submitDispute.isPending || !orderID || evidence.trim().length < 20}
          className="w-full rounded-xl bg-[#0071CE] px-4 py-3 text-sm font-bold text-white hover:bg-[#005ba3] disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {submitDispute.isPending ? 'Submitting…' : 'Submit Dispute'}
        </button>
      </div>
    </div>
  );
}
