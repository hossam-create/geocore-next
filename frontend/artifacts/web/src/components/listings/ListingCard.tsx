import { useState } from "react";
import { Link } from "wouter";
import { CountdownTimer } from "@/components/ui/CountdownTimer";
import { formatPrice } from "@/lib/utils";
import { getKeyAttributePills } from "@/lib/categoryFields";
import { Heart, MapPin, ShoppingCart, Gavel } from "lucide-react";
import type { Listing } from "@/types/listing";

interface ListingCardProps {
  listing: Listing;
  onFavorite?: (id: string) => void;
}

const CONDITION_COLORS: Record<string, string> = {
  new: "text-green-700 bg-green-50 border-green-200",
  "like-new": "text-emerald-700 bg-emerald-50 border-emerald-200",
  good: "text-blue-700 bg-blue-50 border-blue-200",
  fair: "text-amber-700 bg-amber-50 border-amber-200",
  used: "text-gray-600 bg-gray-100 border-gray-200",
};
const CONDITION_LABELS: Record<string, string> = {
  new: "New",
  "like-new": "Like New",
  good: "Good",
  fair: "Fair",
  used: "Used",
};

export function ListingCard({ listing, onFavorite }: ListingCardProps) {
  const [saved, setSaved] = useState(false);
  const [added, setAdded] = useState(false);

  const isAuction = listing.type === "auction" || listing.is_auction || listing.isAuction;
  const price = isAuction
    ? listing.current_bid ?? listing.currentBid ?? listing.start_price ?? listing.startPrice ?? listing.price ?? 0
    : listing.price ?? 0;
  const imageUrl =
    listing.images?.[0]?.url ||
    listing.image_url ||
    `https://picsum.photos/seed/${listing.id}/300/300`;
  const location = listing.city || listing.location?.city || listing.location || "GCC";
  const endsAt = listing.ends_at || listing.auctionEndsAt;
  const condition = listing.condition;
  const bidCount = listing.bid_count ?? listing.bids_count ?? listing.bidCount;

  const categorySlug = listing.category?.slug;
  const attrPills = getKeyAttributePills(categorySlug, listing.attributes);

  const handleSave = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setSaved((s) => !s);
    onFavorite?.(listing.id);
  };

  const handleAdd = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setAdded(true);
    setTimeout(() => setAdded(false), 1800);
  };

  return (
    <Link href={`/listings/${listing.id}`}>
      <div className="bg-white rounded-2xl overflow-hidden shadow-sm hover:shadow-xl border border-gray-100 hover:border-[#0071CE]/20 transition-all duration-200 hover:-translate-y-0.5 cursor-pointer group h-full flex flex-col">

        {/* Image */}
        <div className="relative bg-gray-50 overflow-hidden" style={{ aspectRatio: "4/3" }}>
          <img
            src={imageUrl}
            alt={listing.title}
            className="w-full h-full object-cover group-hover:scale-[1.03] transition-transform duration-500"
            onError={(e) => {
              (e.target as HTMLImageElement).src = `https://picsum.photos/seed/${listing.id ?? "item"}/300/225`;
            }}
          />

          {/* Save heart */}
          <button
            onClick={handleSave}
            className={`absolute top-2.5 right-2.5 w-8 h-8 rounded-full flex items-center justify-center transition-all shadow-sm ${saved ? "bg-red-500 text-white" : "bg-white/90 text-gray-400 hover:text-red-500 hover:bg-white"}`}
          >
            <Heart size={15} fill={saved ? "currentColor" : "none"} />
          </button>

          {/* Featured badge */}
          {(listing.is_featured || listing.isFeatured) && (
            <span className="absolute top-2.5 left-2.5 bg-[#FFC220] text-gray-900 text-[10px] font-black px-2 py-0.5 rounded-full">
              ⭐ Featured
            </span>
          )}

          {/* Auction overlay */}
          {isAuction && endsAt && (
            <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/60 to-transparent px-3 py-2">
              <CountdownTimer endsAt={endsAt} compact />
            </div>
          )}
        </div>

        {/* Body */}
        <div className="p-3 flex flex-col flex-1 gap-1.5">

          {/* Type + condition */}
          <div className="flex items-center gap-1.5 flex-wrap">
            <span className={`text-[10px] font-black px-2 py-0.5 rounded-full text-white ${isAuction ? "bg-rose-500" : "bg-[#0071CE]"}`}>
              {isAuction ? "AUCTION" : "BUY NOW"}
            </span>
            {condition && CONDITION_LABELS[condition] && (
              <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${CONDITION_COLORS[condition] ?? "bg-gray-100 text-gray-500 border-gray-200"}`}>
                {CONDITION_LABELS[condition]}
              </span>
            )}
          </div>

          {/* Title */}
          <p className="text-sm text-gray-800 font-semibold line-clamp-2 leading-snug flex-1 group-hover:text-[#0071CE] transition-colors">
            {listing.title}
          </p>

          {/* Attribute pills */}
          {attrPills.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {attrPills.slice(0, 2).map((pill) => (
                <span key={pill} className="text-[10px] bg-gray-100 text-gray-500 px-2 py-0.5 rounded-full">
                  {pill}
                </span>
              ))}
            </div>
          )}

          {/* Price */}
          <div>
            <p className="text-lg font-black text-gray-900 leading-none">
              {formatPrice(price, listing.currency || "AED")}
            </p>
            {isAuction && (
              <p className="text-xs text-gray-400 mt-0.5">
                {bidCount != null ? `${bidCount} bid${bidCount !== 1 ? "s" : ""}` : "Starting bid"}
              </p>
            )}
          </div>

          {/* Location */}
          <p className="text-xs text-gray-400 flex items-center gap-1">
            <MapPin size={11} className="shrink-0" /> {location}
          </p>

          {/* CTA button */}
          <button
            onClick={handleAdd}
            className={`mt-1 w-full py-2 rounded-xl text-sm font-bold transition-all flex items-center justify-center gap-2 ${
              added
                ? "bg-green-500 text-white"
                : isAuction
                ? "bg-[#0071CE] hover:bg-[#005ea8] text-white"
                : "bg-[#FFC220] hover:bg-yellow-400 text-gray-900"
            }`}
          >
            {added ? (
              "✓ Added!"
            ) : isAuction ? (
              <><Gavel size={14} /> Place Bid</>
            ) : (
              <><ShoppingCart size={14} /> Add to Cart</>
            )}
          </button>
        </div>
      </div>
    </Link>
  );
}
