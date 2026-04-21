'use client'
import { Suspense } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { useRouter, useSearchParams } from 'next/navigation';
import { useState } from "react";
import { useAuthStore } from "@/store/auth";
import { SocialAuthButtons } from "@/components/auth/SocialAuthButtons";
import { useTranslations } from "next-intl";

function LoginContent() {
  const t = useTranslations("auth");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const { login } = useAuthStore();
  const router = useRouter();
  const search = useSearchParams();
  const next = search.get("next") || "/";

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await login(email, password);
      router.push(next);
    } catch (err: any) {
      setError(err?.response?.data?.message || t("invalidCredentials"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-[80vh] flex items-center justify-center px-4 bg-gray-50">
      <div className="bg-white rounded-2xl shadow-md p-8 w-full max-w-md">
        <div className="text-center mb-6">
          <div className="flex justify-center mb-2">
            <Image src="/logo-mnbarh.svg" alt="Mnbarh" width={160} height={44} className="h-11 w-auto" priority />
          </div>
          <h1 className="sr-only">Mnbarh</h1>
          <p className="text-gray-500 mt-1 text-sm">{t("login")}</p>
        </div>

        {/* Social login */}
        <SocialAuthButtons redirectTo={next} />

        <div className="flex items-center gap-3 my-5">
          <div className="flex-1 h-px bg-gray-200" />
          <span className="text-xs text-gray-400 font-medium">{t("orContinueWith")}</span>
          <div className="flex-1 h-px bg-gray-200" />
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="bg-red-50 border border-red-200 text-red-600 rounded-lg px-4 py-3 text-sm text-center">
              {error}
            </div>
          )}
          <div>
            <label className="text-sm font-medium text-gray-700 block mb-1.5">{t("email")}</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              placeholder="you@example.com"
              className="w-full border border-gray-200 rounded-lg px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE] focus:border-transparent"
            />
          </div>
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className="text-sm font-medium text-gray-700">{t("password")}</label>
              <Link href="/forgot-password" className="text-xs text-[#0071CE] hover:underline">
                {t("forgotPassword")}
              </Link>
            </div>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              placeholder="••••••••"
              className="w-full border border-gray-200 rounded-lg px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE] focus:border-transparent"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-[#0071CE] hover:bg-[#005BA1] text-white font-bold py-3 rounded-xl transition-colors disabled:opacity-60"
          >
            {loading ? t("login") + "..." : t("login")}
          </button>
        </form>

        <p className="text-center text-sm text-gray-500 mt-5">
          {t("noAccount")}{" "}
          <Link href="/register" className="text-[#0071CE] font-semibold hover:underline">
            {t("signUp")}
          </Link>
        </p>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={null}>
      <LoginContent />
    </Suspense>
  );
}
