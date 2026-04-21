'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ListingCard } from '@/components/listings/ListingCard';
import { getWatchlist, getWatchlistPriceSnapshot, type WatchlistItem } from '@/lib/api';
import { useAuthStore } from '@/store/auth';
import type { Listing } from '@/types/listing';

function toListing(item: WatchlistItem): Listing {
  return {
    id: item.id,
    title: item.title,
    price: item.price,
    currency: item.currency,
    image_url: item.image_url,
    images: item.images?.map((img, idx) => ({
      id: img.id || `${item.id}_img_${idx}`,
      url: img.url,
    })),
    city: item.city,
    type: item.type as Listing['type'],
    is_auction: item.is_auction,
    is_watched: true,
    created_at: item.created_at,
  };
}

export default function WatchlistPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [removedIds, setRemovedIds] = useState<string[]>([]);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login?next=/watchlist');
    }
  }, [isAuthenticated, router]);

  const { data, isLoading, isError } = useQuery({
    queryKey: ['watchlist'],
    queryFn: () => getWatchlist(1, 50),
    enabled: isAuthenticated,
    retry: false,
  });

  const items = data?.items ?? [];

  useEffect(() => {
    setRemovedIds([]);
  }, [data?.meta?.total]);

  const rows = useMemo(
    () =>
      items.map((item) => {
        const previousPrice = getWatchlistPriceSnapshot(item.id);
        const currentPrice = typeof item.price === 'number' ? item.price : null;
        const dropped =
          previousPrice !== null &&
          currentPrice !== null &&
          previousPrice > 0 &&
          currentPrice < previousPrice;
        const dropPercent = dropped ? Math.round(((previousPrice - currentPrice) / previousPrice) * 100) : 0;
        return { listing: toListing(item), dropped, dropPercent };
      }),
    [items]
  );

  const visibleRows = rows.filter(({ listing }) => !removedIds.includes(listing.id));

  if (!isAuthenticated) return null;

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">My Watchlist</h1>
        <p className="text-sm text-gray-500">Saved listings and price movement since you added them.</p>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-64 animate-pulse rounded-2xl bg-gray-100" />
          ))}
        </div>
      ) : isError ? (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-6 text-sm text-red-700">
          Could not load your watchlist right now. Please try again.
        </div>
      ) : visibleRows.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-gray-300 bg-white p-12 text-center">
          <p className="text-lg font-semibold text-gray-800">Your watchlist is empty</p>
          <p className="mt-1 text-sm text-gray-500">Tap the heart icon on any listing to save it here.</p>
          <Link
            href="/listings"
            className="mt-4 inline-block rounded-xl bg-[#0071CE] px-4 py-2 text-sm font-semibold text-white hover:bg-[#005ba3]"
          >
            Start Browsing
          </Link>
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4">
          {visibleRows.map(({ listing, dropped, dropPercent }) => (
            <div key={listing.id} className="space-y-2">
              {dropped && (
                <div className="rounded-xl border border-emerald-200 bg-emerald-50 px-2 py-1 text-center text-xs font-semibold text-emerald-700">
                  Price dropped {dropPercent}%
                </div>
              )}
              <ListingCard
                listing={listing}
                onFavoriteChange={(id, watched) => {
                  if (!watched) {
                    setRemovedIds((prev) => (prev.includes(id) ? prev : [...prev, id]));
                  }
                }}
              />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
