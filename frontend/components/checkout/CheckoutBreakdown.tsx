'use client';

import { formatPrice } from '@/lib/utils';
import { ShieldCheck, Star, Info, Lock } from 'lucide-react';

interface CheckoutBreakdownProps {
  itemPrice: number;
  deliveryPrice?: number;
  platformFee?: number;
  total: number;
  currency?: string;
  sellerRating?: number;
  sellerVerified?: boolean;
  travelerRating?: number;
  travelerVerified?: boolean;
}

export function CheckoutBreakdown({
  itemPrice, deliveryPrice, platformFee, total, currency = 'AED',
  sellerRating, sellerVerified, travelerRating, travelerVerified,
}: CheckoutBreakdownProps) {
  return (
    <div className="space-y-4">
      {/* Price Breakdown */}
      <div className="rounded-xl border border-gray-200 bg-gray-50 p-4 space-y-2">
        <p className="text-xs text-gray-400 uppercase tracking-wide font-semibold mb-2">Price Breakdown</p>
        <div className="space-y-1.5 text-sm">
          <div className="flex justify-between">
            <span className="text-gray-600">Item</span>
            <span className="font-medium">{formatPrice(itemPrice, currency)}</span>
          </div>
          {deliveryPrice != null && deliveryPrice > 0 && (
            <div className="flex justify-between">
              <span className="text-gray-600">Delivery</span>
              <span className="font-medium">{formatPrice(deliveryPrice, currency)}</span>
            </div>
          )}
          {platformFee != null && platformFee > 0 && (
            <div className="flex justify-between">
              <span className="text-gray-600">Platform fee</span>
              <span className="font-medium">{formatPrice(platformFee, currency)}</span>
            </div>
          )}
          <div className="border-t border-gray-200 pt-2 flex justify-between font-bold">
            <span>Total</span>
            <span>{formatPrice(total, currency)}</span>
          </div>
        </div>
      </div>

      {/* Trust Signals */}
      {(sellerRating != null || travelerRating != null) && (
        <div className="rounded-xl border border-gray-200 bg-white p-4 space-y-2">
          <p className="text-xs text-gray-400 uppercase tracking-wide font-semibold">Trust Signals</p>
          {sellerRating != null && (
            <div className="flex items-center gap-2 text-sm">
              <Star size={13} fill="#FFC220" className="text-[#FFC220]" />
              <span className="text-gray-600">Seller rating:</span>
              <span className="font-semibold text-gray-900">{sellerRating.toFixed(1)}/5</span>
              {sellerVerified && <span className="text-xs text-blue-600 bg-blue-50 px-1.5 py-0.5 rounded-full font-semibold">Verified</span>}
            </div>
          )}
          {travelerRating != null && (
            <div className="flex items-center gap-2 text-sm">
              <Star size={13} fill="#FFC220" className="text-[#FFC220]" />
              <span className="text-gray-600">Traveler rating:</span>
              <span className="font-semibold text-gray-900">{travelerRating.toFixed(1)}/5</span>
              {travelerVerified && <span className="text-xs text-emerald-600 bg-emerald-50 px-1.5 py-0.5 rounded-full font-semibold">Trusted</span>}
            </div>
          )}
        </div>
      )}

      {/* Escrow Clarity */}
      <div className="flex items-start gap-2.5 text-xs text-blue-700 bg-blue-50 rounded-xl p-3 border border-blue-100">
        <ShieldCheck size={16} className="shrink-0 text-blue-500" />
        <div>
          <p className="font-semibold">Funds held in escrow</p>
          <p className="text-blue-600 mt-0.5">Your payment is protected. Funds are released to the seller only after you confirm delivery.</p>
        </div>
      </div>
    </div>
  );
}
