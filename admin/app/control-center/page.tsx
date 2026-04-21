"use client";

import Link from "next/link";
import { useAdminAuth } from "@/lib/auth";
import PageHeader from "@/components/shared/PageHeader";
import { ADMIN_DEPARTMENTS, getDepartmentByRole } from "@/lib/adminAccess";
import { ArrowRight, Building2, ShieldCheck } from "lucide-react";

export default function ControlCenterPage() {
  const { user } = useAdminAuth();
  const currentDept = getDepartmentByRole(user?.role);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Control Center"
        description="Unified administrative gateway for all management units"
      />

      <div className="surface p-5 flex items-start justify-between gap-4">
        <div>
          <p className="text-sm" style={{ color: "var(--text-secondary)" }}>
            Signed in as
          </p>
          <p className="text-lg font-semibold" style={{ color: "var(--text-primary)" }}>
            {user?.name ?? "Administrator"}
          </p>
          <p className="text-sm mt-1" style={{ color: "var(--text-tertiary)" }}>
            Role: {user?.role ?? "-"}
          </p>
        </div>

        {currentDept && (
          <Link
            href={currentDept.defaultRoute}
            className="inline-flex items-center gap-2 px-4 py-2 rounded-lg"
            style={{ background: "var(--bg-surface-active)", color: "var(--color-brand)" }}
          >
            Open My Workspace
            <ArrowRight className="w-4 h-4" />
          </Link>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {ADMIN_DEPARTMENTS.map((dept) => (
          <div key={dept.key} className="surface p-5">
            <div className="flex items-center justify-between mb-3">
              <div className="inline-flex items-center gap-2">
                <Building2 className="w-4 h-4" style={{ color: "var(--color-brand)" }} />
                <h3 className="font-semibold" style={{ color: "var(--text-primary)" }}>
                  {dept.title}
                </h3>
              </div>
              <span className="text-xs px-2 py-1 rounded-md" style={{ background: "var(--bg-surface-active)", color: "var(--text-secondary)" }}>
                {dept.accessLevel}
              </span>
            </div>

            <p className="text-sm mb-3" style={{ color: "var(--text-secondary)" }}>
              {dept.description}
            </p>

            <p className="text-xs mb-2" style={{ color: "var(--text-tertiary)" }}>
              Dedicated login pattern: {dept.loginHint}
            </p>

            <div className="flex flex-wrap gap-2 mb-4">
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

            <Link
              href={dept.defaultRoute}
              className="inline-flex items-center gap-2 text-sm font-medium"
              style={{ color: "var(--color-brand)" }}
            >
              <ShieldCheck className="w-4 h-4" />
              Open workspace
            </Link>
          </div>
        ))}
      </div>
    </div>
  );
}
