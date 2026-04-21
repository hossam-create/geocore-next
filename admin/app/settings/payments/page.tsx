import { permanentRedirect } from "next/navigation";

export default function PaymentsSettingsLegacyPage() {
  permanentRedirect("/admin/settings/payments");
}
