import { Outlet, useLocation } from "react-router-dom"
import { Sidebar } from "./Sidebar"
import { Header } from "./Header"

const pageTitles: Record<string, string> = {
  "/admin": "Dashboard",
  "/admin/listings": "Listings",
  "/admin/auctions": "Auctions",
  "/admin/storefronts": "Storefronts",
  "/admin/categories": "Categories",
  "/admin/users": "Users",
  "/admin/reports": "Reports",
  "/admin/payments": "Payments",
  "/admin/pricing": "Price Plans",
  "/admin/transactions": "Transactions",
  "/admin/settings": "Settings",
  "/admin/emails": "Email Templates",
  "/admin/staff": "Staff Users",
  "/admin/logs": "Audit Logs",
}

export function Layout() {
  const location = useLocation()
  const title = pageTitles[location.pathname] ?? "Admin"

  return (
    <div className="flex h-screen bg-[#F7FAFC] overflow-hidden">
      <Sidebar />
      <div className="flex-1 flex flex-col overflow-hidden">
        <Header title={title} />
        <main className="flex-1 overflow-y-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
