import type {
  CreatePaymentIntentPayload,
  PaymentRepository,
} from "../../domain/repositories/payment.repository";
import type { Money } from "../../domain/entities/listing.entity";
import type {
  PaymentIntent,
  PaymentProvider,
  PaymentStatus,
  WalletTransaction,
} from "../../domain/entities/payment.entity";
import type { Page, PaginationParams } from "../../core/utils/pagination";
import type { HttpClient } from "../api/axios.client";
import { unwrapEnvelope } from "../api/axios.client";
import { PAYMENT_ENDPOINTS } from "../api/endpoints";
import type { ApiEnvelope, ApiMeta } from "../api/types";

interface ApiPaymentIntent {
  id: string;
  client_secret: string;
  provider?: string | null;
  amount: number;
  currency: string;
  status?: string | null;
}

interface ApiWalletTransaction {
  id: string;
  type: "credit" | "debit";
  amount: number;
  currency: string;
  description?: string | null;
  status?: string | null;
  created_at: string;
}

const PROVIDERS: PaymentProvider[] = ["stripe", "paymob", "wallet"];
const STATUSES: PaymentStatus[] = [
  "pending",
  "processing",
  "succeeded",
  "failed",
  "refunded",
];

function normaliseEnum<T extends string>(
  value: string | null | undefined,
  valid: ReadonlyArray<T>,
  fallback: T,
): T {
  if (!value) return fallback;
  const v = value.toLowerCase() as T;
  return valid.includes(v) ? v : fallback;
}

function toIntent(api: ApiPaymentIntent): PaymentIntent {
  return {
    id: api.id,
    clientSecret: api.client_secret,
    provider: normaliseEnum(api.provider, PROVIDERS, "stripe"),
    amount: { amount: api.amount, currency: api.currency },
    status: normaliseEnum(api.status, STATUSES, "pending"),
  };
}

function toTransaction(api: ApiWalletTransaction): WalletTransaction {
  return {
    id: api.id,
    type: api.type,
    amount: { amount: api.amount, currency: api.currency },
    description: api.description ?? "",
    status: normaliseEnum(api.status, STATUSES, "succeeded"),
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

export class HttpPaymentRepository implements PaymentRepository {
  constructor(private readonly http: HttpClient) {}

  async createIntent(
    payload: CreatePaymentIntentPayload,
  ): Promise<PaymentIntent> {
    const { data } = await this.http.instance.post<
      ApiEnvelope<ApiPaymentIntent>
    >(PAYMENT_ENDPOINTS.intent, {
      listing_id: payload.listingId,
      auction_id: payload.auctionId,
      amount: payload.amount,
      currency: payload.currency,
      provider: payload.provider ?? "stripe",
    });
    return toIntent(unwrapEnvelope(data));
  }

  async getPublishableKey(): Promise<string> {
    const { data } = await this.http.instance.get<
      ApiEnvelope<{ publishable_key: string }>
    >(PAYMENT_ENDPOINTS.publishableKey);
    return unwrapEnvelope(data).publishable_key;
  }

  async getWalletBalance(): Promise<Money> {
    const { data } = await this.http.instance.get<
      ApiEnvelope<{ amount: number; currency: string }>
    >(PAYMENT_ENDPOINTS.walletBalance);
    const body = unwrapEnvelope(data);
    return { amount: body.amount, currency: body.currency };
  }

  async listTransactions(
    pagination?: PaginationParams,
  ): Promise<Page<WalletTransaction>> {
    const { data } = await this.http.instance.get<
      ApiEnvelope<ApiWalletTransaction[]>
    >(PAYMENT_ENDPOINTS.walletTransactions, { params: pagination });
    return toPage(unwrapEnvelope(data), toTransaction, data.meta);
  }
}
