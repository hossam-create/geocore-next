import { permanentRedirect } from "next/navigation";

export default function AuctionsSettingsLegacyPage() {
  permanentRedirect("/admin/settings/auctions");
}
