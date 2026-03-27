import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import api from "@/lib/api";
import { AuctionCard } from "@/components/listings/AuctionCard";
import { LoadingGrid } from "@/components/ui/LoadingGrid";
import { getAuctionType } from "@/lib/auctionTypes";
import type { AuctionType } from "@/lib/auctionTypes";

const MOCK_AUCTIONS = [
  { id: "a1", title: "2022 BMW 3 Series", auction_type: "standard", current_bid: 95000, currency: "AED", bid_count: 12, ends_at: new Date(Date.now() + 7200000).toISOString(), listing_id: "1" },
  { id: "a2", title: "iPhone 15 Pro Max Sealed", auction_type: "standard", current_bid: 3800, currency: "AED", bid_count: 28, ends_at: new Date(Date.now() + 1800000).toISOString(), listing_id: "2" },
  { id: "a3", title: "Rolex Submariner 2023", auction_type: "standard", current_bid: 42000, currency: "AED", bid_count: 45, ends_at: new Date(Date.now() + 3600000).toISOString(), listing_id: "3" },
  { id: "a4", title: "DJI Mavic 3 Pro — Bulk Lot (10 units)", auction_type: "dutch", clearing_price: 5800, total_slots: 10, slots_won: 3, currency: "AED", bid_count: 18, ends_at: new Date(Date.now() + 86400000).toISOString(), listing_id: "4" },
  { id: "a5", title: "Louis Vuitton Trunk Vintage", auction_type: "standard", current_bid: 28000, currency: "AED", bid_count: 32, ends_at: new Date(Date.now() + 5400000).toISOString(), listing_id: "5" },
  { id: "a6", title: "Office Renovation Services (Vendor Bid)", auction_type: "reverse", lowest_offer: 38000, current_bid: 38000, currency: "AED", bid_count: 6, ends_at: new Date(Date.now() + 172800000).toISOString(), listing_id: "6" },
  { id: "a7", title: "Patek Philippe Nautilus 5711", auction_type: "standard", current_bid: 380000, currency: "AED", bid_count: 60, ends_at: new Date(Date.now() + 900000).toISOString(), listing_id: "7" },
  { id: "a8", title: "Dubai Marina 3BR Penthouse", auction_type: "standard", current_bid: 8500000, currency: "AED", bid_count: 7, ends_at: new Date(Date.now() + 259200000).toISOString(), listing_id: "8" },
  { id: "a9", title: "Laptop Fleet (50 units) — Dutch Auction", auction_type: "dutch", clearing_price: 2900, total_slots: 50, slots_won: 12, currency: "AED", bid_count: 31, ends_at: new Date(Date.now() + 43200000).toISOString(), listing_id: "9" },
  { id: "a10", title: "IT Support Contract — Vendor Selection", auction_type: "reverse", lowest_offer: 12000, current_bid: 12000, currency: "AED", bid_count: 9, ends_at: new Date(Date.now() + 64800000).toISOString(), listing_id: "10" },
];

const STATUS_TABS = [
  { label: "🔴 Live", value: "active" },
  { label: "⏰ Ending Soon", value: "ending_soon" },
  { label: "🕐 Upcoming", value: "upcoming" },
];

const TYPE_FILTERS: { label: string; value: AuctionType | "all" }[] = [
  { label: "All Types", value: "all" },
  { label: "Standard", value: "standard" },
  { label: "Dutch", value: "dutch" },
  { label: "Reverse", value: "reverse" },
];

const TYPE_BADGE_COLORS: Record<string, string> = {
  all: "bg-gray-100 text-gray-700 border-gray-300",
  standard: "bg-blue-100 text-blue-700 border-blue-300",
  dutch: "bg-purple-100 text-purple-700 border-purple-300",
  reverse: "bg-orange-100 text-orange-700 border-orange-300",
};

const TYPE_BADGE_ACTIVE: Record<string, string> = {
  all: "bg-gray-700 text-white border-gray-700",
  standard: "bg-blue-600 text-white border-blue-600",
  dutch: "bg-purple-600 text-white border-purple-600",
  reverse: "bg-orange-500 text-white border-orange-500",
};

export default function AuctionsPage() {
  const [activeStatus, setActiveStatus] = useState("active");
  const [activeType, setActiveType] = useState<AuctionType | "all">("all");

  const { data: auctions, isLoading } = useQuery({
    queryKey: ["auctions", activeStatus],
    queryFn: () => api.get(`/auctions?status=${activeStatus}&per_page=20`).then((r) => r.data.data),
    retry: false,
  });

  const rawAuctions = auctions?.length ? auctions : MOCK_AUCTIONS;

  const displayAuctions = activeType === "all"
    ? rawAuctions
    : rawAuctions.filter((a: any) => getAuctionType(a) === activeType);

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      <div className="bg-gradient-to-r from-[#0071CE] to-[#003f75] rounded-2xl p-8 text-white mb-8">
        <h1 className="text-3xl font-extrabold flex items-center gap-3">
          🔨 Live Auctions
          <span className="bg-red-500 text-white text-xs font-bold px-2 py-1 rounded-full animate-pulse">LIVE</span>
        </h1>
        <p className="text-blue-100 mt-2 text-sm">
          Bid in real-time on thousands of items across the GCC. Transparent. Secure. Instant.
        </p>
        <div className="flex gap-6 mt-6">
          {[
            { label: "Active Auctions", value: "18,432" },
            { label: "Bids Today", value: "284K" },
            { label: "Items Sold", value: "2.1M" },
          ].map((s) => (
            <div key={s.label}>
              <p className="text-2xl font-extrabold text-[#FFC220]">{s.value}</p>
              <p className="text-xs text-blue-200">{s.label}</p>
            </div>
          ))}
        </div>
      </div>

      <div className="flex gap-3 mb-4 border-b border-gray-200">
        {STATUS_TABS.map((tab) => (
          <button
            key={tab.value}
            onClick={() => setActiveStatus(tab.value)}
            className={`px-4 py-2.5 text-sm font-semibold border-b-2 transition-colors ${
              activeStatus === tab.value
                ? "border-[#0071CE] text-[#0071CE]"
                : "border-transparent text-gray-500 hover:text-gray-800"
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="flex gap-2 mb-6 flex-wrap">
        {TYPE_FILTERS.map((f) => (
          <button
            key={f.value}
            onClick={() => setActiveType(f.value)}
            className={`px-3 py-1.5 text-xs font-semibold rounded-full border transition-colors ${
              activeType === f.value
                ? TYPE_BADGE_ACTIVE[f.value]
                : TYPE_BADGE_COLORS[f.value]
            }`}
          >
            {f.label}
          </button>
        ))}
      </div>

      {isLoading ? (
        <LoadingGrid count={8} />
      ) : displayAuctions.length === 0 ? (
        <div className="text-center py-20 text-gray-400">
          <p className="text-4xl mb-3">🔨</p>
          <p className="font-semibold text-lg">No auctions available</p>
        </div>
      ) : (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
          {displayAuctions.map((auction: any) => (
            <AuctionCard key={auction.id} auction={auction} />
          ))}
        </div>
      )}
    </div>
  );
}
