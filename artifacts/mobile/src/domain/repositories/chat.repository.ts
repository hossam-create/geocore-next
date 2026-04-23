import type { Conversation, Message, SendMessageInput } from "../entities";
import type { Page, PaginationParams } from "../../core/utils/pagination";

export interface ChatRepository {
  listConversations(pagination?: PaginationParams): Promise<Page<Conversation>>;
  startConversation(listingId: string): Promise<Conversation>;
  listMessages(
    conversationId: string,
    pagination?: PaginationParams,
  ): Promise<Page<Message>>;
  sendMessage(input: SendMessageInput): Promise<Message>;
  markConversationRead(conversationId: string): Promise<void>;
}
