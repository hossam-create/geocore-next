import Link from 'next/link';

export default function HelpBuyingPage() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <div className="mb-6">
        <Link href="/help" className="text-sm text-[#0071CE] hover:underline">← Back to Help Center</Link>
      </div>

      <h1 className="text-3xl font-extrabold text-gray-900">Buyer Guide</h1>
      <p className="mt-2 text-sm text-gray-500">Everything you need to know about browsing, bidding, and buying safely on Mnbarh.</p>

      <div className="mt-8 space-y-8 text-sm leading-7 text-gray-700">
        <section>
          <h2 className="text-lg font-bold text-gray-900">1. Browsing & Searching</h2>
          <p>
            Use the search bar or browse categories to find items. Filters let you narrow by price, location, condition, and listing type.
            Save interesting listings to your watchlist by tapping the heart icon.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">2. Placing a Bid</h2>
          <p>
            On auction listings, enter your maximum bid. The system will automatically bid the minimum needed to keep you winning,
            up to your max. You will be notified if you are outbid.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">3. Buy Now</h2>
          <p>
            Some listings offer a "Buy Now" price. Clicking it skips the auction and lets you purchase immediately.
            Add items to your cart and proceed to checkout when ready.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">4. Checkout & Payment</h2>
          <p>
            At checkout, review your order and pay with a card or wallet balance. Payments are held in escrow until you confirm delivery.
            This protects you if something goes wrong.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">5. Tracking & Delivery</h2>
          <p>
            Once the seller ships, you will see tracking info on your order page. Confirm delivery when the item arrives safely.
            If there is an issue, you can open a dispute.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">6. Returns & Refunds</h2>
          <p>
            If the item is not as described, damaged, or never arrives, you can request a refund through the dispute system.
            See our Refund Policy for details.
          </p>
        </section>
      </div>

      <div className="mt-8 flex gap-4 text-sm">
        <Link href="/help/faq" className="text-[#0071CE] hover:underline">FAQ</Link>
        <Link href="/refund-policy" className="text-[#0071CE] hover:underline">Refund Policy</Link>
      </div>
    </div>
  );
}
