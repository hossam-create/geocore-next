import type { Listing, ListingFilter } from "../entities";
import type { Page, PaginationParams } from "../../core/utils/pagination";

export interface CreateListingPayload {
  readonly title: string;
  readonly description: string;
  readonly priceAmount: number;
  readonly priceCurrency: string;
  readonly category: string;
  readonly condition: string;
  readonly location: string;
  readonly lat?: number;
  readonly lng?: number;
  readonly tags: ReadonlyArray<string>;
  readonly isAuction?: boolean;
  readonly imageUris: ReadonlyArray<string>;
}

export interface ListingRepository {
  list(
    filter?: ListingFilter,
    pagination?: PaginationParams,
  ): Promise<Page<Listing>>;
  get(id: string): Promise<Listing>;
  create(payload: CreateListingPayload): Promise<Listing>;
  update(id: string, payload: Partial<CreateListingPayload>): Promise<Listing>;
  delete(id: string): Promise<void>;
  listMine(pagination?: PaginationParams): Promise<Page<Listing>>;
  listFavorites(pagination?: PaginationParams): Promise<Page<Listing>>;
  toggleFavorite(id: string): Promise<{ isFavorited: boolean }>;
}
