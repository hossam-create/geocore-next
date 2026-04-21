import Link from 'next/link';
import { Tag, Percent, Clock, Flame, ArrowRight, Sparkles } from 'lucide-react';

const EVENTS = [
  { icon: Flame, title: 'Ramadan Deals', desc: 'Biggest sale of the year — up to 70% off electronics, fashion, and home.', date: 'Annual · Ramadan Season', color: 'text-red-600', bg: 'bg-red-100' },
  { icon: Sparkles, title: 'Eid Shopping Festival', desc: 'Gifts for everyone — curated collections with free shipping on orders over $50.', date: 'Annual · Eid Al-Fitr & Eid Al-Adha', color: 'text-amber-600', bg: 'bg-amber-100' },
  { icon: Tag, title: 'White Wednesday', desc: 'MENA\'s answer to Black Friday — deals on every category.', date: 'Annual · November', color: 'text-[#0071CE]', bg: 'bg-[#0071CE]/10' },
  { icon: Percent, title: 'Summer Clearance', desc: 'End-of-season markdowns on fashion, sports, and outdoor.', date: 'June – August', color: 'text-emerald-600', bg: 'bg-emerald-100' },
  { icon: Clock, title: 'Flash Sales', desc: '24-hour deals that appear without warning. Follow us to never miss one.', date: 'Surprise · Anytime', color: 'text-violet-600', bg: 'bg-violet-100' },
];

export default function SeasonalSalesPage() {
  return (
    <div className="min-h-screen">
      <section className="bg-gradient-to-br from-[#FFC220]/30 to-[#0071CE]/10 py-14">
        <div className="mx-auto max-w-5xl px-4 text-center">
          <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-[#FFC220]/30 px-4 py-1.5 text-sm font-medium text-gray-900">
            <Tag size={16} /> Seasonal Sales
          </div>
          <h1 className="text-3xl font-extrabold md:text-4xl">Seasonal Sales & Events</h1>
          <p className="mx-auto mt-3 max-w-xl text-sm text-gray-600">
            The best deals happen at the right time. Here&apos;s what&apos;s coming up on Mnbarh.
          </p>
        </div>
      </section>

      <section className="py-14">
        <div className="mx-auto max-w-4xl px-4">
          <div className="space-y-4">
            {EVENTS.map((e) => (
              <div key={e.title} className="flex items-start gap-4 rounded-2xl border border-gray-200 bg-white p-6">
                <div className={`flex h-12 w-12 shrink-0 items-center justify-center rounded-xl ${e.bg}`}>
                  <e.icon size={24} className={e.color} />
                </div>
                <div>
                  <h3 className="text-base font-bold text-gray-900">{e.title}</h3>
                  <p className="text-xs text-gray-500">{e.date}</p>
                  <p className="mt-2 text-sm text-gray-600">{e.desc}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="bg-[#FFC220] py-10 text-center">
        <Link href="/listings" className="inline-flex items-center gap-2 rounded-full bg-gray-900 px-8 py-3.5 text-sm font-bold text-white hover:bg-gray-800">
          Browse Current Deals <ArrowRight size={16} />
        </Link>
      </section>
    </div>
  );
}
