import { useState } from "react";
import { Link, useLocation } from "wouter";
import { useAuthStore } from "@/store/auth";
import { formatPrice } from "@/lib/utils";
import { User, Package, Heart, Gavel, Star, Store, Wallet, Settings, ChevronRight } from "lucide-react";

export default function ProfilePage() {
  const { user, isAuthenticated, logout } = useAuthStore();
  const [, navigate] = useLocation();

  if (!isAuthenticated || !user) {
    navigate("/login?next=/profile");
    return null;
  }

  const MENU_ITEMS = [
    { icon: <Package size={18} />, label: "My Listings", desc: "Manage your active listings", href: "/my-listings" },
    { icon: <Gavel size={18} />, label: "My Bids", desc: "Track auctions you're bidding on", href: "/my-bids" },
    { icon: <Heart size={18} />, label: "Saved Items", desc: "Listings you've favorited", href: "/favorites" },
    { icon: <Store size={18} />, label: "My Storefront", desc: "Manage your seller page", href: "/my-store" },
    { icon: <Wallet size={18} />, label: "Wallet", desc: "Balance & transaction history", href: "/wallet" },
    { icon: <Star size={18} />, label: "Reviews", desc: "Your buyer and seller reviews", href: "/reviews" },
  ];

  return (
    <div className="max-w-2xl mx-auto px-4 py-10">
      <div className="bg-white rounded-2xl shadow-sm p-6 mb-6">
        <div className="flex items-center gap-4">
          <div className="w-16 h-16 rounded-full bg-[#0071CE] flex items-center justify-center text-white text-2xl font-extrabold shrink-0">
            {user.name?.[0]?.toUpperCase() || <User size={28} />}
          </div>
          <div className="flex-1">
            <h1 className="text-xl font-bold text-gray-900">{user.name}</h1>
            <p className="text-sm text-gray-500">{user.email}</p>
            {user.phone && <p className="text-sm text-gray-400">{user.phone}</p>}
          </div>
          {user.isVerified && (
            <span className="bg-blue-50 text-blue-600 text-xs font-bold px-2 py-1 rounded-full">✓ Verified</span>
          )}
        </div>

        <div className="grid grid-cols-3 gap-4 mt-5 pt-5 border-t border-gray-100">
          <div className="text-center">
            <p className="text-xl font-bold text-[#0071CE]">{formatPrice(user.balance ?? 0, "AED")}</p>
            <p className="text-xs text-gray-400">Wallet Balance</p>
          </div>
          <div className="text-center">
            <p className="text-xl font-bold text-[#0071CE]">{user.rating?.toFixed(1) ?? "—"}</p>
            <p className="text-xs text-gray-400">Rating</p>
          </div>
          <div className="text-center">
            <p className="text-xl font-bold text-[#0071CE]">{user.location || "GCC"}</p>
            <p className="text-xs text-gray-400">Location</p>
          </div>
        </div>
      </div>

      <div className="bg-white rounded-2xl shadow-sm overflow-hidden mb-6">
        <ul className="divide-y divide-gray-50">
          {MENU_ITEMS.map((item) => (
            <li key={item.href}>
              <Link href={item.href} className="flex items-center gap-4 px-5 py-4 hover:bg-gray-50 transition-colors group">
                <div className="w-9 h-9 rounded-xl bg-blue-50 text-[#0071CE] flex items-center justify-center shrink-0 group-hover:bg-[#0071CE] group-hover:text-white transition-colors">
                  {item.icon}
                </div>
                <div className="flex-1">
                  <p className="text-sm font-semibold text-gray-800">{item.label}</p>
                  <p className="text-xs text-gray-400">{item.desc}</p>
                </div>
                <ChevronRight size={16} className="text-gray-300 group-hover:text-[#0071CE] transition-colors" />
              </Link>
            </li>
          ))}
        </ul>
      </div>

      <div className="bg-white rounded-2xl shadow-sm overflow-hidden">
        <ul className="divide-y divide-gray-50">
          <li>
            <Link href="/settings" className="flex items-center gap-4 px-5 py-4 hover:bg-gray-50 transition-colors group">
              <div className="w-9 h-9 rounded-xl bg-gray-100 text-gray-500 flex items-center justify-center shrink-0 group-hover:bg-gray-200 transition-colors">
                <Settings size={18} />
              </div>
              <div className="flex-1">
                <p className="text-sm font-semibold text-gray-800">Settings</p>
                <p className="text-xs text-gray-400">Account preferences & notifications</p>
              </div>
              <ChevronRight size={16} className="text-gray-300" />
            </Link>
          </li>
          <li>
            <button
              onClick={() => { logout(); navigate("/"); }}
              className="w-full flex items-center gap-4 px-5 py-4 hover:bg-red-50 transition-colors text-left"
            >
              <div className="w-9 h-9 rounded-xl bg-red-50 text-red-500 flex items-center justify-center shrink-0">
                <User size={18} />
              </div>
              <div className="flex-1">
                <p className="text-sm font-semibold text-red-500">Sign Out</p>
                <p className="text-xs text-gray-400">Sign out of your account</p>
              </div>
            </button>
          </li>
        </ul>
      </div>
    </div>
  );
}
