'use client';

import { Eye, Heart, Gavel } from 'lucide-react';

interface ConversionSignalsProps {
  watchersCount?: number;
  viewsToday?: number;
  offersCount?: number;
  bidCount?: number;
}

export function ConversionSignals({ watchersCount, viewsToday, offersCount, bidCount }: ConversionSignalsProps) {
  const hasData = watchersCount != null || viewsToday != null || offersCount != null || bidCount != null;
  if (!hasData) return null;

  return (
    <div className="flex flex-wrap gap-2 mt-3">
      {viewsToday != null && viewsToday > 0 && (
        <span className="inline-flex items-center gap-1 text-xs text-gray-500 bg-gray-50 px-2.5 py-1 rounded-full">
          <Eye size={12} /> {viewsToday} views today
        </span>
      )}
      {watchersCount != null && watchersCount > 0 && (
        <span className="inline-flex items-center gap-1 text-xs text-amber-700 bg-amber-50 px-2.5 py-1 rounded-full">
          <Heart size={12} /> {watchersCount} watching
        </span>
      )}
      {offersCount != null && offersCount > 0 && (
        <span className="inline-flex items-center gap-1 text-xs text-blue-700 bg-blue-50 px-2.5 py-1 rounded-full">
          <Gavel size={12} /> {offersCount} offer{offersCount !== 1 ? 's' : ''}
        </span>
      )}
      {bidCount != null && bidCount > 0 && (
        <span className="inline-flex items-center gap-1 text-xs text-rose-700 bg-rose-50 px-2.5 py-1 rounded-full">
          <Gavel size={12} /> {bidCount} bid{bidCount !== 1 ? 's' : ''}
        </span>
      )}
    </div>
  );
}
