import Link from 'next/link';
import { Building2, ArrowRight, CheckCircle, Star, BarChart3 } from 'lucide-react';

const COMPARISON = [
  { feature: 'Monthly listings', personal: '50 free', business: 'Unlimited' },
  { feature: 'Success fee', personal: '5%', business: '3%' },
  { feature: 'Multiple user access', personal: 'No', business: 'Yes (5 users)' },
  { feature: 'Analytics dashboard', personal: 'Basic', business: 'Advanced' },
  { feature: 'API access', personal: 'No', business: 'Yes' },
  { feature: 'Dedicated support', personal: 'No', business: 'Yes' },
  { feature: 'Storefront', personal: 'Basic', business: 'Custom domain' },
  { feature: 'Bulk listing tools', personal: 'No', business: 'Yes' },
];

const PLANS = [
  { name: 'Starter', price: 'Free', listings: '50', fee: '5%', features: ['Basic analytics', 'Standard support'] },
  { name: 'Growth', price: '$29/mo', listings: '500', fee: '4%', features: ['Advanced analytics', 'Priority support', 'Bulk listing'] },
  { name: 'Pro', price: '$99/mo', listings: 'Unlimited', fee: '3%', features: ['API access', 'Dedicated manager', 'Custom storefront'] },
];

export default function BusinessSellersPage() {
  return (
    <div className="min-h-screen">
      <section className="bg-gradient-to-br from-[#FFC220]/30 to-[#FFC220]/5 py-14">
        <div className="mx-auto max-w-5xl px-4 text-center">
          <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-[#FFC220]/30 px-4 py-1.5 text-sm font-medium text-gray-900">
            <Building2 size={16} /> Business Sellers
          </div>
          <h1 className="text-3xl font-extrabold md:text-4xl">Grow Your Business with Mnbarh</h1>
          <p className="mx-auto mt-3 max-w-xl text-sm text-gray-600">From small shops to enterprise — we have a plan for every size.</p>
        </div>
      </section>

      {/* Comparison Table */}
      <section className="py-14">
        <div className="mx-auto max-w-3xl px-4">
          <h2 className="mb-6 text-center text-2xl font-extrabold text-gray-900">Personal vs Business Account</h2>
          <div className="overflow-hidden rounded-xl border border-gray-200">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-gray-50 text-xs uppercase text-gray-500">
                  <th className="px-5 py-3 text-left">Feature</th>
                  <th className="px-5 py-3 text-center">Personal</th>
                  <th className="px-5 py-3 text-center">Business</th>
                </tr>
              </thead>
              <tbody>
                {COMPARISON.map((row) => (
                  <tr key={row.feature} className="border-t border-gray-100">
                    <td className="px-5 py-3 text-gray-700">{row.feature}</td>
                    <td className="px-5 py-3 text-center text-gray-500">{row.personal}</td>
                    <td className="px-5 py-3 text-center font-semibold text-emerald-700">{row.business}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </section>

      {/* Plans */}
      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-4xl px-4">
          <h2 className="mb-8 text-center text-2xl font-extrabold text-gray-900">Pricing Plans</h2>
          <div className="grid gap-6 sm:grid-cols-3">
            {PLANS.map((plan) => (
              <div key={plan.name} className="rounded-2xl border border-gray-200 bg-white p-6">
                <h3 className="text-lg font-bold text-gray-900">{plan.name}</h3>
                <p className="mt-1 text-2xl font-extrabold text-[#0071CE]">{plan.price}</p>
                <p className="text-xs text-gray-500">{plan.listings} listings · {plan.fee} fee</p>
                <ul className="mt-4 space-y-1.5">
                  {plan.features.map((f) => (
                    <li key={f} className="flex items-center gap-2 text-xs text-gray-600">
                      <CheckCircle size={14} className="text-emerald-500" /> {f}
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Verification */}
      <section className="py-14">
        <div className="mx-auto max-w-3xl px-4">
          <h2 className="mb-4 text-2xl font-extrabold text-gray-900">Business Verification</h2>
          <p className="text-sm text-gray-600">Required: Trade license or commercial registration. Optional: VAT registration (affects invoice format).</p>
        </div>
      </section>

      <section className="bg-[#FFC220] py-10 text-center">
        <Link href="/register" className="inline-flex items-center gap-2 rounded-full bg-gray-900 px-8 py-3.5 text-sm font-bold text-white hover:bg-gray-800">
          Register as Business <ArrowRight size={16} />
        </Link>
      </section>
    </div>
  );
}
