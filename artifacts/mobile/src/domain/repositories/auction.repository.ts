import type { Auction, Bid, PlaceBidInput } from "../entities";
import type { Page, PaginationParams } from "../../core/utils/pagination";

export interface AuctionRepository {
  list(pagination?: PaginationParams): Promise<Page<Auction>>;
  get(id: string): Promise<Auction>;
  listBids(auctionId: string, pagination?: PaginationParams): Promise<Page<Bid>>;
  placeBid(input: PlaceBidInput): Promise<Bid>;
  listMyBids(pagination?: PaginationParams): Promise<Page<Bid>>;
}
