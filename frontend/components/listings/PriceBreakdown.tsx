'use client';

import { formatPrice } from '@/lib/utils';
import { Receipt, Truck, Shield, Info } from 'lucide-react';

interface PriceBreakdownProps {
  itemPrice: number;
  deliveryPrice?: number;
  platformFee?: number;
  total: number;
  currency?: string;
  escrowed?: boolean;
}

export function PriceBreakdown({ itemPrice, deliveryPrice, platformFee, total, currency = 'AED', escrowed }: PriceBreakdownProps) {
  return (
    <div className="rounded-xl border border-gray-200 bg-white p-4 space-y-3">
      <p className="text-sm font-bold text-gray-800 flex items-center gap-2">
        <Receipt size={14} /> Price Breakdown
      </p>
      <div className="space-y-2 text-sm">
        <div className="flex justify-between">
          <span className="text-gray-600">Item</span>
          <span className="font-medium text-gray-900">{formatPrice(itemPrice, currency)}</span>
        </div>
        {deliveryPrice != null && deliveryPrice > 0 && (
          <div className="flex justify-between">
            <span className="text-gray-600 flex items-center gap-1"><Truck size={12} /> Delivery</span>
            <span className="font-medium text-gray-900">{formatPrice(deliveryPrice, currency)}</span>
          </div>
        )}
        {platformFee != null && platformFee > 0 && (
          <div className="flex justify-between">
            <span className="text-gray-600 flex items-center gap-1"><Shield size={12} /> Platform fee</span>
            <span className="font-medium text-gray-900">{formatPrice(platformFee, currency)}</span>
          </div>
        )}
        <div className="border-t border-gray-100 pt-2 flex justify-between">
          <span className="font-bold text-gray-900">Total</span>
          <span className="font-bold text-gray-900">{formatPrice(total, currency)}</span>
        </div>
      </div>
      {escrowed && (
        <div className="flex items-start gap-2 text-xs text-blue-700 bg-blue-50 rounded-lg p-2.5">
          <Info size={14} className="shrink-0 mt-0.5" />
          <span>Funds held in escrow until delivery is confirmed</span>
        </div>
      )}
    </div>
  );
}
