import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

const MOCK_CATEGORIES = [
  { id: "1", name_en: "Electronics", name_ar: "إلكترونيات", icon: "📱", active: true, custom_fields: [
    { id: "f1", key: "brand", label_en: "Brand", label_ar: "الماركة", type: "select", required: true, options: "Apple,Samsung,Sony" }
  ] },
  { id: "2", name_en: "Vehicles", name_ar: "مركبات", icon: "🚗", active: true, custom_fields: [] },
  { id: "3", name_en: "Real Estate", name_ar: "عقارات", icon: "🏠", active: true, custom_fields: [] },
];

export function useCategories() {
  return useQuery({
    queryKey: ["admin_categories"],
    queryFn: async () => {
      try {
        const res = await api.get("/admin/categories");
        return res.data.data;
      } catch (err) {
        return MOCK_CATEGORIES;
      }
    }
  });
}

export function useCategoryActions() {
  const queryClient = useQueryClient();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["admin_categories"] });

  return {
    saveCategory: useMutation({
      mutationFn: (data: any) => api.put(`/admin/categories/${data.id}`, data).catch(() => true),
      onSuccess: invalidate
    }),
    saveField: useMutation({
      mutationFn: (data: any) => api.post(`/admin/categories/${data.categoryId}/fields`, data).catch(() => true),
      onSuccess: invalidate
    }),
    deleteField: useMutation({
      mutationFn: ({ catId, fieldId }: any) => api.delete(`/admin/categories/${catId}/fields/${fieldId}`).catch(() => true),
      onSuccess: invalidate
    })
  };
}
