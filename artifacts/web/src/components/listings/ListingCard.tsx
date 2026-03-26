import { Link } from "wouter";
import { CountdownTimer } from "@/components/ui/CountdownTimer";
import { formatPrice } from "@/lib/utils";
import { getKeyAttributePills } from "@/lib/categoryFields";
import { Heart } from "lucide-react";

interface ListingCardProps {
  listing: any;
  onFavorite?: (id: string) => void;
}

export function ListingCard({ listing, onFavorite }: ListingCardProps) {
  const isAuction = listing.type === "auction" || listing.is_auction || listing.isAuction;
  const price = isAuction
    ? listing.current_bid ?? listing.currentBid ?? listing.start_price ?? listing.startPrice ?? listing.price ?? 0
    : listing.price ?? 0;
  const imageUrl =
    listing.images?.[0]?.url ||
    listing.image_url ||
    `https://picsum.photos/seed/${listing.id}/300/200`;
  const location = listing.city || listing.location || "GCC";
  const endsAt = listing.ends_at || listing.auctionEndsAt;

  const categorySlug = listing.category?.slug;
  const attrPills = getKeyAttributePills(categorySlug, listing.attributes);

  return (
    <Link href={`/listings/${listing.id}`}>
      <div className="bg-white rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-all hover:-translate-y-0.5 cursor-pointer group h-full flex flex-col">
        <div className="relative">
          <img
            src={imageUrl}
            alt={listing.title}
            className="w-full h-44 object-cover group-hover:scale-[1.02] transition-transform duration-300"
            onError={(e) => {
              (e.target as HTMLImageElement).src = `https://picsum.photos/seed/${listing.id ?? Math.random()}/300/200`;
            }}
          />
          <span
            className={`absolute top-2 left-2 text-white text-xs font-bold px-2 py-1 rounded-md ${
              isAuction ? "bg-red-500" : "bg-[#0071CE]"
            }`}
          >
            {isAuction ? "🔨 AUCTION" : "⚡ BUY NOW"}
          </span>
          {(listing.is_featured || listing.isFeatured) && (
            <span className="absolute top-2 right-2 bg-[#FFC220] text-gray-900 text-xs font-bold px-2 py-1 rounded-md">
              ⭐ TOP
            </span>
          )}
          {onFavorite && (
            <button
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                onFavorite(listing.id);
              }}
              className="absolute bottom-2 right-2 w-7 h-7 rounded-full bg-white bg-opacity-90 flex items-center justify-center hover:bg-red-50 transition-colors shadow"
            >
              <Heart size={14} className="text-gray-400 hover:text-red-500 transition-colors" />
            </button>
          )}
        </div>
        <div className="p-3 flex flex-col flex-1">
          <p className="text-sm text-gray-800 font-medium line-clamp-2 leading-snug flex-1">
            {listing.title}
          </p>
          <p className="text-base font-bold text-[#0071CE] mt-1.5">
            {formatPrice(price, listing.currency || "AED")}
          </p>
          {attrPills.length > 0 && (
            <div className="flex flex-wrap gap-1 mt-1.5">
              {attrPills.map((pill) => (
                <span
                  key={pill}
                  className="text-xs bg-gray-100 text-gray-600 px-2 py-0.5 rounded-full font-medium"
                >
                  {pill}
                </span>
              ))}
            </div>
          )}
          {isAuction && endsAt && <CountdownTimer endsAt={endsAt} />}
          <p className="text-xs text-gray-400 mt-1">📍 {location}</p>
        </div>
      </div>
    </Link>
  );
}
