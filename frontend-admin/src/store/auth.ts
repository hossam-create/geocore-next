import { create } from "zustand"
import { api } from "@/api/client"
import { isInternalRole } from "@/lib/permissions"

interface AdminUser {
  id: string
  name: string
  email: string
  role: string
}

interface AuthState {
  user: AdminUser | null
  token: string | null
  isLoading: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => void
  restore: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  token: null,
  isLoading: false,

  restore: () => {
    const token = localStorage.getItem("admin_token")
    const raw = localStorage.getItem("admin_user")
    if (token && raw) {
      try {
        const user = JSON.parse(raw) as AdminUser
        set({ token, user })
      } catch {
        localStorage.removeItem("admin_token")
        localStorage.removeItem("admin_user")
      }
    }
  },

  login: async (email, password) => {
    set({ isLoading: true })
    try {
      const { data } = await api.post("/auth/login", { email, password })
      const token: string = data.data?.token ?? data.token
      const user: AdminUser = data.data?.user ?? data.user
      if (!user || !isInternalRole(user.role)) {
        throw new Error("Access denied: admin role required")
      }
      localStorage.setItem("admin_token", token)
      localStorage.setItem("admin_user", JSON.stringify(user))
      set({ token, user, isLoading: false })
    } catch (err) {
      set({ isLoading: false })
      throw err
    }
  },

  logout: () => {
    localStorage.removeItem("admin_token")
    localStorage.removeItem("admin_user")
    set({ user: null, token: null })
  },
}))
