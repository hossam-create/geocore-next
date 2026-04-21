import { Suspense, lazy, useEffect } from "react"
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { useAuthStore } from "@/store/auth"
import { Layout } from "@/components/layout/Layout"
import { PERMISSIONS, hasAnyPermission } from "@/lib/permissions"
import { LoginPage } from "@/pages/LoginPage"

const DashboardPage = lazy(() => import("@/pages/DashboardPage").then((mod) => ({ default: mod.DashboardPage })))
const ListingsPage = lazy(() => import("@/pages/listings/ListingsPage").then((mod) => ({ default: mod.ListingsPage })))
const AuctionsPage = lazy(() => import("@/pages/auctions/AuctionsPage").then((mod) => ({ default: mod.AuctionsPage })))
const UsersPage = lazy(() => import("@/pages/users/UsersPage").then((mod) => ({ default: mod.UsersPage })))
const ReportsPage = lazy(() => import("@/pages/reports/ReportsPage").then((mod) => ({ default: mod.ReportsPage })))
const PaymentsPage = lazy(() => import("@/pages/payments/PaymentsPage").then((mod) => ({ default: mod.PaymentsPage })))
const CategoriesPage = lazy(() => import("@/pages/categories/CategoriesPage").then((mod) => ({ default: mod.CategoriesPage })))
const PricingPage = lazy(() => import("@/pages/pricing/PricingPage").then((mod) => ({ default: mod.PricingPage })))
const SettingsPage = lazy(() => import("@/pages/settings/SettingsPage").then((mod) => ({ default: mod.SettingsPage })))
const LogsPage = lazy(() => import("@/pages/logs/LogsPage").then((mod) => ({ default: mod.LogsPage })))

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
})

function RequireAuth({ children }: { children: React.ReactNode }) {
  const { token } = useAuthStore()
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}

function RequirePermission({
  any,
  children,
}: {
  any: Array<(typeof PERMISSIONS)[keyof typeof PERMISSIONS]>
  children: React.ReactNode
}) {
  const { user } = useAuthStore()
  const role = user?.role
  if (!hasAnyPermission(role, any)) return <Navigate to="/admin" replace />
  return <>{children}</>
}

function App() {
  const { restore } = useAuthStore()

  const routeFallback = (
    <div className="p-6 text-sm text-gray-500">Loading...</div>
  )

  useEffect(() => {
    restore()
  }, [restore])

  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route
            path="/admin"
            element={
              <RequireAuth>
                <Layout />
              </RequireAuth>
            }
          >
            <Route path="" element={<RequirePermission any={[PERMISSIONS.ADMIN_DASHBOARD_READ]}><Suspense fallback={routeFallback}><DashboardPage /></Suspense></RequirePermission>} />
            <Route path="listings" element={<RequirePermission any={[PERMISSIONS.LISTINGS_MODERATE]}><Suspense fallback={routeFallback}><ListingsPage /></Suspense></RequirePermission>} />
            <Route path="auctions" element={<RequirePermission any={[PERMISSIONS.LISTINGS_MODERATE]}><Suspense fallback={routeFallback}><AuctionsPage /></Suspense></RequirePermission>} />
            <Route path="users" element={<RequirePermission any={[PERMISSIONS.USERS_READ]}><Suspense fallback={routeFallback}><UsersPage /></Suspense></RequirePermission>} />
            <Route path="reports" element={<RequirePermission any={[PERMISSIONS.REPORTS_REVIEW]}><Suspense fallback={routeFallback}><ReportsPage /></Suspense></RequirePermission>} />
            <Route path="payments" element={<RequirePermission any={[PERMISSIONS.FINANCE_READ]}><Suspense fallback={routeFallback}><PaymentsPage /></Suspense></RequirePermission>} />
            <Route path="transactions" element={<RequirePermission any={[PERMISSIONS.FINANCE_READ]}><Suspense fallback={routeFallback}><PaymentsPage /></Suspense></RequirePermission>} />
            <Route path="categories" element={<RequirePermission any={[PERMISSIONS.CATALOG_MANAGE]}><Suspense fallback={routeFallback}><CategoriesPage /></Suspense></RequirePermission>} />
            <Route path="pricing" element={<RequirePermission any={[PERMISSIONS.PLANS_MANAGE, PERMISSIONS.FINANCE_READ]}><Suspense fallback={routeFallback}><PricingPage /></Suspense></RequirePermission>} />
            <Route path="settings" element={<RequirePermission any={[PERMISSIONS.SETTINGS_READ, PERMISSIONS.SETTINGS_WRITE]}><Suspense fallback={routeFallback}><SettingsPage /></Suspense></RequirePermission>} />
            <Route path="logs" element={<RequirePermission any={[PERMISSIONS.AUDIT_LOGS_READ]}><Suspense fallback={routeFallback}><LogsPage /></Suspense></RequirePermission>} />
            <Route path="*" element={<Navigate to="/admin" replace />} />
          </Route>
          <Route path="/" element={<Navigate to="/admin" replace />} />
          <Route path="*" element={<Navigate to="/admin" replace />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}

export default App
