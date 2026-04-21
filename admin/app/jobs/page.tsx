import { permanentRedirect } from "next/navigation";

export default function JobsLegacyPage() {
  permanentRedirect("/system/jobs");
}
