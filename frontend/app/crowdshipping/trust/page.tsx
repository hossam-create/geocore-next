import Link from 'next/link';
import { ShieldCheck, Mail, Phone, FileCheck, Users, Star, AlertTriangle, Eye, MapPin, MessageCircle, Camera, Globe, ThumbsUp, ArrowRight } from 'lucide-react';

const VERIFICATION = [
  { icon: Mail, title: 'Email Verification', desc: 'Required for all users', level: 'Required' },
  { icon: Phone, title: 'Phone Verification', desc: 'SMS confirmation required', level: 'Required' },
  { icon: FileCheck, title: 'Government ID', desc: 'Required for international shipments', level: 'International' },
  { icon: Users, title: 'Social Account Linking', desc: 'Increases your trust score', level: 'Optional' },
  { icon: Star, title: 'First Transaction Review', desc: 'Our team reviews your first delivery', level: 'Automatic' },
];

const TRUST_LEVELS = ['Beginner', 'Verified', 'Trusted', 'Top Rated', 'Mnbarh Champion'];

const RULES = [
  { icon: Eye, title: 'Only carry what you can see', desc: 'Never accept a sealed package without inspecting contents. You have the right to refuse any delivery at any time.' },
  { icon: MapPin, title: 'Meet in public places', desc: 'First-time meetups should always be in a public location — a café, mall, or airport terminal.' },
  { icon: MessageCircle, title: 'Communicate only through Mnbarh chat', desc: 'All conversations are securely stored. Off-platform communication removes your protection in case of disputes.' },
  { icon: Camera, title: 'Document everything', desc: 'Take photos of the item before and after. Keep all receipts. These are your evidence in case of disputes.' },
  { icon: Globe, title: 'Understand customs rules', desc: 'Both parties are responsible for customs compliance. Mnbarh provides a customs guide per country.' },
  { icon: AlertTriangle, title: 'Report suspicious requests', desc: 'If a request seems unusual, trust your instincts and decline. Report it to our team — we review all reports within 2 hours.' },
  { icon: ThumbsUp, title: 'Respect capacity limits', desc: 'Be honest about what you can carry. Overloading creates risk. Maximum item size: fits in a standard 20 kg luggage.' },
  { icon: Star, title: 'Rate honestly after every delivery', desc: 'Honest ratings protect the entire community. Retaliatory ratings are investigated and removed.' },
];

export default function CrowdshippingTrustPage() {
  return (
    <div className="min-h-screen">
      {/* Hero */}
      <section className="bg-gradient-to-br from-[#0071CE] to-[#003f75] text-white">
        <div className="mx-auto max-w-5xl px-4 py-14 text-center">
          <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-white/15 px-4 py-1.5 text-sm font-medium">
            <ShieldCheck size={16} /> Safety & Trust
          </div>
          <h1 className="text-3xl font-extrabold md:text-4xl">Your Safety Is Our #1 Priority</h1>
          <p className="mx-auto mt-3 max-w-xl text-blue-100">
            Every traveler and buyer goes through multiple verification steps before their first transaction.
          </p>
        </div>
      </section>

      {/* Verification Layers */}
      <section className="py-14">
        <div className="mx-auto max-w-4xl px-4">
          <h2 className="mb-8 text-center text-2xl font-extrabold text-gray-900">Verification Layers</h2>
          <div className="space-y-3">
            {VERIFICATION.map((v) => (
              <div key={v.title} className="flex items-center gap-4 rounded-2xl border border-gray-200 bg-white p-5">
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-[#0071CE]/10">
                  <v.icon size={20} className="text-[#0071CE]" />
                </div>
                <div className="flex-1">
                  <h3 className="text-sm font-bold text-gray-900">{v.title}</h3>
                  <p className="text-xs text-gray-600">{v.desc}</p>
                </div>
                <span className="rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600">{v.level}</span>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Trust Rating System */}
      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-4xl px-4">
          <h2 className="mb-4 text-center text-2xl font-extrabold text-gray-900">Trust Rating System</h2>
          <p className="mb-8 text-center text-sm text-gray-500">After each delivery, both parties rate each other. Ratings are public and cannot be deleted.</p>
          <div className="flex flex-wrap justify-center gap-3">
            {TRUST_LEVELS.map((level, i) => (
              <div key={level} className="flex items-center gap-2 rounded-full border border-gray-200 bg-white px-5 py-2.5">
                <span className="flex h-6 w-6 items-center justify-center rounded-full bg-[#0071CE] text-xs font-bold text-white">{i + 1}</span>
                <span className="text-sm font-semibold text-gray-900">{level}</span>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* 8 Rules */}
      <section className="py-14">
        <div className="mx-auto max-w-4xl px-4">
          <h2 className="mb-8 text-center text-2xl font-extrabold text-gray-900">The 8 Rules of Safe Crowdshipping</h2>
          <div className="grid gap-4 sm:grid-cols-2">
            {RULES.map((rule) => (
              <div key={rule.title} className="rounded-2xl border border-gray-200 bg-white p-5">
                <div className="mb-2 flex items-center gap-2">
                  <rule.icon size={18} className="text-[#0071CE]" />
                  <h3 className="text-sm font-bold text-gray-900">{rule.title}</h3>
                </div>
                <p className="text-xs text-gray-600 leading-relaxed">{rule.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Emergency Contact */}
      <section className="py-14">
        <div className="mx-auto max-w-3xl px-4">
          <div className="rounded-2xl border border-red-200 bg-red-50 p-6 text-center">
            <AlertTriangle size={24} className="mx-auto mb-2 text-red-600" />
            <h3 className="text-sm font-bold text-red-800">Emergency Safety Contact</h3>
            <p className="mt-2 text-sm text-red-700">safety@mnbarh.com</p>
            <p className="text-xs text-red-600">Our Safety Team is available 24/7</p>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="bg-gradient-to-r from-[#0071CE] to-[#003f75] py-14 text-center text-white">
        <h2 className="text-2xl font-extrabold">Feel Safe. Ship Confidently.</h2>
        <div className="mt-6 flex flex-wrap justify-center gap-4">
          <Link href="/buy-via-traveler" className="inline-flex items-center gap-2 rounded-full bg-[#FFC220] px-8 py-3.5 text-sm font-bold text-gray-900 hover:bg-yellow-400">
            Post a Request <ArrowRight size={16} />
          </Link>
          <Link href="/crowdshipping/for-travelers" className="inline-flex items-center gap-2 rounded-full border-2 border-white/30 px-8 py-3.5 text-sm font-bold text-white hover:bg-white/10">
            Become a Traveler
          </Link>
        </div>
      </section>
    </div>
  );
}
