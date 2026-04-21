import path from "path"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) return
          if (id.includes("recharts")) return "recharts"
          if (id.includes("d3-") || id.includes("victory-vendor") || id.includes("internmap")) return "charts-vendor"
          if (id.includes("@tanstack")) return "tanstack"
          if (id.includes("date-fns") || id.includes("lucide-react")) return "ui-vendor"
          return
        },
      },
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
})
