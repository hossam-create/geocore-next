import { permanentRedirect } from "next/navigation";

export default function TicketsLegacyPage() {
  permanentRedirect("/support/tickets");
}
