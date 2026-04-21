import Link from 'next/link';
import { BookOpen, CreditCard, Shield, ShoppingBag, Store, HelpCircle } from 'lucide-react';

const CATEGORIES = [
  {
    title: 'Buying',
    description: 'Learn how to browse, bid, and purchase items safely.',
    icon: ShoppingBag,
    href: '/help/buying',
    links: ['How to place a bid', 'Payment methods', 'Delivery tracking', 'Returns & refunds'],
  },
  {
    title: 'Selling',
    description: 'Create listings, manage orders, and grow your store.',
    icon: Store,
    href: '/help/selling',
    links: ['Create a listing', 'Pricing strategies', 'Shipping best practices', 'Seller fees'],
  },
  {
    title: 'Payments',
    description: 'Understand payment flows, escrow, and payouts.',
    icon: CreditCard,
    href: '/help/faq?category=payments',
    links: ['Accepted payment methods', 'Escrow protection', 'Withdrawals', 'Stripe integration'],
  },
  {
    title: 'Account',
    description: 'Manage your profile, security, and preferences.',
    icon: Shield,
    href: '/help/faq?category=account',
    links: ['Verification (KYC)', 'Two-factor auth', 'Password reset', 'Delete account'],
  },
  {
    title: 'Safety',
    description: 'Stay safe from fraud and suspicious activity.',
    icon: Shield,
    href: '/help/faq?category=safety',
    links: ['Recognize scams', 'Report a user', 'Secure transactions', 'Privacy tips'],
  },
  {
    title: 'FAQ',
    description: 'Quick answers to common questions.',
    icon: HelpCircle,
    href: '/help/faq',
    links: ['General questions', 'Platform rules', 'Technical help', 'Contact support'],
  },
];

export default function HelpCenterPage() {
  return (
    <div className="mx-auto max-w-6xl px-4 py-10">
      <div className="mb-8 text-center">
        <h1 className="text-3xl font-extrabold text-gray-900">Help Center</h1>
        <p className="mt-2 text-sm text-gray-500">Find guides, FAQs, and support resources to get the most out of Mnbarh.</p>
        <div className="mt-4">
          <Link href="/help/faq" className="inline-block rounded-xl bg-[#0071CE] px-5 py-2.5 text-sm font-semibold text-white hover:bg-[#005ba3]">
            Browse FAQ
          </Link>
        </div>
      </div>

      <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
        {CATEGORIES.map((cat) => (
          <Link key={cat.title} href={cat.href} className="group rounded-2xl border border-gray-200 bg-white p-5 hover:border-[#0071CE]/30 hover:shadow-md transition-all">
            <div className="flex items-center gap-3 mb-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-blue-50 text-[#0071CE]">
                <cat.icon size={20} />
              </div>
              <h2 className="text-base font-bold text-gray-900 group-hover:text-[#0071CE]">{cat.title}</h2>
            </div>
            <p className="text-sm text-gray-600 mb-3">{cat.description}</p>
            <ul className="space-y-1">
              {cat.links.map((l) => (
                <li key={l} className="text-xs text-gray-500">• {l}</li>
              ))}
            </ul>
          </Link>
        ))}
      </div>
    </div>
  );
}
