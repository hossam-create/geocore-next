'use client';
import Link from 'next/link';
import { useState } from 'react';
import { UserPlus, Search, Camera, Truck, Wallet, ChevronDown, Lightbulb, Star, TrendingUp, ArrowRight } from 'lucide-react';

const STEPS = [
  {
    icon: UserPlus,
    title: 'Create Your Account',
    desc: 'Register with email or phone. Verify your identity via SMS. Set up your payment method to receive earnings.',
    details: [
      'Sign up in under 2 minutes',
      'Phone verification required for selling',
      'Add your bank account or use Mnbarh Wallet',
    ],
  },
  {
    icon: Search,
    title: 'Research Your Item',
    desc: 'Check what similar items sold for. Use our AI pricing tool for a suggested price. Review category-specific fees.',
    details: [
      'Search completed listings for pricing benchmarks',
      'AI suggests optimal price based on demand',
      'Category fees vary — check before listing',
    ],
  },
  {
    icon: Camera,
    title: 'Create Your Listing',
    desc: 'Upload up to 10 photos (first photo is the thumbnail). Write a clear title with brand, model, and condition. Choose Fixed Price, Auction, or Dutch Auction.',
    details: [
      'First photo = thumbnail — make it count',
      'Include brand, model, size, and condition in title',
      'Set shipping options and return policy',
    ],
  },
  {
    icon: Truck,
    title: 'Manage Your Sale',
    desc: 'Respond to buyer messages within 24h. Ship within 3 business days of payment. Upload tracking number.',
    details: [
      'Fast responses increase conversion by 40%',
      'Ship promptly to maintain your seller rating',
      'Always upload tracking — it protects you in disputes',
    ],
  },
  {
    icon: Wallet,
    title: 'Get Paid',
    desc: 'Payment is released 3 days after buyer confirms delivery. For buyers who don\'t confirm, payment auto-releases after 14 days. Withdraw to bank or wallet.',
    details: [
      'Escrow protects both buyer and seller',
      'Auto-release after 14 days if no dispute',
      'Withdraw to bank (3-5 days) or wallet (instant)',
    ],
  },
];

const TIPS = [
  { icon: Star, text: 'Top Rated Sellers earn 10% more visibility in search results' },
  { icon: Camera, text: 'Listings with 5+ photos get 3× more views' },
  { icon: TrendingUp, text: 'Responding within 1 hour increases conversion by 40%' },
  { icon: Lightbulb, text: 'Auctions with a low starting bid attract more bidders and higher final prices' },
];

const FEE_TABLE = [
  { type: 'Fixed Price', listing: 'Free', success: '5%' },
  { type: 'Auction', listing: 'Free', success: '5%' },
  { type: 'Featured', listing: '$2.99', success: '5%' },
  { type: 'Business Seller', listing: '$0/month', success: '3%' },
];

export default function HowToSellPage() {
  const [openStep, setOpenStep] = useState<number | null>(0);

  return (
    <div className="min-h-screen">
      {/* Hero */}
      <section className="bg-gradient-to-br from-[#FFC220]/20 to-[#FFC220]/5 py-14">
        <div className="mx-auto max-w-4xl px-4 text-center">
          <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-[#FFC220]/30 px-4 py-1.5 text-sm font-medium text-gray-900">
            📖 Seller Guide
          </div>
          <h1 className="text-3xl font-extrabold text-gray-900 md:text-4xl">How to Sell on Mnbarh</h1>
          <p className="mx-auto mt-3 max-w-xl text-sm text-gray-600">
            From listing your first item to getting paid — here&apos;s everything you need to know to start selling successfully.
          </p>
          <Link href="/sell" className="mt-6 inline-flex items-center gap-2 rounded-full bg-[#FFC220] px-8 py-3 text-sm font-bold text-gray-900 hover:bg-yellow-400">
            Start Selling Now <ArrowRight size={16} />
          </Link>
        </div>
      </section>

      {/* Steps Accordion */}
      <section className="py-14">
        <div className="mx-auto max-w-3xl px-4">
          <h2 className="mb-8 text-center text-2xl font-extrabold text-gray-900">Step-by-Step Guide</h2>
          <div className="space-y-3">
            {STEPS.map((step, i) => (
              <div key={step.title} className="rounded-2xl border border-gray-200 bg-white overflow-hidden">
                <button
                  onClick={() => setOpenStep(openStep === i ? null : i)}
                  className="flex w-full items-center gap-4 p-5 text-left hover:bg-gray-50 transition-colors"
                >
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-[#FFC220]/20">
                    <step.icon size={20} className="text-gray-900" />
                  </div>
                  <div className="flex-1">
                    <span className="text-xs text-gray-400">Step {i + 1}</span>
                    <h3 className="text-sm font-bold text-gray-900">{step.title}</h3>
                  </div>
                  <ChevronDown size={18} className={`text-gray-400 transition-transform ${openStep === i ? 'rotate-180' : ''}`} />
                </button>
                {openStep === i && (
                  <div className="border-t border-gray-100 px-5 pb-5 pt-3">
                    <p className="text-sm text-gray-600 leading-relaxed">{step.desc}</p>
                    <ul className="mt-3 space-y-1.5">
                      {step.details.map((d) => (
                        <li key={d} className="flex items-start gap-2 text-xs text-gray-600">
                          <span className="mt-0.5 text-emerald-500">✓</span> {d}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Seller Tips */}
      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-4xl px-4">
          <h2 className="mb-8 text-center text-2xl font-extrabold text-gray-900">Pro Tips for Sellers</h2>
          <div className="grid gap-4 sm:grid-cols-2">
            {TIPS.map((tip) => (
              <div key={tip.text} className="flex items-start gap-3 rounded-xl border border-gray-200 bg-white p-4">
                <tip.icon size={18} className="mt-0.5 shrink-0 text-[#FFC220]" />
                <p className="text-sm text-gray-700">{tip.text}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Fee Table */}
      <section className="py-14">
        <div className="mx-auto max-w-3xl px-4">
          <h2 className="mb-2 text-center text-2xl font-extrabold text-gray-900">Fee Structure</h2>
          <p className="mb-8 text-center text-sm text-gray-500">Transparent fees — no hidden charges</p>
          <div className="overflow-hidden rounded-2xl border border-gray-200">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-gray-50 text-xs uppercase text-gray-500">
                  <th className="px-6 py-3 text-left">Listing Type</th>
                  <th className="px-6 py-3 text-center">Listing Fee</th>
                  <th className="px-6 py-3 text-center">Success Fee</th>
                </tr>
              </thead>
              <tbody>
                {FEE_TABLE.map((row) => (
                  <tr key={row.type} className="border-t border-gray-100">
                    <td className="px-6 py-3 font-medium text-gray-900">{row.type}</td>
                    <td className="px-6 py-3 text-center text-gray-600">{row.listing}</td>
                    <td className="px-6 py-3 text-center font-semibold text-gray-900">{row.success}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="bg-[#FFC220] py-14 text-center">
        <h2 className="text-2xl font-extrabold text-gray-900">Start Selling Today — It&apos;s Free to List</h2>
        <p className="mt-2 text-sm text-gray-800">Your first 50 listings every month are on us.</p>
        <div className="mt-6 flex flex-wrap justify-center gap-4">
          <Link href="/sell" className="inline-flex items-center gap-2 rounded-full bg-gray-900 px-8 py-3.5 text-sm font-bold text-white hover:bg-gray-800">
            Create a Free Listing <ArrowRight size={16} />
          </Link>
          <Link href="/seller-protection" className="inline-flex items-center gap-2 rounded-full border-2 border-gray-900 px-8 py-3.5 text-sm font-bold text-gray-900 hover:bg-gray-900 hover:text-white">
            Seller Protection
          </Link>
        </div>
      </section>
    </div>
  );
}
