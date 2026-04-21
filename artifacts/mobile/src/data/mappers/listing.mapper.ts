import type {
  Listing,
  ListingCategory,
  ListingCondition,
  ListingImage,
  ListingStatus,
  Money,
} from "../../domain/entities";
import { toPublicUserDto, type ApiPublicUser } from "./user.mapper";

export interface ApiListingImage {
  id: string;
  url: string;
  order?: number | null;
}

export interface ApiListing {
  id: string;
  title: string;
  description: string;
  price?: number | null;
  currency?: string | null;
  category?: string | null;
  condition?: string | null;
  status?: string | null;
  location?: string | null;
  lat?: number | null;
  lng?: number | null;
  lon?: number | null;
  images?: ApiListingImage[] | null;
  image_url?: string | null;
  seller?: ApiPublicUser | null;
  seller_id?: string | null;
  seller_name?: string | null;
  views?: number | null;
  favorite_count?: number | null;
  is_favorited?: boolean | null;
  is_featured?: boolean | null;
  is_auction?: boolean | null;
  auction_id?: string | null;
  tags?: string[] | null;
  created_at: string;
  updated_at?: string | null;
}

const CATEGORIES: ListingCategory[] = [
  "vehicles",
  "electronics",
  "furniture",
  "fashion",
  "real-estate",
  "services",
  "sports",
  "other",
];

const CONDITIONS: ListingCondition[] = [
  "new",
  "like-new",
  "good",
  "fair",
  "poor",
];

const STATUSES: ListingStatus[] = [
  "draft",
  "active",
  "sold",
  "expired",
  "removed",
];

function normaliseEnum<T extends string>(
  value: string | null | undefined,
  valid: ReadonlyArray<T>,
  fallback: T,
): T {
  if (!value) return fallback;
  const v = value.toLowerCase() as T;
  return valid.includes(v) ? v : fallback;
}

function toMoney(amount: number | null | undefined, currency: string | null | undefined): Money {
  return { amount: amount ?? 0, currency: currency ?? "USD" };
}

function toImages(
  api: ApiListing,
): ReadonlyArray<ListingImage> {
  if (api.images && api.images.length > 0) {
    return api.images.map((img, idx) => ({
      id: img.id,
      url: img.url,
      order: img.order ?? idx,
    }));
  }
  if (api.image_url) {
    return [{ id: `${api.id}:cover`, url: api.image_url, order: 0 }];
  }
  return [];
}

export function toListing(api: ApiListing): Listing {
  const seller = api.seller
    ? toPublicUserDto(api.seller)
    : {
        id: api.seller_id ?? "unknown",
        name: api.seller_name ?? "Unknown seller",
        rating: 0,
        isVerified: false,
      };

  const lat = api.lat ?? undefined;
  const lng = api.lng ?? api.lon ?? undefined;

  return {
    id: api.id,
    title: api.title,
    description: api.description,
    price: toMoney(api.price, api.currency),
    category: normaliseEnum(api.category, CATEGORIES, "other"),
    condition: normaliseEnum(api.condition, CONDITIONS, "good"),
    status: normaliseEnum(api.status, STATUSES, "active"),
    location: api.location ?? "",
    geo: lat !== undefined && lng !== undefined ? { lat, lng } : undefined,
    images: toImages(api),
    seller,
    views: api.views ?? 0,
    favoriteCount: api.favorite_count ?? 0,
    isFavorited: Boolean(api.is_favorited),
    isFeatured: Boolean(api.is_featured),
    isAuction: Boolean(api.is_auction),
    auctionId: api.auction_id ?? undefined,
    tags: api.tags ?? [],
    createdAt: api.created_at,
    updatedAt: api.updated_at ?? api.created_at,
  };
}
