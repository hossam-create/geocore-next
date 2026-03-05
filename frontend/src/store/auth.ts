import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { User } from '@/types'
import api from '@/lib/api'

interface AuthState {
  user: User | null
  token: string | null
  isLoading: boolean
  login: (email: string, password: string) => Promise<void>
  register: (name: string, email: string, password: string) => Promise<void>
  logout: () => void
  fetchMe: () => Promise<void>
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      token: null,
      isLoading: false,

      login: async (email, password) => {
        set({ isLoading: true })
        const res = await api.login({ email, password })
        const { token, user } = res.data.data
        localStorage.setItem('token', token)
        set({ user, token, isLoading: false })
      },

      register: async (name, email, password) => {
        set({ isLoading: true })
        const res = await api.register({ name, email, password })
        const { token, user } = res.data.data
        localStorage.setItem('token', token)
        set({ user, token, isLoading: false })
      },

      logout: () => {
        localStorage.removeItem('token')
        set({ user: null, token: null })
      },

      fetchMe: async () => {
        try {
          const res = await api.me()
          set({ user: res.data.data })
        } catch {
          get().logout()
        }
      },
    }),
    { name: 'auth-store', partialize: (s) => ({ token: s.token }) }
  )
)
