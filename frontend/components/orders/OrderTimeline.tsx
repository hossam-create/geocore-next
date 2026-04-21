'use client';

import { CheckCircle2, Circle } from 'lucide-react';

interface OrderTimelineData {
  created_at?: string;
  confirmed_at?: string;
  shipped_at?: string;
  delivered_at?: string;
  completed_at?: string;
  cancelled_at?: string;
  status?: string;
}

function formatDateTime(value?: string) {
  if (!value) return 'Pending';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return 'Pending';
  return date.toLocaleString();
}

export function OrderTimeline({ order }: { order: OrderTimelineData }) {
  const steps = [
    { key: 'created_at', label: 'Order Placed', at: order.created_at },
    { key: 'confirmed_at', label: 'Seller Confirmed', at: order.confirmed_at },
    { key: 'shipped_at', label: 'Shipped', at: order.shipped_at },
    { key: 'delivered_at', label: 'Delivered', at: order.delivered_at },
    { key: 'completed_at', label: 'Completed', at: order.completed_at },
  ];

  const cancelled = Boolean(order.cancelled_at || order.status === 'cancelled');

  return (
    <div className="rounded-2xl border border-gray-200 bg-white p-5">
      <h3 className="mb-4 text-sm font-bold text-gray-800">Order Timeline</h3>
      <ol className="space-y-3">
        {steps.map((step) => {
          const done = Boolean(step.at);
          return (
            <li key={step.key} className="flex items-start gap-3">
              <span className="mt-0.5">
                {done ? (
                  <CheckCircle2 className="h-4 w-4 text-green-600" />
                ) : (
                  <Circle className="h-4 w-4 text-gray-300" />
                )}
              </span>
              <div>
                <p className={`text-sm font-medium ${done ? 'text-gray-900' : 'text-gray-500'}`}>{step.label}</p>
                <p className="text-xs text-gray-400">{formatDateTime(step.at)}</p>
              </div>
            </li>
          );
        })}

        {cancelled && (
          <li className="flex items-start gap-3">
            <CheckCircle2 className="mt-0.5 h-4 w-4 text-red-600" />
            <div>
              <p className="text-sm font-medium text-red-700">Order Cancelled</p>
              <p className="text-xs text-gray-400">{formatDateTime(order.cancelled_at)}</p>
            </div>
          </li>
        )}
      </ol>
    </div>
  );
}
