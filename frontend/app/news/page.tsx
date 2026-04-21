import Link from 'next/link';
import { Newspaper, Calendar, ArrowRight } from 'lucide-react';

const ARTICLES = [
  {
    title: 'Mnbarh Raises Seed Round to Expand Across GCC',
    date: '2025-12-01',
    summary: 'We\'re excited to announce our seed funding round, which will fuel our expansion into Saudi Arabia, Kuwait, and Bahrain in 2026.',
    tag: 'Funding',
  },
  {
    title: 'Mnbarh Partners with PayMob for Seamless Egypt Payments',
    date: '2026-01-20',
    summary: 'Egyptian buyers and sellers can now pay and receive funds through PayMob — faster, cheaper, and fully local.',
    tag: 'Partnership',
  },
  {
    title: 'Mnbarh Hits 100,000 Active Listings Milestone',
    date: '2026-03-01',
    summary: 'Our community has listed over 100,000 items — from electronics and fashion to cars and real estate. Thank you for growing with us.',
    tag: 'Milestone',
  },
];

export default function NewsPage() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <div className="mb-8 text-center">
        <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-[#0071CE]/10 px-4 py-1.5 text-sm font-medium text-[#0071CE]">
          <Newspaper size={16} /> News & Press
        </div>
        <h1 className="text-3xl font-extrabold text-gray-900">Latest News</h1>
      </div>

      <section className="space-y-6">
        {ARTICLES.map((a) => (
          <article key={a.title} className="rounded-2xl border border-gray-200 bg-white p-6">
            <div className="flex items-center gap-3 mb-2">
              <span className="rounded-full bg-[#0071CE]/10 px-3 py-1 text-xs font-medium text-[#0071CE]">{a.tag}</span>
              <span className="flex items-center gap-1 text-xs text-gray-500"><Calendar size={12} /> {a.date}</span>
            </div>
            <h2 className="text-lg font-bold text-gray-900">{a.title}</h2>
            <p className="mt-2 text-sm text-gray-600 leading-relaxed">{a.summary}</p>
            <span className="mt-3 inline-flex items-center gap-1 text-xs font-semibold text-[#0071CE] hover:underline cursor-pointer">
              Read more <ArrowRight size={12} />
            </span>
          </article>
        ))}
      </section>

      <section className="mt-10 rounded-2xl border border-gray-200 bg-gray-50 p-6 text-center">
        <h3 className="text-sm font-bold text-gray-900">Press Contact</h3>
        <p className="mt-1 text-sm text-gray-600">press@mnbarh.com</p>
        <p className="text-xs text-gray-500 mt-1">Media kit available upon request</p>
      </section>
    </div>
  );
}
