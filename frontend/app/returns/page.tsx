import Link from 'next/link';
import { RefreshCw, CheckCircle, ArrowRight, Truck, Clock, Package } from 'lucide-react';

const RETURN_STEPS = [
  { icon: Package, title: 'Go to My Orders', desc: 'Find the order with the item you want to return.' },
  { icon: RefreshCw, title: 'Select the Item', desc: 'Click "Return Item" on the order detail page.' },
  { icon: CheckCircle, title: 'Choose Reason', desc: 'Select from: not as described, damaged, wrong item, or changed mind.' },
  { icon: Truck, title: 'Print Return Label', desc: 'Download and print the prepaid shipping label.' },
  { icon: Clock, title: 'Drop Off at Post Office', desc: 'Ship the item back within 5 business days.' },
];

const SHIPPING_TABLE = [
  { reason: 'Item not as described', payer: 'Seller' },
  { reason: 'Damaged in transit', payer: 'Platform' },
  { reason: 'Changed my mind', payer: 'Buyer' },
  { reason: 'Wrong item received', payer: 'Seller' },
];

export default function ReturnsPage() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <div className="mb-8 text-center">
        <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-emerald-100 px-4 py-1.5 text-sm font-medium text-emerald-700">
          <RefreshCw size={16} /> Returns Policy
        </div>
        <h1 className="text-3xl font-extrabold text-gray-900">Returns & Exchanges</h1>
        <p className="mt-2 text-sm text-gray-500">Standard return window: 30 days from delivery</p>
      </div>

      {/* How to Return */}
      <section className="mb-10">
        <h2 className="mb-6 text-xl font-bold text-gray-900">How to Return an Item</h2>
        <div className="space-y-3">
          {RETURN_STEPS.map((step, i) => (
            <div key={step.title} className="flex items-center gap-4 rounded-xl border border-gray-200 bg-white p-4">
              <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-emerald-100 text-sm font-bold text-emerald-700">{i + 1}</span>
              <div>
                <h3 className="text-sm font-semibold text-gray-900">{step.title}</h3>
                <p className="text-xs text-gray-600">{step.desc}</p>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* Who Pays Shipping */}
      <section className="mb-10">
        <h2 className="mb-4 text-xl font-bold text-gray-900">Who Pays Return Shipping?</h2>
        <div className="overflow-hidden rounded-xl border border-gray-200">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-gray-50 text-xs uppercase text-gray-500">
                <th className="px-5 py-3 text-left">Return Reason</th>
                <th className="px-5 py-3 text-right">Shipping Paid By</th>
              </tr>
            </thead>
            <tbody>
              {SHIPPING_TABLE.map((row) => (
                <tr key={row.reason} className="border-t border-gray-100">
                  <td className="px-5 py-3 text-gray-700">{row.reason}</td>
                  <td className="px-5 py-3 text-right font-semibold text-gray-900">{row.payer}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      {/* Refund Timeline */}
      <section className="mb-10 rounded-2xl border border-gray-200 bg-gray-50 p-6">
        <h2 className="mb-3 text-lg font-bold text-gray-900">Refund Timeline</h2>
        <p className="text-sm text-gray-600">3–5 business days after the item is received by the seller.</p>
      </section>

      {/* CTA */}
      <section className="text-center">
        <Link href="/buyer/orders" className="inline-block rounded-xl bg-[#0071CE] px-6 py-3 text-sm font-semibold text-white hover:bg-[#005ba3]">
          Go to My Orders
        </Link>
        <div className="mt-3 flex justify-center gap-4 text-xs">
          <Link href="/refund-policy" className="text-[#0071CE] hover:underline">Refund Policy</Link>
          <span className="text-gray-300">|</span>
          <Link href="/buyer-protection" className="text-[#0071CE] hover:underline">Buyer Protection</Link>
          <span className="text-gray-300">|</span>
          <Link href="/help" className="text-[#0071CE] hover:underline">Help Center</Link>
        </div>
      </section>
    </div>
  );
}
