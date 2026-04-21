"use client";

import { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";
import {
  LayoutDashboard, Package, Heart, AlertCircle,
  Settings, Search, Wallet,
} from "lucide-react";

const NAV = [
  { label: "Overview",  href: "/buyer",            icon: LayoutDashboard, exact: true },
  { label: "Orders",    href: "/buyer/orders",      icon: Package },
  { label: "Watchlist", href: "/buyer/watchlist",   icon: Heart },
  { label: "Disputes",  href: "/buyer/disputes",    icon: AlertCircle },
  { label: "Settings",  href: "/buyer/settings",    icon: Settings },
];

export default function BuyerLayout({ children }: { children: React.ReactNode }) {
  const { user, isAuthenticated } = useAuthStore();
  const pathname = usePathname();
  const router = useRouter();

  useEffect(() => {
    if (!isAuthenticated) router.push("/login?next=/buyer");
  }, [isAuthenticated, router]);

  if (!isAuthenticated) return null;

  return (
    <div className="max-w-6xl mx-auto px-4 py-6">
      {/* ── Buyer top bar ── */}
      <div className="flex items-center justify-between mb-6 gap-4 flex-wrap">
        {/* Identity */}
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-full bg-indigo-600 flex items-center justify-center text-white font-bold text-sm shrink-0">
            {user?.name?.charAt(0)?.toUpperCase() ?? "B"}
          </div>
          <div>
            <p className="text-sm font-semibold text-gray-900 leading-tight">{user?.name ?? "Buyer"}</p>
            <p className="text-[11px] text-gray-400 leading-tight">Buyer Dashboard</p>
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
                    ? "bg-white text-indigo-600 shadow-sm"
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
            href="/listings"
            className="flex items-center gap-1.5 px-3 py-1.5 border border-gray-200 bg-white rounded-lg text-[13px] font-medium text-gray-600 hover:bg-gray-50 transition-colors"
          >
            <Search className="w-3.5 h-3.5" /> Browse
          </Link>
          <Link
            href="/wallet"
            className="flex items-center gap-1.5 px-3 py-1.5 bg-indigo-600 text-white rounded-lg text-[13px] font-semibold hover:bg-indigo-700 transition-colors"
          >
            <Wallet className="w-3.5 h-3.5" /> Wallet
          </Link>
        </div>
      </div>

      {/* ── Page content ── */}
      {children}
    </div>
  );
}
