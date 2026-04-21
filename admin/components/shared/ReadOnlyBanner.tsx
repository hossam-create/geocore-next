"use client";

import { useAdminAuth } from "@/lib/auth";
import { Eye } from "lucide-react";

const WRITE_ROLES = ["super_admin", "admin", "ops_admin"];

export default function ReadOnlyBanner() {
  const user = useAdminAuth((s) => s.user);

  if (!user || WRITE_ROLES.includes(user.role)) return null;

  return (
    <div
      className="flex items-center gap-2 px-4 py-2 text-[13px] font-medium"
      style={{
        background: "var(--color-warning-light)",
        color: "#92400e",
        borderBottom: "1px solid #fcd34d",
      }}
    >
      <Eye className="w-4 h-4 shrink-0" />
      <span>You have <strong>read-only</strong> access. Contact a super admin to request write permissions.</span>
    </div>
  );
}
