"use client";

import { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";
import { isSeller } from "@/lib/permissions";
import {
  LayoutDashboard,
  Package,
  ShoppingBag,
  BarChart2,
  Settings,
  Store,
  Plus,
  AlertTriangle,
} from "lucide-react";

const NAV = [
  { label: "Overview", href: "/seller", icon: LayoutDashboard, exact: true },
  { label: "Listings", href: "/seller/listings", icon: Package },
  { label: "Orders", href: "/seller/orders", icon: ShoppingBag },
  { label: "Analytics", href: "/seller/analytics", icon: BarChart2 },
  { label: "Settings", href: "/seller/settings", icon: Settings },
];

function SellerGuard({ children }: { children: React.ReactNode }) {
  const { user, isAuthenticated } = useAuthStore();
  const router = useRouter();

  useEffect(() => {
    if (!isAuthenticated) {
      router.push("/login?next=/seller");
      return;
    }
    if (user?.role && !isSeller(user.role)) {
      router.push("/sell");
    }
  }, [isAuthenticated, user, router]);

  if (!isAuthenticated) return null;
  if (user?.role && !isSeller(user.role)) {
    return (
      <div className="flex items-center justify-center min-h-[40vh]">
        <div className="flex items-center gap-3 bg-amber-50 border border-amber-200 rounded-xl px-5 py-4 text-sm text-amber-800">
          <AlertTriangle className="w-5 h-5 text-amber-500 shrink-0" />
          <span>Seller access required. <Link href="/sell" className="font-semibold underline">Become a seller →</Link></span>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}

export default function SellerLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { user } = useAuthStore();

  return (
    <SellerGuard>
      <div className="max-w-7xl mx-auto px-4 py-6">
        {/* ── Seller top navigation bar ── */}
        <div className="flex items-center justify-between mb-6 gap-4 flex-wrap">
          {/* Identity */}
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-full bg-[#0071CE] flex items-center justify-center text-white font-bold text-sm shrink-0">
              {user?.name?.charAt(0)?.toUpperCase() ?? "S"}
            </div>
            <div>
              <p className="text-sm font-semibold text-gray-900 leading-tight">{user?.name ?? "Seller"}</p>
              <p className="text-[11px] text-gray-400 leading-tight">Seller Dashboard</p>
            </div>
          </div>

          {/* Nav tabs */}
          <nav className="flex items-center gap-1 bg-gray-100 rounded-xl p-1 overflow-x-auto">
            {NAV.map(({ label, href, icon: Icon, exact }) => {
              const active = exact ? pathname === href : pathname.startsWith(href);
              return (
                <Link
                  key={href}
                  href={href}
                  className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[13px] font-medium whitespace-nowrap transition-all ${
                    active
                      ? "bg-white text-[#0071CE] shadow-sm"
                      : "text-gray-500 hover:text-gray-800"
                  }`}
                >
                  <Icon className="w-3.5 h-3.5" />
                  {label}
                </Link>
              );
            })}
          </nav>

          {/* Actions */}
          <div className="flex items-center gap-2">
            <Link
              href="/my-store"
              className="flex items-center gap-1.5 px-3 py-1.5 border border-gray-200 bg-white rounded-lg text-[13px] font-medium text-gray-600 hover:bg-gray-50 transition-colors"
            >
              <Store className="w-3.5 h-3.5" /> My Store
            </Link>
            <Link
              href="/sell"
              className="flex items-center gap-1.5 px-3 py-1.5 bg-[#0071CE] text-white rounded-lg text-[13px] font-semibold hover:bg-[#005ba3] transition-colors"
            >
              <Plus className="w-3.5 h-3.5" /> New Listing
            </Link>
          </div>
        </div>

        {/* ── Page content ── */}
        {children}
      </div>
    </SellerGuard>
  );
}
