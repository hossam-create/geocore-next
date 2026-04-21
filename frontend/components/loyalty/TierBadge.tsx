'use client';
import { Star, Award, Crown, Zap, Gift } from 'lucide-react';

type Tier = 'bronze' | 'silver' | 'gold' | 'platinum' | 'diamond';

interface TierBadgeProps {
  tier: Tier;
  size?: 'sm' | 'md' | 'lg';
  showLabel?: boolean;
}

const TIER_CONFIG = {
  bronze: { label: 'Bronze', color: 'text-amber-700', bg: 'bg-amber-100', border: 'border-amber-300', icon: Star },
  silver: { label: 'Silver', color: 'text-gray-600', bg: 'bg-gray-100', border: 'border-gray-300', icon: Award },
  gold: { label: 'Gold', color: 'text-yellow-600', bg: 'bg-yellow-100', border: 'border-yellow-400', icon: Crown },
  platinum: { label: 'Platinum', color: 'text-purple-600', bg: 'bg-purple-100', border: 'border-purple-400', icon: Zap },
  diamond: { label: 'Diamond', color: 'text-cyan-600', bg: 'bg-cyan-100', border: 'border-cyan-400', icon: Gift },
};

const SIZE_CONFIG = {
  sm: { container: 'px-2 py-0.5 text-xs gap-1', icon: 12 },
  md: { container: 'px-3 py-1 text-sm gap-1.5', icon: 14 },
  lg: { container: 'px-4 py-1.5 text-base gap-2', icon: 18 },
};

export function TierBadge({ tier, size = 'md', showLabel = true }: TierBadgeProps) {
  const config = TIER_CONFIG[tier] || TIER_CONFIG.bronze;
  const sizeConfig = SIZE_CONFIG[size];
  const Icon = config.icon;

  return (
    <span 
      className={`inline-flex items-center rounded-full border ${config.bg} ${config.border} ${config.color} font-medium ${sizeConfig.container}`}
    >
      <Icon size={sizeConfig.icon} />
      {showLabel && <span>{config.label}</span>}
    </span>
  );
}

export default TierBadge;
