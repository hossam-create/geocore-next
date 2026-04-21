export type AuctionType = "standard" | "dutch" | "reverse" | "buy_now_only";

export function getAuctionType(listing: any): AuctionType {
  const raw = listing?.auction_type || listing?.auctionType || "";
  switch (raw.toLowerCase()) {
    case "dutch":
      return "dutch";
    case "reverse":
      return "reverse";
    case "buy_now_only":
    case "buy_now":
      return "buy_now_only";
    default:
      return "standard";
  }
}

export const AUCTION_TYPE_BADGE: Record<
  AuctionType,
  { label: string; className: string }
> = {
  standard: {
    label: "Standard",
    className: "bg-blue-100 text-blue-700",
  },
  dutch: {
    label: "Dutch",
    className: "bg-purple-100 text-purple-700",
  },
  reverse: {
    label: "Reverse",
    className: "bg-orange-100 text-orange-700",
  },
  buy_now_only: {
    label: "Buy Now",
    className: "bg-green-100 text-green-700",
  },
};
