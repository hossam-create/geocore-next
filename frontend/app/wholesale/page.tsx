'use client'

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import Link from "next/link";
import api from "@/lib/api";
import { formatPrice } from "@/lib/utils";
import { useAuthStore } from "@/store/auth";
import { Package, Search, Filter, CheckCircle, TrendingUp, Users, ChevronRight, Store } from "lucide-react";

interface PriceTier {
  min_quantity: number;
  max_quantity: number;
  unit_price_cents: number;
}

interface WholesaleListing {
  id: string;
  seller_id: string;
  title: string;
  description: string;
  category_slug: string;
  images: string[];
  unit_price_cents: number;
  currency: string;
  tier_pricing: PriceTier[];
  moq: number;
  max_order_quantity: number;
  available_units: number;
  units_per_lot: number;
  shipping_per_unit_cents: number;
  free_shipping_moq: number;
  lead_time_days: number;
  status: string;
  is_verified: boolean;
  views_count: number;
  orders_count: number;
}

export default function WholesalePage() {
  const t = useTranslations("wholesale");
  const { isAuthenticated } = useAuthStore();
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("");
  const [verifiedOnly, setVerifiedOnly] = useState(false);
  const [page, setPage] = useState(1);

  const { data, isLoading } = useQuery({
    queryKey: ["wholesale", "listings", { search, category, verifiedOnly, page }],
    queryFn: async () => {
      const params = new URLSearchParams();
      params.set("page", String(page));
      params.set("page_size", "20");
      if (category) params.set("category", category);
      if (verifiedOnly) params.set("verified", "true");
      const res = await api.get(`/wholesale/listings?${params}`);
      return res.data?.data as { items: WholesaleListing[]; total: number; page: number; page_size: number };
    },
  });

  const listings = data?.items ?? [];
  const total = data?.total ?? 0;

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Hero */}
      <div className="bg-gradient-to-r from-[#1E293B] to-[#334155] text-white">
        <div className="max-w-7xl mx-auto px-4 py-10">
          <div className="flex items-center gap-3 mb-2">
            <Package size={28} className="text-emerald-400" />
            <h1 className="text-3xl font-bold">{t("title")}</h1>
          </div>
          <p className="text-slate-300 text-sm max-w-xl">
            {t("subtitle")}
          </p>
          <div className="flex gap-6 mt-6">
            <div className="flex items-center gap-2 text-sm">
              <TrendingUp size={16} className="text-emerald-400" />
              <span>{t("tieredPricing")}</span>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <CheckCircle size={16} className="text-emerald-400" />
              <span>{t("verifiedSellers")}</span>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <Users size={16} className="text-emerald-400" />
              <span>{t("bulkOrders")}</span>
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 py-6">
        {/* Filters */}
        <div className="flex flex-wrap gap-3 mb-6">
          <div className="relative flex-1 min-w-[240px]">
            <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              placeholder="Search wholesale products..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-9 pr-4 py-2.5 border border-gray-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
            />
          </div>
          <select
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            className="border border-gray-200 rounded-xl px-4 py-2.5 text-sm text-gray-700 focus:outline-none focus:ring-2 focus:ring-emerald-500"
          >
            <option value="">All Categories</option>
            <option value="electronics">Electronics</option>
            <option value="fashion">Fashion</option>
            <option value="home-garden">Home & Garden</option>
            <option value="industrial">Industrial</option>
            <option value="beauty">Beauty & Health</option>
            <option value="sports">Sports & Outdoors</option>
          </select>
          <button
            onClick={() => setVerifiedOnly(!verifiedOnly)}
            className={`flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-medium transition-colors ${
              verifiedOnly
                ? "bg-emerald-100 text-emerald-700 border border-emerald-300"
                : "border border-gray-200 text-gray-600 hover:bg-gray-50"
            }`}
          >
            <CheckCircle size={14} />
            Verified Only
          </button>
          {isAuthenticated && (
            <Link
              href="/wholesale/seller-register"
              className="flex items-center gap-2 px-4 py-2.5 bg-emerald-600 text-white rounded-xl text-sm font-medium hover:bg-emerald-700 transition-colors"
            >
              <Store size={14} />
              Become a Seller
            </Link>
          )}
        </div>

        {/* Results */}
        {isLoading ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="bg-white rounded-xl p-5 animate-pulse">
                <div className="h-40 bg-gray-100 rounded-lg mb-4" />
                <div className="h-4 bg-gray-100 rounded w-3/4 mb-2" />
                <div className="h-3 bg-gray-100 rounded w-1/2" />
              </div>
            ))}
          </div>
        ) : listings.length === 0 ? (
          <div className="text-center py-20">
            <Package size={48} className="text-gray-300 mx-auto mb-4" />
            <h2 className="text-lg font-semibold text-gray-700 mb-1">No wholesale listings found</h2>
            <p className="text-sm text-gray-400">Try adjusting your filters or check back later.</p>
          </div>
        ) : (
          <>
            <p className="text-xs text-gray-400 mb-3">{total} wholesale listing{total !== 1 ? "s" : ""}</p>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {listings.map((l) => {
                const basePrice = l.unit_price_cents / 100;
                const bestTier = l.tier_pricing?.length
                  ? l.tier_pricing.reduce((best, t) => t.unit_price_cents < best.unit_price_cents ? t : best, l.tier_pricing[0])
                  : null;
                const bestPrice = bestTier ? bestTier.unit_price_cents / 100 : basePrice;
                const discount = bestTier ? Math.round((1 - bestPrice / basePrice) * 100) : 0;

                return (
                  <Link
                    key={l.id}
                    href={`/wholesale/${l.id}`}
                    className="bg-white rounded-xl border border-gray-100 hover:border-emerald-200 hover:shadow-md transition-all overflow-hidden group"
                  >
                    {/* Image */}
                    <div className="h-44 bg-gray-100 relative overflow-hidden">
                      {l.images?.[0] ? (
                        <img src={l.images[0]} alt={l.title} className="w-full h-full object-cover group-hover:scale-105 transition-transform" />
                      ) : (
                        <div className="w-full h-full flex items-center justify-center text-gray-300">
                          <Package size={40} />
                        </div>
                      )}
                      {l.is_verified && (
                        <span className="absolute top-2 left-2 bg-emerald-500 text-white text-[10px] font-bold px-2 py-0.5 rounded-full flex items-center gap-1">
                          <CheckCircle size={10} /> Verified
                        </span>
                      )}
                      {discount > 0 && (
                        <span className="absolute top-2 right-2 bg-red-500 text-white text-[10px] font-bold px-2 py-0.5 rounded-full">
                          Up to {discount}% off
                        </span>
                      )}
                    </div>

                    {/* Info */}
                    <div className="p-4">
                      <h3 className="font-semibold text-gray-800 text-sm line-clamp-2 mb-2 group-hover:text-emerald-700 transition-colors">
                        {l.title}
                      </h3>

                      <div className="flex items-baseline gap-2 mb-2">
                        <span className="text-lg font-bold text-emerald-700">{formatPrice(bestPrice, l.currency)}</span>
                        <span className="text-xs text-gray-400">/ unit</span>
                        {bestTier && bestPrice < basePrice && (
                          <span className="text-xs text-gray-400 line-through">{formatPrice(basePrice, l.currency)}</span>
                        )}
                      </div>

                      <div className="flex flex-wrap gap-2 text-[11px]">
                        <span className="bg-amber-50 text-amber-700 px-2 py-0.5 rounded-md font-medium">
                          MOQ: {l.moq}
                        </span>
                        {l.units_per_lot > 1 && (
                          <span className="bg-blue-50 text-blue-700 px-2 py-0.5 rounded-md font-medium">
                            {l.units_per_lot}/lot
                          </span>
                        )}
                        {l.available_units > 0 && (
                          <span className="bg-gray-50 text-gray-600 px-2 py-0.5 rounded-md">
                            {l.available_units} available
                          </span>
                        )}
                        {l.lead_time_days > 0 && (
                          <span className="bg-gray-50 text-gray-600 px-2 py-0.5 rounded-md">
                            {l.lead_time_days}d delivery
                          </span>
                        )}
                      </div>

                      {l.tier_pricing?.length > 0 && (
                        <div className="mt-3 pt-2 border-t border-gray-50">
                          <p className="text-[10px] text-gray-400 font-semibold uppercase mb-1">Volume Pricing</p>
                          <div className="space-y-0.5">
                            {l.tier_pricing.slice(0, 3).map((t, i) => (
                              <div key={i} className="flex justify-between text-[11px]">
                                <span className="text-gray-500">{t.min_quantity}+ units</span>
                                <span className="font-medium text-emerald-600">{formatPrice(t.unit_price_cents / 100, l.currency)}/u</span>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  </Link>
                );
              })}
            </div>

            {/* Pagination */}
            {total > 20 && (
              <div className="flex justify-center gap-2 mt-8">
                <button
                  onClick={() => setPage(Math.max(1, page - 1))}
                  disabled={page <= 1}
                  className="px-4 py-2 text-sm border border-gray-200 rounded-lg disabled:opacity-40 hover:bg-gray-50"
                >
                  Previous
                </button>
                <span className="px-4 py-2 text-sm text-gray-500">Page {page}</span>
                <button
                  onClick={() => setPage(page + 1)}
                  disabled={listings.length < 20}
                  className="px-4 py-2 text-sm border border-gray-200 rounded-lg disabled:opacity-40 hover:bg-gray-50"
                >
                  Next
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
