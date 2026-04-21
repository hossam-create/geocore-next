import { Suspense, lazy } from "react"
import { useQuery } from "@tanstack/react-query"
import {
  DollarSign, Tag, Users, Hammer, UserPlus, Plus, Flag,
  TrendingUp, TrendingDown, ArrowUpRight, AlertTriangle,
  Clock, CheckCircle2, XCircle, Eye, MoreHorizontal,
  Activity, ShieldAlert,
} from "lucide-react"
import { Link } from "react-router-dom"
import { api } from "@/api/client"
import { Card } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"
import { formatCurrency, formatNumber } from "@/lib/utils"
import { format, formatDistanceToNow } from "date-fns"

const RevenueChart = lazy(() =>
  import("@/components/charts/RevenueChart").then((mod) => ({ default: mod.RevenueChart }))
)

interface DashboardStats {
  total_users: number
  total_listings: number
  active_listings: number
  total_auctions: number
  live_auctions: number
  total_revenue: number
  revenue_today: number
  pending_moderation: number
  reports_pending: number
  new_users_this_week: number
  new_listings_today: number
}

interface KPICardProps {
  title: string
  value: string | number | undefined
  sub?: string
  trend?: "up" | "down" | "neutral"
  trendValue?: string
  icon: React.ElementType
  accent: string
  isLoading?: boolean
  href?: string
}

function KPICard({ title, value, sub, trend, trendValue, icon: Icon, accent, isLoading, href }: KPICardProps) {
  const inner = (
    <div className="relative bg-white rounded-xl border border-gray-100 p-5 overflow-hidden hover:shadow-sm transition-shadow group cursor-default">
      {/* Accent bar */}
      <div className={`absolute top-0 left-0 right-0 h-[2px] ${accent}`} />

      <div className="flex items-start justify-between mb-3">
        <p className="text-xs font-semibold text-gray-400 uppercase tracking-wide">{title}</p>
        <div className={`w-8 h-8 rounded-lg flex items-center justify-center ${accent.replace("bg-", "bg-").replace("-500", "-50")} text-current`}
          style={{ background: "rgba(0,113,206,0.07)" }}>
          <Icon className="w-4 h-4 text-gray-500" />
        </div>
      </div>

      {isLoading ? (
        <>
          <Skeleton className="h-8 w-28 mb-1" />
          <Skeleton className="h-3 w-20" />
        </>
      ) : (
        <>
          <p className="text-2xl font-bold text-gray-900 tabular-nums">{value ?? "—"}</p>
          <div className="flex items-center gap-2 mt-1.5">
            {trendValue && (
              <span className={`inline-flex items-center gap-0.5 text-xs font-semibold px-1.5 py-0.5 rounded-full ${
                trend === "up" ? "bg-emerald-50 text-emerald-600" :
                trend === "down" ? "bg-red-50 text-red-500" :
                "bg-gray-100 text-gray-500"
              }`}>
                {trend === "up" ? <TrendingUp className="w-3 h-3" /> : trend === "down" ? <TrendingDown className="w-3 h-3" /> : null}
                {trendValue}
              </span>
            )}
            {sub && <span className="text-xs text-gray-400">{sub}</span>}
          </div>
        </>
      )}

      {href && (
        <ArrowUpRight className="absolute bottom-4 right-4 w-4 h-4 text-gray-200 group-hover:text-gray-400 transition-colors" />
      )}
    </div>
  )
  return href ? <Link to={href}>{inner}</Link> : inner
}

type PendingListing = {
  id: string
  title: string
  images?: { url: string }[]
  user?: { name: string }
  category?: { name_en: string }
  price?: number
  currency?: string
  city?: string
  country?: string
  created_at: string
}

export function DashboardPage() {
  const { data: stats, isLoading } = useQuery<DashboardStats>({
    queryKey: ["admin", "stats"],
    queryFn: () => api.get("/admin/stats").then((r) => r.data.data),
    refetchInterval: 30000,
  })

  const { data: revenue, isLoading: revenueLoading } = useQuery({
    queryKey: ["admin", "revenue"],
    queryFn: () => api.get("/admin/revenue").then((r) => r.data.data),
  })

  const { data: pendingListings, isLoading: queueLoading } = useQuery({
    queryKey: ["admin", "listings", "pending"],
    queryFn: () => api.get("/admin/listings?status=pending&per_page=6").then((r) => r.data),
  })

  const hasPendingAlerts =
    (stats?.pending_moderation ?? 0) > 0 || (stats?.reports_pending ?? 0) > 0

  return (
    <div className="space-y-5">

      {/* ── Alert Banner ── */}
      {hasPendingAlerts && !isLoading && (
        <div className="flex items-center gap-3 bg-amber-50 border border-amber-200/60 rounded-xl px-4 py-3">
          <ShieldAlert className="w-4 h-4 text-amber-500 shrink-0" />
          <p className="text-sm text-amber-800 font-medium flex-1">
            Action required:{" "}
            {(stats?.pending_moderation ?? 0) > 0 && (
              <span><strong>{stats?.pending_moderation}</strong> listings awaiting moderation</span>
            )}
            {(stats?.pending_moderation ?? 0) > 0 && (stats?.reports_pending ?? 0) > 0 && " · "}
            {(stats?.reports_pending ?? 0) > 0 && (
              <span><strong>{stats?.reports_pending}</strong> open reports</span>
            )}
          </p>
          <Link to="/admin/listings?status=pending" className="text-xs font-semibold text-amber-700 hover:text-amber-900 whitespace-nowrap">
            Review now →
          </Link>
        </div>
      )}

      {/* ── KPI Strip ── */}
      <div className="grid grid-cols-2 xl:grid-cols-4 gap-3">
        <KPICard
          title="Total Revenue"
          value={formatCurrency(stats?.total_revenue)}
          trendValue={`${formatCurrency(stats?.revenue_today)} today`}
          trend="up"
          icon={DollarSign}
          accent="bg-blue-500"
          isLoading={isLoading}
          href="/admin/payments"
        />
        <KPICard
          title="Active Listings"
          value={formatNumber(stats?.active_listings)}
          trendValue={`${stats?.pending_moderation ?? 0} pending`}
          trend={((stats?.pending_moderation ?? 0) > 0) ? "down" : "up"}
          icon={Tag}
          accent="bg-emerald-500"
          isLoading={isLoading}
          href="/admin/listings"
        />
        <KPICard
          title="Total Users"
          value={formatNumber(stats?.total_users)}
          trendValue={`+${stats?.new_users_this_week ?? 0} this week`}
          trend="up"
          icon={Users}
          accent="bg-violet-500"
          isLoading={isLoading}
          href="/admin/users"
        />
        <KPICard
          title="Live Auctions"
          value={formatNumber(stats?.live_auctions)}
          sub={`${stats?.total_auctions ?? 0} total`}
          trend="neutral"
          icon={Hammer}
          accent="bg-orange-500"
          isLoading={isLoading}
          href="/admin/auctions"
        />
      </div>

      {/* ── Main Content Row ── */}
      <div className="grid grid-cols-1 xl:grid-cols-3 gap-5">

        {/* Revenue Chart (2/3 width) */}
        <div className="xl:col-span-2">
          <div className="bg-white rounded-xl border border-gray-100 p-5 h-full">
            <div className="flex items-center justify-between mb-1">
              <div>
                <h3 className="text-sm font-semibold text-gray-900">Revenue — Last 30 Days</h3>
                <p className="text-xs text-gray-400 mt-0.5">
                  Total: <span className="font-semibold text-gray-700">{formatCurrency(revenue?.total)}</span>
                </p>
              </div>
              <Link to="/admin/payments" className="inline-flex items-center gap-1 text-xs text-[#0071CE] font-medium hover:underline">
                Full report <ArrowUpRight className="w-3 h-3" />
              </Link>
            </div>
            <Suspense fallback={<Skeleton className="h-[220px] w-full mt-4" />}>
              {revenueLoading ? (
                <Skeleton className="h-[220px] w-full mt-4" />
              ) : (
                <RevenueChart data={revenue?.daily_30days} />
              )}
            </Suspense>
          </div>
        </div>

        {/* Activity Pulse (1/3 width) */}
        <div className="bg-white rounded-xl border border-gray-100 p-5">
          <div className="flex items-center gap-2 mb-4">
            <Activity className="w-4 h-4 text-gray-400" />
            <h3 className="text-sm font-semibold text-gray-900">Activity Pulse</h3>
          </div>
          <div className="space-y-3">
            {[
              { label: "New users (week)", value: stats?.new_users_this_week, icon: UserPlus, color: "text-blue-600", bg: "bg-blue-50" },
              { label: "New listings today", value: stats?.new_listings_today, icon: Plus, color: "text-emerald-600", bg: "bg-emerald-50" },
              { label: "Live auctions", value: stats?.live_auctions, icon: Hammer, color: "text-orange-600", bg: "bg-orange-50" },
              { label: "Revenue today", value: formatCurrency(stats?.revenue_today), icon: DollarSign, color: "text-violet-600", bg: "bg-violet-50" },
              { label: "Open reports", value: stats?.reports_pending, icon: Flag, color: "text-red-600", bg: "bg-red-50" },
            ].map((item) => (
              <div key={item.label} className="flex items-center gap-3">
                <div className={`w-7 h-7 rounded-lg ${item.bg} flex items-center justify-center shrink-0`}>
                  <item.icon className={`w-3.5 h-3.5 ${item.color}`} />
                </div>
                <span className="text-xs text-gray-500 flex-1">{item.label}</span>
                {isLoading ? (
                  <Skeleton className="h-3.5 w-10" />
                ) : (
                  <span className="text-sm font-bold text-gray-900 tabular-nums">{item.value ?? "—"}</span>
                )}
              </div>
            ))}
          </div>

          {/* Quick actions */}
          <div className="mt-5 pt-4 border-t border-gray-50 space-y-2">
            <p className="text-[10px] font-bold uppercase tracking-widest text-gray-300 mb-2">Quick Actions</p>
            <Link to="/admin/listings?status=pending" className="flex items-center gap-2 px-3 py-2 bg-gray-50 hover:bg-gray-100 rounded-lg text-xs font-medium text-gray-600 transition-colors">
              <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500" /> Review pending listings
            </Link>
            <Link to="/admin/reports" className="flex items-center gap-2 px-3 py-2 bg-gray-50 hover:bg-gray-100 rounded-lg text-xs font-medium text-gray-600 transition-colors">
              <Flag className="w-3.5 h-3.5 text-red-400" /> Check open reports
            </Link>
            <Link to="/admin/users" className="flex items-center gap-2 px-3 py-2 bg-gray-50 hover:bg-gray-100 rounded-lg text-xs font-medium text-gray-600 transition-colors">
              <Users className="w-3.5 h-3.5 text-blue-400" /> Browse user accounts
            </Link>
          </div>
        </div>
      </div>

      {/* ── Moderation Queue ── */}
      <div className="bg-white rounded-xl border border-gray-100 overflow-hidden">

        {/* Header */}
        <div className="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Clock className="w-4 h-4 text-amber-500" />
            <div>
              <h3 className="text-sm font-semibold text-gray-900">Moderation Queue</h3>
              <p className="text-xs text-gray-400">
                {queueLoading ? "Loading…" : `${pendingListings?.meta?.total ?? 0} listings awaiting review`}
              </p>
            </div>
          </div>
          <Link
            to="/admin/listings?status=pending"
            className="inline-flex items-center gap-1 text-xs font-semibold text-[#0071CE] hover:underline"
          >
            View all <ArrowUpRight className="w-3 h-3" />
          </Link>
        </div>

        {/* Table head */}
        <div className="hidden md:grid grid-cols-[1fr_160px_120px_100px_90px] gap-4 px-5 py-2.5 bg-gray-50/60 border-b border-gray-100">
          {["Listing", "Seller", "Category", "Price", "Submitted"].map((h) => (
            <p key={h} className="text-[10px] font-bold uppercase tracking-wider text-gray-400">{h}</p>
          ))}
        </div>

        {/* Rows */}
        <div className="divide-y divide-gray-50">
          {queueLoading && Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="flex items-center gap-4 px-5 py-3.5">
              <Skeleton className="w-10 h-10 rounded-lg shrink-0" />
              <div className="flex-1 space-y-1.5">
                <Skeleton className="h-3.5 w-48" />
                <Skeleton className="h-3 w-32" />
              </div>
              <Skeleton className="h-3.5 w-20 hidden md:block" />
              <Skeleton className="h-3.5 w-16 hidden md:block" />
              <Skeleton className="h-3 w-14 hidden md:block" />
            </div>
          ))}

          {!queueLoading && pendingListings?.data?.length === 0 && (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <CheckCircle2 className="w-8 h-8 text-emerald-400 mb-2" />
              <p className="text-sm font-medium text-gray-600">Queue is clear!</p>
              <p className="text-xs text-gray-400 mt-0.5">All listings have been reviewed.</p>
            </div>
          )}

          {!queueLoading && pendingListings?.data?.map((listing: PendingListing) => (
            <div
              key={listing.id}
              className="grid md:grid-cols-[1fr_160px_120px_100px_90px] gap-4 items-center px-5 py-3.5 hover:bg-gray-50/50 transition-colors group"
            >
              {/* Listing */}
              <div className="flex items-center gap-3 min-w-0">
                <img
                  src={listing.images?.[0]?.url || `https://picsum.photos/40/40?random=${listing.id}`}
                  className="w-10 h-10 rounded-lg object-cover shrink-0 border border-gray-100"
                  alt=""
                  onError={(e) => { (e.target as HTMLImageElement).src = `https://picsum.photos/40/40?random=${listing.id}` }}
                />
                <div className="min-w-0">
                  <p className="text-sm font-medium text-gray-900 truncate">{listing.title}</p>
                  <p className="text-xs text-gray-400 truncate md:hidden">
                    {listing.user?.name} · {listing.category?.name_en}
                  </p>
                </div>
              </div>

              {/* Seller */}
              <p className="text-xs text-gray-600 hidden md:block truncate">{listing.user?.name ?? "—"}</p>

              {/* Category */}
              <div className="hidden md:block">
                <span className="inline-block bg-gray-100 text-gray-600 text-[10px] font-semibold px-2 py-0.5 rounded-full truncate max-w-full">
                  {listing.category?.name_en ?? "—"}
                </span>
              </div>

              {/* Price */}
              <p className="text-sm font-semibold text-gray-900 hidden md:block tabular-nums">
                {listing.price ? `${listing.currency ?? ""} ${Number(listing.price).toLocaleString()}` : "—"}
              </p>

              {/* Date */}
              <div className="hidden md:flex items-center gap-1 text-xs text-gray-400">
                <Clock className="w-3 h-3 shrink-0" />
                <span className="truncate">
                  {(() => {
                    try { return formatDistanceToNow(new Date(listing.created_at), { addSuffix: true }) }
                    catch { return format(new Date(listing.created_at), "MMM d") }
                  })()}
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
