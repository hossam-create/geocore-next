'use client';

import { useQuery } from '@tanstack/react-query';
import api from '@/lib/api';
import { formatPrice } from '@/lib/utils';
import { Truck, Users, MapPin } from 'lucide-react';

interface CrowdshippingWidgetProps {
  listingId?: string;
  sellerCity?: string;
  currency?: string;
}

interface TravelerAvailability {
  travelers_available: number;
  estimated_price: number;
  estimated_days: number;
  currency: string;
}

export function CrowdshippingWidget({ listingId, sellerCity, currency = 'AED' }: CrowdshippingWidgetProps) {
  const { data, isLoading } = useQuery<TravelerAvailability>({
    queryKey: ['crowdshipping-availability', listingId],
    queryFn: () => api.get(`/crowdshipping/availability?listing_id=${listingId}`).then(r => r.data.data),
    enabled: Boolean(listingId),
    retry: false,
    staleTime: 60_000,
  });

  if (isLoading || (!data && !sellerCity)) return null;
  if (!data && !sellerCity) return null;

  const travelersCount = data?.travelers_available ?? 0;
  const estimatedPrice = data?.estimated_price;
  const estimatedDays = data?.estimated_days;

  if (travelersCount === 0 && !estimatedPrice) return null;

  // Build strong social proof headline
  const headline = (() => {
    if (travelersCount > 0 && estimatedDays != null && estimatedDays > 0) {
      return `${travelersCount} Traveler${travelersCount !== 1 ? 's' : ''} ready to deliver in ${estimatedDays} day${estimatedDays !== 1 ? 's' : ''}`;
    }
    if (travelersCount > 0) {
      return `${travelersCount} Traveler${travelersCount !== 1 ? 's' : ''} ready to deliver`;
    }
    return 'Crowdshipping Available';
  })();

  return (
    <div className="mt-4 rounded-xl border border-emerald-200 bg-emerald-50 p-4">
      <div className="flex items-center gap-2 mb-2">
        <Truck size={16} className="text-emerald-600" />
        <p className="text-sm font-semibold text-emerald-800">{headline}</p>
      </div>
      <div className="flex flex-wrap gap-3 text-xs text-emerald-700">
        {travelersCount > 0 && (
          <span className="inline-flex items-center gap-1">
            <Users size={12} /> {travelersCount} traveler{travelersCount !== 1 ? 's' : ''} nearby
          </span>
        )}
        {estimatedPrice != null && estimatedPrice > 0 && (
          <span className="inline-flex items-center gap-1">
            <MapPin size={12} /> Est. delivery: {formatPrice(estimatedPrice, data?.currency || currency)}
          </span>
        )}
        {estimatedDays != null && estimatedDays > 0 && (
          <span>~{estimatedDays} day{estimatedDays !== 1 ? 's' : ''}</span>
        )}
      </div>
    </div>
  );
}
