import Link from 'next/link';

const LAST_UPDATED = 'March 30, 2026';

export default function RefundPolicyPage() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <h1 className="text-3xl font-extrabold text-gray-900">Refund Policy</h1>
      <p className="mt-2 text-sm text-gray-500">Last updated: {LAST_UPDATED}</p>

      <div className="mt-8 space-y-6 text-sm leading-7 text-gray-700">
        <section>
          <h2 className="text-lg font-bold text-gray-900">1. Eligibility</h2>
          <p>
            Buyers may request a refund if an item is not delivered, is materially different from listing details,
            arrives damaged, or qualifies under platform dispute rules.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">2. Refund Windows</h2>
          <p>
            Refund requests should be submitted as soon as an issue is identified.
            Delayed requests may require additional verification and evidence.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">3. Required Evidence</h2>
          <p>
            Depending on dispute type, evidence may include delivery screenshots, photos, videos, seller messages,
            tracking details, and order history.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">4. Resolution Outcomes</h2>
          <p>
            Outcomes can include full refund, partial refund, replacement, or no refund based on evidence and policy checks.
            Some cases may escalate to manual review.
          </p>
        </section>
      </div>

      <div className="mt-10 rounded-xl bg-gray-50 p-4 text-xs text-gray-600">
        This page is a placeholder policy template and should be reviewed by legal counsel before public launch.
      </div>

      <div className="mt-6 flex gap-4 text-sm">
        <Link href="/disputes/new" className="text-[#0071CE] hover:underline">Open a Dispute</Link>
        <Link href="/legal/terms" className="text-[#0071CE] hover:underline">Terms of Service</Link>
      </div>
    </div>
  );
}
