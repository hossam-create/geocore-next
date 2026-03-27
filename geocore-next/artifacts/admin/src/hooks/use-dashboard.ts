import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

const MOCK_STATS = {
  total_revenue: 125430.50,
  active_listings: 1452,
  total_users: 8432,
  active_auctions: 124,
  revenue_chart: Array.from({ length: 30 }).map((_, i) => ({
    date: new Date(Date.now() - (29 - i) * 86400000).toISOString().split('T')[0],
    revenue: Math.floor(Math.random() * 5000) + 1000
  })),
  listings_by_category: [
    { name: "Electronics", value: 400 },
    { name: "Real Estate", value: 300 },
    { name: "Vehicles", value: 350 },
    { name: "Fashion", value: 200 },
    { name: "Home", value: 202 },
  ],
  new_users_today: 45,
  new_listings_today: 112,
  bids_today: 342,
  revenue_today: 4250.00,
  open_reports: 12
};

export function useDashboardStats() {
  return useQuery({
    queryKey: ["admin_stats"],
    queryFn: async () => {
      try {
        const res = await api.get("/admin/stats");
        return res.data.data;
      } catch (err) {
        console.warn("Failed to fetch stats, using mock data");
        return MOCK_STATS;
      }
    },
    refetchInterval: 30000,
  });
}
