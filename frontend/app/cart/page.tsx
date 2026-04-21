'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import api from '@/lib/api';
import { useAuthStore } from '@/store/auth';
import { formatPrice } from '@/lib/utils';
import { CartItem } from '@/components/cart/CartItem';

interface CartItemData {
  listing_id: string;
  title: string;
  image_url?: string;
  currency: string;
  unit_price: number;
  quantity: number;
  subtotal: number;
}

interface CartData {
  items: CartItemData[];
  item_count: number;
  total: number;
  currency?: string;
}

interface CartResponse {
  data: CartData;
}

export default function CartPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const { isAuthenticated } = useAuthStore();
  const [removingID, setRemovingID] = useState('');

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login?next=/cart');
    }
  }, [isAuthenticated, router]);

  const { data, isLoading, isError } = useQuery<CartResponse>({
    queryKey: ['cart', 'items'],
    queryFn: async () => {
      const res = await api.get('/cart');
      return res.data as CartResponse;
    },
    enabled: isAuthenticated,
    retry: false,
  });

  const removeItem = useMutation({
    mutationFn: async (listingID: string) => api.delete(`/cart/items/${listingID}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['cart'] });
    },
    onSettled: () => setRemovingID(''),
  });

  if (!isAuthenticated) return null;

  const cart = data?.data ?? { items: [], item_count: 0, total: 0, currency: 'AED' };
  const currency = cart.currency || cart.items[0]?.currency || 'AED';

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">My Cart</h1>
          <p className="text-sm text-gray-500">Review your items before checkout.</p>
        </div>
        <Link href="/listings" className="rounded-xl border border-gray-200 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50">
          Browse Listings
        </Link>
      </div>

      <div className="grid gap-5 lg:grid-cols-3">
        <div className="overflow-hidden rounded-2xl border border-gray-100 bg-white shadow-sm lg:col-span-2">
          {isLoading ? (
            <div className="space-y-2 p-5">{[1, 2, 3].map((i) => <div key={i} className="h-16 animate-pulse rounded-xl bg-gray-100" />)}</div>
          ) : isError ? (
            <div className="p-8 text-center text-sm text-red-500">Could not load your cart. Please try again.</div>
          ) : cart.items.length === 0 ? (
            <div className="p-12 text-center">
              <p className="text-base font-semibold text-gray-700">Your cart is empty</p>
              <p className="mt-1 text-sm text-gray-500">Add products to continue to checkout.</p>
              <Link href="/listings" className="mt-4 inline-block rounded-xl bg-[#0071CE] px-4 py-2 text-sm font-semibold text-white hover:bg-[#005ba3]">
                Browse Listings
              </Link>
            </div>
          ) : (
            <ul className="divide-y divide-gray-100">
              {cart.items.map((item) => (
                <CartItem
                  key={item.listing_id}
                  item={item}
                  removing={removeItem.isPending && removingID === item.listing_id}
                  onRemove={(listingID) => {
                    setRemovingID(listingID);
                    removeItem.mutate(listingID);
                  }}
                />
              ))}
            </ul>
          )}
        </div>

        <aside className="rounded-2xl border border-gray-200 bg-white p-5">
          <h2 className="mb-4 text-sm font-bold text-gray-800">Order Summary</h2>
          <div className="space-y-2 border-b border-gray-100 pb-4 text-sm">
            <div className="flex items-center justify-between text-gray-600">
              <span>Items</span>
              <span>{cart.item_count}</span>
            </div>
            <div className="flex items-center justify-between text-gray-900 font-semibold">
              <span>Subtotal</span>
              <span>{formatPrice(cart.total, currency)}</span>
            </div>
          </div>

          <button
            onClick={() => router.push('/checkout?from=cart')}
            disabled={cart.items.length === 0}
            className="mt-4 w-full rounded-xl bg-[#0071CE] px-4 py-3 text-sm font-bold text-white hover:bg-[#005ba3] disabled:cursor-not-allowed disabled:opacity-50"
          >
            Proceed to Checkout
          </button>
          <p className="mt-2 text-xs text-gray-400">Secure payment via Stripe escrow.</p>
        </aside>
      </div>
    </div>
  );
}
