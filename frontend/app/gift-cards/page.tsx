import Link from 'next/link';
import { Gift, Mail, Truck, ArrowRight, Star } from 'lucide-react';

const DENOMINATIONS = ['$10', '$25', '$50', '$100', 'Custom'];

export default function GiftCardsPage() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <div className="mb-8 text-center">
        <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-[#FFC220]/20 px-4 py-1.5 text-sm font-medium text-gray-900">
          <Gift size={16} /> Gift Cards
        </div>
        <h1 className="text-3xl font-extrabold text-gray-900">Give the Gift of Choice</h1>
        <p className="mt-2 text-sm text-gray-500">Mnbarh Gift Cards — the perfect gift for any occasion.</p>
      </div>

      {/* Denominations */}
      <section className="mb-10">
        <h2 className="mb-4 text-lg font-bold text-gray-900">Choose Your Amount</h2>
        <div className="flex flex-wrap justify-center gap-3">
          {DENOMINATIONS.map((d) => (
            <div key={d} className="flex h-20 w-28 items-center justify-center rounded-2xl border-2 border-[#FFC220] bg-[#FFC220]/10 text-lg font-extrabold text-gray-900 hover:bg-[#FFC220] hover:text-white cursor-pointer transition-colors">
              {d}
            </div>
          ))}
        </div>
      </section>

      {/* Types */}
      <section className="mb-10">
        <h2 className="mb-4 text-lg font-bold text-gray-900">Two Ways to Gift</h2>
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="rounded-2xl border border-gray-200 bg-white p-6 text-center">
            <Mail size={24} className="mx-auto mb-2 text-[#0071CE]" />
            <h3 className="text-sm font-bold text-gray-900">Digital Gift Card</h3>
            <p className="mt-1 text-xs text-gray-600">Instant delivery by email. Perfect for last-minute gifts.</p>
          </div>
          <div className="rounded-2xl border border-gray-200 bg-white p-6 text-center">
            <Truck size={24} className="mx-auto mb-2 text-[#0071CE]" />
            <h3 className="text-sm font-bold text-gray-900">Physical Gift Card</h3>
            <p className="mt-1 text-xs text-gray-600">Shipped to any address. Beautiful packaging included.</p>
          </div>
        </div>
      </section>

      {/* How to Redeem */}
      <section className="mb-10 rounded-2xl border border-gray-200 bg-gray-50 p-6">
        <h2 className="mb-3 text-lg font-bold text-gray-900">How to Redeem</h2>
        <p className="text-sm text-gray-600">Enter your gift card code at checkout → amount applied to order total. It&apos;s that simple.</p>
      </section>

      {/* Terms */}
      <section className="mb-10">
        <h2 className="mb-3 text-lg font-bold text-gray-900">Terms</h2>
        <ul className="space-y-1 text-sm text-gray-600">
          <li>• Valid for 12 months from purchase date</li>
          <li>• Not redeemable for cash</li>
          <li>• Can be used on any listing on Mnbarh</li>
          <li>• Check balance at /gift-cards/check</li>
        </ul>
      </section>

      <section className="text-center">
        <Link href="/listings" className="inline-flex items-center gap-2 rounded-full bg-[#FFC220] px-8 py-3 text-sm font-bold text-gray-900 hover:bg-yellow-400">
          Buy a Gift Card <ArrowRight size={16} />
        </Link>
      </section>
    </div>
  );
}
