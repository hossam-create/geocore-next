import type { Listing } from "../../entities";
import type {
  CreateListingPayload,
  ListingRepository,
} from "../../repositories/listing.repository";
import { LIMITS } from "../../../config/constants";
import { ValidationError } from "../../../core/utils/errors";

export class CreateListingUseCase {
  constructor(private readonly listings: ListingRepository) {}

  async execute(payload: CreateListingPayload): Promise<Listing> {
    const fields: Record<string, string> = {};
    if (!payload.title || payload.title.trim().length < 3) {
      fields.title = "Title must be at least 3 characters";
    }
    if (!payload.description || payload.description.trim().length < 10) {
      fields.description = "Description must be at least 10 characters";
    }
    if (payload.priceAmount < 0) {
      fields.price = "Price cannot be negative";
    }
    if (!payload.priceCurrency) {
      fields.currency = "Currency is required";
    }
    if (!payload.category) {
      fields.category = "Category is required";
    }
    if (payload.imageUris.length > LIMITS.maxImagesPerListing) {
      fields.images = `You can upload at most ${LIMITS.maxImagesPerListing} images`;
    }
    if (Object.keys(fields).length > 0) {
      throw new ValidationError("Invalid listing details", fields);
    }
    return this.listings.create(payload);
  }
}
