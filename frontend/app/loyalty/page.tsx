'use client';
import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { useAuthStore } from '@/store/auth';
import api from '@/lib/api';
import {
  Gift, Star, TrendingUp, Award, Clock, ChevronRight, Loader2,
  Crown, Zap, Users, Calendar, CheckCircle, ArrowLeft
} from 'lucide-react';

// Types
interface LoyaltyAccount {
  id: string;
  user_id: string;
  current_points: number;
  lifetime_points: number;
  tier: 'bronze' | 'silver' | 'gold' | 'platinum' | 'diamond';
  referral_code: string;
  total_referrals: number;
  last_login_bonus?: string;
}

interface PointsTransaction {
  id: string;
  action: string;
  points: number;
  balance: number;
  description?: string;
  created_at: string;
}

interface Reward {
  id: string;
  name: string;
  description: string;
  points_cost: number;
  type: string;
  value: number;
  min_tier: string;
  image_url?: string;
}

interface Redemption {
  id: string;
  reward: Reward;
  points_spent: number;
  code: string;
  status: string;
  expires_at: string;
  created_at: string;
}

const TIER_CONFIG = {
  bronze: { label: 'Bronze', color: 'text-amber-700', bg: 'bg-amber-100', border: 'border-amber-300', icon: Star, progress: 0 },
  silver: { label: 'Silver', color: 'text-gray-600', bg: 'bg-gray-100', border: 'border-gray-300', icon: Award, progress: 1000 },
  gold: { label: 'Gold', color: 'text-yellow-600', bg: 'bg-yellow-100', border: 'border-yellow-400', icon: Crown, progress: 5000 },
  platinum: { label: 'Platinum', color: 'text-purple-600', bg: 'bg-purple-100', border: 'border-purple-400', icon: Zap, progress: 15000 },
  diamond: { label: 'Diamond', color: 'text-cyan-600', bg: 'bg-cyan-100', border: 'border-cyan-400', icon: Gift, progress: 50000 },
};

const TIER_THRESHOLDS = { bronze: 0, silver: 1000, gold: 5000, platinum: 15000, diamond: 50000 };

const ACTION_LABELS: Record<string, string> = {
  purchase: 'Purchase',
  sale: 'Sale',
  review: 'Review',
  referral: 'Referral Bonus',
  daily_login: 'Daily Login',
  profile_complete: 'Profile Completed',
  kyc_verified: 'KYC Verified',
  first_purchase: 'First Purchase',
  auction_win: 'Auction Win',
  redemption: 'Redemption',
  adjustment: 'Adjustment',
};

export default function LoyaltyPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [mounted, setMounted] = useState(false);
  const [account, setAccount] = useState<LoyaltyAccount | null>(null);
  const [transactions, setTransactions] = useState<PointsTransaction[]>([]);
  const [rewards, setRewards] = useState<Reward[]>([]);
  const [redemptions, setRedemptions] = useState<Redemption[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'overview' | 'history' | 'rewards'>('overview');

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (mounted && !isAuthenticated) {
      router.push('/login?redirect=/loyalty');
    }
  }, [mounted, isAuthenticated, router]);

  useEffect(() => {
    if (!isAuthenticated) return;

    const fetchData = async () => {
      setLoading(true);
      setError(null);
      try {
        const [accountRes, txRes, rewardsRes, redemptionsRes] = await Promise.all([
          api.get('/loyalty/account'),
          api.get('/loyalty/transactions'),
          api.get('/loyalty/rewards'),
          api.get('/loyalty/redemptions'),
        ]);
        setAccount(accountRes.data);
        setTransactions(txRes.data || []);
        setRewards(rewardsRes.data || []);
        setRedemptions(redemptionsRes.data || []);
      } catch (err: any) {
        setError(err.response?.data?.error || 'Failed to load loyalty data');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [isAuthenticated]);

  if (!mounted || !isAuthenticated) return null;

  const tierConfig = account ? TIER_CONFIG[account.tier] : TIER_CONFIG.bronze;
  const TierIcon = tierConfig.icon;

  // Calculate progress to next tier
  const currentThreshold = account ? TIER_THRESHOLDS[account.tier] : 0;
  const nextTier = account?.tier === 'diamond' ? null : 
    account?.tier === 'bronze' ? 'silver' :
    account?.tier === 'silver' ? 'gold' :
    account?.tier === 'gold' ? 'platinum' : 'diamond';
  const nextThreshold = nextTier ? TIER_THRESHOLDS[nextTier] : TIER_THRESHOLDS.diamond;
  const progressPercent = account ? 
    ((account.lifetime_points - currentThreshold) / (nextThreshold - currentThreshold)) * 100 : 0;

  const handleRedeem = async (rewardId: string) => {
    try {
      await api.post(`/loyalty/rewards/${rewardId}/redeem`);
      // Refresh data
      const [accountRes, redemptionsRes] = await Promise.all([
        api.get('/loyalty/account'),
        api.get('/loyalty/redemptions'),
      ]);
      setAccount(accountRes.data);
      setRedemptions(redemptionsRes.data || []);
    } catch (err: any) {
      alert(err.response?.data?.error || 'Failed to redeem reward');
    }
  };

  const handleDailyBonus = async () => {
    try {
      const res = await api.post('/loyalty/daily-bonus');
      setAccount(res.data);
    } catch (err: any) {
      alert(err.response?.data?.error || 'Failed to claim bonus');
    }
  };

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      {/* Header */}
      <div className="mb-6 flex items-center gap-3">
        <Link href="/dashboard" className="rounded-lg p-2 hover:bg-gray-100">
          <ArrowLeft size={20} className="text-gray-600" />
        </Link>
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Loyalty Program</h1>
          <p className="text-sm text-gray-500">Earn points, unlock rewards, and level up</p>
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
          {/* Tier Card */}
          <div className={`mb-6 rounded-2xl border-2 ${tierConfig.border} ${tierConfig.bg} p-6`}>
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-3">
                <div className={`flex h-12 w-12 items-center justify-center rounded-xl ${tierConfig.bg}`}>
                  <TierIcon size={24} className={tierConfig.color} />
                </div>
                <div>
                  <p className="text-xs text-gray-500">Current Tier</p>
                  <p className={`text-xl font-bold ${tierConfig.color}`}>{tierConfig.label}</p>
                </div>
              </div>
              {account?.referral_code && (
                <div className="text-right">
                  <p className="text-xs text-gray-500">Referral Code</p>
                  <p className="font-mono font-bold text-gray-900">{account.referral_code}</p>
                </div>
              )}
            </div>

            {/* Points */}
            <div className="mb-4 grid grid-cols-2 gap-4">
              <div className="rounded-xl bg-white/50 p-3">
                <p className="text-xs text-gray-500">Available Points</p>
                <p className="text-2xl font-bold text-gray-900">{account?.current_points.toLocaleString() || 0}</p>
              </div>
              <div className="rounded-xl bg-white/50 p-3">
                <p className="text-xs text-gray-500">Lifetime Points</p>
                <p className="text-2xl font-bold text-gray-900">{account?.lifetime_points.toLocaleString() || 0}</p>
              </div>
            </div>

            {/* Progress to next tier */}
            {nextTier && (
              <div className="mb-4">
                <div className="flex items-center justify-between text-xs mb-1">
                  <span className="text-gray-600">Progress to {TIER_CONFIG[nextTier].label}</span>
                  <span className="font-medium">{Math.min(100, Math.round(progressPercent))}%</span>
                </div>
                <div className="h-2 rounded-full bg-white/50 overflow-hidden">
                  <div 
                    className={`h-full ${tierConfig.bg.replace('bg-', 'bg-')}`}
                    style={{ width: `${Math.min(100, progressPercent)}%` }}
                  />
                </div>
                <p className="text-xs text-gray-500 mt-1">
                  {(nextThreshold - (account?.lifetime_points || 0)).toLocaleString()} points to next tier
                </p>
              </div>
            )}

            {/* Daily Bonus */}
            <button
              onClick={handleDailyBonus}
              disabled={!!(account?.last_login_bonus && new Date(account.last_login_bonus).toDateString() === new Date().toDateString())}
              className="w-full rounded-xl bg-[#0071CE] py-2.5 text-sm font-semibold text-white hover:bg-[#005ba3] disabled:bg-gray-300 disabled:cursor-not-allowed transition-colors"
            >
              {account?.last_login_bonus && new Date(account.last_login_bonus).toDateString() === new Date().toDateString()
                ? '✓ Daily Bonus Claimed'
                : 'Claim Daily Bonus (+10 Points)'}
            </button>
          </div>

          {/* Tabs */}
          <div className="flex gap-2 mb-4">
            {(['overview', 'history', 'rewards'] as const).map((tab) => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition-colors ${
                  activeTab === tab
                    ? 'bg-[#0071CE] text-white'
                    : 'bg-white border border-gray-200 text-gray-600 hover:bg-gray-50'
                }`}
              >
                {tab === 'overview' && <TrendingUp size={16} />}
                {tab === 'history' && <Clock size={16} />}
                {tab === 'rewards' && <Gift size={16} />}
                {tab.charAt(0).toUpperCase() + tab.slice(1)}
              </button>
            ))}
          </div>

          {/* Overview Tab */}
          {activeTab === 'overview' && (
            <div className="space-y-4">
              <div className="rounded-xl border border-gray-200 bg-white p-5">
                <h3 className="text-sm font-semibold text-gray-900 mb-3">Ways to Earn Points</h3>
                <div className="space-y-2">
                  {[
                    { action: 'Daily Login', points: '+10', icon: Calendar },
                    { action: 'Make a Purchase', points: '+1 per AED', icon: TrendingUp },
                    { action: 'Complete a Sale', points: '+2 per AED', icon: Star },
                    { action: 'Write a Review', points: '+50', icon: Award },
                    { action: 'Refer a Friend', points: '+500', icon: Users },
                  ].map((item) => (
                    <div key={item.action} className="flex items-center justify-between py-2 border-b border-gray-100 last:border-0">
                      <div className="flex items-center gap-2">
                        <item.icon size={16} className="text-gray-400" />
                        <span className="text-sm text-gray-700">{item.action}</span>
                      </div>
                      <span className="text-sm font-semibold text-emerald-600">{item.points}</span>
                    </div>
                  ))}
                </div>
              </div>

              {redemptions.length > 0 && (
                <div className="rounded-xl border border-gray-200 bg-white p-5">
                  <h3 className="text-sm font-semibold text-gray-900 mb-3">Active Rewards</h3>
                  <div className="space-y-2">
                    {redemptions.filter(r => r.status === 'active').slice(0, 3).map((redemption) => (
                      <div key={redemption.id} className="flex items-center justify-between py-2">
                        <div>
                          <p className="text-sm font-medium text-gray-900">{redemption.reward.name}</p>
                          <p className="text-xs text-gray-400">Code: {redemption.code}</p>
                        </div>
                        <span className="text-xs text-gray-500">
                          Expires {new Date(redemption.expires_at).toLocaleDateString()}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* History Tab */}
          {activeTab === 'history' && (
            <div className="rounded-xl border border-gray-200 bg-white overflow-hidden">
              {transactions.length === 0 ? (
                <div className="p-8 text-center text-gray-400 text-sm">
                  No points history yet
                </div>
              ) : (
                <div className="divide-y divide-gray-100">
                  {transactions.map((tx) => (
                    <div key={tx.id} className="flex items-center justify-between px-5 py-3">
                      <div>
                        <p className="text-sm font-medium text-gray-900">
                          {ACTION_LABELS[tx.action] || tx.action}
                        </p>
                        <p className="text-xs text-gray-400">
                          {new Date(tx.created_at).toLocaleDateString()}
                        </p>
                      </div>
                      <span className={`text-sm font-semibold ${tx.points > 0 ? 'text-emerald-600' : 'text-red-600'}`}>
                        {tx.points > 0 ? '+' : ''}{tx.points}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Rewards Tab */}
          {activeTab === 'rewards' && (
            <div className="grid gap-4 sm:grid-cols-2">
              {rewards.length === 0 ? (
                <div className="col-span-2 p-8 text-center text-gray-400 text-sm">
                  No rewards available
                </div>
              ) : (
                rewards.map((reward) => {
                  const canRedeem = account && account.current_points >= reward.points_cost;
                  const tierMet = account && TIER_THRESHOLDS[account.tier] >= TIER_THRESHOLDS[reward.min_tier as keyof typeof TIER_THRESHOLDS];
                  
                  return (
                    <div key={reward.id} className="rounded-xl border border-gray-200 bg-white p-4">
                      <div className="flex items-start justify-between mb-2">
                        <div>
                          <h4 className="text-sm font-semibold text-gray-900">{reward.name}</h4>
                          <p className="text-xs text-gray-500">{reward.description}</p>
                        </div>
                        <Gift size={20} className="text-[#0071CE]" />
                      </div>
                      <div className="flex items-center justify-between mt-3">
                        <span className="text-sm font-bold text-[#0071CE]">{reward.points_cost} pts</span>
                        <button
                          onClick={() => handleRedeem(reward.id)}
                          disabled={!canRedeem || !tierMet}
                          className="rounded-lg bg-[#0071CE] px-3 py-1.5 text-xs font-semibold text-white hover:bg-[#005ba3] disabled:bg-gray-300 disabled:cursor-not-allowed transition-colors"
                        >
                          Redeem
                        </button>
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          )}
        </>
      )}
    </div>
  );
}
