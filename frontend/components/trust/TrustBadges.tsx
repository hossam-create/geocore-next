'use client';

import { ShieldCheck, Star, AlertTriangle, Truck } from 'lucide-react';

interface TrustBadgesProps {
  isVerifiedSeller?: boolean;
  isTrustedTraveler?: boolean;
  rating?: number;
  reviewCount?: number;
  trustLevel?: 'high' | 'medium' | 'low';
  size?: 'sm' | 'md';
}

export function TrustBadges({ isVerifiedSeller, isTrustedTraveler, rating, reviewCount, trustLevel, size = 'sm' }: TrustBadgesProps) {
  const sz = size === 'sm' ? 'text-[10px] px-2 py-0.5' : 'text-xs px-2.5 py-1';
  const iconSz = size === 'sm' ? 11 : 13;

  return (
    <div className="flex flex-wrap gap-1.5">
      {isVerifiedSeller && (
        <span className={`inline-flex items-center gap-1 font-bold rounded-full bg-blue-50 text-blue-700 border border-blue-200 ${sz}`}>
          <ShieldCheck size={iconSz} /> Verified Seller
        </span>
      )}
      {isTrustedTraveler && (
        <span className={`inline-flex items-center gap-1 font-bold rounded-full bg-emerald-50 text-emerald-700 border border-emerald-200 ${sz}`}>
          <Truck size={iconSz} /> Trusted Traveler
        </span>
      )}
      {rating != null && rating > 0 && (
        <span className={`inline-flex items-center gap-1 font-semibold rounded-full bg-amber-50 text-amber-700 border border-amber-200 ${sz}`}>
          <Star size={iconSz} fill="#FFC220" className="text-[#FFC220]" />
          {rating.toFixed(1)}{reviewCount != null ? ` (${reviewCount})` : ''}
        </span>
      )}
      {trustLevel === 'low' && (
        <span className={`inline-flex items-center gap-1 font-bold rounded-full bg-amber-50 text-amber-700 border border-amber-200 ${sz}`}>
          <AlertTriangle size={iconSz} /> Low trust
        </span>
      )}
    </div>
  );
}

interface SellerTrustCardProps {
  sellerName: string;
  rating?: number;
  reviewCount?: number;
  isVerified?: boolean;
  trustLevel?: 'high' | 'medium' | 'low';
  memberSince?: string;
}

export function SellerTrustCard({ sellerName, rating, reviewCount, isVerified, trustLevel, memberSince }: SellerTrustCardProps) {
  return (
    <div className="rounded-xl border border-gray-200 bg-white p-4 space-y-2">
      <p className="text-sm font-semibold text-gray-800">Seller Trust</p>
      <TrustBadges
        isVerifiedSeller={isVerified}
        rating={rating}
        reviewCount={reviewCount}
        trustLevel={trustLevel}
        size="md"
      />
      {memberSince && (
        <p className="text-xs text-gray-400">Member since {new Date(memberSince).getFullYear()}</p>
      )}
      {trustLevel === 'low' && (
        <p className="text-xs text-amber-600 bg-amber-50 rounded-lg p-2 flex items-start gap-1.5">
          <AlertTriangle size={13} className="shrink-0 mt-0.5" />
          This seller has limited transaction history. Proceed with caution.
        </p>
      )}
    </div>
  );
}
