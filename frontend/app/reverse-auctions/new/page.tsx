"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import axios from "axios";
import { ArrowLeft, Tag } from "lucide-react";

export default function NewReverseAuctionPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    title: "",
    description: "",
    category_id: "",
    max_budget: "",
    deadline: "",
  });

  const set = (k: string, v: string) => setForm((p) => ({ ...p, [k]: v }));

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.title || !form.deadline) {
      setError("Title and deadline are required");
      return;
    }
    setLoading(true);
    setError("");
    try {
      const body: Record<string, unknown> = {
        title: form.title,
        description: form.description,
        deadline: new Date(form.deadline).toISOString(),
      };
      if (form.max_budget) body.max_budget = parseFloat(form.max_budget);
      if (form.category_id) body.category_id = form.category_id;

      await axios.post("/api/v1/reverse-auctions", body);
      router.push("/reverse-auctions");
    } catch (err: any) {
      setError(err?.response?.data?.message || "Failed to create request");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-lg mx-auto px-4 py-8">
      <Link
        href="/reverse-auctions"
        className="flex items-center gap-2 text-gray-500 hover:text-gray-700 mb-6 text-sm"
      >
        <ArrowLeft className="w-4 h-4" /> Back
      </Link>

      <div className="flex items-center gap-3 mb-6">
        <div className="bg-purple-100 p-2 rounded-lg">
          <Tag className="w-6 h-6 text-purple-600" />
        </div>
        <h1 className="text-2xl font-bold">Request a Product</h1>
      </div>

      {error && (
        <div className="bg-red-50 text-red-600 border border-red-200 rounded-lg p-3 mb-4 text-sm">
          {error}
        </div>
      )}

      <form onSubmit={submit} className="space-y-5">
        <div className="bg-white rounded-xl border p-5 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              What are you looking for? *
            </label>
            <input
              className="w-full border rounded-lg px-3 py-2 text-sm"
              value={form.title}
              onChange={(e) => set("title", e.target.value)}
              placeholder="e.g. Laptop Core i7, 16GB RAM"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              className="w-full border rounded-lg px-3 py-2 text-sm"
              rows={4}
              value={form.description}
              onChange={(e) => set("description", e.target.value)}
              placeholder="Describe what you need in detail..."
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Max Budget
              </label>
              <input
                type="number"
                step="0.01"
                className="w-full border rounded-lg px-3 py-2 text-sm"
                value={form.max_budget}
                onChange={(e) => set("max_budget", e.target.value)}
                placeholder="15000"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Deadline *
              </label>
              <input
                type="datetime-local"
                className="w-full border rounded-lg px-3 py-2 text-sm"
                value={form.deadline}
                onChange={(e) => set("deadline", e.target.value)}
              />
            </div>
          </div>
        </div>

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-purple-600 text-white py-3 rounded-lg font-medium hover:bg-purple-700 disabled:opacity-50 transition"
        >
          {loading ? "Submitting..." : "Submit Request"}
        </button>
      </form>
    </div>
  );
}
