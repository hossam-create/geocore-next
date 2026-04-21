import { permanentRedirect } from "next/navigation";

export default function DisputesLegacyPage() {
  permanentRedirect("/support/disputes");
}
