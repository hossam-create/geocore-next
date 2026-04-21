import { permanentRedirect } from "next/navigation";

export default function EmailSettingsLegacyPage() {
  permanentRedirect("/admin/settings/email");
}
