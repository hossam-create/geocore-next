import Link from 'next/link';
import { Shield, Eye, Lock, AlertTriangle, Plane, Flag, ArrowRight } from 'lucide-react';

const SECTIONS = [
  { icon: Eye, title: 'Buying Safely', desc: 'Recognize fake listings, verify sellers, and always pay through Mnbarh checkout.', color: 'text-[#0071CE]', bg: 'bg-[#0071CE]/10' },
  { icon: Lock, title: 'Selling Safely', desc: 'Recognize scam buyers, avoid payment fraud, and ship only to verified addresses.', color: 'text-emerald-600', bg: 'bg-emerald-100' },
  { icon: Shield, title: 'Account Security', desc: 'Enable 2FA, recognize phishing emails, and protect your credentials.', color: 'text-amber-600', bg: 'bg-amber-100' },
  { icon: Plane, title: 'Crowdshipping Safety', desc: 'Meet in public, inspect items, and communicate only through Mnbarh chat.', color: 'text-violet-600', bg: 'bg-violet-100' },
  { icon: Flag, title: 'Report an Issue', desc: 'How to report fraud, scams, and abuse. Our team reviews within 2 hours.', color: 'text-red-600', bg: 'bg-red-100' },
];

export default function SafetyCenterPage() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <div className="mb-8 text-center">
        <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-red-100 px-4 py-1.5 text-sm font-medium text-red-700">
          <Shield size={16} /> Safety Center
        </div>
        <h1 className="text-3xl font-extrabold text-gray-900">Stay Safe on Mnbarh</h1>
        <p className="mt-2 text-sm text-gray-500">Everything you need to know about buying, selling, and shipping safely.</p>
      </div>

      {/* Emergency */}
      <div className="mb-8 rounded-2xl border border-red-200 bg-red-50 p-5 text-center">
        <AlertTriangle size={20} className="mx-auto mb-1 text-red-600" />
        <p className="text-sm font-bold text-red-800">Emergency: safety@mnbarh.com · Available 24/7</p>
      </div>

      <section className="mb-10">
        <div className="space-y-4">
          {SECTIONS.map((s) => (
            <Link key={s.title} href={s.title === 'Crowdshipping Safety' ? '/crowdshipping/trust' : s.title === 'Account Security' ? '/security-center' : s.title === 'Report an Issue' ? '/contact' : s.title === 'Buying Safely' ? '/buyer-protection' : '/seller-protection'} className="flex items-center gap-4 rounded-2xl border border-gray-200 bg-white p-5 hover:border-[#0071CE] transition-colors">
              <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-xl ${s.bg}`}>
                <s.icon size={20} className={s.color} />
              </div>
              <div className="flex-1">
                <h3 className="text-sm font-bold text-gray-900">{s.title}</h3>
                <p className="text-xs text-gray-600">{s.desc}</p>
              </div>
              <ArrowRight size={16} className="text-gray-400" />
            </Link>
          ))}
        </div>
      </section>

      <section className="text-center">
        <Link href="/help" className="inline-block rounded-xl bg-[#0071CE] px-6 py-3 text-sm font-semibold text-white hover:bg-[#005ba3]">
          Help Center
        </Link>
      </section>
    </div>
  );
}
