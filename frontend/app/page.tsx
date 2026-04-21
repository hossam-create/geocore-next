'use client'
import { useQuery } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { HeroBanner } from "@/components/home/HeroBanner";
import { DeliveryBar } from "@/components/home/DeliveryBar";
import { CategorySection } from "@/components/home/CategorySection";
import { PromoGrid } from "@/components/home/PromoGrid";
import { LiveAuctions } from "@/components/home/LiveAuctions";
import { FeaturedListings } from "@/components/home/FeaturedListings";
import { Recommendations } from "@/components/home/Recommendations";
import { ProductCarousel } from "@/components/home/ProductCarousel";
import api from "@/lib/api";

const NEW_ARRIVALS = [
  { id: "n1", title: "Samsung Galaxy S24 Ultra 512GB Titanium", price: 5200, currency: "AED", type: "fixed", city: "Dubai", condition: "new" },
  { id: "n2", title: "IKEA MALM Bed Frame King + Mattress", price: 1800, currency: "AED", type: "fixed", city: "Abu Dhabi", condition: "new" },
  { id: "n3", title: "PlayStation VR2 Complete Bundle", price: 1650, currency: "AED", type: "fixed", city: "Riyadh", condition: "new" },
  { id: "n4", title: "Dyson V15 Detect Absolute Cordless", price: 2200, currency: "AED", type: "fixed", city: "Dubai", condition: "new" },
  { id: "n5", title: "Gucci Ophidia GG Tote Bag — Beige", price: 6500, currency: "AED", type: "fixed", city: "Kuwait City", condition: "new" },
  { id: "n6", title: "Sony A7R V Full-Frame Mirrorless Camera", price: 14500, currency: "AED", type: "fixed", city: "Doha", condition: "new" },
  { id: "n7", title: "Apple Watch Ultra 2 Titanium 49mm", price: 4200, currency: "AED", type: "fixed", city: "Muscat", condition: "new" },
  { id: "n8", title: "Lenovo ThinkPad X1 Carbon Gen 12", price: 7800, currency: "AED", type: "fixed", city: "Riyadh", condition: "new" },
];

const FLASH_DEALS = [
  { id: "d1", title: 'LG C3 65" OLED evo 4K Smart TV — Open Box', price: 4800, currency: "AED", type: "fixed", city: "Dubai", condition: "like-new" },
  { id: "d2", title: "AirPods Pro 2nd Gen with MagSafe Case", price: 750, currency: "AED", type: "fixed", city: "Abu Dhabi", condition: "new" },
  { id: "d3", title: "iPad Pro 12.9\" M2 + Magic Keyboard", price: 4400, currency: "AED", type: "fixed", city: "Riyadh", condition: "new" },
  { id: "d4", title: "Canon EOS R6 Mark II Body Only", price: 9200, currency: "AED", type: "fixed", city: "Dubai", condition: "new" },
  { id: "d5", title: "Nintendo Switch OLED + 10 Games", price: 1450, currency: "AED", type: "fixed", city: "Doha", condition: "like-new" },
  { id: "d6", title: 'Dell XPS 15 9530 i9 RTX 4070', price: 9800, currency: "AED", type: "fixed", city: "Kuwait City", condition: "new" },
  { id: "d7", title: "Bose QuietComfort Ultra Headphones", price: 1650, currency: "AED", type: "fixed", city: "Muscat", condition: "new" },
];

export default function HomePage() {
  const t = useTranslations("home");
  const { data: newListings } = useQuery({
    queryKey: ["listings", "newest"],
    queryFn: () => api.get("/listings?sort=newest&per_page=20").then((r) => r.data.data),
    retry: false,
  });

  return (
    <div className="bg-[#f2f8fd] min-h-screen">
      <HeroBanner />
      <DeliveryBar />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 py-8 space-y-10">
        {/* Shop by Department */}
        <CategorySection />

        {/* Promo Grid */}
        <PromoGrid />

        {/* Today's Flash Deals */}
        <ProductCarousel
          title={t("trending")}
          icon="⚡"
          listings={FLASH_DEALS}
          viewAllHref="/listings?sort=price_asc"
          badge="Limited Time"
          badgeColor="#F97316"
        />

        {/* Live Auctions */}
        <LiveAuctions />

        {/* New Arrivals */}
        <ProductCarousel
          title={t("newArrivals")}
          icon="🆕"
          listings={newListings?.length ? newListings : NEW_ARRIVALS}
          viewAllHref="/listings?sort=newest"
          badge="Just listed"
          badgeColor="#059669"
        />

        {/* AI Recommendations */}
        <Recommendations />

        {/* Featured Listings */}
        <FeaturedListings />
      </div>
    </div>
  );
}
