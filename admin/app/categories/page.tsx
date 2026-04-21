import { permanentRedirect } from "next/navigation";

export default function CategoriesLegacyPage() {
  permanentRedirect("/admin/categories");
}
