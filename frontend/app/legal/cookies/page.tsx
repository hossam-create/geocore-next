import Link from 'next/link';

const LAST_UPDATED = 'March 30, 2026';

export default function CookiePolicyPage() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-10">
      <h1 className="text-3xl font-extrabold text-gray-900">Cookie Policy</h1>
      <p className="mt-2 text-sm text-gray-500">Last updated: {LAST_UPDATED}</p>

      <div className="mt-8 space-y-6 text-sm leading-7 text-gray-700">
        <section>
          <h2 className="text-lg font-bold text-gray-900">1. What Are Cookies</h2>
          <p>
            Cookies are small text files stored on your device to remember session state, security preferences,
            and user settings across visits.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">2. Cookies We Use</h2>
          <p>
            We use essential cookies for login and security, functional cookies for preferences, and analytics cookies
            to improve browsing and listing discovery experiences.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">3. Managing Cookie Preferences</h2>
          <p>
            You can control cookie behavior through browser settings. Disabling some cookies may affect site features,
            including authentication and saved preferences.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900">4. Third-Party Technologies</h2>
          <p>
            Some integrations may set their own cookies for fraud detection, performance, and payment processing.
            These providers operate under their own policies.
          </p>
        </section>
      </div>

      <div className="mt-10 rounded-xl bg-gray-50 p-4 text-xs text-gray-600">
        This page is a placeholder legal template and should be reviewed by legal counsel before public launch.
      </div>

      <div className="mt-6 flex gap-4 text-sm">
        <Link href="/legal/terms" className="text-[#0071CE] hover:underline">Terms of Service</Link>
        <Link href="/legal/privacy" className="text-[#0071CE] hover:underline">Privacy Policy</Link>
      </div>
    </div>
  );
}
