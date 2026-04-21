'use client';

import { Radio, CheckCircle, XCircle, Clock, Loader2, AlertTriangle, ShoppingCart } from 'lucide-react';

type LiveItemStatus = 'pending' | 'active' | 'settling' | 'sold' | 'sold_buy_now' | 'unsold' | 'payment_failed' | 'cancelled';

interface LiveStatusBadgeProps {
  status: LiveItemStatus | string;
}

const config: Record<string, { bg: string; text: string; icon: React.ReactNode; label: string }> = {
  pending:        { bg: 'bg-amber-50 border-amber-200',   text: 'text-amber-700',   icon: <Clock className="w-3 h-3" />,                  label: 'Pending' },
  active:         { bg: 'bg-red-50 border-red-200',       text: 'text-red-600',     icon: <Radio className="w-3 h-3 animate-pulse" />,   label: 'LIVE' },
  settling:       { bg: 'bg-blue-50 border-blue-200',     text: 'text-blue-700',    icon: <Loader2 className="w-3 h-3 animate-spin" />,  label: 'Settling' },
  sold:           { bg: 'bg-green-50 border-green-200',   text: 'text-green-700',   icon: <CheckCircle className="w-3 h-3" />,           label: 'Sold' },
  sold_buy_now:   { bg: 'bg-emerald-50 border-emerald-200', text: 'text-emerald-700', icon: <ShoppingCart className="w-3 h-3" />,        label: 'Sold (Buy Now)' },
  unsold:         { bg: 'bg-gray-100 border-gray-200',    text: 'text-gray-600',    icon: <XCircle className="w-3 h-3" />,               label: 'Unsold' },
  payment_failed: { bg: 'bg-red-50 border-red-300',      text: 'text-red-800',     icon: <AlertTriangle className="w-3 h-3" />,         label: 'Payment Failed' },
  cancelled:      { bg: 'bg-gray-100 border-gray-200',   text: 'text-gray-500',    icon: <XCircle className="w-3 h-3" />,               label: 'Cancelled' },
};

export default function LiveStatusBadge({ status }: LiveStatusBadgeProps) {
  const c = config[status] ?? config.pending;
  return (
    <span className={`inline-flex items-center gap-1 text-xs font-semibold px-2 py-0.5 rounded-full border ${c.bg} ${c.text}`}>
      {c.icon} {c.label}
    </span>
  );
}
