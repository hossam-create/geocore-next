"use client";

import { create } from "zustand";

export type ToastType = "success" | "error" | "info";

export type ToastInput = {
  title: string;
  message?: string;
  type?: ToastType;
  durationMs?: number;
};

type ToastItem = {
  id: string;
  title: string;
  message?: string;
  type: ToastType;
};

type ToastStore = {
  toasts: ToastItem[];
  showToast: (input: ToastInput) => void;
  removeToast: (id: string) => void;
  clearToasts: () => void;
};

export const useToastStore = create<ToastStore>((set) => ({
  toasts: [],
  showToast: ({ title, message, type = "info", durationMs = 3500 }) => {
    const id = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
    const toast: ToastItem = { id, title, message, type };
    set((state) => ({ toasts: [...state.toasts, toast] }));

    if (typeof window !== "undefined") {
      window.setTimeout(() => {
        set((state) => ({ toasts: state.toasts.filter((x) => x.id !== id) }));
      }, durationMs);
    }
  },
  removeToast: (id) =>
    set((state) => ({ toasts: state.toasts.filter((x) => x.id !== id) })),
  clearToasts: () => set({ toasts: [] }),
}));
