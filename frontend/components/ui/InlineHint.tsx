'use client';

import { Lightbulb, TrendingDown, Flame, Info } from 'lucide-react';

interface InlineHintProps {
  type: 'price_tip' | 'high_demand' | 'trust_tip' | 'info';
  message: string;
}

const HINT_STYLES: Record<string, { icon: typeof Lightbulb; color: string; bg: string; border: string }> = {
  price_tip: { icon: TrendingDown, color: 'text-blue-700', bg: 'bg-blue-50', border: 'border-blue-200' },
  high_demand: { icon: Flame, color: 'text-orange-700', bg: 'bg-orange-50', border: 'border-orange-200' },
  trust_tip: { icon: Info, color: 'text-emerald-700', bg: 'bg-emerald-50', border: 'border-emerald-200' },
  info: { icon: Lightbulb, color: 'text-amber-700', bg: 'bg-amber-50', border: 'border-amber-200' },
};

export function InlineHint({ type, message }: InlineHintProps) {
  const style = HINT_STYLES[type] || HINT_STYLES.info;
  const Icon = style.icon;

  return (
    <div className={`flex items-center gap-2 text-xs font-medium rounded-lg px-3 py-2 border ${style.bg} ${style.border} ${style.color}`}>
      <Icon size={13} className="shrink-0" />
      <span>{message}</span>
    </div>
  );
}
