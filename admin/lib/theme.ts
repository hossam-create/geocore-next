"use client";

import { create } from "zustand";

interface ThemeState {
  dark: boolean;
  toggle: () => void;
  init: () => void;
}

export const useTheme = create<ThemeState>((set, get) => ({
  dark: false,

  toggle: () => {
    const next = !get().dark;
    set({ dark: next });
    if (next) {
      document.documentElement.classList.add("dark");
      localStorage.setItem("theme", "dark");
    } else {
      document.documentElement.classList.remove("dark");
      localStorage.setItem("theme", "light");
    }
  },

  init: () => {
    try {
      const stored = localStorage.getItem("theme");
      const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
      const dark = stored === "dark" || (!stored && prefersDark);
      set({ dark });
      if (dark) document.documentElement.classList.add("dark");
      else document.documentElement.classList.remove("dark");
    } catch {
      /* SSR safe */
    }
  },
}));
