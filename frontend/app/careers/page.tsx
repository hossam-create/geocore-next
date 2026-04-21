import Link from 'next/link';
import { Briefcase, Globe, Heart, Laptop, GraduationCap, ArrowRight } from 'lucide-react';

const OPENINGS = [
  { title: 'Senior Go Engineer', location: 'Remote / Cairo', type: 'Full-time', dept: 'Engineering' },
  { title: 'Product Manager', location: 'Dubai', type: 'Full-time', dept: 'Product' },
  { title: 'Customer Success Lead', location: 'Riyadh', type: 'Full-time', dept: 'Support' },
  { title: 'Frontend Engineer (React/Next.js)', location: 'Remote', type: 'Full-time', dept: 'Engineering' },
  { title: 'Growth Marketing Manager', location: 'Dubai / Remote', type: 'Full-time', dept: 'Marketing' },
];

const PERKS = [
  { icon: Laptop, title: 'Remote-First', desc: 'Work from anywhere in the MENA region.' },
  { icon: Heart, title: 'Health Insurance', desc: 'Comprehensive coverage for you and family.' },
  { icon: GraduationCap, title: 'Learning Budget', desc: '$2,000/year for courses and conferences.' },
  { icon: Globe, title: 'Equity', desc: 'Stock options — you own a piece of the future.' },
];

export default function CareersPage() {
  return (
    <div className="min-h-screen">
      <section className="bg-gradient-to-br from-[#0071CE] to-[#003f75] text-white py-14">
        <div className="mx-auto max-w-5xl px-4 text-center">
          <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-white/15 px-4 py-1.5 text-sm font-medium">
            <Briefcase size={16} /> Careers
          </div>
          <h1 className="text-3xl font-extrabold md:text-4xl">Build the Future of Commerce in the Arab World</h1>
          <p className="mx-auto mt-3 max-w-xl text-blue-100">
            We&apos;re building the marketplace where every person can buy, sell, and thrive.
          </p>
        </div>
      </section>

      {/* Openings */}
      <section className="py-14">
        <div className="mx-auto max-w-4xl px-4">
          <h2 className="mb-6 text-2xl font-extrabold text-gray-900">Current Openings</h2>
          <div className="space-y-3">
            {OPENINGS.map((job) => (
              <div key={job.title} className="flex items-center justify-between rounded-xl border border-gray-200 bg-white p-5">
                <div>
                  <h3 className="text-sm font-bold text-gray-900">{job.title}</h3>
                  <p className="text-xs text-gray-500">{job.dept} · {job.location} · {job.type}</p>
                </div>
                <Link href="/contact" className="rounded-full bg-[#0071CE] px-4 py-2 text-xs font-semibold text-white hover:bg-[#005ba3]">
                  Apply
                </Link>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Culture */}
      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-4xl px-4 text-center">
          <h2 className="mb-4 text-2xl font-extrabold text-gray-900">Our Culture</h2>
          <p className="mx-auto max-w-2xl text-sm text-gray-600 leading-relaxed">
            We&apos;re a diverse team from across the Arab world — engineers, designers, and operators united by a mission to build the best marketplace for our region. We move fast, ship often, and learn from every mistake.
          </p>
        </div>
      </section>

      {/* Perks */}
      <section className="py-14">
        <div className="mx-auto max-w-4xl px-4">
          <h2 className="mb-8 text-center text-2xl font-extrabold text-gray-900">Perks & Benefits</h2>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {PERKS.map((p) => (
              <div key={p.title} className="rounded-2xl border border-gray-200 bg-white p-5 text-center">
                <p.icon size={22} className="mx-auto mb-2 text-[#0071CE]" />
                <h3 className="text-sm font-bold text-gray-900">{p.title}</h3>
                <p className="mt-1 text-xs text-gray-600">{p.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="bg-[#FFC220] py-10 text-center">
        <p className="text-sm text-gray-800">Don&apos;t see your role? Send your CV to <strong>careers@mnbarh.com</strong></p>
      </section>
    </div>
  );
}
