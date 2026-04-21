import Link from 'next/link';
import { Megaphone, Calendar, ArrowRight } from 'lucide-react';

const ANNOUNCEMENTS = [
  {
    title: 'Mnbarh Launches in Saudi Arabia',
    date: '2026-01-15',
    summary: 'We\'re expanding to KSA — new payment methods (Mada, STC Pay) and local support team coming this quarter.',
    tag: 'Expansion',
  },
  {
    title: 'Buy via Traveler Is Now Available in 50+ Countries',
    date: '2026-02-01',
    summary: 'Our crowdshipping network just hit 50,000 verified travelers worldwide. Post a request from anywhere.',
    tag: 'Feature',
  },
  {
    title: 'New: AI Search Now in Beta',
    date: '2026-03-10',
    summary: 'Upload a photo of any product and find it on Mnbarh instantly. Try it in the search bar.',
    tag: 'Beta',
  },
];

export default function AnnouncementsPage() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <div className="mb-8 text-center">
        <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-[#0071CE]/10 px-4 py-1.5 text-sm font-medium text-[#0071CE]">
          <Megaphone size={16} /> Announcements
        </div>
        <h1 className="text-3xl font-extrabold text-gray-900">Latest Announcements</h1>
      </div>

      <section className="space-y-6">
        {ANNOUNCEMENTS.map((a) => (
          <article key={a.title} className="rounded-2xl border border-gray-200 bg-white p-6">
            <div className="flex items-center gap-3 mb-2">
              <span className="rounded-full bg-[#0071CE]/10 px-3 py-1 text-xs font-medium text-[#0071CE]">{a.tag}</span>
              <span className="flex items-center gap-1 text-xs text-gray-500"><Calendar size={12} /> {a.date}</span>
            </div>
            <h2 className="text-lg font-bold text-gray-900">{a.title}</h2>
            <p className="mt-2 text-sm text-gray-600 leading-relaxed">{a.summary}</p>
            <span className="mt-3 inline-flex items-center gap-1 text-xs font-semibold text-[#0071CE] cursor-pointer hover:underline">
              Read more <ArrowRight size={12} />
            </span>
          </article>
        ))}
      </section>
    </div>
  );
}
