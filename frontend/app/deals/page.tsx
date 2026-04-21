'use client';
import { useState, useEffect } from 'react';
import Link from 'next/link';
import api from '@/lib/api';
import {
  Tag, Clock, Percent, TrendingDown, Loader2, Search, Filter
} from 'lucide-react';

interface Deal {
  id: string;
  listing_id: string;
  listing_title: string;
  listing_image: string;
  seller_id: string;
  seller_name: string;
  original_price: number;
  deal_price: number;
  discount_pct: number;
  currency: string;
  start_at: string;
  end_at: string;
  status: string;
  time_remaining: string;
}

export default function DealsPage() {
  const [deals, setDeals] = useState<Deal[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchDeals = async () => {
      setLoading(true);
      try {
        const res = await api.get('/deals');
        setDeals(res.data || []);
      } catch (err: any) {
        setError(err.response?.data?.error || 'Failed to load deals');
      } finally {
        setLoading(false);
      }
    };

    fetchDeals();
  }, []);

  const formatPrice = (price: number, currency: string) => {
    return new Intl.NumberFormat('en-AE', {
      style: 'currency',
      currency: currency || 'AED',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(price);
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Hero Section */}
      <div className="bg-gradient-to-r from-[#0071CE] to-[#003f75] text-white py-12 px-4">
        <div className="max-w-6xl mx-auto text-center">
          <div className="inline-flex items-center gap-2 bg-white/20 rounded-full px-4 py-1.5 text-sm mb-4">
            <TrendingDown size={16} />
            <span>Limited Time Offers</span>
          </div>
          <h1 className="text-4xl font-bold mb-3">Hot Deals</h1>
          <p className="text-blue-100 max-w-xl mx-auto">
            Discover amazing discounts on premium items. Deals updated daily with savings up to 70% off.
          </p>
        </div>
      </div>

      <div className="max-w-6xl mx-auto px-4 py-8">
        {/* Filters */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-500">{deals.length} active deals</span>
          </div>
          <div className="flex items-center gap-2">
            <button className="flex items-center gap-2 px-4 py-2 rounded-xl border border-gray-200 bg-white text-sm text-gray-600 hover:bg-gray-50">
              <Filter size={16} />
              Filter
            </button>
          </div>
        </div>

        {loading ? (
          <div className="flex items-center justify-center py-20">
            <Loader2 size={32} className="animate-spin text-[#0071CE]" />
          </div>
        ) : error ? (
          <div className="text-center py-20 text-gray-500">
            {error}
          </div>
        ) : deals.length === 0 ? (
          <div className="text-center py-20">
            <Tag size={48} className="text-gray-300 mx-auto mb-4" />
            <h2 className="text-xl font-semibold text-gray-700 mb-2">No Active Deals</h2>
            <p className="text-gray-400 text-sm">Check back soon for amazing offers!</p>
          </div>
        ) : (
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {deals.map((deal) => (
              <Link
                key={deal.id}
                href={`/listings/${deal.listing_id}`}
                className="group bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden hover:shadow-lg transition-shadow"
              >
                {/* Image */}
                <div className="relative aspect-square bg-gray-100">
                  {deal.listing_image ? (
                    <img
                      src={deal.listing_image}
                      alt={deal.listing_title}
                      className="w-full h-full object-cover group-hover:scale-105 transition-transform"
                    />
                  ) : (
                    <div className="w-full h-full flex items-center justify-center">
                      <Tag size={40} className="text-gray-300" />
                    </div>
                  )}
                  {/* Discount Badge */}
                  <div className="absolute top-3 left-3 bg-red-500 text-white text-sm font-bold px-2.5 py-1 rounded-lg shadow">
                    -{deal.discount_pct}%
                  </div>
                  {/* Timer */}
                  <div className="absolute bottom-3 left-3 right-3 bg-black/70 text-white text-xs px-3 py-2 rounded-lg flex items-center gap-2">
                    <Clock size={12} />
                    <span>{deal.time_remaining}</span>
                  </div>
                </div>

                {/* Content */}
                <div className="p-4">
                  <h3 className="font-semibold text-gray-900 text-sm line-clamp-2 mb-2 group-hover:text-[#0071CE]">
                    {deal.listing_title}
                  </h3>
                  <p className="text-xs text-gray-400 mb-3">by {deal.seller_name}</p>
                  <div className="flex items-center gap-2">
                    <span className="text-lg font-bold text-red-600">
                      {formatPrice(deal.deal_price, deal.currency)}
                    </span>
                    <span className="text-sm text-gray-400 line-through">
                      {formatPrice(deal.original_price, deal.currency)}
                    </span>
                  </div>
                </div>
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
