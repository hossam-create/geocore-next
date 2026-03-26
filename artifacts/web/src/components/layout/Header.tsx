import { useState } from "react";
import { Link, useLocation } from "wouter";
import { Heart, MessageCircle, User, Search, LogOut, Wallet, Store, LayoutDashboard, Sparkles } from "lucide-react";
import { useAuthStore } from "@/store/auth";
import { formatPrice } from "@/lib/utils";

const NAV_LINKS = [
  { label: "All Listings", href: "/listings" },
  { label: "🚗 Vehicles", href: "/listings?category=vehicles" },
  { label: "🏠 Real Estate", href: "/listings?category=real-estate" },
  { label: "📱 Electronics", href: "/listings?category=electronics" },
  { label: "👕 Clothing", href: "/listings?category=clothing" },
  { label: "🛋️ Furniture", href: "/listings?category=furniture" },
  { label: "🏪 Stores", href: "/stores" },
  { label: "⚡ Auctions", href: "/auctions" },
];

export function Header() {
  const [query, setQuery] = useState("");
  const [, navigate] = useLocation();
  const { user, isAuthenticated, logout } = useAuthStore();

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (query.trim()) navigate(`/search?q=${encodeURIComponent(query.trim())}`);
  };

  return (
    <header className="bg-[#0071CE] text-white sticky top-0 z-50 shadow-lg">
      <div className="max-w-7xl mx-auto px-4">
        <div className="flex items-center gap-4 py-3">
          <Link href="/" className="text-[#FFC220] font-extrabold text-2xl tracking-wide shrink-0 hover:text-yellow-300 transition-colors">
            GeoCore
          </Link>

          <form onSubmit={handleSearch} className="flex-1 flex max-w-2xl relative">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={15} />
              <input
                type="text"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Search in Arabic or English…"
                className="w-full pl-9 pr-24 py-2.5 text-gray-800 text-sm rounded-l-lg outline-none focus:ring-2 focus:ring-yellow-400"
              />
              <span className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-1 text-[10px] font-semibold text-[#0071CE] bg-blue-50 px-1.5 py-0.5 rounded-full pointer-events-none">
                <Sparkles size={9} /> AI
              </span>
            </div>
            <button
              type="submit"
              className="bg-[#FFC220] text-gray-900 px-5 py-2.5 rounded-r-lg font-bold text-sm hover:bg-yellow-400 transition-colors flex items-center gap-1 shrink-0"
            >
              <Search size={14} /> Search
            </button>
          </form>

          <div className="flex items-center gap-4 shrink-0">
            <Link href="/favorites" className="hidden md:flex flex-col items-center text-xs hover:text-[#FFC220] transition-colors gap-0.5">
              <Heart size={20} />
              <span>Saved</span>
            </Link>
            <Link href="/messages" className="hidden md:flex flex-col items-center text-xs hover:text-[#FFC220] transition-colors gap-0.5">
              <MessageCircle size={20} />
              <span>Messages</span>
            </Link>

            {isAuthenticated ? (
              <>
                <Link href="/wallet" className="hidden md:flex flex-col items-center text-xs hover:text-[#FFC220] transition-colors gap-0.5 group">
                  <Wallet size={20} />
                  <span className="text-[#FFC220] font-semibold">
                    {formatPrice(user?.balance ?? 0, "AED")}
                  </span>
                </Link>
                <Link href="/profile" className="flex flex-col items-center text-xs hover:text-[#FFC220] transition-colors gap-0.5">
                  <div className="w-7 h-7 rounded-full bg-[#FFC220] text-gray-900 font-bold text-xs flex items-center justify-center">
                    {user?.name?.[0]?.toUpperCase() || <User size={14} />}
                  </div>
                  <span>{user?.name?.split(" ")[0]}</span>
                </Link>
                <Link href="/dashboard" className="hidden lg:flex flex-col items-center text-xs hover:text-[#FFC220] transition-colors gap-0.5">
                  <LayoutDashboard size={20} />
                  <span>Dashboard</span>
                </Link>
                <Link href="/my-store" className="hidden lg:flex flex-col items-center text-xs hover:text-[#FFC220] transition-colors gap-0.5">
                  <Store size={20} />
                  <span>My Store</span>
                </Link>
                <button
                  onClick={logout}
                  className="hidden md:flex flex-col items-center text-xs hover:text-[#FFC220] transition-colors gap-0.5"
                  title="Sign Out"
                >
                  <LogOut size={20} />
                  <span>Sign Out</span>
                </button>
              </>
            ) : (
              <Link href="/login" className="flex flex-col items-center text-xs hover:text-[#FFC220] transition-colors gap-0.5">
                <User size={20} />
                <span>Sign In</span>
              </Link>
            )}

            <Link href="/sell" className="flex items-center gap-1 bg-[#FFC220] text-gray-900 text-xs font-bold px-3 py-2 rounded-lg hover:bg-yellow-400 transition-colors">
              + Post Listing
            </Link>
          </div>
        </div>

        <nav className="flex items-center gap-6 py-2 text-sm overflow-x-auto scrollbar-none border-t border-blue-500 border-opacity-40">
          {NAV_LINKS.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className="whitespace-nowrap hover:text-[#FFC220] transition-colors text-blue-100"
            >
              {item.label}
            </Link>
          ))}
        </nav>
      </div>
    </header>
  );
}
