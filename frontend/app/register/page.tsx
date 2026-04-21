'use client'
import Link from 'next/link';
import Image from 'next/image';
import { useRouter } from 'next/navigation';
import { useState } from "react";
import { useAuthStore } from "@/store/auth";
import { SocialAuthButtons } from "@/components/auth/SocialAuthButtons";
import { useTranslations } from "next-intl";

export default function RegisterPage() {
  const t = useTranslations("auth");
  const [form, setForm] = useState({ name: "", email: "", phone: "", password: "", confirm: "" });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const { register } = useAuthStore();
  const router = useRouter();

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setForm((f) => ({ ...f, [e.target.name]: e.target.value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (form.password !== form.confirm) {
      setError("Passwords do not match");
      return;
    }
    if (form.password.length < 8) {
      setError("Password must be at least 8 characters");
      return;
    }
    setLoading(true);
    try {
      await register(form.name, form.email, form.phone, form.password);
      router.push("/");
    } catch (err: any) {
      setError(err?.response?.data?.message || t("invalidCredentials"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-[80vh] flex items-center justify-center px-4 py-10 bg-gray-50">
      <div className="bg-white rounded-2xl shadow-md p-8 w-full max-w-md">
        <div className="text-center mb-6">
          <div className="flex justify-center mb-2">
            <Image src="/logo-mnbarh.svg" alt="Mnbarh" width={160} height={44} className="h-11 w-auto" priority />
          </div>
          <h1 className="sr-only">Mnbarh</h1>
          <p className="text-gray-500 mt-1 text-sm">{t("register")}</p>
        </div>

        {/* Social login */}
        <SocialAuthButtons redirectTo="/" />

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

          {[
            { name: "name",     labelKey: "email",    type: "text",     placeholder: "Ahmed Al-Rashidi" },
            { name: "email",    labelKey: "email",    type: "email",    placeholder: "you@example.com" },
            { name: "phone",    labelKey: "email",    type: "tel",      placeholder: "+971 50 000 0000" },
            { name: "password", labelKey: "password", type: "password", placeholder: "Min 8 characters" },
            { name: "confirm",  labelKey: "confirmPassword", type: "password", placeholder: "Repeat password" },
          ].map((f) => (
            <div key={f.name}>
              <label className="text-sm font-medium text-gray-700 block mb-1.5">{t(f.labelKey)}</label>
              <input
                type={f.type}
                name={f.name}
                value={form[f.name as keyof typeof form]}
                onChange={handleChange}
                required
                placeholder={f.placeholder}
                className="w-full border border-gray-200 rounded-lg px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE] focus:border-transparent"
              />
            </div>
          ))}

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-[#FFC220] hover:bg-yellow-400 text-gray-900 font-bold py-3 rounded-xl transition-colors disabled:opacity-60"
          >
            {loading ? t("register") + "..." : t("register")}
          </button>
        </form>

        <p className="text-center text-sm text-gray-500 mt-5">
          {t("alreadyHaveAccount")}{" "}
          <Link href="/login" className="text-[#0071CE] font-semibold hover:underline">
            {t("signIn")}
          </Link>
        </p>
      </div>
    </div>
  );
}
