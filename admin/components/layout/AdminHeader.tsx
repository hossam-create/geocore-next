"use client";

import { useEffect, useCallback } from "react";
import { usePathname } from "next/navigation";
import { useAdminAuth } from "@/lib/auth";
import { useTheme } from "@/lib/theme";
import { useCommandPalette } from "@/lib/commandPalette";
import { Bell, Search, Sun, Moon, Command, AlertCircle, Clock, Shield } from "lucide-react";

function useBreadcrumb() {
  const pathname = usePathname();
  const segments = pathname.split("/").filter(Boolean);
  return segments.map((s, i) => ({
    label: s.replace(/-/g, " ").replace(/\b\w/g, (c) => c.toUpperCase()),
    href: "/" + segments.slice(0, i + 1).join("/"),
    isLast: i === segments.length - 1,
  }));
}

export default function AdminHeader() {
  const { user } = useAdminAuth();
  const { dark, toggle } = useTheme();
  const openPalette = useCommandPalette((s) => s.open);
  const crumbs = useBreadcrumb();

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if ((e.metaKey || e.ctrlKey) && e.key === "k") {
      e.preventDefault();
      openPalette();
    }
  }, [openPalette]);

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  return (
    <header
      className="sticky top-0 z-20 flex items-center justify-between px-6 shrink-0"
      style={{
        height: "var(--topbar-height)",
        background: "var(--topbar-bg)",
        borderBottom: "1px solid var(--topbar-border)",
      }}
    >
      {/* Left: Breadcrumb + Search trigger */}
      <div className="flex items-center gap-4">
        {/* Breadcrumb */}
        <nav className="flex items-center gap-1 text-[13px]">
          {crumbs.map((c, i) => (
            <span key={c.href} className="flex items-center gap-1">
              {i > 0 && <span style={{ color: "var(--text-tertiary)" }}>/</span>}
              <span
                className={c.isLast ? "font-semibold" : ""}
                style={{ color: c.isLast ? "var(--text-primary)" : "var(--text-tertiary)" }}
              >
                {c.label}
              </span>
            </span>
          ))}
        </nav>

        {/* Search trigger (opens Cmd+K) */}
        <button
          onClick={openPalette}
          className="flex items-center gap-2 pl-3 pr-2 py-1.5 rounded-lg text-sm transition-all"
          style={{
            background: "var(--bg-surface-active)",
            border: "1px solid var(--border-default)",
            color: "var(--text-tertiary)",
            minWidth: "220px",
          }}
        >
          <Search className="w-3.5 h-3.5" />
          <span className="flex-1 text-left text-[13px]">Search...</span>
          <kbd
            className="flex items-center gap-0.5 text-[10px] font-medium px-1.5 py-0.5 rounded"
            style={{ background: "var(--bg-inset)", color: "var(--text-tertiary)" }}
          >
            <Command className="w-3 h-3" />K
          </kbd>
        </button>
      </div>

      {/* Right: Actions */}
      <div className="flex items-center gap-1.5">
        {/* Alert badges */}
        <button
          className="relative p-2 rounded-lg transition-colors"
          style={{ color: "var(--text-tertiary)" }}
          title="Pending Decisions"
          onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-surface-active)"; }}
          onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
        >
          <Shield className="w-4 h-4" />
          <span className="absolute -top-0.5 -right-0.5 min-w-[16px] h-4 flex items-center justify-center text-[9px] font-bold text-white rounded-full px-1" style={{ background: "var(--color-warning)" }}>
            3
          </span>
        </button>

        <button
          className="relative p-2 rounded-lg transition-colors"
          style={{ color: "var(--text-tertiary)" }}
          title="Pending Listings"
          onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-surface-active)"; }}
          onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
        >
          <Clock className="w-4 h-4" />
          <span className="absolute -top-0.5 -right-0.5 min-w-[16px] h-4 flex items-center justify-center text-[9px] font-bold text-white rounded-full px-1" style={{ background: "var(--color-brand)" }}>
            7
          </span>
        </button>

        {/* Notifications */}
        <button
          className="relative p-2 rounded-lg transition-colors"
          style={{ color: "var(--text-tertiary)" }}
          title="Notifications"
          onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-surface-active)"; }}
          onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
        >
          <Bell className="w-4 h-4" />
          <span className="absolute top-1 right-1 w-2 h-2 rounded-full" style={{ background: "var(--color-danger)" }} />
        </button>

        {/* Dark mode toggle */}
        <button
          onClick={toggle}
          className="p-2 rounded-lg transition-colors"
          style={{ color: "var(--text-tertiary)" }}
          title={dark ? "Light mode" : "Dark mode"}
          onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-surface-active)"; }}
          onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
        >
          {dark ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
        </button>

        {/* Separator */}
        <div className="w-px h-6 mx-1" style={{ background: "var(--border-default)" }} />

        {/* User */}
        <div className="flex items-center gap-2.5 pl-1">
          <div
            className="w-8 h-8 rounded-full flex items-center justify-center text-white text-xs font-bold"
            style={{ background: "var(--color-brand)" }}
          >
            {user?.name?.charAt(0)?.toUpperCase() ?? "A"}
          </div>
          <div className="text-sm hidden lg:block">
            <p className="font-medium leading-tight" style={{ color: "var(--text-primary)" }}>
              {user?.name ?? "Admin"}
            </p>
            <p className="text-[11px] capitalize" style={{ color: "var(--text-tertiary)" }}>
              {user?.role?.replace(/_/g, " ") ?? "admin"}
            </p>
          </div>
        </div>
      </div>
    </header>
  );
}
