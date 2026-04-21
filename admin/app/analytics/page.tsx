import { permanentRedirect } from "next/navigation";

export default function AnalyticsLegacyPage() {
  permanentRedirect("/analytics/overview");
}
