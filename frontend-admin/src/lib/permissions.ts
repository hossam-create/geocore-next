export const PERMISSIONS = {
  ADMIN_DASHBOARD_READ: "admin.dashboard.read",
  USERS_READ: "users.read",
  USERS_WRITE: "users.write",
  USERS_DELETE: "users.delete",
  USERS_BAN: "users.ban",
  LISTINGS_MODERATE: "listings.moderate",
  LISTINGS_DELETE: "listings.delete",
  FINANCE_READ: "finance.read",
  AUDIT_LOGS_READ: "audit.logs.read",
  CATALOG_MANAGE: "catalog.manage",
  PLANS_MANAGE: "plans.manage",
  REPORTS_REVIEW: "reports.review",
  OPS_READ: "ops.read",
  OPS_MANAGE: "ops.manage",
  SETTINGS_READ: "settings.read",
  SETTINGS_WRITE: "settings.write",
  SUPPORT_TICKETS_READ: "support.tickets.read",
  SUPPORT_TICKETS_REPLY: "support.tickets.reply",
  SUPPORT_TICKETS_WRITE: "support.tickets.write",
} as const

export type Permission = (typeof PERMISSIONS)[keyof typeof PERMISSIONS]

function setOf(...perms: Permission[]): Set<Permission> {
  return new Set(perms)
}

const ROLE_PERMISSIONS: Record<string, Set<Permission>> = {
  admin: setOf(
    PERMISSIONS.ADMIN_DASHBOARD_READ,
    PERMISSIONS.USERS_READ,
    PERMISSIONS.USERS_WRITE,
    PERMISSIONS.USERS_DELETE,
    PERMISSIONS.USERS_BAN,
    PERMISSIONS.LISTINGS_MODERATE,
    PERMISSIONS.LISTINGS_DELETE,
    PERMISSIONS.FINANCE_READ,
    PERMISSIONS.AUDIT_LOGS_READ,
    PERMISSIONS.CATALOG_MANAGE,
    PERMISSIONS.PLANS_MANAGE,
    PERMISSIONS.REPORTS_REVIEW,
    PERMISSIONS.OPS_READ,
    PERMISSIONS.OPS_MANAGE,
    PERMISSIONS.SETTINGS_READ,
    PERMISSIONS.SETTINGS_WRITE,
    PERMISSIONS.SUPPORT_TICKETS_READ,
    PERMISSIONS.SUPPORT_TICKETS_REPLY,
    PERMISSIONS.SUPPORT_TICKETS_WRITE,
  ),
  ops_admin: setOf(
    PERMISSIONS.ADMIN_DASHBOARD_READ,
    PERMISSIONS.LISTINGS_MODERATE,
    PERMISSIONS.REPORTS_REVIEW,
    PERMISSIONS.OPS_READ,
    PERMISSIONS.OPS_MANAGE,
  ),
  finance_admin: setOf(
    PERMISSIONS.ADMIN_DASHBOARD_READ,
    PERMISSIONS.FINANCE_READ,
    PERMISSIONS.AUDIT_LOGS_READ,
  ),
  support_admin: setOf(
    PERMISSIONS.ADMIN_DASHBOARD_READ,
    PERMISSIONS.SUPPORT_TICKETS_READ,
    PERMISSIONS.SUPPORT_TICKETS_REPLY,
    PERMISSIONS.SUPPORT_TICKETS_WRITE,
    PERMISSIONS.REPORTS_REVIEW,
  ),
}

export function isInternalRole(role?: string | null): boolean {
  if (!role) return false
  return role === "super_admin" || role in ROLE_PERMISSIONS
}

export function hasPermission(role: string | undefined | null, perm: Permission): boolean {
  if (!role) return false
  if (role === "super_admin") return true
  return ROLE_PERMISSIONS[role]?.has(perm) ?? false
}

export function hasAnyPermission(role: string | undefined | null, perms: Permission[]): boolean {
  return perms.some((p) => hasPermission(role, p))
}
