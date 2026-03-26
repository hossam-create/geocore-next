import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

const MOCK_SETTINGS = {
  general: {
    site_name: "GeoCore Marketplace",
    tagline: "The premium GCC marketplace",
    currency: "AED",
    language: "en",
    maintenance_mode: false,
    contact_email: "support@geocore.app",
    max_price: 10000000
  },
  listing_rules: {
    require_approval: true,
    max_images: 10,
    duration_days: 30,
    allowed_countries: ["UAE", "KSA", "Kuwait"]
  }
};

export function useSettings() {
  return useQuery({
    queryKey: ["admin_settings"],
    queryFn: async () => {
      try {
        const res = await api.get("/admin/settings");
        return res.data.data;
      } catch (err) {
        return MOCK_SETTINGS;
      }
    }
  });
}

export function useSaveSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: any) => api.post("/admin/settings", data).catch(() => true),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["admin_settings"] })
  });
}
