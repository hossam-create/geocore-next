import type { PublicUser } from "./user.entity";

export type MessageType = "text" | "image" | "location" | "system";

export interface Message {
  readonly id: string;
  readonly conversationId: string;
  readonly senderId: string;
  readonly type: MessageType;
  readonly text: string;
  readonly attachmentUrl?: string;
  readonly isRead: boolean;
  readonly createdAt: string;
}

export interface Conversation {
  readonly id: string;
  readonly listingId?: string;
  readonly listingTitle?: string;
  readonly listingImage?: string;
  readonly otherUser: PublicUser;
  readonly lastMessage?: Message;
  readonly unreadCount: number;
  readonly updatedAt: string;
}

export interface SendMessageInput {
  readonly conversationId: string;
  readonly type?: MessageType;
  readonly text: string;
  readonly attachmentUrl?: string;
}
