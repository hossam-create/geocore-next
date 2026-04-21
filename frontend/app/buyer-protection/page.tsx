import Link from 'next/link';
import { Shield, Lock, RefreshCw, BadgeCheck, Clock, CheckCircle, AlertTriangle } from 'lucide-react';

const PROTECTION_FEATURES = [
  {
    icon: Lock,
    title: 'Escrow Protection',
    desc: 'Your payment is held securely in escrow until you confirm delivery. Sellers only receive funds after you verify the item matches the description.',
    color: 'text-[#0071CE]',
    bg: 'bg-[#0071CE]/10',
  },
  {
    icon: RefreshCw,
    title: 'Money-Back Guarantee',
    desc: 'If your item does not arrive or is significantly different from the listing, you are eligible for a full refund. No questions asked within the protection window.',
    color: 'text-emerald-600',
    bg: 'bg-emerald-100',
  },
  {
    icon: AlertTriangle,
    title: 'Dispute Resolution',
    desc: 'Our dedicated support team mediates disputes fairly. Submit evidence, track progress, and receive a resolution within 5-7 business days.',
    color: 'text-amber-600',
    bg: 'bg-amber-100',
  },
  {
    icon: BadgeCheck,
    title: 'Verified Sellers',
    desc: 'Sellers undergo identity verification and performance monitoring. Look for the Verified Seller badge for trusted partners.',
    color: 'text-violet-600',
    bg: 'bg-violet-100',
  },
];

const HOW_IT_WORKS = [
  { step: 1, title: 'Place Order', desc: 'Checkout securely with card or wallet balance.' },
  { step: 2, title: 'Payment Held', desc: 'Funds are held in escrow, not released to seller.' },
  { step: 3, title: 'Receive Item', desc: 'Item ships to your address with tracking.' },
  { step: 4, title: 'Confirm Delivery', desc: 'Verify item matches description to release payment.' },
];

export default function BuyerProtectionPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10">
      <div className="mb-8 text-center">
        <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-[#0071CE]/10 px-4 py-1.5 text-sm font-medium text-[#0071CE]">
          <Shield size={16} />
          Buyer Protection
        </div>
        <h1 className="text-3xl font-extrabold text-gray-900">Shop with Confidence</h1>
        <p className="mt-2 text-sm text-gray-500 max-w-xl mx-auto">
          Every purchase on Mnbarh is protected by our comprehensive buyer safeguards. Your satisfaction is our priority.
        </p>
      </div>

      {/* Protection Features */}
      <section className="mb-10">
        <div className="grid gap-4 sm:grid-cols-2">
          {PROTECTION_FEATURES.map((feature) => (
            <div key={feature.title} className="rounded-2xl border border-gray-200 bg-white p-5">
              <div className={`mb-3 inline-flex h-10 w-10 items-center justify-center rounded-xl ${feature.bg}`}>
                <feature.icon size={20} className={feature.color} />
              </div>
              <h3 className="text-base font-bold text-gray-900">{feature.title}</h3>
              <p className="mt-1 text-sm text-gray-600 leading-relaxed">{feature.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* How Escrow Works */}
      <section className="mb-10 rounded-2xl border border-gray-200 bg-gray-50 p-6">
        <h2 className="mb-4 text-lg font-bold text-gray-900">How Escrow Protects You</h2>
        <div className="grid gap-3 sm:grid-cols-4">
          {HOW_IT_WORKS.map((item) => (
            <div key={item.step} className="flex items-start gap-2">
              <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-[#0071CE] text-xs font-bold text-white">
                {item.step}
              </span>
              <div>
                <h4 className="text-sm font-semibold text-gray-900">{item.title}</h4>
                <p className="text-xs text-gray-600">{item.desc}</p>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* Coverage Details */}
      <section className="mb-10">
        <h2 className="mb-4 text-lg font-bold text-gray-900">What Is Covered</h2>
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-4">
            <h3 className="mb-2 flex items-center gap-2 text-sm font-semibold text-emerald-800">
              <CheckCircle size={16} />
              Protected Scenarios
            </h3>
            <ul className="space-y-1 text-xs text-emerald-700">
              <li>Item not received within estimated delivery</li>
              <li>Item significantly different from description</li>
              <li>Damaged during shipping (with proof)</li>
              <li>Counterfeit or fake items</li>
              <li>Wrong item sent by seller</li>
            </ul>
          </div>
          <div className="rounded-xl border border-gray-200 bg-white p-4">
            <h3 className="mb-2 flex items-center gap-2 text-sm font-semibold text-gray-700">
              <Clock size={16} />
              Protection Window
            </h3>
            <ul className="space-y-1 text-xs text-gray-600">
              <li>30 days from delivery confirmation</li>
              <li>Dispute must be opened within this period</li>
              <li>Resolution typically within 5-7 business days</li>
              <li>Extended protection for high-value items</li>
            </ul>
          </div>
        </div>
      </section>

      {/* CTAs */}
      <section className="text-center">
        <Link href="/listings" className="inline-block rounded-xl bg-[#0071CE] px-6 py-3 text-sm font-semibold text-white hover:bg-[#005ba3]">
          Shop with Confidence
        </Link>
        <div className="mt-3 flex justify-center gap-4 text-xs">
          <Link href="/disputes/new" className="text-[#0071CE] hover:underline">Open a Dispute</Link>
          <span className="text-gray-300">|</span>
          <Link href="/help" className="text-[#0071CE] hover:underline">Help Center</Link>
          <span className="text-gray-300">|</span>
          <Link href="/legal/terms" className="text-[#0071CE] hover:underline">Terms of Service</Link>
        </div>
      </section>
    </div>
  );
}
