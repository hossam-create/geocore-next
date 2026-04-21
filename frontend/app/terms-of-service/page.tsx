'use client';

export default function TermsOfServicePage() {
  return (
    <div className="max-w-3xl mx-auto px-4 py-12">
      <h1 className="text-3xl font-bold text-gray-900 mb-2">Terms of Service</h1>
      <p className="text-sm text-gray-500 mb-8">Last Updated: April 2026</p>

      <div className="prose prose-gray max-w-none space-y-6 text-sm leading-relaxed text-gray-700">

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">1. Acceptance</h2>
          <p>By using Mnbarh, you agree to these terms. If you disagree, do not use the platform.</p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">2. Eligibility</h2>
          <ul className="list-disc pl-5 space-y-1">
            <li>Must be 18+ years old</li>
            <li>Must have legal capacity to enter contracts</li>
            <li>One account per person</li>
          </ul>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">3. User Accounts</h2>
          <ul className="list-disc pl-5 space-y-1">
            <li>You are responsible for your account security</li>
            <li>Do not share your password</li>
            <li>Report unauthorized access immediately</li>
            <li>We reserve the right to suspend accounts violating these terms</li>
          </ul>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">4. Buying</h2>
          <ul className="list-disc pl-5 space-y-1">
            <li>Bids and offers are legally binding commitments</li>
            <li>Payment is required within 24 hours of winning an auction</li>
            <li>Buyer Protection covers items not received or significantly not as described</li>
          </ul>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">5. Selling</h2>
          <ul className="list-disc pl-5 space-y-1">
            <li>Listings must be accurate and complete</li>
            <li>Prohibited items include: weapons, counterfeit goods, stolen items, adult content, illegal services</li>
            <li>Sellers are responsible for compliance with customs and import laws</li>
            <li>Mnbarh charges a commission on completed sales as detailed on our fees page</li>
          </ul>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">6. Traveler Service (Buy via Traveler)</h2>
          <ul className="list-disc pl-5 space-y-1">
            <li>Travelers agree to only transport legal items</li>
            <li>Travelers assume no liability for items not declared by buyer</li>
            <li>Escrow is released upon confirmed delivery</li>
          </ul>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">7. Fees</h2>
          <p>Detailed fee schedule available at the fees page. Fees include listing fees, success fees on completed sales, and optional promotional add-ons.</p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">8. Prohibited Conduct</h2>
          <ul className="list-disc pl-5 space-y-1">
            <li>Shill bidding or bid manipulation</li>
            <li>Feedback manipulation</li>
            <li>Circumventing fees (off-platform transactions)</li>
            <li>Spamming or harassing other users</li>
            <li>Scraping or automated access without permission</li>
          </ul>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">9. Dispute Resolution</h2>
          <ol className="list-decimal pl-5 space-y-1">
            <li>Contact the other party directly</li>
            <li>Open a dispute in our Resolution Center</li>
            <li>Mnbarh&apos;s decision is final for disputes under $1,000</li>
            <li>Binding arbitration for larger disputes</li>
          </ol>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">10. Limitation of Liability</h2>
          <p>
            Mnbarh is a marketplace — we are not a party to transactions between users.
            Maximum liability is limited to fees paid in the last 12 months.
          </p>
        </section>

        <p className="text-xs text-gray-400 mt-10 pt-6 border-t border-gray-200">
          &copy; {new Date().getFullYear()} Mnbarh. All rights reserved.
        </p>
      </div>
    </div>
  );
}
