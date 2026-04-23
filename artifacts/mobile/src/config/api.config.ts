import { env } from "./env";
import { TIMEOUTS } from "./constants";

export const apiConfig = {
  baseURL: env.apiBaseUrl,
  timeout: TIMEOUTS.requestMs,
  headers: {
    "Content-Type": "application/json",
    Accept: "application/json",
  },
} as const;
