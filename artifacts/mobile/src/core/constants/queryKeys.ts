import type { ListingFilter } from "../../domain/entities/listing.entity";

export const queryKeys = {
  auth: {
    me: () => ["auth", "me"] as const,
  },
  listings: {
    all: () => ["listings"] as const,
    list: (filter?: ListingFilter) =>
      ["listings", "list", filter ?? {}] as const,
    detail: (id: string) => ["listings", "detail", id] as const,
    mine: () => ["listings", "mine"] as const,
    favorites: () => ["listings", "favorites"] as const,
  },
  auctions: {
    all: () => ["auctions"] as const,
    list: () => ["auctions", "list"] as const,
    detail: (id: string) => ["auctions", "detail", id] as const,
    bids: (id: string) => ["auctions", id, "bids"] as const,
    myBids: () => ["auctions", "myBids"] as const,
  },
  chat: {
    conversations: () => ["chat", "conversations"] as const,
    messages: (conversationId: string) =>
      ["chat", "messages", conversationId] as const,
  },
  notifications: {
    list: () => ["notifications", "list"] as const,
  },
  wallet: {
    balance: () => ["wallet", "balance"] as const,
    transactions: () => ["wallet", "transactions"] as const,
  },
} as const;
