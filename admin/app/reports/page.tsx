import { permanentRedirect } from "next/navigation";

export default function ReportsLegacyPage() {
  permanentRedirect("/analytics/reports");
}
