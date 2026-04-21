import type {
  Conversation,
  Message,
  MessageType,
} from "../../domain/entities";
import { toPublicUserDto, type ApiPublicUser } from "./user.mapper";

const MESSAGE_TYPES: MessageType[] = ["text", "image", "location", "system"];

function normaliseType(value: string | null | undefined): MessageType {
  if (!value) return "text";
  const v = value.toLowerCase() as MessageType;
  return MESSAGE_TYPES.includes(v) ? v : "text";
}

export interface ApiMessage {
  id: string;
  conversation_id: string;
  sender_id: string;
  type?: string | null;
  text?: string | null;
  body?: string | null;
  attachment_url?: string | null;
  is_read?: boolean | null;
  created_at: string;
}

export function toMessage(api: ApiMessage): Message {
  return {
    id: api.id,
    conversationId: api.conversation_id,
    senderId: api.sender_id,
    type: normaliseType(api.type),
    text: api.text ?? api.body ?? "",
    attachmentUrl: api.attachment_url ?? undefined,
    isRead: Boolean(api.is_read),
    createdAt: api.created_at,
  };
}

export interface ApiConversation {
  id: string;
  listing_id?: string | null;
  listing_title?: string | null;
  listing_image?: string | null;
  other_user?: ApiPublicUser | null;
  other_user_id?: string | null;
  other_user_name?: string | null;
  last_message?: ApiMessage | null;
  unread_count?: number | null;
  updated_at: string;
}

export function toConversation(api: ApiConversation): Conversation {
  const otherUser = api.other_user
    ? toPublicUserDto(api.other_user)
    : {
        id: api.other_user_id ?? "unknown",
        name: api.other_user_name ?? "Unknown",
        rating: 0,
        isVerified: false,
      };
  return {
    id: api.id,
    listingId: api.listing_id ?? undefined,
    listingTitle: api.listing_title ?? undefined,
    listingImage: api.listing_image ?? undefined,
    otherUser,
    lastMessage: api.last_message ? toMessage(api.last_message) : undefined,
    unreadCount: api.unread_count ?? 0,
    updatedAt: api.updated_at,
  };
}
