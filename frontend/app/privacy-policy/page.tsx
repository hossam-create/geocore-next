'use client';

export default function PrivacyPolicyPage() {
  return (
    <div className="max-w-3xl mx-auto px-4 py-12">
      <h1 className="text-3xl font-bold text-gray-900 mb-2">Privacy Policy</h1>
      <p className="text-sm text-gray-500 mb-8">Last Updated: April 2026</p>

      <div className="prose prose-gray max-w-none space-y-6 text-sm leading-relaxed text-gray-700">

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">1. Who We Are</h2>
          <p>
            Mnbarh (&quot;we,&quot; &quot;us,&quot; &quot;our&quot;) operates a global marketplace platform
            connecting buyers, sellers, and travelers. Contact: <a href="mailto:legal@mnbarh.com" className="text-[#0071CE] hover:underline">legal@mnbarh.com</a>
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">2. Data We Collect</h2>
          <h3 className="font-semibold text-gray-800 mt-4 mb-2">2a. Data You Provide</h3>
          <ul className="list-disc pl-5 space-y-1">
            <li>Account information: name, email, password (hashed), phone number</li>
            <li>KYC documents: government ID, selfie (encrypted at rest)</li>
            <li>Payment info: processed by Stripe — we store only last 4 digits &amp; expiry</li>
            <li>Listings: photos, descriptions, location</li>
            <li>Communications: messages between users</li>
          </ul>
          <h3 className="font-semibold text-gray-800 mt-4 mb-2">2b. Data We Collect Automatically</h3>
          <ul className="list-disc pl-5 space-y-1">
            <li>IP address, browser type, device information</li>
            <li>Pages visited, time spent, interactions (analytics)</li>
            <li>Cookies (see our Cookie Policy)</li>
          </ul>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">3. How We Use Your Data</h2>
          <ul className="list-disc pl-5 space-y-1">
            <li>Provide and improve our services</li>
            <li>Process payments and prevent fraud</li>
            <li>Send transactional emails (not marketing without consent)</li>
            <li>Comply with legal obligations</li>
            <li>Resolve disputes between users</li>
          </ul>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">4. Data Sharing</h2>
          <p>We <strong>never sell</strong> your data. We share with:</p>
          <ul className="list-disc pl-5 space-y-1">
            <li><strong>Stripe</strong> — payment processing</li>
            <li><strong>Cloudflare</strong> — CDN and DDoS protection</li>
            <li><strong>Cloud hosting providers</strong> — infrastructure</li>
            <li><strong>Law enforcement</strong> — when legally required</li>
          </ul>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">5. Your Rights</h2>
          <ul className="list-disc pl-5 space-y-1">
            <li><strong>Access:</strong> Request a copy of your data</li>
            <li><strong>Correction:</strong> Fix inaccurate data</li>
            <li><strong>Deletion:</strong> &quot;Right to be forgotten&quot; — email legal@mnbarh.com</li>
            <li><strong>Portability:</strong> Export your data in JSON format</li>
            <li><strong>Objection:</strong> Opt out of analytics</li>
          </ul>
          <p className="mt-3 text-xs text-gray-500">
            For Egypt/GCC users: We comply with Egypt&apos;s Personal Data Protection Law
            (Law No. 151 of 2020) and UAE Federal Decree-Law No. 45 of 2021.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">6. Data Retention</h2>
          <ul className="list-disc pl-5 space-y-1">
            <li>Active accounts: retained while account exists</li>
            <li>Deleted accounts: purged within 90 days</li>
            <li>Transaction records: 7 years (financial regulations)</li>
            <li>Security logs: 12 months</li>
          </ul>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">7. Security</h2>
          <ul className="list-disc pl-5 space-y-1">
            <li>TLS 1.2/1.3 encryption in transit</li>
            <li>Encryption at rest for sensitive data</li>
            <li>Regular security audits</li>
            <li>Rate limiting and DDoS protection</li>
          </ul>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">8. Children</h2>
          <p>Our platform is for users 18+. We do not knowingly collect data from minors.</p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">9. Changes</h2>
          <p>We will notify users 30 days before material changes via email.</p>
        </section>

        <section>
          <h2 className="text-lg font-bold text-gray-900 mt-8 mb-3">10. Contact</h2>
          <p>
            <a href="mailto:privacy@mnbarh.com" className="text-[#0071CE] hover:underline">privacy@mnbarh.com</a>
            {' | '}
            <a href="mailto:legal@mnbarh.com" className="text-[#0071CE] hover:underline">legal@mnbarh.com</a>
          </p>
        </section>

        <p className="text-xs text-gray-400 mt-10 pt-6 border-t border-gray-200">
          &copy; {new Date().getFullYear()} Mnbarh. All rights reserved.
        </p>
      </div>
    </div>
  );
}
