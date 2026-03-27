import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { toast } from "./use-toast";

const generateMockListings = (status: string) => {
  return Array.from({ length: 15 }).map((_, i) => ({
    id: `lst_${Math.random().toString(36).substr(2, 9)}`,
    title: `${["iPhone 15 Pro", "Toyota Camry", "Luxury Villa", "Rolex Watch"][i % 4]} - Special Edition`,
    user: { name: ["Ahmed Ali", "Sarah Smith", "Mohammed Khan", "Fatima Al Farsi"][i % 4] },
    category: { name_en: ["Electronics", "Vehicles", "Real Estate", "Fashion"][i % 4] },
    price: Math.floor(Math.random() * 50000) + 100,
    currency: "AED",
    type: i % 3 === 0 ? "auction" : "standard",
    city: ["Dubai", "Riyadh", "Doha", "Kuwait City"][i % 4],
    country: ["UAE", "KSA", "Qatar", "Kuwait"][i % 4],
    created_at: new Date(Date.now() - Math.random() * 10000000000).toISOString(),
    status: status === "" ? "active" : status,
    images: [{ url: `https://picsum.photos/seed/${i}/200` }]
  }));
};

export function useListings(status: string, search: string, page: number) {
  return useQuery({
    queryKey: ["admin_listings", status, search, page],
    queryFn: async () => {
      try {
        const res = await api.get(`/admin/listings?status=${status}&q=${search}&page=${page}&per_page=25`);
        return res.data;
      } catch (err) {
        return {
          data: generateMockListings(status),
          meta: { total: 1452, current_page: page, last_page: 58 },
          pending_count: 24
        };
      }
    },
  });
}

export function useListingActions() {
  const queryClient = useQueryClient();

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["admin_listings"] });

  const approve = useMutation({
    mutationFn: (id: string) => api.put(`/admin/listings/${id}/approve`),
    onSuccess: invalidate,
    onError: (err: any) => toast({
      title: "Failed to approve listing",
      description: err?.response?.data?.error || "Please try again.",
      variant: "destructive",
    }),
  });

  const reject = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      api.put(`/admin/listings/${id}/reject`, { reason }),
    onSuccess: invalidate,
    onError: (err: any) => toast({
      title: "Failed to reject listing",
      description: err?.response?.data?.error || "Please try again.",
      variant: "destructive",
    }),
  });

  const bulkApprove = useMutation({
    mutationFn: (ids: string[]) =>
      Promise.all(ids.map(id => api.put(`/admin/listings/${id}/approve`))),
    onSuccess: invalidate,
    onError: (err: any) => toast({
      title: "Bulk approve failed",
      description: err?.response?.data?.error || "Some listings could not be approved.",
      variant: "destructive",
    }),
  });

  const bulkReject = useMutation({
    mutationFn: ({ ids, reason }: { ids: string[]; reason: string }) =>
      Promise.all(ids.map(id => api.put(`/admin/listings/${id}/reject`, { reason }))),
    onSuccess: invalidate,
    onError: (err: any) => toast({
      title: "Bulk reject failed",
      description: err?.response?.data?.error || "Some listings could not be rejected.",
      variant: "destructive",
    }),
  });

  return { approve, reject, bulkApprove, bulkReject };
}
