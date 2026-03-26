import { useParams, useLocation, Link } from "wouter";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import api from "@/lib/api";
import { formatPrice } from "@/lib/utils";
import { getCategorySchema, formatFieldValue } from "@/lib/categoryFields";
import { CountdownTimer } from "@/components/ui/CountdownTimer";
import { useAuthStore } from "@/store/auth";
import { Heart, Star, MessageCircle, Share2, ChevronLeft, TrendingDown, Layers, Trophy, Store } from "lucide-react";
import { getAuctionType, AUCTION_TYPE_BADGE } from "@/lib/auctionTypes";
import { SimilarListings } from "@/components/listings/SimilarListings";
import { ALL_LISTINGS } from "@/lib/recommendations";
import { ChatPanel } from "@/components/chat/ChatPanel";

const MOCK_DUTCH_EXTRA = {
  auction_type: "dutch",
  clearing_price: 5800,
  total_slots: 10,
  slots_won: 3,
  winners: [
    { id: "w1", name: "Ahmed K.", bid: 6200, avatar: "A" },
    { id: "w2", name: "Sara M.", bid: 6100, avatar: "S" },
    { id: "w3", name: "John D.", bid: 5900, avatar: "J" },
  ],
};

const MOCK_REVERSE_EXTRA = {
  auction_type: "reverse",
  lowest_offer: 38000,
  offers: [
    { id: "o1", vendor: "TechFix LLC", amount: 38000 },
    { id: "o2", vendor: "QuickServe Co.", amount: 41500 },
    { id: "o3", vendor: "ProBuild Ltd.", amount: 44000 },
    { id: "o4", vendor: "FastTools ME", amount: 47800 },
  ],
};

function AuctionTypeBadge({ auctionType }: { auctionType: string }) {
  const badge = AUCTION_TYPE_BADGE[auctionType as keyof typeof AUCTION_TYPE_BADGE] || AUCTION_TYPE_BADGE.standard;
  return (
    <span className={`text-xs font-bold px-3 py-1 rounded-full ${badge.className}`}>
      {badge.label} Auction
    </span>
  );
}

function DutchAuctionPanel({ listing, auctionId, onBid, bidAmount, setBidAmount, bidMessage, setBidMessage, isPending, isAuthenticated, navigate }: any) {
  const clearingPrice = listing.clearing_price ?? MOCK_DUTCH_EXTRA.clearing_price;
  const totalSlots = listing.total_slots ?? MOCK_DUTCH_EXTRA.total_slots;
  const slotsWon = listing.slots_won ?? MOCK_DUTCH_EXTRA.slots_won;
  const slotsRemaining = totalSlots - slotsWon;
  const currency = listing.currency || "AED";

  const { data: winnersData } = useQuery({
    queryKey: ["auction-winners", auctionId],
    queryFn: () =>
      api.get(`/auctions/${auctionId}/winners`).then((r) => r.data.data),
    refetchInterval: 10000,
    retry: false,
  });

  const winners = winnersData ?? listing.winners ?? MOCK_DUTCH_EXTRA.winners;

  return (
    <div className="space-y-5">
      <div className="bg-purple-50 border border-purple-200 rounded-xl p-4">
        <p className="text-xs text-purple-500 font-semibold uppercase tracking-wider">Clearing Price</p>
        <p className="text-3xl font-extrabold text-purple-700 mt-1">{formatPrice(clearingPrice, currency)}</p>
        <p className="text-xs text-purple-400 mt-1">All winners pay this price</p>
      </div>

      <div className="bg-gray-50 rounded-xl p-4">
        <p className="text-xs text-gray-500 font-semibold uppercase tracking-wider mb-2 flex items-center gap-1.5">
          <Layers size={13} /> Slots Available
        </p>
        <div className="flex items-center gap-3">
          <div className="flex-1 bg-gray-200 rounded-full h-2.5">
            <div
              className="bg-purple-500 h-2.5 rounded-full transition-all"
              style={{ width: `${(slotsWon / totalSlots) * 100}%` }}
            />
          </div>
          <span className="text-sm font-bold text-gray-700 whitespace-nowrap">
            {slotsRemaining} of {totalSlots} remaining
          </span>
        </div>
        <p className="text-xs text-gray-400 mt-1">{slotsWon} slots won so far</p>
      </div>

      <div className="space-y-3">
        <p className="text-xs text-gray-500 font-semibold uppercase tracking-wider flex items-center gap-1.5">
          <Trophy size={13} /> Current Winners (Top {totalSlots})
        </p>
        {winners.length === 0 ? (
          <p className="text-sm text-gray-400 italic">No bids yet — be the first!</p>
        ) : (
          <div className="space-y-2">
            {winners.map((w: any, i: number) => (
              <div key={w.id} className="flex items-center gap-3 bg-purple-50 rounded-lg px-3 py-2">
                <span className="w-5 h-5 rounded-full bg-purple-200 text-purple-700 text-xs font-bold flex items-center justify-center shrink-0">
                  {i + 1}
                </span>
                <span className="text-sm text-gray-700 font-medium flex-1">{w.name}</span>
                <span className="text-sm font-bold text-purple-700">{formatPrice(w.bid, currency)}</span>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="flex gap-2">
        <input
          type="number"
          value={bidAmount}
          onChange={(e) => { setBidAmount(e.target.value); setBidMessage(""); }}
          placeholder={`Min to enter top-${totalSlots}: ${formatPrice(clearingPrice, currency)}`}
          className="flex-1 border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-purple-500"
        />
        <button
          onClick={onBid}
          disabled={isPending}
          className="bg-purple-600 hover:bg-purple-700 text-white font-bold px-5 py-3 rounded-xl transition-colors disabled:opacity-60 whitespace-nowrap"
        >
          {isPending ? "Placing..." : "Place Offer"}
        </button>
      </div>
      {bidMessage && (
        <p className={`text-sm ${bidMessage.includes("success") ? "text-green-600" : "text-red-500"}`}>
          {bidMessage}
        </p>
      )}
    </div>
  );
}

function ReverseAuctionPanel({ listing, onBid, bidAmount, setBidAmount, bidMessage, setBidMessage, isPending }: any) {
  const lowestOffer = listing.lowest_offer ?? MOCK_REVERSE_EXTRA.lowest_offer;
  const offers = listing.offers ?? MOCK_REVERSE_EXTRA.offers;
  const currency = listing.currency || "AED";

  return (
    <div className="space-y-5">
      <div className="bg-orange-50 border border-orange-200 rounded-xl p-4">
        <p className="text-xs text-orange-500 font-semibold uppercase tracking-wider">Lowest Offer</p>
        <p className="text-3xl font-extrabold text-orange-600 mt-1">{formatPrice(lowestOffer, currency)}</p>
        <p className="text-xs text-orange-400 mt-1 flex items-center gap-1">
          <TrendingDown size={12} /> Vendors compete with lower prices
        </p>
      </div>

      <div className="bg-orange-50 border border-orange-100 rounded-xl p-3">
        <p className="text-xs font-semibold text-orange-600 mb-1">About this auction</p>
        <p className="text-xs text-gray-600 leading-relaxed">
          The <strong>Buyer</strong> has posted a requirement. <strong>Sellers / Vendors</strong> submit
          offers and compete to win by offering the lowest price.
        </p>
      </div>

      <div className="space-y-2">
        <p className="text-xs text-gray-500 font-semibold uppercase tracking-wider">All Offers (Ascending)</p>
        {offers.length === 0 ? (
          <p className="text-sm text-gray-400 italic">No offers yet — be the first vendor!</p>
        ) : (
          [...offers]
            .sort((a: any, b: any) => a.amount - b.amount)
            .map((o: any, i: number) => (
              <div
                key={o.id}
                className={`flex items-center gap-3 rounded-lg px-3 py-2 ${i === 0 ? "bg-green-50 border border-green-200" : "bg-gray-50"}`}
              >
                {i === 0 && <span className="text-green-600 text-xs font-bold">★ LOWEST</span>}
                <span className="text-sm text-gray-700 font-medium flex-1">{o.vendor}</span>
                <span className={`text-sm font-bold ${i === 0 ? "text-green-700" : "text-gray-700"}`}>
                  {formatPrice(o.amount, currency)}
                </span>
              </div>
            ))
        )}
      </div>

      <div className="flex gap-2">
        <input
          type="number"
          value={bidAmount}
          onChange={(e) => { setBidAmount(e.target.value); setBidMessage(""); }}
          placeholder={`Must be below ${formatPrice(lowestOffer, currency)}`}
          className="flex-1 border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-orange-500"
        />
        <button
          onClick={() => onBid(lowestOffer)}
          disabled={isPending}
          className="bg-orange-500 hover:bg-orange-600 text-white font-bold px-5 py-3 rounded-xl transition-colors disabled:opacity-60 whitespace-nowrap"
        >
          {isPending ? "Submitting..." : "Submit Offer"}
        </button>
      </div>
      {bidMessage && (
        <p className={`text-sm ${bidMessage.includes("success") ? "text-green-600" : "text-red-500"}`}>
          {bidMessage}
        </p>
      )}
    </div>
  );
}

export default function ListingDetailPage() {
  const params = useParams<{ id: string }>();
  const id = params.id;
  const [, navigate] = useLocation();
  const { isAuthenticated } = useAuthStore();
  const qc = useQueryClient();
  const [activeImage, setActiveImage] = useState(0);
  const [bidAmount, setBidAmount] = useState("");
  const [bidMessage, setBidMessage] = useState("");
  const [chatOpen, setChatOpen] = useState(false);

  const { data: apiListing, isLoading, error } = useQuery({
    queryKey: ["listing", id],
    queryFn: () => api.get(`/listings/${id}`, { timeout: 4000 }).then((r) => r.data.data),
    retry: 0,
    staleTime: 30_000,
  });

  // Mock fallback for demo listing IDs (lst_001 – lst_020)
  const mockFallbackListing = (() => {
    if (apiListing) return null;
    const mockItem = ALL_LISTINGS.find(l => l.id === id);
    if (!mockItem) return null;
    return {
      id: mockItem.id,
      title: mockItem.title,
      description: `Premium ${mockItem.category.toLowerCase()} listing available in ${mockItem.location}. In ${mockItem.condition.toLowerCase()} condition. Contact the seller for more details or to arrange a viewing.`,
      price: mockItem.price,
      currency: mockItem.currency,
      category: { slug: mockItem.category.toLowerCase().replace(/\s+/g, "-"), name: mockItem.category },
      location: { city: mockItem.location.split(",")[0].trim(), country: "UAE" },
      condition: mockItem.condition,
      images: [{ url: mockItem.image }],
      type: "fixed",
      is_auction: false,
      seller: { id: "seller_demo", name: mockItem.seller, rating: mockItem.rating, verified: true },
      attributes: {},
      created_at: mockItem.created_at,
      views: 247,
      saves: 38,
    };
  })();

  const listing = apiListing || mockFallbackListing;

  const bidMutation = useMutation({
    mutationFn: (amount: number) =>
      api.post(`/listings/${id}/bid`, { amount }).then((r) => r.data),
    onSuccess: () => {
      setBidMessage("Bid placed successfully! 🎉");
      qc.invalidateQueries({ queryKey: ["listing", id] });
    },
    onError: (err: any) => {
      setBidMessage(err?.response?.data?.message || "Failed to place bid.");
    },
  });

  const handleBid = (currentLowestOffer?: number) => {
    if (!isAuthenticated) {
      navigate("/login");
      return;
    }
    const amount = Number(bidAmount);
    if (!amount || amount <= 0) {
      setBidMessage("Please enter a valid amount.");
      return;
    }
    if (auctionType === "reverse" && currentLowestOffer != null && amount >= currentLowestOffer) {
      setBidMessage(`Your offer must be lower than the current lowest (${formatPrice(currentLowestOffer, listing?.currency || "AED")}).`);
      return;
    }
    bidMutation.mutate(amount);
  };

  if (isLoading) {
    return (
      <div className="max-w-6xl mx-auto px-4 py-10">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-10">
          <div className="h-96 bg-gray-100 rounded-2xl animate-pulse" />
          <div className="space-y-4">
            <div className="h-8 bg-gray-100 rounded animate-pulse" />
            <div className="h-12 w-1/2 bg-gray-100 rounded animate-pulse" />
            <div className="h-32 bg-gray-100 rounded animate-pulse" />
          </div>
        </div>
      </div>
    );
  }

  if (!listing) {
    return (
      <div className="text-center py-20 text-gray-400">
        <p className="text-5xl mb-4">😕</p>
        <p className="text-lg font-semibold">Listing not found</p>
        <button onClick={() => navigate("/listings")} className="mt-4 text-[#0071CE] hover:underline text-sm">
          ← Back to listings
        </button>
      </div>
    );
  }

  const images = listing.images?.length
    ? listing.images.map((img: any) => img.url || img)
    : [`https://picsum.photos/seed/${id}/600/400`];

  const isAuction = listing.type === "auction" || listing.is_auction;
  const auctionType = getAuctionType(listing);
  const price = isAuction
    ? listing.current_bid ?? listing.start_price ?? 0
    : listing.price ?? 0;

  const categorySlug = listing.category?.slug;
  const schema = getCategorySchema(categorySlug);
  const attributes: Record<string, unknown> = listing.attributes ?? {};

  const specRows = schema
    ? schema.fields
        .map((field) => {
          const raw = attributes[field.name];
          const formatted = formatFieldValue(field, raw);
          if (!formatted) return null;
          return { label: field.label, value: formatted };
        })
        .filter((r): r is { label: string; value: string } => r !== null)
    : [];

  const priceLabel = auctionType === "dutch"
    ? "Clearing Price"
    : auctionType === "reverse"
    ? "Lowest Offer"
    : "Current Bid";

  const displayPrice = auctionType === "dutch"
    ? (listing.clearing_price ?? price)
    : auctionType === "reverse"
    ? (listing.lowest_offer ?? price)
    : price;

  const sellerLabel = auctionType === "reverse" ? "Buyer" : "Seller";

  return (
    <div className="max-w-6xl mx-auto px-4 py-8">
      <button
        onClick={() => navigate("/listings")}
        className="flex items-center gap-1.5 text-gray-500 hover:text-[#0071CE] text-sm mb-6 transition-colors"
      >
        <ChevronLeft size={16} /> Back to listings
      </button>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-10">
        <div>
          <div className="rounded-2xl overflow-hidden bg-gray-50 mb-3 shadow-sm">
            <img
              src={images[activeImage]}
              alt={listing.title}
              className="w-full h-96 object-cover"
              onError={(e) => {
                (e.target as HTMLImageElement).src = `https://picsum.photos/seed/${id}/600/400`;
              }}
            />
          </div>
          {images.length > 1 && (
            <div className="flex gap-2 overflow-x-auto">
              {images.map((img: string, i: number) => (
                <button
                  key={i}
                  onClick={() => setActiveImage(i)}
                  className={`shrink-0 rounded-lg overflow-hidden border-2 transition-colors ${
                    activeImage === i ? "border-[#0071CE]" : "border-transparent"
                  }`}
                >
                  <img src={img} alt={`img-${i}`} className="w-16 h-16 object-cover" />
                </button>
              ))}
            </div>
          )}
        </div>

        <div>
          <div className="flex items-start justify-between gap-4">
            <div>
              <div className="flex items-center gap-2 mb-2">
                {isAuction && <AuctionTypeBadge auctionType={auctionType} />}
              </div>
              <h1 className="text-2xl font-bold text-gray-900 leading-snug">{listing.title}</h1>
            </div>
            <div className="flex gap-2 shrink-0">
              <button className="p-2 border rounded-xl hover:bg-gray-50 transition-colors" title="Add to watchlist">
                <Heart size={18} className="text-gray-400 hover:text-red-500" />
              </button>
              <button className="p-2 border rounded-xl hover:bg-gray-50 transition-colors" title="Share">
                <Share2 size={18} className="text-gray-400" />
              </button>
            </div>
          </div>

          {isAuction && (auctionType === "standard" || auctionType === "buy_now_only") && (
            <div className="mt-3 flex items-baseline gap-3">
              <p className="text-3xl font-extrabold text-[#0071CE]">
                {formatPrice(displayPrice, listing.currency || "AED")}
              </p>
              <span className="text-sm text-gray-500">{priceLabel}</span>
            </div>
          )}

          {isAuction && listing.ends_at && (
            <div className="mt-2 flex items-center gap-2">
              <span className="text-sm text-gray-500">Ends in:</span>
              <CountdownTimer endsAt={listing.ends_at} />
            </div>
          )}

          <div className="flex flex-wrap gap-3 mt-4">
            {listing.condition && (
              <span className="bg-blue-50 text-blue-700 text-xs font-semibold px-3 py-1.5 rounded-full capitalize">
                {listing.condition}
              </span>
            )}
            {listing.city && (
              <span className="bg-gray-100 text-gray-600 text-xs font-medium px-3 py-1.5 rounded-full">
                📍 {listing.city}
              </span>
            )}
            {listing.category?.name && (
              <span className="bg-gray-100 text-gray-600 text-xs font-medium px-3 py-1.5 rounded-full">
                {listing.category.name}
              </span>
            )}
          </div>

          <div className="mt-5">
            <h3 className="font-semibold text-gray-800 mb-2 text-sm">Description</h3>
            <p className="text-sm text-gray-600 leading-relaxed whitespace-pre-wrap">
              {listing.description || "No description provided."}
            </p>
          </div>

          {specRows.length > 0 && (
            <div className="mt-5 bg-gray-50 rounded-xl p-4">
              <h3 className="font-semibold text-gray-800 mb-3 text-sm">Specifications</h3>
              <div className="grid grid-cols-2 gap-x-4 gap-y-2">
                {specRows.map((row) => (
                  <div key={row.label}>
                    <p className="text-xs text-gray-400 font-medium uppercase tracking-wide">{row.label}</p>
                    <p className="text-sm text-gray-800 font-semibold mt-0.5">{row.value}</p>
                  </div>
                ))}
              </div>
            </div>
          )}

          {listing.seller && (
            <div className="mt-5 p-4 bg-gray-50 rounded-xl">
              <div className="flex items-center justify-between">
                <Link href={`/sellers/${listing.seller.id}`} className="flex items-center gap-3 group">
                  <div className="w-10 h-10 rounded-full bg-[#0071CE] flex items-center justify-center text-white font-bold">
                    {listing.seller.name?.[0] || "S"}
                  </div>
                  <div>
                    <p className="font-semibold text-sm text-gray-800 group-hover:text-[#0071CE] transition-colors">
                      {listing.seller.name}
                    </p>
                    <p className="text-xs text-gray-400">{sellerLabel}</p>
                    {listing.seller.rating && (
                      <p className="text-xs text-gray-500 flex items-center gap-1">
                        <Star size={11} fill="#FFC220" className="text-[#FFC220]" />
                        {listing.seller.rating.toFixed(1)} · Member
                      </p>
                    )}
                  </div>
                </Link>
                <button
                  onClick={() => { if (!isAuthenticated) { navigate("/login"); return; } setChatOpen(true); }}
                  className="flex items-center gap-1.5 text-sm text-[#0071CE] font-semibold hover:underline"
                >
                  <MessageCircle size={15} /> Message
                </button>
              </div>
              <Link
                href={`/sellers/${listing.seller.id}`}
                className="mt-3 w-full flex items-center justify-center gap-1.5 border border-gray-200 rounded-lg py-2 text-xs text-gray-600 hover:bg-white hover:border-[#0071CE] hover:text-[#0071CE] transition-colors"
              >
                <Store size={13} /> View Storefront
              </Link>
            </div>
          )}

          <div className="mt-6 space-y-3">
            {isAuction ? (
              auctionType === "dutch" ? (
                <DutchAuctionPanel
                  listing={listing}
                  auctionId={listing.auction_id || id}
                  onBid={handleBid}
                  bidAmount={bidAmount}
                  setBidAmount={setBidAmount}
                  bidMessage={bidMessage}
                  setBidMessage={setBidMessage}
                  isPending={bidMutation.isPending}
                  isAuthenticated={isAuthenticated}
                  navigate={navigate}
                />
              ) : auctionType === "reverse" ? (
                <ReverseAuctionPanel
                  listing={listing}
                  onBid={handleBid}
                  bidAmount={bidAmount}
                  setBidAmount={setBidAmount}
                  bidMessage={bidMessage}
                  setBidMessage={setBidMessage}
                  isPending={bidMutation.isPending}
                />
              ) : (
                <>
                  <div className="flex gap-2">
                    <input
                      type="number"
                      value={bidAmount}
                      onChange={(e) => {
                        setBidAmount(e.target.value);
                        setBidMessage("");
                      }}
                      placeholder={`Min: ${formatPrice(Number(price) + 100, listing.currency || "AED")}`}
                      className="flex-1 border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE]"
                    />
                    <button
                      onClick={() => handleBid()}
                      disabled={bidMutation.isPending}
                      className="bg-red-500 hover:bg-red-600 text-white font-bold px-5 py-3 rounded-xl transition-colors disabled:opacity-60"
                    >
                      {bidMutation.isPending ? "Bidding..." : "🔨 Bid"}
                    </button>
                  </div>
                  {bidMessage && (
                    <p className={`text-sm ${bidMessage.includes("success") ? "text-green-600" : "text-red-500"}`}>
                      {bidMessage}
                    </p>
                  )}
                </>
              )
            ) : (
              <button
                onClick={() => {
                  if (!isAuthenticated) {
                    navigate("/login");
                    return;
                  }
                  const sellerID = listing.seller?.id || "";
                  const params = new URLSearchParams({
                    listing_id: id || "",
                    seller_id: sellerID,
                    amount: String(price),
                    currency: listing.currency || "AED",
                    description: listing.title || "Purchase",
                  });
                  navigate(`/checkout?${params.toString()}`);
                }}
                className="w-full bg-[#0071CE] hover:bg-[#005BA1] text-white font-bold py-3.5 rounded-xl transition-colors text-sm"
              >
                Buy Now · {formatPrice(price, listing.currency || "AED")}
              </button>
            )}

            <button
              onClick={() => { if (!isAuthenticated) { navigate("/login"); return; } setChatOpen(true); }}
              className="w-full border-2 border-[#0071CE] text-[#0071CE] font-bold py-3.5 rounded-xl hover:bg-blue-50 transition-colors text-sm flex items-center justify-center gap-2"
            >
              <MessageCircle size={16} /> Message {sellerLabel}
            </button>
          </div>
        </div>
      </div>

      {/* ── Similar Listings (AI Recommendation Engine) ────────────────────── */}
      {(() => {
        // Find in local catalog or build from current listing data
        const catalogItem = ALL_LISTINGS.find(l => l.id === id);
        const categoryStr = typeof listing.category === "string"
          ? listing.category
          : listing.category?.name || "Other";
        const locationStr = typeof listing.location === "string"
          ? listing.location
          : listing.location?.city
          ? `${listing.location.city}, UAE`
          : "Dubai, UAE";
        const forRec = catalogItem || {
          id: id || "unknown",
          title: listing.title || "Listing",
          price: typeof price === "number" ? price : Number(price) || 0,
          currency: listing.currency || "AED",
          category: categoryStr,
          location: locationStr,
          condition: listing.condition || "Good",
          image: images[0] || `https://picsum.photos/seed/${id}/400/300`,
          seller: listing.seller?.name || "Seller",
          rating: listing.seller?.rating || 4.5,
          created_at: listing.created_at || new Date().toISOString(),
        };
        return <SimilarListings listing={forRec} />;
      })()}

      {chatOpen && listing.seller && (
        <ChatPanel
          sellerId={listing.seller.id}
          sellerName={listing.seller.name || "Seller"}
          listingId={id}
          onClose={() => setChatOpen(false)}
        />
      )}
    </div>
  );
}
