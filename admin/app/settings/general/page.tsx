import { permanentRedirect } from "next/navigation";

export default function GeneralSettingsLegacyPage() {
  permanentRedirect("/admin/settings/general");
}
