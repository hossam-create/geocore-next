import { useState } from "react";
import { Link } from "wouter";
import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";
import { Tag, Star, ChevronRight, Flame, BadgePercent, Store } from "lucide-react";

const BRANDS = [
  { name: "Apple", slug: "apple", color: "#1d1d1f", bg: "#f5f5f7", deals: 142, discount: "Up to 30% off", logo: "🍎" },
  { name: "Samsung", slug: "samsung", color: "#1428A0", bg: "#e8ecf8", deals: 218, discount: "Up to 40% off", logo: "📱" },
  { name: "Nike", slug: "nike", color: "#111111", bg: "#f5f5f5", deals: 305, discount: "Up to 50% off", logo: "👟" },
  { name: "Sony", slug: "sony", color: "#003087", bg: "#e6ebf5", deals: 97, discount: "Up to 35% off", logo: "🎧" },
  { name: "IKEA", slug: "ikea", color: "#0058A3", bg: "#e6f0fa", deals: 184, discount: "Up to 25% off", logo: "🪑" },
  { name: "Adidas", slug: "adidas", color: "#000000", bg: "#f0f0f0", deals: 261, discount: "Up to 45% off", logo: "🏃" },
  { name: "LG", slug: "lg", color: "#a50034", bg: "#fceef2", deals: 113, discount: "Up to 38% off", logo: "📺" },
  { name: "Dyson", slug: "dyson", color: "#C0392B", bg: "#fdf0ee", deals: 48, discount: "Up to 20% off", logo: "🌀" },
  { name: "Philips", slug: "philips", color: "#0B5ED7", bg: "#e8f0fc", deals: 89, discount: "Up to 28% off", logo: "💡" },
  { name: "HP", slug: "hp", color: "#0096D6", bg: "#e5f4fc", deals: 76, discount: "Up to 32% off", logo: "💻" },
  { name: "Xiaomi", slug: "xiaomi", color: "#FF6900", bg: "#fff3eb", deals: 197, discount: "Up to 42% off", logo: "📡" },
  { name: "Bosch", slug: "bosch", color: "#D40000", bg: "#fdeaea", deals: 134, discount: "Up to 22% off", logo: "🔧" },
];

const OUTLET_DEALS = [
  {
    id: 1, brand: "Apple", title: "MacBook Air M2 – Space Gray 256GB",
    original: "AED 4,999", price: "AED 3,499", discount: "30%",
    img: "https://images.unsplash.com/photo-1611186871348-b1ce696e52c9?w=300&h=220&fit=crop",
    badge: "Official Refurbished", rating: 4.8, reviews: 312,
  },
  {
    id: 2, brand: "Samsung", title: "Galaxy S24 Ultra 256GB Titanium Black",
    original: "AED 5,499", price: "AED 3,999", discount: "27%",
    img: "https://images.unsplash.com/photo-1610945415295-d9bbf067e59c?w=300&h=220&fit=crop",
    badge: "Brand New Sealed", rating: 4.9, reviews: 541,
  },
  {
    id: 3, brand: "Nike", title: "Air Max 270 – Men's Size 42 White/Black",
    original: "AED 699", price: "AED 349", discount: "50%",
    img: "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=300&h=220&fit=crop",
    badge: "Last Season Stock", rating: 4.7, reviews: 208,
  },
  {
    id: 4, brand: "Sony", title: "WH-1000XM5 Wireless Noise Cancelling Headphones",
    original: "AED 1,599", price: "AED 999", discount: "38%",
    img: "https://images.unsplash.com/photo-1546435770-a3e426bf472b?w=300&h=220&fit=crop",
    badge: "Box Opened", rating: 4.9, reviews: 189,
  },
  {
    id: 5, brand: "IKEA", title: "MARKUS Ergonomic Office Chair – Vissle Dark Gray",
    original: "AED 1,299", price: "AED 899", discount: "31%",
    img: "https://images.unsplash.com/photo-1592078615290-033ee584e267?w=300&h=220&fit=crop",
    badge: "Display Model", rating: 4.6, reviews: 97,
  },
  {
    id: 6, brand: "Dyson", title: "V15 Detect Absolute Cordless Vacuum Cleaner",
    original: "AED 3,199", price: "AED 2,199", discount: "31%",
    img: "https://images.unsplash.com/photo-1558618666-fcd25c85cd64?w=300&h=220&fit=crop",
    badge: "Certified Pre-owned", rating: 4.8, reviews: 143,
  },
  {
    id: 7, brand: "Adidas", title: "Ultraboost 23 Running Shoes – Core Black",
    original: "AED 899", price: "AED 499", discount: "44%",
    img: "https://images.unsplash.com/photo-1608231387042-66d1773070a5?w=300&h=220&fit=crop",
    badge: "Last Season Stock", rating: 4.7, reviews: 276,
  },
  {
    id: 8, brand: "LG", title: "OLED C3 55\" 4K Smart TV – 2023 Model",
    original: "AED 5,999", price: "AED 3,799", discount: "37%",
    img: "https://images.unsplash.com/photo-1593784991095-a205069470b6?w=300&h=220&fit=crop",
    badge: "Open Box", rating: 4.8, reviews: 88,
  },
];

const FILTERS = ["All", "Electronics", "Fashion", "Home & Living", "Sports", "Appliances"];

export default function BrandOutletPage() {
  const [activeFilter, setActiveFilter] = useState("All");
  const [activeBrand, setActiveBrand] = useState<string | null>(null);

  const { data: realStores } = useQuery({
    queryKey: ["stores-brand-outlet"],
    queryFn: () => api.get("/stores?limit=12").then((r) => r.data.data ?? []),
    retry: false,
  });

  const filtered = OUTLET_DEALS.filter((d) => {
    if (activeBrand && d.brand !== activeBrand) return false;
    return true;
  });

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Hero */}
      <div className="bg-gradient-to-r from-[#0071CE] to-[#004F9A] text-white py-10 px-4">
        <div className="max-w-7xl mx-auto">
          <div className="flex items-center gap-2 mb-2">
            <BadgePercent size={20} className="text-[#FFC220]" />
            <span className="text-[#FFC220] text-sm font-semibold tracking-wide uppercase">Official Brand Stores</span>
          </div>
          <h1 className="text-3xl font-bold mb-1">Brand Outlet</h1>
          <p className="text-white/70 text-sm max-w-xl">
            Shop directly from official brand stores — authentic products, exclusive deals, and guaranteed savings on top brands.
          </p>
          <div className="flex items-center gap-3 mt-4 text-xs text-white/60">
            <span className="flex items-center gap-1"><Tag size={12} /> Authentic & Guaranteed</span>
            <span>·</span>
            <span className="flex items-center gap-1"><Flame size={12} /> Up to 50% off</span>
            <span>·</span>
            <span>Free returns on all brand items</span>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 py-8">
        {/* Brand grid */}
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-gray-900">Shop by Brand</h2>
          <button className="text-[#0071CE] text-sm hover:underline flex items-center gap-1">
            See all brands <ChevronRight size={14} />
          </button>
        </div>

        <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-6 gap-3 mb-8">
          {BRANDS.map((brand) => (
            <button
              key={brand.slug}
              onClick={() => setActiveBrand(activeBrand === brand.name ? null : brand.name)}
              className={`rounded-xl p-3 text-center transition-all border-2 group ${
                activeBrand === brand.name
                  ? "border-[#0071CE] bg-blue-50 shadow-md"
                  : "border-transparent bg-white hover:border-gray-200 shadow-sm hover:shadow-md"
              }`}
            >
              <div className="text-2xl mb-1">{brand.logo}</div>
              <p className="text-xs font-semibold text-gray-800">{brand.name}</p>
              <p className="text-[10px] text-[#0071CE] font-medium mt-0.5">{brand.discount}</p>
              <p className="text-[10px] text-gray-400">{brand.deals} deals</p>
            </button>
          ))}
        </div>

        {/* Real storefronts from marketplace */}
        {realStores && realStores.length > 0 && (
          <div className="mb-8">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-bold text-gray-900 flex items-center gap-2">
                <Store size={18} className="text-[#0071CE]" /> Verified Seller Stores
              </h2>
              <Link href="/stores" className="text-[#0071CE] text-sm hover:underline flex items-center gap-1">
                All stores <ChevronRight size={14} />
              </Link>
            </div>
            <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3">
              {realStores.slice(0, 8).map((store: any) => (
                <Link key={store.id} href={`/stores/${store.slug}`}>
                  <div className="bg-white rounded-xl shadow-sm hover:shadow-md transition-all overflow-hidden group cursor-pointer border border-gray-100">
                    <div className="h-16 bg-gradient-to-br from-[#0071CE] to-[#003f75]" />
                    <div className="px-4 pb-4">
                      <div className="w-10 h-10 rounded-lg border-2 border-white shadow bg-[#FFC220] flex items-center justify-center text-sm font-extrabold text-gray-900 -mt-5 mb-2 overflow-hidden">
                        {store.logo_url ? (
                          <img src={store.logo_url} alt={store.name} className="w-full h-full object-cover" />
                        ) : (
                          store.name?.[0]?.toUpperCase()
                        )}
                      </div>
                      <p className="text-xs font-bold text-gray-900 group-hover:text-[#0071CE] transition-colors line-clamp-1">{store.name}</p>
                      <p className="text-[10px] text-gray-400 mt-0.5">{store.views?.toLocaleString() ?? 0} views</p>
                    </div>
                  </div>
                </Link>
              ))}
            </div>
          </div>
        )}

        {/* Category filter */}
        <div className="flex items-center gap-2 mb-6 overflow-x-auto scrollbar-none pb-1">
          {FILTERS.map((f) => (
            <button
              key={f}
              onClick={() => setActiveFilter(f)}
              className={`whitespace-nowrap px-4 py-1.5 rounded-full text-sm border transition-colors ${
                activeFilter === f
                  ? "bg-[#0071CE] text-white border-[#0071CE]"
                  : "bg-white text-gray-700 border-gray-300 hover:border-[#0071CE] hover:text-[#0071CE]"
              }`}
            >
              {f}
            </button>
          ))}
          {activeBrand && (
            <button
              onClick={() => setActiveBrand(null)}
              className="whitespace-nowrap px-4 py-1.5 rounded-full text-sm border bg-blue-50 text-[#0071CE] border-[#0071CE] flex items-center gap-1"
            >
              {activeBrand} ✕
            </button>
          )}
        </div>

        {/* Deals grid */}
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-gray-900">
            {activeBrand ? `${activeBrand} Outlet Deals` : "Top Outlet Deals"}
          </h2>
          <span className="text-sm text-gray-500">{filtered.length} items</span>
        </div>

        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
          {filtered.map((deal) => (
            <Link key={deal.id} href={`/listings/${deal.id}`}>
              <div className="bg-white rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-all group cursor-pointer">
                <div className="relative overflow-hidden">
                  <img
                    src={deal.img}
                    alt={deal.title}
                    className="w-full h-44 object-cover group-hover:scale-105 transition-transform duration-300"
                  />
                  <div className="absolute top-2 left-2 bg-red-500 text-white text-[10px] font-bold px-2 py-0.5 rounded-full">
                    -{deal.discount}
                  </div>
                  <div className="absolute top-2 right-2 bg-white/90 text-gray-700 text-[10px] font-medium px-2 py-0.5 rounded-full">
                    {deal.badge}
                  </div>
                </div>
                <div className="p-3">
                  <p className="text-[11px] text-[#0071CE] font-semibold mb-0.5">{deal.brand}</p>
                  <p className="text-sm text-gray-800 line-clamp-2 leading-snug mb-2">{deal.title}</p>
                  <div className="flex items-baseline gap-2 mb-1">
                    <span className="text-base font-bold text-gray-900">{deal.price}</span>
                    <span className="text-xs text-gray-400 line-through">{deal.original}</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <Star size={11} className="text-[#FFC220] fill-[#FFC220]" />
                    <span className="text-xs text-gray-600">{deal.rating}</span>
                    <span className="text-xs text-gray-400">({deal.reviews})</span>
                  </div>
                </div>
                <div className="px-3 pb-3">
                  <button className="w-full bg-[#FFC220] hover:bg-yellow-400 text-gray-900 text-sm font-semibold py-2 rounded-lg transition-colors">
                    Add to Cart
                  </button>
                </div>
              </div>
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
}
