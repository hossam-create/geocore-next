import { permanentRedirect } from "next/navigation";

export default function AuctionsLegacyPage() {
  permanentRedirect("/operations/auctions");
}
