import { useQuery } from "@tanstack/react-query";
import { useState, useEffect, useRef } from "react";
import api from "@/lib/api";
import { AuctionCard } from "@/components/listings/AuctionCard";
import { LoadingGrid } from "@/components/ui/LoadingGrid";
import { getAuctionType } from "@/lib/auctionTypes";
import type { AuctionType } from "@/lib/auctionTypes";
import type { Auction } from "@/lib/types";


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

interface LiveBidEvent {
  auctionId: string;
  bid: number;
  user: string;
  ts: number;
}

export default function AuctionsPage() {
  const [activeStatus, setActiveStatus] = useState("active");
  const [activeType, setActiveType] = useState<AuctionType | "all">("all");
  const [liveTicker, setLiveTicker] = useState<LiveBidEvent[]>([]);
  const [liveAuctionBids, setLiveAuctionBids] = useState<Record<string, number>>({});
  const wsMap = useRef<Map<string, WebSocket>>(new Map());

  const { data: auctions, isLoading } = useQuery<Auction[]>({
    queryKey: ["auctions", activeStatus],
    queryFn: () => api.get(`/auctions?status=${activeStatus}&per_page=20`).then((r) => r.data.data as Auction[]),
    retry: false,
  });

  const rawAuctions: Auction[] = auctions ?? [];

  const displayAuctions: Auction[] = activeType === "all"
    ? rawAuctions
    : rawAuctions.filter((a: Auction) => getAuctionType(a) === activeType);

  useEffect(() => {
    const activeAuctions = displayAuctions.slice(0, 20);
    const backoffMap = new Map<string, number>();
    const timerMap = new Map<string, ReturnType<typeof setTimeout>>();
    let destroyed = false;

    const connectAuction = (auctionId: string) => {
      if (destroyed) return;
      const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
      const ws = new WebSocket(`${proto}//${window.location.host}/ws/auctions/${auctionId}`);
      wsMap.current.set(auctionId, ws);

      ws.onopen = () => {
        backoffMap.set(auctionId, 1000);
      };

      ws.onmessage = (e) => {
        try {
          const msg = JSON.parse(e.data) as { bid: number; user: string };
          setLiveAuctionBids((prev) => ({ ...prev, [auctionId]: msg.bid }));
          setLiveTicker((prev) => [
            { auctionId, bid: msg.bid, user: msg.user, ts: Date.now() },
            ...prev.slice(0, 4),
          ]);
        } catch (err) {
          console.error("[WS] Failed to parse auction bid event:", err);
        }
      };

      ws.onclose = () => {
        wsMap.current.delete(auctionId);
        if (!destroyed) {
          const delay = backoffMap.get(auctionId) ?? 1000;
          const timer = setTimeout(() => {
            backoffMap.set(auctionId, Math.min(delay * 2, 30_000));
            connectAuction(auctionId);
          }, delay);
          timerMap.set(auctionId, timer);
        }
      };

      ws.onerror = () => ws.close();
    };

    for (const a of activeAuctions) {
      if (!wsMap.current.has(a.id)) {
        backoffMap.set(a.id, 1000);
        connectAuction(a.id);
      }
    }

    return () => {
      destroyed = true;
      timerMap.forEach((t) => clearTimeout(t));
      wsMap.current.forEach((ws) => ws.close());
      wsMap.current.clear();
    };
  }, [displayAuctions.map((a) => a.id).join(",")]);

  const enrichedAuctions: Auction[] = displayAuctions.map((a: Auction) => ({
    ...a,
    current_bid: liveAuctionBids[a.id] ?? a.current_bid,
  }));

  const liveCount = rawAuctions.length;
  const totalBids = rawAuctions.reduce((s: number, a: Auction) => s + (a.bid_count || 0), 0);

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
        {liveTicker.length > 0 && (
          <div className="mt-4 bg-white/10 rounded-xl px-4 py-2.5 text-sm flex items-center gap-2 overflow-hidden">
            <span className="text-yellow-300 font-bold shrink-0">⚡ LIVE:</span>
            <span className="text-white truncate">
              New bid of {liveTicker[0].bid.toLocaleString()} AED just placed
            </span>
          </div>
        )}
        <div className="flex gap-6 mt-6">
          {[
            { label: "Active Auctions", value: liveCount > 0 ? liveCount.toLocaleString() : "—" },
            { label: "Total Bids", value: totalBids > 0 ? totalBids.toLocaleString() : "—" },
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
      ) : enrichedAuctions.length === 0 ? (
        <div className="text-center py-20 text-gray-400">
          <p className="text-4xl mb-3">🔨</p>
          <p className="font-semibold text-lg">No auctions available</p>
        </div>
      ) : (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
          {enrichedAuctions.map((auction: Auction) => (
            <AuctionCard key={auction.id} auction={auction} />
          ))}
        </div>
      )}
    </div>
  );
}
