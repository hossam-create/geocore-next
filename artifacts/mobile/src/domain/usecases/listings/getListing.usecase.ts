import type { Listing } from "../../entities";
import type { ListingRepository } from "../../repositories/listing.repository";

export class GetListingUseCase {
  constructor(private readonly listings: ListingRepository) {}

  execute(id: string): Promise<Listing> {
    return this.listings.get(id);
  }
}
