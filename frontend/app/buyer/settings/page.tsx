"use client";

import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import {
  MapPin, Bell, Globe, DollarSign,
  Save, CheckCircle, AlertCircle, Loader2, Plus, Trash2,
} from "lucide-react";

interface Address {
  id: string;
  label: string;
  full_name: string;
  phone: string;
  line1: string;
  line2?: string;
  city: string;
  country: string;
  is_default: boolean;
}

interface NotifPrefs {
  order_updates: boolean;
  price_drops: boolean;
  auction_ending: boolean;
  new_messages: boolean;
  promotions: boolean;
  dispute_updates: boolean;
}

interface BuyerPrefs {
  currency: string;
  language: string;
  notifications: NotifPrefs;
}

const DEFAULT_PREFS: BuyerPrefs = {
  currency: "AED",
  language: "en",
  notifications: {
    order_updates:   true,
    price_drops:     true,
    auction_ending:  true,
    new_messages:    true,
    promotions:      false,
    dispute_updates: true,
  },
};

const MOCK_ADDRESSES: Address[] = [
  { id: "addr-001", label: "Home", full_name: "Ahmed Al-Rashid", phone: "+971501234567", line1: "Villa 14, Street 5B", city: "Dubai", country: "UAE", is_default: true },
  { id: "addr-002", label: "Office", full_name: "Ahmed Al-Rashid", phone: "+971501234567", line1: "Floor 12, Business Bay Tower", city: "Dubai", country: "UAE", is_default: false },
];

const CURRENCIES = ["AED", "SAR", "KWD", "QAR", "BHD", "OMR", "EGP", "USD", "EUR", "GBP"];
const LANGUAGES  = [{ value: "en", label: "English" }, { value: "ar", label: "العربية" }];

function Toggle({ value, onChange, label, sub }: { value: boolean; onChange: (v: boolean) => void; label: string; sub?: string }) {
  return (
    <div className="flex items-center justify-between py-3 border-b border-gray-50 last:border-0">
      <div>
        <p className="text-sm font-medium text-gray-800">{label}</p>
        {sub && <p className="text-xs text-gray-400 mt-0.5">{sub}</p>}
      </div>
      <button
        type="button"
        onClick={() => onChange(!value)}
        className={`relative w-10 h-5 rounded-full transition-colors shrink-0 ${value ? "bg-indigo-600" : "bg-gray-200"}`}
      >
        <span className={`absolute top-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${value ? "translate-x-5" : "translate-x-0.5"}`} />
      </button>
    </div>
  );
}

function AddressCard({ addr, onDelete, onSetDefault }: {
  addr: Address;
  onDelete: (id: string) => void;
  onSetDefault: (id: string) => void;
}) {
  return (
    <div className={`rounded-xl border p-4 relative ${addr.is_default ? "border-indigo-300 bg-indigo-50/40" : "border-gray-200 bg-white"}`}>
      {addr.is_default && (
        <span className="absolute top-3 right-3 text-[10px] font-bold text-indigo-600 bg-indigo-100 px-2 py-0.5 rounded-full">Default</span>
      )}
      <p className="text-sm font-bold text-gray-800 mb-1">{addr.label}</p>
      <p className="text-sm text-gray-700">{addr.full_name}</p>
      <p className="text-xs text-gray-500 mt-0.5">{addr.line1}{addr.line2 ? `, ${addr.line2}` : ""}</p>
      <p className="text-xs text-gray-500">{addr.city}, {addr.country}</p>
      <p className="text-xs text-gray-500">{addr.phone}</p>
      <div className="flex items-center gap-3 mt-3">
        {!addr.is_default && (
          <button onClick={() => onSetDefault(addr.id)} className="text-xs text-indigo-600 font-medium hover:underline">
            Set as default
          </button>
        )}
        <button onClick={() => onDelete(addr.id)} className="text-xs text-red-500 font-medium hover:underline flex items-center gap-1">
          <Trash2 className="w-3 h-3" /> Remove
        </button>
      </div>
    </div>
  );
}

export default function BuyerSettingsPage() {
  const qc = useQueryClient();
  const { isAuthenticated } = useAuthStore();
  const [prefs, setPrefs] = useState<BuyerPrefs>(DEFAULT_PREFS);
  const [addresses, setAddresses] = useState<Address[]>(MOCK_ADDRESSES);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");
  const [showAddForm, setShowAddForm] = useState(false);
  const [newAddr, setNewAddr] = useState({ label: "", full_name: "", phone: "", line1: "", line2: "", city: "", country: "UAE" });

  const { data: prefsData } = useQuery<BuyerPrefs>({
    queryKey: ["buyer-prefs"],
    queryFn: async () => { const r = await api.get("/users/me/preferences"); return r.data?.data ?? DEFAULT_PREFS; },
    enabled: isAuthenticated,
    retry: false,
  });

  useEffect(() => { if (prefsData) setPrefs({ ...DEFAULT_PREFS, ...prefsData }); }, [prefsData]);

  const { data: addrData } = useQuery<Address[]>({
    queryKey: ["buyer-addresses"],
    queryFn: async () => { const r = await api.get("/users/me/addresses"); const d = r.data?.data ?? []; return d.length ? d : MOCK_ADDRESSES; },
    enabled: isAuthenticated,
    retry: false,
  });

  useEffect(() => { if (addrData?.length) setAddresses(addrData); }, [addrData]);

  const saveMutation = useMutation({
    mutationFn: () => api.put("/users/me/preferences", prefs),
    onSuccess: () => { setSaved(true); setError(""); qc.invalidateQueries({ queryKey: ["buyer-prefs"] }); setTimeout(() => setSaved(false), 3000); },
    onError: () => setError("Failed to save preferences."),
  });

  const setNotif = (key: keyof NotifPrefs) => (v: boolean) =>
    setPrefs((p) => ({ ...p, notifications: { ...p.notifications, [key]: v } }));

  const handleAddAddress = () => {
    if (!newAddr.label || !newAddr.line1 || !newAddr.city) return;
    const a: Address = { id: `addr-${Date.now()}`, ...newAddr, is_default: addresses.length === 0 };
    setAddresses((prev) => [...prev, a]);
    setNewAddr({ label: "", full_name: "", phone: "", line1: "", line2: "", city: "", country: "UAE" });
    setShowAddForm(false);
    try { api.post("/users/me/addresses", a); } catch {}
  };

  const handleDelete = (id: string) => {
    setAddresses((prev) => prev.filter((a) => a.id !== id));
    try { api.delete(`/users/me/addresses/${id}`); } catch {}
  };

  const handleSetDefault = (id: string) => {
    setAddresses((prev) => prev.map((a) => ({ ...a, is_default: a.id === id })));
    try { api.put(`/users/me/addresses/${id}/default`, {}); } catch {}
  };

  return (
    <div className="max-w-2xl space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-xl font-bold text-gray-900">Account Settings</h1>
        <p className="text-sm text-gray-400">Manage your shipping addresses and notification preferences</p>
      </div>

      {saved && (
        <div className="flex items-center gap-2 bg-emerald-50 border border-emerald-200 rounded-xl px-4 py-3 text-sm text-emerald-700">
          <CheckCircle className="w-4 h-4" /> Settings saved successfully.
        </div>
      )}
      {error && (
        <div className="flex items-center gap-2 bg-red-50 border border-red-200 rounded-xl px-4 py-3 text-sm text-red-700">
          <AlertCircle className="w-4 h-4" /> {error}
        </div>
      )}

      {/* Shipping Addresses */}
      <div className="bg-white rounded-2xl border border-gray-100 p-5 space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <MapPin className="w-4 h-4 text-indigo-600" />
            <h2 className="text-sm font-bold text-gray-800">Shipping Addresses</h2>
          </div>
          <button
            onClick={() => setShowAddForm((v) => !v)}
            className="flex items-center gap-1 text-xs font-semibold text-indigo-600 hover:underline"
          >
            <Plus className="w-3.5 h-3.5" /> Add Address
          </button>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          {addresses.map((a) => (
            <AddressCard key={a.id} addr={a} onDelete={handleDelete} onSetDefault={handleSetDefault} />
          ))}
        </div>

        {showAddForm && (
          <div className="rounded-xl border border-gray-200 p-4 space-y-3 bg-gray-50">
            <p className="text-xs font-bold text-gray-700 mb-1">New Address</p>
            <div className="grid grid-cols-2 gap-3">
              {[
                { key: "label",     placeholder: "Label (Home, Office...)" },
                { key: "full_name", placeholder: "Full name" },
                { key: "phone",     placeholder: "Phone number" },
                { key: "line1",     placeholder: "Address line 1" },
                { key: "line2",     placeholder: "Address line 2 (optional)" },
                { key: "city",      placeholder: "City" },
              ].map(({ key, placeholder }) => (
                <input
                  key={key}
                  value={newAddr[key as keyof typeof newAddr]}
                  onChange={(e) => setNewAddr((p) => ({ ...p, [key]: e.target.value }))}
                  placeholder={placeholder}
                  className="col-span-2 sm:col-span-1 text-sm border border-gray-200 rounded-lg px-3 py-2 outline-none focus:ring-2 focus:ring-indigo-300"
                />
              ))}
              <select
                value={newAddr.country}
                onChange={(e) => setNewAddr((p) => ({ ...p, country: e.target.value }))}
                className="col-span-2 sm:col-span-1 text-sm border border-gray-200 rounded-lg px-3 py-2 bg-white outline-none focus:ring-2 focus:ring-indigo-300"
              >
                {["UAE", "Saudi Arabia", "Kuwait", "Qatar", "Bahrain", "Oman", "Egypt", "Other"].map((c) => (
                  <option key={c} value={c}>{c}</option>
                ))}
              </select>
            </div>
            <div className="flex gap-2 pt-1">
              <button onClick={handleAddAddress} className="px-4 py-2 bg-indigo-600 text-white text-xs font-semibold rounded-lg hover:bg-indigo-700 transition-colors">
                Save Address
              </button>
              <button onClick={() => setShowAddForm(false)} className="px-4 py-2 border border-gray-200 text-xs font-semibold rounded-lg hover:bg-gray-50">
                Cancel
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Notifications */}
      <div className="bg-white rounded-2xl border border-gray-100 p-5">
        <div className="flex items-center gap-2 mb-4">
          <Bell className="w-4 h-4 text-indigo-600" />
          <h2 className="text-sm font-bold text-gray-800">Notification Preferences</h2>
        </div>
        <Toggle value={prefs.notifications.order_updates}  onChange={setNotif("order_updates")}  label="Order Updates"     sub="Confirmations, shipping, and delivery" />
        <Toggle value={prefs.notifications.price_drops}    onChange={setNotif("price_drops")}    label="Price Drops"       sub="When watchlisted items drop in price" />
        <Toggle value={prefs.notifications.auction_ending} onChange={setNotif("auction_ending")} label="Auction Ending"    sub="Alerts 1h before auctions you&apos;re watching end" />
        <Toggle value={prefs.notifications.new_messages}   onChange={setNotif("new_messages")}   label="New Messages"      sub="Direct messages from sellers" />
        <Toggle value={prefs.notifications.dispute_updates}onChange={setNotif("dispute_updates")}label="Dispute Updates"   sub="Status changes on your open disputes" />
        <Toggle value={prefs.notifications.promotions}     onChange={setNotif("promotions")}     label="Promotions"        sub="Deals, offers and platform news" />
      </div>

      {/* Preferences */}
      <div className="bg-white rounded-2xl border border-gray-100 p-5 space-y-4">
        <div className="flex items-center gap-2 mb-1">
          <Globe className="w-4 h-4 text-indigo-600" />
          <h2 className="text-sm font-bold text-gray-800">Display Preferences</h2>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs font-semibold text-gray-600 mb-1.5">
              <DollarSign className="w-3.5 h-3.5 inline mr-1" />Currency
            </label>
            <select
              value={prefs.currency}
              onChange={(e) => setPrefs((p) => ({ ...p, currency: e.target.value }))}
              className="w-full text-sm border border-gray-200 rounded-xl px-3 py-2.5 bg-white outline-none focus:ring-2 focus:ring-indigo-300"
            >
              {CURRENCIES.map((c) => <option key={c} value={c}>{c}</option>)}
            </select>
          </div>
          <div>
            <label className="block text-xs font-semibold text-gray-600 mb-1.5">
              <Globe className="w-3.5 h-3.5 inline mr-1" />Language
            </label>
            <select
              value={prefs.language}
              onChange={(e) => setPrefs((p) => ({ ...p, language: e.target.value }))}
              className="w-full text-sm border border-gray-200 rounded-xl px-3 py-2.5 bg-white outline-none focus:ring-2 focus:ring-indigo-300"
            >
              {LANGUAGES.map((l) => <option key={l.value} value={l.value}>{l.label}</option>)}
            </select>
          </div>
        </div>
      </div>

      {/* Save */}
      <div className="flex justify-end pb-4">
        <button
          onClick={() => saveMutation.mutate()}
          disabled={saveMutation.isPending}
          className="flex items-center gap-2 px-6 py-2.5 bg-indigo-600 text-white rounded-xl text-sm font-semibold hover:bg-indigo-700 transition-colors disabled:opacity-60"
        >
          {saveMutation.isPending ? <><Loader2 className="w-4 h-4 animate-spin" /> Saving…</> : <><Save className="w-4 h-4" /> Save Settings</>}
        </button>
      </div>
    </div>
  );
}
