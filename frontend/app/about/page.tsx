import Link from 'next/link';
import { Target, Users, Heart, Globe, Shield, TrendingUp, Mail, MapPin, Phone } from 'lucide-react';

const VALUES = [
  { icon: Shield, title: 'Trust & Safety', desc: 'We prioritize secure transactions and protect both buyers and sellers through escrow and verification systems.' },
  { icon: Globe, title: 'Regional Focus', desc: 'Built for the GCC region, we understand local needs, currencies, and shipping requirements.' },
  { icon: Heart, title: 'Customer First', desc: 'Every decision we make starts with the question: does this help our users succeed?' },
  { icon: TrendingUp, title: 'Empowering Sellers', desc: 'We provide tools and support to help businesses of all sizes grow and reach new customers.' },
];

const TEAM = [
  { name: 'Ahmed Al-Rashid', role: 'CEO & Co-Founder', initial: 'A' },
  { name: 'Fatima Hassan', role: 'CTO & Co-Founder', initial: 'F' },
  { name: 'Omar Khalil', role: 'Head of Operations', initial: 'O' },
  { name: 'Layla Ibrahim', role: 'Head of Product', initial: 'L' },
];

export default function AboutPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10">
      <div className="mb-10 text-center">
        <h1 className="text-3xl font-extrabold text-gray-900">About Mnbarh</h1>
        <p className="mt-2 text-sm text-gray-500 max-w-2xl mx-auto">
          Mnbarh is the premier marketplace for the GCC region, connecting buyers and sellers across the Gulf with a secure, trusted platform.
        </p>
      </div>

      {/* Mission */}
      <section className="mb-12 rounded-2xl border border-gray-200 bg-white p-6">
        <div className="flex items-center gap-3 mb-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-[#0071CE]/10">
            <Target size={20} className="text-[#0071CE]" />
          </div>
          <h2 className="text-lg font-bold text-gray-900">Our Mission</h2>
        </div>
        <p className="text-sm text-gray-700 leading-relaxed">
          To empower commerce across the Gulf Cooperation Council region by providing a secure, transparent, and user-friendly marketplace. We believe everyone should have access to buy and sell with confidence, whether they are a small business owner in Dubai, a collector in Riyadh, or a family in Kuwait City.
        </p>
      </section>

      {/* Values */}
      <section className="mb-12">
        <h2 className="mb-4 text-lg font-bold text-gray-900">Our Values</h2>
        <div className="grid gap-4 sm:grid-cols-2">
          {VALUES.map((value) => (
            <div key={value.title} className="rounded-xl border border-gray-200 bg-white p-4">
              <div className="flex items-center gap-2 mb-2">
                <value.icon size={18} className="text-[#0071CE]" />
                <h3 className="text-sm font-semibold text-gray-900">{value.title}</h3>
              </div>
              <p className="text-xs text-gray-600 leading-relaxed">{value.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Team */}
      <section className="mb-12">
        <div className="flex items-center gap-3 mb-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-[#FFC220]/20">
            <Users size={20} className="text-gray-900" />
          </div>
          <h2 className="text-lg font-bold text-gray-900">Leadership Team</h2>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {TEAM.map((member) => (
            <div key={member.name} className="rounded-xl border border-gray-200 bg-white p-4 text-center">
              <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-[#0071CE]/10 text-lg font-bold text-[#0071CE]">
                {member.initial}
              </div>
              <h3 className="text-sm font-semibold text-gray-900">{member.name}</h3>
              <p className="text-xs text-gray-500">{member.role}</p>
            </div>
          ))}
        </div>
        <p className="mt-3 text-xs text-gray-400 text-center italic">Team information is placeholder — update before launch.</p>
      </section>

      {/* Contact */}
      <section className="rounded-2xl border border-gray-200 bg-gray-50 p-6">
        <h2 className="mb-4 text-lg font-bold text-gray-900">Contact Us</h2>
        <div className="grid gap-4 sm:grid-cols-3">
          <div className="flex items-center gap-2">
            <Mail size={16} className="text-gray-400" />
            <span className="text-sm text-gray-600">support@mnbarh.com</span>
          </div>
          <div className="flex items-center gap-2">
            <Phone size={16} className="text-gray-400" />
            <span className="text-sm text-gray-600">+971 4 XXX XXXX</span>
          </div>
          <div className="flex items-center gap-2">
            <MapPin size={16} className="text-gray-400" />
            <span className="text-sm text-gray-600">Dubai, UAE</span>
          </div>
        </div>
        <div className="mt-4 flex flex-wrap gap-3">
          <Link href="/help" className="rounded-xl border border-gray-200 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100">
            Help Center
          </Link>
          <Link href="/legal/terms" className="rounded-xl border border-gray-200 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100">
            Terms of Service
          </Link>
        </div>
      </section>
    </div>
  );
}
