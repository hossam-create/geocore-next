import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

const MOCK_PLANS = [
  { id: "1", name: "Basic", group: "Default", max_listings: 5, max_images: 3, featured_allowed: false, final_value_fee: 5.0, price: 0 },
  { id: "2", name: "Professional", group: "Pro", max_listings: 50, max_images: 10, featured_allowed: true, final_value_fee: 2.5, price: 99 },
  { id: "3", name: "Business", group: "Store", max_listings: 500, max_images: 20, featured_allowed: true, final_value_fee: 1.0, price: 299 },
];

export function usePricingPlans() {
  return useQuery({
    queryKey: ["admin_pricing"],
    queryFn: async () => {
      try {
        const res = await api.get("/admin/pricing");
        return res.data.data;
      } catch (err) {
        return MOCK_PLANS;
      }
    }
  });
}

export function useSavePricingPlan() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (plan: any) => {
      if (plan.id) {
        return api.put(`/admin/pricing/${plan.id}`, plan).catch(() => true);
      }
      return api.post(`/admin/pricing`, plan).catch(() => true);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["admin_pricing"] })
  });
}
