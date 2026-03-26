import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";
import { ProductCarousel } from "./ProductCarousel";

const MOCK_AUCTIONS = [
  { id: "a1", title: "2022 BMW 3 Series Low Mileage Full Service", price: 95000, currency: "AED", type: "auction", bid_count: 12, ends_at: new Date(Date.now() + 7200000).toISOString(), city: "Dubai", condition: "like-new" },
  { id: "a2", title: "iPhone 15 Pro Max 256GB Factory Sealed", price: 3800, currency: "AED", type: "auction", bid_count: 28, ends_at: new Date(Date.now() + 1800000).toISOString(), city: "Abu Dhabi", condition: "new" },
  { id: "a3", title: "Rolex Submariner 2023 Unworn with Card", price: 42000, currency: "AED", type: "auction", bid_count: 45, ends_at: new Date(Date.now() + 3600000).toISOString(), city: "Kuwait City", condition: "new" },
  { id: "a4", title: "DJI Mavic 3 Pro Fly More Combo", price: 6500, currency: "AED", type: "auction", bid_count: 8, ends_at: new Date(Date.now() + 86400000).toISOString(), city: "Riyadh", condition: "new" },
  { id: "a5", title: "Hermès Birkin 30 Togo Leather Gold", price: 85000, currency: "AED", type: "auction", bid_count: 19, ends_at: new Date(Date.now() + 5400000).toISOString(), city: "Doha", condition: "like-new" },
  { id: "a6", title: "Ferrari 488 GTB 2019 — 12,000km", price: 680000, currency: "AED", type: "auction", bid_count: 7, ends_at: new Date(Date.now() + 10800000).toISOString(), city: "Dubai", condition: "good" },
  { id: "a7", title: "Apple Vision Pro 256GB + Travel Case", price: 14000, currency: "AED", type: "auction", bid_count: 33, ends_at: new Date(Date.now() + 2700000).toISOString(), city: "Riyadh", condition: "new" },
  { id: "a8", title: "Patek Philippe Nautilus 5711 Steel", price: 320000, currency: "AED", type: "auction", bid_count: 22, ends_at: new Date(Date.now() + 14400000).toISOString(), city: "Manama", condition: "like-new" },
];

export function LiveAuctions() {
  const { data: auctions, isLoading } = useQuery({
    queryKey: ["auctions", "active"],
    queryFn: () => api.get("/auctions?per_page=12&status=active").then((r) => r.data.data),
    retry: false,
  });

  const display = auctions?.length ? auctions : MOCK_AUCTIONS;

  return (
    <ProductCarousel
      title="Live Auctions"
      icon="🔴"
      listings={display}
      viewAllHref="/auctions"
      badge="LIVE"
      badgeColor="#EF4444"
      isLoading={isLoading}
    />
  );
}
