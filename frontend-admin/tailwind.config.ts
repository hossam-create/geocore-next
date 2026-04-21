import type { Config } from "tailwindcss"

const config: Config = {
  darkMode: ["class"],
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: "#0071CE",
          foreground: "#ffffff",
        },
        secondary: {
          DEFAULT: "#FFC220",
          foreground: "#1A202C",
        },
        sidebar: {
          bg: "#1A1A2E",
          text: "#A0AEC0",
          active: "#0071CE",
        },
        background: "#F7FAFC",
        card: "#FFFFFF",
        border: "#E2E8F0",
        text: {
          DEFAULT: "#1A202C",
          muted: "#718096",
        },
        success: "#48BB78",
        warning: "#ECC94B",
        danger: "#FC8181",
      },
      borderRadius: {
        lg: "0.5rem",
        md: "calc(0.5rem - 2px)",
        sm: "calc(0.5rem - 4px)",
      },
    },
  },
  plugins: [],
}

export default config
