import * as SecureStore from "expo-secure-store";

import type { SecureStorageKey } from "../constants/storageKeys";

/**
 * Wraps expo-secure-store. Reads/writes never throw — callers are expected
 * to treat missing values as "no value". SecureStore is backed by the
 * iOS Keychain / Android Keystore, so values persist across app launches.
 */
export const secureStorage = {
  async get(key: SecureStorageKey): Promise<string | null> {
    try {
      return await SecureStore.getItemAsync(key);
    } catch {
      return null;
    }
  },
  async set(key: SecureStorageKey, value: string): Promise<void> {
    try {
      await SecureStore.setItemAsync(key, value);
    } catch {
      // best-effort — surface via logging infra in production
    }
  },
  async remove(key: SecureStorageKey): Promise<void> {
    try {
      await SecureStore.deleteItemAsync(key);
    } catch {
      // best-effort
    }
  },
};

export type SecureStorageService = typeof secureStorage;
