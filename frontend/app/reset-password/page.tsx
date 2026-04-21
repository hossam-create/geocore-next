'use client';

import { Suspense, useState, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import api from '@/lib/api';

function ResetPasswordContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get('token') || '';

  const [form, setForm] = useState({ password: '', confirm: '' });
  const [loading, setLoading] = useState(false);
  const [validating, setValidating] = useState(true);
  const [tokenValid, setTokenValid] = useState(false);
  const [maskedEmail, setMaskedEmail] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(false);

  useEffect(() => {
    if (!token) {
      setValidating(false);
      return;
    }
    api.post('/auth/validate-reset-token', { token })
      .then((res) => {
        setTokenValid(true);
        setMaskedEmail(res.data?.data?.email || '');
      })
      .catch(() => {
        setTokenValid(false);
      })
      .finally(() => setValidating(false));
  }, [token]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (form.password !== form.confirm) {
      setError('كلمتا المرور غير متطابقتين.');
      return;
    }
    if (form.password.length < 8) {
      setError('كلمة المرور يجب أن تكون 8 حروف على الأقل.');
      return;
    }
    setLoading(true);
    try {
      await api.post('/auth/reset-password', {
        token,
        new_password: form.password,
        confirm_password: form.confirm,
      });
      setSuccess(true);
      setTimeout(() => router.push('/login'), 3000);
    } catch (err: any) {
      setError(
        err?.response?.data?.message ||
        err?.response?.data?.error ||
        'فشل تغيير كلمة المرور. الرابط قد يكون منتهي الصلاحية.'
      );
    } finally {
      setLoading(false);
    }
  };

  /* ── Loading state while validating token ── */
  if (validating) {
    return (
      <div className="min-h-[80vh] flex items-center justify-center">
        <div className="w-8 h-8 border-2 border-gray-200 border-t-[#0071CE] rounded-full animate-spin" />
      </div>
    );
  }

  /* ── Invalid / missing token ── */
  if (!token || !tokenValid) {
    return (
      <div className="min-h-[80vh] flex items-center justify-center px-4 bg-gray-50">
        <div className="bg-white rounded-2xl shadow-md p-8 w-full max-w-md text-center space-y-4">
          <div className="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mx-auto">
            <svg className="w-8 h-8 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </div>
          <h2 className="text-lg font-bold text-gray-800">الرابط غير صالح أو منتهي</h2>
          <p className="text-sm text-gray-500 leading-relaxed">
            رابط إعادة تعيين كلمة المرور منتهي الصلاحية أو غير صحيح.
            <br />يمكنك طلب رابط جديد.
          </p>
          <Link
            href="/forgot-password"
            className="block w-full bg-[#0071CE] hover:bg-[#005BA1] text-white font-bold py-3 rounded-xl transition-colors text-sm"
          >
            طلب رابط جديد
          </Link>
        </div>
      </div>
    );
  }

  /* ── Success state ── */
  if (success) {
    return (
      <div className="min-h-[80vh] flex items-center justify-center px-4 bg-gray-50">
        <div className="bg-white rounded-2xl shadow-md p-8 w-full max-w-md text-center space-y-4">
          <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto">
            <svg className="w-8 h-8 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <h2 className="text-lg font-bold text-gray-800">تم تغيير كلمة المرور بنجاح!</h2>
          <p className="text-sm text-gray-500">سيتم توجيهك لصفحة تسجيل الدخول خلال ثوانٍ...</p>
          <Link
            href="/login"
            className="block w-full bg-[#0071CE] hover:bg-[#005BA1] text-white font-bold py-3 rounded-xl transition-colors text-sm"
          >
            تسجيل الدخول الآن
          </Link>
        </div>
      </div>
    );
  }

  /* ── Main form ── */
  return (
    <div className="min-h-[80vh] flex items-center justify-center px-4 bg-gray-50">
      <div className="bg-white rounded-2xl shadow-md p-8 w-full max-w-md">
        <div className="text-center mb-6">
          <h1 className="text-2xl font-extrabold text-[#0071CE]">mnbarh</h1>
          <p className="text-gray-500 mt-1 text-sm">تعيين كلمة مرور جديدة</p>
          {maskedEmail && (
            <p className="text-xs text-gray-400 mt-1 dir-ltr">{maskedEmail}</p>
          )}
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="bg-red-50 border border-red-200 text-red-600 rounded-lg px-4 py-3 text-sm text-center">
              {error}
            </div>
          )}

          <div>
            <label className="text-sm font-medium text-gray-700 block mb-1.5">
              كلمة المرور الجديدة
            </label>
            <input
              type="password"
              value={form.password}
              onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
              required
              placeholder="أدنى 8 حروف، تضمّن رقم وحرف كبير"
              className="w-full border border-gray-200 rounded-lg px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE] focus:border-transparent"
            />
          </div>

          <div>
            <label className="text-sm font-medium text-gray-700 block mb-1.5">
              تأكيد كلمة المرور
            </label>
            <input
              type="password"
              value={form.confirm}
              onChange={(e) => setForm((f) => ({ ...f, confirm: e.target.value }))}
              required
              placeholder="أعد كتابة كلمة المرور"
              className="w-full border border-gray-200 rounded-lg px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE] focus:border-transparent"
            />
          </div>

          {/* Password strength hint */}
          <p className="text-xs text-gray-400 leading-relaxed">
            يجب أن تحتوي كلمة المرور على: 8 أحرف على الأقل، حرف كبير، حرف صغير، ورقم.
          </p>

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-[#0071CE] hover:bg-[#005BA1] text-white font-bold py-3 rounded-xl transition-colors disabled:opacity-60"
          >
            {loading ? 'جارٍ الحفظ...' : 'حفظ كلمة المرور الجديدة'}
          </button>
        </form>
      </div>
    </div>
  );
}

export default function ResetPasswordPage() {
  return (
    <Suspense fallback={
      <div className="min-h-[80vh] flex items-center justify-center">
        <div className="w-8 h-8 border-2 border-gray-200 border-t-[#0071CE] rounded-full animate-spin" />
      </div>
    }>
      <ResetPasswordContent />
    </Suspense>
  );
}
