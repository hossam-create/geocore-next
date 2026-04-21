"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  LayoutDashboard, Clock, Gavel, Heart, ShoppingBag,
  RefreshCcw, Tag, Store, Search, CreditCard, Wallet,
  Settings, MessageSquare, LogOut, User, ChevronDown,
  Rss, Users, Car, Library, Shield,
} from "lucide-react";
import { useAuthStore } from "@/store/auth";

interface MenuItem {
  type?: "divider";
  label?: string;
  icon?: React.ReactNode;
  href?: string;
  action?: "logout";
  variant?: "danger";
}

const MENU: MenuItem[] = [
  { label: "Summary",           icon: <LayoutDashboard size={14} />, href: "/dashboard" },
  { label: "Recently Viewed",   icon: <Clock size={14} />,          href: "/recently-viewed" },
  { label: "Bids/Offers",       icon: <Gavel size={14} />,          href: "/my-bids" },
  { label: "Watchlist",         icon: <Heart size={14} />,          href: "/buyer/watchlist" },
  { label: "Purchase History",  icon: <ShoppingBag size={14} />,    href: "/buyer/orders" },
  { label: "Buy Again",         icon: <RefreshCcw size={14} />,     href: "/buyer/orders?filter=delivered" },
  { type: "divider" },
  { label: "Selling",           icon: <Tag size={14} />,            href: "/seller" },
  { label: "Saved Feed",        icon: <Rss size={14} />,            href: "/buyer/watchlist" },
  { label: "Saved Searches",    icon: <Search size={14} />,         href: "/saved-searches" },
  { label: "Saved Sellers",     icon: <Users size={14} />,          href: "/stores" },
  { type: "divider" },
  { label: "Payments",          icon: <CreditCard size={14} />,     href: "/payments" },
  { label: "My Garage",         icon: <Car size={14} />,            href: "/profile" },
  { label: "Preferences",       icon: <Settings size={14} />,       href: "/buyer/settings" },
  { label: "My Collection",     icon: <Library size={14} />,        href: "/profile" },
  { label: "Messages",          icon: <MessageSquare size={14} />,  href: "/messages" },
  { label: "PSA Vault",         icon: <Shield size={14} />,         href: "/profile" },
  { type: "divider" },
  { label: "My Store",          icon: <Store size={14} />,          href: "/my-store" },
  { label: "Wallet",            icon: <Wallet size={14} />,         href: "/wallet" },
];

export function AccountDropdown() {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const closeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const { user, isAuthenticated, logout } = useAuthStore();
  const router = useRouter();

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const openMenu = () => { if (closeTimer.current) clearTimeout(closeTimer.current); setOpen(true); };
  const scheduleClose = () => { closeTimer.current = setTimeout(() => setOpen(false), 180); };

  const handleAction = (item: MenuItem) => {
    setOpen(false);
    if (item.action === "logout") { logout(); router.push("/"); return; }
    if (item.href) router.push(item.href);
  };

  if (!isAuthenticated) {
    return (
      <div className="relative" ref={ref} onMouseEnter={openMenu} onMouseLeave={scheduleClose}>
        <button className="flex items-center gap-0.5 font-semibold text-xs text-white/80 hover:text-white transition-colors">
          My Mnbarh <ChevronDown size={10} className={`transition-transform ${open ? "rotate-180" : ""}`} />
        </button>
        {open && (
          <div className="absolute right-0 top-full mt-1 w-56 bg-white rounded-xl shadow-2xl border border-gray-100 z-[90] text-gray-800 py-3">
            <div className="px-4 pb-3 border-b border-gray-100 mb-1">
              <p className="text-sm font-bold text-gray-900 mb-2">Hi there!</p>
              <div className="flex gap-2">
                <Link href="/login" onClick={() => setOpen(false)} className="flex-1 bg-[#0071CE] text-white text-xs font-bold py-1.5 rounded-full text-center hover:bg-[#005ba3] transition-colors">Sign in</Link>
                <Link href="/register" onClick={() => setOpen(false)} className="flex-1 border border-gray-300 text-gray-700 text-xs font-bold py-1.5 rounded-full text-center hover:bg-gray-50 transition-colors">Register</Link>
              </div>
            </div>
            {MENU.slice(0, 4).map((item, i) =>
              item.type === "divider" ? null : (
                <Link key={i} href={item.href!} onClick={() => setOpen(false)} className="flex items-center gap-2.5 px-4 py-2 text-sm text-gray-700 hover:bg-blue-50 hover:text-[#0071CE] transition-colors">
                  <span className="text-gray-400">{item.icon}</span>{item.label}
                </Link>
              )
            )}
          </div>
        )}
      </div>
    );
  }

  return (
    <div className="relative" ref={ref} onMouseEnter={openMenu} onMouseLeave={scheduleClose}>
      <button className="flex items-center gap-0.5 font-semibold text-xs text-white/80 hover:text-white transition-colors">
        My Mnbarh <ChevronDown size={10} className={`transition-transform ${open ? "rotate-180" : ""}`} />
      </button>

      {open && (
        <div className="absolute right-0 top-full mt-1 w-72 bg-white rounded-xl shadow-2xl border border-gray-100 z-[90] text-gray-800 flex flex-col" style={{ maxHeight: 'calc(100vh - 80px)' }}>
          {/* User info — sticky at top */}
          <div className="flex items-center gap-3 px-4 py-3 bg-blue-50/60 border-b border-gray-100 shrink-0">
            <div className="w-8 h-8 rounded-full bg-[#0071CE] text-white font-bold text-sm flex items-center justify-center shrink-0">
              {user?.name?.[0]?.toUpperCase() ?? <User size={14} />}
            </div>
            <div className="min-w-0">
              <p className="text-sm font-bold text-gray-900 truncate">{user?.name}</p>
              <p className="text-[10px] text-gray-500 truncate">{user?.email}</p>
            </div>
          </div>

          {/* Scrollable menu items */}
          <div className="py-1.5 overflow-y-auto flex-1">
            {MENU.map((item, i) => {
              if (item.type === "divider") return <div key={i} className="my-1 border-t border-gray-100" />;
              return (
                <button
                  key={i}
                  onClick={() => handleAction(item)}
                  className={`w-full flex items-center gap-2.5 px-4 py-2 text-sm transition-colors text-left ${
                    item.variant === "danger"
                      ? "text-red-600 hover:bg-red-50"
                      : "text-gray-700 hover:bg-blue-50 hover:text-[#0071CE]"
                  }`}
                >
                  <span className={item.variant === "danger" ? "text-red-400" : "text-gray-400"}>{item.icon}</span>
                  {item.label}
                </button>
              );
            })}
          </div>

          {/* Sign Out pinned at bottom */}
          <div className="border-t border-gray-100 shrink-0">
            <button
              onClick={() => { setOpen(false); logout(); router.push("/"); }}
              className="w-full flex items-center gap-2.5 px-4 py-3 text-sm text-red-600 hover:bg-red-50 transition-colors text-left"
            >
              <LogOut size={14} className="text-red-400" />
              Sign Out
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
