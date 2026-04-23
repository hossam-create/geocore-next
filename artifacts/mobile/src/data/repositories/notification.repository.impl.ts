import type { NotificationRepository } from "../../domain/repositories/notification.repository";
import type {
  AppNotification,
  NotificationType,
} from "../../domain/entities/notification.entity";
import type { Page, PaginationParams } from "../../core/utils/pagination";
import type { HttpClient } from "../api/axios.client";
import { unwrapEnvelope } from "../api/axios.client";
import { NOTIFICATION_ENDPOINTS } from "../api/endpoints";
import type { ApiEnvelope, ApiMeta } from "../api/types";

interface ApiNotification {
  id: string;
  type: string;
  title: string;
  body: string;
  data?: Record<string, string> | null;
  is_read?: boolean | null;
  created_at: string;
}

const TYPES: NotificationType[] = [
  "listing_sold",
  "new_message",
  "bid_outbid",
  "auction_ending",
  "auction_won",
  "payment_received",
  "system",
];

function toNotification(api: ApiNotification): AppNotification {
  const type = (TYPES as string[]).includes(api.type)
    ? (api.type as NotificationType)
    : "system";
  return {
    id: api.id,
    type,
    title: api.title,
    body: api.body,
    data: api.data ?? undefined,
    isRead: Boolean(api.is_read),
    createdAt: api.created_at,
  };
}

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

export class HttpNotificationRepository implements NotificationRepository {
  constructor(private readonly http: HttpClient) {}

  async list(pagination?: PaginationParams): Promise<Page<AppNotification>> {
    const { data } = await this.http.instance.get<
      ApiEnvelope<ApiNotification[]>
    >(NOTIFICATION_ENDPOINTS.list, { params: pagination });
    return toPage(unwrapEnvelope(data), toNotification, data.meta);
  }

  async markRead(id: string): Promise<void> {
    await this.http.instance.post(NOTIFICATION_ENDPOINTS.markRead(id));
  }

  async markAllRead(): Promise<void> {
    await this.http.instance.post(NOTIFICATION_ENDPOINTS.markAllRead);
  }

  async registerPushToken(
    token: string,
    platform: "ios" | "android",
  ): Promise<void> {
    await this.http.instance.post(NOTIFICATION_ENDPOINTS.registerPush, {
      token,
      platform,
    });
  }
}
