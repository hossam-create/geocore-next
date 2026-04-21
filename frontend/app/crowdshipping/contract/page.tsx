import Link from 'next/link';
import { FileText, CheckCircle, Download, ArrowRight, ShieldCheck } from 'lucide-react';

const CONTRACT_ITEMS = [
  'Item description and declared value',
  'Traveler and buyer names and contact info',
  'Trip route and estimated delivery date',
  'Agreed tip amount',
  'Platform fee breakdown',
  'Insurance coverage confirmation',
  'Dispute resolution procedure',
];

export default function CrowdshippingContractPage() {
  return (
    <div className="min-h-screen">
      {/* Hero */}
      <section className="bg-gradient-to-br from-[#0071CE] to-[#003f75] text-white">
        <div className="mx-auto max-w-5xl px-4 py-14 text-center">
          <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-white/15 px-4 py-1.5 text-sm font-medium">
            <FileText size={16} /> Transport Contract
          </div>
          <h1 className="text-3xl font-extrabold md:text-4xl">Your Trip, Formalized</h1>
          <p className="mx-auto mt-3 max-w-xl text-blue-100">
            Once payment is made, Mnbarh automatically generates a Crowdshipping Transport Contract between buyer and traveler.
          </p>
        </div>
      </section>

      {/* What the Contract Includes */}
      <section className="py-14">
        <div className="mx-auto max-w-3xl px-4">
          <h2 className="mb-6 text-2xl font-extrabold text-gray-900">What the Contract Includes</h2>
          <div className="space-y-3">
            {CONTRACT_ITEMS.map((item, i) => (
              <div key={i} className="flex items-center gap-3 rounded-xl border border-gray-200 bg-white p-4">
                <CheckCircle size={18} className="shrink-0 text-[#0071CE]" />
                <p className="text-sm text-gray-700">{item}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Why It Matters */}
      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-3xl px-4">
          <h2 className="mb-6 text-2xl font-extrabold text-gray-900">Why It Matters</h2>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="rounded-2xl border border-gray-200 bg-white p-5">
              <ShieldCheck size={20} className="mb-2 text-[#0071CE]" />
              <h3 className="text-sm font-bold text-gray-900">Confirms What Was Agreed</h3>
              <p className="mt-1 text-xs text-gray-600">Both parties have a clear record of the terms — no misunderstandings.</p>
            </div>
            <div className="rounded-2xl border border-gray-200 bg-white p-5">
              <FileText size={20} className="mb-2 text-[#0071CE]" />
              <h3 className="text-sm font-bold text-gray-900">Lists Responsibilities</h3>
              <p className="mt-1 text-xs text-gray-600">Each party knows exactly what they need to do and when.</p>
            </div>
            <div className="rounded-2xl border border-gray-200 bg-white p-5">
              <Download size={20} className="mb-2 text-[#0071CE]" />
              <h3 className="text-sm font-bold text-gray-900">Printable for Customs</h3>
              <p className="mt-1 text-xs text-gray-600">Can be printed and presented at customs if needed for declaration.</p>
            </div>
            <div className="rounded-2xl border border-gray-200 bg-white p-5">
              <ShieldCheck size={20} className="mb-2 text-[#0071CE]" />
              <h3 className="text-sm font-bold text-gray-900">Evidence in Disputes</h3>
              <p className="mt-1 text-xs text-gray-600">Serves as official evidence if a dispute arises between parties.</p>
            </div>
          </div>
        </div>
      </section>

      {/* Auto-Generated */}
      <section className="py-14">
        <div className="mx-auto max-w-3xl px-4">
          <div className="rounded-2xl border border-[#0071CE]/20 bg-[#0071CE]/5 p-6 text-center">
            <FileText size={24} className="mx-auto mb-2 text-[#0071CE]" />
            <h3 className="text-sm font-bold text-gray-900">Automatically Generated</h3>
            <p className="mt-2 text-sm text-gray-600">
              The contract is generated automatically when payment is made and sent to both parties by email. No manual steps required.
            </p>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="bg-gradient-to-r from-[#0071CE] to-[#003f75] py-14 text-center text-white">
        <h2 className="text-2xl font-extrabold">Ship with Full Documentation</h2>
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
