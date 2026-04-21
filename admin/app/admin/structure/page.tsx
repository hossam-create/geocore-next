"use client";

import PageHeader from "@/components/shared/PageHeader";
import { ADMIN_DEPARTMENTS } from "@/lib/adminAccess";

export default function AdminStructurePage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="Administrative Structure"
        description="Role-based hierarchy, responsibilities, and dedicated access profiles"
      />

      <div className="surface p-5">
        <h3 className="text-sm font-semibold mb-3" style={{ color: "var(--text-primary)" }}>
          Governance Model
        </h3>
        <ul className="space-y-2 text-sm" style={{ color: "var(--text-secondary)" }}>
          <li>- Level 1: Super Admin (cross-department governance).</li>
          <li>- Level 2: Admin (platform-wide operations and policy execution).</li>
          <li>- Level 3: Department Admins (Ops / Finance / Support).</li>
          <li>- Access is role-scoped and enforceable by backend RBAC middleware.</li>
        </ul>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {ADMIN_DEPARTMENTS.map((dept) => (
          <div key={dept.key} className="surface p-5">
            <div className="flex items-center justify-between mb-2">
              <h4 className="font-semibold" style={{ color: "var(--text-primary)" }}>
                {dept.title}
              </h4>
              <span className="text-xs px-2 py-1 rounded-md" style={{ background: "var(--bg-surface-active)", color: "var(--text-secondary)" }}>
                {dept.key}
              </span>
            </div>

            <p className="text-sm mb-3" style={{ color: "var(--text-secondary)" }}>
              {dept.description}
            </p>

            <div className="text-xs mb-2" style={{ color: "var(--text-tertiary)" }}>
              Dedicated login account pattern
            </div>
            <div className="text-sm font-mono mb-4" style={{ color: "var(--color-brand)" }}>
              {dept.loginHint}
            </div>

            <div className="text-xs mb-2" style={{ color: "var(--text-tertiary)" }}>
              Main permissions
            </div>
            <div className="flex flex-wrap gap-2">
              {dept.permissions.map((perm) => (
                <span
                  key={perm}
                  className="text-xs font-mono px-2 py-1 rounded"
                  style={{ background: "var(--bg-surface-active)", color: "var(--text-secondary)" }}
                >
                  {perm}
                </span>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
