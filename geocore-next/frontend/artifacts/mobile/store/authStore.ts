import * as SecureStore from "expo-secure-store";
import { create } from "zustand";

import { authAPI } from "@/utils/api";

export interface AuthUser {
  id: string;
  name: string;
  email: string;
  phone?: string;
  location?: string;
  avatar?: string;
  rating?: number;
  totalSales?: number;
  balance?: number;
  isVerified?: boolean;
}

interface AuthState {
  user: AuthUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  login: (email: string, password: string) => Promise<void>;
  register: (
    name: string,
    email: string,
    password: string,
    phone?: string
  ) => Promise<void>;
  socialLogin: (
    provider: "google" | "apple" | "facebook",
    token: string,
    name?: string,
    email?: string
  ) => Promise<void>;
  logout: () => Promise<void>;
  restoreSession: () => Promise<void>;
  clearError: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,
  isLoading: false,
  error: null,

  clearError: () => set({ error: null }),

  restoreSession: async () => {
    try {
      const token = await SecureStore.getItemAsync("access_token");
      if (!token) return;
      const { data } = await authAPI.me();
      const user = data?.data ?? data;
      set({ user, isAuthenticated: true });
    } catch {
      await SecureStore.deleteItemAsync("access_token").catch(() => {});
      await SecureStore.deleteItemAsync("refresh_token").catch(() => {});
    }
  },

  login: async (email, password) => {
    set({ isLoading: true, error: null });
    try {
      const { data } = await authAPI.login(email, password);
      const { access_token, refresh_token, user } = data?.data ?? data;
      await SecureStore.setItemAsync("access_token", access_token);
      await SecureStore.setItemAsync("refresh_token", refresh_token);
      set({ user, isAuthenticated: true, isLoading: false });
    } catch (err: any) {
      const msg =
        err?.response?.data?.message ??
        err?.message ??
        "Login failed. Please check your credentials.";
      set({ error: msg, isLoading: false });
      throw err;
    }
  },

  socialLogin: async (provider, token, name, email) => {
    set({ isLoading: true, error: null });
    try {
      const { data } = await authAPI.socialLogin(provider, token, name, email);
      const payload = data?.data ?? data;
      const accessToken = payload.access_token ?? payload.token;
      const refreshToken = payload.refresh_token ?? "";
      const user = payload.user;
      if (accessToken) {
        await SecureStore.setItemAsync("access_token", accessToken);
      }
      if (refreshToken) {
        await SecureStore.setItemAsync("refresh_token", refreshToken);
      }
      set({ user, isAuthenticated: true, isLoading: false });
    } catch (err: any) {
      const msg =
        err?.response?.data?.message ??
        err?.message ??
        `${provider} sign-in failed. Please try again.`;
      set({ error: msg, isLoading: false });
      throw err;
    }
  },

  register: async (name, email, password, phone) => {
    set({ isLoading: true, error: null });
    try {
      const { data } = await authAPI.register(name, email, password, phone);
      const { access_token, refresh_token, user } = data?.data ?? data;
      await SecureStore.setItemAsync("access_token", access_token);
      await SecureStore.setItemAsync("refresh_token", refresh_token);
      set({ user, isAuthenticated: true, isLoading: false });
    } catch (err: any) {
      const msg =
        err?.response?.data?.message ??
        err?.message ??
        "Registration failed. Please try again.";
      set({ error: msg, isLoading: false });
      throw err;
    }
  },

  logout: async () => {
    try {
      await authAPI.logout();
    } catch {}
    await SecureStore.deleteItemAsync("access_token").catch(() => {});
    await SecureStore.deleteItemAsync("refresh_token").catch(() => {});
    set({ user: null, isAuthenticated: false, error: null });
  },
}));
