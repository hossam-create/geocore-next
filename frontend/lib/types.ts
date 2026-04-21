export interface AuctionBid {
  id: string;
  amount: number;
  placed_at: string;
  user_id: string;
}

export interface Auction {
  id: string;
  title: string;
  status: "active" | "ended" | "cancelled" | "pending";
  seller_id: string;
  winner_id?: string | null;
  current_bid: number;
  start_price: number;
  buy_now_price?: number | null;
  bid_count: number;
  currency: string;
  ends_at: string;
  starts_at?: string;
  auction_type?: string;
  listing?: { title?: string };
  bids?: AuctionBid[];
}

export interface WalletTransaction {
  id: string;
  kind: string;
  amount: number;
  currency: string;
  status: string;
  description: string;
  created_at: string;
}

export interface WalletBalance {
  balance: number;
  pending?: number;
  escrowed?: number;
  total_card_spent: number;
  total_refunded: number;
  currency: string;
}

export interface ApiError {
  response?: {
    data?: {
      message?: string;
    };
  };
  message?: string;
}
