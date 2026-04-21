'use client';

import { useState } from 'react';
import Link from 'next/link';
import { ChevronDown, Search } from 'lucide-react';

type FAQItem = { q: string; a: string };
type FAQCategory = { title: string; items: FAQItem[] };

const FAQ_DATA: FAQCategory[] = [
  {
    title: 'General',
    items: [
      { q: 'What is Mnbarh?', a: 'Mnbarh is a GCC-focused marketplace connecting buyers and sellers across the region. We support fixed-price listings, auctions, and secure escrow payments.' },
      { q: 'Which countries are supported?', a: 'We currently serve the UAE, Saudi Arabia, Kuwait, Qatar, Bahrain, and Oman. More markets will be added soon.' },
      { q: 'Is registration free?', a: 'Yes, creating an account is completely free. Sellers only pay fees when they make a sale.' },
      { q: 'How do I contact support?', a: 'You can reach support through the Contact Us link in the footer, or by opening a dispute for order-related issues.' },
    ],
  },
  {
    title: 'Buying',
    items: [
      { q: 'How do I place a bid?', a: 'Open any auction listing, enter your bid amount, and confirm. If you are outbid, you will receive a notification so you can bid again.' },
      { q: 'Can I buy instantly without bidding?', a: 'Yes, if a listing has a "Buy Now" price, you can purchase immediately at that price.' },
      { q: 'How is my payment protected?', a: 'Payments are held in escrow until you confirm delivery. If there is an issue, you can open a dispute.' },
      { q: 'When will I receive my item?', a: 'Delivery times depend on the seller and carrier. You can track progress from your Orders page.' },
      { q: 'Can I cancel an order?', a: 'You can request cancellation before the seller ships. After shipping, you may need to go through the return or dispute process.' },
    ],
  },
  {
    title: 'Selling',
    items: [
      { q: 'How do I create a listing?', a: 'Click "Sell" in the header, choose a category, fill in details, set pricing, and publish. Listings go live immediately.' },
      { q: 'What are the seller fees?', a: 'Sellers pay a small commission on each sale. Exact rates depend on category and listing type. Check the Seller Center for details.' },
      { q: 'How do I get paid?', a: 'Once the buyer confirms delivery (or the escrow period ends), funds are released to your wallet. You can withdraw to your bank.' },
      { q: 'Can I edit a live listing?', a: 'Yes, you can edit title, description, and images. Price changes may require re-approval for auctions with active bids.' },
      { q: 'What happens if a buyer disputes?', a: 'You will be notified and can respond with evidence. Our team reviews both sides before deciding on a resolution.' },
    ],
  },
  {
    title: 'Payments',
    items: [
      { q: 'Which payment methods are accepted?', a: 'We accept major credit/debit cards via Stripe. Wallet balance can also be used for purchases.' },
      { q: 'What is escrow?', a: 'Escrow holds your payment securely until the item is delivered and confirmed. This protects both buyer and seller.' },
      { q: 'How do refunds work?', a: 'If a dispute is resolved in your favor, a refund is issued to your original payment method or wallet.' },
      { q: 'How do I withdraw from my wallet?', a: 'Go to your Wallet page, click Withdraw, and link a bank account. Withdrawals typically process within 3–5 business days.' },
    ],
  },
  {
    title: 'Account',
    items: [
      { q: 'How do I verify my account?', a: 'Complete KYC verification from your Profile page. You will need to upload ID and proof of address.' },
      { q: 'I forgot my password. What now?', a: 'Click "Forgot password" on the login page and follow the email instructions to reset it.' },
      { q: 'Can I change my email or phone?', a: 'Yes, update your contact details from your Profile settings. Verification may be required for security.' },
      { q: 'How do I delete my account?', a: 'Contact support to request account deletion. Note that active orders must be resolved first.' },
    ],
  },
  {
    title: 'Safety',
    items: [
      { q: 'How do I recognize a scam?', a: 'Be wary of deals that look too good to be true, requests to pay outside the platform, or sellers avoiding escrow.' },
      { q: 'How do I report a suspicious user?', a: 'Open the user’s profile or listing, click Report, and choose the reason. Our safety team will investigate.' },
      { q: 'Is my personal data safe?', a: 'We follow strict privacy practices. See our Privacy Policy for details on data handling and your rights.' },
    ],
  },
];

function AccordionItem({ item, open, onToggle }: { item: FAQItem; open: boolean; onToggle: () => void }) {
  return (
    <div className="border-b border-gray-100">
      <button
        onClick={onToggle}
        className="flex w-full items-center justify-between gap-3 py-3 text-left"
      >
        <span className="text-sm font-medium text-gray-900">{item.q}</span>
        <ChevronDown size={16} className={`shrink-0 text-gray-400 transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>
      {open && <p className="pb-3 text-sm text-gray-600 leading-relaxed">{item.a}</p>}
    </div>
  );
}

export default function FAQPage() {
  const [query, setQuery] = useState('');
  const [openMap, setOpenMap] = useState<Record<string, boolean>>({});

  const filtered = query.trim()
    ? FAQ_DATA.map((cat) => ({
        ...cat,
        items: cat.items.filter(
          (it) =>
            it.q.toLowerCase().includes(query.toLowerCase()) ||
            it.a.toLowerCase().includes(query.toLowerCase())
        ),
      })).filter((cat) => cat.items.length > 0)
    : FAQ_DATA;

  const toggle = (key: string) => setOpenMap((m) => ({ ...m, [key]: !m[key] }));

  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <div className="mb-6 text-center">
        <h1 className="text-3xl font-extrabold text-gray-900">Frequently Asked Questions</h1>
        <p className="mt-2 text-sm text-gray-500">Quick answers to common questions about buying, selling, and using Mnbarh.</p>
      </div>

      <div className="mb-6 flex justify-center">
        <div className="relative w-full max-w-md">
          <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search FAQ…"
            className="w-full rounded-xl border border-gray-200 py-2.5 pl-9 pr-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
          />
        </div>
      </div>

      {filtered.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-gray-300 bg-white p-8 text-center">
          <p className="text-sm text-gray-600">No results for "{query}". Try a different keyword.</p>
        </div>
      ) : (
        <div className="space-y-6">
          {filtered.map((cat) => (
            <section key={cat.title} className="rounded-2xl border border-gray-200 bg-white p-5">
              <h2 className="mb-3 text-base font-bold text-gray-900">{cat.title}</h2>
              <div>
                {cat.items.map((item, idx) => (
                  <AccordionItem
                    key={`${cat.title}-${idx}`}
                    item={item}
                    open={openMap[`${cat.title}-${idx}`] ?? false}
                    onToggle={() => toggle(`${cat.title}-${idx}`)}
                  />
                ))}
              </div>
            </section>
          ))}
        </div>
      )}

      <div className="mt-8 text-center text-sm">
        <p className="text-gray-500">Can’t find what you’re looking for?</p>
        <Link href="/help" className="text-[#0071CE] font-semibold hover:underline">Back to Help Center</Link>
      </div>
    </div>
  );
}
