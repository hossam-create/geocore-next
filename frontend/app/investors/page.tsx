import Link from 'next/link';
import { TrendingUp, Globe, Users, BarChart3, ArrowRight, Mail } from 'lucide-react';

const METRICS = [
  { icon: BarChart3, label: 'Countries Active', value: '7' },
  { icon: TrendingUp, label: 'MENA E-Commerce Market (2024)', value: '$29B' },
  { icon: Globe, label: 'Projected Market (2028)', value: '$60B+' },
  { icon: Users, label: 'GCC Internet Users', value: '45M+' },
];

export default function InvestorsPage() {
  return (
    <div className="min-h-screen">
      <section className="bg-gradient-to-br from-[#0071CE] to-[#003f75] text-white py-14">
        <div className="mx-auto max-w-5xl px-4 text-center">
          <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-white/15 px-4 py-1.5 text-sm font-medium">
            <TrendingUp size={16} /> Investors
          </div>
          <h1 className="text-3xl font-extrabold md:text-4xl">Invest in the Future of MENA Commerce</h1>
          <p className="mx-auto mt-3 max-w-2xl text-blue-100">
            The MENA e-commerce market reached $29B in 2024 and is projected to double by 2028. With Souq.com acquired by Amazon and the market underserved, Mnbarh is positioned to become the leading independent marketplace for the Arab world.
          </p>
        </div>
      </section>

      {/* Key Metrics */}
      <section className="py-14">
        <div className="mx-auto max-w-4xl px-4">
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {METRICS.map((m) => (
              <div key={m.label} className="rounded-2xl border border-gray-200 bg-white p-5 text-center">
                <m.icon size={20} className="mx-auto mb-2 text-[#0071CE]" />
                <p className="text-2xl font-extrabold text-gray-900">{m.value}</p>
                <p className="text-xs text-gray-500">{m.label}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Opportunity */}
      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-3xl px-4">
          <h2 className="mb-4 text-2xl font-extrabold text-gray-900">Why Mnbarh</h2>
          <ul className="space-y-3 text-sm text-gray-600">
            <li className="flex items-start gap-2"><span className="text-emerald-500 mt-0.5">✓</span> Only independent marketplace built specifically for the Arab world</li>
            <li className="flex items-start gap-2"><span className="text-emerald-500 mt-0.5">✓</span> Unique crowdshipping model — no competitor offers Buy via Traveler</li>
            <li className="flex items-start gap-2"><span className="text-emerald-500 mt-0.5">✓</span> Live commerce with real-time auctions and streaming</li>
            <li className="flex items-start gap-2"><span className="text-emerald-500 mt-0.5">✓</span> Full Arabic RTL support — not an afterthought but a core design principle</li>
            <li className="flex items-start gap-2"><span className="text-emerald-500 mt-0.5">✓</span> Escrow-first payment system — trust from day one</li>
          </ul>
        </div>
      </section>

      {/* Contact */}
      <section className="py-14">
        <div className="mx-auto max-w-3xl px-4 text-center">
          <div className="rounded-2xl border border-[#0071CE]/20 bg-[#0071CE]/5 p-8">
            <Mail size={24} className="mx-auto mb-2 text-[#0071CE]" />
            <h3 className="text-lg font-bold text-gray-900">Investor Relations</h3>
            <p className="mt-2 text-sm text-gray-600">investors@mnbarh.com</p>
          </div>
        </div>
      </section>
    </div>
  );
}
