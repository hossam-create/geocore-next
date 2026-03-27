import { useQuery } from "@tanstack/react-query";
import { HeroBanner } from "@/components/home/HeroBanner";
import { DeliveryBar } from "@/components/home/DeliveryBar";
import { CategorySection } from "@/components/home/CategorySection";
import { PromoGrid } from "@/components/home/PromoGrid";
import { LiveAuctions } from "@/components/home/LiveAuctions";
import { FeaturedListings } from "@/components/home/FeaturedListings";
import { Recommendations } from "@/components/home/Recommendations";
import { ProductCarousel } from "@/components/home/ProductCarousel";
import api from "@/lib/api";

export default function HomePage() {
  const { data: newListings, isError: newListingsError } = useQuery({
    queryKey: ["listings", "newest"],
    queryFn: () => api.get("/listings?sort=newest&per_page=20").then((r) => r.data.data),
    retry: 1,
  });

  const { data: flashDeals, isError: flashDealsError } = useQuery({
    queryKey: ["listings", "flash-deals"],
    queryFn: () =>
      api.get("/listings?sort=price_asc&per_page=12&condition=new").then((r) => r.data.data),
    retry: 1,
  });

  return (
    <div className="bg-[#f2f8fd] min-h-screen">
      <HeroBanner />
      <DeliveryBar />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 py-8 space-y-10">
        <CategorySection />

        <PromoGrid />

        {flashDealsError ? (
          <div className="bg-red-50 border border-red-100 rounded-2xl px-6 py-4 text-sm text-red-600">
            Something went wrong loading flash deals. Please refresh the page.
          </div>
        ) : (
          <ProductCarousel
            title="Today's Flash Deals"
            icon="⚡"
            listings={flashDeals ?? []}
            viewAllHref="/listings?sort=price_asc"
            badge="Limited Time"
            badgeColor="#F97316"
          />
        )}

        <LiveAuctions />

        {newListingsError ? (
          <div className="bg-red-50 border border-red-100 rounded-2xl px-6 py-4 text-sm text-red-600">
            Something went wrong loading new arrivals. Please refresh the page.
          </div>
        ) : (
          <ProductCarousel
            title="New Arrivals"
            icon="🆕"
            listings={newListings ?? []}
            viewAllHref="/listings?sort=newest"
            badge="Just listed"
            badgeColor="#059669"
          />
        )}

        <Recommendations />

        <FeaturedListings />
      </div>
    </div>
  );
}
