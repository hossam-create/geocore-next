'use client';
import { useState } from 'react';
import Link from 'next/link';
import { Calculator, Info, DollarSign, Percent, TrendingDown } from 'lucide-react';

const PLATFORM_FEE_PERCENT = 0.05; // 5%
const PAYMENT_FEE_PERCENT = 0.025; // 2.5%
const MIN_PLATFORM_FEE = 2; // Minimum platform fee in AED

export default function FeeCalculatorPage() {
  const [salePrice, setSalePrice] = useState<string>('');
  const [category, setCategory] = useState<string>('standard');

  const price = parseFloat(salePrice) || 0;

  // Category-specific fee adjustments
  const categoryMultiplier = category === 'electronics' ? 1.0 : category === 'vehicles' ? 0.8 : 1.0;

  const platformFee = Math.max(price * PLATFORM_FEE_PERCENT * categoryMultiplier, price > 0 ? MIN_PLATFORM_FEE : 0);
  const paymentFee = price * PAYMENT_FEE_PERCENT;
  const totalFees = platformFee + paymentFee;
  const netPayout = price - totalFees;

  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <div className="mb-8 text-center">
        <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-[#FFC220]/20 px-4 py-1.5 text-sm font-medium text-gray-900">
          <Calculator size={16} />
          Fee Calculator
        </div>
        <h1 className="text-3xl font-extrabold text-gray-900">Seller Fee Calculator</h1>
        <p className="mt-2 text-sm text-gray-500 max-w-xl mx-auto">
          Estimate how much you'll earn after platform and payment processing fees. All calculations are client-side for transparency.
        </p>
      </div>

      {/* Calculator */}
      <section className="mb-10 rounded-2xl border border-gray-200 bg-white p-6">
        <div className="grid gap-6 md:grid-cols-2">
          {/* Inputs */}
          <div>
            <label className="mb-2 block text-sm font-semibold text-gray-700">
              Sale Price (AED)
            </label>
            <div className="relative">
              <DollarSign size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
              <input
                type="number"
                value={salePrice}
                onChange={(e) => setSalePrice(e.target.value)}
                placeholder="Enter sale price"
                className="w-full rounded-xl border border-gray-300 py-2.5 pl-10 pr-4 text-sm focus:border-[#0071CE] focus:outline-none focus:ring-1 focus:ring-[#0071CE]"
              />
            </div>

            <label className="mb-2 mt-4 block text-sm font-semibold text-gray-700">
              Category
            </label>
            <select
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              className="w-full rounded-xl border border-gray-300 py-2.5 px-4 text-sm focus:border-[#0071CE] focus:outline-none focus:ring-1 focus:ring-[#0071CE]"
            >
              <option value="standard">Standard (5% platform fee)</option>
              <option value="electronics">Electronics (5% platform fee)</option>
              <option value="vehicles">Vehicles (4% platform fee)</option>
            </select>
          </div>

          {/* Results */}
          <div className="rounded-xl bg-gray-50 p-4">
            <h3 className="mb-3 text-sm font-semibold text-gray-700">Fee Breakdown</h3>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-gray-600">Sale Price</span>
                <span className="font-medium text-gray-900">AED {price.toFixed(2)}</span>
              </div>
              <div className="flex justify-between text-rose-600">
                <span className="flex items-center gap-1">
                  <TrendingDown size={14} />
                  Platform Fee ({(PLATFORM_FEE_PERCENT * 100 * (category === 'vehicles' ? 0.8 : 1)).toFixed(1)}%)
                </span>
                <span>-AED {platformFee.toFixed(2)}</span>
              </div>
              <div className="flex justify-between text-rose-600">
                <span className="flex items-center gap-1">
                  <Percent size={14} />
                  Payment Processing (2.5%)
                </span>
                <span>-AED {paymentFee.toFixed(2)}</span>
              </div>
              <div className="border-t border-gray-200 pt-2 mt-2">
                <div className="flex justify-between text-base font-bold">
                  <span className="text-gray-900">Net Payout</span>
                  <span className="text-emerald-600">AED {netPayout.toFixed(2)}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Fee Explanation */}
      <section className="mb-10 grid gap-4 sm:grid-cols-3">
        <div className="rounded-xl border border-gray-200 bg-white p-4">
          <div className="mb-2 flex items-center gap-2">
            <Info size={16} className="text-[#0071CE]" />
            <h3 className="text-sm font-semibold text-gray-900">Platform Fee</h3>
          </div>
          <p className="text-xs text-gray-600">
            A percentage of the sale price that supports marketplace operations, buyer protection, and seller tools.
          </p>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-4">
          <div className="mb-2 flex items-center gap-2">
            <Info size={16} className="text-[#0071CE]" />
            <h3 className="text-sm font-semibold text-gray-900">Payment Processing</h3>
          </div>
          <p className="text-xs text-gray-600">
            Covers card processing, fraud prevention, and secure payment handling by our payment provider.
          </p>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-4">
          <div className="mb-2 flex items-center gap-2">
            <Info size={16} className="text-[#0071CE]" />
            <h3 className="text-sm font-semibold text-gray-900">Minimum Fee</h3>
          </div>
          <p className="text-xs text-gray-600">
            A minimum platform fee of AED 2 applies to ensure sustainable operations for low-value items.
          </p>
        </div>
      </section>

      {/* Notes */}
      <section className="rounded-xl border border-amber-200 bg-amber-50 p-4 mb-10">
        <h3 className="mb-2 flex items-center gap-2 text-sm font-semibold text-amber-800">
          <Info size={16} />
          Important Notes
        </h3>
        <ul className="space-y-1 text-xs text-amber-700">
          <li>Fees are deducted from your payout after the buyer confirms delivery.</li>
          <li>Escrow protects both parties — funds are held securely until transaction completion.</li>
          <li>Category-specific fees may change — verify current rates in the seller dashboard.</li>
          <li>This calculator provides estimates only — actual fees may vary slightly.</li>
        </ul>
      </section>

      {/* CTAs */}
      <section className="text-center">
        <Link href="/sell" className="inline-block rounded-xl bg-[#FFC220] px-6 py-3 text-sm font-semibold text-gray-900 hover:bg-yellow-400">
          Start Selling
        </Link>
        <div className="mt-3 flex justify-center gap-4 text-xs">
          <Link href="/seller-protection" className="text-[#0071CE] hover:underline">Seller Protection</Link>
          <span className="text-gray-300">|</span>
          <Link href="/help/selling" className="text-[#0071CE] hover:underline">Seller Guide</Link>
          <span className="text-gray-300">|</span>
          <Link href="/dashboard" className="text-[#0071CE] hover:underline">Seller Dashboard</Link>
        </div>
      </section>
    </div>
  );
}
