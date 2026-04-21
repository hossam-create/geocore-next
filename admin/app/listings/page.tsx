import { permanentRedirect } from "next/navigation";

export default function ListingsLegacyPage() {
  permanentRedirect("/operations/listings");
}
