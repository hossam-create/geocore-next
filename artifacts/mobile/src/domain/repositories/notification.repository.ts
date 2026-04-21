import type { AppNotification } from "../entities/notification.entity";
import type { Page, PaginationParams } from "../../core/utils/pagination";

export interface NotificationRepository {
  list(pagination?: PaginationParams): Promise<Page<AppNotification>>;
  markRead(id: string): Promise<void>;
  markAllRead(): Promise<void>;
  registerPushToken(token: string, platform: "ios" | "android"): Promise<void>;
}
