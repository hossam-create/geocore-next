/**
 * Backend endpoint paths. All paths are relative to `env.apiBaseUrl`.
 * Keeping these in one place makes it easy to audit routes against the
 * Go API and update on breaking changes.
 */
export const AUTH_ENDPOINTS = {
  register: "/auth/register",
  login: "/auth/login",
  logout: "/auth/logout",
  refresh: "/auth/refresh",
  me: "/auth/me",
  forgotPassword: "/auth/forgot-password",
  resetPassword: "/auth/reset-password",
  verifyOtp: "/auth/verify-otp",
  resendOtp: "/auth/resend-otp",
} as const;

export const USER_ENDPOINTS = {
  me: "/users/me",
  publicProfile: (id: string) => `/users/${id}/profile`,
} as const;

export const LISTING_ENDPOINTS = {
  list: "/listings",
  detail: (id: string) => `/listings/${id}`,
  mine: "/listings/me",
  favorites: "/listings/favorites",
  toggleFavorite: (id: string) => `/listings/${id}/favorite`,
  categories: "/categories",
} as const;

export const AUCTION_ENDPOINTS = {
  list: "/auctions",
  detail: (id: string) => `/auctions/${id}`,
  bids: (id: string) => `/auctions/${id}/bids`,
  placeBid: (id: string) => `/auctions/${id}/bid`,
  myBids: "/users/me/bids",
} as const;

export const CHAT_ENDPOINTS = {
  conversations: "/chat/conversations",
  startConversation: "/chat/conversations",
  messages: (conversationId: string) =>
    `/chat/conversations/${conversationId}/messages`,
  markRead: (conversationId: string) =>
    `/chat/conversations/${conversationId}/read`,
} as const;

export const PAYMENT_ENDPOINTS = {
  intent: "/payments/intent",
  publishableKey: "/payments/key",
  walletBalance: "/wallet/balance",
  walletTransactions: "/wallet/transactions",
} as const;

export const NOTIFICATION_ENDPOINTS = {
  list: "/notifications",
  markRead: (id: string) => `/notifications/${id}/read`,
  markAllRead: "/notifications/read-all",
  registerPush: "/notifications/push-token",
} as const;
