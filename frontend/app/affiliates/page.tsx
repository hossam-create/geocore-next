import Link from 'next/link';
import { Users, DollarSign, Link2, Share2, ArrowRight, CheckCircle } from 'lucide-react';

const STEPS = [
  { icon: Users, title: 'Sign Up for Free', desc: 'Register at /affiliates/register — instant approval.' },
  { icon: Link2, title: 'Get Your Unique Link', desc: 'Receive a personalized referral link to share.' },
  { icon: Share2, title: 'Share with Your Audience', desc: 'Blog, social media, email — anywhere your audience is.' },
  { icon: DollarSign, title: 'Earn Commissions', desc: 'Get paid for every buyer and seller you refer.' },
];

const COMMISSIONS = [
  { action: 'New buyer registers', earning: '$1 flat bonus' },
  { action: 'New buyer first purchase', earning: '2% of order value' },
  { action: 'New seller first listing', earning: '$5 flat bonus' },
  { action: 'New seller first sale', earning: '1% of sale value' },
];

const WHO = [
  'Bloggers and content creators',
  'Social media influencers',
  'Price comparison website owners',
  'Anyone with an audience interested in shopping',
];

export default function AffiliatesPage() {
  return (
    <div className="min-h-screen">
      <section className="bg-gradient-to-br from-[#FFC220]/30 to-[#FFC220]/5 py-14">
        <div className="mx-auto max-w-5xl px-4 text-center">
          <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-[#FFC220]/30 px-4 py-1.5 text-sm font-medium text-gray-900">
            <Users size={16} /> Affiliates
          </div>
          <h1 className="text-3xl font-extrabold md:text-4xl">Earn by Referring — Unlimited Income Potential</h1>
          <p className="mx-auto mt-3 max-w-xl text-sm text-gray-600">
            Share Mnbarh with your audience and earn commissions on every new buyer and seller you bring.
          </p>
          <Link href="/register" className="mt-6 inline-flex items-center gap-2 rounded-full bg-[#FFC220] px-8 py-3.5 text-sm font-bold text-gray-900 hover:bg-yellow-400">
            Join Affiliate Program <ArrowRight size={16} />
          </Link>
        </div>
      </section>

      {/* How It Works */}
      <section className="py-14">
        <div className="mx-auto max-w-4xl px-4">
          <h2 className="mb-8 text-center text-2xl font-extrabold text-gray-900">How It Works</h2>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {STEPS.map((s) => (
              <div key={s.title} className="rounded-2xl border border-gray-200 bg-white p-5 text-center">
                <s.icon size={22} className="mx-auto mb-2 text-[#FFC220]" />
                <h3 className="text-sm font-bold text-gray-900">{s.title}</h3>
                <p className="mt-1 text-xs text-gray-600">{s.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Commission Table */}
      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-3xl px-4">
          <h2 className="mb-6 text-center text-2xl font-extrabold text-gray-900">Commission Structure</h2>
          <div className="overflow-hidden rounded-xl border border-gray-200">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-gray-100 text-xs uppercase text-gray-500">
                  <th className="px-5 py-3 text-left">Action</th>
                  <th className="px-5 py-3 text-right">Your Earnings</th>
                </tr>
              </thead>
              <tbody>
                {COMMISSIONS.map((c) => (
                  <tr key={c.action} className="border-t border-gray-100">
                    <td className="px-5 py-3 text-gray-700">{c.action}</td>
                    <td className="px-5 py-3 text-right font-semibold text-emerald-700">{c.earning}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="mt-4 text-xs text-gray-500">Payment: Monthly via Mnbarh Wallet or bank transfer · Minimum payout: $20 · Cookie duration: 30 days</p>
        </div>
      </section>

      {/* Who Should Apply */}
      <section className="py-14">
        <div className="mx-auto max-w-3xl px-4">
          <h2 className="mb-4 text-2xl font-extrabold text-gray-900">Who Should Apply</h2>
          <ul className="space-y-2">
            {WHO.map((w) => (
              <li key={w} className="flex items-center gap-2 text-sm text-gray-700">
                <CheckCircle size={16} className="text-emerald-500" /> {w}
              </li>
            ))}
          </ul>
        </div>
      </section>
    </div>
  );
}
