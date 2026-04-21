'use client';

import { useState } from 'react';
import Link from 'next/link';
import api from '@/lib/api';

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await api.post('/auth/forgot-password', { email });
      setSent(true);
    } catch (err: any) {
      const msg = err?.response?.data?.message || err?.response?.data?.error;
      if (err?.response?.status === 429) {
        setError(msg || 'طلبات كثيرة جداً. انتظر قليلاً ثم حاول مجدداً.');
      } else {
        setError(msg || 'حدث خطأ. حاول مرة أخرى.');
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-[80vh] flex items-center justify-center px-4 bg-gray-50">
      <div className="bg-white rounded-2xl shadow-md p-8 w-full max-w-md">

        {/* Logo */}
        <div className="text-center mb-6">
          <h1 className="text-2xl font-extrabold text-[#0071CE]">mnbarh</h1>
          <p className="text-gray-500 mt-1 text-sm">استعادة كلمة المرور</p>
        </div>

        {sent ? (
          /* ── Success state ── */
          <div className="text-center space-y-4">
            <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto">
              <svg className="w-8 h-8 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            </div>
            <div>
              <h2 className="text-lg font-bold text-gray-800">تم إرسال الرابط!</h2>
              <p className="text-sm text-gray-500 mt-2 leading-relaxed">
                أرسلنا رابط إعادة تعيين كلمة المرور إلى
                <br />
                <span className="font-semibold text-gray-700 dir-ltr">{email}</span>
                <br />
                تحقق من بريدك الوارد (وصندوق الرسائل غير المرغوب فيها).
              </p>
            </div>
            <div className="pt-2 space-y-2">
              <button
                onClick={() => { setSent(false); setEmail(''); }}
                className="w-full border border-gray-200 text-gray-600 text-sm py-2.5 rounded-xl hover:bg-gray-50 transition-colors"
              >
                إرسال مرة أخرى بعنوان مختلف
              </button>
              <Link
                href="/login"
                className="block w-full text-center text-sm text-[#0071CE] font-medium py-2.5 hover:underline"
              >
                العودة لتسجيل الدخول
              </Link>
            </div>
          </div>
        ) : (
          /* ── Form state ── */
          <>
            <p className="text-sm text-gray-500 text-center mb-6 leading-relaxed">
              أدخل بريدك الإلكتروني وسنرسل لك رابطاً لإعادة تعيين كلمة المرور.
            </p>

            <form onSubmit={handleSubmit} className="space-y-4">
              {error && (
                <div className="bg-red-50 border border-red-200 text-red-600 rounded-lg px-4 py-3 text-sm text-center">
                  {error}
                </div>
              )}

              <div>
                <label className="text-sm font-medium text-gray-700 block mb-1.5">
                  البريد الإلكتروني
                </label>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  placeholder="you@example.com"
                  className="w-full border border-gray-200 rounded-lg px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE] focus:border-transparent"
                  dir="ltr"
                />
              </div>

              <button
                type="submit"
                disabled={loading}
                className="w-full bg-[#0071CE] hover:bg-[#005BA1] text-white font-bold py-3 rounded-xl transition-colors disabled:opacity-60"
              >
                {loading ? 'جارٍ الإرسال...' : 'إرسال رابط الاستعادة'}
              </button>
            </form>

            <p className="text-center text-sm text-gray-500 mt-5">
              تذكرت كلمة السر؟{' '}
              <Link href="/login" className="text-[#0071CE] font-semibold hover:underline">
                تسجيل الدخول
              </Link>
            </p>
          </>
        )}
      </div>
    </div>
  );
}
