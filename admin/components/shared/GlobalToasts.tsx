"use client";

import { CheckCircle2, AlertCircle, Info, X } from "lucide-react";
import { useToastStore } from "@/lib/toast";

const TYPE_STYLES = {
  success: {
    icon: CheckCircle2,
    iconColor: "var(--color-success)",
    border: "var(--color-success)",
  },
  error: {
    icon: AlertCircle,
    iconColor: "var(--color-danger)",
    border: "var(--color-danger)",
  },
  info: {
    icon: Info,
    iconColor: "var(--color-info)",
    border: "var(--color-info)",
  },
} as const;

export default function GlobalToasts() {
  const toasts = useToastStore((s) => s.toasts);
  const removeToast = useToastStore((s) => s.removeToast);

  if (toasts.length === 0) return null;

  return (
    <div className="fixed top-4 right-4 z-[100] flex flex-col gap-2 w-[340px] max-w-[calc(100vw-2rem)] pointer-events-none">
      {toasts.map((toast) => {
        const style = TYPE_STYLES[toast.type];
        const Icon = style.icon;
        return (
          <div
            key={toast.id}
            className="surface pointer-events-auto p-3"
            style={{ borderLeft: `3px solid ${style.border}` }}
          >
            <div className="flex items-start gap-2">
              <Icon className="w-4 h-4 mt-0.5" style={{ color: style.iconColor }} />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium" style={{ color: "var(--text-primary)" }}>
                  {toast.title}
                </p>
                {toast.message ? (
                  <p className="text-xs mt-0.5" style={{ color: "var(--text-secondary)" }}>
                    {toast.message}
                  </p>
                ) : null}
              </div>
              <button
                className="p-1 rounded"
                style={{ color: "var(--text-tertiary)" }}
                onClick={() => removeToast(toast.id)}
                aria-label="Dismiss notification"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        );
      })}
    </div>
  );
}
