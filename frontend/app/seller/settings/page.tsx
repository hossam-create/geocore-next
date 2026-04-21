"use client";

import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import {
  Store, Camera, Globe, Phone, Mail, MapPin,
  Save, CheckCircle, AlertCircle, Loader2, Eye,
} from "lucide-react";
import Link from "next/link";

interface StoreProfile {
  name: string;
  slug: string;
  description: string;
  logo_url: string;
  banner_url: string;
  website: string;
  phone: string;
  email: string;
  city: string;
  country: string;
  is_active: boolean;
}

const DEFAULT: StoreProfile = {
  name: "",
  slug: "",
  description: "",
  logo_url: "",
  banner_url: "",
  website: "",
  phone: "",
  email: "",
  city: "",
  country: "UAE",
  is_active: true,
};

function Field({
  label, value, onChange, placeholder, type = "text", icon: Icon, hint,
}: {
  label: string; value: string; onChange: (v: string) => void;
  placeholder?: string; type?: string;
  icon?: React.ElementType; hint?: string;
}) {
  return (
    <div>
      <label className="block text-xs font-semibold text-gray-600 mb-1.5">{label}</label>
      <div className="relative">
        {Icon && (
          <Icon className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        )}
        <input
          type={type}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          className={`w-full text-sm border border-gray-200 rounded-xl py-2.5 outline-none focus:ring-2 focus:ring-[#0071CE]/20 focus:border-[#0071CE] transition-all ${Icon ? "pl-9 pr-3" : "px-3"}`}
        />
      </div>
      {hint && <p className="text-[11px] text-gray-400 mt-1">{hint}</p>}
    </div>
  );
}

function TextArea({
  label, value, onChange, placeholder, rows = 3, hint,
}: {
  label: string; value: string; onChange: (v: string) => void;
  placeholder?: string; rows?: number; hint?: string;
}) {
  return (
    <div>
      <label className="block text-xs font-semibold text-gray-600 mb-1.5">{label}</label>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        rows={rows}
        className="w-full text-sm border border-gray-200 rounded-xl px-3 py-2.5 outline-none focus:ring-2 focus:ring-[#0071CE]/20 focus:border-[#0071CE] transition-all resize-none"
      />
      {hint && <p className="text-[11px] text-gray-400 mt-1">{hint}</p>}
    </div>
  );
}

export default function SellerSettingsPage() {
  const qc = useQueryClient();
  const { isAuthenticated } = useAuthStore();
  const [form, setForm] = useState<StoreProfile>(DEFAULT);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");

  const set = (key: keyof StoreProfile) => (value: string | boolean) =>
    setForm((prev) => ({ ...prev, [key]: value }));

  const { isLoading, data: profileData } = useQuery<StoreProfile>({
    queryKey: ["seller-store-profile"],
    queryFn: async () => {
      const res = await api.get("/stores/me");
      return res.data?.data ?? res.data ?? DEFAULT;
    },
    enabled: isAuthenticated,
    retry: false,
  });

  useEffect(() => {
    if (profileData) setForm({ ...DEFAULT, ...profileData });
  }, [profileData]);

  const saveMutation = useMutation({
    mutationFn: (payload: Partial<StoreProfile>) => api.put("/stores/me", payload),
    onSuccess: () => {
      setSaved(true);
      setError("");
      qc.invalidateQueries({ queryKey: ["seller-store-profile"] });
      setTimeout(() => setSaved(false), 3000);
    },
    onError: (err: { response?: { data?: { error?: string } } }) => {
      setError(err?.response?.data?.error ?? "Failed to save settings.");
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.name.trim()) { setError("Store name is required."); return; }
    setError("");
    saveMutation.mutate(form);
  };

  return (
    <div className="max-w-2xl space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-gray-900">Store Settings</h1>
          <p className="text-sm text-gray-400">Manage your public store profile and contact info</p>
        </div>
        {form.slug && (
          <Link
            href={`/stores/${form.slug}`}
            className="flex items-center gap-1.5 text-xs font-medium text-[#0071CE] hover:underline"
          >
            <Eye className="w-3.5 h-3.5" /> View Store
          </Link>
        )}
      </div>

      {/* Status messages */}
      {saved && (
        <div className="flex items-center gap-2 bg-emerald-50 border border-emerald-200 rounded-xl px-4 py-3 text-sm text-emerald-700">
          <CheckCircle className="w-4 h-4 shrink-0" />
          Settings saved successfully.
        </div>
      )}
      {error && (
        <div className="flex items-center gap-2 bg-red-50 border border-red-200 rounded-xl px-4 py-3 text-sm text-red-700">
          <AlertCircle className="w-4 h-4 shrink-0" />
          {error}
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center py-16">
          <Loader2 className="w-7 h-7 animate-spin text-[#0071CE]" />
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Store Identity */}
          <div className="bg-white rounded-2xl border border-gray-100 p-5 space-y-4">
            <div className="flex items-center gap-2 mb-1">
              <Store className="w-4 h-4 text-[#0071CE]" />
              <h2 className="text-sm font-bold text-gray-800">Store Identity</h2>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <Field
                label="Store Name *"
                value={form.name}
                onChange={set("name")}
                placeholder="My Awesome Store"
                icon={Store}
              />
              <Field
                label="URL Slug"
                value={form.slug}
                onChange={set("slug")}
                placeholder="my-awesome-store"
                hint="mnbarh.com/stores/my-awesome-store"
              />
            </div>

            <TextArea
              label="Store Description"
              value={form.description}
              onChange={set("description") as (v: string) => void}
              placeholder="Tell buyers what you sell and what makes your store special…"
              rows={3}
              hint="Max 500 characters. Shown on your store page."
            />
          </div>

          {/* Media */}
          <div className="bg-white rounded-2xl border border-gray-100 p-5 space-y-4">
            <div className="flex items-center gap-2 mb-1">
              <Camera className="w-4 h-4 text-[#0071CE]" />
              <h2 className="text-sm font-bold text-gray-800">Media</h2>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <Field
                label="Logo URL"
                value={form.logo_url}
                onChange={set("logo_url")}
                placeholder="https://..."
                hint="Square image, 256×256px recommended"
              />
              <Field
                label="Banner URL"
                value={form.banner_url}
                onChange={set("banner_url")}
                placeholder="https://..."
                hint="Wide image, 1200×300px recommended"
              />
            </div>

            {/* Preview */}
            {(form.logo_url || form.banner_url) && (
              <div className="rounded-xl overflow-hidden border border-gray-100">
                {form.banner_url && (
                  <div className="h-24 bg-gray-100 overflow-hidden">
                    <img src={form.banner_url} alt="Banner preview" className="w-full h-full object-cover" onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }} />
                  </div>
                )}
                <div className="flex items-center gap-3 p-3">
                  {form.logo_url ? (
                    <img src={form.logo_url} alt="Logo" className="w-10 h-10 rounded-full object-cover border border-gray-200" onError={(e) => { (e.target as HTMLImageElement).src = ""; }} />
                  ) : (
                    <div className="w-10 h-10 rounded-full bg-[#0071CE] flex items-center justify-center text-white font-bold text-sm">
                      {form.name.charAt(0) || "S"}
                    </div>
                  )}
                  <div>
                    <p className="text-sm font-semibold text-gray-900">{form.name || "Store Name"}</p>
                    <p className="text-xs text-gray-400">Preview</p>
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Contact Info */}
          <div className="bg-white rounded-2xl border border-gray-100 p-5 space-y-4">
            <div className="flex items-center gap-2 mb-1">
              <Globe className="w-4 h-4 text-[#0071CE]" />
              <h2 className="text-sm font-bold text-gray-800">Contact Information</h2>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <Field label="Website" value={form.website} onChange={set("website")} placeholder="https://mystore.com" icon={Globe} />
              <Field label="Phone" value={form.phone} onChange={set("phone")} placeholder="+971 50 000 0000" icon={Phone} type="tel" />
              <Field label="Contact Email" value={form.email} onChange={set("email")} placeholder="store@example.com" icon={Mail} type="email" />
              <Field label="City" value={form.city} onChange={set("city")} placeholder="Dubai" icon={MapPin} />
            </div>

            <div>
              <label className="block text-xs font-semibold text-gray-600 mb-1.5">Country</label>
              <select
                value={form.country}
                onChange={(e) => set("country")(e.target.value)}
                className="w-full text-sm border border-gray-200 rounded-xl px-3 py-2.5 outline-none focus:ring-2 focus:ring-[#0071CE]/20 focus:border-[#0071CE] bg-white"
              >
                {["UAE", "Saudi Arabia", "Kuwait", "Qatar", "Bahrain", "Oman", "Egypt", "Jordan", "Lebanon", "Other"].map((c) => (
                  <option key={c} value={c}>{c}</option>
                ))}
              </select>
            </div>
          </div>

          {/* Store visibility */}
          <div className="bg-white rounded-2xl border border-gray-100 p-5">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-semibold text-gray-800">Store Visibility</p>
                <p className="text-xs text-gray-400 mt-0.5">When active, your store is visible to buyers</p>
              </div>
              <button
                type="button"
                onClick={() => set("is_active")(!form.is_active)}
                className={`relative w-11 h-6 rounded-full transition-colors ${form.is_active ? "bg-[#0071CE]" : "bg-gray-200"}`}
              >
                <span className={`absolute top-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform ${form.is_active ? "translate-x-5" : "translate-x-0.5"}`} />
              </button>
            </div>
          </div>

          {/* Submit */}
          <div className="flex items-center justify-end gap-3 pb-4">
            <button
              type="submit"
              disabled={saveMutation.isPending}
              className="flex items-center gap-2 px-6 py-2.5 bg-[#0071CE] text-white rounded-xl text-sm font-semibold hover:bg-[#005ba3] transition-colors disabled:opacity-60"
            >
              {saveMutation.isPending ? (
                <><Loader2 className="w-4 h-4 animate-spin" /> Saving…</>
              ) : (
                <><Save className="w-4 h-4" /> Save Settings</>
              )}
            </button>
          </div>
        </form>
      )}
    </div>
  );
}
