import { HeroBanner } from "@/components/home/HeroBanner";
import { CategorySection } from "@/components/home/CategorySection";
import { LiveAuctions } from "@/components/home/LiveAuctions";
import { FeaturedListings } from "@/components/home/FeaturedListings";
import { Recommendations } from "@/components/home/Recommendations";

export default function HomePage() {
  return (
    <div>
      <HeroBanner />
      <div className="max-w-7xl mx-auto px-4 py-10 space-y-12">
        <CategorySection />
        <Recommendations />
        <LiveAuctions />
        <FeaturedListings />
      </div>
    </div>
  );
}
