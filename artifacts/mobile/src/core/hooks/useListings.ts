import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  type InfiniteData,
} from "@tanstack/react-query";

import type { Listing, ListingFilter } from "../../domain/entities";
import { CreateListingUseCase } from "../../domain/usecases/listings/createListing.usecase";
import { GetListingUseCase } from "../../domain/usecases/listings/getListing.usecase";
import { ListListingsUseCase } from "../../domain/usecases/listings/listListings.usecase";
import { ToggleFavoriteUseCase } from "../../domain/usecases/listings/toggleFavorite.usecase";
import type { CreateListingPayload } from "../../domain/repositories/listing.repository";
import type { Page } from "../utils/pagination";
import { PAGINATION } from "../../config/constants";
import { queryKeys } from "../constants/queryKeys";
import { getContainer } from "../../store/container";

export function useListings(filter?: ListingFilter) {
  return useInfiniteQuery<Page<Listing>, Error, InfiniteData<Page<Listing>>>({
    queryKey: queryKeys.listings.list(filter),
    initialPageParam: PAGINATION.defaultPage,
    queryFn: async ({ pageParam }) => {
      const useCase = new ListListingsUseCase(getContainer().listings);
      return useCase.execute(filter, {
        page: pageParam as number,
        pageSize: PAGINATION.defaultPageSize,
      });
    },
    getNextPageParam: (lastPage) =>
      lastPage.hasMore ? lastPage.page + 1 : undefined,
  });
}

export function useListing(id: string | undefined) {
  return useQuery<Listing, Error>({
    queryKey: id ? queryKeys.listings.detail(id) : ["listings", "detail", "none"],
    enabled: Boolean(id),
    queryFn: () => {
      const useCase = new GetListingUseCase(getContainer().listings);
      return useCase.execute(id as string);
    },
  });
}

export function useCreateListing() {
  const client = useQueryClient();
  return useMutation<Listing, Error, CreateListingPayload>({
    mutationFn: (payload) => {
      const useCase = new CreateListingUseCase(getContainer().listings);
      return useCase.execute(payload);
    },
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: queryKeys.listings.all() });
    },
  });
}

export function useToggleFavorite() {
  const client = useQueryClient();
  return useMutation<{ isFavorited: boolean }, Error, string>({
    mutationFn: (id) => {
      const useCase = new ToggleFavoriteUseCase(getContainer().listings);
      return useCase.execute(id);
    },
    onSuccess: (_, id) => {
      void client.invalidateQueries({ queryKey: queryKeys.listings.detail(id) });
      void client.invalidateQueries({ queryKey: queryKeys.listings.favorites() });
    },
  });
}
