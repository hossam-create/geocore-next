import Link from 'next/link';
import { Users, Globe, Heart, Accessibility, ArrowRight } from 'lucide-react';

const PILLARS = [
  { icon: Users, title: 'Our Commitment', desc: 'Mnbarh is built for everyone — across the Arab world and beyond. We believe diverse teams build better products for diverse communities.' },
  { icon: Heart, title: 'Equal Opportunity', desc: 'We hire, promote, and compensate based on merit and potential. No discrimination based on gender, ethnicity, religion, age, disability, or sexual orientation.' },
  { icon: Accessibility, title: 'Accessibility', desc: 'Our platform is designed to be usable by everyone, including people with visual, motor, or cognitive disabilities. WCAG 2.1 Level AA compliance.' },
  { icon: Globe, title: 'Languages Supported', desc: 'Full Arabic and English support with RTL layout. We\'re expanding to French, Urdu, and Malay to serve our diverse user base.' },
];

export default function DiversityPage() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <div className="mb-8 text-center">
        <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-violet-100 px-4 py-1.5 text-sm font-medium text-violet-700">
          <Users size={16} /> Diversity & Inclusion
        </div>
        <h1 className="text-3xl font-extrabold text-gray-900">Mnbarh Is Built for Everyone</h1>
        <p className="mt-2 text-sm text-gray-500">Across the Arab world and beyond</p>
      </div>

      <section className="mb-10">
        <div className="grid gap-6 sm:grid-cols-2">
          {PILLARS.map((p) => (
            <div key={p.title} className="rounded-2xl border border-gray-200 bg-white p-6">
              <p.icon size={22} className="mb-3 text-violet-600" />
              <h3 className="text-base font-bold text-gray-900">{p.title}</h3>
              <p className="mt-2 text-sm text-gray-600 leading-relaxed">{p.desc}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mb-10 rounded-2xl border border-violet-200 bg-violet-50 p-6 text-center">
        <p className="text-sm text-violet-700">Our team speaks 12+ languages and represents 8+ nationalities. Diversity isn&apos;t a program — it&apos;s who we are.</p>
      </section>

      <section className="text-center">
        <Link href="/careers" className="inline-block rounded-xl bg-[#0071CE] px-6 py-3 text-sm font-semibold text-white hover:bg-[#005ba3]">
          Join Our Team
        </Link>
      </section>
    </div>
  );
}
