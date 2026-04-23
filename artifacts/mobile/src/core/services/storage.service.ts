import AsyncStorage from "@react-native-async-storage/async-storage";

import type { StorageKey } from "../constants/storageKeys";

export const storage = {
  async get(key: StorageKey): Promise<string | null> {
    try {
      return await AsyncStorage.getItem(key);
    } catch {
      return null;
    }
  },
  async set(key: StorageKey, value: string): Promise<void> {
    try {
      await AsyncStorage.setItem(key, value);
    } catch {
      // best-effort
    }
  },
  async remove(key: StorageKey): Promise<void> {
    try {
      await AsyncStorage.removeItem(key);
    } catch {
      // best-effort
    }
  },
  async getJson<T>(key: StorageKey): Promise<T | null> {
    const raw = await storage.get(key);
    if (!raw) return null;
    try {
      return JSON.parse(raw) as T;
    } catch {
      return null;
    }
  },
  async setJson<T>(key: StorageKey, value: T): Promise<void> {
    await storage.set(key, JSON.stringify(value));
  },
};

export type StorageService = typeof storage;
