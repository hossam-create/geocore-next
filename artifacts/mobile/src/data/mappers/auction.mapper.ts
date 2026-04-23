import type {
  Auction,
  AuctionStatus,
  AuctionType,
  Bid,
  Money,
} from "../../domain/entities";
import { toListing, type ApiListing } from "./listing.mapper";
import { toPublicUserDto, type ApiPublicUser } from "./user.mapper";

const TYPES: AuctionType[] = ["standard", "dutch", "reverse"];
const STATUSES: AuctionStatus[] = ["scheduled", "live", "ended", "cancelled"];

function toMoney(
  amount: number | null | undefined,
  currency: string | null | undefined,
): Money {
  return { amount: amount ?? 0, currency: currency ?? "USD" };
}

function normaliseEnum<T extends string>(
  value: string | null | undefined,
  valid: ReadonlyArray<T>,
  fallback: T,
): T {
  if (!value) return fallback;
  const v = value.toLowerCase() as T;
  return valid.includes(v) ? v : fallback;
}

export interface ApiAuction {
  id: string;
  listing_id: string;
  listing?: ApiListing | null;
  type?: string | null;
  status?: string | null;
  currency?: string | null;
  starting_bid?: number | null;
  current_bid?: number | null;
  min_increment?: number | null;
  bid_count?: number | null;
  top_bidder?: ApiPublicUser | null;
  starts_at?: string | null;
  ends_at: string;
  reserve_price?: number | null;
  buy_now_price?: number | null;
}

export function toAuction(api: ApiAuction): Auction {
  return {
    id: api.id,
    listingId: api.listing_id,
    listing: api.listing ? toListing(api.listing) : undefined,
    type: normaliseEnum(api.type, TYPES, "standard"),
    status: normaliseEnum(api.status, STATUSES, "live"),
    startingBid: toMoney(api.starting_bid, api.currency),
    currentBid: toMoney(api.current_bid, api.currency),
    minIncrement: toMoney(api.min_increment ?? 1, api.currency),
    bidCount: api.bid_count ?? 0,
    topBidder: api.top_bidder ? toPublicUserDto(api.top_bidder) : undefined,
    startsAt: api.starts_at ?? new Date().toISOString(),
    endsAt: api.ends_at,
    reservePrice:
      api.reserve_price !== null && api.reserve_price !== undefined
        ? toMoney(api.reserve_price, api.currency)
        : undefined,
    buyNowPrice:
      api.buy_now_price !== null && api.buy_now_price !== undefined
        ? toMoney(api.buy_now_price, api.currency)
        : undefined,
  };
}

export interface ApiBid {
  id: string;
  auction_id: string;
  bidder?: ApiPublicUser | null;
  bidder_id?: string | null;
  bidder_name?: string | null;
  amount: number;
  currency?: string | null;
  is_auto?: boolean | null;
  max_amount?: number | null;
  created_at: string;
}

export function toBid(api: ApiBid): Bid {
  const bidder = api.bidder
    ? toPublicUserDto(api.bidder)
    : {
        id: api.bidder_id ?? "unknown",
        name: api.bidder_name ?? "Anonymous",
        rating: 0,
        isVerified: false,
      };
  return {
    id: api.id,
    auctionId: api.auction_id,
    bidder,
    amount: toMoney(api.amount, api.currency),
    isAuto: Boolean(api.is_auto),
    maxAmount:
      api.max_amount !== null && api.max_amount !== undefined
        ? toMoney(api.max_amount, api.currency)
        : undefined,
    createdAt: api.created_at,
  };
}
