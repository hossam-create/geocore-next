'use client';

interface UrgencyBadgeProps {
  watchersCount?: number;
  viewsToday?: number;
  offersCount?: number;
  endsAt?: string;
  isAuction?: boolean;
}

export function UrgencyBadge({ watchersCount, viewsToday, offersCount, endsAt, isAuction }: UrgencyBadgeProps) {
  const isHighDemand = (watchersCount != null && watchersCount >= 5) || (viewsToday != null && viewsToday >= 20) || (offersCount != null && offersCount >= 3);
  const isEndingSoon = endsAt != null && new Date(endsAt).getTime() - Date.now() > 0 && new Date(endsAt).getTime() - Date.now() < 24 * 60 * 60 * 1000;

  if (!isHighDemand && !isEndingSoon) return null;

  return (
    <div className="flex flex-wrap gap-2 mt-2">
      {isHighDemand && (
        <span className="inline-flex items-center gap-1 text-xs font-bold text-orange-700 bg-orange-50 border border-orange-200 px-2.5 py-1 rounded-full animate-pulse">
          🔥 High demand
        </span>
      )}
      {isEndingSoon && (
        <span className="inline-flex items-center gap-1 text-xs font-bold text-red-700 bg-red-50 border border-red-200 px-2.5 py-1 rounded-full">
          ⏳ Ending soon
        </span>
      )}
    </div>
  );
}
