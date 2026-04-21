import type { Listing, Money } from "./listing.entity";
import type { PublicUser } from "./user.entity";

export type AuctionType = "standard" | "dutch" | "reverse";
export type AuctionStatus = "scheduled" | "live" | "ended" | "cancelled";

export interface Bid {
  readonly id: string;
  readonly auctionId: string;
  readonly bidder: PublicUser;
  readonly amount: Money;
  readonly isAuto: boolean;
  readonly maxAmount?: Money;
  readonly createdAt: string;
}

export interface Auction {
  readonly id: string;
  readonly listingId: string;
  readonly listing?: Listing;
  readonly type: AuctionType;
  readonly status: AuctionStatus;
  readonly startingBid: Money;
  readonly currentBid: Money;
  readonly minIncrement: Money;
  readonly bidCount: number;
  readonly topBidder?: PublicUser;
  readonly startsAt: string;
  readonly endsAt: string;
  readonly reservePrice?: Money;
  readonly buyNowPrice?: Money;
}

export interface PlaceBidInput {
  readonly auctionId: string;
  readonly amount: number;
  readonly currency?: string;
  readonly isAuto?: boolean;
  readonly maxAmount?: number;
}

export function isAuctionLive(auction: Auction, now: Date = new Date()): boolean {
  if (auction.status !== "live") return false;
  const end = new Date(auction.endsAt).getTime();
  return now.getTime() < end;
}
