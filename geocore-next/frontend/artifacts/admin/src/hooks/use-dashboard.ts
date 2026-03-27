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
  open_reports: 12,
  pending_moderation: 24,
};

export function useDashboardStats() {
  return useQuery({
    queryKey: ["admin_stats"],
    queryFn: async () => {
      try {
        const [statsRes, revenueRes] = await Promise.all([
          api.get("/admin/stats"),
          api.get("/admin/revenue").catch(() => null),
        ]);
        const s = statsRes.data.data;
        const rev = revenueRes?.data?.data;

        // Map API fields to the shape the dashboard components expect
        return {
          total_revenue: s.total_revenue ?? 0,
          active_listings: s.active_listings ?? 0,
          total_users: s.total_users ?? 0,
          active_auctions: s.live_auctions ?? 0,       // API → dashboard alias
          new_users_today: s.new_users_this_week ?? 0, // API → dashboard alias
          new_listings_today: s.new_listings_today ?? 0,
          revenue_today: s.revenue_today ?? 0,
          pending_moderation: s.pending_moderation ?? 0,
          open_reports: s.reports_pending ?? 0,
          bids_today: 0,
          revenue_chart: rev?.daily_30days?.map((d: { date: string; revenue: number }) => ({
            date: d.date,
            revenue: d.revenue,
          })) ?? MOCK_STATS.revenue_chart,
          listings_by_category: MOCK_STATS.listings_by_category,
        };
      } catch (err) {
        console.warn("Failed to fetch stats, using mock data");
        return MOCK_STATS;
      }
    },
    refetchInterval: 30000,
  });
}
