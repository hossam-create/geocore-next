import { Bell, Search, ChevronRight, Clock } from "lucide-react"
import { useAuthStore } from "@/store/auth"
import { useLocation } from "react-router-dom"
import { format } from "date-fns"

interface HeaderProps {
  title?: string
}

const PAGE_META: Record<string, { title: string; description: string }> = {
  "/admin": { title: "Dashboard", description: "Platform overview & key metrics" },
  "/admin/listings": { title: "Listings", description: "Review & moderate marketplace listings" },
  "/admin/auctions": { title: "Auctions", description: "Monitor live & scheduled auctions" },
  "/admin/users": { title: "Users", description: "Manage user accounts & roles" },
  "/admin/reports": { title: "Reports", description: "Review flagged content & user reports" },
  "/admin/payments": { title: "Payments & Revenue", description: "Financial overview & transactions" },
  "/admin/transactions": { title: "Transactions", description: "Full transaction ledger" },
  "/admin/categories": { title: "Categories", description: "Manage catalog & category tree" },
  "/admin/pricing": { title: "Price Plans", description: "Subscription plans & pricing config" },
  "/admin/settings": { title: "Site Settings", description: "Platform-wide configuration" },
  "/admin/logs": { title: "Audit Logs", description: "Security & activity trail" },
}

export function Header({ title }: HeaderProps) {
  const { user } = useAuthStore()
  const location = useLocation()
  const meta = PAGE_META[location.pathname]
  const pageTitle = meta?.title ?? title ?? "Admin"
  const pageDesc = meta?.description

  const now = new Date()

  return (
    <header className="h-14 bg-white border-b border-gray-100 flex items-center px-5 gap-4 shrink-0">

      {/* Page identity */}
      <div className="flex items-center gap-2 min-w-0">
        <span className="text-xs text-gray-400 hidden sm:flex items-center gap-1 shrink-0">
          <span>GeoCore</span>
          <ChevronRight className="w-3 h-3" />
        </span>
        <div className="min-w-0">
          <h1 className="text-sm font-semibold text-gray-900 leading-tight truncate">{pageTitle}</h1>
          {pageDesc && (
            <p className="text-[11px] text-gray-400 leading-tight hidden md:block truncate">{pageDesc}</p>
          )}
        </div>
      </div>

      <div className="flex-1" />

      {/* Timestamp */}
      <div className="hidden lg:flex items-center gap-1.5 text-[11px] text-gray-400">
        <Clock className="w-3 h-3" />
        <span>{format(now, "EEE, MMM d · HH:mm")}</span>
      </div>

      {/* Search */}
      <button className="hidden md:flex items-center gap-2 bg-gray-50 border border-gray-200 rounded-lg px-3 py-1.5 text-[12px] text-gray-400 cursor-pointer hover:bg-gray-100 transition-colors min-w-[160px]">
        <Search className="w-3.5 h-3.5 shrink-0" />
        <span className="flex-1 text-left">Quick search...</span>
        <kbd className="text-[10px] bg-gray-100 border border-gray-200 rounded px-1 py-0.5 font-mono">⌘K</kbd>
      </button>

      {/* Notification bell */}
      <button className="relative p-2 rounded-lg hover:bg-gray-50 transition-colors text-gray-400 hover:text-gray-600">
        <Bell className="w-4 h-4" />
        <span className="absolute top-1.5 right-1.5 w-1.5 h-1.5 bg-red-500 rounded-full ring-2 ring-white" />
      </button>

      {/* User chip */}
      <div className="flex items-center gap-2 pl-2 border-l border-gray-100">
        <div className="w-7 h-7 rounded-full bg-gradient-to-br from-[#0071CE] to-[#005ba1] flex items-center justify-center text-white text-[11px] font-bold shadow-sm">
          {user?.name?.[0]?.toUpperCase() ?? "A"}
        </div>
        <span className="text-xs font-medium text-gray-700 hidden sm:block max-w-[100px] truncate">
          {user?.name ?? "Admin"}
        </span>
      </div>
    </header>
  )
}
