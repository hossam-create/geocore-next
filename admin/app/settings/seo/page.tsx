import { permanentRedirect } from "next/navigation";

export default function SeoSettingsLegacyPage() {
  permanentRedirect("/admin/settings/seo");
}
