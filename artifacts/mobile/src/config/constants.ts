export const APP_NAME = "GeoCore";

export const PAGINATION = {
  defaultPage: 1,
  defaultPageSize: 20,
  maxPageSize: 50,
} as const;

export const TIMEOUTS = {
  requestMs: 30_000,
  socketReconnectMs: 3_000,
  otpCountdownSec: 60,
} as const;

export const LIMITS = {
  maxImagesPerListing: 10,
  maxMessageLength: 2_000,
  minPasswordLength: 8,
  minAuctionBidStep: 1,
} as const;
