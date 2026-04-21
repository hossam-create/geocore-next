import Link from 'next/link';
import { Package, ShoppingBag, BarChart3, Store, Wallet, Star, Shield, ArrowRight, BookOpen, Calculator, Truck, BadgeCheck, HelpCircle, TrendingUp, CheckCircle } from 'lucide-react';

const DASHBOARD_LINKS = [
  { icon: Package, title: 'My Listings', href: '/dashboard', desc: 'Manage your active and draft listings' },
  { icon: ShoppingBag, title: 'My Sales', href: '/dashboard', desc: 'Track orders and shipments' },
  { icon: BarChart3, title: 'Analytics', href: '/dashboard', desc: 'Views, clicks, and conversion data' },
  { icon: Store, title: 'Storefront', href: '/dashboard', desc: 'Customize your shop appearance' },
  { icon: Wallet, title: 'Payouts', href: '/wallet', desc: 'Withdraw earnings to bank or wallet' },
];

const RESOURCES = [
  { icon: BookOpen, title: 'Getting Started Guide', href: '/how-to-sell' },
  { icon: Calculator, title: 'Fee Calculator', href: '/fees' },
  { icon: Truck, title: 'Shipping Guide', href: '/help' },
  { icon: Shield, title: 'Seller Protection', href: '/seller-protection' },
  { icon: HelpCircle, title: 'Dispute Help', href: '/help/faq' },
  { icon: TrendingUp, title: 'Upgrade to Business', href: '/business-sellers' },
];

const TOP_RATED = {
  requirements: ['98%+ positive feedback', '100+ completed sales', 'Ships within 3 days'],
  benefits: ['10% search boost', 'Top Rated badge', 'Priority support', 'Lower fees (3%)'],
};

export default function SellerCenterPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10">
      <div className="mb-8 text-center">
        <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-[#FFC220]/20 px-4 py-1.5 text-sm font-medium text-gray-900">
          <Store size={16} /> Seller Center
        </div>
        <h1 className="text-3xl font-extrabold text-gray-900">Everything You Need to Succeed as a Seller</h1>
      </div>

      {/* Dashboard Links */}
      <section className="mb-10">
        <h2 className="mb-4 text-lg font-bold text-gray-900">Your Dashboard</h2>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {DASHBOARD_LINKS.map((l) => (
            <Link key={l.title} href={l.href} className="flex items-center gap-3 rounded-xl border border-gray-200 bg-white p-4 hover:border-[#0071CE] transition-colors">
              <l.icon size={20} className="shrink-0 text-[#0071CE]" />
              <div>
                <h3 className="text-sm font-semibold text-gray-900">{l.title}</h3>
                <p className="text-xs text-gray-500">{l.desc}</p>
              </div>
            </Link>
          ))}
        </div>
      </section>

      {/* Resources */}
      <section className="mb-10">
        <h2 className="mb-4 text-lg font-bold text-gray-900">Resources</h2>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {RESOURCES.map((r) => (
            <Link key={r.title} href={r.href} className="flex items-center gap-3 rounded-xl border border-gray-200 bg-white p-4 hover:border-[#FFC220] transition-colors">
              <r.icon size={18} className="shrink-0 text-[#FFC220]" />
              <span className="text-sm font-semibold text-gray-900">{r.title}</span>
            </Link>
          ))}
        </div>
      </section>

      {/* Top Rated */}
      <section className="mb-10 rounded-2xl border border-[#FFC220] bg-[#FFC220]/10 p-6">
        <div className="flex items-center gap-2 mb-4">
          <Star size={20} className="text-[#FFC220]" />
          <h2 className="text-lg font-bold text-gray-900">Top Rated Seller Program</h2>
        </div>
        <div className="grid gap-6 sm:grid-cols-2">
          <div>
            <h3 className="mb-2 text-sm font-semibold text-gray-700">Requirements</h3>
            <ul className="space-y-1">
              {TOP_RATED.requirements.map((r) => (
                <li key={r} className="flex items-center gap-2 text-xs text-gray-600"><BadgeCheck size={14} className="text-amber-500" /> {r}</li>
              ))}
            </ul>
          </div>
          <div>
            <h3 className="mb-2 text-sm font-semibold text-gray-700">Benefits</h3>
            <ul className="space-y-1">
              {TOP_RATED.benefits.map((b) => (
                <li key={b} className="flex items-center gap-2 text-xs text-gray-600"><CheckCircle size={14} className="text-emerald-500" /> {b}</li>
              ))}
            </ul>
          </div>
        </div>
      </section>

      {/* Performance Metrics */}
      <section className="mb-10">
        <h2 className="mb-4 text-lg font-bold text-gray-900">Performance Metrics</h2>
        <div className="grid gap-3 sm:grid-cols-3">
          <div className="rounded-xl border border-gray-200 bg-white p-4 text-center">
            <p className="text-xs text-gray-500">Transaction Defect Rate</p>
            <p className="text-lg font-bold text-gray-900">Keep below 2%</p>
          </div>
          <div className="rounded-xl border border-gray-200 bg-white p-4 text-center">
            <p className="text-xs text-gray-500">Late Shipment Rate</p>
            <p className="text-lg font-bold text-gray-900">Keep below 5%</p>
          </div>
          <div className="rounded-xl border border-gray-200 bg-white p-4 text-center">
            <p className="text-xs text-gray-500">Cases Closed Without Resolution</p>
            <p className="text-lg font-bold text-gray-900">Keep at 0%</p>
          </div>
        </div>
      </section>

      <section className="text-center">
        <Link href="/sell" className="inline-flex items-center gap-2 rounded-xl bg-[#FFC220] px-6 py-3 text-sm font-semibold text-gray-900 hover:bg-yellow-400">
          Start Selling <ArrowRight size={16} />
        </Link>
      </section>
    </div>
  );
}
