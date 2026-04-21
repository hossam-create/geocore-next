import Link from 'next/link';
import { Plane, ShieldCheck, DollarSign, Package, MessageCircle, Star, Globe, Users, ArrowRight, CheckCircle } from 'lucide-react';

const STEPS = [
  {
    icon: Package,
    title: 'Post Your Request',
    desc: 'Tell us what you want, where it\'s from, and what you\'ll pay as a tip. It takes 2 minutes.',
    color: 'text-[#0071CE]',
    bg: 'bg-[#0071CE]/10',
  },
  {
    icon: Globe,
    title: 'Match with a Traveler',
    desc: 'We connect you with verified travelers already heading that way. Chat, agree on details, confirm.',
    color: 'text-emerald-600',
    bg: 'bg-emerald-100',
  },
  {
    icon: ShieldCheck,
    title: 'Secure Payment',
    desc: 'Pay through our secure escrow system. Your money is locked until delivery is confirmed. You\'re always protected.',
    color: 'text-amber-600',
    bg: 'bg-amber-100',
  },
  {
    icon: Plane,
    title: 'Receive Your Item',
    desc: 'The traveler delivers to your door or a meeting point. Confirm receipt, release payment. Done.',
    color: 'text-violet-600',
    bg: 'bg-violet-100',
  },
];

const COMPARISON = [
  { method: 'International Post', cost: '$45 – $80', highlight: false },
  { method: 'DHL Express', cost: '$90 – $150', highlight: false },
  { method: 'Mnbarh Traveler', cost: '$15 – $30 tip', highlight: true },
];

const STORIES = [
  {
    name: 'Ahmed',
    location: 'Cairo',
    story: 'Wanted the latest iPhone not yet available in Egypt. Connected with Sara, a student returning from London, who bought it for him — saving $200 vs local import prices.',
    emoji: '📱',
  },
  {
    name: 'Mariam',
    location: 'Dubai',
    story: 'Missed her mom\'s homemade foods from Cairo. Found Khaled, a frequent traveler on the Cairo–Dubai route, who brings her favorite items monthly for a small tip.',
    emoji: '🍽️',
  },
  {
    name: 'Youssef',
    location: 'Cairo',
    story: 'Needed a specific spare part for his vintage car only sold in Germany. Thomas, a German-Egyptian returning from Berlin, picked it up for a $20 tip vs $200 in shipping.',
    emoji: '🔧',
  },
];

const TRUST_STATS = [
  { label: 'Verified Travelers', value: '50,000+', icon: Users },
  { label: 'Countries Covered', value: '120+', icon: Globe },
  { label: 'Saved in Shipping', value: '$5M+', icon: DollarSign },
  { label: 'Average Rating', value: '4.8/5', icon: Star },
];

export default function BuyViaTravelerPage() {
  return (
    <div className="min-h-screen">
      {/* Hero */}
      <section className="bg-gradient-to-br from-[#0071CE] to-[#003f75] text-white">
        <div className="mx-auto max-w-6xl px-4 py-16 text-center">
          <div className="mb-4 inline-flex items-center gap-2 rounded-full bg-white/15 px-4 py-1.5 text-sm font-medium">
            <Plane size={16} /> Buy via Traveler
          </div>
          <h1 className="text-4xl font-extrabold md:text-5xl">
            Buy Anything From Anywhere —<br />Delivered by Real Travelers
          </h1>
          <p className="mx-auto mt-4 max-w-2xl text-lg text-blue-100">
            Can&apos;t find it locally? Our global community of travelers will buy it and bring it to you — at a fraction of courier costs.
          </p>
          <div className="mt-8 flex flex-wrap justify-center gap-4">
            <Link href="/crowdshipping" className="inline-flex items-center gap-2 rounded-full bg-[#FFC220] px-8 py-3.5 text-sm font-bold text-gray-900 hover:bg-yellow-400 transition-colors">
              Find a Traveler <ArrowRight size={16} />
            </Link>
            <Link href="/crowdshipping/for-travelers" className="inline-flex items-center gap-2 rounded-full border-2 border-white/30 px-8 py-3.5 text-sm font-bold text-white hover:bg-white/10 transition-colors">
              I&apos;m a Traveler — Earn Money
            </Link>
          </div>
        </div>
      </section>

      {/* Trust Stats Bar */}
      <section className="border-b border-gray-100 bg-white">
        <div className="mx-auto grid max-w-6xl grid-cols-2 gap-4 px-4 py-6 sm:grid-cols-4">
          {TRUST_STATS.map((stat) => (
            <div key={stat.label} className="text-center">
              <stat.icon size={20} className="mx-auto mb-1 text-[#0071CE]" />
              <p className="text-2xl font-extrabold text-gray-900">{stat.value}</p>
              <p className="text-xs text-gray-500">{stat.label}</p>
            </div>
          ))}
        </div>
      </section>

      {/* How It Works */}
      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-5xl px-4">
          <h2 className="mb-8 text-center text-2xl font-extrabold text-gray-900">How It Works</h2>
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
            {STEPS.map((step, i) => (
              <div key={step.title} className="relative rounded-2xl border border-gray-200 bg-white p-5">
                {i < STEPS.length - 1 && (
                  <div className="absolute -right-3 top-1/2 hidden h-6 text-gray-300 lg:block">&rarr;</div>
                )}
                <div className={`mb-3 inline-flex h-11 w-11 items-center justify-center rounded-xl ${step.bg}`}>
                  <step.icon size={22} className={step.color} />
                </div>
                <h3 className="text-sm font-bold text-gray-900">{step.title}</h3>
                <p className="mt-1 text-xs text-gray-600 leading-relaxed">{step.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Cost Comparison */}
      <section className="py-14">
        <div className="mx-auto max-w-3xl px-4">
          <h2 className="mb-2 text-center text-2xl font-extrabold text-gray-900">Why It&apos;s Cheaper</h2>
          <p className="mb-8 text-center text-sm text-gray-500">Cost comparison for a 1 kg package: Cairo → London</p>
          <div className="overflow-hidden rounded-2xl border border-gray-200">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-gray-50 text-xs uppercase text-gray-500">
                  <th className="px-6 py-3 text-left">Delivery Method</th>
                  <th className="px-6 py-3 text-right">Cost</th>
                </tr>
              </thead>
              <tbody>
                {COMPARISON.map((row) => (
                  <tr key={row.method} className={`border-t border-gray-100 ${row.highlight ? 'bg-emerald-50' : ''}`}>
                    <td className="px-6 py-4 font-medium text-gray-900">
                      {row.highlight && <CheckCircle size={14} className="mr-2 inline text-emerald-600" />}
                      {row.method}
                    </td>
                    <td className={`px-6 py-4 text-right font-bold ${row.highlight ? 'text-emerald-700' : 'text-gray-700'}`}>
                      {row.cost}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </section>

      {/* Real Stories */}
      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-5xl px-4">
          <h2 className="mb-8 text-center text-2xl font-extrabold text-gray-900">Real Stories from Our Community</h2>
          <div className="grid gap-6 sm:grid-cols-3">
            {STORIES.map((s) => (
              <div key={s.name} className="rounded-2xl border border-gray-200 bg-white p-5">
                <div className="mb-3 text-3xl">{s.emoji}</div>
                <h3 className="text-sm font-bold text-gray-900">{s.name} from {s.location}</h3>
                <p className="mt-2 text-xs text-gray-600 leading-relaxed">{s.story}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* What Can Be Sent */}
      <section className="py-14">
        <div className="mx-auto max-w-4xl px-4">
          <h2 className="mb-8 text-center text-2xl font-extrabold text-gray-900">What Can Be Sent</h2>
          <div className="grid gap-6 sm:grid-cols-2">
            <div className="rounded-2xl border border-emerald-200 bg-emerald-50 p-6">
              <h3 className="mb-3 flex items-center gap-2 text-sm font-bold text-emerald-800">
                <CheckCircle size={16} /> Allowed Items
              </h3>
              <ul className="space-y-1.5 text-sm text-emerald-700">
                <li>✅ Electronics, clothes, cosmetics, books</li>
                <li>✅ Sealed food items</li>
                <li>✅ Documents, jewelry, small gifts</li>
                <li>✅ Items up to 5 kg per package</li>
              </ul>
            </div>
            <div className="rounded-2xl border border-red-200 bg-red-50 p-6">
              <h3 className="mb-3 flex items-center gap-2 text-sm font-bold text-red-800">
                <span className="text-red-600">✕</span> Prohibited Items
              </h3>
              <ul className="space-y-1.5 text-sm text-red-700">
                <li>❌ Weapons, drugs, currency</li>
                <li>❌ Live animals</li>
                <li>❌ Items over 5 kg</li>
                <li>❌ Anything prohibited by destination customs</li>
              </ul>
            </div>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="bg-gradient-to-r from-[#0071CE] to-[#003f75] py-14 text-center text-white">
        <h2 className="text-2xl font-extrabold">Ready to Save on Shipping?</h2>
        <p className="mt-2 text-blue-100">Post your request in 2 minutes — travelers are waiting.</p>
        <div className="mt-6 flex flex-wrap justify-center gap-4">
          <Link href="/crowdshipping" className="inline-flex items-center gap-2 rounded-full bg-[#FFC220] px-8 py-3.5 text-sm font-bold text-gray-900 hover:bg-yellow-400">
            Post a Request <ArrowRight size={16} />
          </Link>
          <Link href="/crowdshipping/for-travelers" className="inline-flex items-center gap-2 rounded-full border-2 border-white/30 px-8 py-3.5 text-sm font-bold text-white hover:bg-white/10">
            Become a Traveler
          </Link>
        </div>
        <div className="mt-6 flex justify-center gap-6 text-xs text-blue-200">
          <Link href="/crowdshipping/trust" className="hover:text-white">Safety & Trust</Link>
          <Link href="/crowdshipping/insurance" className="hover:text-white">Insurance</Link>
          <Link href="/crowdshipping/payment" className="hover:text-white">Secure Payment</Link>
          <Link href="/buyer-protection" className="hover:text-white">Buyer Protection</Link>
        </div>
      </section>
    </div>
  );
}
