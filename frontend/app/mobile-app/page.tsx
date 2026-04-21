import Link from 'next/link';
import { Smartphone, Bell, Camera, Shield, Zap, ArrowRight, Download } from 'lucide-react';

const FEATURES = [
  { icon: Bell, title: 'Instant Notifications', desc: 'Never miss a bid, sale, or message. Real-time push alerts.' },
  { icon: Camera, title: 'Scan & Search', desc: 'Point your camera at any product and find it on Mnbarh instantly.' },
  { icon: Shield, title: 'Biometric Login', desc: 'Face ID and fingerprint for fast, secure access.' },
  { icon: Zap, title: 'Quick Bid', desc: 'One-tap bidding in live auctions. No delays.' },
];

export default function MobileAppPage() {
  return (
    <div className="min-h-screen">
      <section className="bg-gradient-to-br from-[#0071CE] to-[#003f75] text-white py-16">
        <div className="mx-auto max-w-5xl px-4 text-center">
          <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-white/15 px-4 py-1.5 text-sm font-medium">
            <Smartphone size={16} /> Mobile App
          </div>
          <h1 className="text-3xl font-extrabold md:text-4xl">Mnbarh in Your Pocket</h1>
          <p className="mx-auto mt-3 max-w-xl text-blue-100">
            Buy, sell, and bid on the go. Available for iOS and Android.
          </p>
          <div className="mt-8 flex flex-wrap justify-center gap-4">
            <button className="inline-flex items-center gap-2 rounded-full bg-white px-6 py-3 text-sm font-bold text-[#0071CE] hover:bg-gray-100">
              <Download size={16} /> Download for iOS
            </button>
            <button className="inline-flex items-center gap-2 rounded-full bg-white px-6 py-3 text-sm font-bold text-[#0071CE] hover:bg-gray-100">
              <Download size={16} /> Download for Android
            </button>
          </div>
        </div>
      </section>

      <section className="py-14">
        <div className="mx-auto max-w-4xl px-4">
          <h2 className="mb-8 text-center text-2xl font-extrabold text-gray-900">App Features</h2>
          <div className="grid gap-4 sm:grid-cols-2">
            {FEATURES.map((f) => (
              <div key={f.title} className="rounded-2xl border border-gray-200 bg-white p-6">
                <f.icon size={22} className="mb-3 text-[#0071CE]" />
                <h3 className="text-sm font-bold text-gray-900">{f.title}</h3>
                <p className="mt-1 text-xs text-gray-600">{f.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="bg-gray-50 py-14 text-center">
        <p className="text-sm text-gray-500">Coming soon to the App Store and Google Play.</p>
        <p className="mt-1 text-xs text-gray-400">Sign up to be notified when we launch.</p>
      </section>
    </div>
  );
}
