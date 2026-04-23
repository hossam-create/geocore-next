import { create } from "zustand";

import type { User } from "../../../domain/entities";
import type {
  LoginCredentials,
  RegisterPayload,
} from "../../../domain/repositories/auth.repository";
import {
  LoginUseCase,
  LogoutUseCase,
  RegisterUseCase,
  RestoreSessionUseCase,
} from "../../../domain/usecases/auth";
import { SECURE_STORAGE_KEYS } from "../../../core/constants/storageKeys";
import { AppError } from "../../../core/utils/errors";
import { secureStorage } from "../../../core/services/secure-storage.service";
import { analytics } from "../../../core/services/analytics.service";
import { socketService } from "../../../core/services/socket.service";
import { getContainer } from "../../../store/container";

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  isRestoring: boolean;
  error: string | null;
  login: (credentials: LoginCredentials) => Promise<boolean>;
  register: (payload: RegisterPayload) => Promise<boolean>;
  logout: () => Promise<void>;
  restoreSession: () => Promise<void>;
  clearError: () => void;
}

async function persistTokens(access: string, refresh: string): Promise<void> {
  await secureStorage.set(SECURE_STORAGE_KEYS.accessToken, access);
  await secureStorage.set(SECURE_STORAGE_KEYS.refreshToken, refresh);
}

function formatError(err: unknown): string {
  if (err instanceof AppError) return err.message;
  if (err instanceof Error) return err.message;
  return "Something went wrong";
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,
  isLoading: false,
  isRestoring: true,
  error: null,

  async login(credentials) {
    set({ isLoading: true, error: null });
    try {
      const useCase = new LoginUseCase(getContainer().auth);
      const session = await useCase.execute(credentials);
      await persistTokens(session.tokens.accessToken, session.tokens.refreshToken);
      socketService.connect(session.tokens.accessToken);
      analytics.identify(session.user.id, { email: session.user.email });
      set({
        user: session.user,
        isAuthenticated: true,
        isLoading: false,
        error: null,
      });
      return true;
    } catch (err) {
      set({ isLoading: false, error: formatError(err) });
      return false;
    }
  },

  async register(payload) {
    set({ isLoading: true, error: null });
    try {
      const useCase = new RegisterUseCase(getContainer().auth);
      const session = await useCase.execute(payload);
      await persistTokens(session.tokens.accessToken, session.tokens.refreshToken);
      socketService.connect(session.tokens.accessToken);
      analytics.identify(session.user.id, { email: session.user.email });
      set({
        user: session.user,
        isAuthenticated: true,
        isLoading: false,
        error: null,
      });
      return true;
    } catch (err) {
      set({ isLoading: false, error: formatError(err) });
      return false;
    }
  },

  async logout() {
    set({ isLoading: true });
    try {
      const useCase = new LogoutUseCase(getContainer().auth);
      await useCase.execute();
    } finally {
      socketService.disconnect();
      analytics.reset();
      set({
        user: null,
        isAuthenticated: false,
        isLoading: false,
        error: null,
      });
    }
  },

  async restoreSession() {
    set({ isRestoring: true });
    try {
      const useCase = new RestoreSessionUseCase(getContainer().auth);
      const user = await useCase.execute();
      if (user) {
        const token = await secureStorage.get(SECURE_STORAGE_KEYS.accessToken);
        if (token) socketService.connect(token);
        analytics.identify(user.id, { email: user.email });
        set({
          user,
          isAuthenticated: true,
          isRestoring: false,
        });
      } else {
        set({
          user: null,
          isAuthenticated: false,
          isRestoring: false,
        });
      }
    } catch {
      set({ user: null, isAuthenticated: false, isRestoring: false });
    }
  },

  clearError() {
    set({ error: null });
  },
}));
