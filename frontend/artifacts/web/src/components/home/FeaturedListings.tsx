import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";
import { ProductCarousel } from "./ProductCarousel";

const MOCK_LISTINGS = [
  { id: "f1", title: "iPhone 15 Pro Max 256GB Space Black", price: 4200, currency: "AED", type: "fixed", is_featured: true, city: "Dubai", condition: "new" },
  { id: "f2", title: "Toyota Camry 2023 Full Option V6", price: 89000, currency: "AED", type: "fixed", is_featured: true, city: "Abu Dhabi", condition: "like-new" },
  { id: "f3", title: "MacBook Pro M3 Max 14-inch Silver", price: 9500, currency: "AED", type: "fixed", is_featured: true, city: "Riyadh", condition: "new" },
  { id: "f4", title: "Chanel Classic Flap Medium Beige", price: 18000, currency: "AED", type: "fixed", is_featured: true, city: "Kuwait City", condition: "good" },
  { id: "f5", title: 'Samsung 85" Neo QLED 8K Smart TV', price: 12000, currency: "AED", type: "fixed", is_featured: false, city: "Doha", condition: "new" },
  { id: "f6", title: "Dubai JBR Sea View 2BR Apartment", price: 4200000, currency: "AED", type: "fixed", is_featured: true, city: "Dubai", condition: "new" },
  { id: "f7", title: "Rolex Datejust 41mm Oyster Perpetual", price: 38000, currency: "AED", type: "fixed", is_featured: true, city: "Manama", condition: "like-new" },
  { id: "f8", title: "PlayStation 5 Slim + 3 Games Bundle", price: 2100, currency: "AED", type: "fixed", is_featured: false, city: "Muscat", condition: "new" },
  { id: "f9", title: "DJI Mavic 3 Pro Fly More Combo", price: 6800, currency: "AED", type: "fixed", is_featured: true, city: "Dubai", condition: "new" },
  { id: "f10", title: "Nike Air Jordan 1 Retro High OG", price: 850, currency: "AED", type: "fixed", is_featured: false, city: "Riyadh", condition: "new" },
];

export function FeaturedListings() {
  const { data: listings, isLoading } = useQuery({
    queryKey: ["listings", "featured"],
    queryFn: () => api.get("/listings?is_featured=true&per_page=20").then((r) => r.data.data),
    retry: false,
  });

  const display = listings?.length ? listings : MOCK_LISTINGS;

  return (
    <ProductCarousel
      title="Featured Listings"
      icon="⭐"
      listings={display}
      viewAllHref="/listings?is_featured=true"
      badge="Top Picks"
      badgeColor="#0071CE"
      isLoading={isLoading}
    />
  );
}
