"use client";

import { useState } from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { formatPrice } from "@/lib/utils";
import {
  Heart, Trash2, Tag, Clock, Gavel, ShoppingCart,
  Search, Eye, Package,
} from "lucide-react";
import { formatDistanceToNow, parseISO } from "date-fns";

interface WatchlistItem {
  id: string;
  title: string;
  price: number;
  currency: string;
  type: string;
  ends_at?: string;
  image?: string;
  image_url?: string;
  price_drop?: boolean;
  views?: number;
  is_auction?: boolean;
}

const MOCK: WatchlistItem[] = [
  { id: "wl-001", title: "MacBook Pro M3 14\" — Space Black",    price: 8200,  currency: "AED", type: "buy_now",         image: "https://picsum.photos/seed/mbp/200/200",    price_drop: true,  views: 312 },
  { id: "wl-002", title: "Canon EOS R6 Mark II Body",            price: 11500, currency: "AED", type: "standard_auction", image: "https://picsum.photos/seed/canon/200/200",  ends_at: new Date(Date.now() + 5 * 3600000).toISOString(),  views: 88 },
  { id: "wl-003", title: "PlayStation 5 Slim + Extra Controller",price: 2300,  currency: "AED", type: "buy_now",         image: "https://picsum.photos/seed/ps5s/200/200",   views: 540 },
  { id: "wl-004", title: "Vintage Rolex Datejust 36mm",          price: 24000, currency: "AED", type: "sealed_auction",  image: "https://picsum.photos/seed/rolex2/200/200", ends_at: new Date(Date.now() + 2 * 86400000).toISOString(), views: 710 },
  { id: "wl-005", title: "Nike Air Jordan 1 Retro High OG",      price: 850,   currency: "AED", type: "buy_now",         image: "https://picsum.photos/seed/jordan1/200/200",price_drop: true,  views: 230 },
  { id: "wl-006", title: "Leica M11 Rangefinder Camera",         price: 32000, currency: "AED", type: "buy_now",         image: "https://picsum.photos/seed/leica/200/200",  views: 95 },
];

function timeLeft(dateStr: string): string {
  const diff = new Date(dateStr).getTime() - Date.now();
  if (diff <= 0) return "Ended";
  const h = Math.floor(diff / 3600000);
  const d = Math.floor(diff / 86400000);
  if (d >= 1) return `${d}d left`;
  return `${h}h left`;
}

function typeLabel(type: string): { label: string; cls: string; icon: React.ElementType } {
  if (type.includes("auction")) return { label: "Auction", cls: "bg-purple-50 text-purple-700 border-purple-200", icon: Gavel };
  return { label: "Buy Now", cls: "bg-emerald-50 text-emerald-700 border-emerald-200", icon: ShoppingCart };
}

type SortKey = "added" | "price_asc" | "price_desc" | "ending_soon";

export default function BuyerWatchlistPage() {
  const qc = useQueryClient();
  const { isAuthenticated } = useAuthStore();
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<SortKey>("added");

  const { data: items = MOCK, isLoading } = useQuery<WatchlistItem[]>({
    queryKey: ["buyer-watchlist-full"],
    queryFn: async () => {
      try {
        const d = (await api.get("/watchlist?page=1&per_page=100")).data?.data ?? [];
        return Array.isArray(d) && d.length ? d : MOCK;
      } catch { return MOCK; }
    },
    enabled: isAuthenticated,
  });

  const removeMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/watchlist/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["buyer-watchlist-full"] }),
  });

  const priceDrops = items.filter((i) => i.price_drop).length;
  const endingSoon = items.filter((i) => i.ends_at && new Date(i.ends_at).getTime() - Date.now() < 24 * 3600000 && new Date(i.ends_at).getTime() > Date.now()).length;

  let sorted = [...items].filter((i) => !search || i.title.toLowerCase().includes(search.toLowerCase()));
  if (sort === "price_asc")    sorted.sort((a, b) => a.price - b.price);
  if (sort === "price_desc")   sorted.sort((a, b) => b.price - a.price);
  if (sort === "ending_soon")  sorted.sort((a, b) => {
    const aEnd = a.ends_at ? new Date(a.ends_at).getTime() : Infinity;
    const bEnd = b.ends_at ? new Date(b.ends_at).getTime() : Infinity;
    return aEnd - bEnd;
  });

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div>
          <h1 className="text-xl font-bold text-gray-900">Watchlist</h1>
          <p className="text-sm text-gray-400">{items.length} saved items</p>
        </div>
        <Link
          href="/listings"
          className="flex items-center gap-1.5 px-4 py-2 bg-indigo-600 text-white rounded-xl text-sm font-semibold hover:bg-indigo-700 transition-colors"
        >
          Browse Listings
        </Link>
      </div>

      {/* Alert banners */}
      {(priceDrops > 0 || endingSoon > 0) && (
        <div className="flex flex-wrap gap-3">
          {priceDrops > 0 && (
            <div className="flex items-center gap-2 bg-rose-50 border border-rose-200 rounded-xl px-4 py-2.5 text-sm text-rose-700">
              <Tag className="w-4 h-4 shrink-0" />
              <span><strong>{priceDrops}</strong> item{priceDrops > 1 ? "s" : ""} dropped in price</span>
            </div>
          )}
          {endingSoon > 0 && (
            <div className="flex items-center gap-2 bg-amber-50 border border-amber-200 rounded-xl px-4 py-2.5 text-sm text-amber-700">
              <Clock className="w-4 h-4 shrink-0" />
              <span><strong>{endingSoon}</strong> auction{endingSoon > 1 ? "s" : ""} ending within 24h</span>
            </div>
          )}
        </div>
      )}

      {/* Search + Sort */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search watchlist..."
            className="pl-8 pr-3 py-1.5 text-sm border border-gray-200 rounded-xl bg-white outline-none focus:ring-2 focus:ring-indigo-300 focus:border-indigo-400 w-56"
          />
        </div>
        <select
          value={sort}
          onChange={(e) => setSort(e.target.value as SortKey)}
          className="text-sm border border-gray-200 rounded-xl px-3 py-1.5 bg-white outline-none focus:ring-2 focus:ring-indigo-300 focus:border-indigo-400"
        >
          <option value="added">Recently Added</option>
          <option value="price_asc">Price: Low → High</option>
          <option value="price_desc">Price: High → Low</option>
          <option value="ending_soon">Ending Soon</option>
        </select>
      </div>

      {/* Grid */}
      {isLoading ? (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
          {[1, 2, 3, 4].map((i) => <div key={i} className="h-64 animate-pulse rounded-2xl bg-gray-100" />)}
        </div>
      ) : sorted.length === 0 ? (
        <div className="py-20 text-center bg-white rounded-2xl border border-gray-100">
          <Heart className="w-12 h-12 mx-auto mb-3 text-gray-200" />
          <p className="text-sm font-semibold text-gray-500">
            {search ? `No items matching "${search}"` : "Your watchlist is empty"}
          </p>
          <Link href="/listings" className="mt-3 inline-block text-xs text-indigo-600 hover:underline font-medium">
            Browse listings to add items →
          </Link>
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
          {sorted.map((item) => {
            const tl = typeLabel(item.type);
            const TypeIcon = tl.icon;
            const img = item.image ?? item.image_url;
            return (
              <div key={item.id} className="group relative bg-white rounded-2xl border border-gray-100 overflow-hidden hover:shadow-md transition-shadow">
                {/* Image */}
                <div className="relative h-40 bg-gray-100 overflow-hidden">
                  {img ? (
                    <img src={img} alt={item.title} className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300" />
                  ) : (
                    <div className="w-full h-full flex items-center justify-center">
                      <Package className="w-10 h-10 text-gray-300" />
                    </div>
                  )}
                  {/* Badges */}
                  <div className="absolute top-2 left-2 flex flex-col gap-1">
                    {item.price_drop && (
                      <span className="inline-flex items-center gap-0.5 text-[10px] font-bold bg-rose-500 text-white px-1.5 py-0.5 rounded-full">
                        <Tag className="w-2.5 h-2.5" /> Price Drop
                      </span>
                    )}
                    {item.ends_at && new Date(item.ends_at).getTime() > Date.now() && (
                      <span className="inline-flex items-center gap-0.5 text-[10px] font-bold bg-amber-500 text-white px-1.5 py-0.5 rounded-full">
                        <Clock className="w-2.5 h-2.5" /> {timeLeft(item.ends_at)}
                      </span>
                    )}
                  </div>
                  {/* Remove */}
                  <button
                    onClick={(e) => { e.preventDefault(); removeMutation.mutate(item.id); }}
                    className="absolute top-2 right-2 p-1.5 rounded-full bg-white/80 hover:bg-red-50 text-gray-400 hover:text-red-500 transition-colors opacity-0 group-hover:opacity-100"
                    title="Remove from watchlist"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>

                {/* Info */}
                <Link href={`/listings/${item.id}`} className="block p-3">
                  <p className="text-[13px] font-semibold text-gray-900 leading-snug line-clamp-2 mb-2">{item.title}</p>
                  <div className="flex items-center justify-between gap-2">
                    <p className="text-sm font-bold text-gray-900 tabular-nums">{formatPrice(item.price, item.currency)}</p>
                    <span className={`inline-flex items-center gap-0.5 text-[10px] font-bold px-1.5 py-0.5 rounded-full border ${tl.cls}`}>
                      <TypeIcon className="w-2.5 h-2.5" /> {tl.label}
                    </span>
                  </div>
                  {item.views && (
                    <p className="text-[11px] text-gray-400 mt-1.5 flex items-center gap-1">
                      <Eye className="w-3 h-3" /> {item.views.toLocaleString()} views
                    </p>
                  )}
                </Link>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
