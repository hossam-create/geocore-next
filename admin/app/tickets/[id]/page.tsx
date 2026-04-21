"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ticketsApi, usersApi } from "@/lib/api";
import { useToastStore } from "@/lib/toast";
import { format } from "date-fns";
import { Ban, CreditCard, Trash2, StickyNote, User, ShieldAlert } from "lucide-react";
import type { SupportTicket } from "@/lib/types";
import StatusBadge from "@/components/shared/StatusBadge";

type UserProfile = {
  id: string;
  name: string;
  email: string;
  role: string;
  status: string;
  created_at: string;
  total_sales: number;
  flag_count: number;
  open_disputes: number;
};

function normalizeUserProfile(data: unknown): UserProfile | null {
  if (!data || typeof data !== "object") return null;
  const d = data as Record<string, unknown>;
  return {
    id: String(d.id ?? ""),
    name: String(d.name ?? d.username ?? "Unknown"),
    email: String(d.email ?? ""),
    role: String(d.role ?? "user"),
    status: String(d.status ?? "active"),
    created_at: String(d.created_at ?? d.joined ?? new Date().toISOString()),
    total_sales: Number(d.total_sales ?? 0),
    flag_count: Number(d.flag_count ?? 0),
    open_disputes: Number(d.open_disputes ?? 0),
  };
}

export default function TicketDetailPage() {
  const { id } = useParams<{ id: string }>();
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [reply, setReply] = useState("");
  const [internalNote, setInternalNote] = useState("");
  const [sending, setSending] = useState(false);
  const [sendingNote, setSendingNote] = useState(false);

  const { data: ticket, isLoading } = useQuery<SupportTicket>({
    queryKey: ["admin-ticket", id],
    queryFn: () => ticketsApi.get(id),
  });

  const userId = (ticket as unknown as Record<string, unknown>)?.user_id as string | undefined;

  const { data: rawUser } = useQuery({
    queryKey: ["admin-user-profile", userId],
    queryFn: async () => {
      if (!userId) return null;
      try { return await usersApi.get(userId); } catch { return null; }
    },
    enabled: !!userId,
  });

  const userProfile = normalizeUserProfile(rawUser);

  const handleReply = async () => {
    if (!reply.trim()) return;
    setSending(true);
    try {
      await ticketsApi.reply(id, reply);
      setReply("");
      qc.invalidateQueries({ queryKey: ["admin-ticket", id] });
      showToast({ type: "success", title: "Reply sent", message: "Your reply has been posted." });
    } catch {
      showToast({ type: "error", title: "Reply failed", message: "Could not send reply." });
    } finally {
      setSending(false);
    }
  };

  const handleInternalNote = async () => {
    if (!internalNote.trim()) return;
    setSendingNote(true);
    try {
      await ticketsApi.reply(id, `[INTERNAL NOTE] ${internalNote}`);
      setInternalNote("");
      qc.invalidateQueries({ queryKey: ["admin-ticket", id] });
      showToast({ type: "success", title: "Note added", message: "Internal note saved." });
    } catch {
      showToast({ type: "error", title: "Note failed", message: "Could not save internal note." });
    } finally {
      setSendingNote(false);
    }
  };

  const handleStatus = async (status: string) => {
    try {
      await ticketsApi.updateStatus(id, status);
      qc.invalidateQueries({ queryKey: ["admin-ticket", id] });
      showToast({ type: "success", title: "Status updated", message: `Ticket is now ${status}.` });
    } catch {
      showToast({ type: "error", title: "Update failed", message: "Could not change status." });
    }
  };

  const handleBanUser = async () => {
    if (!userId || !confirm("Ban this user? This action is reversible from the Users page.")) return;
    try {
      await usersApi.ban(userId);
      showToast({ type: "success", title: "User banned", message: "The user has been banned." });
    } catch {
      showToast({ type: "error", title: "Ban failed", message: "Could not ban user." });
    }
  };

  if (isLoading) return <div className="text-center py-16" style={{ color: "var(--text-tertiary)" }}>Loading...</div>;
  if (!ticket) return <div className="text-center py-16" style={{ color: "var(--text-tertiary)" }}>Ticket not found</div>;

  const accountAge = userProfile ? Math.round((Date.now() - new Date(userProfile.created_at).getTime()) / 86400000) : 0;

  return (
    <div className="flex gap-6">
      {/* Main Content */}
      <div className="flex-1 min-w-0 space-y-5">
        {/* Header */}
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-xl font-bold" style={{ color: "var(--text-primary)" }}>{ticket.subject}</h1>
            <div className="flex items-center gap-2 mt-1">
              <StatusBadge status={ticket.priority} dot />
              <StatusBadge status={ticket.status} />
              <span className="text-xs" style={{ color: "var(--text-tertiary)" }}>{ticket.user_name}</span>
            </div>
          </div>
          <div className="flex gap-2 flex-shrink-0">
            {ticket.status !== "resolved" && (
              <button onClick={() => handleStatus("resolved")} className="px-3 py-1.5 text-white text-xs rounded-lg font-medium" style={{ background: "var(--color-success)" }}>Resolve</button>
            )}
            {ticket.status !== "closed" && (
              <button onClick={() => handleStatus("closed")} className="px-3 py-1.5 text-white text-xs rounded-lg font-medium" style={{ background: "var(--color-danger)" }}>Close</button>
            )}
          </div>
        </div>

        {/* Conversation Thread */}
        <div className="surface rounded-xl divide-y" style={{ borderColor: "var(--border-default)" }}>
          {(ticket.messages ?? []).length === 0 && (
            <div className="p-6 text-center text-sm" style={{ color: "var(--text-tertiary)" }}>No messages yet.</div>
          )}
          {(ticket.messages ?? []).map((msg) => {
            const isInternal = msg.body?.startsWith("[INTERNAL NOTE]");
            return (
              <div key={msg.id} className="p-4" style={{ background: isInternal ? "rgba(245,158,11,0.05)" : msg.is_admin ? "rgba(59,130,246,0.04)" : undefined }}>
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-xs font-semibold" style={{ color: "var(--text-primary)" }}>{msg.sender_name ?? (msg.is_admin ? "Admin" : "User")}</span>
                  {msg.is_admin && !isInternal && <span className="px-1.5 py-0.5 text-[10px] rounded font-medium" style={{ background: "rgba(59,130,246,0.1)", color: "#3b82f6" }}>ADMIN</span>}
                  {isInternal && <span className="px-1.5 py-0.5 text-[10px] rounded font-medium" style={{ background: "rgba(245,158,11,0.15)", color: "#d97706" }}>INTERNAL</span>}
                  <span className="text-[10px]" style={{ color: "var(--text-tertiary)" }}>{format(new Date(msg.created_at), "MMM d, HH:mm")}</span>
                </div>
                <p className="text-sm whitespace-pre-wrap" style={{ color: "var(--text-secondary)" }}>
                  {isInternal ? msg.body.replace("[INTERNAL NOTE] ", "") : msg.body}
                </p>
              </div>
            );
          })}
        </div>

        {/* Reply Box */}
        <div className="surface rounded-xl p-4 space-y-3">
          <textarea
            value={reply}
            onChange={(e) => setReply(e.target.value)}
            rows={3}
            placeholder="Write a reply (visible to user)..."
            className="w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
            style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }}
          />
          <button onClick={handleReply} disabled={sending || !reply.trim()} className="px-4 py-2 text-white text-sm rounded-lg font-medium disabled:opacity-50" style={{ background: "var(--color-brand)" }}>
            {sending ? "Sending..." : "Send Reply"}
          </button>
        </div>

        {/* Internal Notes */}
        <div className="surface rounded-xl p-4 space-y-3" style={{ border: "1px dashed var(--color-warning)" }}>
          <div className="flex items-center gap-2">
            <StickyNote className="w-4 h-4" style={{ color: "var(--color-warning)" }} />
            <span className="text-xs font-semibold uppercase tracking-wider" style={{ color: "var(--color-warning)" }}>Internal Note (admin-only)</span>
          </div>
          <textarea
            value={internalNote}
            onChange={(e) => setInternalNote(e.target.value)}
            rows={2}
            placeholder="Add an internal note (never shown to user)..."
            className="w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-amber-400 resize-none"
            style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-primary)" }}
          />
          <button onClick={handleInternalNote} disabled={sendingNote || !internalNote.trim()} className="px-4 py-2 text-white text-sm rounded-lg font-medium disabled:opacity-50" style={{ background: "#d97706" }}>
            {sendingNote ? "Saving..." : "Save Note"}
          </button>
        </div>
      </div>

      {/* Sidebar: User Profile + Quick Actions */}
      <div className="w-72 flex-shrink-0 space-y-4">
        {/* User Profile */}
        <div className="surface rounded-xl p-4 space-y-3">
          <div className="flex items-center gap-2 mb-1">
            <User className="w-4 h-4" style={{ color: "var(--color-brand)" }} />
            <span className="text-xs font-semibold uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>User Profile</span>
          </div>
          {userProfile ? (
            <div className="space-y-2">
              <div>
                <p className="text-sm font-medium" style={{ color: "var(--text-primary)" }}>{userProfile.name}</p>
                <p className="text-xs" style={{ color: "var(--text-tertiary)" }}>{userProfile.email}</p>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div className="p-2 rounded-lg" style={{ background: "var(--bg-inset)" }}>
                  <p className="text-[10px] uppercase" style={{ color: "var(--text-tertiary)" }}>Account Age</p>
                  <p className="text-sm font-bold" style={{ color: "var(--text-primary)" }}>{accountAge}d</p>
                </div>
                <div className="p-2 rounded-lg" style={{ background: "var(--bg-inset)" }}>
                  <p className="text-[10px] uppercase" style={{ color: "var(--text-tertiary)" }}>Total Sales</p>
                  <p className="text-sm font-bold" style={{ color: "var(--text-primary)" }}>{userProfile.total_sales}</p>
                </div>
                <div className="p-2 rounded-lg" style={{ background: "var(--bg-inset)" }}>
                  <p className="text-[10px] uppercase" style={{ color: "var(--text-tertiary)" }}>Flags</p>
                  <p className="text-sm font-bold" style={{ color: userProfile.flag_count > 0 ? "var(--color-danger)" : "var(--text-primary)" }}>{userProfile.flag_count}</p>
                </div>
                <div className="p-2 rounded-lg" style={{ background: "var(--bg-inset)" }}>
                  <p className="text-[10px] uppercase" style={{ color: "var(--text-tertiary)" }}>Disputes</p>
                  <p className="text-sm font-bold" style={{ color: userProfile.open_disputes > 0 ? "var(--color-warning)" : "var(--text-primary)" }}>{userProfile.open_disputes}</p>
                </div>
              </div>
              <StatusBadge status={userProfile.status} dot />
            </div>
          ) : (
            <p className="text-xs" style={{ color: "var(--text-tertiary)" }}>User profile unavailable</p>
          )}
        </div>

        {/* Quick Actions */}
        <div className="surface rounded-xl p-4 space-y-2">
          <div className="flex items-center gap-2 mb-1">
            <ShieldAlert className="w-4 h-4" style={{ color: "var(--color-danger)" }} />
            <span className="text-xs font-semibold uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>Quick Actions</span>
          </div>
          <button onClick={handleBanUser} className="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-xs font-medium text-left transition-colors hover:opacity-80" style={{ background: "rgba(239,68,68,0.1)", color: "var(--color-danger)" }}>
            <Ban className="w-3.5 h-3.5" /> Ban User
          </button>
          <button className="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-xs font-medium text-left transition-colors hover:opacity-80" style={{ background: "rgba(59,130,246,0.1)", color: "#3b82f6" }}>
            <CreditCard className="w-3.5 h-3.5" /> Refund Payment
          </button>
          <button className="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-xs font-medium text-left transition-colors hover:opacity-80" style={{ background: "rgba(245,158,11,0.1)", color: "#d97706" }}>
            <Trash2 className="w-3.5 h-3.5" /> Remove Listing
          </button>
        </div>

        {/* Ticket Metadata */}
        <div className="surface rounded-xl p-4 space-y-2">
          <span className="text-xs font-semibold uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>Details</span>
          <div className="space-y-1.5">
            <div className="flex justify-between">
              <span className="text-xs" style={{ color: "var(--text-tertiary)" }}>Created</span>
              <span className="text-xs font-medium" style={{ color: "var(--text-secondary)" }}>{format(new Date(ticket.created_at), "MMM d, HH:mm")}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-xs" style={{ color: "var(--text-tertiary)" }}>Priority</span>
              <StatusBadge status={ticket.priority} dot />
            </div>
            <div className="flex justify-between">
              <span className="text-xs" style={{ color: "var(--text-tertiary)" }}>Status</span>
              <StatusBadge status={ticket.status} />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
