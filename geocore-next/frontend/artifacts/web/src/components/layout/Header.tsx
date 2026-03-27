import { useState, useRef } from "react";
import { Link, useLocation } from "wouter";
import {
  Search, User, LogOut, Store, LayoutDashboard,
  ChevronDown, ShoppingCart, Menu, MapPin, Camera, Bell,
  SlidersHorizontal, ChevronRight,
} from "lucide-react";
import { useAuthStore } from "@/store/auth";
import { formatPrice } from "@/lib/utils";

/* ─── Mega Menu Data ─── */
interface MegaMenuData {
  popular: { label: string; href: string }[];
  more: { label: string; href: string }[];
  promo: { title: string; subtitle: string; cta: string; imgSeed: string; href: string; accent: string };
}

const MEGA_MENUS: Record<string, MegaMenuData> = {
  electronics: {
    popular: [
      { label: "Smartphones & Accessories", href: "/listings?category=electronics&q=smartphone" },
      { label: "Laptops & Computers", href: "/listings?category=electronics&q=laptop" },
      { label: "Tablets", href: "/listings?category=electronics&q=tablet" },
      { label: "Cameras & Photography", href: "/listings?category=electronics&q=camera" },
      { label: "Smart Home Devices", href: "/listings?category=electronics&q=smart+home" },
      { label: "TV & Audio", href: "/listings?category=electronics&q=tv" },
      { label: "Refurbished Electronics", href: "/listings?category=electronics&q=refurbished" },
    ],
    more: [
      { label: "Apple", href: "/listings?category=electronics&q=apple" },
      { label: "Samsung", href: "/listings?category=electronics&q=samsung" },
      { label: "Sony", href: "/listings?category=electronics&q=sony" },
      { label: "Xiaomi", href: "/listings?category=electronics&q=xiaomi" },
      { label: "Gaming Consoles", href: "/listings?category=gaming" },
      { label: "Flash Deals", href: "/listings?category=electronics&sort=price_asc" },
    ],
    promo: { title: "Electronics", subtitle: "Smart devices, always with you.", cta: "Explore now", imgSeed: "smartphone-laptop-tech", href: "/listings?category=electronics", accent: "#0071CE" },
  },
  vehicles: {
    popular: [
      { label: "Cars — Sedans & Saloon", href: "/listings?category=vehicles&q=sedan" },
      { label: "SUVs & 4WD", href: "/listings?category=vehicles&q=suv" },
      { label: "Motorcycles", href: "/listings?category=vehicles&q=motorcycle" },
      { label: "Boats & Marine", href: "/listings?category=vehicles&q=boat" },
      { label: "Heavy Vehicles & Trucks", href: "/listings?category=vehicles&q=truck" },
      { label: "Vehicle Parts & Accessories", href: "/listings?category=vehicles&q=parts" },
      { label: "Electric Vehicles", href: "/listings?category=vehicles&q=electric" },
    ],
    more: [
      { label: "Toyota", href: "/listings?category=vehicles&q=toyota" },
      { label: "BMW", href: "/listings?category=vehicles&q=bmw" },
      { label: "Mercedes-Benz", href: "/listings?category=vehicles&q=mercedes" },
      { label: "Nissan", href: "/listings?category=vehicles&q=nissan" },
      { label: "Auctions Only", href: "/auctions" },
      { label: "New Arrivals", href: "/listings?category=vehicles&sort=newest" },
    ],
    promo: { title: "Vehicles", subtitle: "Find your next car in the GCC.", cta: "Browse vehicles", imgSeed: "luxury-car-road-gcc", href: "/listings?category=vehicles", accent: "#1e40af" },
  },
  clothing: {
    popular: [
      { label: "Women's Clothing", href: "/listings?category=clothing&q=women" },
      { label: "Men's Fashion", href: "/listings?category=clothing&q=men" },
      { label: "Abayas & Kaftans", href: "/listings?category=clothing&q=abaya" },
      { label: "Shoes & Footwear", href: "/listings?category=clothing&q=shoes" },
      { label: "Bags & Purses", href: "/listings?category=clothing&q=bag" },
      { label: "Kids' Clothing", href: "/listings?category=clothing&q=kids" },
      { label: "Luxury Fashion", href: "/listings?category=clothing&q=luxury" },
    ],
    more: [
      { label: "Gucci", href: "/listings?category=clothing&q=gucci" },
      { label: "Chanel", href: "/listings?category=clothing&q=chanel" },
      { label: "Nike & Adidas", href: "/listings?category=clothing&q=nike" },
      { label: "Louis Vuitton", href: "/listings?category=clothing&q=louis+vuitton" },
      { label: "New Arrivals", href: "/listings?category=clothing&sort=newest" },
      { label: "On Sale", href: "/listings?category=clothing&sort=price_asc" },
    ],
    promo: { title: "Fashion", subtitle: "Top brands. Unbeatable prices.", cta: "Shop fashion", imgSeed: "fashion-dress-luxury-gcc", href: "/listings?category=clothing", accent: "#7c3aed" },
  },
  "real-estate": {
    popular: [
      { label: "Apartments for Sale", href: "/listings?category=real-estate&q=apartment" },
      { label: "Villas & Houses", href: "/listings?category=real-estate&q=villa" },
      { label: "Apartments for Rent", href: "/listings?category=real-estate&q=rent" },
      { label: "Commercial Properties", href: "/listings?category=real-estate&q=commercial" },
      { label: "Land & Plots", href: "/listings?category=real-estate&q=land" },
      { label: "Off-Plan Projects", href: "/listings?category=real-estate&q=off+plan" },
      { label: "Holiday Homes", href: "/listings?category=real-estate&q=holiday" },
    ],
    more: [
      { label: "Dubai", href: "/listings?category=real-estate&city=Dubai" },
      { label: "Abu Dhabi", href: "/listings?category=real-estate&city=Abu+Dhabi" },
      { label: "Riyadh", href: "/listings?category=real-estate&city=Riyadh" },
      { label: "Kuwait City", href: "/listings?category=real-estate&city=Kuwait+City" },
      { label: "Doha", href: "/listings?category=real-estate&city=Doha" },
      { label: "New Projects", href: "/listings?category=real-estate&sort=newest" },
    ],
    promo: { title: "Real Estate", subtitle: "Premium properties across the GCC.", cta: "Find property", imgSeed: "dubai-skyline-luxury-apartment", href: "/listings?category=real-estate", accent: "#059669" },
  },
  jewelry: {
    popular: [
      { label: "Gold Jewelry", href: "/listings?category=jewelry&q=gold" },
      { label: "Diamond Rings", href: "/listings?category=jewelry&q=diamond+ring" },
      { label: "Luxury Watches", href: "/listings?category=jewelry&q=watch" },
      { label: "Necklaces & Chains", href: "/listings?category=jewelry&q=necklace" },
      { label: "Bracelets", href: "/listings?category=jewelry&q=bracelet" },
      { label: "Earrings", href: "/listings?category=jewelry&q=earring" },
      { label: "Gemstones", href: "/listings?category=jewelry&q=gemstone" },
    ],
    more: [
      { label: "Rolex", href: "/listings?category=jewelry&q=rolex" },
      { label: "Cartier", href: "/listings?category=jewelry&q=cartier" },
      { label: "Patek Philippe", href: "/listings?category=jewelry&q=patek" },
      { label: "Tiffany & Co.", href: "/listings?category=jewelry&q=tiffany" },
      { label: "Live Auctions", href: "/auctions" },
      { label: "New Arrivals", href: "/listings?category=jewelry&sort=newest" },
    ],
    promo: { title: "Jewelry & Watches", subtitle: "Authentic luxury from trusted GCC sellers.", cta: "Browse jewelry", imgSeed: "rolex-gold-watch-diamond", href: "/listings?category=jewelry", accent: "#b45309" },
  },
  furniture: {
    popular: [
      { label: "Sofas & Living Room", href: "/listings?category=furniture&q=sofa" },
      { label: "Bedroom Sets", href: "/listings?category=furniture&q=bedroom" },
      { label: "Dining Tables", href: "/listings?category=furniture&q=dining" },
      { label: "Office Furniture", href: "/listings?category=furniture&q=office" },
      { label: "Kitchen Appliances", href: "/listings?category=furniture&q=kitchen" },
      { label: "Outdoor & Garden", href: "/listings?category=furniture&q=outdoor" },
      { label: "Home Decor", href: "/listings?category=furniture&q=decor" },
    ],
    more: [
      { label: "IKEA", href: "/listings?category=furniture&q=ikea" },
      { label: "Ashley", href: "/listings?category=furniture&q=ashley" },
      { label: "Used Furniture", href: "/listings?category=furniture&condition=used" },
      { label: "New Condition", href: "/listings?category=furniture&condition=new" },
      { label: "Flash Deals", href: "/listings?category=furniture&sort=price_asc" },
      { label: "New Arrivals", href: "/listings?category=furniture&sort=newest" },
    ],
    promo: { title: "Furniture & Home", subtitle: "Style your space for less.", cta: "Shop home", imgSeed: "living-room-sofa-modern", href: "/listings?category=furniture", accent: "#0071CE" },
  },
  sports: {
    popular: [
      { label: "Fitness Equipment", href: "/listings?category=sports&q=fitness" },
      { label: "Football & Soccer", href: "/listings?category=sports&q=football" },
      { label: "Cycling & Bikes", href: "/listings?category=sports&q=cycling" },
      { label: "Swimming & Water Sports", href: "/listings?category=sports&q=swimming" },
      { label: "Gym Equipment", href: "/listings?category=sports&q=gym" },
      { label: "Outdoor & Camping", href: "/listings?category=sports&q=camping" },
      { label: "Sports Clothing", href: "/listings?category=sports&q=sportswear" },
    ],
    more: [
      { label: "Nike", href: "/listings?category=sports&q=nike" },
      { label: "Adidas", href: "/listings?category=sports&q=adidas" },
      { label: "Under Armour", href: "/listings?category=sports&q=under+armour" },
      { label: "Used Equipment", href: "/listings?category=sports&condition=used" },
      { label: "New Arrivals", href: "/listings?category=sports&sort=newest" },
      { label: "Deals", href: "/listings?category=sports&sort=price_asc" },
    ],
    promo: { title: "Sports & Outdoors", subtitle: "Equipment for every sport.", cta: "Shop sports", imgSeed: "sports-fitness-gym-dubai", href: "/listings?category=sports", accent: "#16a34a" },
  },
  gaming: {
    popular: [
      { label: "PlayStation 5", href: "/listings?category=gaming&q=ps5" },
      { label: "Xbox Series X|S", href: "/listings?category=gaming&q=xbox" },
      { label: "Nintendo Switch", href: "/listings?category=gaming&q=nintendo" },
      { label: "PC Gaming", href: "/listings?category=gaming&q=gaming+pc" },
      { label: "Games & Titles", href: "/listings?category=gaming&q=game+disc" },
      { label: "Controllers & Accessories", href: "/listings?category=gaming&q=controller" },
      { label: "Gaming Chairs & Desks", href: "/listings?category=gaming&q=gaming+chair" },
    ],
    more: [
      { label: "Sony PlayStation", href: "/listings?category=gaming&q=sony" },
      { label: "Microsoft Xbox", href: "/listings?category=gaming&q=microsoft" },
      { label: "Nintendo", href: "/listings?category=gaming&q=nintendo" },
      { label: "VR Headsets", href: "/listings?category=gaming&q=vr" },
      { label: "Auctions", href: "/auctions" },
      { label: "New Arrivals", href: "/listings?category=gaming&sort=newest" },
    ],
    promo: { title: "Gaming", subtitle: "Level up your game.", cta: "Shop gaming", imgSeed: "ps5-controller-gaming-setup", href: "/listings?category=gaming", accent: "#7c3aed" },
  },
};

const CATEGORY_LINKS = [
  { label: "⚡ Auctions", href: "/auctions", key: null },
  { label: "Electronics", href: "/listings?category=electronics", key: "electronics" },
  { label: "Vehicles", href: "/listings?category=vehicles", key: "vehicles" },
  { label: "Fashion", href: "/listings?category=clothing", key: "clothing" },
  { label: "Real Estate", href: "/listings?category=real-estate", key: "real-estate" },
  { label: "Jewelry", href: "/listings?category=jewelry", key: "jewelry" },
  { label: "Furniture", href: "/listings?category=furniture", key: "furniture" },
  { label: "Sports", href: "/listings?category=sports", key: "sports" },
  { label: "Gaming", href: "/listings?category=gaming", key: "gaming" },
  { label: "🏷️ Brand Outlet", href: "/brand-outlet", key: null },
];

const ALL_CATS = [
  "All Categories", "Vehicles", "Real Estate", "Electronics",
  "Fashion", "Furniture", "Jewelry", "Tools", "Gaming", "Books", "Sports",
];

export function Header() {
  const [query, setQuery] = useState("");
  const [selectedCat, setSelectedCat] = useState("All Categories");
  const [catOpen, setCatOpen] = useState(false);
  const [activeMenu, setActiveMenu] = useState<string | null>(null);
  const closeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [watchlistOpen, setWatchlistOpen] = useState(false);
  const watchTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [notifOpen, setNotifOpen] = useState(false);
  const notifTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [, navigate] = useLocation();
  const { user, isAuthenticated, logout } = useAuthStore();

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (query.trim()) navigate(`/search?q=${encodeURIComponent(query.trim())}`);
    else navigate("/listings");
  };

  const openMenu = (key: string | null) => {
    if (closeTimer.current) clearTimeout(closeTimer.current);
    setActiveMenu(key);
  };

  const scheduleClose = () => {
    closeTimer.current = setTimeout(() => setActiveMenu(null), 180);
  };

  const activeMegaData = activeMenu && activeMenu !== "more" ? MEGA_MENUS[activeMenu] : null;
  const moreMenuOpen = activeMenu === "more";

  return (
    /* position:relative so absolute mega-panel is positioned against the header */
    <header className="bg-[#0071CE] sticky top-0 z-50 shadow-md" style={{ position: "sticky" }}>

      {/* ── Top utility bar ── */}
      <div className="bg-[#004F9A] text-white/80 text-xs">
        <div className="max-w-7xl mx-auto px-4 flex items-center justify-between h-7">
          <div className="flex items-center gap-4">
            {isAuthenticated ? (
              <span className="text-white/90">
                Welcome, <span className="text-[#FFC220] font-semibold">{user?.name?.split(" ")[0]}</span>!
              </span>
            ) : (
              <span>
                <Link href="/login" className="text-[#FFC220] hover:underline font-semibold">Sign in</Link>
                {" "}or{" "}
                <Link href="/register" className="text-[#FFC220] hover:underline font-semibold">Register</Link>
              </span>
            )}
            <span className="text-white/30">|</span>
            <Link href="/auctions" className="hover:text-white transition-colors">Daily Deals</Link>
            <Link href="/brand-outlet" className="hover:text-white transition-colors">Brand Outlet</Link>
            <Link href="#" className="hover:text-white transition-colors">Help & Contact</Link>
          </div>
          <div className="flex items-center gap-4">
            <span className="flex items-center gap-1 text-white/70">
              <MapPin size={10} /> Ship to: UAE
            </span>
            <span className="text-white/30">|</span>
            {/* ── Notifications bell ── */}
            <div
              className="relative"
              onMouseEnter={() => { if (notifTimer.current) clearTimeout(notifTimer.current); setNotifOpen(true); }}
              onMouseLeave={() => { notifTimer.current = setTimeout(() => setNotifOpen(false), 200); }}
            >
              <button className="relative flex items-center text-white/80 hover:text-[#FFC220] transition-colors">
                <Bell size={14} />
                {isAuthenticated && (
                  <span className="absolute -top-1 -right-1.5 w-3.5 h-3.5 bg-red-500 rounded-full text-[9px] text-white font-bold flex items-center justify-center">3</span>
                )}
              </button>

              {notifOpen && (
                <div className="absolute right-0 top-full mt-2 w-80 bg-white rounded-lg shadow-2xl border border-gray-100 z-[80] text-gray-800">
                  <div className="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
                    <span className="font-bold text-sm text-gray-900">Notifications</span>
                    {isAuthenticated && <button className="text-[#0071CE] text-xs font-semibold hover:underline">Mark all read</button>}
                  </div>
                  {isAuthenticated ? (
                    <div>
                      {[
                        { icon: "🏷️", text: "Your bid on iPhone 15 Pro was outbid", time: "2 min ago", unread: true },
                        { icon: "✅", text: "Order #GC-4821 has been shipped", time: "1 hour ago", unread: true },
                        { icon: "💬", text: "New message from seller Ahmed K.", time: "3 hours ago", unread: true },
                        { icon: "⏰", text: "Auction ending in 30 min: MacBook Pro M3", time: "4 hours ago", unread: false },
                      ].map((n, i) => (
                        <div key={i} className={`flex items-start gap-3 px-4 py-2.5 hover:bg-blue-50 transition-colors cursor-pointer ${n.unread ? "bg-blue-50/50" : ""}`}>
                          <span className="text-base mt-0.5 shrink-0">{n.icon}</span>
                          <div className="flex-1 min-w-0">
                            <p className="text-xs text-gray-800 leading-snug">{n.text}</p>
                            <p className="text-[10px] text-gray-400 mt-0.5">{n.time}</p>
                          </div>
                          {n.unread && <span className="w-2 h-2 bg-[#0071CE] rounded-full mt-1.5 shrink-0" />}
                        </div>
                      ))}
                      <div className="px-4 py-3 border-t border-gray-100 text-center">
                        <Link href="/notifications" className="text-sm font-semibold text-[#0071CE] hover:underline">See all notifications →</Link>
                      </div>
                    </div>
                  ) : (
                    <div className="px-4 py-5 text-center">
                      <Bell size={28} className="text-[#0071CE] mx-auto mb-2" />
                      <p className="font-bold text-sm text-gray-900 mb-1">Stay in the loop</p>
                      <p className="text-xs text-gray-500 mb-4">Sign in to get alerts for bids, messages, and deals.</p>
                      <div className="flex gap-2">
                        <Link href="/login" className="flex-1 bg-[#0071CE] text-white text-sm font-semibold py-2 rounded-full text-center hover:bg-[#0058a3] transition-colors">Sign in</Link>
                        <Link href="/register" className="flex-1 border border-[#0071CE] text-[#0071CE] text-sm font-semibold py-2 rounded-full text-center hover:bg-blue-50 transition-colors">Register</Link>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* ── Watchlist dropdown ── */}
            <div
              className="relative"
              onMouseEnter={() => { if (watchTimer.current) clearTimeout(watchTimer.current); setWatchlistOpen(true); }}
              onMouseLeave={() => { watchTimer.current = setTimeout(() => setWatchlistOpen(false), 200); }}
            >
              <button className="flex items-center gap-0.5 text-white/80 hover:text-[#FFC220] transition-colors">
                <span className="font-semibold">Watchlist</span>
                <ChevronDown size={11} className={`transition-transform duration-200 ${watchlistOpen ? "rotate-180" : ""}`} />
              </button>

              {watchlistOpen && (
                <div className="absolute right-0 top-full mt-2 w-80 bg-white rounded-lg shadow-2xl border border-gray-100 z-[80] text-gray-800">
                  <div className="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
                    <span className="font-bold text-sm text-gray-900">Watchlist</span>
                    <Link href="/favorites" className="text-[#0071CE] text-xs font-semibold hover:underline">Go to Watchlist</Link>
                  </div>
                  {isAuthenticated ? (
                    <div>
                      <p className="text-xs text-gray-500 px-4 pt-3 pb-1 font-medium uppercase tracking-wide">Recently watched</p>
                      {[
                        { title: "iPhone 15 Pro Max 256GB Natural Titanium", price: "AED 4,299", ends: "Ends in 2h 14m", img: "https://images.unsplash.com/photo-1695048133142-1a20484429be?w=60&h=60&fit=crop" },
                        { title: "Nike Air Max 270 – Size 42 EU", price: "AED 380", ends: "Ends in 5h 40m", img: "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=60&h=60&fit=crop" },
                        { title: "Samsung 55\" QLED Smart TV 4K", price: "AED 2,199", ends: "Ends in 1d 3h", img: "https://images.unsplash.com/photo-1593784991095-a205069470b6?w=60&h=60&fit=crop" },
                      ].map((item, i) => (
                        <Link key={i} href="/favorites" className="flex items-center gap-3 px-4 py-2.5 hover:bg-blue-50 transition-colors group">
                          <img src={item.img} alt="" className="w-12 h-12 rounded object-cover shrink-0" />
                          <div className="flex-1 min-w-0">
                            <p className="text-xs font-medium text-gray-800 line-clamp-2 group-hover:text-[#0071CE]">{item.title}</p>
                            <p className="text-[11px] text-orange-500 font-semibold mt-0.5">{item.ends}</p>
                            <p className="text-xs font-bold text-gray-900">{item.price}</p>
                          </div>
                        </Link>
                      ))}
                      <div className="px-4 py-3 border-t border-gray-100">
                        <Link href="/favorites" className="block w-full text-center text-sm font-semibold text-[#0071CE] hover:underline">
                          See all watched items →
                        </Link>
                      </div>
                    </div>
                  ) : (
                    <div className="px-4 py-5 text-center">
                      <div className="w-12 h-12 bg-blue-50 rounded-full flex items-center justify-center mx-auto mb-3">
                        <Bell size={22} className="text-[#0071CE]" />
                      </div>
                      <p className="font-bold text-sm text-gray-900 mb-1">Don't miss out!</p>
                      <p className="text-xs text-gray-500 mb-4 leading-relaxed">Sign in to see items you've been watching for deals, price drops, and ending soon alerts.</p>
                      <div className="flex gap-2">
                        <Link href="/login" className="flex-1 bg-[#0071CE] text-white text-sm font-semibold py-2 rounded-full text-center hover:bg-[#0058a3] transition-colors">Sign in</Link>
                        <Link href="/register" className="flex-1 border border-[#0071CE] text-[#0071CE] text-sm font-semibold py-2 rounded-full text-center hover:bg-blue-50 transition-colors">Register</Link>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
            <Link href="/cart" className="flex items-center text-white/80 hover:text-[#FFC220] transition-colors">
              <ShoppingCart size={13} />
            </Link>
            {isAuthenticated && (
              <Link href="/wallet" className="text-[#FFC220] font-bold">
                {formatPrice(user?.balance ?? 0, "AED")}
              </Link>
            )}
          </div>
        </div>
      </div>

      {/* ── Main header ── */}
      <div className="max-w-7xl mx-auto px-4 py-3 flex items-center gap-3">
        <Link href="/" className="shrink-0">
          <span className="text-white font-black text-2xl tracking-tight">
            Geo<span className="text-[#FFC220]">Core</span>
          </span>
        </Link>

        <div className="flex-1 flex items-center gap-2 max-w-3xl">
          <form onSubmit={handleSearch} className="flex-1 flex">
            <div className="flex flex-1 bg-white rounded-full overflow-hidden shadow-inner">
              <div className="relative shrink-0 hidden sm:block">
                <button
                  type="button"
                  onClick={() => setCatOpen(!catOpen)}
                  className="flex items-center gap-1 px-3 h-full border-r border-gray-200 text-xs text-gray-600 hover:bg-gray-50 transition-colors whitespace-nowrap rounded-l-full"
                >
                  {selectedCat === "All Categories" ? "All" : selectedCat.slice(0, 10)}
                  <ChevronDown size={11} />
                </button>
                {catOpen && (
                  <div className="absolute top-full left-0 mt-1 bg-white border border-gray-200 rounded-xl shadow-xl z-50 w-52 py-1.5 overflow-hidden">
                    {ALL_CATS.map((c) => (
                      <button
                        key={c}
                        type="button"
                        onClick={() => { setSelectedCat(c); setCatOpen(false); }}
                        className={`w-full text-left px-4 py-2 text-sm hover:bg-blue-50 hover:text-[#0071CE] transition-colors ${selectedCat === c ? "text-[#0071CE] font-semibold bg-blue-50" : "text-gray-700"}`}
                      >
                        {c}
                      </button>
                    ))}
                  </div>
                )}
              </div>
              <input
                type="text"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Search GeoCore…"
                className="flex-1 px-4 py-2.5 text-sm text-gray-800 outline-none placeholder-gray-400 bg-transparent"
              />
              <button type="button" className="hidden sm:flex items-center px-3 text-gray-400 hover:text-[#0071CE] transition-colors">
                <Camera size={17} />
              </button>
            </div>
            <button
              type="submit"
              className="bg-[#FFC220] hover:bg-yellow-400 text-gray-900 px-6 py-2.5 rounded-full ml-2 font-bold text-sm transition-colors shrink-0 flex items-center gap-1.5"
            >
              <Search size={15} /> Search
            </button>
          </form>
          <Link
            href="/advanced-search"
            className="shrink-0 flex items-center gap-1 text-white/80 hover:text-white text-xs font-medium transition-colors whitespace-nowrap"
          >
            <SlidersHorizontal size={14} />
            <span className="hidden lg:inline">Advanced</span>
          </Link>
        </div>

        <div className="flex items-center gap-2 shrink-0 text-white">
          {isAuthenticated ? (
            <>
              <Link href="/profile" className="flex flex-col items-center text-xs hover:text-[#FFC220] transition-colors gap-0.5">
                <div className="w-7 h-7 rounded-full bg-[#FFC220] text-gray-900 font-bold text-xs flex items-center justify-center">
                  {user?.name?.[0]?.toUpperCase() || <User size={13} />}
                </div>
                <span className="hidden lg:block">{user?.name?.split(" ")[0]}</span>
              </Link>
              <Link href="/dashboard" className="hidden md:flex flex-col items-center text-xs hover:text-[#FFC220] transition-colors gap-0.5">
                <LayoutDashboard size={22} />
                <span className="hidden lg:block">Dashboard</span>
              </Link>
              <Link href="/my-store" className="hidden md:flex flex-col items-center text-xs hover:text-[#FFC220] transition-colors gap-0.5">
                <Store size={22} />
                <span className="hidden lg:block">Store</span>
              </Link>
              <button onClick={logout} className="hidden md:flex flex-col items-center text-xs hover:text-[#FFC220] transition-colors gap-0.5">
                <LogOut size={22} />
                <span className="hidden lg:block">Sign Out</span>
              </button>
            </>
          ) : null}
          <Link
            href="/sell"
            className="hidden sm:flex items-center gap-1 bg-[#FFC220] hover:bg-yellow-400 text-gray-900 text-xs font-bold px-3.5 py-2 rounded-full transition-colors"
          >
            + Post Listing
          </Link>
        </div>
      </div>

      {/* ── Category nav bar — light blue outer (Walmart: #E9F1FE), white boxes inner ── */}
      <div className="bg-[#EAF2FF] border-b border-[#cde0ff]">
        <div className="max-w-7xl mx-auto px-4 flex items-center gap-1.5 py-1.5 overflow-x-auto scrollbar-none">
          {/* Departments – bordered white box */}
          <button
            onMouseEnter={() => openMenu("more")}
            onMouseLeave={scheduleClose}
            className={`whitespace-nowrap shrink-0 h-7 px-3 text-[13px] font-bold flex items-center gap-1.5 rounded-full border bg-white transition-colors ${moreMenuOpen ? "border-[#0071CE] text-[#0071CE]" : "border-gray-400 text-gray-800 hover:border-[#0071CE] hover:text-[#0071CE]"}`}
          >
            Departments
            <ChevronDown size={12} className={`transition-transform duration-200 ${moreMenuOpen ? "rotate-180" : ""}`} />
          </button>

          {CATEGORY_LINKS.map((item) => (
            <button
              key={item.href}
              onMouseEnter={() => openMenu(item.key)}
              onMouseLeave={scheduleClose}
              onClick={() => navigate(item.href)}
              className={`whitespace-nowrap px-3 h-7 text-[13px] transition-colors flex items-center gap-1 rounded-full bg-white border ${activeMenu === item.key && item.key ? "border-[#0071CE] text-[#0071CE]" : "border-transparent text-gray-800 hover:border-gray-300 hover:text-[#0071CE]"}`}
            >
              {item.label}
            </button>
          ))}

          {/* More button – bordered white box */}
          <button
            onMouseEnter={() => openMenu("more")}
            onMouseLeave={scheduleClose}
            className={`whitespace-nowrap ml-auto shrink-0 h-7 px-3 text-[13px] flex items-center gap-1 rounded-full border bg-white transition-colors ${moreMenuOpen ? "border-[#0071CE] text-[#0071CE]" : "border-gray-400 text-gray-800 hover:border-[#0071CE] hover:text-[#0071CE]"}`}
          >
            More
            <ChevronDown size={12} className={`transition-transform duration-200 ${moreMenuOpen ? "rotate-180" : ""}`} />
          </button>
        </div>
      </div>

      {/* ── More dropdown panel ── */}
      {moreMenuOpen && (
        <div
          className="absolute right-0 z-[60]"
          style={{ top: "100%" }}
          onMouseEnter={() => openMenu("more")}
          onMouseLeave={scheduleClose}
        >
          <div className="max-w-7xl mx-auto px-4 pb-4 flex justify-end">
            <div className="bg-white rounded-b-xl shadow-2xl border border-t-0 border-gray-200 overflow-hidden w-56">
              <ul className="py-2">
                {CATEGORY_LINKS.filter((c) => c.key).map((item) => (
                  <li key={item.href}>
                    <button
                      onClick={() => { navigate(item.href); setActiveMenu(null); }}
                      className="w-full text-left px-5 py-2.5 text-sm text-gray-700 hover:text-[#0071CE] hover:bg-blue-50 transition-colors flex items-center gap-2"
                    >
                      <ChevronRight size={12} className="text-gray-300 shrink-0" />
                      {item.label}
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      )}

      {/* ── Mega dropdown panel — rendered at header level, NOT inside the nav overflow container ── */}
      {activeMegaData && (
        <div
          className="absolute left-0 right-0 z-[60]"
          style={{ top: "100%" }}
          onMouseEnter={() => openMenu(activeMenu)}
          onMouseLeave={scheduleClose}
        >
          <div className="max-w-7xl mx-auto px-4 pb-4">
            <div className="bg-white rounded-b-2xl shadow-2xl border border-t-0 border-gray-200 overflow-hidden flex">

              {/* Column 1: Most popular */}
              <div className="w-56 border-r border-gray-100 p-5 shrink-0 bg-gray-50">
                <p className="text-[10px] font-black text-gray-400 uppercase tracking-widest mb-3">
                  Most popular
                </p>
                <ul className="space-y-0.5">
                  {activeMegaData.popular.map((item) => (
                    <li key={item.label}>
                      <Link
                        href={item.href}
                        onClick={() => setActiveMenu(null)}
                        className="flex items-center gap-2 text-sm text-gray-700 hover:text-[#0071CE] hover:bg-blue-50 px-2 py-1.5 rounded-lg transition-colors group"
                      >
                        <ChevronRight size={12} className="text-gray-300 group-hover:text-[#0071CE] shrink-0" />
                        {item.label}
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>

              {/* Column 2: More categories */}
              <div className="w-48 border-r border-gray-100 p-5 shrink-0">
                <p className="text-[10px] font-black text-gray-400 uppercase tracking-widest mb-3">
                  More categories
                </p>
                <ul className="space-y-0.5">
                  {activeMegaData.more.map((item) => (
                    <li key={item.label}>
                      <Link
                        href={item.href}
                        onClick={() => setActiveMenu(null)}
                        className="flex items-center gap-2 text-sm text-gray-700 hover:text-[#0071CE] hover:bg-blue-50 px-2 py-1.5 rounded-lg transition-colors group"
                      >
                        <ChevronRight size={12} className="text-gray-300 group-hover:text-[#0071CE] shrink-0" />
                        {item.label}
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>

              {/* Column 3: Promo card with image */}
              <div className="flex-1 p-4">
                <Link
                  href={activeMegaData.promo.href}
                  onClick={() => setActiveMenu(null)}
                  className="block h-full rounded-xl overflow-hidden relative group cursor-pointer min-h-[200px]"
                >
                  <img
                    src={`https://picsum.photos/seed/${activeMegaData.promo.imgSeed}/600/300`}
                    alt={activeMegaData.promo.title}
                    className="absolute inset-0 w-full h-full object-cover group-hover:scale-[1.03] transition-transform duration-500"
                  />
                  {/* Dark gradient overlay */}
                  <div className="absolute inset-0 bg-gradient-to-r from-black/70 via-black/40 to-transparent" />
                  {/* Text */}
                  <div className="relative z-10 p-6 h-full flex flex-col justify-between">
                    <div>
                      <h3 className="text-2xl font-black text-white leading-tight">
                        {activeMegaData.promo.title}
                      </h3>
                      <p className="text-sm text-white/80 mt-1.5 leading-relaxed max-w-xs">
                        {activeMegaData.promo.subtitle}
                      </p>
                    </div>
                    <span
                      className="mt-4 inline-flex items-center gap-1.5 text-sm font-bold px-5 py-2 rounded-full text-white w-fit shadow-lg hover:opacity-90 transition-opacity"
                      style={{ backgroundColor: activeMegaData.promo.accent }}
                    >
                      {activeMegaData.promo.cta} <ChevronRight size={14} />
                    </span>
                  </div>
                </Link>
              </div>
            </div>
          </div>
        </div>
      )}
    </header>
  );
}
