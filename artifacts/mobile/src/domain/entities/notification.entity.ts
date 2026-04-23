export type NotificationType =
  | "listing_sold"
  | "new_message"
  | "bid_outbid"
  | "auction_ending"
  | "auction_won"
  | "payment_received"
  | "system";

export interface AppNotification {
  readonly id: string;
  readonly type: NotificationType;
  readonly title: string;
  readonly body: string;
  readonly data?: Readonly<Record<string, string>>;
  readonly isRead: boolean;
  readonly createdAt: string;
}
