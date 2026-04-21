'use client'
import Link from 'next/link';
import { CountdownTimer } from "@/components/ui/CountdownTimer";
import { formatPrice } from "@/lib/utils";
import { Users, TrendingDown, Layers } from "lucide-react";
import { getAuctionType, AUCTION_TYPE_BADGE } from "@/lib/auctionTypes";

export function AuctionCard({ auction }: { auction: any }) {
  const auctionType = getAuctionType(auction);
  const badge = AUCTION_TYPE_BADGE[auctionType];
  const currentBid = auction.current_bid ?? auction.currentBid ?? auction.start_price ?? 0;
  const imageUrl =
    auction.listing?.images?.[0]?.url ||
    auction.images?.[0]?.url ||
    `https://picsum.photos/seed/${auction.id}/300/200`;
  const title = auction.listing?.title || auction.title || "Auction Item";
  const endsAt = auction.ends_at || auction.endsAt || auction.auctionEndsAt;

  const totalSlots = auction.total_slots ?? auction.totalSlots;
  const slotsWon = auction.slots_won ?? auction.slotsWon ?? 0;
  const slotsRemaining = totalSlots != null ? totalSlots - slotsWon : null;

  const lowestOffer = auction.lowest_offer ?? auction.lowestOffer ?? currentBid;
  const clearingPrice = auction.clearing_price ?? auction.clearingPrice ?? currentBid;

  return (
    <Link href={`/auctions/${auction.id}`}>
      <div className="bg-white rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-all hover:-translate-y-0.5 cursor-pointer group">
        <div className="relative">
          <img
            src={imageUrl}
            alt={title}
            className="w-full h-44 object-cover group-hover:scale-[1.02] transition-transform duration-300"
            onError={(e) => {
              (e.target as HTMLImageElement).src = `https://picsum.photos/seed/${auction.id}/300/200`;
            }}
          />
          <span className="absolute top-2 left-2 bg-red-500 text-white text-xs font-bold px-2 py-1 rounded-md">
            🔨 LIVE
          </span>
          <span className={`absolute top-2 right-2 text-xs font-bold px-2 py-1 rounded-md ${badge.className}`}>
            {badge.label}
          </span>
        </div>
        <div className="p-3">
          <p className="text-sm text-gray-800 font-medium line-clamp-2 leading-snug">{title}</p>

          {auctionType === "dutch" && (
            <div className="mt-2 bg-purple-600 rounded-lg p-2.5 text-white">
              <p className="text-xs text-purple-200">Clearing Price</p>
              <p className="text-lg font-extrabold">
                {formatPrice(clearingPrice, auction.currency || "AED")}
              </p>
              {endsAt && <CountdownTimer endsAt={endsAt} className="text-yellow-300" />}
              {slotsRemaining != null && (
                <p className="text-xs text-purple-200 mt-1 flex items-center gap-1">
                  <Layers size={11} /> {slotsRemaining} of {totalSlots} slots remaining
                </p>
              )}
            </div>
          )}

          {auctionType === "reverse" && (
            <div className="mt-2 bg-orange-500 rounded-lg p-2.5 text-white">
              <p className="text-xs text-orange-100">Lowest Offer</p>
              <p className="text-lg font-extrabold">
                {formatPrice(lowestOffer, auction.currency || "AED")}
              </p>
              {endsAt && <CountdownTimer endsAt={endsAt} className="text-yellow-300" />}
              {auction.bid_count !== undefined && (
                <p className="text-xs text-orange-100 mt-1 flex items-center gap-1">
                  <TrendingDown size={11} /> {auction.bid_count} offers submitted
                </p>
              )}
            </div>
          )}

          {(auctionType === "standard" || auctionType === "buy_now_only") && (
            <div className="mt-2 bg-[#0071CE] rounded-lg p-2.5 text-white">
              <p className="text-xs text-blue-200">Current Bid</p>
              <p className="text-lg font-extrabold">
                {formatPrice(currentBid, auction.currency || "AED")}
              </p>
              {endsAt && <CountdownTimer endsAt={endsAt} className="text-yellow-300" />}
              {auction.bid_count !== undefined && (
                <p className="text-xs text-blue-200 mt-1 flex items-center gap-1">
                  <Users size={11} /> {auction.bid_count} bids
                </p>
              )}
            </div>
          )}
        </div>
      </div>
    </Link>
  );
}
