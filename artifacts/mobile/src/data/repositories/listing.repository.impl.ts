import type {
  CreateListingPayload,
  ListingRepository,
} from "../../domain/repositories/listing.repository";
import type { Listing, ListingFilter } from "../../domain/entities";
import type { Page, PaginationParams } from "../../core/utils/pagination";
import type { HttpClient } from "../api/axios.client";
import { unwrapEnvelope } from "../api/axios.client";
import { LISTING_ENDPOINTS } from "../api/endpoints";
import type { ApiEnvelope, ApiMeta } from "../api/types";
import { toListing, type ApiListing } from "../mappers/listing.mapper";

function toPage(list: ApiListing[], meta?: ApiMeta): Page<Listing> {
  const page = meta?.page ?? 1;
  const pageSize = meta?.per_page ?? list.length;
  const total = meta?.total ?? list.length;
  return {
    items: list.map(toListing),
    page,
    pageSize,
    total,
    hasMore: meta?.has_more ?? page * pageSize < total,
  };
}

function toQuery(
  filter?: ListingFilter,
  pagination?: PaginationParams,
): Record<string, string | number | boolean> {
  const q: Record<string, string | number | boolean> = {};
  if (pagination?.page) q.page = pagination.page;
  if (pagination?.pageSize) q.per_page = pagination.pageSize;
  if (filter?.search) q.search = filter.search;
  if (filter?.category) q.category = filter.category;
  if (filter?.condition) q.condition = filter.condition;
  if (filter?.minPrice !== undefined) q.min_price = filter.minPrice;
  if (filter?.maxPrice !== undefined) q.max_price = filter.maxPrice;
  if (filter?.currency) q.currency = filter.currency;
  if (filter?.country) q.country = filter.country;
  if (filter?.city) q.city = filter.city;
  if (filter?.sortBy) q.sort = filter.sortBy;
  if (filter?.isAuction !== undefined) q.is_auction = filter.isAuction;
  if (filter?.sellerId) q.seller_id = filter.sellerId;
  return q;
}

function buildFormData(payload: CreateListingPayload): FormData {
  const fd = new FormData();
  fd.append("title", payload.title);
  fd.append("description", payload.description);
  fd.append("price", String(payload.priceAmount));
  fd.append("currency", payload.priceCurrency);
  fd.append("category", payload.category);
  fd.append("condition", payload.condition);
  fd.append("location", payload.location);
  if (payload.lat !== undefined) fd.append("lat", String(payload.lat));
  if (payload.lng !== undefined) fd.append("lng", String(payload.lng));
  if (payload.isAuction !== undefined) {
    fd.append("is_auction", String(payload.isAuction));
  }
  for (const tag of payload.tags) {
    fd.append("tags[]", tag);
  }
  payload.imageUris.forEach((uri, idx) => {
    // React Native `FormData` supports `{ uri, name, type }` entries.
    fd.append("images[]", {
      uri,
      name: `image-${idx}.jpg`,
      type: "image/jpeg",
    } as unknown as Blob);
  });
  return fd;
}

export class HttpListingRepository implements ListingRepository {
  constructor(private readonly http: HttpClient) {}

  async list(
    filter?: ListingFilter,
    pagination?: PaginationParams,
  ): Promise<Page<Listing>> {
    const { data } = await this.http.instance.get<ApiEnvelope<ApiListing[]>>(
      LISTING_ENDPOINTS.list,
      { params: toQuery(filter, pagination) },
    );
    return toPage(unwrapEnvelope(data), data.meta);
  }

  async get(id: string): Promise<Listing> {
    const { data } = await this.http.instance.get<ApiEnvelope<ApiListing>>(
      LISTING_ENDPOINTS.detail(id),
    );
    return toListing(unwrapEnvelope(data));
  }

  async create(payload: CreateListingPayload): Promise<Listing> {
    const { data } = await this.http.instance.post<ApiEnvelope<ApiListing>>(
      LISTING_ENDPOINTS.list,
      buildFormData(payload),
      { headers: { "Content-Type": "multipart/form-data" } },
    );
    return toListing(unwrapEnvelope(data));
  }

  async update(
    id: string,
    payload: Partial<CreateListingPayload>,
  ): Promise<Listing> {
    const { data } = await this.http.instance.patch<ApiEnvelope<ApiListing>>(
      LISTING_ENDPOINTS.detail(id),
      payload,
    );
    return toListing(unwrapEnvelope(data));
  }

  async delete(id: string): Promise<void> {
    await this.http.instance.delete(LISTING_ENDPOINTS.detail(id));
  }

  async listMine(pagination?: PaginationParams): Promise<Page<Listing>> {
    const { data } = await this.http.instance.get<ApiEnvelope<ApiListing[]>>(
      LISTING_ENDPOINTS.mine,
      { params: toQuery(undefined, pagination) },
    );
    return toPage(unwrapEnvelope(data), data.meta);
  }

  async listFavorites(pagination?: PaginationParams): Promise<Page<Listing>> {
    const { data } = await this.http.instance.get<ApiEnvelope<ApiListing[]>>(
      LISTING_ENDPOINTS.favorites,
      { params: toQuery(undefined, pagination) },
    );
    return toPage(unwrapEnvelope(data), data.meta);
  }

  async toggleFavorite(id: string): Promise<{ isFavorited: boolean }> {
    const { data } = await this.http.instance.post<
      ApiEnvelope<{ is_favorited: boolean }>
    >(LISTING_ENDPOINTS.toggleFavorite(id));
    const body = unwrapEnvelope(data);
    return { isFavorited: Boolean(body.is_favorited) };
  }
}
