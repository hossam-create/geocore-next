import { Link, useLocation } from "react-router-dom"
import {
  LayoutDashboard, Tag, Hammer, Store, Grid, Users, Flag,
  CreditCard, DollarSign, Wallet, Settings, Bell, Shield,
  LogOut, ScrollText, Activity,
  Zap,
} from "lucide-react"
import { useAuthStore } from "@/store/auth"
import { PERMISSIONS, hasAnyPermission, type Permission } from "@/lib/permissions"

type NavItem = {
  icon: React.ElementType
  label: string
  path: string
  requiredAny?: Permission[]
  badge?: string
  badgeColor?: string
}

type NavSection = {
  section: string
  icon: React.ElementType
  items: NavItem[]
}

const navigation: NavSection[] = [
  {
    section: "Overview",
    icon: Activity,
    items: [
      { icon: LayoutDashboard, label: "Dashboard", path: "/admin", requiredAny: [PERMISSIONS.ADMIN_DASHBOARD_READ] },
    ],
  },
  {
    section: "Marketplace",
    icon: Store,
    items: [
      { icon: Tag, label: "Listings", path: "/admin/listings", requiredAny: [PERMISSIONS.LISTINGS_MODERATE] },
      { icon: Hammer, label: "Auctions", path: "/admin/auctions", requiredAny: [PERMISSIONS.LISTINGS_MODERATE] },
      { icon: Store, label: "Storefronts", path: "/admin/storefronts" },
      { icon: Grid, label: "Categories", path: "/admin/categories", requiredAny: [PERMISSIONS.CATALOG_MANAGE] },
    ],
  },
  {
    section: "Users & Trust",
    icon: Users,
    items: [
      { icon: Users, label: "All Users", path: "/admin/users", requiredAny: [PERMISSIONS.USERS_READ] },
      { icon: Flag, label: "Reports", path: "/admin/reports", requiredAny: [PERMISSIONS.REPORTS_REVIEW] },
    ],
  },
  {
    section: "Finance",
    icon: DollarSign,
    items: [
      { icon: CreditCard, label: "Payments", path: "/admin/payments", requiredAny: [PERMISSIONS.FINANCE_READ] },
      { icon: DollarSign, label: "Price Plans", path: "/admin/pricing", requiredAny: [PERMISSIONS.FINANCE_READ, PERMISSIONS.PLANS_MANAGE] },
      { icon: Wallet, label: "Transactions", path: "/admin/transactions", requiredAny: [PERMISSIONS.FINANCE_READ] },
    ],
  },
  {
    section: "System",
    icon: Settings,
    items: [
      { icon: Settings, label: "Site Settings", path: "/admin/settings", requiredAny: [PERMISSIONS.SETTINGS_READ, PERMISSIONS.SETTINGS_WRITE] },
      { icon: Bell, label: "Email Templates", path: "/admin/emails" },
      { icon: Shield, label: "Staff Users", path: "/admin/staff" },
      { icon: ScrollText, label: "Audit Logs", path: "/admin/logs", requiredAny: [PERMISSIONS.AUDIT_LOGS_READ] },
    ],
  },
]

const ROLE_LABELS: Record<string, { label: string; color: string }> = {
  super_admin: { label: "Super Admin", color: "bg-amber-500/20 text-amber-300 border border-amber-500/30" },
  admin: { label: "Admin", color: "bg-blue-500/20 text-blue-300 border border-blue-500/30" },
  ops_admin: { label: "Ops", color: "bg-green-500/20 text-green-300 border border-green-500/30" },
  finance_admin: { label: "Finance", color: "bg-purple-500/20 text-purple-300 border border-purple-500/30" },
  support_admin: { label: "Support", color: "bg-pink-500/20 text-pink-300 border border-pink-500/30" },
}

export function Sidebar() {
  const location = useLocation()
  const { user, logout } = useAuthStore()
  const role = user?.role
  const roleInfo = role ? ROLE_LABELS[role] : undefined

  const visibleNavigation = navigation
    .map((section) => ({
      ...section,
      items: section.items.filter((item) => {
        if (!item.requiredAny || item.requiredAny.length === 0) return true
        return hasAnyPermission(role, item.requiredAny)
      }),
    }))
    .filter((section) => section.items.length > 0)

  return (
    <aside className="w-[240px] min-h-screen bg-[#0F172A] flex flex-col shrink-0 border-r border-white/[0.06]">

      {/* Logo */}
      <div className="px-5 py-4 border-b border-white/[0.06]">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-[#0071CE] to-[#005ba1] flex items-center justify-center shrink-0 shadow-lg shadow-blue-900/30">
            <Zap className="w-4 h-4 text-white" />
          </div>
          <div>
            <p className="text-white font-semibold text-sm leading-tight tracking-tight">GeoCore</p>
            <p className="text-white/30 text-[10px] font-medium tracking-widest uppercase">Control Center</p>
          </div>
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-3 py-4 space-y-5 overflow-y-auto">
        {visibleNavigation.map((section) => (
          <div key={section.section}>
            <div className="flex items-center gap-1.5 px-2 mb-1.5">
              <section.icon className="w-3 h-3 text-white/20" />
              <p className="text-white/25 text-[10px] font-bold uppercase tracking-widest">
                {section.section}
              </p>
            </div>
            <div className="space-y-0.5">
              {section.items.map((item) => {
                const isActive =
                  item.path === "/admin"
                    ? location.pathname === "/admin"
                    : location.pathname.startsWith(item.path)
                return (
                  <Link
                    key={item.path}
                    to={item.path}
                    className={`group relative flex items-center gap-2.5 px-3 py-2 rounded-lg text-[13px] font-medium transition-all duration-150 ${
                      isActive
                        ? "bg-[#0071CE]/15 text-[#60aef5]"
                        : "text-white/45 hover:text-white/80 hover:bg-white/[0.05]"
                    }`}
                  >
                    {isActive && (
                      <span className="absolute left-0 top-1/2 -translate-y-1/2 w-[3px] h-5 bg-[#0071CE] rounded-r-full" />
                    )}
                    <item.icon className={`w-4 h-4 shrink-0 ${isActive ? "text-[#60aef5]" : "text-white/30 group-hover:text-white/60"}`} />
                    <span className="flex-1 truncate">{item.label}</span>
                    {item.badge && (
                      <span className={`text-[10px] font-bold px-1.5 py-0.5 rounded-full ${item.badgeColor ?? "bg-red-500/20 text-red-400"}`}>
                        {item.badge}
                      </span>
                    )}
                  </Link>
                )
              })}
            </div>
          </div>
        ))}
      </nav>

      {/* Divider */}
      <div className="mx-4 border-t border-white/[0.06]" />

      {/* User card */}
      <div className="p-3">
        <div className="flex items-center gap-3 px-2 py-2.5 rounded-lg hover:bg-white/[0.04] transition-colors group">
          {/* Avatar */}
          <div className="w-7 h-7 rounded-full bg-gradient-to-br from-[#0071CE] to-[#005ba1] flex items-center justify-center shrink-0 text-white text-xs font-bold shadow">
            {user?.name?.[0]?.toUpperCase() ?? "A"}
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-white/80 text-xs font-semibold truncate leading-tight">{user?.name ?? "Admin User"}</p>
            {roleInfo ? (
              <span className={`inline-block mt-0.5 text-[9px] font-bold px-1.5 py-0.5 rounded-full tracking-wide ${roleInfo.color}`}>
                {roleInfo.label}
              </span>
            ) : (
              <p className="text-white/30 text-[10px] capitalize">{role ?? "admin"}</p>
            )}
          </div>
          <button
            onClick={logout}
            title="Sign out"
            className="text-white/20 hover:text-red-400 transition-colors p-1 rounded-md hover:bg-red-500/10"
          >
            <LogOut className="w-3.5 h-3.5" />
          </button>
        </div>

        {/* System status indicator */}
        <div className="mt-2 flex items-center gap-2 px-2 py-1.5 rounded-md bg-green-500/[0.07] border border-green-500/10">
          <span className="relative flex h-2 w-2 shrink-0">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-50" />
            <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500" />
          </span>
          <span className="text-[10px] text-green-400/80 font-medium">All systems operational</span>
        </div>
      </div>
    </aside>
  )
}
