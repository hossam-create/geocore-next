'use client';
import { useState, useEffect } from 'react';
import Link from 'next/link';
import api from '@/lib/api';
import { useAuthStore } from '@/store/auth';
import { Star, Gift, ChevronRight, Loader2 } from 'lucide-react';

interface LoyaltyAccount {
  current_points: number;
  tier: 'bronze' | 'silver' | 'gold' | 'platinum' | 'diamond';
}

const TIER_COLORS = {
  bronze: 'text-amber-700 bg-amber-100',
  silver: 'text-gray-600 bg-gray-100',
  gold: 'text-yellow-600 bg-yellow-100',
  platinum: 'text-purple-600 bg-purple-100',
  diamond: 'text-cyan-600 bg-cyan-100',
};

interface PointsDisplayProps {
  compact?: boolean;
}

export function PointsDisplay({ compact = false }: PointsDisplayProps) {
  const { isAuthenticated } = useAuthStore();
  const [account, setAccount] = useState<LoyaltyAccount | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!isAuthenticated) return;

    const fetchAccount = async () => {
      try {
        const res = await api.get('/loyalty/account');
        setAccount(res.data);
      } catch {
        // Silently fail - user may not have loyalty account yet
      } finally {
        setLoading(false);
      }
    };

    fetchAccount();
  }, [isAuthenticated]);

  if (!isAuthenticated || loading) return null;

  if (!account) {
    return (
      <Link
        href="/loyalty"
        className="flex items-center gap-2 rounded-xl border border-gray-200 bg-white px-4 py-3 hover:bg-gray-50 transition-colors"
      >
        <Star size={18} className="text-[#0071CE]" />
        <span className="text-sm text-gray-600">Join Loyalty Program</span>
        <ChevronRight size={14} className="text-gray-400 ml-auto" />
      </Link>
    );
  }

  const tierColor = TIER_COLORS[account.tier] || TIER_COLORS.bronze;

  if (compact) {
    return (
      <Link
        href="/loyalty"
        className="flex items-center gap-2 rounded-xl border border-gray-200 bg-white px-3 py-2 hover:bg-gray-50 transition-colors"
      >
        <Gift size={14} className="text-[#0071CE]" />
        <span className="text-sm font-semibold text-gray-900">{account.current_points.toLocaleString()} pts</span>
        <span className={`text-xs px-2 py-0.5 rounded-full capitalize ${tierColor}`}>
          {account.tier}
        </span>
      </Link>
    );
  }

  return (
    <Link
      href="/loyalty"
      className="block rounded-xl border border-gray-200 bg-white p-4 hover:shadow-sm transition-shadow"
    >
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <Gift size={18} className="text-[#0071CE]" />
          <span className="text-sm font-medium text-gray-600">Loyalty Points</span>
        </div>
        <span className={`text-xs px-2 py-0.5 rounded-full capitalize ${tierColor}`}>
          {account.tier}
        </span>
      </div>
      <div className="flex items-end justify-between">
        <p className="text-2xl font-bold text-gray-900">{account.current_points.toLocaleString()}</p>
        <span className="text-xs text-[#0071CE] font-medium flex items-center gap-1">
          View Rewards <ChevronRight size={12} />
        </span>
      </div>
    </Link>
  );
}

export default PointsDisplay;
