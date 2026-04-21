export const INTERNAL_ADMIN_ROLES = [
  "super_admin",
  "admin",
  "ops_admin",
  "finance_admin",
  "support_admin",
] as const;

export type InternalAdminRole = (typeof INTERNAL_ADMIN_ROLES)[number];

export type AdminDepartment = {
  key: InternalAdminRole;
  title: string;
  description: string;
  accessLevel: "global" | "department";
  defaultRoute: string;
  permissions: string[];
  loginHint: string;
};

export const ADMIN_DEPARTMENTS: AdminDepartment[] = [
  {
    key: "super_admin",
    title: "Executive Control",
    description: "Full platform access across all administrative domains.",
    accessLevel: "global",
    defaultRoute: "/dashboard",
    permissions: ["*"],
    loginHint: "chief-admin@company.com",
  },
  {
    key: "admin",
    title: "Platform Administration",
    description: "Core platform operations, users, settings, and moderation workflows.",
    accessLevel: "global",
    defaultRoute: "/dashboard",
    permissions: [
      "admin.dashboard.read",
      "users.read/write",
      "settings.read/write",
      "reports.review",
    ],
    loginHint: "admin@company.com",
  },
  {
    key: "ops_admin",
    title: "Operations Management",
    description: "Listings moderation, operational incidents, and operational health.",
    accessLevel: "department",
    defaultRoute: "/operations/listings",
    permissions: ["listings.moderate", "reports.review", "ops.read/manage"],
    loginHint: "ops-admin@company.com",
  },
  {
    key: "finance_admin",
    title: "Finance Management",
    description: "Pricing, payment visibility, settlements, and finance audit review.",
    accessLevel: "department",
    defaultRoute: "/pricing/plans",
    permissions: ["finance.read", "audit.logs.read"],
    loginHint: "finance-admin@company.com",
  },
  {
    key: "support_admin",
    title: "Support Management",
    description: "Tickets, escalations, customer operations, and support oversight.",
    accessLevel: "department",
    defaultRoute: "/support/tickets",
    permissions: ["support.tickets.read/reply/write", "reports.review"],
    loginHint: "support-admin@company.com",
  },
];

export function isInternalAdminRole(role: string): role is InternalAdminRole {
  return INTERNAL_ADMIN_ROLES.includes(role as InternalAdminRole);
}

export function getDepartmentByRole(role?: string | null): AdminDepartment | undefined {
  if (!role) {
    return undefined;
  }
  return ADMIN_DEPARTMENTS.find((dept) => dept.key === role);
}
