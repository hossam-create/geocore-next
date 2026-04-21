import type { ListingRepository } from "../../repositories/listing.repository";

export class ToggleFavoriteUseCase {
  constructor(private readonly listings: ListingRepository) {}

  execute(id: string): Promise<{ isFavorited: boolean }> {
    return this.listings.toggleFavorite(id);
  }
}
