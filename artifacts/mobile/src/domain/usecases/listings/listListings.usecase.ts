import type { Listing, ListingFilter } from "../../entities";
import type { Page, PaginationParams } from "../../../core/utils/pagination";
import type { ListingRepository } from "../../repositories/listing.repository";

export class ListListingsUseCase {
  constructor(private readonly listings: ListingRepository) {}

  execute(
    filter?: ListingFilter,
    pagination?: PaginationParams,
  ): Promise<Page<Listing>> {
    return this.listings.list(filter, pagination);
  }
}
