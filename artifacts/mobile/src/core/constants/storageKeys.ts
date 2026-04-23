/**
 * Keys used for AsyncStorage / SecureStore. Centralised to prevent typos.
 * SECURE_* keys must only be read/written via SecureStorageService.
 */
export const SECURE_STORAGE_KEYS = {
  accessToken: "access_token",
  refreshToken: "refresh_token",
  biometricEnabled: "biometric_enabled",
} as const;

export const STORAGE_KEYS = {
  activeCurrency: "active_currency",
  themeMode: "theme_mode",
  language: "language",
  onboardingCompleted: "onboarding_completed",
  pushToken: "push_token",
} as const;

export type SecureStorageKey =
  (typeof SECURE_STORAGE_KEYS)[keyof typeof SECURE_STORAGE_KEYS];
export type StorageKey = (typeof STORAGE_KEYS)[keyof typeof STORAGE_KEYS];
