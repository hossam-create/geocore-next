import { permanentRedirect } from "next/navigation";

export default function UsersLegacyPage() {
  permanentRedirect("/admin/users");
}
