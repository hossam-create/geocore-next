import { useQuery } from "@tanstack/react-query";
import { Link } from "wouter";
import api from "@/lib/api";
import { ListingCard } from "@/components/listings/ListingCard";
import { LoadingGrid } from "@/components/ui/LoadingGrid";

const MOCK_LISTINGS = [
  { id: "1", title: "iPhone 15 Pro Max 256GB Space Black", price: 4200, currency: "AED", type: "fixed", is_featured: true, city: "Dubai", condition: "new" },
  { id: "2", title: "Toyota Camry 2023 — Full Option", price: 89000, currency: "AED", type: "auction", is_featured: true, city: "Abu Dhabi", condition: "like-new" },
  { id: "3", title: "MacBook Pro M3 Max 14-inch", price: 9500, currency: "AED", type: "fixed", is_featured: true, city: "Riyadh", condition: "new" },
  { id: "4", title: "Chanel Classic Flap Medium — Beige", price: 18000, currency: "AED", type: "auction", is_featured: true, city: "Kuwait City", condition: "good" },
  { id: "5", title: "Samsung 85\" Neo QLED 8K Smart TV", price: 12000, currency: "AED", type: "fixed", is_featured: false, city: "Doha", condition: "new" },
  { id: "6", title: "Dubai JBR Sea View 2BR Apartment", price: 4200000, currency: "AED", type: "fixed", is_featured: true, city: "Dubai", condition: "new" },
  { id: "7", title: "Rolex Datejust 41mm Oyster", price: 38000, currency: "AED", type: "auction", is_featured: true, city: "Manama", condition: "like-new" },
  { id: "8", title: "PlayStation 5 Slim + 3 Games Bundle", price: 2100, currency: "AED", type: "fixed", is_featured: false, city: "Muscat", condition: "new" },
];

export function FeaturedListings() {
  const { data: listings, isLoading } = useQuery({
    queryKey: ["listings", "featured"],
    queryFn: () => api.get("/listings?is_featured=true&per_page=12").then((r) => r.data.data),
    retry: false,
  });

  const displayListings = listings?.length ? listings : MOCK_LISTINGS;

  return (
    <section>
      <div className="flex items-center justify-between mb-5">
        <h2 className="text-2xl font-bold text-gray-900">⭐ Featured Listings</h2>
        <Link href="/listings" className="text-[#0071CE] text-sm font-semibold hover:underline">
          See all →
        </Link>
      </div>
      {isLoading ? (
        <LoadingGrid count={8} />
      ) : (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
          {displayListings.slice(0, 12).map((listing: any) => (
            <ListingCard key={listing.id} listing={listing} />
          ))}
        </div>
      )}
    </section>
  );
}
