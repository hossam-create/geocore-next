import Link from 'next/link';

const LAST_UPDATED = 'March 30, 2026';

export default function TermsOfServicePage() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <h1 className="text-3xl font-extrabold text-gray-900">Terms of Service</h1>
      <p className="mt-2 text-sm text-gray-500">Last updated: {LAST_UPDATED}</p>

      <div className="mt-8 space-y-6 text-sm leading-7 text-gray-700">
        <section>
          <h2 className="text-lg font-bold text-gray-900">1. Marketplace Role</h2>
          <p>
            Mnbarh is a marketplace platform that connects buyers and sellers. We are not the owner of listed products unless explicitly stated.
            Transaction decisions, listing content, and shipping details remain the responsibility of users.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">2. Account Responsibilities</h2>
          <p>
            You must provide accurate account information and keep your credentials secure. You are responsible for all activity under your account,
            including bids, purchases, and listing updates.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">3. Payments, Escrow, and Orders</h2>
          <p>
            Payments are processed using integrated providers. For eligible transactions, funds may be held in escrow until delivery confirmation.
            Fraudulent or abusive payment behavior may result in account restrictions.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">4. Prohibited Conduct</h2>
          <p>
            Users may not post illegal items, misleading descriptions, counterfeit goods, malware links, or abusive content.
            Mnbarh reserves the right to remove listings and suspend accounts that violate platform policies.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">5. Limitation of Liability</h2>
          <p>
            To the extent permitted by law, Mnbarh is not liable for indirect or consequential damages arising from marketplace use,
            listing disputes, delays, or third-party service outages.
          </p>
        </section>
      </div>

      <div className="mt-10 rounded-xl bg-gray-50 p-4 text-xs text-gray-600">
        This page is a placeholder legal template and should be reviewed by legal counsel before public launch.
      </div>

      <div className="mt-6 flex gap-4 text-sm">
        <Link href="/legal/privacy" className="text-[#0071CE] hover:underline">Privacy Policy</Link>
        <Link href="/legal/cookies" className="text-[#0071CE] hover:underline">Cookie Policy</Link>
      </div>
    </div>
  );
}
