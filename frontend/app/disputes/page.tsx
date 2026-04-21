'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/auth';

interface Dispute {
  id: string;
  order_id?: string;
  reason: string;
  status: string;
  amount: number;
  currency: string;
  created_at: string;
}

interface DisputesResponse {
  data: Dispute[];
}

export default function MyDisputesPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login?next=/disputes');
    }
  }, [isAuthenticated, router]);

  const { data, isLoading, isError } = useQuery<DisputesResponse>({
    queryKey: ['disputes', 'me'],
    queryFn: async () => {
      const res = await api.get('/disputes?role=buyer&page=1');
      return res.data as DisputesResponse;
    },
    enabled: isAuthenticated,
    retry: false,
  });

  if (!isAuthenticated) return null;

  const disputes = data?.data ?? [];

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">My Disputes</h1>
          <p className="text-sm text-gray-500">Track open and resolved disputes for your orders.</p>
        </div>
        <Link href="/disputes/new" className="rounded-xl bg-[#0071CE] px-4 py-2 text-sm font-semibold text-white hover:bg-[#005ba3]">
          Open Dispute
        </Link>
      </div>

      <div className="overflow-hidden rounded-2xl border border-gray-100 bg-white shadow-sm">
        {isLoading ? (
          <div className="space-y-2 p-5">{[1, 2, 3].map((i) => <div key={i} className="h-14 animate-pulse rounded-xl bg-gray-100" />)}</div>
        ) : isError ? (
          <div className="p-8 text-center text-sm text-red-500">Could not load disputes right now.</div>
        ) : disputes.length === 0 ? (
          <div className="p-10 text-center">
            <p className="text-base font-semibold text-gray-700">No disputes submitted</p>
            <p className="mt-1 text-sm text-gray-500">If you face an order issue, you can open a dispute from here.</p>
            <Link href="/disputes/new" className="mt-4 inline-block rounded-xl bg-[#0071CE] px-4 py-2 text-sm font-semibold text-white hover:bg-[#005ba3]">
              Open Your First Dispute
            </Link>
          </div>
        ) : (
          <ul className="divide-y divide-gray-100">
            {disputes.map((d) => (
              <li key={d.id} className="px-5 py-4">
                <div className="flex items-center justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-semibold text-[#0071CE]">Dispute #{d.id.slice(0, 8)}</p>
                    <p className="mt-1 text-sm text-gray-700">Reason: {d.reason.replaceAll('_', ' ')}</p>
                    <p className="mt-1 text-xs text-gray-400">Created: {new Date(d.created_at).toLocaleString()}</p>
                  </div>
                  <div className="text-right">
                    <p className="text-sm font-bold text-gray-900">{d.amount} {d.currency || 'AED'}</p>
                    <span className="mt-1 inline-flex rounded-full border border-orange-200 bg-orange-50 px-2.5 py-1 text-xs font-semibold text-orange-700">
                      {d.status.replaceAll('_', ' ')}
                    </span>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
