import { permanentRedirect } from "next/navigation";

export default function OrdersLegacyPage() {
  permanentRedirect("/operations/orders");
}
