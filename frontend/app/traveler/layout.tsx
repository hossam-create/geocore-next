"use client";

import { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";
import { isTraveler } from "@/lib/permissions";
import {
  LayoutDashboard, Plane, Package, DollarSign, Plus, AlertTriangle,
} from "lucide-react";

const NAV = [
  { label: "Overview",  href: "/traveler",          icon: LayoutDashboard, exact: true },
  { label: "My Trips",  href: "/traveler/trips",     icon: Plane },
  { label: "Orders",    href: "/traveler/orders",    icon: Package },
  { label: "Earnings",  href: "/traveler/earnings",  icon: DollarSign },
];

export default function TravelerLayout({ children }: { children: React.ReactNode }) {
  const { user, isAuthenticated } = useAuthStore();
  const pathname = usePathname();
  const router = useRouter();

  useEffect(() => {
    if (!isAuthenticated) {
      router.push("/login?next=/traveler");
      return;
    }
    if (user?.role && !isTraveler(user.role)) {
      router.push("/traveler/onboarding");
    }
  }, [isAuthenticated, user, router]);

  if (!isAuthenticated) return null;

  if (user?.role && !isTraveler(user.role)) {
    return (
      <div className="flex items-center justify-center min-h-[40vh] px-4">
        <div className="flex items-center gap-3 bg-amber-50 border border-amber-200 rounded-xl px-5 py-4 text-sm text-amber-800">
          <AlertTriangle className="w-5 h-5 text-amber-500 shrink-0" />
          <span>Traveler access required. You need to activate your traveler account first.</span>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-6xl mx-auto px-4 py-6">
      {/* ── Traveler top bar ── */}
      <div className="flex items-center justify-between mb-6 gap-4 flex-wrap">
        {/* Identity */}
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-full bg-[#0071CE] flex items-center justify-center text-white font-bold text-sm shrink-0">
            <Plane className="w-4 h-4" />
          </div>
          <div>
            <p className="text-sm font-semibold text-gray-900 leading-tight">{user?.name ?? "Traveler"}</p>
            <p className="text-[11px] text-gray-400 leading-tight">Traveler Dashboard</p>
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
            href="/traveler/orders"
            className="flex items-center gap-1.5 px-3 py-1.5 border border-gray-200 bg-white rounded-lg text-[13px] font-medium text-gray-600 hover:bg-gray-50 transition-colors"
          >
            <Package className="w-3.5 h-3.5" /> Browse Orders
          </Link>
          <Link
            href="/traveler/trips/new"
            className="flex items-center gap-1.5 px-3 py-1.5 bg-[#0071CE] text-white rounded-lg text-[13px] font-semibold hover:bg-[#005ba3] transition-colors"
          >
            <Plus className="w-3.5 h-3.5" /> Post Trip
          </Link>
        </div>
      </div>

      {/* ── Page content ── */}
      {children}
    </div>
  );
}
