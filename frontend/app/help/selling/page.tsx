import Link from 'next/link';

export default function HelpSellingPage() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <div className="mb-6">
        <Link href="/help" className="text-sm text-[#0071CE] hover:underline">← Back to Help Center</Link>
      </div>

      <h1 className="text-3xl font-extrabold text-gray-900">Seller Guide</h1>
      <p className="mt-2 text-sm text-gray-500">Learn how to list items, manage orders, and grow your business on Mnbarh.</p>

      <div className="mt-8 space-y-8 text-sm leading-7 text-gray-700">
        <section>
          <h2 className="text-lg font-bold text-gray-900">1. Creating a Listing</h2>
          <p>
            Click "Sell" in the header. Choose a category, add a clear title, detailed description, and high-quality photos.
            Set your price and choose between fixed-price or auction format.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">2. Pricing & Fees</h2>
          <p>
            Competitive pricing attracts more buyers. When your item sells, a small commission is deducted.
            Check the Seller Center for current fee rates by category.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">3. Managing Orders</h2>
          <p>
            When a buyer purchases, you will see the order in your Sales Orders page. Confirm the order, prepare the package,
            and mark it as shipped with tracking info.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">4. Shipping Best Practices</h2>
          <p>
            Use reliable carriers, pack items securely, and upload tracking promptly. Fast shipping and good communication
            lead to better reviews and repeat buyers.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">5. Getting Paid</h2>
          <p>
            Payments are held in escrow until the buyer confirms delivery. Once released, funds appear in your wallet.
            Withdraw to your bank at any time.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">6. Handling Disputes</h2>
          <p>
            If a buyer opens a dispute, respond quickly with evidence (photos, messages, tracking). Our team reviews both sides
            before deciding on a resolution.
          </p>
        </section>
      </div>

      <div className="mt-8 flex gap-4 text-sm">
        <Link href="/help/faq" className="text-[#0071CE] hover:underline">FAQ</Link>
        <Link href="/dashboard" className="text-[#0071CE] hover:underline">Seller Dashboard</Link>
      </div>
    </div>
  );
}
