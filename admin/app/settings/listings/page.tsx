import { permanentRedirect } from "next/navigation";

export default function ListingsSettingsLegacyPage() {
  permanentRedirect("/admin/settings/listings");
}
