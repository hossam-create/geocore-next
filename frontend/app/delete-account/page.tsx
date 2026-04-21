'use client';
import Link from 'next/link';
import { useState } from 'react';
import { AlertTriangle, CheckCircle, XCircle, ArrowRight, Shield } from 'lucide-react';

const DELETED = [
  'Your profile and personal data',
  'Your listings (active ones will be cancelled)',
  'Your messages and conversations',
];

const KEPT = [
  'Transaction records (kept for legal compliance for 7 years)',
  'Reviews you\'ve received (anonymized but kept)',
];

export default function DeleteAccountPage() {
  const [confirmed, setConfirmed] = useState(false);

  return (
    <div className="mx-auto max-w-3xl px-4 py-10">
      <div className="mb-8 text-center">
        <div className="mb-3 inline-flex items-center gap-2 rounded-full bg-red-100 px-4 py-1.5 text-sm font-medium text-red-700">
          <AlertTriangle size={16} /> Account Deletion
        </div>
        <h1 className="text-3xl font-extrabold text-gray-900">Delete Your Mnbarh Account</h1>
      </div>

      {/* Warning */}
      <div className="mb-8 rounded-2xl border border-amber-200 bg-amber-50 p-6">
        <div className="flex items-start gap-3">
          <AlertTriangle size={20} className="mt-0.5 shrink-0 text-amber-600" />
          <div>
            <h3 className="text-sm font-bold text-amber-800">This action is permanent and cannot be undone</h3>
            <p className="mt-1 text-sm text-amber-700">All your listings, orders, and messages will be deleted.</p>
          </div>
        </div>
      </div>

      {/* Before You Delete */}
      <section className="mb-8">
        <h2 className="mb-4 text-lg font-bold text-gray-900">Before You Delete, Consider:</h2>
        <ul className="space-y-2 text-sm text-gray-600">
          <li className="flex items-start gap-2"><CheckCircle size={16} className="mt-0.5 shrink-0 text-emerald-500" /> Deactivate your account temporarily instead</li>
          <li className="flex items-start gap-2"><CheckCircle size={16} className="mt-0.5 shrink-0 text-emerald-500" /> Download your data first (your right under data protection laws)</li>
          <li className="flex items-start gap-2"><CheckCircle size={16} className="mt-0.5 shrink-0 text-emerald-500" /> Resolve any open orders or disputes</li>
        </ul>
      </section>

      {/* What Gets Deleted */}
      <section className="mb-8">
        <h2 className="mb-4 text-lg font-bold text-gray-900">What Gets Deleted</h2>
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-4">
            <h3 className="mb-2 flex items-center gap-2 text-sm font-semibold text-emerald-800">
              <CheckCircle size={16} /> Deleted
            </h3>
            <ul className="space-y-1 text-xs text-emerald-700">
              {DELETED.map((d) => <li key={d}>✅ {d}</li>)}
            </ul>
          </div>
          <div className="rounded-xl border border-gray-200 bg-white p-4">
            <h3 className="mb-2 flex items-center gap-2 text-sm font-semibold text-gray-700">
              <XCircle size={16} /> Not Deleted
            </h3>
            <ul className="space-y-1 text-xs text-gray-600">
              {KEPT.map((k) => <li key={k}>❌ {k}</li>)}
            </ul>
          </div>
        </div>
      </section>

      {/* How to Delete */}
      <section className="mb-8">
        <h2 className="mb-4 text-lg font-bold text-gray-900">How to Delete</h2>
        <div className="space-y-3">
          <div className="rounded-xl border border-gray-200 bg-white p-4">
            <p className="text-sm text-gray-700">Go to <strong>Account Settings → Privacy → Delete Account → Confirm with password</strong></p>
          </div>
          <div className="rounded-xl border border-gray-200 bg-white p-4">
            <p className="text-sm text-gray-700">Or email: <strong>privacy@mnbarh.com</strong> with subject &quot;Account Deletion Request&quot;</p>
          </div>
        </div>
      </section>

      {/* Confirmation */}
      <section className="text-center">
        <label className="flex items-center justify-center gap-2 text-sm text-gray-600 mb-4 cursor-pointer">
          <input type="checkbox" checked={confirmed} onChange={(e) => setConfirmed(e.target.checked)} className="rounded" />
          I understand this action is permanent
        </label>
        <button
          disabled={!confirmed}
          className="inline-flex items-center gap-2 rounded-xl bg-red-600 px-6 py-3 text-sm font-semibold text-white hover:bg-red-700 disabled:opacity-40 disabled:cursor-not-allowed"
        >
          <AlertTriangle size={16} /> Delete My Account
        </button>
        <div className="mt-3 text-xs text-gray-500">
          <Link href="/help" className="text-[#0071CE] hover:underline">Need help?</Link> · <Link href="/buyer/settings" className="text-[#0071CE] hover:underline">Account Settings</Link>
        </div>
      </section>
    </div>
  );
}
