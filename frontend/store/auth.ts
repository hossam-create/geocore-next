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
  role?: string;
}

interface AuthStore {
  user: AuthUser | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (name: string, email: string, phone: string, password: string) => Promise<void>;
  logout: () => void;
  restoreSession: () => void;
}

// ── Demo/mock users for offline / development use ────────────────────────────
const MOCK_USERS: Array<{ email: string; password: string; user: AuthUser }> = [
  {
    email: "demo@mnbarh.com",
    password: "demo1234",
    user: {
      id: "usr_demo_001",
      name: "Ahmed Al-Rashid",
      email: "demo@mnbarh.com",
      phone: "+971501234567",
      location: "Dubai, UAE",
      rating: 4.8,
      balance: 5000,
      isVerified: true,
    },
  },
  {
    email: "seller@mnbarh.com",
    password: "seller123",
    user: {
      id: "usr_demo_002",
      name: "Sara Mohammed",
      email: "seller@mnbarh.com",
      phone: "+966501234567",
      location: "Riyadh, KSA",
      rating: 4.6,
      balance: 12500,
      isVerified: true,
    },
  },
  {
    email: "test@test.com",
    password: "test123",
    user: {
      id: "usr_demo_003",
      name: "Test User",
      email: "test@test.com",
      phone: "+97150000000",
      location: "Abu Dhabi, UAE",
      rating: 4.0,
      balance: 1000,
      isVerified: false,
    },
  },
];

function mockToken() {
  const chars = "abcdef0123456789";
  return "mock_" + Array.from({ length: 32 }, () => chars[Math.floor(Math.random() * chars.length)]).join("");
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
    try {
      const { data } = await api.post("/auth/login", { email, password });
      const { user, access_token, refresh_token } = data.data;
      localStorage.setItem("access_token", access_token);
      localStorage.setItem("refresh_token", refresh_token);
      localStorage.setItem("auth_user", JSON.stringify(user));
      set({ user, isAuthenticated: true });
    } catch {
      // Fallback: check mock users (dev / demo mode)
      const match = MOCK_USERS.find(
        (u) => u.email.toLowerCase() === email.toLowerCase() && u.password === password
      );
      if (match) {
        const access_token = mockToken();
        const refresh_token = mockToken();
        localStorage.setItem("access_token", access_token);
        localStorage.setItem("refresh_token", refresh_token);
        localStorage.setItem("auth_user", JSON.stringify(match.user));
        set({ user: match.user, isAuthenticated: true });
        return;
      }
      throw new Error("Invalid credentials");
    }
  },

  register: async (name, email, phone, password) => {
    try {
      const { data } = await api.post("/auth/register", { name, email, phone, password });
      const { user, access_token, refresh_token } = data.data;
      localStorage.setItem("access_token", access_token);
      localStorage.setItem("refresh_token", refresh_token);
      localStorage.setItem("auth_user", JSON.stringify(user));
      set({ user, isAuthenticated: true });
    } catch {
      // Fallback: create a local demo user
      const user: AuthUser = {
        id: `usr_${Date.now()}`,
        name,
        email,
        phone,
        location: "GCC",
        rating: 0,
        balance: 0,
        isVerified: false,
      };
      const access_token = mockToken();
      const refresh_token = mockToken();
      localStorage.setItem("access_token", access_token);
      localStorage.setItem("refresh_token", refresh_token);
      localStorage.setItem("auth_user", JSON.stringify(user));
      set({ user, isAuthenticated: true });
    }
  },

  logout: () => {
    localStorage.removeItem("access_token");
    localStorage.removeItem("refresh_token");
    localStorage.removeItem("auth_user");
    set({ user: null, isAuthenticated: false });
  },
}));
