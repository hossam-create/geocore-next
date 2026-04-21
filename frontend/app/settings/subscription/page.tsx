'use client';
import { useState, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { useAuthStore } from '@/store/auth';
import api from '@/lib/api';
import {
  ArrowLeft, CheckCircle, AlertCircle, Loader2,
  Crown, Zap, Star, Building2, Calendar, XCircle
} from 'lucide-react';
import { Suspense } from 'react';

interface Plan {
  id: string;
  name: string;
  display_name: string;
  price_monthly: number;
  currency: string;
  listing_limit: number;
  features: string[];
}

interface Subscription {
  id: string;
  status: string;
  cancel_at_period_end: boolean;
  current_period_end?: string;
  current_period_start?: string;
}

interface SubData {
  subscription: Subscription | null;
  plan: Plan | null;
  on_free_plan: boolean;
}

const PLAN_ICONS: Record<string, React.ElementType> = {
  free: Star, basic: Zap, pro: Crown, enterprise: Building2,
};

function SubscriptionContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { isAuthenticated } = useAuthStore();

  const [data, setData] = useState<SubData | null>(null);
  const [loading, setLoading] = useState(true);
  const [cancelling, setCancelling] = useState(false);
  const [error, setError] = useState('');
  const [showActivated] = useState(searchParams.get('activated') === '1');

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login?redirect=/settings/subscription');
      return;
    }
    api.get('/api/v1/subscriptions/me')
      .then(res => setData(res.data))
      .catch(() => setError('Failed to load subscription'))
      .finally(() => setLoading(false));
  }, [isAuthenticated, router]);

  const handleCancel = async () => {
    if (!confirm('Cancel subscription at end of billing period?')) return;
    setCancelling(true);
    setError('');
    try {
      await api.delete('/api/v1/subscriptions/me');
      const res = await api.get('/api/v1/subscriptions/me');
      setData(res.data);
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } };
      setError(e?.response?.data?.error ?? 'Failed to cancel subscription');
    } finally {
      setCancelling(false);
    }
  };

  const formatDate = (d?: string) =>
    d ? new Date(d).toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' }) : '—';

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
      </div>
    );
  }

  const plan = data?.plan;
  const sub = data?.subscription;
  const Icon = plan ? (PLAN_ICONS[plan.name] ?? Star) : Star;

  return (
    <div className="space-y-6">
      {showActivated && (
        <div className="flex items-center gap-2 p-4 bg-green-50 border border-green-200 rounded-xl text-green-800 text-sm">
          <CheckCircle className="w-5 h-5 flex-shrink-0" />
          Subscription activated successfully!
        </div>
      )}

      {error && (
        <div className="flex items-center gap-2 p-4 bg-red-50 border border-red-200 rounded-xl text-red-700 text-sm">
          <AlertCircle className="w-5 h-5 flex-shrink-0" />
          {error}
        </div>
      )}

      {/* Current plan card */}
      <div className="bg-white rounded-2xl border border-gray-200 p-6">
        <h2 className="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-4">Current Plan</h2>

        <div className="flex items-start justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-blue-100 flex items-center justify-center">
              <Icon className="w-5 h-5 text-blue-600" />
            </div>
            <div>
              <p className="font-bold text-gray-900 text-lg">{plan?.display_name ?? 'Free'}</p>
              <p className="text-sm text-gray-500">
                {plan?.listing_limit === 0
                  ? 'Unlimited listings'
                  : `${plan?.listing_limit ?? 5} active listings`}
              </p>
            </div>
          </div>

          {!data?.on_free_plan && plan && (
            <div className="text-right">
              <p className="text-2xl font-extrabold text-gray-900">
                {plan.currency} {plan.price_monthly.toLocaleString()}
              </p>
              <p className="text-xs text-gray-400">/ month</p>
            </div>
          )}
        </div>

        {/* Status */}
        {sub && (
          <div className="mt-4 pt-4 border-t border-gray-100 space-y-2 text-sm">
            <div className="flex justify-between text-gray-600">
              <span>Status</span>
              <span className={`font-medium capitalize ${sub.status === 'active' ? 'text-green-600' : 'text-orange-600'}`}>
                {sub.cancel_at_period_end ? 'Cancels at period end' : sub.status}
              </span>
            </div>
            {sub.current_period_start && (
              <div className="flex justify-between text-gray-600">
                <span className="flex items-center gap-1"><Calendar className="w-3.5 h-3.5" /> Period start</span>
                <span>{formatDate(sub.current_period_start)}</span>
              </div>
            )}
            {sub.current_period_end && (
              <div className="flex justify-between text-gray-600">
                <span className="flex items-center gap-1"><Calendar className="w-3.5 h-3.5" />
                  {sub.cancel_at_period_end ? 'Cancels on' : 'Renews on'}
                </span>
                <span>{formatDate(sub.current_period_end)}</span>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Actions */}
      <div className="bg-white rounded-2xl border border-gray-200 p-6">
        <h2 className="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-4">Actions</h2>
        <div className="space-y-3">
          <Link
            href="/plans"
            className="flex items-center justify-between w-full px-4 py-3 border border-blue-200 bg-blue-50 hover:bg-blue-100 text-blue-700 rounded-xl text-sm font-medium transition-colors"
          >
            <span>View all plans</span>
            <span>→</span>
          </Link>

          {sub && !sub.cancel_at_period_end && sub.status === 'active' && (
            <button
              onClick={handleCancel}
              disabled={cancelling}
              className="flex items-center justify-between w-full px-4 py-3 border border-red-200 bg-red-50 hover:bg-red-100 text-red-700 rounded-xl text-sm font-medium transition-colors disabled:opacity-60"
            >
              <span className="flex items-center gap-2">
                <XCircle className="w-4 h-4" />
                {cancelling ? 'Cancelling…' : 'Cancel subscription'}
              </span>
              <span className="text-xs text-red-400">At period end</span>
            </button>
          )}

          {sub?.cancel_at_period_end && (
            <div className="p-3 bg-orange-50 border border-orange-200 rounded-xl text-sm text-orange-700">
              Your subscription will not renew after {formatDate(sub.current_period_end)}.
              You can resubscribe any time from the{' '}
              <Link href="/plans" className="underline font-medium">plans page</Link>.
            </div>
          )}
        </div>
      </div>

      {/* Plan features */}
      {plan && plan.features.length > 0 && (
        <div className="bg-white rounded-2xl border border-gray-200 p-6">
          <h2 className="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-4">Your plan includes</h2>
          <ul className="space-y-2">
            {plan.features.map((f, i) => (
              <li key={i} className="flex items-center gap-2 text-sm text-gray-700">
                <CheckCircle className="w-4 h-4 text-green-500 flex-shrink-0" />
                {f}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

export default function SubscriptionSettingsPage() {
  const router = useRouter();
  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-xl mx-auto px-4 py-8">
        <Link href="/settings" className="inline-flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 mb-6">
          <ArrowLeft className="w-4 h-4" />
          Back to Settings
        </Link>

        <h1 className="text-2xl font-bold text-gray-900 mb-6">Subscription</h1>

        <Suspense fallback={<div className="flex justify-center py-12"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>}>
          <SubscriptionContent />
        </Suspense>
      </div>
    </div>
  );
}
