"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import axios from "axios";
import { ArrowLeft, Package } from "lucide-react";
import Link from "next/link";

export default function NewDeliveryRequestPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    item_name: "",
    item_description: "",
    item_url: "",
    item_price: "",
    item_weight: "",
    pickup_country: "",
    pickup_city: "",
    delivery_country: "",
    delivery_city: "",
    reward: "",
    currency: "AED",
    deadline: "",
    notes: "",
  });

  const set = (k: string, v: string) => setForm((p) => ({ ...p, [k]: v }));

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (!form.item_name || !form.pickup_country || !form.delivery_country || !form.reward) {
      setError("Item name, pickup/delivery countries, and reward are required");
      return;
    }
    setLoading(true);
    try {
      const payload = {
        ...form,
        item_price: parseFloat(form.item_price) || 0,
        item_weight: form.item_weight ? parseFloat(form.item_weight) : undefined,
        reward: parseFloat(form.reward) || 0,
        deadline: form.deadline ? new Date(form.deadline).toISOString() : undefined,
      };
      await axios.post("/api/v1/delivery-requests", payload);
      router.push("/traveler");
    } catch (err: any) {
      setError(err?.response?.data?.message || "Failed to create request");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-2xl mx-auto px-4 py-8">
      <Link href="/traveler" className="flex items-center gap-2 text-gray-500 hover:text-gray-700 mb-6 text-sm">
        <ArrowLeft className="w-4 h-4" /> Back to Dashboard
      </Link>

      <div className="flex items-center gap-3 mb-6">
        <div className="bg-orange-100 p-2 rounded-lg"><Package className="w-6 h-6 text-orange-600" /></div>
        <div>
          <h1 className="text-2xl font-bold">Request a Delivery</h1>
          <p className="text-gray-500 text-sm">Find a traveler to bring your item</p>
        </div>
      </div>

      {error && <div className="bg-red-50 text-red-600 border border-red-200 rounded-lg p-3 mb-4 text-sm">{error}</div>}

      <form onSubmit={submit} className="space-y-6">
        {/* Item Info */}
        <div className="bg-white rounded-xl border p-5 space-y-4">
          <h2 className="font-semibold text-lg">Item Details</h2>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Item Name *</label>
            <input className="w-full border rounded-lg px-3 py-2 text-sm" value={form.item_name} onChange={(e) => set("item_name", e.target.value)} placeholder="e.g. Swiss Chocolate" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
            <textarea className="w-full border rounded-lg px-3 py-2 text-sm" rows={2} value={form.item_description} onChange={(e) => set("item_description", e.target.value)} />
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Price *</label>
              <input type="number" step="0.01" className="w-full border rounded-lg px-3 py-2 text-sm" value={form.item_price} onChange={(e) => set("item_price", e.target.value)} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Weight (kg)</label>
              <input type="number" step="0.1" className="w-full border rounded-lg px-3 py-2 text-sm" value={form.item_weight} onChange={(e) => set("item_weight", e.target.value)} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Item URL</label>
              <input className="w-full border rounded-lg px-3 py-2 text-sm" value={form.item_url} onChange={(e) => set("item_url", e.target.value)} />
            </div>
          </div>
        </div>

        {/* Route */}
        <div className="bg-white rounded-xl border p-5 space-y-4">
          <h2 className="font-semibold text-lg">Route</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Pickup Country *</label>
              <input className="w-full border rounded-lg px-3 py-2 text-sm" value={form.pickup_country} onChange={(e) => set("pickup_country", e.target.value)} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Pickup City *</label>
              <input className="w-full border rounded-lg px-3 py-2 text-sm" value={form.pickup_city} onChange={(e) => set("pickup_city", e.target.value)} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Delivery Country *</label>
              <input className="w-full border rounded-lg px-3 py-2 text-sm" value={form.delivery_country} onChange={(e) => set("delivery_country", e.target.value)} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Delivery City *</label>
              <input className="w-full border rounded-lg px-3 py-2 text-sm" value={form.delivery_city} onChange={(e) => set("delivery_city", e.target.value)} />
            </div>
          </div>
        </div>

        {/* Reward & Deadline */}
        <div className="bg-white rounded-xl border p-5 space-y-4">
          <h2 className="font-semibold text-lg">Reward & Deadline</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Reward ({form.currency}) *</label>
              <input type="number" step="0.01" className="w-full border rounded-lg px-3 py-2 text-sm" value={form.reward} onChange={(e) => set("reward", e.target.value)} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Deadline</label>
              <input type="date" className="w-full border rounded-lg px-3 py-2 text-sm" value={form.deadline} onChange={(e) => set("deadline", e.target.value)} />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Notes</label>
            <textarea className="w-full border rounded-lg px-3 py-2 text-sm" rows={2} value={form.notes} onChange={(e) => set("notes", e.target.value)} />
          </div>
        </div>

        <button type="submit" disabled={loading} className="w-full bg-orange-600 text-white py-3 rounded-lg font-medium hover:bg-orange-700 disabled:opacity-50 transition">
          {loading ? "Submitting..." : "Submit Delivery Request"}
        </button>
      </form>
    </div>
  );
}
