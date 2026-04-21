"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import axios from "axios";
import { Puzzle, Search, Star, Download, Plus } from "lucide-react";

interface Plugin {
  id: string;
  name: string;
  slug: string;
  description: string;
  category: string;
  icon_url: string;
  version: string;
  price: number;
  currency: string;
  is_free: boolean;
  install_count: number;
  avg_rating: number;
}

const CATEGORIES = ["general", "payments", "shipping", "analytics", "marketing", "ai", "security"];

export default function PluginMarketplacePage() {
  const [category, setCategory] = useState("");
  const [search, setSearch] = useState("");

  const { data: plugins = [], isLoading } = useQuery<Plugin[]>({
    queryKey: ["plugins", category, search],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (category) params.set("category", category);
      if (search) params.set("q", search);
      const { data } = await axios.get(`/api/v1/plugins?${params}`);
      return data.data ?? [];
    },
  });

  return (
    <div className="max-w-6xl mx-auto px-4 py-8">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <div className="bg-purple-100 p-2 rounded-lg"><Puzzle className="w-6 h-6 text-purple-600" /></div>
          <div>
            <h1 className="text-2xl font-bold">Plugin Marketplace</h1>
            <p className="text-gray-500 text-sm">Extend your store with powerful plugins</p>
          </div>
        </div>
        <Link href="/plugins/new" className="flex items-center gap-2 bg-purple-600 text-white px-4 py-2 rounded-lg hover:bg-purple-700 transition text-sm">
          <Plus className="w-4 h-4" /> Create Plugin
        </Link>
      </div>

      {/* Search + Categories */}
      <div className="flex flex-wrap gap-3 mb-6">
        <div className="flex items-center gap-2 bg-white border rounded-lg px-3 py-2 flex-1 min-w-[200px]">
          <Search className="w-4 h-4 text-gray-400" />
          <input className="text-sm outline-none w-full" placeholder="Search plugins..." value={search} onChange={(e) => setSearch(e.target.value)} />
        </div>
        <div className="flex gap-1 flex-wrap">
          <button onClick={() => setCategory("")} className={`text-xs px-3 py-1.5 rounded-full font-medium transition ${!category ? "bg-purple-600 text-white" : "bg-gray-100 text-gray-600 hover:bg-gray-200"}`}>All</button>
          {CATEGORIES.map((c) => (
            <button key={c} onClick={() => setCategory(c)} className={`text-xs px-3 py-1.5 rounded-full font-medium capitalize transition ${category === c ? "bg-purple-600 text-white" : "bg-gray-100 text-gray-600 hover:bg-gray-200"}`}>{c}</button>
          ))}
        </div>
      </div>

      <div className="flex gap-3 mb-6">
        <Link href="/plugins/installed" className="text-sm text-purple-600 hover:underline">My Installed Plugins</Link>
      </div>

      {isLoading && <p className="text-center py-12 text-gray-400">Loading...</p>}

      {!isLoading && plugins.length === 0 && (
        <div className="text-center py-16 text-gray-400">
          <Puzzle className="w-12 h-12 mx-auto mb-3 opacity-40" />
          <p>No plugins found. Be the first to publish one!</p>
        </div>
      )}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {plugins.map((p) => (
          <Link key={p.id} href={`/plugins/${p.slug}`} className="bg-white rounded-xl border p-5 hover:shadow-md transition">
            <div className="flex items-center gap-3 mb-3">
              {p.icon_url ? (
                <img src={p.icon_url} alt="" className="w-10 h-10 rounded-lg object-cover" />
              ) : (
                <div className="w-10 h-10 bg-purple-100 rounded-lg flex items-center justify-center">
                  <Puzzle className="w-5 h-5 text-purple-500" />
                </div>
              )}
              <div>
                <p className="font-semibold text-sm">{p.name}</p>
                <span className="text-xs text-gray-400">v{p.version}</span>
              </div>
            </div>
            {p.description && <p className="text-sm text-gray-600 line-clamp-2 mb-3">{p.description}</p>}
            <div className="flex items-center justify-between text-xs text-gray-500">
              <div className="flex items-center gap-3">
                <span className="flex items-center gap-1"><Download className="w-3 h-3" /> {p.install_count}</span>
                {p.avg_rating > 0 && <span className="flex items-center gap-1"><Star className="w-3 h-3 text-yellow-500" /> {p.avg_rating}</span>}
              </div>
              <span className={p.is_free ? "text-green-600 font-medium" : "text-gray-700 font-medium"}>
                {p.is_free ? "Free" : `${p.currency} ${p.price}`}
              </span>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
