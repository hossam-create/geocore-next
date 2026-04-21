import Link from 'next/link';
import { Building2, Package, Users, Warehouse, Code, Layers, ArrowRight, Mail } from 'lucide-react';

const SECTIONS = [
  { icon: Package, title: 'Wholesale Marketplace', desc: 'Buy in bulk directly from verified suppliers across the GCC. Volume discounts and escrow protection.' },
  { icon: Users, title: 'Corporate Accounts', desc: 'Team purchasing with invoice management, approval workflows, and spending limits.' },
  { icon: Warehouse, title: 'Fulfillment Center', desc: 'Store your inventory in our warehouse. We handle picking, packing, and shipping for you.' },
  { icon: Code, title: 'API Integration', desc: 'Connect your ERP or e-commerce platform to Mnbarh. REST API with webhooks and real-time sync.' },
  { icon: Layers, title: 'White Label', desc: 'Build your own marketplace on our infrastructure. Custom branding, your domain, our technology.' },
];

export default function BusinessPage() {
  return (
    <div className="min-h-screen">
      <section className="bg-gradient-to-br from-[#0071CE] to-[#003f75] text-white py-14">
        <div className="mx-auto max-w-5xl px-4 text-center">
          <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-white/15 px-4 py-1.5 text-sm font-medium">
            <Building2 size={16} /> For Business
          </div>
          <h1 className="text-3xl font-extrabold md:text-4xl">Mnbarh for Business</h1>
          <p className="mx-auto mt-3 max-w-xl text-blue-100">The B2B marketplace — wholesale, fulfillment, API, and more.</p>
        </div>
      </section>

      <section className="py-14">
        <div className="mx-auto max-w-4xl px-4">
          <div className="space-y-4">
            {SECTIONS.map((s) => (
              <div key={s.title} className="flex items-start gap-4 rounded-2xl border border-gray-200 bg-white p-6">
                <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-[#0071CE]/10">
                  <s.icon size={24} className="text-[#0071CE]" />
                </div>
                <div>
                  <h3 className="text-base font-bold text-gray-900">{s.title}</h3>
                  <p className="mt-1 text-sm text-gray-600 leading-relaxed">{s.desc}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-3xl px-4 text-center">
          <h2 className="text-2xl font-extrabold text-gray-900">Contact Sales</h2>
          <p className="mt-2 text-sm text-gray-600">sales@mnbarh.com</p>
          <Link href="/contact" className="mt-4 inline-flex items-center gap-2 rounded-full bg-[#0071CE] px-6 py-3 text-sm font-semibold text-white hover:bg-[#005ba3]">
            Get in Touch <ArrowRight size={16} />
          </Link>
        </div>
      </section>
    </div>
  );
}
