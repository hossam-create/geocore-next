"use client";

import clsx from "clsx";

type Variant = "success" | "warning" | "danger" | "info" | "neutral" | "brand";

const STYLES: Record<Variant, { bg: string; color: string }> = {
  success: { bg: "var(--color-success-light)", color: "var(--color-success)" },
  warning: { bg: "var(--color-warning-light)", color: "var(--color-warning)" },
  danger:  { bg: "var(--color-danger-light)",  color: "var(--color-danger)" },
  info:    { bg: "var(--color-info-light)",    color: "var(--color-info)" },
  neutral: { bg: "var(--bg-inset)",            color: "var(--text-secondary)" },
  brand:   { bg: "var(--color-brand-light)",   color: "var(--color-brand)" },
};

const STATUS_MAP: Record<string, Variant> = {
  approved: "success", active: "success", resolved: "success", released: "success", completed: "success",
  pending: "warning", in_progress: "warning", waiting: "warning", scheduled: "warning",
  rejected: "danger", banned: "danger", failed: "danger", urgent: "danger", flagged: "danger", suspended: "danger",
  open: "info", live: "info", new: "info", under_review: "info", high: "warning", normal: "neutral", low: "neutral",
  closed: "neutral", expired: "neutral", cancelled: "neutral", draft: "neutral",
};

interface Props {
  status: string;
  variant?: Variant;
  dot?: boolean;
  className?: string;
}

export default function StatusBadge({ status, variant, dot, className }: Props) {
  const v = variant ?? STATUS_MAP[status.toLowerCase()] ?? "neutral";
  const s = STYLES[v];

  return (
    <span
      className={clsx("inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-semibold uppercase tracking-wide", className)}
      style={{ background: s.bg, color: s.color }}
    >
      {dot && <span className="w-1.5 h-1.5 rounded-full" style={{ background: s.color }} />}
      {status.replace(/_/g, " ")}
    </span>
  );
}
