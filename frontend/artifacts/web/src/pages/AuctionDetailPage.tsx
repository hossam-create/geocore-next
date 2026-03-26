import { useEffect, useRef, useState } from "react";
import { useParams, useLocation } from "wouter";
import { useQuery, useMutation } from "@tanstack/react-query";
import api from "@/lib/api";
import { formatPrice, formatRelativeTime } from "@/lib/utils";
import { CountdownTimer } from "@/components/ui/CountdownTimer";
import { useAuthStore } from "@/store/auth";
import { ChevronLeft, Trophy, Users, Zap, TrendingUp } from "lucide-react";
import type { Auction, AuctionBid, ApiError } from "@/lib/types";

interface LiveBid {
  bid: number;
  user: string;
  ts: number;
}

interface FeedBid {
  id: string;
  amount: number;
  placed_at: string;
  user_id: string;
  isLive: boolean;
}

export default function AuctionDetailPage() {
  const { id } = useParams<{ id: string }>();
  const [, navigate] = useLocation();
  const { isAuthenticated, user } = useAuthStore();

  const [currentBid, setCurrentBid] = useState<number | null>(null);
  const [bidCount, setBidCount] = useState<number | null>(null);
  const [liveFeed, setLiveFeed] = useState<LiveBid[]>([]);
  const [bidAmount, setBidAmount] = useState("");
  const [bidMsg, setBidMsg] = useState("");
  const [wsConnected, setWsConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  const { data: auction, isLoading, isError } = useQuery<Auction>({
    queryKey: ["auction", id],
    queryFn: () => api.get(`/auctions/${id}`).then((r) => r.data.data as Auction),
    retry: false,
    staleTime: 10_000,
  });

  useEffect(() => {
    if (auction) {
      setCurrentBid((prev) => prev ?? auction.current_bid);
      setBidCount((prev) => prev ?? auction.bid_count);
    }
  }, [auction]);

  useEffect(() => {
    if (!id) return;

    let ws: WebSocket | null = null;
    let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
    let backoffMs = 1000;
    const maxBackoffMs = 30_000;
    let destroyed = false;

    const connect = () => {
      if (destroyed) return;
      const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
      ws = new WebSocket(`${proto}//${window.location.host}/ws/auctions/${id}`);
      wsRef.current = ws;

      ws.onopen = () => {
        setWsConnected(true);
        backoffMs = 1000;
      };

      ws.onmessage = (e) => {
        try {
          const msg = JSON.parse(e.data) as { bid: number; user: string };
          setCurrentBid(msg.bid);
          setBidCount((prev) => (prev ?? 0) + 1);
          setLiveFeed((prev) => [
            { bid: msg.bid, user: msg.user, ts: Date.now() },
            ...prev.slice(0, 19),
          ]);
        } catch (err) {
          console.error("[WS] Failed to parse bid event:", err);
        }
      };

      ws.onclose = () => {
        setWsConnected(false);
        if (!destroyed) {
          reconnectTimeout = setTimeout(() => {
            backoffMs = Math.min(backoffMs * 2, maxBackoffMs);
            connect();
          }, backoffMs);
        }
      };

      ws.onerror = () => {
        ws?.close();
      };
    };

    connect();

    return () => {
      destroyed = true;
      if (reconnectTimeout) clearTimeout(reconnectTimeout);
      ws?.close();
    };
  }, [id]);

  const bidMutation = useMutation({
    mutationFn: (amount: number) =>
      api.post(`/auctions/${id}/bid`, { amount }).then((r) => r.data),
    onSuccess: (data: { data?: { amount?: number } }) => {
      setBidMsg("Bid placed successfully!");
      setBidAmount("");
      setCurrentBid(data.data?.amount ?? Number(bidAmount));
    },
    onError: (err: ApiError) => {
      setBidMsg(err?.response?.data?.message ?? "Failed to place bid.");
    },
  });

  const handleBid = () => {
    if (!isAuthenticated) {
      navigate("/login?next=/auctions/" + id);
      return;
    }
    const amount = Number(bidAmount);
    if (!amount || amount <= 0) {
      setBidMsg("Please enter a valid amount.");
      return;
    }
    // Backend rule: bid must be strictly greater than current_bid;
    // for first bid (current_bid === 0), the floor is start_price - 0.01
    // so amount >= start_price is valid on the first bid.
    const activeBid = currentBid ?? auction?.current_bid ?? 0;
    const floor = activeBid > 0 ? activeBid : (auction?.start_price ?? 1) - 0.01;
    if (amount <= floor) {
      const displayFloor = activeBid > 0 ? activeBid : (auction?.start_price ?? 0);
      setBidMsg(`Bid must be at least ${formatPrice(displayFloor, auction?.currency ?? "AED")}`);
      return;
    }
    setBidMsg("");
    bidMutation.mutate(amount);
  };

  if (isLoading) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-20 text-center text-gray-400">
        <p className="text-4xl mb-3 animate-pulse">🔨</p>
        <p className="font-semibold text-lg">Loading auction…</p>
      </div>
    );
  }

  if (isError || !auction) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-20 text-center text-gray-400">
        <p className="text-4xl mb-3">⚠️</p>
        <p className="font-semibold text-lg">Auction not found</p>
        <button onClick={() => navigate("/auctions")} className="mt-4 text-sm text-[#0071CE] hover:underline">
          ← Back to Auctions
        </button>
      </div>
    );
  }

  const displayBid = currentBid ?? auction.current_bid ?? auction.start_price ?? 0;
  const displayBidCount = bidCount ?? auction.bid_count ?? 0;
  const currency = auction.currency || "AED";

  // Min next bid mirrors backend: must be strictly > current bid
  // Show start_price as the floor when no bids have been placed yet
  const minNext = displayBid > 0 ? displayBid : auction.start_price;

  const liveFeedBids: FeedBid[] = liveFeed.map((f) => ({
    id: `live-${f.ts}`,
    amount: f.bid,
    placed_at: new Date(f.ts).toISOString(),
    user_id: f.user,
    isLive: true,
  }));

  const historicBids: FeedBid[] = (auction.bids ?? []).map((b: AuctionBid) => ({
    ...b,
    isLive: false,
  }));

  const recentBids: FeedBid[] = [...liveFeedBids, ...historicBids].slice(0, 15);

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <button
        onClick={() => navigate("/auctions")}
        className="flex items-center gap-1.5 text-gray-500 hover:text-[#0071CE] text-sm mb-6 transition-colors"
      >
        <ChevronLeft size={16} /> Back to Auctions
      </button>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
        <div>
          <img
            src={`https://picsum.photos/seed/${id || "auction"}/600/400`}
            alt={auction.title || "Auction"}
            className="w-full h-72 object-cover rounded-2xl shadow-sm"
            onError={(e) => {
              (e.target as HTMLImageElement).src = `https://picsum.photos/seed/auction/600/400`;
            }}
          />

          <div className="mt-5 bg-white rounded-2xl shadow-sm p-5">
            <div className="flex items-center justify-between mb-3">
              <h2 className="font-bold text-gray-800 flex items-center gap-2">
                <TrendingUp size={16} className="text-[#0071CE]" /> Live Bid Feed
              </h2>
              <div className={`flex items-center gap-1.5 text-xs font-semibold px-2 py-1 rounded-full ${wsConnected ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-500"}`}>
                <div className={`w-1.5 h-1.5 rounded-full ${wsConnected ? "bg-green-500 animate-pulse" : "bg-gray-400"}`} />
                {wsConnected ? "Connected" : "Offline"}
              </div>
            </div>
            {recentBids.length === 0 ? (
              <p className="text-sm text-gray-400 italic text-center py-4">No bids yet — be the first!</p>
            ) : (
              <ul className="space-y-2 max-h-64 overflow-y-auto">
                {recentBids.map((bid) => (
                  <li
                    key={bid.id}
                    className={`flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition-colors ${bid.isLive ? "bg-green-50 border border-green-200" : "bg-gray-50"}`}
                  >
                    <div className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold shrink-0 ${bid.isLive ? "bg-green-500 text-white" : "bg-[#0071CE] text-white"}`}>
                      {bid.isLive ? <Zap size={12} /> : <Users size={12} />}
                    </div>
                    <div className="flex-1 min-w-0">
                      <span className="font-medium text-gray-700">
                        {formatPrice(bid.amount, currency)}
                      </span>
                      {bid.isLive && (
                        <span className="ml-2 text-xs text-green-600 font-semibold">NEW</span>
                      )}
                    </div>
                    <span className="text-xs text-gray-400 shrink-0">
                      {formatRelativeTime(bid.placed_at)}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>

        <div>
          <div className="bg-red-50 border border-red-100 rounded-xl px-3 py-1.5 inline-flex items-center gap-1.5 text-xs font-bold text-red-600 mb-3">
            <span className="w-1.5 h-1.5 rounded-full bg-red-500 animate-pulse" />
            LIVE AUCTION
          </div>

          <h1 className="text-2xl font-bold text-gray-900 leading-snug mb-4">
            {auction.title || auction.listing?.title || "Auction Item"}
          </h1>

          <div className="bg-[#0071CE] rounded-2xl p-5 text-white mb-5">
            <p className="text-blue-200 text-sm">Current Bid</p>
            <p className="text-4xl font-extrabold mt-1">{formatPrice(displayBid, currency)}</p>
            <div className="flex items-center justify-between mt-3">
              <p className="text-blue-200 text-xs flex items-center gap-1">
                <Users size={12} /> {displayBidCount} bids placed
              </p>
              {auction.ends_at && (
                <CountdownTimer endsAt={auction.ends_at} className="text-yellow-300" />
              )}
            </div>
          </div>

          {auction.status === "active" ? (
            <div className="space-y-3">
              <div className="bg-gray-50 rounded-xl p-3 text-xs text-gray-500">
                Minimum next bid: <span className="font-bold text-gray-800">{formatPrice(minNext, currency)}</span>
                <span className="ml-1 text-gray-400">(must be strictly higher)</span>
              </div>

              <div className="flex gap-2">
                <input
                  type="number"
                  value={bidAmount}
                  onChange={(e) => { setBidAmount(e.target.value); setBidMsg(""); }}
                  placeholder={`Min: ${formatPrice(minNext, currency)}`}
                  className="flex-1 border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
                />
                <button
                  onClick={handleBid}
                  disabled={bidMutation.isPending}
                  className="bg-red-500 hover:bg-red-600 text-white font-bold px-5 py-3 rounded-xl transition-colors disabled:opacity-60 flex items-center gap-2 whitespace-nowrap"
                >
                  <Trophy size={16} />
                  {bidMutation.isPending ? "Placing..." : "Place Bid"}
                </button>
              </div>

              {[
                minNext + 1,
                minNext + 500,
                minNext + 1000,
                minNext + 5000,
              ].map((quick) => (
                <button
                  key={quick}
                  onClick={() => setBidAmount(String(quick))}
                  className="mr-2 mb-1 px-3 py-1.5 text-xs font-semibold border border-[#0071CE] text-[#0071CE] rounded-full hover:bg-blue-50 transition-colors"
                >
                  {formatPrice(quick, currency)}
                </button>
              ))}

              {bidMsg && (
                <p className={`text-sm font-medium ${bidMsg.includes("success") ? "text-green-600" : "text-red-500"}`}>
                  {bidMsg}
                </p>
              )}

              {!isAuthenticated && (
                <p className="text-xs text-gray-400 text-center">
                  <a href="/login" className="text-[#0071CE] hover:underline font-semibold">Sign in</a> to place a bid
                </p>
              )}
            </div>
          ) : auction.status === "ended" && isAuthenticated && user?.id && auction.winner_id === user.id ? (
            <div className="bg-green-50 border border-green-200 rounded-2xl p-5 space-y-3">
              <div className="flex items-center gap-2 text-green-700 font-bold text-lg">
                <Trophy size={20} className="text-yellow-500" />
                Congratulations! You won this auction.
              </div>
              <p className="text-sm text-green-600">
                Winning bid: <span className="font-bold">{formatPrice(displayBid, currency)}</span>
              </p>
              <button
                onClick={() =>
                  navigate(
                    `/checkout?auction_id=${id}&amount=${displayBid}&currency=${currency}`
                  )
                }
                className="w-full bg-green-600 hover:bg-green-700 text-white font-bold py-3 rounded-xl transition-colors flex items-center justify-center gap-2"
              >
                <Trophy size={16} />
                Complete Payment
              </button>
              <p className="text-xs text-gray-400 text-center">
                Secure payment via Stripe. Your item will be held until payment is confirmed.
              </p>
            </div>
          ) : (
            <div className="bg-gray-100 rounded-xl p-4 text-center text-gray-500 text-sm font-semibold">
              {auction.status === "ended"
                ? "This auction has ended."
                : "Auction is not currently accepting bids."}
            </div>
          )}

          {auction.start_price && (
            <div className="mt-5 pt-4 border-t border-gray-100 grid grid-cols-2 gap-3 text-sm">
              <div>
                <p className="text-xs text-gray-400 uppercase tracking-wide">Starting Price</p>
                <p className="font-bold text-gray-800 mt-0.5">{formatPrice(auction.start_price, currency)}</p>
              </div>
              {auction.buy_now_price && (
                <div>
                  <p className="text-xs text-gray-400 uppercase tracking-wide">Buy Now</p>
                  <p className="font-bold text-green-600 mt-0.5">{formatPrice(auction.buy_now_price, currency)}</p>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
