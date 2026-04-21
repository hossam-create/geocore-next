"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import axios from "axios";
import Link from "next/link";
import { ArrowLeft, ArrowLeftRight } from "lucide-react";

const CURRENCIES = ["USD", "AED", "SAR", "EGP", "EUR", "GBP"];

export default function NewExchangeRequestPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    from_currency: "",
    to_currency: "",
    from_amount: "",
    to_amount: "",
    use_escrow: false,
    notes: "",
  });

  const [rate, setRate] = useState(0);

  const set = (k: string, v: string | boolean) => setForm((p) => ({ ...p, [k]: v }));

  useEffect(() => {
    const fa = parseFloat(form.from_amount);
    const ta = parseFloat(form.to_amount);
    if (fa > 0 && ta > 0) setRate(ta / fa);
    else setRate(0);
  }, [form.from_amount, form.to_amount]);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (!form.from_currency || !form.to_currency) { setError("Select both currencies"); return; }
    if (form.from_currency === form.to_currency) { setError("Currencies must be different"); return; }
    if (!form.from_amount || !form.to_amount) { setError("Enter both amounts"); return; }

    setLoading(true);
    try {
      await axios.post("/api/v1/p2p/requests", {
        from_currency: form.from_currency,
        to_currency: form.to_currency,
        from_amount: parseFloat(form.from_amount),
        to_amount: parseFloat(form.to_amount),
        use_escrow: form.use_escrow,
        notes: form.notes,
      });
      router.push("/p2p");
    } catch (err: any) {
      setError(err?.response?.data?.message || "Failed to create request");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-lg mx-auto px-4 py-8">
      <Link href="/p2p" className="flex items-center gap-2 text-gray-500 hover:text-gray-700 mb-6 text-sm">
        <ArrowLeft className="w-4 h-4" /> Back to Marketplace
      </Link>

      <h1 className="text-2xl font-bold mb-1">Create Exchange Request</h1>
      <p className="text-gray-500 text-sm mb-6">Set your currency pair, amounts, and rate</p>

      {error && <div className="bg-red-50 text-red-600 border border-red-200 rounded-lg p-3 mb-4 text-sm">{error}</div>}

      <form onSubmit={submit} className="space-y-5">
        {/* From */}
        <div className="bg-white rounded-xl border p-5 space-y-3">
          <h2 className="font-semibold">You Send</h2>
          <div className="grid grid-cols-2 gap-3">
            <select className="border rounded-lg px-3 py-2 text-sm" value={form.from_currency} onChange={(e) => set("from_currency", e.target.value)}>
              <option value="">Currency</option>
              {CURRENCIES.map((c) => <option key={c} value={c}>{c}</option>)}
            </select>
            <input type="number" step="0.01" placeholder="Amount" className="border rounded-lg px-3 py-2 text-sm" value={form.from_amount} onChange={(e) => set("from_amount", e.target.value)} />
          </div>
        </div>

        <div className="flex justify-center"><ArrowLeftRight className="w-5 h-5 text-gray-400" /></div>

        {/* To */}
        <div className="bg-white rounded-xl border p-5 space-y-3">
          <h2 className="font-semibold">You Receive</h2>
          <div className="grid grid-cols-2 gap-3">
            <select className="border rounded-lg px-3 py-2 text-sm" value={form.to_currency} onChange={(e) => set("to_currency", e.target.value)}>
              <option value="">Currency</option>
              {CURRENCIES.map((c) => <option key={c} value={c}>{c}</option>)}
            </select>
            <input type="number" step="0.01" placeholder="Amount" className="border rounded-lg px-3 py-2 text-sm" value={form.to_amount} onChange={(e) => set("to_amount", e.target.value)} />
          </div>
        </div>

        {/* Rate display */}
        {rate > 0 && (
          <div className="text-center text-sm text-gray-600 bg-gray-50 rounded-lg py-2">
            Exchange Rate: <strong>1 {form.from_currency || "?"} = {rate.toFixed(4)} {form.to_currency || "?"}</strong>
          </div>
        )}

        {/* Escrow */}
        <label className="flex items-center gap-2 bg-white border rounded-xl p-4 cursor-pointer">
          <input type="checkbox" className="h-4 w-4 text-blue-600 rounded" checked={form.use_escrow} onChange={(e) => set("use_escrow", e.target.checked)} />
          <div>
            <span className="text-sm font-medium">Use Escrow Protection</span>
            <p className="text-xs text-gray-500">Funds held safely until both parties confirm</p>
          </div>
        </label>

        {/* Notes */}
        <div className="bg-white rounded-xl border p-5">
          <label className="block text-sm font-medium text-gray-700 mb-1">Notes (optional)</label>
          <textarea className="w-full border rounded-lg px-3 py-2 text-sm" rows={2} value={form.notes} onChange={(e) => set("notes", e.target.value)} placeholder="Payment methods, time preferences..." />
        </div>

        <button type="submit" disabled={loading} className="w-full bg-blue-600 text-white py-3 rounded-lg font-medium hover:bg-blue-700 disabled:opacity-50 transition">
          {loading ? "Creating..." : "Create Exchange Request"}
        </button>
      </form>
    </div>
  );
}
