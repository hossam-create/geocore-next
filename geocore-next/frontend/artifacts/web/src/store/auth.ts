import { create } from "zustand";
import api from "@/lib/api";

interface AuthUser {
  id: string;
  name: string;
  email: string;
  phone?: string;
  location?: string;
  rating?: number;
  balance?: number;
  isVerified?: boolean;
}

interface AuthStore {
  user: AuthUser | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (name: string, email: string, phone: string, password: string) => Promise<void>;
  logout: () => void;
  restoreSession: () => void;
}


export const useAuthStore = create<AuthStore>((set) => ({
  user: null,
  isAuthenticated: false,

  restoreSession: () => {
    const raw = localStorage.getItem("auth_user");
    const token = localStorage.getItem("access_token");
    if (raw && token) {
      try {
        set({ user: JSON.parse(raw), isAuthenticated: true });
      } catch {
        localStorage.removeItem("auth_user");
      }
    }
  },

  login: async (email, password) => {
    const { data } = await api.post("/auth/login", { email, password });
    const payload = data.data;
    const user: AuthUser = payload.user;
    const access_token: string = payload.access_token ?? payload.token ?? "";
    const refresh_token: string = payload.refresh_token ?? "";
    localStorage.setItem("access_token", access_token);
    if (refresh_token) localStorage.setItem("refresh_token", refresh_token);
    localStorage.setItem("auth_user", JSON.stringify(user));
    set({ user, isAuthenticated: true });
  },

  register: async (name, email, phone, password) => {
    const { data } = await api.post("/auth/register", { name, email, phone, password });
    const payload = data.data;
    const user: AuthUser = payload.user;
    const access_token: string = payload.access_token ?? payload.token ?? "";
    const refresh_token: string = payload.refresh_token ?? "";
    localStorage.setItem("access_token", access_token);
    if (refresh_token) localStorage.setItem("refresh_token", refresh_token);
    localStorage.setItem("auth_user", JSON.stringify(user));
    set({ user, isAuthenticated: true });
  },

  logout: () => {
    localStorage.removeItem("access_token");
    localStorage.removeItem("refresh_token");
    localStorage.removeItem("auth_user");
    set({ user: null, isAuthenticated: false });
  },
}));
