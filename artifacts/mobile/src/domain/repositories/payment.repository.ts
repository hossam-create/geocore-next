import type {
  PaymentIntent,
  WalletTransaction,
} from "../entities/payment.entity";
import type { Money } from "../entities/listing.entity";
import type { Page, PaginationParams } from "../../core/utils/pagination";

export interface CreatePaymentIntentPayload {
  readonly listingId?: string;
  readonly auctionId?: string;
  readonly amount: number;
  readonly currency: string;
  readonly provider?: "stripe" | "paymob" | "wallet";
}

export interface PaymentRepository {
  createIntent(payload: CreatePaymentIntentPayload): Promise<PaymentIntent>;
  getPublishableKey(): Promise<string>;
  getWalletBalance(): Promise<Money>;
  listTransactions(
    pagination?: PaginationParams,
  ): Promise<Page<WalletTransaction>>;
}
