"use client";

import PageHeader from "@/components/shared/PageHeader";
import { ADMIN_DEPARTMENTS } from "@/lib/adminAccess";

export default function RolesPage() {
  return (
    <div>
      <PageHeader title="Roles & Permissions" description="RBAC matrix for team access control" />
      <div className="space-y-3">
        {ADMIN_DEPARTMENTS.map((r) => (
          <div key={r.key} className="surface p-4">
            <div className="flex items-center justify-between mb-2">
              <p className="font-semibold capitalize" style={{ color: "var(--text-primary)" }}>{r.key.replace(/_/g, " ")}</p>
              <span className="text-xs" style={{ color: "var(--text-tertiary)" }}>{r.accessLevel}</span>
            </div>
            <div className="flex flex-wrap gap-2">
              {r.permissions.map((p) => (
                <span key={p} className="px-2 py-1 rounded-md text-xs font-mono" style={{ background: "var(--bg-surface-active)", color: "var(--text-secondary)" }}>
                  {p}
                </span>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
