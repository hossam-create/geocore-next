'use client';

import Link from 'next/link';
import { Trash2 } from 'lucide-react';
import { formatPrice } from '@/lib/utils';

interface CartItemData {
  listing_id: string;
  title: string;
  image_url?: string;
  currency: string;
  unit_price: number;
  quantity: number;
  subtotal: number;
}

interface CartItemProps {
  item: CartItemData;
  onRemove: (listingID: string) => void;
  removing: boolean;
}

export function CartItem({ item, onRemove, removing }: CartItemProps) {
  const thumb = item.image_url || `https://picsum.photos/seed/${item.listing_id}/140/140`;

  return (
    <li className="flex items-center gap-4 px-5 py-4 hover:bg-gray-50 transition-colors">
      <img src={thumb} alt={item.title} className="h-16 w-16 rounded-xl object-cover bg-gray-100" />

      <div className="min-w-0 flex-1">
        <Link href={`/listings/${item.listing_id}`} className="block truncate text-sm font-semibold text-gray-900 hover:text-[#0071CE]">
          {item.title}
        </Link>
        <p className="mt-1 text-xs text-gray-500">Qty: {item.quantity}</p>
        <p className="text-xs text-gray-500">Unit: {formatPrice(item.unit_price, item.currency || 'AED')}</p>
      </div>

      <div className="text-right">
        <p className="text-sm font-bold text-gray-900">{formatPrice(item.subtotal, item.currency || 'AED')}</p>
        <button
          onClick={() => onRemove(item.listing_id)}
          disabled={removing}
          className="mt-2 inline-flex items-center gap-1 rounded-lg border border-red-100 px-2 py-1 text-xs text-red-600 hover:bg-red-50 disabled:opacity-50"
        >
          <Trash2 className="h-3.5 w-3.5" /> Remove
        </button>
      </div>
    </li>
  );
}
