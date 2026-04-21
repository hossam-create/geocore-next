import Link from 'next/link';
import { Megaphone, Image, Search, Mail, Star, ArrowRight, BarChart3 } from 'lucide-react';

const FORMATS = [
  { icon: Image, title: 'Homepage Banner', desc: '1200×400px, full-width, 24h rotation. Maximum visibility.', price: 'CPM-based' },
  { icon: Star, title: 'Category Sponsorship', desc: 'Be the featured brand in your category. Exclusive placement.', price: 'Monthly' },
  { icon: Search, title: 'Search Sponsored Listings', desc: 'Appear at top of relevant searches. Pay only when clicked.', price: 'CPC' },
  { icon: Mail, title: 'Email Newsletter', desc: 'Featured in weekly deals email. 50k+ subscribers across MENA.', price: 'Per send' },
];

export default function AdvertisePage() {
  return (
    <div className="min-h-screen">
      <section className="bg-gradient-to-br from-[#0071CE] to-[#003f75] text-white py-14">
        <div className="mx-auto max-w-5xl px-4 text-center">
          <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-white/15 px-4 py-1.5 text-sm font-medium">
            <Megaphone size={16} /> Advertise
          </div>
          <h1 className="text-3xl font-extrabold md:text-4xl">Advertise with Us</h1>
          <p className="mx-auto mt-3 max-w-xl text-blue-100">
            Reach millions of buyers across the GCC and MENA region through Mnbarh&apos;s advertising platform.
          </p>
        </div>
      </section>

      {/* Ad Formats */}
      <section className="py-14">
        <div className="mx-auto max-w-4xl px-4">
          <h2 className="mb-8 text-center text-2xl font-extrabold text-gray-900">Ad Formats Available</h2>
          <div className="grid gap-4 sm:grid-cols-2">
            {FORMATS.map((f) => (
              <div key={f.title} className="rounded-2xl border border-gray-200 bg-white p-6">
                <f.icon size={22} className="mb-3 text-[#0071CE]" />
                <h3 className="text-sm font-bold text-gray-900">{f.title}</h3>
                <p className="mt-1 text-xs text-gray-600">{f.desc}</p>
                <span className="mt-2 inline-block rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600">{f.price}</span>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Pricing */}
      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-3xl px-4 text-center">
          <h2 className="mb-4 text-2xl font-extrabold text-gray-900">Flexible Pricing</h2>
          <p className="text-sm text-gray-600">Choose CPM (cost per 1,000 impressions) or CPC (cost per click) — whatever works for your goals.</p>
        </div>
      </section>

      {/* Contact */}
      <section className="py-14">
        <div className="mx-auto max-w-3xl px-4">
          <div className="rounded-2xl border border-[#0071CE]/20 bg-[#0071CE]/5 p-8 text-center">
            <BarChart3 size={24} className="mx-auto mb-3 text-[#0071CE]" />
            <h3 className="text-lg font-bold text-gray-900">Get Started</h3>
            <p className="mt-2 text-sm text-gray-600">Contact our advertising team for a custom proposal.</p>
            <p className="mt-1 text-sm font-semibold text-[#0071CE]">ads@mnbarh.com</p>
            <Link href="/contact" className="mt-4 inline-flex items-center gap-2 rounded-full bg-[#0071CE] px-6 py-3 text-sm font-semibold text-white hover:bg-[#005ba3]">
              Contact Us <ArrowRight size={16} />
            </Link>
          </div>
        </div>
      </section>
    </div>
  );
}
