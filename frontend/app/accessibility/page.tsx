import Link from 'next/link';
import { Accessibility, Eye, Keyboard, Globe, Type, Mail } from 'lucide-react';

const FEATURES = [
  { icon: Eye, title: 'Screen Reader Support', desc: 'Full ARIA labels and semantic HTML across all pages.' },
  { icon: Keyboard, title: 'Keyboard Navigation', desc: 'Every feature is accessible without a mouse.' },
  { icon: Type, title: 'Text Size Adjustment', desc: 'Resize text in your browser — our layout adapts.' },
  { icon: Globe, title: 'Arabic RTL Support', desc: 'Full right-to-left layout and Arabic font stack for MENA users.' },
  { icon: Eye, title: 'High Contrast Mode', desc: 'Supports OS-level high contrast and dark mode preferences.' },
  { icon: Accessibility, title: 'WCAG 2.1 Level AA', desc: 'We aim for AA compliance across all user-facing pages.' },
];

export default function AccessibilityPage() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <div className="mb-8 text-center">
        <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-[#0071CE]/10 px-4 py-1.5 text-sm font-medium text-[#0071CE]">
          <Accessibility size={16} /> Accessibility
        </div>
        <h1 className="text-3xl font-extrabold text-gray-900">Accessibility Statement</h1>
        <p className="mt-2 text-sm text-gray-500 max-w-xl mx-auto">
          Mnbarh is committed to making our platform accessible to all users, regardless of ability.
        </p>
      </div>

      <section className="mb-10">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {FEATURES.map((f) => (
            <div key={f.title} className="rounded-2xl border border-gray-200 bg-white p-5">
              <f.icon size={20} className="mb-2 text-[#0071CE]" />
              <h3 className="text-sm font-bold text-gray-900">{f.title}</h3>
              <p className="mt-1 text-xs text-gray-600">{f.desc}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mb-10 rounded-2xl border border-gray-200 bg-gray-50 p-6">
        <h2 className="mb-3 text-lg font-bold text-gray-900">Our Commitment</h2>
        <p className="text-sm text-gray-600 leading-relaxed">
          We continuously audit our platform for accessibility issues and prioritize fixes. We follow WCAG 2.1 Level AA guidelines and test with assistive technologies including NVDA, JAWS, and VoiceOver.
        </p>
      </section>

      <section className="mb-10 rounded-2xl border border-[#0071CE]/20 bg-[#0071CE]/5 p-6 text-center">
        <Mail size={20} className="mx-auto mb-2 text-[#0071CE]" />
        <h3 className="text-sm font-bold text-gray-900">Report an Accessibility Issue</h3>
        <p className="mt-1 text-sm text-gray-600">accessibility@mnbarh.com</p>
      </section>

      <section className="text-center">
        <Link href="/help" className="inline-block rounded-xl bg-[#0071CE] px-6 py-3 text-sm font-semibold text-white hover:bg-[#005ba3]">
          Help Center
        </Link>
      </section>
    </div>
  );
}
