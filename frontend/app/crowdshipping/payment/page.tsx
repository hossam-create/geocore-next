import Link from 'next/link';
import { Lock, CreditCard, Wallet, ArrowRight, CheckCircle, Clock, ShieldCheck, RefreshCw } from 'lucide-react';

const ESCROW_STEPS = [
  'Buyer sets total amount (item price if buying + traveler tip)',
  'Mnbarh adds 10% platform fee (covers insurance, escrow, support)',
  'Traveler accepts the amount',
  'Buyer pays via card or wallet — funds locked in escrow',
  'Traveler picks up / buys the item',
  'Traveler delivers and confirms',
  'Buyer confirms receipt → funds released to traveler within 24h',
];

const METHODS = [
  { icon: CreditCard, name: 'Credit/Debit Card', detail: 'Visa, Mastercard' },
  { icon: Wallet, name: 'Mnbarh Wallet', detail: 'Instant' },
  { icon: CreditCard, name: 'PayMob', detail: 'Egypt & MENA' },
  { icon: Clock, name: 'PayPal', detail: 'Coming soon' },
];

const REFUNDS = [
  { case: 'Traveler cancels', result: '100% refund immediately', icon: RefreshCw },
  { case: 'Item not delivered', result: '100% refund after dispute resolution', icon: ShieldCheck },
  { case: 'Item not as described', result: 'Refund after investigation (3–5 days)', icon: CheckCircle },
];

export default function CrowdshippingPaymentPage() {
  return (
    <div className="min-h-screen">
      {/* Hero */}
      <section className="bg-gradient-to-br from-[#0071CE] to-[#003f75] text-white">
        <div className="mx-auto max-w-5xl px-4 py-14 text-center">
          <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-white/15 px-4 py-1.5 text-sm font-medium">
            <Lock size={16} /> Secure Payment
          </div>
          <h1 className="text-3xl font-extrabold md:text-4xl">Pay Securely — Your Money Is Protected Until Delivery</h1>
          <p className="mx-auto mt-3 max-w-xl text-blue-100">
            Your payment goes into Mnbarh&apos;s secure escrow — held safely and only released after you confirm delivery.
          </p>
        </div>
      </section>

      {/* How Escrow Works */}
      <section className="py-14">
        <div className="mx-auto max-w-3xl px-4">
          <h2 className="mb-6 text-2xl font-extrabold text-gray-900">How Escrow Works</h2>
          <p className="mb-6 text-sm text-gray-600 leading-relaxed">
            When you pay for a crowdshipping request, your money goes into Mnbarh&apos;s secure escrow — it&apos;s held safely and only released to the traveler after YOU confirm delivery. If anything goes wrong, your money comes back to you.
          </p>
          <div className="space-y-3">
            {ESCROW_STEPS.map((step, i) => (
              <div key={i} className="flex items-center gap-3 rounded-xl border border-gray-200 bg-white p-4">
                <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-[#0071CE] text-xs font-bold text-white">{i + 1}</span>
                <p className="text-sm text-gray-700">{step}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Payment Methods */}
      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-4xl px-4">
          <h2 className="mb-8 text-center text-2xl font-extrabold text-gray-900">Payment Methods Accepted</h2>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {METHODS.map((m) => (
              <div key={m.name} className="rounded-2xl border border-gray-200 bg-white p-5 text-center">
                <m.icon size={22} className="mx-auto mb-2 text-[#0071CE]" />
                <h3 className="text-sm font-bold text-gray-900">{m.name}</h3>
                <p className="text-xs text-gray-500">{m.detail}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Refund Cases */}
      <section className="py-14">
        <div className="mx-auto max-w-3xl px-4">
          <h2 className="mb-6 text-2xl font-extrabold text-gray-900">Refund Cases</h2>
          <div className="space-y-3">
            {REFUNDS.map((r) => (
              <div key={r.case} className="flex items-center gap-4 rounded-xl border border-gray-200 bg-white p-4">
                <r.icon size={20} className="shrink-0 text-emerald-600" />
                <div>
                  <p className="text-sm font-semibold text-gray-900">{r.case}</p>
                  <p className="text-xs text-gray-600">{r.result}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Security Note */}
      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-3xl px-4">
          <div className="rounded-2xl border border-[#0071CE]/20 bg-[#0071CE]/5 p-6 text-center">
            <Lock size={24} className="mx-auto mb-2 text-[#0071CE]" />
            <p className="text-sm text-gray-700">
              Your payment data is encrypted by Stripe. Mnbarh never stores your card number or CVV.
            </p>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="bg-gradient-to-r from-[#0071CE] to-[#003f75] py-14 text-center text-white">
        <h2 className="text-2xl font-extrabold">Pay with Confidence</h2>
        <div className="mt-6 flex flex-wrap justify-center gap-4">
          <Link href="/buy-via-traveler" className="inline-flex items-center gap-2 rounded-full bg-[#FFC220] px-8 py-3.5 text-sm font-bold text-gray-900 hover:bg-yellow-400">
            Post a Request <ArrowRight size={16} />
          </Link>
          <Link href="/crowdshipping/insurance" className="inline-flex items-center gap-2 rounded-full border-2 border-white/30 px-8 py-3.5 text-sm font-bold text-white hover:bg-white/10">
            Insurance Coverage
          </Link>
        </div>
      </section>
    </div>
  );
}
