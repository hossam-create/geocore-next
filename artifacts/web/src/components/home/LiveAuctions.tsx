import { useQuery } from "@tanstack/react-query";
import { Link } from "wouter";
import api from "@/lib/api";
import { AuctionCard } from "@/components/listings/AuctionCard";
import { LoadingGrid } from "@/components/ui/LoadingGrid";

const MOCK_AUCTIONS = [
  { id: "a1", title: "2022 BMW 3 Series — Low Mileage", current_bid: 95000, currency: "AED", bid_count: 12, ends_at: new Date(Date.now() + 7200000).toISOString(), listing_id: "1" },
  { id: "a2", title: "iPhone 15 Pro Max 256GB — Factory Sealed", current_bid: 3800, currency: "AED", bid_count: 28, ends_at: new Date(Date.now() + 1800000).toISOString(), listing_id: "2" },
  { id: "a3", title: "Rolex Submariner 2023 — Unworn", current_bid: 42000, currency: "AED", bid_count: 45, ends_at: new Date(Date.now() + 3600000).toISOString(), listing_id: "3" },
  { id: "a4", title: "DJI Mavic 3 Pro Fly More Combo", current_bid: 6500, currency: "AED", bid_count: 8, ends_at: new Date(Date.now() + 86400000).toISOString(), listing_id: "4" },
];

export function LiveAuctions() {
  const { data: auctions, isLoading } = useQuery({
    queryKey: ["auctions", "active"],
    queryFn: () => api.get("/auctions?per_page=8&status=active").then((r) => r.data.data),
    retry: false,
  });

  const displayAuctions = auctions?.length ? auctions : MOCK_AUCTIONS;

  return (
    <section>
      <div className="flex items-center justify-between mb-5">
        <h2 className="text-2xl font-bold text-gray-900 flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-red-500 animate-ping inline-block" />
          Live Auctions
        </h2>
        <Link href="/auctions" className="text-[#0071CE] text-sm font-semibold hover:underline">
          See all →
        </Link>
      </div>
      {isLoading ? (
        <LoadingGrid count={4} />
      ) : (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {displayAuctions.slice(0, 8).map((auction: any) => (
            <AuctionCard key={auction.id} auction={auction} />
          ))}
        </div>
      )}
    </section>
  );
}
