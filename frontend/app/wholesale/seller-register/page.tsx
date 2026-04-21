'use client'

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useMutation } from "@tanstack/react-query";
import api from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { Store, CheckCircle, ChevronLeft, Package, TrendingUp, Shield } from "lucide-react";
import { useTranslations } from "next-intl";

export default function WholesaleSellerRegisterPage() {
  const t = useTranslations("wholesale");
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();

  const [form, setForm] = useState({
    company_name: "",
    tax_id: "",
    business_type: "",
    categories: [] as string[],
    min_order_value_cents: 0,
  });

  const [categoryInput, setCategoryInput] = useState("");

  const register = useMutation({
    mutationFn: async () => {
      const res = await api.post("/wholesale/sellers", form);
      return res.data?.data;
    },
    onSuccess: () => {
      router.push("/wholesale?registered=1");
    },
  });

  if (!isAuthenticated) {
    return (
      <div className="max-w-lg mx-auto px-4 py-20 text-center">
        <Store size={48} className="text-gray-300 mx-auto mb-4" />
        <h1 className="text-xl font-bold text-gray-700 mb-2">Login Required</h1>
        <p className="text-sm text-gray-400 mb-4">You need to be logged in to register as a wholesale seller.</p>
        <button onClick={() => router.push("/login?next=/wholesale/seller-register")} className="bg-emerald-600 text-white font-bold px-6 py-3 rounded-xl hover:bg-emerald-700 transition-colors">
          Login / Register
        </button>
      </div>
    );
  }

  const addCategory = () => {
    const cat = categoryInput.trim();
    if (cat && !form.categories.includes(cat)) {
      setForm({ ...form, categories: [...form.categories, cat] });
      setCategoryInput("");
    }
  };

  const removeCategory = (cat: string) => {
    setForm({ ...form, categories: form.categories.filter(c => c !== cat) });
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Hero */}
      <div className="bg-gradient-to-r from-[#1E293B] to-[#334155] text-white">
        <div className="max-w-2xl mx-auto px-4 py-8">
          <button onClick={() => router.push("/wholesale")} className="flex items-center gap-1.5 text-slate-300 hover:text-white text-sm mb-4 transition-colors">
            <ChevronLeft size={16} /> Back to Wholesale
          </button>
          <div className="flex items-center gap-3 mb-2">
            <Store size={28} className="text-emerald-400" />
            <h1 className="text-2xl font-bold">{t("becomeSeller")}</h1>
          </div>
          <p className="text-slate-300 text-sm">
            List your products in bulk with tiered pricing. Reach thousands of B2B buyers across the region.
          </p>
          <div className="flex gap-6 mt-4">
            <div className="flex items-center gap-2 text-sm">
              <TrendingUp size={16} className="text-emerald-400" />
              <span>Tiered Pricing</span>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <Shield size={16} className="text-emerald-400" />
              <span>Escrow Protection</span>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <Package size={16} className="text-emerald-400" />
              <span>Bulk Orders</span>
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-2xl mx-auto px-4 py-8">
        <div className="bg-white rounded-2xl border border-gray-100 p-6 space-y-5">
          {/* Company Name */}
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-1.5">{t("companyName")} *</label>
            <input
              type="text"
              value={form.company_name}
              onChange={(e) => setForm({ ...form, company_name: e.target.value })}
              placeholder="Your company or business name"
              className="w-full border border-gray-200 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
            />
          </div>

          {/* Tax ID */}
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-1.5">{t("taxId")}</label>
            <input
              type="text"
              value={form.tax_id}
              onChange={(e) => setForm({ ...form, tax_id: e.target.value })}
              placeholder="e.g. 300-123-4567"
              className="w-full border border-gray-200 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
            />
          </div>

          {/* Business Type */}
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-1.5">{t("businessType")}</label>
            <select
              value={form.business_type}
              onChange={(e) => setForm({ ...form, business_type: e.target.value })}
              className="w-full border border-gray-200 rounded-xl px-4 py-2.5 text-sm text-gray-700 focus:outline-none focus:ring-2 focus:ring-emerald-500"
            >
              <option value="">Select type...</option>
              <option value="manufacturer">Manufacturer</option>
              <option value="distributor">Distributor</option>
              <option value="importer">Importer</option>
              <option value="wholesaler">Wholesaler</option>
              <option value="other">Other</option>
            </select>
          </div>

          {/* Categories */}
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-1.5">Product Categories</label>
            <div className="flex gap-2">
              <input
                type="text"
                value={categoryInput}
                onChange={(e) => setCategoryInput(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && (e.preventDefault(), addCategory())}
                placeholder="e.g. Electronics, Fashion"
                className="flex-1 border border-gray-200 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
              />
              <button onClick={addCategory} className="px-4 py-2.5 bg-emerald-50 text-emerald-700 rounded-xl text-sm font-medium hover:bg-emerald-100 transition-colors">
                Add
              </button>
            </div>
            {form.categories.length > 0 && (
              <div className="flex flex-wrap gap-2 mt-2">
                {form.categories.map((cat) => (
                  <span key={cat} className="bg-emerald-50 text-emerald-700 text-xs font-medium px-3 py-1 rounded-full flex items-center gap-1">
                    {cat}
                    <button onClick={() => removeCategory(cat)} className="text-emerald-400 hover:text-emerald-700 ml-1">×</button>
                  </span>
                ))}
              </div>
            )}
          </div>

          {/* Min Order Value */}
          <div>
            <label className="block text-sm font-semibold text-gray-700 mb-1.5">Minimum Order Value (EGP)</label>
            <input
              type="number"
              value={form.min_order_value_cents ? form.min_order_value_cents / 100 : ""}
              onChange={(e) => setForm({ ...form, min_order_value_cents: Math.round(parseFloat(e.target.value) * 100) || 0 })}
              placeholder="0 = no minimum"
              className="w-full border border-gray-200 rounded-xl px-4 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
            />
            <p className="text-xs text-gray-400 mt-1">Leave 0 for no minimum order value requirement.</p>
          </div>

          {/* Verification Notice */}
          <div className="bg-amber-50 border border-amber-200 rounded-xl p-4 flex items-start gap-3">
            <CheckCircle size={18} className="text-amber-600 mt-0.5 shrink-0" />
            <div>
              <p className="text-sm font-semibold text-amber-800">{t("verificationRequired")}</p>
              <p className="text-xs text-amber-700 mt-0.5">
                After registration, our team will review and verify your business. Verified sellers get a badge and higher visibility.
              </p>
            </div>
          </div>

          {/* Submit */}
          <button
            onClick={() => register.mutate()}
            disabled={!form.company_name || register.isPending}
            className="w-full bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 text-white font-bold py-3.5 rounded-xl transition-colors text-sm"
          >
            {register.isPending ? "Registering..." : t("registerSeller")}
          </button>

          {register.isError && (
            <p className="text-xs text-red-500 text-center">
              Registration failed. Please check your information and try again.
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
