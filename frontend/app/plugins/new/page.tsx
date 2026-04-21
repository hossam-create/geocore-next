"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import axios from "axios";
import { ArrowLeft, Puzzle } from "lucide-react";

const CATEGORIES = ["general", "payments", "shipping", "analytics", "marketing", "ai", "security"];

export default function CreatePluginPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    name: "",
    description: "",
    category: "general",
    icon_url: "",
    repo_url: "",
    price: "",
    is_free: true,
  });

  const set = (k: string, v: string | boolean) => setForm((p) => ({ ...p, [k]: v }));

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.name) { setError("Name is required"); return; }
    setLoading(true);
    setError("");
    try {
      await axios.post("/api/v1/plugins", {
        ...form,
        price: parseFloat(form.price) || 0,
      });
      router.push("/plugins");
    } catch (err: any) {
      setError(err?.response?.data?.message || "Failed to create plugin");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-lg mx-auto px-4 py-8">
      <Link href="/plugins" className="flex items-center gap-2 text-gray-500 hover:text-gray-700 mb-6 text-sm">
        <ArrowLeft className="w-4 h-4" /> Back
      </Link>

      <div className="flex items-center gap-3 mb-6">
        <div className="bg-purple-100 p-2 rounded-lg"><Puzzle className="w-6 h-6 text-purple-600" /></div>
        <h1 className="text-2xl font-bold">Create Plugin</h1>
      </div>

      {error && <div className="bg-red-50 text-red-600 border border-red-200 rounded-lg p-3 mb-4 text-sm">{error}</div>}

      <form onSubmit={submit} className="space-y-5">
        <div className="bg-white rounded-xl border p-5 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Name *</label>
            <input className="w-full border rounded-lg px-3 py-2 text-sm" value={form.name} onChange={(e) => set("name", e.target.value)} placeholder="My Awesome Plugin" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
            <textarea className="w-full border rounded-lg px-3 py-2 text-sm" rows={3} value={form.description} onChange={(e) => set("description", e.target.value)} />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Category</label>
              <select className="w-full border rounded-lg px-3 py-2 text-sm" value={form.category} onChange={(e) => set("category", e.target.value)}>
                {CATEGORIES.map((c) => <option key={c} value={c} className="capitalize">{c}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Icon URL</label>
              <input className="w-full border rounded-lg px-3 py-2 text-sm" value={form.icon_url} onChange={(e) => set("icon_url", e.target.value)} />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Repository URL</label>
            <input className="w-full border rounded-lg px-3 py-2 text-sm" value={form.repo_url} onChange={(e) => set("repo_url", e.target.value)} placeholder="https://github.com/..." />
          </div>
          <label className="flex items-center gap-2">
            <input type="checkbox" className="h-4 w-4 text-purple-600 rounded" checked={form.is_free} onChange={(e) => set("is_free", e.target.checked)} />
            <span className="text-sm">Free plugin</span>
          </label>
          {!form.is_free && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Price (USD)</label>
              <input type="number" step="0.01" className="w-full border rounded-lg px-3 py-2 text-sm" value={form.price} onChange={(e) => set("price", e.target.value)} />
            </div>
          )}
        </div>

        <button type="submit" disabled={loading} className="w-full bg-purple-600 text-white py-3 rounded-lg font-medium hover:bg-purple-700 disabled:opacity-50 transition">
          {loading ? "Creating..." : "Create Plugin"}
        </button>
      </form>
    </div>
  );
}
