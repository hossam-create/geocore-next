import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  type InfiniteData,
} from "@tanstack/react-query";

import type { Auction, Bid, PlaceBidInput } from "../../domain/entities";
import { PlaceBidUseCase } from "../../domain/usecases/auctions/placeBid.usecase";
import type { Page } from "../utils/pagination";
import { PAGINATION } from "../../config/constants";
import { queryKeys } from "../constants/queryKeys";
import { getContainer } from "../../store/container";

export function useAuctions() {
  return useInfiniteQuery<Page<Auction>, Error, InfiniteData<Page<Auction>>>({
    queryKey: queryKeys.auctions.list(),
    initialPageParam: PAGINATION.defaultPage,
    queryFn: async ({ pageParam }) => {
      return getContainer().auctions.list({
        page: pageParam as number,
        pageSize: PAGINATION.defaultPageSize,
      });
    },
    getNextPageParam: (lastPage) =>
      lastPage.hasMore ? lastPage.page + 1 : undefined,
  });
}

export function useAuction(id: string | undefined) {
  return useQuery<Auction, Error>({
    queryKey: id ? queryKeys.auctions.detail(id) : ["auctions", "detail", "none"],
    enabled: Boolean(id),
    queryFn: () => getContainer().auctions.get(id as string),
  });
}

export function useAuctionBids(auctionId: string | undefined) {
  return useQuery<Page<Bid>, Error>({
    queryKey: auctionId
      ? queryKeys.auctions.bids(auctionId)
      : ["auctions", "bids", "none"],
    enabled: Boolean(auctionId),
    queryFn: () =>
      getContainer().auctions.listBids(auctionId as string, {
        page: 1,
        pageSize: PAGINATION.defaultPageSize,
      }),
  });
}

export function usePlaceBid() {
  const client = useQueryClient();
  return useMutation<Bid, Error, PlaceBidInput>({
    mutationFn: (input) => {
      const useCase = new PlaceBidUseCase(getContainer().auctions);
      return useCase.execute(input);
    },
    onSuccess: (_, input) => {
      void client.invalidateQueries({
        queryKey: queryKeys.auctions.detail(input.auctionId),
      });
      void client.invalidateQueries({
        queryKey: queryKeys.auctions.bids(input.auctionId),
      });
      void client.invalidateQueries({ queryKey: queryKeys.auctions.myBids() });
    },
  });
}
