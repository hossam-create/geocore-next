import { useQuery } from "@tanstack/react-query";
import axios from "axios";

export interface CategoryField {
  id: string;
  category_id: string;
  name: string;
  label: string;
  label_en: string;
  field_type: "text" | "number" | "select" | "boolean" | "range" | "date";
  options: string; // JSON string of [{value, label}]
  is_required: boolean;
  placeholder: string;
  unit: string;
  sort_order: number;
}

export function useCategoryFields(categoryId: string | null | undefined) {
  return useQuery<CategoryField[]>({
    queryKey: ["category-fields", categoryId],
    queryFn: async () => {
      if (!categoryId) return [];
      const { data } = await axios.get(`/api/v1/categories/${categoryId}/fields`);
      return data.data ?? [];
    },
    enabled: !!categoryId,
    staleTime: 60_000,
  });
}
