import Link from 'next/link';
import { Shield, Lock, CreditCard, BadgeCheck, TrendingUp, AlertCircle, CheckCircle, Clock } from 'lucide-react';

const PROTECTION_FEATURES = [
  {
    icon: Lock,
    title: 'Fraud Prevention',
    desc: 'Advanced fraud detection systems monitor transactions in real time. Suspicious activity is flagged before it impacts your business.',
    color: 'text-[#0071CE]',
    bg: 'bg-[#0071CE]/10',
  },
  {
    icon: CreditCard,
    title: 'Secure Payouts',
    desc: 'Funds are released to your wallet only after buyer confirmation. Withdraw to your bank account securely at any time.',
    color: 'text-emerald-600',
    bg: 'bg-emerald-100',
  },
  {
    icon: AlertCircle,
    title: 'Chargeback Coverage',
    desc: 'We support sellers in disputing unjustified chargebacks. Provide evidence and our team will advocate on your behalf.',
    color: 'text-amber-600',
    bg: 'bg-amber-100',
  },
  {
    icon: BadgeCheck,
    title: 'Verified Buyers',
    desc: 'Buyers undergo verification for high-value transactions. Reduce risk by transacting with trusted, verified members.',
    color: 'text-violet-600',
    bg: 'bg-violet-100',
  },
];

const HOW_IT_WORKS = [
  { step: 1, title: 'List Item', desc: 'Create listings with photos and descriptions.' },
  { step: 2, title: 'Receive Order', desc: 'Get notified when a buyer places an order.' },
  { step: 3, title: 'Ship Item', desc: 'Mark as shipped with tracking number.' },
  { step: 4, title: 'Get Paid', desc: 'Funds released to wallet after buyer confirms.' },
];

const SELLER_TIPS = [
  'Provide accurate descriptions and clear photos',
  'Ship within stated handling time',
  'Upload tracking information promptly',
  'Communicate with buyers through the platform',
  'Maintain high ratings to unlock seller benefits',
];

export default function SellerProtectionPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10">
      <div className="mb-8 text-center">
        <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-[#FFC220]/20 px-4 py-1.5 text-sm font-medium text-gray-900">
          <Shield size={16} />
          Seller Protection
        </div>
        <h1 className="text-3xl font-extrabold text-gray-900">Sell with Confidence</h1>
        <p className="mt-2 text-sm text-gray-500 max-w-xl mx-auto">
          Mnbarh provides robust protections for sellers, ensuring you get paid fairly and securely for every successful transaction.
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

      {/* How Escrow Works for Sellers */}
      <section className="mb-10 rounded-2xl border border-gray-200 bg-gray-50 p-6">
        <h2 className="mb-4 text-lg font-bold text-gray-900">How Escrow Protects You</h2>
        <div className="grid gap-3 sm:grid-cols-4">
          {HOW_IT_WORKS.map((item) => (
            <div key={item.step} className="flex items-start gap-2">
              <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-[#FFC220] text-xs font-bold text-gray-900">
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

      {/* Seller Tips */}
      <section className="mb-10">
        <h2 className="mb-4 text-lg font-bold text-gray-900">Best Practices for Sellers</h2>
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-4">
            <h3 className="mb-2 flex items-center gap-2 text-sm font-semibold text-emerald-800">
              <CheckCircle size={16} />
              Follow These Tips
            </h3>
            <ul className="space-y-1 text-xs text-emerald-700">
              {SELLER_TIPS.map((tip, i) => (
                <li key={i}>{tip}</li>
              ))}
            </ul>
          </div>
          <div className="rounded-xl border border-gray-200 bg-white p-4">
            <h3 className="mb-2 flex items-center gap-2 text-sm font-semibold text-gray-700">
              <Clock size={16} />
              Dispute Response Time
            </h3>
            <ul className="space-y-1 text-xs text-gray-600">
              <li>Respond to disputes within 3 business days</li>
              <li>Upload tracking and communication evidence</li>
              <li>Our team reviews both parties fairly</li>
              <li>Resolution typically within 5-7 business days</li>
            </ul>
          </div>
        </div>
      </section>

      {/* Seller Stats */}
      <section className="mb-10 rounded-2xl border border-[#FFC220] bg-[#FFC220]/10 p-6">
        <div className="flex items-center gap-3 mb-3">
          <TrendingUp size={20} className="text-gray-900" />
          <h2 className="text-lg font-bold text-gray-900">Seller Performance Matters</h2>
        </div>
        <p className="text-sm text-gray-700">
          Maintain a positive rating above 95% and ship on time to unlock benefits like reduced selling fees, priority support, and the Verified Seller badge.
        </p>
      </section>

      {/* CTAs */}
      <section className="text-center">
        <Link href="/sell" className="inline-block rounded-xl bg-[#FFC220] px-6 py-3 text-sm font-semibold text-gray-900 hover:bg-yellow-400">
          Start Selling
        </Link>
        <div className="mt-3 flex justify-center gap-4 text-xs">
          <Link href="/dashboard" className="text-[#0071CE] hover:underline">Seller Dashboard</Link>
          <span className="text-gray-300">|</span>
          <Link href="/help/selling" className="text-[#0071CE] hover:underline">Seller Guide</Link>
          <span className="text-gray-300">|</span>
          <Link href="/legal/terms" className="text-[#0071CE] hover:underline">Terms of Service</Link>
        </div>
      </section>
    </div>
  );
}
