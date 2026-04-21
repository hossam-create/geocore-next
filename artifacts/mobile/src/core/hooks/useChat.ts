import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  type InfiniteData,
} from "@tanstack/react-query";

import type {
  Conversation,
  Message,
  SendMessageInput,
} from "../../domain/entities";
import { SendMessageUseCase } from "../../domain/usecases/chat/sendMessage.usecase";
import type { Page } from "../utils/pagination";
import { PAGINATION } from "../../config/constants";
import { queryKeys } from "../constants/queryKeys";
import { getContainer } from "../../store/container";

export function useConversations() {
  return useQuery<Page<Conversation>, Error>({
    queryKey: queryKeys.chat.conversations(),
    queryFn: () =>
      getContainer().chat.listConversations({
        page: 1,
        pageSize: PAGINATION.defaultPageSize,
      }),
  });
}

export function useMessages(conversationId: string | undefined) {
  return useInfiniteQuery<Page<Message>, Error, InfiniteData<Page<Message>>>({
    queryKey: conversationId
      ? queryKeys.chat.messages(conversationId)
      : ["chat", "messages", "none"],
    enabled: Boolean(conversationId),
    initialPageParam: PAGINATION.defaultPage,
    queryFn: async ({ pageParam }) =>
      getContainer().chat.listMessages(conversationId as string, {
        page: pageParam as number,
        pageSize: PAGINATION.defaultPageSize,
      }),
    getNextPageParam: (lastPage) =>
      lastPage.hasMore ? lastPage.page + 1 : undefined,
  });
}

export function useSendMessage() {
  const client = useQueryClient();
  return useMutation<Message, Error, SendMessageInput>({
    mutationFn: (input) => {
      const useCase = new SendMessageUseCase(getContainer().chat);
      return useCase.execute(input);
    },
    onSuccess: (_, input) => {
      void client.invalidateQueries({
        queryKey: queryKeys.chat.messages(input.conversationId),
      });
      void client.invalidateQueries({
        queryKey: queryKeys.chat.conversations(),
      });
    },
  });
}
