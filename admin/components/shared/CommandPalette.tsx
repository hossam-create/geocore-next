"use client";

import { useState, useEffect, useRef, useMemo } from "react";
import { useRouter } from "next/navigation";
import { useCommandPalette } from "@/lib/commandPalette";
import {
  Search,
  LayoutDashboard,
  Users,
  List,
  Gavel,
  ShoppingCart,
  Shield,
  ScrollText,
  Tag,
  Settings,
  Flag,
  BarChart3,
  TicketCheck,
  Store,
  Puzzle,
  FileText,
  DollarSign,
  Landmark,
  AlertTriangle,
  Briefcase,
  CreditCard,
  Mail,
  Megaphone,
  MapPin,
  Receipt,
} from "lucide-react";

interface CommandItem {
  label: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  section: string;
  keywords?: string;
}

const COMMANDS: CommandItem[] = [
  { label: "Dashboard", href: "/dashboard", icon: LayoutDashboard, section: "Overview" },
  { label: "Listings Queue", href: "/operations/listings", icon: List, section: "Operations", keywords: "moderation pending review" },
  { label: "Auctions Monitor", href: "/operations/auctions", icon: Gavel, section: "Operations", keywords: "bids live" },
  { label: "Orders", href: "/operations/orders", icon: ShoppingCart, section: "Operations" },
  { label: "Escrow Tracking", href: "/operations/escrow", icon: Landmark, section: "Operations", keywords: "funds release" },
  { label: "Decision Queue", href: "/custodii/decisions", icon: Shield, section: "Custodii", keywords: "approve reject" },
  { label: "Audit Logs", href: "/custodii/audit", icon: ScrollText, section: "Custodii", keywords: "history trail" },
  { label: "Users", href: "/admin/users", icon: Users, section: "Admin", keywords: "accounts members" },
  { label: "Feature Flags", href: "/admin/features", icon: Flag, section: "Admin", keywords: "toggle rollout" },
  { label: "Categories", href: "/admin/categories", icon: Tag, section: "Admin", keywords: "catalog" },
  { label: "Settings", href: "/admin/settings", icon: Settings, section: "Admin", keywords: "configuration" },
  { label: "Price Plans", href: "/pricing/plans", icon: DollarSign, section: "Pricing", keywords: "subscription" },
  { label: "Gateways", href: "/pricing/gateways", icon: CreditCard, section: "Pricing", keywords: "payment stripe" },
  { label: "Invoices", href: "/pricing/invoices", icon: Receipt, section: "Pricing", keywords: "billing" },
  { label: "Storefronts", href: "/storefronts", icon: Store, section: "Storefronts", keywords: "shops sellers" },
  { label: "Email Templates", href: "/content/emails", icon: Mail, section: "Content" },
  { label: "Static Pages", href: "/content/pages", icon: FileText, section: "Content" },
  { label: "Announcements", href: "/content/announcements", icon: Megaphone, section: "Content" },
  { label: "Geography", href: "/content/geography", icon: MapPin, section: "Content", keywords: "countries regions" },
  { label: "Analytics", href: "/analytics/overview", icon: BarChart3, section: "Analytics", keywords: "metrics kpi" },
  { label: "Tickets", href: "/support/tickets", icon: TicketCheck, section: "Support" },
  { label: "Disputes", href: "/support/disputes", icon: AlertTriangle, section: "Support" },
  { label: "Jobs", href: "/system/jobs", icon: Briefcase, section: "System", keywords: "background queue" },
  { label: "Addons", href: "/addons", icon: Puzzle, section: "Addons", keywords: "plugins marketplace extensions" },
];

export default function CommandPalette() {
  const { isOpen, close } = useCommandPalette();
  const router = useRouter();
  const [query, setQuery] = useState("");
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => {
    if (!query.trim()) return COMMANDS;
    const q = query.toLowerCase();
    return COMMANDS.filter(
      (c) =>
        c.label.toLowerCase().includes(q) ||
        c.section.toLowerCase().includes(q) ||
        c.keywords?.toLowerCase().includes(q)
    );
  }, [query]);

  useEffect(() => {
    if (isOpen) {
      setQuery("");
      setSelectedIndex(0);
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }, [isOpen]);

  useEffect(() => {
    setSelectedIndex(0);
  }, [query]);

  useEffect(() => {
    if (!isOpen) return;
    const handle = (e: KeyboardEvent) => {
      if (e.key === "Escape") { close(); return; }
      if (e.key === "ArrowDown") { e.preventDefault(); setSelectedIndex((i) => Math.min(i + 1, filtered.length - 1)); return; }
      if (e.key === "ArrowUp") { e.preventDefault(); setSelectedIndex((i) => Math.max(i - 1, 0)); return; }
      if (e.key === "Enter" && filtered[selectedIndex]) {
        e.preventDefault();
        router.push(filtered[selectedIndex].href);
        close();
      }
    };
    document.addEventListener("keydown", handle);
    return () => document.removeEventListener("keydown", handle);
  }, [isOpen, filtered, selectedIndex, close, router]);

  // Scroll selected into view
  useEffect(() => {
    if (!listRef.current) return;
    const el = listRef.current.children[selectedIndex] as HTMLElement;
    el?.scrollIntoView({ block: "nearest" });
  }, [selectedIndex]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh]">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={close} />

      {/* Panel */}
      <div
        className="relative w-full max-w-lg rounded-xl overflow-hidden"
        style={{
          background: "var(--bg-surface)",
          border: "1px solid var(--border-default)",
          boxShadow: "var(--shadow-lg)",
        }}
      >
        {/* Input */}
        <div className="flex items-center gap-3 px-4 py-3" style={{ borderBottom: "1px solid var(--border-default)" }}>
          <Search className="w-5 h-5 shrink-0" style={{ color: "var(--text-tertiary)" }} />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search pages, actions..."
            className="flex-1 bg-transparent outline-none text-[15px]"
            style={{ color: "var(--text-primary)" }}
          />
          <kbd
            className="text-[10px] font-medium px-1.5 py-0.5 rounded"
            style={{ background: "var(--bg-inset)", color: "var(--text-tertiary)" }}
          >
            ESC
          </kbd>
        </div>

        {/* Results */}
        <div ref={listRef} className="max-h-[320px] overflow-y-auto py-2 px-2">
          {filtered.length === 0 && (
            <p className="text-center py-8 text-sm" style={{ color: "var(--text-tertiary)" }}>
              No results for &quot;{query}&quot;
            </p>
          )}
          {filtered.map((item, i) => {
            const Icon = item.icon;
            const isSelected = i === selectedIndex;
            return (
              <button
                key={item.href}
                onClick={() => { router.push(item.href); close(); }}
                onMouseEnter={() => setSelectedIndex(i)}
                className="flex items-center gap-3 w-full px-3 py-2.5 rounded-lg text-left transition-colors"
                style={{
                  background: isSelected ? "var(--bg-surface-active)" : "transparent",
                  color: isSelected ? "var(--text-primary)" : "var(--text-secondary)",
                }}
              >
                <span style={{ color: isSelected ? "var(--color-brand)" : "var(--text-tertiary)" }}><Icon className="w-4 h-4 shrink-0" /></span>
                <div className="flex-1 min-w-0">
                  <span className="text-[13px] font-medium">{item.label}</span>
                </div>
                <span className="text-[11px] shrink-0" style={{ color: "var(--text-tertiary)" }}>{item.section}</span>
              </button>
            );
          })}
        </div>

        {/* Footer hints */}
        <div
          className="flex items-center gap-4 px-4 py-2 text-[11px]"
          style={{ borderTop: "1px solid var(--border-default)", color: "var(--text-tertiary)" }}
        >
          <span><kbd className="font-mono">↑↓</kbd> navigate</span>
          <span><kbd className="font-mono">↵</kbd> open</span>
          <span><kbd className="font-mono">esc</kbd> close</span>
        </div>
      </div>
    </div>
  );
}
