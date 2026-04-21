import Link from 'next/link';
import { Shield, Lock, Eye, AlertTriangle, Fingerprint, Key, Smartphone, Mail, ArrowRight } from 'lucide-react';

const SECTIONS = [
  {
    icon: Lock,
    title: 'Account Security',
    items: ['Enable two-factor authentication (2FA) for extra protection', 'Use a strong, unique password — at least 10 characters', 'Review active sessions and log out of unfamiliar devices', 'Never share your password with anyone'],
    color: 'text-[#0071CE]',
    bg: 'bg-[#0071CE]/10',
  },
  {
    icon: Eye,
    title: 'Safe Transactions',
    items: ['Always pay through Mnbarh checkout — never pay outside the platform', 'Check seller trust scores and reviews before buying', 'Use escrow for high-value items', 'Report suspicious listings immediately'],
    color: 'text-emerald-600',
    bg: 'bg-emerald-100',
  },
  {
    icon: AlertTriangle,
    title: 'Spotting Scams',
    items: ['Deals that seem too good to be true usually are', 'Sellers asking for payment outside Mnbarh are likely scammers', 'Fake "Mnbarh support" emails — we never ask for your password', 'Phishing links — always check the URL before logging in'],
    color: 'text-amber-600',
    bg: 'bg-amber-100',
  },
  {
    icon: Fingerprint,
    title: 'Our Security Measures',
    items: ['256-bit SSL encryption on all pages', 'Real-time fraud detection engine', 'Stripe-secured payment processing', 'Regular security audits and penetration testing'],
    color: 'text-violet-600',
    bg: 'bg-violet-100',
  },
];

export default function SecurityCenterPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10">
      <div className="mb-8 text-center">
        <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-[#0071CE]/10 px-4 py-1.5 text-sm font-medium text-[#0071CE]">
          <Shield size={16} /> Security Center
        </div>
        <h1 className="text-3xl font-extrabold text-gray-900">Stay Safe on Mnbarh</h1>
        <p className="mt-2 text-sm text-gray-500 max-w-xl mx-auto">
          Your security is our top priority. Learn how to protect your account and spot potential threats.
        </p>
      </div>

      {/* Warning Box */}
      <div className="mb-10 rounded-2xl border border-amber-200 bg-amber-50 p-6">
        <div className="flex items-start gap-3">
          <AlertTriangle size={20} className="mt-0.5 shrink-0 text-amber-600" />
          <div>
            <h3 className="text-sm font-bold text-amber-800">Important Safety Notice</h3>
            <p className="mt-1 text-sm text-amber-700">Mnbarh will <strong>NEVER</strong> ask for your password by email or phone. Mnbarh will <strong>NEVER</strong> ask you to pay outside our secure checkout.</p>
          </div>
        </div>
      </div>

      {/* Security Sections */}
      <section className="mb-10">
        <div className="grid gap-6 sm:grid-cols-2">
          {SECTIONS.map((sec) => (
            <div key={sec.title} className="rounded-2xl border border-gray-200 bg-white p-6">
              <div className={`mb-3 inline-flex h-10 w-10 items-center justify-center rounded-xl ${sec.bg}`}>
                <sec.icon size={20} className={sec.color} />
              </div>
              <h3 className="text-base font-bold text-gray-900">{sec.title}</h3>
              <ul className="mt-3 space-y-2">
                {sec.items.map((item) => (
                  <li key={item} className="flex items-start gap-2 text-xs text-gray-600">
                    <span className="mt-0.5 text-emerald-500">✓</span> {item}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </section>

      {/* Report */}
      <section className="mb-10 rounded-2xl border border-red-200 bg-red-50 p-6 text-center">
        <h3 className="text-sm font-bold text-red-800">Report Suspicious Activity</h3>
        <p className="mt-1 text-sm text-red-700">safety@mnbarh.com — Reviewed within 2 hours, 24/7</p>
      </section>

      {/* CTA */}
      <section className="text-center">
        <Link href="/help" className="inline-block rounded-xl bg-[#0071CE] px-6 py-3 text-sm font-semibold text-white hover:bg-[#005ba3]">
          Visit Help Center
        </Link>
        <div className="mt-3 flex justify-center gap-4 text-xs">
          <Link href="/buyer-protection" className="text-[#0071CE] hover:underline">Buyer Protection</Link>
          <span className="text-gray-300">|</span>
          <Link href="/seller-protection" className="text-[#0071CE] hover:underline">Seller Protection</Link>
          <span className="text-gray-300">|</span>
          <Link href="/crowdshipping/trust" className="text-[#0071CE] hover:underline">Crowdshipping Trust</Link>
        </div>
      </section>
    </div>
  );
}
