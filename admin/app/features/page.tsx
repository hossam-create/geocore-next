import { permanentRedirect } from "next/navigation";

export default function FeaturesLegacyPage() {
  permanentRedirect("/admin/features");
}
