"use client";

import "./globals.css";
import { useEffect } from "react";
import { usePathname } from "next/navigation";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useAdminAuth } from "@/lib/auth";
import { useTheme } from "@/lib/theme";
import AdminSidebar from "@/components/layout/AdminSidebar";
import AdminHeader from "@/components/layout/AdminHeader";
import GlobalToasts from "@/components/shared/GlobalToasts";
import CommandPalette from "@/components/shared/CommandPalette";
import ReadOnlyBanner from "@/components/shared/ReadOnlyBanner";
import { Metadata } from "next";

const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 30_000, retry: 1 } },
});

function ThemeInit() {
  const init = useTheme((s) => s.init);
  useEffect(() => { init(); }, [init]);
  return null;
}

function AuthGate({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading, restore } = useAdminAuth();
  const pathname = usePathname();

  useEffect(() => { restore(); }, [restore]);

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center" style={{ background: "var(--bg-root)" }}>
        <div className="animate-spin w-8 h-8 border-4 rounded-full" style={{ borderColor: "var(--color-brand)", borderTopColor: "transparent" }} />
      </div>
    );
  }

  if (!isAuthenticated && pathname !== "/login" && pathname !== "/register") {
    if (typeof window !== "undefined") window.location.href = "/login";
    return null;
  }

  if (pathname === "/login" || pathname === "/register") {
    return <>{children}</>;
  }

  return (
    <div className="flex min-h-screen" style={{ background: "var(--bg-root)" }}>
      <AdminSidebar />
      <div className="flex-1 flex flex-col min-w-0 transition-all duration-200" style={{ marginLeft: "var(--sidebar-width)" }}>
        <AdminHeader />
        <ReadOnlyBanner />
        <main className="flex-1 p-6 overflow-auto">{children}</main>
      </div>
    </div>
  );
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <QueryClientProvider client={queryClient}>
          <ThemeInit />
          <GlobalToasts />
          <CommandPalette />
          <AuthGate>{children}</AuthGate>
        </QueryClientProvider>
      </body>
    </html>
  );
}
