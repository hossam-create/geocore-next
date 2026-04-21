import type { ChatRepository } from "../../domain/repositories/chat.repository";
import type {
  Conversation,
  Message,
  SendMessageInput,
} from "../../domain/entities";
import type { Page, PaginationParams } from "../../core/utils/pagination";
import type { HttpClient } from "../api/axios.client";
import { unwrapEnvelope } from "../api/axios.client";
import { CHAT_ENDPOINTS } from "../api/endpoints";
import type { ApiEnvelope, ApiMeta } from "../api/types";
import {
  toConversation,
  toMessage,
  type ApiConversation,
  type ApiMessage,
} from "../mappers/chat.mapper";

function toPage<T, U>(
  items: ReadonlyArray<T>,
  mapper: (t: T) => U,
  meta?: ApiMeta,
): Page<U> {
  const page = meta?.page ?? 1;
  const pageSize = meta?.per_page ?? items.length;
  const total = meta?.total ?? items.length;
  return {
    items: items.map(mapper),
    page,
    pageSize,
    total,
    hasMore: meta?.has_more ?? page * pageSize < total,
  };
}

export class HttpChatRepository implements ChatRepository {
  constructor(private readonly http: HttpClient) {}

  async listConversations(
    pagination?: PaginationParams,
  ): Promise<Page<Conversation>> {
    const { data } = await this.http.instance.get<
      ApiEnvelope<ApiConversation[]>
    >(CHAT_ENDPOINTS.conversations, { params: pagination });
    return toPage(unwrapEnvelope(data), toConversation, data.meta);
  }

  async startConversation(listingId: string): Promise<Conversation> {
    const { data } = await this.http.instance.post<
      ApiEnvelope<ApiConversation>
    >(CHAT_ENDPOINTS.startConversation, { listing_id: listingId });
    return toConversation(unwrapEnvelope(data));
  }

  async listMessages(
    conversationId: string,
    pagination?: PaginationParams,
  ): Promise<Page<Message>> {
    const { data } = await this.http.instance.get<ApiEnvelope<ApiMessage[]>>(
      CHAT_ENDPOINTS.messages(conversationId),
      { params: pagination },
    );
    return toPage(unwrapEnvelope(data), toMessage, data.meta);
  }

  async sendMessage(input: SendMessageInput): Promise<Message> {
    const { data } = await this.http.instance.post<ApiEnvelope<ApiMessage>>(
      CHAT_ENDPOINTS.messages(input.conversationId),
      {
        type: input.type ?? "text",
        text: input.text,
        attachment_url: input.attachmentUrl,
      },
    );
    return toMessage(unwrapEnvelope(data));
  }

  async markConversationRead(conversationId: string): Promise<void> {
    await this.http.instance.post(CHAT_ENDPOINTS.markRead(conversationId));
  }
}
