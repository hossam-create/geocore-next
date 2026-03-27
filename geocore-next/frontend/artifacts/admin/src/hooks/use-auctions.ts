import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { toast } from "./use-toast";

const MOCK_AUCTIONS = Array.from({ length: 12 }).map((_, i) => ({
  id: `auc_${i}`,
  title: `Rare Collectible Item ${i + 1}`,
  type: ["dutch", "reverse", "standard"][i % 3],
  seller: { name: "VIP Seller" },
  start_price: 1000,
  current_bid: 1000 + (i * 150),
  currency: "AED",
  bids_count: Math.floor(Math.random() * 20),
  status: ["live", "upcoming", "ended"][i % 3],
  ends_at: new Date(Date.now() + 86400000 * (i - 2)).toISOString()
}));

export function useAuctions(status: string, page: number) {
  return useQuery({
    queryKey: ["admin_auctions", status, page],
    queryFn: async () => {
      try {
        const res = await api.get(`/admin/auctions?status=${status}&page=${page}`);
        return res.data;
      } catch (err) {
        return {
          data: status === "all" ? MOCK_AUCTIONS : MOCK_AUCTIONS.filter(a => a.status === status),
          meta: { total: 120, current_page: page, last_page: 5 }
        };
      }
    },
    refetchInterval: 10000
  });
}

export function useAuctionActions() {
  const queryClient = useQueryClient();
  return {
    endNow: useMutation({
      mutationFn: (id: string) => api.post(`/admin/auctions/${id}/end`),
      onSuccess: () => queryClient.invalidateQueries({ queryKey: ["admin_auctions"] }),
      onError: (err: any) => toast({
        title: "Could not end auction",
        description: err?.response?.data?.error || "Please try again.",
        variant: "destructive",
      }),
    }),
    deleteAuction: useMutation({
      mutationFn: (id: string) => api.delete(`/admin/auctions/${id}`),
      onSuccess: () => queryClient.invalidateQueries({ queryKey: ["admin_auctions"] }),
      onError: (err: any) => toast({
        title: "Could not delete auction",
        description: err?.response?.data?.error || "Please try again.",
        variant: "destructive",
      }),
    })
  };
}
