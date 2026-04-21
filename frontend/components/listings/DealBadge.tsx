'use client';
import { Clock, TrendingDown } from 'lucide-react';

interface DealBadgeProps {
  discountPct: number;
  size?: 'sm' | 'md' | 'lg';
  showTimer?: boolean;
  timeRemaining?: string;
}

export function DealBadge({ discountPct, size = 'md', showTimer = false, timeRemaining }: DealBadgeProps) {
  const sizeConfig = {
    sm: { badge: 'text-xs px-1.5 py-0.5', icon: 10 },
    md: { badge: 'text-sm px-2 py-1', icon: 12 },
    lg: { badge: 'text-base px-3 py-1.5', icon: 14 },
  };

  const config = sizeConfig[size];

  return (
    <div className="flex flex-col gap-1">
      <span className={`inline-flex items-center gap-1 bg-red-500 text-white font-bold rounded-lg shadow ${config.badge}`}>
        <TrendingDown size={config.icon} />
        -{discountPct}%
      </span>
      {showTimer && timeRemaining && (
        <span className="inline-flex items-center gap-1 text-xs text-gray-500">
          <Clock size={10} />
          {timeRemaining}
        </span>
      )}
    </div>
  );
}

export default DealBadge;
