"use client";

import { useState, useRef, useEffect } from "react";
import { MoreVertical } from "lucide-react";

interface ActionItem {
  label: string;
  icon?: React.ReactNode;
  onClick: () => void;
  variant?: "default" | "danger";
  disabled?: boolean;
}

interface ActionMenuProps {
  actions: ActionItem[];
  align?: "left" | "right";
}

export default function ActionMenu({ actions, align = "right" }: ActionMenuProps) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  if (actions.length === 0) return null;

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={(e) => { e.stopPropagation(); setOpen(!open); }}
        className="p-1.5 rounded-md transition-colors"
        style={{ color: "var(--text-tertiary)" }}
        onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-inset)"; }}
        onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
        title="Actions"
      >
        <MoreVertical className="w-4 h-4" />
      </button>

      {open && (
        <div
          className="absolute top-full mt-1 z-50 min-w-[160px] rounded-lg shadow-lg py-1"
          style={{
            background: "var(--bg-surface)",
            border: "1px solid var(--border-default)",
            [align === "right" ? "right" : "left"]: 0,
          }}
        >
          {actions.map((action) => (
            <button
              key={action.label}
              onClick={(e) => {
                e.stopPropagation();
                setOpen(false);
                action.onClick();
              }}
              disabled={action.disabled}
              className="flex items-center gap-2 w-full px-3 py-2 text-left text-sm transition-colors disabled:opacity-50"
              style={{
                color: action.variant === "danger" ? "var(--color-danger)" : "var(--text-primary)",
              }}
              onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-inset)"; }}
              onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
            >
              {action.icon && <span className="shrink-0">{action.icon}</span>}
              {action.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
