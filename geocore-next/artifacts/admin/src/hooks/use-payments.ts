import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

const MOCK_PAYMENTS = Array.from({ length: 15 }).map((_, i) => ({
  id: `pay_${i}`,
  user: { name: `User ${i}`, email: `user${i}@mail.com` },
  type: ["listing_fee", "auction_fee", "deposit", "withdrawal"][i % 4],
  amount: Math.floor(Math.random() * 500) + 50,
  currency: "AED",
  status: ["completed", "pending", "failed"][i % 3],
  created_at: new Date(Date.now() - Math.random() * 10000000).toISOString()
}));

export function usePayments(page: number) {
  return useQuery({
    queryKey: ["admin_payments", page],
    queryFn: async () => {
      try {
        const res = await api.get(`/admin/payments?page=${page}`);
        return res.data;
      } catch (err) {
        return {
          summary: {
            total_revenue: 125430.50,
            this_month: 24500.00,
            this_week: 5600.00,
            avg_transaction: 145.50
          },
          monthly_chart: Array.from({ length: 12 }).map((_, i) => ({
            month: new Date(2023, i, 1).toLocaleString('default', { month: 'short' }),
            revenue: Math.floor(Math.random() * 30000) + 10000
          })),
          data: MOCK_PAYMENTS,
          meta: { total: 1500, current_page: page, last_page: 60 }
        };
      }
    }
  });
}
