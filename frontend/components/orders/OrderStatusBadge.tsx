'use client';

import type { ComponentType } from 'react';
import { CheckCircle2, Clock3, PackageCheck, RefreshCcw, ShieldAlert, Truck, XCircle } from 'lucide-react';

type OrderStatus =
  | 'pending'
  | 'confirmed'
  | 'processing'
  | 'shipped'
  | 'delivered'
  | 'completed'
  | 'cancelled'
  | 'disputed'
  | 'refunded'
  | string;

const STATUS_META: Record<string, { label: string; className: string; Icon: ComponentType<{ className?: string }> }> = {
  pending: { label: 'Pending', className: 'text-amber-700 bg-amber-50 border-amber-200', Icon: Clock3 },
  confirmed: { label: 'Confirmed', className: 'text-indigo-700 bg-indigo-50 border-indigo-200', Icon: CheckCircle2 },
  processing: { label: 'Processing', className: 'text-blue-700 bg-blue-50 border-blue-200', Icon: RefreshCcw },
  shipped: { label: 'Shipped', className: 'text-sky-700 bg-sky-50 border-sky-200', Icon: Truck },
  delivered: { label: 'Delivered', className: 'text-green-700 bg-green-50 border-green-200', Icon: PackageCheck },
  completed: { label: 'Completed', className: 'text-emerald-700 bg-emerald-50 border-emerald-200', Icon: CheckCircle2 },
  cancelled: { label: 'Cancelled', className: 'text-red-700 bg-red-50 border-red-200', Icon: XCircle },
  disputed: { label: 'Disputed', className: 'text-orange-700 bg-orange-50 border-orange-200', Icon: ShieldAlert },
  refunded: { label: 'Refunded', className: 'text-purple-700 bg-purple-50 border-purple-200', Icon: RefreshCcw },
};

export function OrderStatusBadge({ status }: { status: OrderStatus }) {
  const meta = STATUS_META[status] ?? {
    label: status,
    className: 'text-gray-700 bg-gray-50 border-gray-200',
    Icon: Clock3,
  };

  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs font-semibold ${meta.className}`}
      title={`Order status: ${meta.label}`}
    >
      <meta.Icon className="h-3.5 w-3.5" />
      {meta.label}
    </span>
  );
}
