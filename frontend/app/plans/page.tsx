'use client';
import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/store/auth';
import api from '@/lib/api';
import { Check, Zap, Crown, Star, Building2, Loader2, ArrowRight } from 'lucide-react';

interface Plan {
  id: string;
  name: string;
  display_name: string;
  price_monthly: number;
  currency: string;
  listing_limit: number;
  features: string[];
  is_active: boolean;
}

interface CurrentSubscription {
  subscription: { plan_id: string; status: string; cancel_at_period_end: boolean; current_period_end?: string } | null;
  plan: Plan | null;
  on_free_plan: boolean;
}

const PLAN_ICONS: Record<string, React.ElementType> = {
  free: Star,
  basic: Zap,
  pro: Crown,
  enterprise: Building2,
};

const PLAN_COLORS: Record<string, string> = {
  free: 'border-gray-200',
  basic: 'border-blue-300',
  pro: 'border-purple-400 ring-2 ring-purple-400',
  enterprise: 'border-gray-900',
};

const PLAN_BADGE: Record<string, string | null> = {
  free: null,
  basic: null,
  pro: 'Most Popular',
  enterprise: null,
};

export default function PlansPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [current, setCurrent] = useState<CurrentSubscription | null>(null);
  const [loading, setLoading] = useState(true);
  const [subscribing, setSubscribing] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const plansRes = await api.get('/api/v1/plans');
        setPlans(plansRes.data ?? []);
        if (isAuthenticated) {
          const subRes = await api.get('/api/v1/subscriptions/me');
          setCurrent(subRes.data);
        }
      } catch (err) {
        console.error('Failed to load plans', err);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [isAuthenticated]);

  const handleSubscribe = async (plan: Plan) => {
    if (!isAuthenticated) {
      router.push('/login?redirect=/plans');
      return;
    }
    if (plan.name === 'free') return;

    setSubscribing(plan.id);
    try {
      const res = await api.post('/api/v1/subscriptions', { plan_id: plan.id });
      if (res.data.client_secret) {
        // In production: redirect to Stripe checkout or use Stripe.js
        router.push('/settings/subscription?activated=1');
      } else {
        router.push('/settings/subscription?activated=1');
      }
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } };
      alert(e?.response?.data?.error ?? 'Failed to start subscription');
    } finally {
      setSubscribing(null);
    }
  };

  const isCurrentPlan = (plan: Plan) => {
    if (!isAuthenticated || !current) return plan.name === 'free';
    if (current.on_free_plan) return plan.name === 'free';
    return current.plan?.id === plan.id;
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-6xl mx-auto px-4 py-12">
        {/* Header */}
        <div className="text-center mb-12">
          <h1 className="text-3xl font-bold text-gray-900 mb-3">Simple, transparent pricing</h1>
          <p className="text-gray-500 max-w-lg mx-auto">
            Scale your selling with a plan that fits your needs. All plans include secure payments and buyer protection.
          </p>
        </div>

        {/* Plans grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {plans.map(plan => {
            const Icon = PLAN_ICONS[plan.name] ?? Star;
            const badge = PLAN_BADGE[plan.name];
            const isCurrent = isCurrentPlan(plan);

            return (
              <div
                key={plan.id}
                className={`relative bg-white rounded-2xl border-2 p-6 flex flex-col ${PLAN_COLORS[plan.name] ?? 'border-gray-200'}`}
              >
                {badge && (
                  <span className="absolute -top-3 left-1/2 -translate-x-1/2 bg-purple-600 text-white text-xs font-bold px-3 py-1 rounded-full">
                    {badge}
                  </span>
                )}

                <div className="flex items-center gap-2 mb-4">
                  <Icon className="w-5 h-5 text-blue-600" />
                  <span className="font-bold text-gray-900">{plan.display_name}</span>
                </div>

                <div className="mb-5">
                  {plan.price_monthly === 0 ? (
                    <span className="text-3xl font-extrabold text-gray-900">Free</span>
                  ) : (
                    <>
                      <span className="text-3xl font-extrabold text-gray-900">
                        {plan.currency} {plan.price_monthly.toLocaleString()}
                      </span>
                      <span className="text-sm text-gray-400">/mo</span>
                    </>
                  )}
                </div>

                <p className="text-xs text-gray-500 mb-4">
                  {plan.listing_limit === 0
                    ? 'Unlimited active listings'
                    : `Up to ${plan.listing_limit} active listings`}
                </p>

                <ul className="space-y-2 mb-6 flex-1">
                  {plan.features.map((f, i) => (
                    <li key={i} className="flex items-start gap-2 text-sm text-gray-600">
                      <Check className="w-4 h-4 text-green-500 flex-shrink-0 mt-0.5" />
                      {f}
                    </li>
                  ))}
                </ul>

                {isCurrent ? (
                  <div className="w-full py-2.5 text-center text-sm font-semibold text-gray-500 bg-gray-100 rounded-xl">
                    Current Plan
                  </div>
                ) : plan.name === 'free' ? (
                  <div className="w-full py-2.5 text-center text-sm text-gray-400 bg-gray-50 rounded-xl border border-gray-200">
                    Default plan
                  </div>
                ) : (
                  <button
                    onClick={() => handleSubscribe(plan)}
                    disabled={subscribing === plan.id}
                    className="w-full py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-60 text-white rounded-xl text-sm font-semibold transition-colors flex items-center justify-center gap-2"
                  >
                    {subscribing === plan.id ? (
                      <Loader2 className="w-4 h-4 animate-spin" />
                    ) : (
                      <>
                        Upgrade <ArrowRight className="w-4 h-4" />
                      </>
                    )}
                  </button>
                )}
              </div>
            );
          })}
        </div>

        {/* Footer note */}
        <p className="text-center text-sm text-gray-400 mt-8">
          All prices in AED. Cancel anytime.{' '}
          {isAuthenticated && (
            <Link href="/settings/subscription" className="text-blue-600 hover:underline font-medium">
              Manage your subscription →
            </Link>
          )}
        </p>
      </div>
    </div>
  );
}
