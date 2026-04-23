import Constants from "expo-constants";

type ExpoExtra = {
  apiBaseUrl?: string;
  socketUrl?: string;
  sentryDsn?: string;
};

const extra = (Constants.expoConfig?.extra ?? {}) as ExpoExtra;

function required(value: string | undefined, fallback: string): string {
  return value && value.length > 0 ? value : fallback;
}

export const env = {
  apiBaseUrl: required(
    process.env.EXPO_PUBLIC_API_BASE_URL ?? extra.apiBaseUrl,
    "https://geo-core-next.replit.app/api/v1",
  ),
  socketUrl: required(
    process.env.EXPO_PUBLIC_SOCKET_URL ?? extra.socketUrl,
    "wss://geo-core-next.replit.app/ws",
  ),
  sentryDsn: process.env.EXPO_PUBLIC_SENTRY_DSN ?? extra.sentryDsn ?? "",
  isProduction: process.env.NODE_ENV === "production",
  appVersion: Constants.expoConfig?.version ?? "0.0.0",
} as const;

export type Env = typeof env;
