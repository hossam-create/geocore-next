import { http } from "../data/api/axios.client";
import {
  HttpAuctionRepository,
  HttpAuthRepository,
  HttpChatRepository,
  HttpListingRepository,
  HttpNotificationRepository,
  HttpPaymentRepository,
  HttpUserRepository,
} from "../data/repositories";
import type {
  AuctionRepository,
  AuthRepository,
  ChatRepository,
  ListingRepository,
  NotificationRepository,
  PaymentRepository,
  UserRepository,
} from "../domain/repositories";

/**
 * Dependency-injection container — gives features a single place to resolve
 * repositories. Tests can call `setContainer` with fakes before rendering.
 */
export interface Container {
  readonly auth: AuthRepository;
  readonly users: UserRepository;
  readonly listings: ListingRepository;
  readonly auctions: AuctionRepository;
  readonly chat: ChatRepository;
  readonly payments: PaymentRepository;
  readonly notifications: NotificationRepository;
}

function build(): Container {
  return {
    auth: new HttpAuthRepository(http),
    users: new HttpUserRepository(http),
    listings: new HttpListingRepository(http),
    auctions: new HttpAuctionRepository(http),
    chat: new HttpChatRepository(http),
    payments: new HttpPaymentRepository(http),
    notifications: new HttpNotificationRepository(http),
  };
}

let current: Container = build();

export function getContainer(): Container {
  return current;
}

export function setContainer(next: Container): void {
  current = next;
}

export function resetContainer(): void {
  current = build();
}
