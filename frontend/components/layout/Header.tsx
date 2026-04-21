'use client'
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState, useRef } from "react";
import {
  Search, ChevronDown, Camera, Bell,
  SlidersHorizontal, ChevronRight, Plane, Package,
  Heart, Menu, X, Radio, HelpCircle, Tag,
} from "lucide-react";
import { PERMISSIONS, hasAnyPermission, isInternalRole, isSeller, isTraveler } from "@/lib/permissions";
import { useAuthStore } from "@/store/auth";
import { CartIcon } from "@/components/cart/CartIcon";
import { AccountDropdown } from "@/components/header/AccountDropdown";
import { CountryDetector } from "@/components/header/CountryDetector";
import { LanguageSwitcher } from "@/components/header/LanguageSwitcher";
import { useTranslations } from "next-intl";

/* ─── eBay Category Tree Data ─── */

interface L1Category {
  slug: string;
  label: string;
  color: string;
  icon: string;
  children: { slug: string; label: string; href: string }[];
}

// Full eBay L1 category list (starting with Antiques) — matches parse-ebay-categories-file.ts
const CATEGORY_TREE: L1Category[] = [
  { slug: "antiques", label: "Antiques", color: "#735200", icon: "🏺",
    children: [] },
  { slug: "art", label: "Art", color: "#735200", icon: "🎨",
    children: [] },
  { slug: "baby", label: "Baby", color: "#E53238", icon: "🍼",
    children: [] },
  { slug: "boats", label: "Boats", color: "#0064D2", icon: "⛵",
    children: [] },
  { slug: "books", label: "Books", color: "#333333", icon: "📚",
    children: [] },
  { slug: "business-industrial", label: "Business & Industrial", color: "#333333", icon: "🏭",
    children: [] },
  { slug: "cameras-photo", label: "Cameras & Photo", color: "#E53238", icon: "📷",
    children: [] },
  { slug: "cell-phones-accessories", label: "Cell Phones & Accessories", color: "#E53238", icon: "📱",
    children: [] },
  { slug: "clothing-shoes-accessories", label: "Clothing, Shoes & Accessories", color: "#F5AF02", icon: "👗",
    children: [] },
  { slug: "coins-paper-money", label: "Coins & Paper Money", color: "#735200", icon: "🪙",
    children: [] },
  { slug: "collectibles", label: "Collectibles", color: "#735200", icon: "🏺",
    children: [] },
  { slug: "computers-tablets-networking", label: "Computers/Tablets & Networking", color: "#E53238", icon: "💻",
    children: [] },
  { slug: "consumer-electronics", label: "Consumer Electronics", color: "#E53238", icon: "🔌",
    children: [] },
  { slug: "crafts", label: "Crafts", color: "#86B817", icon: "✂️",
    children: [] },
  { slug: "dolls-bears", label: "Dolls & Bears", color: "#E53238", icon: "🧸",
    children: [] },
  { slug: "dvds-movies", label: "DVDs & Movies", color: "#333333", icon: "🎬",
    children: [] },
  { slug: "ebay-motors", label: "eBay Motors", color: "#86B817", icon: "🚗",
    children: [] },
  { slug: "entertainment-memorabilia", label: "Entertainment Memorabilia", color: "#735200", icon: "🎭",
    children: [] },
  { slug: "gift-cards", label: "Gift Cards", color: "#F5AF02", icon: "🎁",
    children: [] },
  { slug: "health-beauty", label: "Health & Beauty", color: "#F5AF02", icon: "💄",
    children: [] },
  { slug: "home-garden", label: "Home & Garden", color: "#86B817", icon: "🏠",
    children: [] },
  { slug: "jewelry-watches", label: "Jewelry & Watches", color: "#b45309", icon: "💎",
    children: [] },
  { slug: "music", label: "Music", color: "#333333", icon: "🎵",
    children: [] },
  { slug: "musical-instruments", label: "Musical Instruments", color: "#333333", icon: "🎸",
    children: [] },
  { slug: "pet-supplies", label: "Pet Supplies", color: "#86B817", icon: "🐾",
    children: [] },
  { slug: "pottery-glass", label: "Pottery & Glass", color: "#735200", icon: "🏺",
    children: [] },
  { slug: "real-estate", label: "Real Estate", color: "#0064D2", icon: "🏢",
    children: [] },
  { slug: "specialty-services", label: "Specialty Services", color: "#735200", icon: "🔧",
    children: [] },
  { slug: "sporting-goods", label: "Sporting Goods", color: "#86B817", icon: "⚽",
    children: [] },
  { slug: "sports-mem-cards-fan-shop", label: "Sports Mem, Cards & Fan Shop", color: "#86B817", icon: "🏆",
    children: [] },
  { slug: "stamps", label: "Stamps", color: "#735200", icon: "📮",
    children: [] },
  { slug: "tickets-experiences", label: "Tickets & Experiences", color: "#0064D2", icon: "🎫",
    children: [] },
  { slug: "toys-hobbies", label: "Toys & Hobbies", color: "#E53238", icon: "🧸",
    children: [] },
  { slug: "travel", label: "Travel", color: "#0064D2", icon: "✈️",
    children: [] },
  { slug: "video-games-consoles", label: "Video Games & Consoles", color: "#7c3aed", icon: "🎮",
    children: [] },
  { slug: "everything-else", label: "Everything Else", color: "#333333", icon: "📦",
    children: [] },
];

// 12 main categories with subcategories — for the Categories mega dropdown (starts with Electronics)
const DEPARTMENTS_CATS: L1Category[] = [
  { slug: "electronics", label: "Electronics", color: "#E53238", icon: "📱",
    children: [
      { slug: "cell-phones-accessories", label: "Cell Phones & Accessories", href: "/category/cell-phones-accessories" },
      { slug: "computers-tablets-networking", label: "Computers/Tablets & Networking", href: "/category/computers-tablets-networking" },
      { slug: "cameras-photo", label: "Cameras & Photo", href: "/category/cameras-photo" },
      { slug: "consumer-electronics", label: "Consumer Electronics", href: "/category/consumer-electronics" },
      { slug: "video-games-consoles", label: "Video Games & Consoles", href: "/category/video-games-consoles" },
      { slug: "smart-home", label: "Smart Home", href: "/category/smart-home" },
      { slug: "wearables", label: "Wearables", href: "/category/wearables" },
    ] },
  { slug: "ebay-motors", label: "Motors", color: "#86B817", icon: "🚗",
    children: [
      { slug: "cars-trucks", label: "Cars & Trucks", href: "/category/cars-trucks" },
      { slug: "motorcycles", label: "Motorcycles & ATVs", href: "/category/motorcycles" },
      { slug: "boats", label: "Boats", href: "/category/boats" },
      { slug: "auto-parts", label: "Auto Parts & Vehicles", href: "/category/auto-parts" },
      { slug: "car-electronics", label: "Car Electronics", href: "/category/car-electronics" },
    ] },
  { slug: "clothing-shoes-accessories", label: "Fashion", color: "#F5AF02", icon: "👗",
    children: [
      { slug: "mens-clothing", label: "Men's Clothing", href: "/category/mens-clothing" },
      { slug: "womens-clothing", label: "Women's Clothing", href: "/category/womens-clothing" },
      { slug: "kids-clothing", label: "Kids' Clothing", href: "/category/kids-clothing" },
      { slug: "shoes", label: "Shoes", href: "/category/shoes" },
      { slug: "bags-handbags", label: "Bags & Handbags", href: "/category/bags-handbags" },
      { slug: "jewelry-watches", label: "Jewelry & Watches", href: "/category/jewelry-watches" },
    ] },
  { slug: "home-garden", label: "Home & Garden", color: "#86B817", icon: "🏠",
    children: [
      { slug: "furniture", label: "Furniture", href: "/category/furniture" },
      { slug: "kitchen", label: "Kitchen & Dining", href: "/category/kitchen" },
      { slug: "bedding", label: "Bedding & Linens", href: "/category/bedding" },
      { slug: "appliances", label: "Major Appliances", href: "/category/appliances" },
      { slug: "garden", label: "Garden & Patio", href: "/category/garden" },
      { slug: "home-decor", label: "Home Décor", href: "/category/home-decor" },
      { slug: "tools", label: "Tools & Workshop", href: "/category/tools" },
    ] },
  { slug: "sporting-goods", label: "Sporting Goods", color: "#86B817", icon: "⚽",
    children: [
      { slug: "exercise", label: "Exercise & Fitness", href: "/category/exercise" },
      { slug: "team-sports", label: "Team Sports", href: "/category/team-sports" },
      { slug: "water-sports", label: "Water Sports", href: "/category/water-sports" },
      { slug: "cycling", label: "Cycling", href: "/category/cycling" },
      { slug: "camping", label: "Camping & Hiking", href: "/category/camping" },
    ] },
  { slug: "toys-hobbies", label: "Toys & Hobbies", color: "#E53238", icon: "🧸",
    children: [
      { slug: "toys", label: "Toys", href: "/category/toys" },
      { slug: "hobbies", label: "Hobbies", href: "/category/hobbies" },
      { slug: "educational", label: "Educational Toys", href: "/category/educational" },
      { slug: "outdoor-play", label: "Outdoor Toys", href: "/category/outdoor-play" },
      { slug: "dolls-bears", label: "Dolls & Bears", href: "/category/dolls-bears" },
    ] },
  { slug: "business-industrial", label: "Business & Industrial", color: "#333333", icon: "🏭",
    children: [
      { slug: "office", label: "Office Supplies", href: "/category/office" },
      { slug: "industrial", label: "Industrial Equipment", href: "/category/industrial" },
      { slug: "wholesale", label: "Wholesale Lots", href: "/category/wholesale" },
      { slug: "printing", label: "Printing & Signage", href: "/category/printing" },
    ] },
  { slug: "health-beauty", label: "Health & Beauty", color: "#F5AF02", icon: "💄",
    children: [
      { slug: "skincare", label: "Skin Care", href: "/category/skincare" },
      { slug: "haircare", label: "Hair Care", href: "/category/haircare" },
      { slug: "makeup", label: "Makeup", href: "/category/makeup" },
      { slug: "vitamins", label: "Vitamins & Supplements", href: "/category/vitamins" },
      { slug: "medical", label: "Medical & Mobility", href: "/category/medical" },
    ] },
  { slug: "collectibles", label: "Collectibles & Art", color: "#735200", icon: "🏺",
    children: [
      { slug: "antiques", label: "Antiques", href: "/category/antiques" },
      { slug: "art", label: "Art", href: "/category/art" },
      { slug: "coins-paper-money", label: "Coins & Paper Money", href: "/category/coins-paper-money" },
      { slug: "stamps", label: "Stamps", href: "/category/stamps" },
      { slug: "pottery-glass", label: "Pottery & Glass", href: "/category/pottery-glass" },
    ] },
  { slug: "real-estate", label: "Real Estate", color: "#0064D2", icon: "🏢",
    children: [
      { slug: "apartments-sale", label: "Apartments for Sale", href: "/category/apartments-sale" },
      { slug: "apartments-rent", label: "Apartments for Rent", href: "/category/apartments-rent" },
      { slug: "villas-sale", label: "Villas for Sale", href: "/category/villas-sale" },
      { slug: "commercial", label: "Commercial Property", href: "/category/commercial" },
      { slug: "land", label: "Land & Plots", href: "/category/land" },
    ] },
  { slug: "pet-supplies", label: "Pet Supplies", color: "#86B817", icon: "🐾",
    children: [
      { slug: "dog", label: "Dog Supplies", href: "/category/dog" },
      { slug: "cat", label: "Cat Supplies", href: "/category/cat" },
      { slug: "fish", label: "Fish & Aquatic", href: "/category/fish" },
      { slug: "birds", label: "Bird Supplies", href: "/category/birds" },
    ] },
  { slug: "books", label: "Books, Movies & Music", color: "#333333", icon: "📚",
    children: [
      { slug: "books", label: "Books", href: "/category/books" },
      { slug: "dvds-movies", label: "DVDs & Movies", href: "/category/dvds-movies" },
      { slug: "music", label: "Music", href: "/category/music" },
      { slug: "musical-instruments", label: "Musical Instruments", href: "/category/musical-instruments" },
    ] },
];

/* ─── Mega Menu Data ─── */
interface MegaMenuData {
  popular: { label: string; href: string }[];
  more: { label: string; href: string }[];
  promo: { title: string; subtitle: string; cta: string; imgSeed: string; href: string; accent: string };
}

const MEGA_MENUS: Record<string, MegaMenuData> = {
  electronics: {
    popular: [
      { label: "Cell Phones & Accessories", href: "/category/phones-accessories" },
      { label: "Computers, Tablets & More", href: "/category/computers-tablets" },
      { label: "TV, Audio & Video", href: "/category/tv-audio-video" },
      { label: "Cameras & Photo", href: "/category/cameras-photo" },
      { label: "Video Games & Consoles", href: "/category/video-games-consoles" },
      { label: "Smart Home", href: "/category/smart-home" },
      { label: "Wearables", href: "/category/wearables" },
    ],
    more: [
      { label: "Smartphones", href: "/category/smartphones" },
      { label: "Phone Cases & Covers", href: "/category/phone-cases" },
      { label: "Chargers & Cables", href: "/category/chargers" },
      { label: "Headphones", href: "/category/headphones" },
      { label: "Flash Deals", href: "/listings?category=electronics&sort=price_asc" },
    ],
    promo: { title: "Electronics", subtitle: "Smart devices, always with you.", cta: "Explore now", imgSeed: "smartphone-laptop-tech", href: "/category/electronics", accent: "#E53238" },
  },
  motors: {
    popular: [
      { label: "Cars & Trucks", href: "/category/cars-trucks" },
      { label: "Motorcycles & ATVs", href: "/category/motorcycles" },
      { label: "Boats", href: "/category/boats" },
      { label: "Auto Parts & Vehicles", href: "/category/auto-parts" },
      { label: "Car Electronics", href: "/category/car-electronics" },
      { label: "Car Care & Detailing", href: "/category/car-care" },
      { label: "Sedans", href: "/category/sedan" },
    ],
    more: [
      { label: "SUVs", href: "/category/suv" },
      { label: "Pickups", href: "/category/pickup" },
      { label: "Vans", href: "/category/van" },
      { label: "Auctions Only", href: "/auctions" },
      { label: "New Arrivals", href: "/listings?category=motors&sort=newest" },
    ],
    promo: { title: "Motors", subtitle: "Find your next ride.", cta: "Browse motors", imgSeed: "luxury-car-road-gcc", href: "/category/motors", accent: "#86B817" },
  },
  fashion: {
    popular: [
      { label: "Men's Clothing", href: "/category/mens-clothing" },
      { label: "Women's Clothing", href: "/category/womens-clothing" },
      { label: "Kids' Clothing", href: "/category/kids-clothing" },
      { label: "Shoes", href: "/category/shoes" },
      { label: "Bags & Handbags", href: "/category/bags-handbags" },
      { label: "Jewelry & Watches", href: "/category/jewelry" },
      { label: "Sportswear", href: "/category/sportswear" },
    ],
    more: [
      { label: "New Arrivals", href: "/listings?category=fashion&sort=newest" },
      { label: "On Sale", href: "/listings?category=fashion&sort=price_asc" },
      { label: "Luxury", href: "/listings?category=fashion&q=luxury" },
    ],
    promo: { title: "Fashion", subtitle: "Top brands. Unbeatable prices.", cta: "Shop fashion", imgSeed: "fashion-dress-luxury-gcc", href: "/category/fashion", accent: "#F5AF02" },
  },
  "home-garden": {
    popular: [
      { label: "Furniture", href: "/category/furniture" },
      { label: "Kitchen & Dining", href: "/category/kitchen" },
      { label: "Bedding & Linens", href: "/category/bedding" },
      { label: "Major Appliances", href: "/category/appliances" },
      { label: "Garden & Patio", href: "/category/garden" },
      { label: "Home Décor", href: "/category/home-decor" },
      { label: "Tools & Workshop", href: "/category/tools" },
    ],
    more: [
      { label: "Flash Deals", href: "/listings?category=home-garden&sort=price_asc" },
      { label: "New Arrivals", href: "/listings?category=home-garden&sort=newest" },
    ],
    promo: { title: "Home & Garden", subtitle: "Style your space for less.", cta: "Shop home", imgSeed: "living-room-sofa-modern", href: "/category/home-garden", accent: "#86B817" },
  },
  "real-estate": {
    popular: [
      { label: "Apartments for Sale", href: "/category/apartments-sale" },
      { label: "Apartments for Rent", href: "/category/apartments-rent" },
      { label: "Villas for Sale", href: "/category/villas-sale" },
      { label: "Commercial Property", href: "/category/commercial" },
      { label: "Land & Plots", href: "/category/land" },
      { label: "Rooms for Rent", href: "/category/rooms-rent" },
    ],
    more: [
      { label: "New Projects", href: "/listings?category=real-estate&sort=newest" },
    ],
    promo: { title: "Real Estate", subtitle: "Premium properties across the GCC.", cta: "Find property", imgSeed: "dubai-skyline-luxury-apartment", href: "/category/real-estate", accent: "#0064D2" },
  },
  sports: {
    popular: [
      { label: "Exercise & Fitness", href: "/category/exercise" },
      { label: "Team Sports", href: "/category/football" },
      { label: "Water Sports", href: "/category/water-sports" },
      { label: "Cycling", href: "/category/cycling" },
      { label: "Camping & Hiking", href: "/category/camping" },
      { label: "Martial Arts", href: "/category/martial-arts" },
    ],
    more: [
      { label: "Deals", href: "/listings?category=sports&sort=price_asc" },
      { label: "New Arrivals", href: "/listings?category=sports&sort=newest" },
    ],
    promo: { title: "Sporting Goods", subtitle: "Equipment for every sport.", cta: "Shop sports", imgSeed: "sports-fitness-gym-dubai", href: "/category/sports", accent: "#86B817" },
  },
  collectibles: {
    popular: [
      { label: "Antiques", href: "/category/antiques" },
      { label: "Art", href: "/category/art" },
      { label: "Coins & Paper Money", href: "/category/coins" },
      { label: "Stamps", href: "/category/stamps" },
      { label: "Comics & Books", href: "/category/comics" },
    ],
    more: [
      { label: "Live Auctions", href: "/auctions" },
      { label: "New Arrivals", href: "/listings?category=collectibles&sort=newest" },
    ],
    promo: { title: "Collectibles & Art", subtitle: "Rare finds and unique treasures.", cta: "Explore collectibles", imgSeed: "antique-vase-collectible", href: "/category/collectibles", accent: "#735200" },
  },
};

const CATEGORY_LINKS = [
  { label: "Saved",         href: "/buyer/watchlist",              key: null },
  { label: "New Arrivals",  href: "/listings?sort=newest",         key: null },
  { label: "Electronics",   href: "/category/electronics",  key: "electronics" },
  { label: "Fashion",       href: "/category/fashion",     key: "fashion" },
  { label: "Home & Garden", href: "/category/home-garden",  key: "home-garden" },
  { label: "Real Estate",   href: "/category/real-estate",  key: "real-estate" },
  { label: "Motors",        href: "/category/motors",       key: "motors" },
  { label: "Sports",        href: "/category/sports",       key: "sports" },
  { label: "Collectibles",  href: "/category/collectibles",  key: "collectibles" },
];

// ALL dropdown: L1 categories in English (eBay order, starting from Antiques)
const ALL_DROPDOWN_CATS = CATEGORY_TREE.map(c => ({ slug: c.slug, label: c.label, color: c.color, icon: c.icon }));

const TRAVELER_ITEMS = [
  { icon: "🛍", label: "Request a Product", sub: "Ask a traveler to bring it", href: "/reverse-auctions/new" },
  { icon: "🧳", label: "Browse Travelers",  sub: "See available trip routes",   href: "/traveler/orders" },
  { icon: "❓", label: "How It Works",       sub: "Learn about crowdshipping",   href: "/how-it-works" },
];

const FULFILLMENT_ITEMS = [
  { icon: "📦", label: "Shipping & Delivery",  sub: "Estimate costs & times",    href: "/shipping" },
  { icon: "🏪", label: "Fulfillment Center",    sub: "Store, pick & ship",        href: "/help" },
  { icon: "🔄", label: "Returns & Refunds",     sub: "Easy 30-day returns",       href: "/refund-policy" },
];

export function Header() {
  const t = useTranslations("nav");
  const [query, setQuery] = useState("");
  const [selectedCat, setSelectedCat] = useState("All Categories");
  const [catOpen, setCatOpen] = useState(false);
  const [activeMenu, setActiveMenu] = useState<string | null>(null);
  const closeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [watchlistOpen, setWatchlistOpen] = useState(false);
  const watchTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [notifOpen, setNotifOpen] = useState(false);
  const notifTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [travelerOpen, setTravelerOpen] = useState(false);
  const travelerTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [fulfillOpen, setFulfillOpen] = useState(false);
  const fulfillTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [mobileOpen, setMobileOpen] = useState(false);

  const router = useRouter();
  const { user, isAuthenticated } = useAuthStore();
  const role = user?.role;
  const canAccessOps = hasAnyPermission(role, [PERMISSIONS.OPS_READ, PERMISSIONS.OPS_MANAGE]);
  const canAccessFounder = isInternalRole(role) && hasAnyPermission(role, [PERMISSIONS.ADMIN_DASHBOARD_READ]);
  const canAccessSeller = isSeller(role);
  const canAccessTraveler = isTraveler(role);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (query.trim()) router.push(`/search?q=${encodeURIComponent(query.trim())}`);
    else router.push("/listings");
  };

  const openMenu = (key: string | null) => {
    if (closeTimer.current) clearTimeout(closeTimer.current);
    setActiveMenu(key);
  };
  const scheduleClose = () => {
    closeTimer.current = setTimeout(() => setActiveMenu(null), 180);
  };

  const activeMegaData = activeMenu && activeMenu !== "categories" ? MEGA_MENUS[activeMenu] : null;

  return (
    <header className="sticky top-0 z-50 shadow-md" style={{ position: "sticky" }}>

      {/* ══════════════════════════════════════════════
          ROW 1 — TOP BAR  (dark navy)
      ══════════════════════════════════════════════ */}
      <div className="bg-[#003D7A] text-white/80 text-[11px]">
        <div className="max-w-7xl mx-auto px-4 flex items-center justify-between h-8">

          {/* LEFT — greeting + quick links */}
          <div className="flex items-center gap-3 overflow-x-auto scrollbar-none">
            {isAuthenticated ? (
              <span className="text-white/90 whitespace-nowrap">
                Hi! <span className="text-[#FFC220] font-semibold">{user?.name?.split(" ")[0]}</span>
              </span>
            ) : (
              <span className="whitespace-nowrap">
                Hi! <Link href="/login" className="text-[#FFC220] hover:underline font-semibold">Sign in</Link>
                {" "}or{" "}
                <Link href="/register" className="text-[#FFC220] hover:underline font-semibold">Register</Link>
              </span>
            )}
            <span className="text-white/20 hidden sm:block">|</span>
            <Link href="/auctions" className="hover:text-white transition-colors whitespace-nowrap hidden sm:block">Daily Deals</Link>
            <Link href="/help" className="hover:text-white transition-colors whitespace-nowrap hidden md:block">Help &amp; Contact</Link>
            <Link href="/how-it-works" className="hover:text-white transition-colors whitespace-nowrap hidden lg:block">How It Works</Link>
            {isAuthenticated && <Link href="/buyer"   className="hover:text-white transition-colors whitespace-nowrap hidden lg:block">My Orders</Link>}
            {isAuthenticated && canAccessSeller   && <Link href="/seller"   className="hover:text-white transition-colors whitespace-nowrap hidden lg:block">Seller Hub</Link>}
            {isAuthenticated && canAccessTraveler && <Link href="/traveler" className="hover:text-white transition-colors whitespace-nowrap hidden xl:block">Traveler Hub</Link>}
            {isAuthenticated && canAccessOps      && <Link href="/ops"      className="hover:text-white transition-colors whitespace-nowrap hidden xl:block">Ops</Link>}
            {isAuthenticated && canAccessFounder  && <Link href="/founder"  className="hover:text-white transition-colors whitespace-nowrap hidden xl:block">Founder</Link>}
          </div>

          {/* RIGHT — Sell · Watchlist · My Mnbarh · Bell · Cart */}
          <div className="flex items-center gap-3 shrink-0">

            {/* Sell */}
            <Link href="/sell" className="hover:text-white font-semibold transition-colors whitespace-nowrap flex items-center gap-1">
              <Tag size={11} /> {t("sell")}
            </Link>

            <span className="text-white/20">|</span>

            {/* Watchlist */}
            <div
              className="relative"
              onMouseEnter={() => { if (watchTimer.current) clearTimeout(watchTimer.current); setWatchlistOpen(true); }}
              onMouseLeave={() => { watchTimer.current = setTimeout(() => setWatchlistOpen(false), 200); }}
            >
              <button className="flex items-center gap-0.5 hover:text-[#FFC220] transition-colors whitespace-nowrap">
                <Heart size={12} className="hidden sm:block" />
                <span className="font-semibold">{t("favorites")}</span>
                <ChevronDown size={10} className={`transition-transform duration-200 ${watchlistOpen ? "rotate-180" : ""}`} />
              </button>
              {watchlistOpen && (
                <div className="absolute right-0 top-full mt-1 w-80 bg-white rounded-xl shadow-2xl border border-gray-100 z-[90] text-gray-800">
                  <div className="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
                    <span className="font-bold text-sm text-gray-900">{t("favorites")}</span>
                    <Link href="/buyer/watchlist" className="text-[#0071CE] text-xs font-semibold hover:underline">Go to Watchlist</Link>
                  </div>
                  {isAuthenticated ? (
                    <>
                      <p className="text-xs text-gray-500 px-4 pt-3 pb-1 font-medium uppercase tracking-wide">Recently watched</p>
                      {[
                        { title: "iPhone 15 Pro Max 256GB Natural Titanium", price: "AED 4,299", ends: "Ends in 2h 14m", img: "https://images.unsplash.com/photo-1695048133142-1a20484429be?w=60&h=60&fit=crop" },
                        { title: "Nike Air Max 270 – Size 42 EU",             price: "AED 380",   ends: "Ends in 5h 40m", img: "https://images.unsplash.com/photo-1542291026-7eec264c27ff?w=60&h=60&fit=crop" },
                        { title: "Samsung 55\" QLED Smart TV 4K",             price: "AED 2,199", ends: "Ends in 1d 3h",  img: "https://images.unsplash.com/photo-1593784991095-a205069470b6?w=60&h=60&fit=crop" },
                      ].map((item, i) => (
                        <Link key={i} href="/buyer/watchlist" className="flex items-center gap-3 px-4 py-2.5 hover:bg-blue-50 transition-colors group">
                          {/* eslint-disable-next-line @next/next/no-img-element */}
                          <img src={item.img} alt="" className="w-12 h-12 rounded object-cover shrink-0" />
                          <div className="flex-1 min-w-0">
                            <p className="text-xs font-medium text-gray-800 line-clamp-2 group-hover:text-[#0071CE]">{item.title}</p>
                            <p className="text-[11px] text-orange-500 font-semibold mt-0.5">{item.ends}</p>
                            <p className="text-xs font-bold text-gray-900">{item.price}</p>
                          </div>
                        </Link>
                      ))}
                      <div className="px-4 py-3 border-t border-gray-100 text-center">
                        <Link href="/buyer/watchlist" className="text-sm font-semibold text-[#0071CE] hover:underline">See all watched items →</Link>
                      </div>
                    </>
                  ) : (
                    <div className="px-4 py-5 text-center">
                      <Heart size={28} className="text-[#0071CE] mx-auto mb-2" />
                      <p className="font-bold text-sm text-gray-900 mb-1">Don&apos;t miss out!</p>
                      <p className="text-xs text-gray-500 mb-4">Sign in to see items you&apos;ve been watching.</p>
                      <div className="flex gap-2">
                        <Link href="/login"    className="flex-1 bg-[#0071CE] text-white text-sm font-semibold py-2 rounded-full text-center hover:bg-[#0058a3] transition-colors">Sign in</Link>
                        <Link href="/register" className="flex-1 border border-[#0071CE] text-[#0071CE] text-sm font-semibold py-2 rounded-full text-center hover:bg-blue-50 transition-colors">Register</Link>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* My Mnbarh account dropdown */}
            <AccountDropdown />

            {/* Notifications */}
            <div
              className="relative"
              onMouseEnter={() => { if (notifTimer.current) clearTimeout(notifTimer.current); setNotifOpen(true); }}
              onMouseLeave={() => { notifTimer.current = setTimeout(() => setNotifOpen(false), 200); }}
            >
              <button className="relative flex items-center hover:text-[#FFC220] transition-colors">
                <Bell size={15} />
                {isAuthenticated && (
                  <span className="absolute -top-1 -right-1.5 w-3.5 h-3.5 bg-red-500 rounded-full text-[9px] text-white font-bold flex items-center justify-center">3</span>
                )}
              </button>
              {notifOpen && (
                <div className="absolute right-0 top-full mt-1 w-80 bg-white rounded-xl shadow-2xl border border-gray-100 z-[90] text-gray-800">
                  <div className="px-4 py-3 border-b border-gray-100 flex items-center justify-between">
                    <span className="font-bold text-sm text-gray-900">Notifications</span>
                    {isAuthenticated && <button className="text-[#0071CE] text-xs font-semibold hover:underline">Mark all read</button>}
                  </div>
                  {isAuthenticated ? (
                    <>
                      {[
                        { icon: "🏷️", text: "Your bid on iPhone 15 Pro was outbid",     time: "2 min ago",  unread: true },
                        { icon: "✅", text: "Order #GC-4821 has been shipped",           time: "1 hour ago", unread: true },
                        { icon: "💬", text: "New message from seller Ahmed K.",           time: "3 hours ago",unread: true },
                        { icon: "⏰", text: "Auction ending in 30 min: MacBook Pro M3",  time: "4 hours ago",unread: false },
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
                    </>
                  ) : (
                    <div className="px-4 py-5 text-center">
                      <Bell size={28} className="text-[#0071CE] mx-auto mb-2" />
                      <p className="font-bold text-sm text-gray-900 mb-1">Stay in the loop</p>
                      <p className="text-xs text-gray-500 mb-4">Sign in to get alerts for bids, messages, and deals.</p>
                      <div className="flex gap-2">
                        <Link href="/login"    className="flex-1 bg-[#0071CE] text-white text-sm font-semibold py-2 rounded-full text-center hover:bg-[#0058a3] transition-colors">Sign in</Link>
                        <Link href="/register" className="flex-1 border border-[#0071CE] text-[#0071CE] text-sm font-semibold py-2 rounded-full text-center hover:bg-blue-50 transition-colors">Register</Link>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Cart */}
            <CartIcon />

            {/* Mobile hamburger */}
            <button
              onClick={() => setMobileOpen((v) => !v)}
              className="md:hidden hover:text-[#FFC220] transition-colors"
            >
              {mobileOpen ? <X size={18} /> : <Menu size={18} />}
            </button>
          </div>
        </div>
      </div>

      {/* ══════════════════════════════════════════════
          ROW 2 — MAIN BAR  (brand blue)
      ══════════════════════════════════════════════ */}
      <div className="bg-[#0071CE]">
        <div className="max-w-7xl mx-auto px-4 py-3 flex items-center gap-3">

          {/* Logo */}
          <Link href="/" className="shrink-0">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src="/logo-mnbarh.svg"
              alt="mnbarh"
              height={40}
              width={130}
              className="h-10 w-auto"
            />
          </Link>

          {/* Feature dropdowns */}
          <div className="hidden md:flex items-center gap-2 shrink-0">

            {/* Buy via Traveler */}
            <div
              className="relative"
              onMouseEnter={() => { if (travelerTimer.current) clearTimeout(travelerTimer.current); setTravelerOpen(true); }}
              onMouseLeave={() => { travelerTimer.current = setTimeout(() => setTravelerOpen(false), 180); }}
            >
              <button className="flex items-center gap-2 bg-[#003D7A] hover:bg-[#002d5e] text-white text-[12px] font-semibold px-3 py-2 rounded-xl transition-colors whitespace-nowrap">
                <Plane size={14} />
                <span className="hidden lg:inline">Buy via Traveler</span>
                <span className="lg:hidden">Traveler</span>
                <ChevronDown size={11} className={`transition-transform duration-200 ${travelerOpen ? "rotate-180" : ""}`} />
              </button>
              {travelerOpen && (
                <div className="absolute left-0 top-full mt-1 w-64 bg-white rounded-xl shadow-2xl border border-gray-100 z-[90] text-gray-800 py-2">
                  {TRAVELER_ITEMS.map((item) => (
                    <Link key={item.href} href={item.href} onClick={() => setTravelerOpen(false)}
                      className="flex items-start gap-3 px-4 py-3 hover:bg-blue-50 transition-colors group">
                      <span className="text-xl shrink-0 mt-0.5">{item.icon}</span>
                      <div>
                        <p className="text-sm font-semibold text-gray-900 group-hover:text-[#0071CE]">{item.label}</p>
                        <p className="text-xs text-gray-500 mt-0.5">{item.sub}</p>
                      </div>
                    </Link>
                  ))}
                </div>
              )}
            </div>

            {/* Fulfillment Center */}
            <div
              className="relative"
              onMouseEnter={() => { if (fulfillTimer.current) clearTimeout(fulfillTimer.current); setFulfillOpen(true); }}
              onMouseLeave={() => { fulfillTimer.current = setTimeout(() => setFulfillOpen(false), 180); }}
            >
              <button className="flex items-center gap-2 bg-[#003D7A] hover:bg-[#002d5e] text-white text-[12px] font-semibold px-3 py-2 rounded-xl transition-colors whitespace-nowrap">
                <Package size={14} />
                <span className="hidden lg:inline">Fulfillment Center</span>
                <span className="lg:hidden">Fulfillment</span>
                <ChevronDown size={11} className={`transition-transform duration-200 ${fulfillOpen ? "rotate-180" : ""}`} />
              </button>
              {fulfillOpen && (
                <div className="absolute left-0 top-full mt-1 w-64 bg-white rounded-xl shadow-2xl border border-gray-100 z-[90] text-gray-800 py-2">
                  {FULFILLMENT_ITEMS.map((item) => (
                    <Link key={item.href} href={item.href} onClick={() => setFulfillOpen(false)}
                      className="flex items-start gap-3 px-4 py-3 hover:bg-blue-50 transition-colors group">
                      <span className="text-xl shrink-0 mt-0.5">{item.icon}</span>
                      <div>
                        <p className="text-sm font-semibold text-gray-900 group-hover:text-[#0071CE]">{item.label}</p>
                        <p className="text-xs text-gray-500 mt-0.5">{item.sub}</p>
                      </div>
                    </Link>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Search bar — unified white pill, dropdown rendered via portal-like positioning */}
          <form onSubmit={handleSearch} className="flex-1 flex items-center min-w-0 relative">
            {/* The white pill: category button + input + camera */}
            <div className="flex flex-1 bg-white rounded-full shadow-inner h-10">
              {/* Category picker button (part of the pill) */}
              <button
                type="button"
                onClick={() => setCatOpen(!catOpen)}
                className="flex items-center gap-1 px-3 text-xs text-gray-600 hover:bg-gray-50 transition-colors whitespace-nowrap rounded-l-full shrink-0"
              >
                {selectedCat === "All Categories" ? "All" : selectedCat.length > 12 ? selectedCat.slice(0, 12) + "…" : selectedCat}
                <ChevronDown size={11} className={`transition-transform duration-200 ${catOpen ? "rotate-180" : ""}`} />
              </button>
              <div className="w-px bg-gray-200 self-stretch my-2 shrink-0" />
              <input
                type="text"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Search products or paste a product link…"
                className="flex-1 px-3 text-sm text-gray-800 outline-none placeholder-gray-400 bg-transparent min-w-0"
              />
              <button type="button" className="flex items-center px-3 text-gray-400 hover:text-[#0071CE] transition-colors shrink-0">
                <Camera size={16} />
              </button>
            </div>
            <button type="submit"
              className="bg-[#FFC220] hover:bg-yellow-400 text-gray-900 px-5 py-2.5 rounded-full font-bold text-sm transition-colors shrink-0 flex items-center gap-1.5 ml-2"
            >
              <Search size={15} />
              <span className="hidden sm:inline">Search</span>
            </button>

            {/* Advanced search */}
            <Link href="/advanced-search"
              className="shrink-0 hidden lg:flex items-center gap-1 text-white/80 hover:text-white text-xs font-medium transition-colors whitespace-nowrap"
            >
              <SlidersHorizontal size={13} /> Advanced
            </Link>

            {/* Category dropdown — rendered at form level so it escapes any inner overflow */}
            {catOpen && (
              <div className="absolute top-full left-0 mt-1 bg-white border border-gray-200 rounded-xl shadow-2xl z-[100] overflow-hidden"
                style={{ width: 240, maxHeight: 480 }}
                onMouseLeave={() => setCatOpen(false)}
              >
                <div className="overflow-y-auto max-h-[480px] py-1">
                  <button type="button"
                    onClick={() => { setSelectedCat("All Categories"); setCatOpen(false); }}
                    className={`w-full text-left px-4 py-2 text-sm flex items-center gap-2 transition-colors ${selectedCat === "All Categories" ? "bg-blue-50 text-[#0071CE] font-semibold" : "text-gray-700 hover:bg-blue-50 hover:text-[#0071CE]"}`}
                  >
                    <Search size={12} className="shrink-0" />
                    All Categories
                  </button>
                  {ALL_DROPDOWN_CATS.map((c) => (
                    <button key={c.slug} type="button"
                      onClick={() => { setSelectedCat(c.label); setCatOpen(false); }}
                      className={`w-full text-left px-4 py-1.5 text-sm flex items-center gap-2 transition-colors ${selectedCat === c.label ? "bg-blue-50 text-[#0071CE] font-semibold" : "text-gray-700 hover:bg-blue-50 hover:text-[#0071CE]"}`}
                    >
                      {c.label}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </form>

          {/* Country detector + Language switcher */}
          <div className="shrink-0 relative flex items-center gap-2">
            <LanguageSwitcher />
            <CountryDetector />
          </div>
        </div>
      </div>

      {/* ══════════════════════════════════════════════
          ROW 3 — CATEGORY NAV BAR
      ══════════════════════════════════════════════ */}
      <div className="bg-[#ECEFF3] border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 flex items-center gap-2 py-2 overflow-x-auto scrollbar-none">

          {/* Live badge */}
          <Link href="/auctions/live"
            className="shrink-0 flex items-center gap-1.5 px-4 h-8 text-[13px] font-bold text-[#0071CE] bg-white border border-gray-200 hover:bg-blue-50 rounded-full transition-colors whitespace-nowrap"
          >
            <Radio size={13} className="text-red-500 animate-pulse" /> Mnbarh Live
          </Link>

          {/* Categories — shows all L1 categories */}
          <button
            onMouseEnter={() => openMenu("categories")}
            onMouseLeave={scheduleClose}
            className={`whitespace-nowrap shrink-0 h-8 px-4 text-[13px] font-semibold flex items-center gap-1.5 rounded-full border transition-colors ${activeMenu === "categories" ? "bg-blue-50 text-[#0071CE] border-[#0071CE]/30" : "bg-white border-gray-200 text-gray-700 hover:bg-gray-50 hover:text-[#0071CE]"}`}
          >
            <Menu size={13} /> Categories
            <ChevronDown size={11} className={`transition-transform duration-200 ${activeMenu === "categories" ? "rotate-180" : ""}`} />
          </button>

          {/* Category links */}
          {CATEGORY_LINKS.map((item) => (
            <button
              key={item.href}
              onMouseEnter={() => openMenu(item.key)}
              onMouseLeave={scheduleClose}
              onClick={() => router.push(item.href)}
              className={`whitespace-nowrap px-4 h-8 text-[13px] transition-colors flex items-center rounded-full border ${activeMenu === item.key && item.key ? "bg-blue-50 text-[#0071CE] border-[#0071CE]/30" : "bg-white border-gray-200 text-gray-700 hover:bg-gray-50 hover:text-[#0071CE]"}`}
            >
              {item.label}
            </button>
          ))}

          {/* Gift Cards — replaces MORE */}
          <Link href="/gift-cards"
            className="whitespace-nowrap ml-auto shrink-0 h-8 px-4 text-[13px] font-semibold flex items-center gap-1.5 rounded-full border bg-white border-gray-200 text-gray-700 hover:bg-gray-50 hover:text-[#0071CE] transition-colors"
          >
            <Tag size={13} /> Gift Cards
          </Link>
        </div>
      </div>

      {/* ── Categories dropdown ── */}
      {activeMenu === "categories" && (
        <div className="absolute left-0 right-0 z-[60]" style={{ top: "100%" }}
          onMouseEnter={() => openMenu("categories")} onMouseLeave={scheduleClose}
        >
          <div className="max-w-7xl mx-auto px-4 pb-4">
            <div className="bg-white rounded-b-xl shadow-2xl border border-t-0 border-gray-200 overflow-hidden" style={{ maxHeight: 480 }}>
              <div className="overflow-y-auto max-h-[480px]">
              <div className="grid grid-cols-4 gap-0">
                {DEPARTMENTS_CATS.map((cat) => (
                  <div key={cat.slug} className="border-r border-b border-gray-100 last:border-r-0">
                    <Link href={`/category/${cat.slug}`}
                      onClick={() => setActiveMenu(null)}
                      className="flex items-center gap-2.5 px-4 py-3 hover:bg-blue-50 transition-colors group"
                    >
                      <span className="text-lg">{cat.icon}</span>
                      <p className="text-sm font-semibold text-gray-800 group-hover:text-[#0071CE]">{cat.label}</p>
                    </Link>
                    {cat.children.length > 0 && (
                      <div className="px-4 pb-2">
                        {cat.children.map((ch) => (
                          <Link key={ch.slug} href={ch.href}
                            onClick={() => setActiveMenu(null)}
                            className="block py-0.5 text-xs text-gray-500 hover:text-[#0071CE] transition-colors"
                          >
                            {ch.label}
                          </Link>
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* ── Mega dropdown ── */}
      {activeMegaData && (
        <div className="absolute left-0 right-0 z-[60]" style={{ top: "100%" }}
          onMouseEnter={() => openMenu(activeMenu)} onMouseLeave={scheduleClose}
        >
          <div className="max-w-7xl mx-auto px-4 pb-4">
            <div className="bg-white rounded-b-2xl shadow-2xl border border-t-0 border-gray-200 overflow-hidden flex">
              <div className="w-56 border-r border-gray-100 p-5 shrink-0 bg-gray-50">
                <p className="text-[10px] font-black text-gray-400 uppercase tracking-widest mb-3">Most popular</p>
                <ul className="space-y-0.5">
                  {activeMegaData.popular.map((item) => (
                    <li key={item.label}>
                      <Link href={item.href} onClick={() => setActiveMenu(null)}
                        className="flex items-center gap-2 text-sm text-gray-700 hover:text-[#0071CE] hover:bg-blue-50 px-2 py-1.5 rounded-lg transition-colors group"
                      >
                        <ChevronRight size={12} className="text-gray-300 group-hover:text-[#0071CE] shrink-0" /> {item.label}
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>
              <div className="w-48 border-r border-gray-100 p-5 shrink-0">
                <p className="text-[10px] font-black text-gray-400 uppercase tracking-widest mb-3">More categories</p>
                <ul className="space-y-0.5">
                  {activeMegaData.more.map((item) => (
                    <li key={item.label}>
                      <Link href={item.href} onClick={() => setActiveMenu(null)}
                        className="flex items-center gap-2 text-sm text-gray-700 hover:text-[#0071CE] hover:bg-blue-50 px-2 py-1.5 rounded-lg transition-colors group"
                      >
                        <ChevronRight size={12} className="text-gray-300 group-hover:text-[#0071CE] shrink-0" /> {item.label}
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>
              <div className="flex-1 p-4">
                <Link href={activeMegaData.promo.href} onClick={() => setActiveMenu(null)}
                  className="block h-full rounded-xl overflow-hidden relative group cursor-pointer min-h-[200px]"
                >
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img src={`https://picsum.photos/seed/${activeMegaData.promo.imgSeed}/600/300`}
                    alt={activeMegaData.promo.title}
                    className="absolute inset-0 w-full h-full object-cover group-hover:scale-[1.03] transition-transform duration-500"
                  />
                  <div className="absolute inset-0 bg-gradient-to-r from-black/70 via-black/40 to-transparent" />
                  <div className="relative z-10 p-6 h-full flex flex-col justify-between">
                    <div>
                      <h3 className="text-2xl font-black text-white leading-tight">{activeMegaData.promo.title}</h3>
                      <p className="text-sm text-white/80 mt-1.5 leading-relaxed max-w-xs">{activeMegaData.promo.subtitle}</p>
                    </div>
                    <span className="mt-4 inline-flex items-center gap-1.5 text-sm font-bold px-5 py-2 rounded-full text-white w-fit shadow-lg hover:opacity-90 transition-opacity"
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

      {/* ── Mobile menu ── */}
      {mobileOpen && (
        <div className="md:hidden bg-[#003D7A] text-white px-4 py-4 space-y-3 border-t border-white/10">
          <form onSubmit={handleSearch} className="flex gap-2">
            <input value={query} onChange={(e) => setQuery(e.target.value)}
              placeholder="Search Mnbarh…"
              className="flex-1 px-4 py-2 rounded-full text-sm text-gray-800 outline-none"
            />
            <button type="submit" className="bg-[#FFC220] text-gray-900 px-4 py-2 rounded-full font-bold text-sm">
              <Search size={15} />
            </button>
          </form>
          <div className="grid grid-cols-2 gap-2 pt-2">
            {[
              { label: "Daily Deals",   href: "/auctions" },
              { label: "Help",          href: "/help" },
              { label: "My Orders",     href: "/buyer/orders" },
              { label: "Watchlist",     href: "/buyer/watchlist" },
              { label: "Seller Hub",    href: "/seller" },
              { label: "Traveler Hub",  href: "/traveler" },
              { label: "Wallet",        href: "/wallet" },
              { label: "Sell",          href: "/sell" },
              { label: "Gift Cards",   href: "/gift-cards" },
            ].map(({ label, href }) => (
              <Link key={href} href={href} onClick={() => setMobileOpen(false)}
                className="flex items-center gap-2 text-sm font-medium text-white/80 hover:text-white hover:bg-white/10 px-3 py-2 rounded-lg transition-colors"
              >
                {label}
              </Link>
            ))}
          </div>
        </div>
      )}
    </header>
  );
}
