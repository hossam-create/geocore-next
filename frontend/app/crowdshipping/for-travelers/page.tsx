import Link from 'next/link';
import { Plane, MapPin, MessageCircle, ShieldCheck, Wallet, ArrowRight, CheckCircle, DollarSign, Star, Clock, Package } from 'lucide-react';

const STEPS = [
  { icon: MapPin, title: 'Post Your Trip', desc: 'Enter your route, date, and available luggage space. Your trip goes live immediately.' },
  { icon: Package, title: 'Review Requests', desc: 'See all buyer requests matching your route. Review item details, photos, and tip amount.' },
  { icon: MessageCircle, title: 'Chat & Confirm', desc: 'Message the buyer through our secure chat. Agree on item, meeting point, tip, and delivery method.' },
  { icon: ShieldCheck, title: 'Secure Payment', desc: 'The buyer pays through escrow — you\'re guaranteed payment before you travel. If the trip is cancelled, payment is instantly refunded.' },
  { icon: Wallet, title: 'Pick Up & Deliver', desc: 'Pick up the item (or buy it yourself if requested). Deliver at destination, confirm in the app, receive your tip within 24h.' },
];

const EARNINGS = [
  { route: 'Cairo → Dubai', freq: 'Once a month', earning: '$80–150' },
  { route: 'London → Cairo', freq: 'Twice a year', earning: '$60–100' },
  { route: 'Weekly Gulf routes', freq: 'Weekly', earning: '$200–400/mo' },
];

const PROTECTIONS = [
  'Escrow guarantees you\'re paid before traveling',
  'Insurance covers items during transport (up to $1,500)',
  'Trust scores protect you from problematic requesters',
  'Mnbarh covers customs disputes if declared correctly',
];

export default function ForTravelersPage() {
  return (
    <div className="min-h-screen">
      {/* Hero */}
      <section className="bg-gradient-to-br from-emerald-600 to-emerald-800 text-white">
        <div className="mx-auto max-w-5xl px-4 py-14 text-center">
          <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-white/15 px-4 py-1.5 text-sm font-medium">
            <Plane size={16} /> For Travelers
          </div>
          <h1 className="text-3xl font-extrabold md:text-4xl">Turn Your Travels Into Income</h1>
          <p className="mx-auto mt-3 max-w-xl text-emerald-100">
            You&apos;re already going — why not earn extra for your trip expenses?
          </p>
          <Link href="/traveler/trips/new" className="mt-6 inline-flex items-center gap-2 rounded-full bg-[#FFC220] px-8 py-3.5 text-sm font-bold text-gray-900 hover:bg-yellow-400">
            Post Your First Trip — It&apos;s Free <ArrowRight size={16} />
          </Link>
        </div>
      </section>

      {/* How Travelers Earn */}
      <section className="py-14">
        <div className="mx-auto max-w-4xl px-4">
          <h2 className="mb-3 text-2xl font-extrabold text-gray-900">How Travelers Earn</h2>
          <p className="text-sm text-gray-600 leading-relaxed">
            Post your travel route on Mnbarh. Buyers who need items from your origin will contact you. You agree on a tip, pick up the item, and deliver it. It&apos;s that simple.
          </p>
        </div>
      </section>

      {/* Real Earning Examples */}
      <section className="bg-gray-50 py-14">
        <div className="mx-auto max-w-3xl px-4">
          <h2 className="mb-8 text-center text-2xl font-extrabold text-gray-900">Real Earning Examples</h2>
          <div className="grid gap-4 sm:grid-cols-3">
            {EARNINGS.map((e) => (
              <div key={e.route} className="rounded-2xl border border-gray-200 bg-white p-5 text-center">
                <DollarSign size={20} className="mx-auto mb-2 text-emerald-600" />
                <p className="text-sm font-bold text-gray-900">{e.route}</p>
                <p className="text-xs text-gray-500">{e.freq}</p>
                <p className="mt-2 text-lg font-extrabold text-emerald-700">{e.earning}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Step-by-Step */}
      <section className="py-14">
        <div className="mx-auto max-w-4xl px-4">
          <h2 className="mb-8 text-center text-2xl font-extrabold text-gray-900">Step-by-Step Guide</h2>
          <div className="space-y-4">
            {STEPS.map((step, i) => (
              <div key={step.title} className="flex items-start gap-4 rounded-2xl border border-gray-200 bg-white p-5">
                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-emerald-100 text-sm font-bold text-emerald-700">
                  {i + 1}
                </span>
                <div>
                  <h3 className="text-sm font-bold text-gray-900">{step.title}</h3>
                  <p className="mt-1 text-xs text-gray-600 leading-relaxed">{step.desc}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Traveler Protections */}
      <section className="bg-emerald-50 py-14">
        <div className="mx-auto max-w-4xl px-4">
          <h2 className="mb-6 text-center text-2xl font-extrabold text-gray-900">Traveler Protections</h2>
          <div className="grid gap-3 sm:grid-cols-2">
            {PROTECTIONS.map((p) => (
              <div key={p} className="flex items-start gap-2 rounded-xl border border-emerald-200 bg-white p-4">
                <CheckCircle size={16} className="mt-0.5 shrink-0 text-emerald-600" />
                <p className="text-sm text-gray-700">{p}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Safety Tip */}
      <section className="py-14">
        <div className="mx-auto max-w-3xl px-4">
          <div className="rounded-2xl border border-amber-200 bg-amber-50 p-6">
            <h3 className="flex items-center gap-2 text-sm font-bold text-amber-800">
              <Star size={16} /> Tips for Successful Travelers
            </h3>
            <p className="mt-2 text-sm text-amber-700 leading-relaxed">
              Only carry items you can inspect yourself. Never carry sealed packages you haven&apos;t verified. If anything feels wrong, decline — you&apos;re always protected.
            </p>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="bg-gradient-to-r from-emerald-600 to-emerald-800 py-14 text-center text-white">
        <h2 className="text-2xl font-extrabold">Start Earning on Your Next Trip</h2>
        <Link href="/traveler/trips/new" className="mt-6 inline-flex items-center gap-2 rounded-full bg-[#FFC220] px-8 py-3.5 text-sm font-bold text-gray-900 hover:bg-yellow-400">
          Post Your First Trip <ArrowRight size={16} />
        </Link>
        <div className="mt-6 flex justify-center gap-6 text-xs text-emerald-200">
          <Link href="/crowdshipping/trust" className="hover:text-white">Safety & Trust</Link>
          <Link href="/crowdshipping/insurance" className="hover:text-white">Insurance</Link>
          <Link href="/crowdshipping/payment" className="hover:text-white">Payment</Link>
        </div>
      </section>
    </div>
  );
}
