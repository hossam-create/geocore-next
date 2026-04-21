import type { AuctionRepository } from "../../domain/repositories/auction.repository";
import type { Auction, Bid, PlaceBidInput } from "../../domain/entities";
import type { Page, PaginationParams } from "../../core/utils/pagination";
import type { HttpClient } from "../api/axios.client";
import { unwrapEnvelope } from "../api/axios.client";
import { AUCTION_ENDPOINTS } from "../api/endpoints";
import type { ApiEnvelope, ApiMeta } from "../api/types";
import {
  toAuction,
  toBid,
  type ApiAuction,
  type ApiBid,
} from "../mappers/auction.mapper";

function toPage<T, U>(
  items: ReadonlyArray<T>,
  mapper: (item: T) => U,
  meta?: ApiMeta,
): Page<U> {
  const page = meta?.page ?? 1;
  const pageSize = meta?.per_page ?? items.length;
  const total = meta?.total ?? items.length;
  return {
    items: items.map(mapper),
    page,
    pageSize,
    total,
    hasMore: meta?.has_more ?? page * pageSize < total,
  };
}

export class HttpAuctionRepository implements AuctionRepository {
  constructor(private readonly http: HttpClient) {}

  async list(pagination?: PaginationParams): Promise<Page<Auction>> {
    const { data } = await this.http.instance.get<ApiEnvelope<ApiAuction[]>>(
      AUCTION_ENDPOINTS.list,
      { params: pagination },
    );
    return toPage(unwrapEnvelope(data), toAuction, data.meta);
  }

  async get(id: string): Promise<Auction> {
    const { data } = await this.http.instance.get<ApiEnvelope<ApiAuction>>(
      AUCTION_ENDPOINTS.detail(id),
    );
    return toAuction(unwrapEnvelope(data));
  }

  async listBids(
    auctionId: string,
    pagination?: PaginationParams,
  ): Promise<Page<Bid>> {
    const { data } = await this.http.instance.get<ApiEnvelope<ApiBid[]>>(
      AUCTION_ENDPOINTS.bids(auctionId),
      { params: pagination },
    );
    return toPage(unwrapEnvelope(data), toBid, data.meta);
  }

  async placeBid(input: PlaceBidInput): Promise<Bid> {
    const body: Record<string, unknown> = { amount: input.amount };
    if (input.currency) body.currency = input.currency;
    if (input.isAuto) {
      body.is_auto = true;
      if (input.maxAmount !== undefined) body.max_amount = input.maxAmount;
    }
    const { data } = await this.http.instance.post<ApiEnvelope<ApiBid>>(
      AUCTION_ENDPOINTS.placeBid(input.auctionId),
      body,
    );
    return toBid(unwrapEnvelope(data));
  }

  async listMyBids(pagination?: PaginationParams): Promise<Page<Bid>> {
    const { data } = await this.http.instance.get<ApiEnvelope<ApiBid[]>>(
      AUCTION_ENDPOINTS.myBids,
      { params: pagination },
    );
    return toPage(unwrapEnvelope(data), toBid, data.meta);
  }
}
