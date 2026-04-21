import Link from 'next/link';
import { Truck, Globe, Clock, Package, MapPin, AlertCircle, CheckCircle } from 'lucide-react';

const DELIVERY_TIMES = [
  { region: 'UAE (Domestic)', standard: '2-4 business days', express: '1-2 business days' },
  { region: 'Saudi Arabia', standard: '3-5 business days', express: '1-3 business days' },
  { region: 'Kuwait', standard: '3-5 business days', express: '2-3 business days' },
  { region: 'Qatar', standard: '3-5 business days', express: '2-3 business days' },
  { region: 'Bahrain', standard: '3-5 business days', express: '2-3 business days' },
  { region: 'Oman', standard: '3-5 business days', express: '2-3 business days' },
];

const SHIPPING_INFO = [
  {
    icon: Truck,
    title: 'Domestic Shipping',
    desc: 'Orders within the UAE are shipped via local couriers. Standard shipping is free for orders over AED 200. Express shipping is available for an additional fee.',
  },
  {
    icon: Globe,
    title: 'International Shipping',
    desc: 'We ship to all GCC countries. International orders are handled by trusted logistics partners. Customs clearance is the responsibility of the buyer where applicable.',
  },
  {
    icon: Clock,
    title: 'Processing Time',
    desc: 'Sellers have 1-2 business days to process and ship orders. You will receive a tracking number once the item is dispatched.',
  },
  {
    icon: Package,
    title: 'Tracking',
    desc: 'All shipments include tracking. Track your order from the Orders page or via the carrier\'s website using the provided tracking number.',
  },
];

export default function ShippingPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10">
      <div className="mb-8 text-center">
        <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-[#0071CE]/10 px-4 py-1.5 text-sm font-medium text-[#0071CE]">
          <Truck size={16} />
          Shipping & Delivery
        </div>
        <h1 className="text-3xl font-extrabold text-gray-900">Shipping Information</h1>
        <p className="mt-2 text-sm text-gray-500 max-w-xl mx-auto">
          Learn about our shipping options, delivery times, and tracking for orders across the GCC region.
        </p>
      </div>

      {/* Shipping Options */}
      <section className="mb-10">
        <div className="grid gap-4 sm:grid-cols-2">
          {SHIPPING_INFO.map((info) => (
            <div key={info.title} className="rounded-xl border border-gray-200 bg-white p-5">
              <div className="flex items-center gap-2 mb-2">
                <info.icon size={18} className="text-[#0071CE]" />
                <h3 className="text-sm font-semibold text-gray-900">{info.title}</h3>
              </div>
              <p className="text-xs text-gray-600 leading-relaxed">{info.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Delivery Times Table */}
      <section className="mb-10">
        <h2 className="mb-4 text-lg font-bold text-gray-900">Estimated Delivery Times</h2>
        <div className="overflow-x-auto rounded-xl border border-gray-200 bg-white">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 bg-gray-50">
                <th className="px-4 py-3 text-left font-semibold text-gray-700">
                  <div className="flex items-center gap-2">
                    <MapPin size={14} />
                    Region
                  </div>
                </th>
                <th className="px-4 py-3 text-left font-semibold text-gray-700">Standard Shipping</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-700">Express Shipping</th>
              </tr>
            </thead>
            <tbody>
              {DELIVERY_TIMES.map((row, i) => (
                <tr key={row.region} className={i < DELIVERY_TIMES.length - 1 ? 'border-b border-gray-100' : ''}>
                  <td className="px-4 py-3 font-medium text-gray-900">{row.region}</td>
                  <td className="px-4 py-3 text-gray-600">{row.standard}</td>
                  <td className="px-4 py-3 text-gray-600">{row.express}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="mt-2 text-xs text-gray-400">Delivery times are estimates and may vary during peak seasons or due to customs processing.</p>
      </section>

      {/* Important Notes */}
      <section className="mb-10 grid gap-4 sm:grid-cols-2">
        <div className="rounded-xl border border-amber-200 bg-amber-50 p-4">
          <h3 className="mb-2 flex items-center gap-2 text-sm font-semibold text-amber-800">
            <AlertCircle size={16} />
            Important Notes
          </h3>
          <ul className="space-y-1 text-xs text-amber-700">
            <li>Delivery times start from when the seller ships the item, not from order placement.</li>
            <li>Customs duties and taxes are the buyer's responsibility for international orders.</li>
            <li>Some items may have restricted shipping due to size, weight, or category.</li>
            <li>Delivery delays may occur during public holidays and sales events.</li>
          </ul>
        </div>
        <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-4">
          <h3 className="mb-2 flex items-center gap-2 text-sm font-semibold text-emerald-800">
            <CheckCircle size={16} />
            What to Expect
          </h3>
          <ul className="space-y-1 text-xs text-emerald-700">
            <li>Email notification when your order is placed.</li>
            <li>Shipping confirmation with tracking number.</li>
            <li>Real-time tracking updates via the Orders page.</li>
            <li>Delivery confirmation once the item arrives.</li>
          </ul>
        </div>
      </section>

      {/* CTAs */}
      <section className="text-center">
        <div className="flex flex-wrap justify-center gap-3">
          <Link href="/orders" className="inline-block rounded-xl bg-[#0071CE] px-5 py-2.5 text-sm font-semibold text-white hover:bg-[#005ba3]">
            Track Your Orders
          </Link>
          <Link href="/help" className="inline-block rounded-xl border border-gray-200 bg-white px-5 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-100">
            Help Center
          </Link>
        </div>
      </section>
    </div>
  );
}
