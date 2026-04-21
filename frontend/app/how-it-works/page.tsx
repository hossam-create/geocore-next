import Link from 'next/link';
import { Search, CreditCard, Package, Store, Tag, Gavel, TrendingUp, Shield, Users } from 'lucide-react';

const BUYER_STEPS = [
  { num: 1, title: 'Browse & Search', desc: 'Explore thousands of listings across categories. Use filters to find exactly what you need.', icon: Search },
  { num: 2, title: 'Bid or Buy Now', desc: 'Place bids in auctions or buy instantly at fixed prices. Add items to your cart for easy checkout.', icon: Tag },
  { num: 3, title: 'Secure Payment', desc: 'Pay with cards or wallet balance. Funds are held in escrow until delivery is confirmed.', icon: CreditCard },
  { num: 4, title: 'Receive & Confirm', desc: 'Track your shipment, receive your item, and confirm delivery to release payment to the seller.', icon: Package },
];

const SELLER_STEPS = [
  { num: 1, title: 'Create Listings', desc: 'Upload photos, write descriptions, set prices, and choose fixed-price or auction format.', icon: Store },
  { num: 2, title: 'Get Orders', desc: 'Receive order notifications. Confirm orders and prepare packages for shipping.', icon: Package },
  { num: 3, title: 'Ship & Track', desc: 'Mark items as shipped with tracking numbers. Buyers receive real-time updates.', icon: TrendingUp },
  { num: 4, title: 'Get Paid', desc: 'Once delivery is confirmed, funds are released to your wallet. Withdraw anytime.', icon: CreditCard },
];

const AUCTION_STEPS = [
  { num: 1, title: 'Find Auctions', desc: 'Browse auction listings with countdown timers. Watch items to get notified before they end.', icon: Search },
  { num: 2, title: 'Place Your Bid', desc: 'Enter your maximum bid. The system auto-bids for you up to your limit.', icon: Gavel },
  { num: 3, title: 'Win & Pay', desc: 'If you are the highest bidder when time runs out, you win! Complete payment to claim the item.', icon: CreditCard },
  { num: 4, title: 'Enjoy Protection', desc: 'Escrow holds your payment until the item arrives safely. Dispute if there is an issue.', icon: Shield },
];

export default function HowItWorksPage() {
  return (
    <div className="mx-auto max-w-6xl px-4 py-10">
      <div className="mb-10 text-center">
        <h1 className="text-3xl font-extrabold text-gray-900">How Mnbarh Works</h1>
        <p className="mt-2 text-sm text-gray-500">A simple, secure marketplace for buyers and sellers across the GCC region.</p>
      </div>

      {/* For Buyers */}
      <section className="mb-12">
        <div className="mb-4 flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-[#0071CE] text-white">
            <Users size={16} />
          </div>
          <h2 className="text-xl font-bold text-gray-900">For Buyers</h2>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {BUYER_STEPS.map((step) => (
            <div key={step.num} className="rounded-2xl border border-gray-200 bg-white p-4">
              <div className="mb-2 flex items-center gap-2">
                <span className="flex h-6 w-6 items-center justify-center rounded-full bg-[#0071CE] text-xs font-bold text-white">{step.num}</span>
                <step.icon size={16} className="text-[#0071CE]" />
              </div>
              <h3 className="text-sm font-bold text-gray-900">{step.title}</h3>
              <p className="mt-1 text-xs text-gray-600 leading-relaxed">{step.desc}</p>
            </div>
          ))}
        </div>
        <div className="mt-4 text-center">
          <Link href="/listings" className="inline-block rounded-xl bg-[#0071CE] px-5 py-2.5 text-sm font-semibold text-white hover:bg-[#005ba3]">
            Start Buying
          </Link>
        </div>
      </section>

      {/* For Sellers */}
      <section className="mb-12">
        <div className="mb-4 flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-[#FFC220] text-gray-900">
            <Store size={16} />
          </div>
          <h2 className="text-xl font-bold text-gray-900">For Sellers</h2>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {SELLER_STEPS.map((step) => (
            <div key={step.num} className="rounded-2xl border border-gray-200 bg-white p-4">
              <div className="mb-2 flex items-center gap-2">
                <span className="flex h-6 w-6 items-center justify-center rounded-full bg-[#FFC220] text-xs font-bold text-gray-900">{step.num}</span>
                <step.icon size={16} className="text-[#FFC220]" />
              </div>
              <h3 className="text-sm font-bold text-gray-900">{step.title}</h3>
              <p className="mt-1 text-xs text-gray-600 leading-relaxed">{step.desc}</p>
            </div>
          ))}
        </div>
        <div className="mt-4 text-center">
          <Link href="/sell" className="inline-block rounded-xl bg-[#FFC220] px-5 py-2.5 text-sm font-semibold text-gray-900 hover:bg-yellow-400">
            Start Selling
          </Link>
        </div>
      </section>

      {/* For Auction Bidders */}
      <section className="mb-12">
        <div className="mb-4 flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-rose-500 text-white">
            <Gavel size={16} />
          </div>
          <h2 className="text-xl font-bold text-gray-900">For Auction Bidders</h2>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {AUCTION_STEPS.map((step) => (
            <div key={step.num} className="rounded-2xl border border-gray-200 bg-white p-4">
              <div className="mb-2 flex items-center gap-2">
                <span className="flex h-6 w-6 items-center justify-center rounded-full bg-rose-500 text-xs font-bold text-white">{step.num}</span>
                <step.icon size={16} className="text-rose-500" />
              </div>
              <h3 className="text-sm font-bold text-gray-900">{step.title}</h3>
              <p className="mt-1 text-xs text-gray-600 leading-relaxed">{step.desc}</p>
            </div>
          ))}
        </div>
        <div className="mt-4 text-center">
          <Link href="/auctions" className="inline-block rounded-xl bg-rose-500 px-5 py-2.5 text-sm font-semibold text-white hover:bg-rose-600">
            Browse Auctions
          </Link>
        </div>
      </section>

      {/* Trust & Safety */}
      <section className="rounded-2xl border border-gray-200 bg-gray-50 p-6 text-center">
        <h2 className="text-lg font-bold text-gray-900">Built for Trust & Safety</h2>
        <p className="mt-2 text-sm text-gray-600">
          Every transaction is protected by escrow. Sellers only get paid after you confirm delivery.
          If something goes wrong, our dispute system is here to help.
        </p>
        <div className="mt-4 flex flex-wrap justify-center gap-3">
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
