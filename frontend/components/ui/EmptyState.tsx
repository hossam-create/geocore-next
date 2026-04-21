'use client';

import Link from 'next/link';
import { Search, Plus, Gavel, Truck, ShoppingBag, Package } from 'lucide-react';

interface EmptyStateProps {
  type: 'search' | 'listings' | 'orders' | 'offers' | 'requests' | 'wallet' | 'custom';
  title?: string;
  description?: string;
  actionLabel?: string;
  actionHref?: string;
  icon?: typeof Search;
}

const DEFAULTS: Record<string, { icon: typeof Search; title: string; description: string; actionLabel: string; actionHref: string }> = {
  search: { icon: Search, title: 'No results found', description: 'Try adjusting your search or filters.', actionLabel: 'Browse Listings', actionHref: '/listings' },
  listings: { icon: ShoppingBag, title: 'No listings yet', description: 'Start selling by creating your first listing.', actionLabel: 'Create Listing', actionHref: '/sell' },
  orders: { icon: Package, title: 'No orders yet', description: 'Start browsing listings and place your first order.', actionLabel: 'Browse Listings', actionHref: '/listings' },
  offers: { icon: Gavel, title: 'No offers yet', description: 'Make an offer on a listing to start negotiating.', actionLabel: 'Browse Listings', actionHref: '/listings' },
  requests: { icon: Truck, title: 'No delivery requests', description: 'Create a delivery request to find travelers.', actionLabel: 'Create Request', actionHref: '/requests' },
  wallet: { icon: ShoppingBag, title: 'No transactions yet', description: 'Add funds to get started.', actionLabel: 'Add Funds', actionHref: '/wallet' },
};

export function EmptyState({ type, title, description, actionLabel, actionHref, icon }: EmptyStateProps) {
  const defaults = DEFAULTS[type] || DEFAULTS.search;
  const Icon = icon || defaults.icon;
  const t = title || defaults.title;
  const d = description || defaults.description;
  const aLabel = actionLabel || defaults.actionLabel;
  const aHref = actionHref || defaults.actionHref;

  return (
    <div className="text-center py-16 px-4">
      <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-gray-50 mb-4">
        <Icon size={28} className="text-gray-300" />
      </div>
      <p className="text-lg font-semibold text-gray-700">{t}</p>
      <p className="text-sm text-gray-400 mt-1 max-w-xs mx-auto">{d}</p>
      {aHref && (
        <Link
          href={aHref}
          className="mt-5 inline-flex items-center gap-2 rounded-xl bg-[#0071CE] px-5 py-2.5 text-sm font-semibold text-white hover:bg-[#005ba3] transition-colors"
        >
          <Plus size={14} /> {aLabel}
        </Link>
      )}
    </div>
  );
}
