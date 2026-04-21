'use client';

import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { ShoppingCart } from 'lucide-react';
import api from '@/lib/api';
import { useAuthStore } from '@/store/auth';

interface CartData {
  item_count: number;
}

interface CartResponse {
  data: CartData;
}

export function CartIcon() {
  const { isAuthenticated } = useAuthStore();

  const { data } = useQuery<CartResponse>({
    queryKey: ['cart', 'summary'],
    queryFn: async () => {
      const res = await api.get('/cart');
      return res.data as CartResponse;
    },
    enabled: isAuthenticated,
    retry: false,
    refetchOnWindowFocus: true,
    refetchInterval: 10000,
  });

  const count = data?.data?.item_count ?? 0;

  return (
    <Link href="/cart" className="relative flex items-center text-white/80 hover:text-[#FFC220] transition-colors" aria-label="Open cart">
      <ShoppingCart size={13} />
      {isAuthenticated && count > 0 && (
        <span className="absolute -top-1.5 -right-2 min-w-4 h-4 px-1 rounded-full bg-[#FFC220] text-[10px] font-bold text-gray-900 flex items-center justify-center">
          {count > 99 ? '99+' : count}
        </span>
      )}
    </Link>
  );
}
