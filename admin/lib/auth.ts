import { create } from "zustand";
import api from "./api";
import { isInternalAdminRole } from "./adminAccess";

interface AdminUser {
  id: string;
  name: string;
  email: string;
  role: string;
}

interface AuthState {
  user: AdminUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  restore: () => void;
}

export const useAdminAuth = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,

  login: async (email, password) => {
    const res = await api.post("/auth/login", { email, password });
    const data = res.data?.data ?? res.data;
    const user = data.user;
    const token = data.access_token ?? data.token;

    if (!isInternalAdminRole(user.role)) {
      throw new Error("Admin access required");
    }

    localStorage.setItem("admin_token", token);
    if (data.refresh_token) {
      localStorage.setItem("admin_refresh_token", data.refresh_token);
    }
    localStorage.setItem("admin_user", JSON.stringify(user));
    set({ user, isAuthenticated: true, isLoading: false });
  },

  logout: () => {
    localStorage.removeItem("admin_token");
    localStorage.removeItem("admin_user");
    set({ user: null, isAuthenticated: false, isLoading: false });
    window.location.href = "/login";
  },

  restore: () => {
    try {
      const token = localStorage.getItem("admin_token");
      const raw = localStorage.getItem("admin_user");
      if (token && raw) {
        const user = JSON.parse(raw);
        set({ user, isAuthenticated: true, isLoading: false });
      } else {
        set({ isLoading: false });
      }
    } catch {
      set({ isLoading: false });
    }
  },
}));
