"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import axios from "axios";
import { ArrowLeft, Plane } from "lucide-react";
import Link from "next/link";

export default function NewTripPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    origin_country: "",
    origin_city: "",
    origin_address: "",
    dest_country: "",
    dest_city: "",
    dest_address: "",
    departure_date: "",
    arrival_date: "",
    available_weight: "",
    max_items: "5",
    price_per_kg: "",
    base_price: "",
    currency: "AED",
    notes: "",
    frequency: "one-time",
  });

  const set = (k: string, v: string) => setForm((p) => ({ ...p, [k]: v }));

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (!form.origin_country || !form.dest_country || !form.departure_date || !form.arrival_date) {
      setError("Origin, destination, and dates are required");
      return;
    }
    setLoading(true);
    try {
      const payload = {
        ...form,
        departure_date: new Date(form.departure_date).toISOString(),
        arrival_date: new Date(form.arrival_date).toISOString(),
        available_weight: parseFloat(form.available_weight) || 0,
        max_items: parseInt(form.max_items) || 5,
        price_per_kg: parseFloat(form.price_per_kg) || 0,
        base_price: parseFloat(form.base_price) || 0,
      };
      await axios.post("/api/v1/trips", payload);
      router.push("/traveler");
    } catch (err: any) {
      setError(err?.response?.data?.message || "Failed to create trip");
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
        <div className="bg-blue-100 p-2 rounded-lg"><Plane className="w-6 h-6 text-blue-600" /></div>
        <div>
          <h1 className="text-2xl font-bold">Create New Trip</h1>
          <p className="text-gray-500 text-sm">Plan your journey and accept delivery requests</p>
        </div>
      </div>

      {error && <div className="bg-red-50 text-red-600 border border-red-200 rounded-lg p-3 mb-4 text-sm">{error}</div>}

      <form onSubmit={submit} className="space-y-6">
        {/* Route */}
        <div className="bg-white rounded-xl border p-5 space-y-4">
          <h2 className="font-semibold text-lg">Route</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Origin Country *</label>
              <input className="w-full border rounded-lg px-3 py-2 text-sm" value={form.origin_country} onChange={(e) => set("origin_country", e.target.value)} placeholder="e.g. UAE" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Origin City *</label>
              <input className="w-full border rounded-lg px-3 py-2 text-sm" value={form.origin_city} onChange={(e) => set("origin_city", e.target.value)} placeholder="e.g. Dubai" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Destination Country *</label>
              <input className="w-full border rounded-lg px-3 py-2 text-sm" value={form.dest_country} onChange={(e) => set("dest_country", e.target.value)} placeholder="e.g. Egypt" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Destination City *</label>
              <input className="w-full border rounded-lg px-3 py-2 text-sm" value={form.dest_city} onChange={(e) => set("dest_city", e.target.value)} placeholder="e.g. Cairo" />
            </div>
          </div>
        </div>

        {/* Schedule */}
        <div className="bg-white rounded-xl border p-5 space-y-4">
          <h2 className="font-semibold text-lg">Schedule</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Departure Date *</label>
              <input type="date" className="w-full border rounded-lg px-3 py-2 text-sm" value={form.departure_date} onChange={(e) => set("departure_date", e.target.value)} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Arrival Date *</label>
              <input type="date" className="w-full border rounded-lg px-3 py-2 text-sm" value={form.arrival_date} onChange={(e) => set("arrival_date", e.target.value)} />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Frequency</label>
            <select className="w-full border rounded-lg px-3 py-2 text-sm" value={form.frequency} onChange={(e) => set("frequency", e.target.value)}>
              <option value="one-time">One-time</option>
              <option value="weekly">Weekly</option>
              <option value="monthly">Monthly</option>
            </select>
          </div>
        </div>

        {/* Capacity & Pricing */}
        <div className="bg-white rounded-xl border p-5 space-y-4">
          <h2 className="font-semibold text-lg">Capacity & Pricing</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Available Weight (kg)</label>
              <input type="number" step="0.1" className="w-full border rounded-lg px-3 py-2 text-sm" value={form.available_weight} onChange={(e) => set("available_weight", e.target.value)} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Max Items</label>
              <input type="number" className="w-full border rounded-lg px-3 py-2 text-sm" value={form.max_items} onChange={(e) => set("max_items", e.target.value)} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Price per kg ({form.currency})</label>
              <input type="number" step="0.01" className="w-full border rounded-lg px-3 py-2 text-sm" value={form.price_per_kg} onChange={(e) => set("price_per_kg", e.target.value)} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Base Price ({form.currency})</label>
              <input type="number" step="0.01" className="w-full border rounded-lg px-3 py-2 text-sm" value={form.base_price} onChange={(e) => set("base_price", e.target.value)} />
            </div>
          </div>
        </div>

        {/* Notes */}
        <div className="bg-white rounded-xl border p-5">
          <label className="block text-sm font-medium text-gray-700 mb-1">Notes</label>
          <textarea className="w-full border rounded-lg px-3 py-2 text-sm" rows={3} value={form.notes} onChange={(e) => set("notes", e.target.value)} placeholder="Any restrictions or details..." />
        </div>

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-blue-600 text-white py-3 rounded-lg font-medium hover:bg-blue-700 disabled:opacity-50 transition"
        >
          {loading ? "Creating..." : "Create Trip"}
        </button>
      </form>
    </div>
  );
}
