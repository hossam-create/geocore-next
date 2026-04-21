"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState, useCallback } from "react";
import {
  LayoutDashboard,
  Users,
  List,
  Gavel,
  ShoppingCart,
  AlertTriangle,
  TicketCheck,
  Tag,
  DollarSign,
  BarChart3,
  Settings,
  Flag,
  FileText,
  Briefcase,
  LogOut,
  Shield,
  Scale,
  ScrollText,
  AlertOctagon,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Landmark,
  TrendingUp,
  PieChart,
  BellRing,
  Server,
  Mail,
  Megaphone,
  CreditCard,
  Percent,
  MapPin,
  UsersRound,
  FormInput,
  Store,
  Puzzle,
  Receipt,
  Globe,
  CircleDot,
  Building2,
  ShieldAlert,
  Trophy,
  Activity,
  ScanSearch,
  Truck,
  Calculator,
  UserCheck,
} from "lucide-react";
import { useAdminAuth } from "@/lib/auth";

interface NavItem {
  label: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  badge?: number;
}

interface NavSection {
  title: string;
  key: string;
  accent?: boolean;
  items: NavItem[];
}

const NAV_SECTIONS: NavSection[] = [
  {
    title: "Overview",
    key: "overview",
    items: [
      { label: "Control Center", href: "/control-center", icon: Globe },
      { label: "Dashboard", href: "/dashboard", icon: LayoutDashboard },
    ],
  },
  {
    title: "Operations",
    key: "operations",
    items: [
      { label: "Listings Queue", href: "/operations/listings", icon: List },
      { label: "Moderation Queue", href: "/operations/listings/moderation", icon: ScanSearch },
      { label: "Auctions Monitor", href: "/operations/auctions", icon: Gavel },
      { label: "Orders", href: "/operations/orders", icon: ShoppingCart },
      { label: "Escrow Tracking", href: "/operations/escrow", icon: Landmark },
    ],
  },
  {
    title: "Custodii",
    key: "custodii",
    accent: true,
    items: [
      { label: "Decision Queue", href: "/custodii/decisions", icon: Shield },
      { label: "Policy Engine", href: "/custodii/policies", icon: Scale },
      { label: "Audit Logs", href: "/custodii/audit", icon: ScrollText },
      { label: "Risk Alerts", href: "/custodii/risks", icon: AlertOctagon },
    ],
  },
  {
    title: "Admin",
    key: "admin",
    items: [
      { label: "Users", href: "/admin/users", icon: Users },
      { label: "KYC Queue", href: "/admin/users/kyc", icon: UserCheck },
      { label: "Org Structure", href: "/admin/structure", icon: Building2 },
      { label: "User Groups", href: "/admin/user-groups", icon: UsersRound },
      { label: "User Fields", href: "/admin/user-fields", icon: FormInput },
      { label: "Roles & Perms", href: "/admin/roles", icon: Shield },
      { label: "Feature Flags", href: "/admin/features", icon: Flag },
      { label: "Categories", href: "/admin/categories", icon: Tag },
      { label: "Settings", href: "/admin/settings", icon: Settings },
    ],
  },
  {
    title: "Pricing",
    key: "pricing",
    items: [
      { label: "Price Plans", href: "/pricing/plans", icon: DollarSign },
      { label: "Gateways", href: "/pricing/gateways", icon: CreditCard },
      { label: "Discount Codes", href: "/pricing/discounts", icon: Percent },
      { label: "Invoices", href: "/pricing/invoices", icon: Receipt },
      { label: "Listing Extras", href: "/pricing/extras", icon: Tag },
    ],
  },
  {
    title: "Storefronts",
    key: "storefronts",
    items: [
      { label: "All Stores", href: "/storefronts", icon: Store },
    ],
  },
  {
    title: "Content",
    key: "content",
    items: [
      { label: "Email Templates", href: "/content/emails", icon: Mail },
      { label: "Static Pages", href: "/content/pages", icon: FileText },
      { label: "Announcements", href: "/content/announcements", icon: Megaphone },
      { label: "Geography", href: "/content/geography", icon: MapPin },
    ],
  },
  {
    title: "Trust & Safety",
    key: "trust",
    accent: true,
    items: [
      { label: "Flags Dashboard", href: "/trust", icon: ShieldAlert },
    ],
  },
  {
    title: "Sellers",
    key: "sellers",
    items: [
      { label: "Seller Hub", href: "/sellers", icon: Trophy },
    ],
  },
  {
    title: "Analytics",
    key: "analytics",
    items: [
      { label: "Overview", href: "/analytics/overview", icon: BarChart3 },
      { label: "Traffic Watch", href: "/analytics/traffic", icon: Activity },
      { label: "Revenue", href: "/analytics/revenue", icon: TrendingUp },
      { label: "Reports", href: "/analytics/reports", icon: PieChart },
    ],
  },
  {
    title: "Support",
    key: "support",
    items: [
      { label: "Tickets", href: "/support/tickets", icon: TicketCheck },
      { label: "Disputes", href: "/support/disputes", icon: AlertTriangle },
    ],
  },
  {
    title: "Ops Health",
    key: "ops",
    items: [
      { label: "Real-Time Health", href: "/ops", icon: Activity },
    ],
  },
  {
    title: "Compliance",
    key: "compliance",
    items: [
      { label: "GDPR & Audit", href: "/compliance", icon: ShieldAlert },
    ],
  },
  {
    title: "System",
    key: "system",
    items: [
      { label: "Jobs", href: "/system/jobs", icon: Briefcase },
      { label: "Notifications", href: "/system/notifications", icon: BellRing },
      { label: "Health", href: "/system/health", icon: Server },
    ],
  },
  {
    title: "Addons",
    key: "addons",
    items: [
      { label: "Marketplace", href: "/addons", icon: Puzzle },
    ],
  },
];

function roleBadgeColor(role: string) {
  switch (role) {
    case "super_admin": return { bg: "rgba(239,68,68,0.15)", text: "#f87171" };
    case "admin": return { bg: "rgba(0,113,206,0.15)", text: "#38bdf8" };
    case "ops_admin": return { bg: "rgba(245,158,11,0.15)", text: "#fbbf24" };
    case "finance_admin": return { bg: "rgba(16,185,129,0.15)", text: "#34d399" };
    case "support_admin": return { bg: "rgba(99,102,241,0.15)", text: "#a5b4fc" };
    default: return { bg: "rgba(100,116,139,0.15)", text: "#94a3b8" };
  }
}

export default function AdminSidebar() {
  const pathname = usePathname();
  const { user, logout } = useAdminAuth();
  const [collapsed, setCollapsed] = useState(false);
  const [openSections, setOpenSections] = useState<Record<string, boolean>>(() => {
    const initial: Record<string, boolean> = {};
    NAV_SECTIONS.forEach((s) => { initial[s.key] = true; });
    return initial;
  });

  const toggleSection = useCallback((key: string) => {
    setOpenSections((prev) => ({ ...prev, [key]: !prev[key] }));
  }, []);

  const isSectionActive = useCallback((section: NavSection) => {
    return section.items.some((item) => pathname === item.href || pathname.startsWith(item.href + "/"));
  }, [pathname]);

  const badge = roleBadgeColor(user?.role ?? "");

  return (
    <aside
      className="fixed left-0 top-0 bottom-0 flex flex-col z-30 transition-all duration-200"
      style={{
        width: collapsed ? "var(--sidebar-collapsed)" : "var(--sidebar-width)",
        background: "var(--sidebar-bg)",
        borderRight: "1px solid var(--sidebar-border)",
      }}
    >
      {/* ── Brand ── */}
      <div
        className="flex items-center shrink-0 border-b"
        style={{
          height: "var(--topbar-height)",
          borderColor: "var(--sidebar-border)",
          padding: collapsed ? "0 16px" : "0 20px",
          justifyContent: collapsed ? "center" : "flex-start",
        }}
      >
        {collapsed ? (
          <div className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ background: "var(--sidebar-active-bg)" }}>
            <Globe className="w-4.5 h-4.5" style={{ color: "var(--sidebar-active)" }} />
          </div>
        ) : (
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ background: "var(--sidebar-active-bg)" }}>
              <Globe className="w-4 h-4" style={{ color: "var(--sidebar-active)" }} />
            </div>
            <div>
              <span className="text-white font-bold text-[15px] tracking-tight leading-none">GeoCore</span>
              <span className="ml-1.5 text-[9px] px-1.5 py-[1px] rounded font-bold tracking-wider" style={{ background: "var(--sidebar-active-bg)", color: "var(--sidebar-active)" }}>
                CTRL
              </span>
            </div>
          </div>
        )}
      </div>

      {/* ── Navigation ── */}
      <nav className="flex-1 overflow-y-auto" style={{ padding: collapsed ? "8px 6px" : "8px 10px" }}>
        {NAV_SECTIONS.map((section) => {
          const isOpen = openSections[section.key] ?? true;
          const sectionActive = isSectionActive(section);

          return (
            <div key={section.key} className="mb-1">
              {/* Section header */}
              {!collapsed ? (
                <button
                  onClick={() => toggleSection(section.key)}
                  className="flex items-center justify-between w-full px-2 py-1.5 rounded-md group transition-colors"
                  style={{ marginTop: "4px" }}
                >
                  <span
                    className="text-[10px] font-semibold uppercase tracking-[0.08em] transition-colors"
                    style={{
                      color: section.accent
                        ? "var(--sidebar-active)"
                        : sectionActive
                          ? "var(--sidebar-text-hover)"
                          : "var(--sidebar-text)",
                    }}
                  >
                    {section.title}
                  </span>
                  <ChevronDown
                    className="w-3 h-3 transition-transform duration-200"
                    style={{
                      color: "var(--sidebar-text)",
                      transform: isOpen ? "rotate(0deg)" : "rotate(-90deg)",
                    }}
                  />
                </button>
              ) : (
                <div className="w-full h-px my-2" style={{ background: "var(--sidebar-border)" }} />
              )}

              {/* Section items */}
              {(collapsed || isOpen) && (
                <div className={collapsed ? "" : "mt-0.5"}>
                  {section.items.map((item) => {
                    const Icon = item.icon;
                    const active = pathname === item.href || pathname.startsWith(item.href + "/");
                    return (
                      <Link
                        key={item.href}
                        href={item.href}
                        title={collapsed ? item.label : undefined}
                        className="group flex items-center gap-2.5 rounded-md transition-all text-[13px] font-medium relative"
                        style={{
                          padding: collapsed ? "8px" : "6px 10px",
                          justifyContent: collapsed ? "center" : "flex-start",
                          background: active ? "var(--sidebar-active-bg)" : "transparent",
                          color: active ? "var(--sidebar-active)" : "var(--sidebar-text)",
                        }}
                        onMouseEnter={(e) => {
                          if (!active) {
                            e.currentTarget.style.color = "var(--sidebar-text-hover)";
                            e.currentTarget.style.background = "var(--sidebar-surface)";
                          }
                        }}
                        onMouseLeave={(e) => {
                          if (!active) {
                            e.currentTarget.style.color = "var(--sidebar-text)";
                            e.currentTarget.style.background = "transparent";
                          }
                        }}
                      >
                        {/* Active indicator bar */}
                        {active && !collapsed && (
                          <span
                            className="absolute left-0 top-1/2 -translate-y-1/2 w-[3px] h-4 rounded-r-full"
                            style={{ background: "var(--sidebar-active)" }}
                          />
                        )}
                        <Icon className="w-4 h-4 shrink-0" />
                        {!collapsed && <span>{item.label}</span>}
                        {!collapsed && item.badge && item.badge > 0 && (
                          <span className="ml-auto text-[10px] font-bold px-1.5 py-0.5 rounded-full" style={{ background: "var(--color-danger)", color: "#fff" }}>
                            {item.badge}
                          </span>
                        )}
                      </Link>
                    );
                  })}
                </div>
              )}
            </div>
          );
        })}
      </nav>

      {/* ── User Card + Footer ── */}
      <div className="shrink-0 border-t" style={{ borderColor: "var(--sidebar-border)" }}>
        {/* System status */}
        {!collapsed && (
          <div className="flex items-center gap-2 px-4 py-2" style={{ borderBottom: "1px solid var(--sidebar-border)" }}>
            <CircleDot className="w-3 h-3" style={{ color: "var(--color-success)" }} />
            <span className="text-[11px]" style={{ color: "var(--sidebar-text)" }}>All systems operational</span>
          </div>
        )}

        {/* User card */}
        <div style={{ padding: collapsed ? "8px 6px" : "10px 12px" }}>
          {!collapsed ? (
            <div className="flex items-center gap-2.5 mb-2">
              <div
                className="w-8 h-8 rounded-full flex items-center justify-center text-white text-xs font-bold shrink-0"
                style={{ background: "var(--color-brand)" }}
              >
                {user?.name?.charAt(0)?.toUpperCase() ?? "A"}
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-[13px] font-medium text-white leading-tight truncate">
                  {user?.name ?? "Admin"}
                </p>
                <span
                  className="inline-block text-[9px] font-bold uppercase tracking-wider px-1.5 py-[1px] rounded mt-0.5"
                  style={{ background: badge.bg, color: badge.text }}
                >
                  {user?.role?.replace(/_/g, " ") ?? "admin"}
                </span>
              </div>
            </div>
          ) : (
            <div className="flex justify-center mb-2">
              <div
                className="w-8 h-8 rounded-full flex items-center justify-center text-white text-xs font-bold"
                style={{ background: "var(--color-brand)" }}
                title={user?.name ?? "Admin"}
              >
                {user?.name?.charAt(0)?.toUpperCase() ?? "A"}
              </div>
            </div>
          )}

          {/* Actions */}
          <div className="flex gap-1">
            <button
              onClick={logout}
              title="Logout"
              className="flex items-center gap-2 flex-1 py-1.5 rounded-md transition-all text-[12px]"
              style={{
                padding: collapsed ? "6px" : "4px 8px",
                justifyContent: collapsed ? "center" : "flex-start",
                color: "var(--sidebar-text)",
              }}
              onMouseEnter={(e) => { e.currentTarget.style.color = "var(--color-danger)"; e.currentTarget.style.background = "rgba(239,68,68,0.1)"; }}
              onMouseLeave={(e) => { e.currentTarget.style.color = "var(--sidebar-text)"; e.currentTarget.style.background = "transparent"; }}
            >
              <LogOut className="w-3.5 h-3.5" />
              {!collapsed && <span>Logout</span>}
            </button>
            <button
              onClick={() => setCollapsed(!collapsed)}
              title={collapsed ? "Expand" : "Collapse"}
              className="py-1.5 rounded-md transition-all"
              style={{
                padding: collapsed ? "6px" : "4px 8px",
                color: "var(--sidebar-text)",
              }}
              onMouseEnter={(e) => { e.currentTarget.style.color = "var(--sidebar-text-hover)"; e.currentTarget.style.background = "var(--sidebar-surface)"; }}
              onMouseLeave={(e) => { e.currentTarget.style.color = "var(--sidebar-text)"; e.currentTarget.style.background = "transparent"; }}
            >
              {collapsed ? <ChevronRight className="w-3.5 h-3.5" /> : <ChevronLeft className="w-3.5 h-3.5" />}
            </button>
          </div>
        </div>
      </div>
    </aside>
  );
}
