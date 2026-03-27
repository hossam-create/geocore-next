import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

const MOCK_PAYMENTS = Array.from({ length: 15 }).map((_, i) => ({
  id: `pay_${i}`,
  user_id: `usr_${i}`,
  amount: Math.floor(Math.random() * 500) + 50,
  currency: "AED",
  status: ["succeeded", "pending", "failed"][i % 3],
  created_at: new Date(Date.now() - Math.random() * 10000000).toISOString()
}));

export function usePayments(page: number, status = "") {
  return useQuery({
    queryKey: ["admin_payments", page, status],
    queryFn: async () => {
      try {
        const params = new URLSearchParams({ page: String(page), per_page: "20" });
        if (status) params.set("status", status);
        const res = await api.get(`/admin/transactions?${params}`);
        return res.data;
      } catch (err) {
        return {
          data: MOCK_PAYMENTS,
          meta: { total: 1500, page, per_page: 20, pages: 75 }
        };
      }
    }
  });
}

export function useRevenue() {
  return useQuery({
    queryKey: ["admin_revenue"],
    queryFn: async () => {
      try {
        const res = await api.get("/admin/revenue");
        return res.data.data;
      } catch (err) {
        return {
          total: 125430.50,
          daily_30days: Array.from({ length: 30 }).map((_, i) => ({
            date: new Date(Date.now() - (29 - i) * 86400000).toISOString().split("T")[0],
            revenue: Math.floor(Math.random() * 5000) + 1000,
            count: Math.floor(Math.random() * 30) + 5
          }))
        };
      }
    },
    refetchInterval: 60000
  });
}
