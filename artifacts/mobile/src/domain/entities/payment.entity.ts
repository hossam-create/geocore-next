import type { Money } from "./listing.entity";

export type PaymentStatus =
  | "pending"
  | "processing"
  | "succeeded"
  | "failed"
  | "refunded";

export type PaymentProvider = "stripe" | "paymob" | "wallet";

export interface PaymentIntent {
  readonly id: string;
  readonly clientSecret: string;
  readonly provider: PaymentProvider;
  readonly amount: Money;
  readonly status: PaymentStatus;
}

export interface WalletTransaction {
  readonly id: string;
  readonly type: "credit" | "debit";
  readonly amount: Money;
  readonly description: string;
  readonly status: PaymentStatus;
  readonly createdAt: string;
}
