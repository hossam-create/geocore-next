import { permanentRedirect } from "next/navigation";

export default function SettingsLegacyPage() {
  permanentRedirect("/admin/settings");
}
