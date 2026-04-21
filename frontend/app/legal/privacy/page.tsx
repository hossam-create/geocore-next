import Link from 'next/link';

const LAST_UPDATED = 'March 30, 2026';

export default function PrivacyPolicyPage() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <h1 className="text-3xl font-extrabold text-gray-900">Privacy Policy</h1>
      <p className="mt-2 text-sm text-gray-500">Last updated: {LAST_UPDATED}</p>

      <div className="mt-8 space-y-6 text-sm leading-7 text-gray-700">
        <section>
          <h2 className="text-lg font-bold text-gray-900">1. Information We Collect</h2>
          <p>
            We collect account details, listing content, transaction metadata, device diagnostics, and communication records needed to operate
            marketplace features, payment flows, and support.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">2. How We Use Data</h2>
          <p>
            Your data is used to provide core services, protect users from abuse, improve search relevance, process payments,
            and meet regulatory obligations in supported regions.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">3. Data Sharing</h2>
          <p>
            We share data with payment providers, infrastructure partners, and legal authorities only when required for service delivery,
            fraud prevention, or legal compliance.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">4. Data Retention</h2>
          <p>
            We retain data for as long as necessary to provide services, resolve disputes, and comply with legal record-keeping obligations.
            You can request account deletion, subject to required retention windows.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">5. Your Rights</h2>
          <p>
            Depending on your jurisdiction, you may request access, correction, deletion, or export of personal data.
            You may also object to certain processing activities.
          </p>
        </section>
      </div>

      <div className="mt-10 rounded-xl bg-gray-50 p-4 text-xs text-gray-600">
        This page is a placeholder legal template and should be reviewed by legal counsel before public launch.
      </div>

      <div className="mt-6 flex gap-4 text-sm">
        <Link href="/legal/terms" className="text-[#0071CE] hover:underline">Terms of Service</Link>
        <Link href="/legal/cookies" className="text-[#0071CE] hover:underline">Cookie Policy</Link>
      </div>
    </div>
  );
}
