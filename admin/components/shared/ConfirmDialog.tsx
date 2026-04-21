"use client";

import { useState } from "react";
import { AlertTriangle, X } from "lucide-react";

interface Props {
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: "danger" | "warning" | "info";
  requireReason?: boolean;
  reasonLabel?: string;
  onConfirm: (reason?: string) => void;
  onCancel: () => void;
  isLoading?: boolean;
}

const VARIANT_COLORS = {
  danger: { bg: "#fee2e2", icon: "#ef4444", btn: "#ef4444" },
  warning: { bg: "#fef3c7", icon: "#f59e0b", btn: "#f59e0b" },
  info: { bg: "#dbeafe", icon: "#3b82f6", btn: "#3b82f6" },
};

export default function ConfirmDialog({
  open, title, message, confirmLabel = "Confirm", cancelLabel = "Cancel",
  variant = "danger", requireReason = false, reasonLabel = "Reason",
  onConfirm, onCancel, isLoading,
}: Props) {
  const [reason, setReason] = useState("");
  const colors = VARIANT_COLORS[variant];

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4" style={{ background: "rgba(0,0,0,0.4)" }} onClick={onCancel}>
      <div className="bg-white rounded-xl shadow-xl max-w-md w-full p-6" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-start gap-3">
          <div className="w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0" style={{ background: colors.bg }}>
            <AlertTriangle className="w-5 h-5" style={{ color: colors.icon }} />
          </div>
          <div className="flex-1">
            <div className="flex items-center justify-between">
              <h3 className="text-base font-semibold text-slate-800">{title}</h3>
              <button onClick={onCancel} className="p-1 hover:bg-slate-100 rounded"><X className="w-4 h-4 text-slate-400" /></button>
            </div>
            <p className="text-sm text-slate-500 mt-1">{message}</p>
          </div>
        </div>

        {requireReason && (
          <div className="mt-4">
            <label className="block text-xs font-medium text-slate-600 mb-1">{reasonLabel}</label>
            <textarea
              rows={3}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder={`Enter ${reasonLabel.toLowerCase()}...`}
              className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-200"
            />
          </div>
        )}

        <div className="flex justify-end gap-2 mt-5">
          <button onClick={onCancel} className="px-4 py-2 text-sm font-medium rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50">{cancelLabel}</button>
          <button
            onClick={() => { onConfirm(requireReason ? reason : undefined); setReason(""); }}
            disabled={isLoading || (requireReason && !reason.trim())}
            className="px-4 py-2 text-sm font-medium rounded-lg text-white disabled:opacity-50"
            style={{ background: colors.btn }}
          >
            {isLoading ? "Processing..." : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
