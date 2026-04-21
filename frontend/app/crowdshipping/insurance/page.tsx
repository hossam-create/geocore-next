import Link from 'next/link';
import { Shield, CheckCircle, XCircle, ArrowRight, FileText, Clock, Mail, DollarSign } from 'lucide-react';

const COVERAGE = [
  { icon: Shield, title: 'Automatic Coverage', desc: 'All items shipped through Mnbarh escrow are automatically insured.' },
  { icon: DollarSign, title: 'Up to $1,500', desc: 'Coverage per shipment for loss, damage during transport, or theft.' },
  { icon: CheckCircle, title: 'Free — Included', desc: 'Insurance is included in Mnbarh\'s platform fee. No extra charge.' },
];

const ACTIVATION = [
  'Complete payment through Mnbarh escrow (required)',
  'Declare item value accurately when creating the request',
  'Coverage activates the moment the traveler accepts',
];

const NOT_COVERED = [
  'Items paid outside Mnbarh platform',
  'Prohibited items (weapons, drugs, currency)',
  'Deliberate damage by the buyer',
  'Customs confiscation (buyer\'s responsibility)',
];

const CLAIM_STEPS = [
  'Report within 48 hours of delivery (or expected delivery date)',
  'Provide: photos of damage, proof of value, conversation history',
  'Submit to: claims@mnbarh.com',
  'Response within 5 business days',
];

export default function CrowdshippingInsurancePage() {
  return (
    <div className="min-h-screen">
      {/* Hero */}
      <section className="bg-gradient-to-br from-[#0071CE] to-[#003f75] text-white">
        <div className="mx-auto max-w-5xl px-4 py-14 text-center">
          <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-white/15 px-4 py-1.5 text-sm font-medium">
            <Shield size={16} /> Insurance
          </div>
          <h1 className="text-3xl font-extrabold md:text-4xl">Every Delivery Insured — Up to $1,500</h1>
          <p className="mx-auto mt-3 max-w-xl text-blue-100">
            Automatic coverage for all crowdshipping deliveries. No extra cost, no extra steps.
          </p>
        </div>
      </section>

      {/* Coverage Details */}
      <section className="py-14">
        <div className="mx-auto max-w-4xl px-4">
          <div className="grid gap-6 sm:grid-cols-3">
            {COVERAGE.map((c) => (
              <div key={c.title} className="rounded-2xl border border-gray-200 bg-white p-6 text-center">
                <c.icon size={24} className="mx-auto mb-3 text-[#0071CE]" />
                <h3 className="text-sm font-bold text-gray-900">{c.title}</h3>
                <p className="mt-2 text-xs text-gray-600 leading-relaxed">{c.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* How Coverage Activates */}
      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-3xl px-4">
          <h2 className="mb-6 text-2xl font-extrabold text-gray-900">How Coverage Is Activated</h2>
          <div className="space-y-3">
            {ACTIVATION.map((step, i) => (
              <div key={i} className="flex items-center gap-3 rounded-xl border border-gray-200 bg-white p-4">
                <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-[#0071CE] text-xs font-bold text-white">{i + 1}</span>
                <p className="text-sm text-gray-700">{step}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Not Covered */}
      <section className="py-14">
        <div className="mx-auto max-w-3xl px-4">
          <h2 className="mb-6 text-2xl font-extrabold text-gray-900">Not Covered</h2>
          <div className="rounded-2xl border border-red-200 bg-red-50 p-6">
            <ul className="space-y-2">
              {NOT_COVERED.map((item) => (
                <li key={item} className="flex items-start gap-2 text-sm text-red-700">
                  <XCircle size={16} className="mt-0.5 shrink-0" /> {item}
                </li>
              ))}
            </ul>
          </div>
        </div>
      </section>

      {/* How to File a Claim */}
      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-3xl px-4">
          <h2 className="mb-6 text-2xl font-extrabold text-gray-900">How to File a Claim</h2>
          <div className="space-y-3">
            {CLAIM_STEPS.map((step, i) => (
              <div key={i} className="flex items-center gap-3 rounded-xl border border-gray-200 bg-white p-4">
                <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-amber-500 text-xs font-bold text-white">{i + 1}</span>
                <p className="text-sm text-gray-700">{step}</p>
              </div>
            ))}
          </div>
          <div className="mt-4 flex items-center gap-2 rounded-xl bg-white border border-gray-200 p-4">
            <Mail size={16} className="text-[#0071CE]" />
            <span className="text-sm text-gray-600">Submit claims to: <strong>claims@mnbarh.com</strong></span>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="bg-gradient-to-r from-[#0071CE] to-[#003f75] py-14 text-center text-white">
        <h2 className="text-2xl font-extrabold">Ship with Confidence</h2>
        <div className="mt-6 flex flex-wrap justify-center gap-4">
          <Link href="/buy-via-traveler" className="inline-flex items-center gap-2 rounded-full bg-[#FFC220] px-8 py-3.5 text-sm font-bold text-gray-900 hover:bg-yellow-400">
            Post a Request <ArrowRight size={16} />
          </Link>
          <Link href="/crowdshipping/trust" className="inline-flex items-center gap-2 rounded-full border-2 border-white/30 px-8 py-3.5 text-sm font-bold text-white hover:bg-white/10">
            Safety & Trust
          </Link>
        </div>
      </section>
    </div>
  );
}
