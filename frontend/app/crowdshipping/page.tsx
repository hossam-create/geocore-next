import Link from 'next/link';
import { Package, Plane, ShieldCheck, DollarSign, ArrowRight, CheckCircle, XCircle, Globe, Truck } from 'lucide-react';

const TWO_TYPES = [
  {
    icon: Package,
    title: 'CrowdShopping (Buy + Deliver)',
    desc: 'The traveler buys the item on your behalf at the source and brings it to you. Perfect for: limited edition products, cheaper prices abroad, items not available in your country.',
    color: 'text-[#0071CE]',
    bg: 'bg-[#0071CE]/10',
  },
  {
    icon: Truck,
    title: 'CrowdShipping (Deliver Only)',
    desc: 'You already have the item and need it transported to another city or country. The traveler carries it in their luggage.',
    color: 'text-emerald-600',
    bg: 'bg-emerald-100',
  },
];

const ALLOWED = ['Electronics', 'Clothes & fashion', 'Cosmetics & beauty', 'Books & media', 'Sealed food items', 'Documents', 'Jewelry', 'Small gifts'];
const PROHIBITED = ['Weapons & ammunition', 'Drugs & controlled substances', 'Currency & cash', 'Live animals', 'Items over 5 kg', 'Customs-prohibited items'];

const PRICING = [
  { distance: 'Local', small: '$5–10', medium: '$15–25', large: '$30–50' },
  { distance: 'Domestic', small: '$10–20', medium: '$20–35', large: '$40–70' },
  { distance: 'Regional', small: '$15–30', medium: '$30–50', large: '$60–100' },
  { distance: 'International', small: '$20–40', medium: '$40–70', large: '$80–150' },
];

export default function CrowdshippingPage() {
  return (
    <div className="min-h-screen">
      {/* Hero */}
      <section className="bg-gradient-to-br from-[#0071CE] to-[#003f75] text-white">
        <div className="mx-auto max-w-5xl px-4 py-14 text-center">
          <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-white/15 px-4 py-1.5 text-sm font-medium">
            <Plane size={16} /> Crowdshipping
          </div>
          <h1 className="text-3xl font-extrabold md:text-4xl">Crowdshipping — The Future of Peer-to-Peer Delivery</h1>
          <p className="mx-auto mt-3 max-w-2xl text-blue-100">
            Connect people who need items transported with travelers who have spare luggage space. Like asking a trusted friend — except we handle matching, payment, and protection.
          </p>
        </div>
      </section>

      {/* What is Crowdshipping */}
      <section className="py-14">
        <div className="mx-auto max-w-4xl px-4">
          <h2 className="mb-4 text-2xl font-extrabold text-gray-900">What Is Crowdshipping?</h2>
          <p className="text-sm text-gray-600 leading-relaxed">
            Unlike courier companies, travelers are real people going to these destinations anyway. There&apos;s no extra carbon footprint, no warehouse, no corporate fees — just a community helping each other. Mnbarh handles the matching, secure payment, and full buyer protection.
          </p>
        </div>
      </section>

      {/* Two Types */}
      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-5xl px-4">
          <h2 className="mb-8 text-center text-2xl font-extrabold text-gray-900">Two Types of Requests</h2>
          <div className="grid gap-6 sm:grid-cols-2">
            {TWO_TYPES.map((t) => (
              <div key={t.title} className="rounded-2xl border border-gray-200 bg-white p-6">
                <div className={`mb-3 inline-flex h-11 w-11 items-center justify-center rounded-xl ${t.bg}`}>
                  <t.icon size={22} className={t.color} />
                </div>
                <h3 className="text-base font-bold text-gray-900">{t.title}</h3>
                <p className="mt-2 text-sm text-gray-600 leading-relaxed">{t.desc}</p>
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
                <CheckCircle size={16} /> Allowed
              </h3>
              <ul className="space-y-1 text-sm text-emerald-700">
                {ALLOWED.map((item) => <li key={item}>✅ {item}</li>)}
              </ul>
            </div>
            <div className="rounded-2xl border border-red-200 bg-red-50 p-6">
              <h3 className="mb-3 flex items-center gap-2 text-sm font-bold text-red-800">
                <XCircle size={16} /> Prohibited
              </h3>
              <ul className="space-y-1 text-sm text-red-700">
                {PROHIBITED.map((item) => <li key={item}>❌ {item}</li>)}
              </ul>
            </div>
          </div>
        </div>
      </section>

      {/* Pricing Guide */}
      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-3xl px-4">
          <h2 className="mb-2 text-center text-2xl font-extrabold text-gray-900">Tip Pricing Guide</h2>
          <p className="mb-8 text-center text-xs text-gray-500">Guidelines — you negotiate directly with the traveler</p>
          <div className="overflow-hidden rounded-2xl border border-gray-200">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-gray-100 text-xs uppercase text-gray-500">
                  <th className="px-5 py-3 text-left">Distance</th>
                  <th className="px-5 py-3 text-center">Small (&lt;0.5 kg)</th>
                  <th className="px-5 py-3 text-center">Medium (0.5–2 kg)</th>
                  <th className="px-5 py-3 text-center">Large (2–5 kg)</th>
                </tr>
              </thead>
              <tbody>
                {PRICING.map((row) => (
                  <tr key={row.distance} className="border-t border-gray-100">
                    <td className="px-5 py-3 font-medium text-gray-900">{row.distance}</td>
                    <td className="px-5 py-3 text-center text-gray-600">{row.small}</td>
                    <td className="px-5 py-3 text-center text-gray-600">{row.medium}</td>
                    <td className="px-5 py-3 text-center text-gray-600">{row.large}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="bg-gradient-to-r from-[#0071CE] to-[#003f75] py-14 text-center text-white">
        <h2 className="text-2xl font-extrabold">Ready to Try Crowdshipping?</h2>
        <div className="mt-6 flex flex-wrap justify-center gap-4">
          <Link href="/buy-via-traveler" className="inline-flex items-center gap-2 rounded-full bg-[#FFC220] px-8 py-3.5 text-sm font-bold text-gray-900 hover:bg-yellow-400">
            Post a Request <ArrowRight size={16} />
          </Link>
          <Link href="/crowdshipping/for-travelers" className="inline-flex items-center gap-2 rounded-full border-2 border-white/30 px-8 py-3.5 text-sm font-bold text-white hover:bg-white/10">
            Earn as a Traveler
          </Link>
        </div>
        <div className="mt-6 flex justify-center gap-6 text-xs text-blue-200">
          <Link href="/crowdshipping/trust" className="hover:text-white">Safety & Trust</Link>
          <Link href="/crowdshipping/insurance" className="hover:text-white">Insurance</Link>
          <Link href="/crowdshipping/payment" className="hover:text-white">Secure Payment</Link>
        </div>
      </section>
    </div>
  );
}
