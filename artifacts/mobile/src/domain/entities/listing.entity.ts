import type { PublicUser } from "./user.entity";

export type ListingCategory =
  | "vehicles"
  | "electronics"
  | "furniture"
  | "fashion"
  | "real-estate"
  | "services"
  | "sports"
  | "other";

export type ListingCondition =
  | "new"
  | "like-new"
  | "good"
  | "fair"
  | "poor";

export type ListingStatus = "draft" | "active" | "sold" | "expired" | "removed";

export interface ListingImage {
  readonly id: string;
  readonly url: string;
  readonly order: number;
}

export interface Money {
  readonly amount: number;
  readonly currency: string;
}

export interface GeoPoint {
  readonly lat: number;
  readonly lng: number;
}

export interface Listing {
  readonly id: string;
  readonly title: string;
  readonly description: string;
  readonly price: Money;
  readonly category: ListingCategory;
  readonly condition: ListingCondition;
  readonly status: ListingStatus;
  readonly location: string;
  readonly geo?: GeoPoint;
  readonly images: ReadonlyArray<ListingImage>;
  readonly seller: PublicUser;
  readonly views: number;
  readonly favoriteCount: number;
  readonly isFavorited: boolean;
  readonly isFeatured: boolean;
  readonly isAuction: boolean;
  readonly auctionId?: string;
  readonly tags: ReadonlyArray<string>;
  readonly createdAt: string;
  readonly updatedAt: string;
}

export interface ListingFilter {
  readonly search?: string;
  readonly category?: ListingCategory;
  readonly condition?: ListingCondition;
  readonly minPrice?: number;
  readonly maxPrice?: number;
  readonly currency?: string;
  readonly country?: string;
  readonly city?: string;
  readonly sortBy?: "newest" | "price_asc" | "price_desc" | "most_viewed";
  readonly isAuction?: boolean;
  readonly sellerId?: string;
}
