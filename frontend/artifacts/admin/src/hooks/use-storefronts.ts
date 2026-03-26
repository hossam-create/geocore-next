import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

const MOCK_STORES = Array.from({ length: 8 }).map((_, i) => ({
  id: `store_${i}`,
  name: `Premium Store ${i + 1}`,
  slug: `premium-store-${i + 1}`,
  owner: { name: "Ahmed Business" },
  listings_count: Math.floor(Math.random() * 200) + 10,
  rating: (Math.random() * 2 + 3).toFixed(1),
  location: "Dubai, UAE",
  status: i === 2 ? "suspended" : "active",
  is_featured: i === 0
}));

export function useStorefronts(search: string, page: number) {
  return useQuery({
    queryKey: ["admin_storefronts", search, page],
    queryFn: async () => {
      try {
        const res = await api.get(`/admin/storefronts?q=${search}&page=${page}`);
        return res.data;
      } catch (err) {
        return {
          data: MOCK_STORES,
          meta: { total: 45, current_page: page, last_page: 5 }
        };
      }
    }
  });
}

export function useStorefrontActions() {
  const queryClient = useQueryClient();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["admin_storefronts"] });

  return {
    toggleStatus: useMutation({
      mutationFn: ({ id, suspend }: { id: string, suspend: boolean }) => 
        api.post(`/admin/storefronts/${id}/${suspend ? 'suspend' : 'activate'}`).catch(() => true),
      onSuccess: invalidate
    }),
    toggleFeature: useMutation({
      mutationFn: (id: string) => api.post(`/admin/storefronts/${id}/feature`).catch(() => true),
      onSuccess: invalidate
    })
  };
}
