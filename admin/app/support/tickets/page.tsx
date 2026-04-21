"use client";

import { useState, useMemo } from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import { mockTickets } from "@/lib/mockData";
import { ticketsApi } from "@/lib/api";
import { useToastStore } from "@/lib/toast";
import { Clock, CheckSquare, XSquare } from "lucide-react";

type TicketRow = {
  id: string;
  user: string;
  subject: string;
  priority: string;
  category: string;
  status: string;
  created: string;
};

const PRIORITY_ORDER: Record<string, number> = { urgent: 0, high: 1, normal: 2, medium: 2, low: 3 };
const CATEGORY_COLORS: Record<string, string> = {
  payment: "#ef4444",
  account: "#f59e0b",
  listing: "#3b82f6",
  shipping: "#8b5cf6",
  dispute: "#f97316",
  general: "#6b7280",
};

function normalizeTickets(payload: unknown): TicketRow[] {
  const box = payload as
    | { data?: Array<Record<string, unknown>>; meta?: unknown }
    | Array<Record<string, unknown>>
    | null
    | undefined;
  const rows = Array.isArray(box) ? box : Array.isArray(box?.data) ? box.data : [];

  return rows
    .map((item) => ({
      id: String(item.id ?? ""),
      user: String(item.user_name ?? item.user_id ?? "Unknown"),
      subject: String(item.subject ?? "No subject"),
      priority: String(item.priority ?? "medium"),
      category: String(item.category ?? "general"),
      status: String(item.status ?? "open"),
      created: String(item.created_at ?? new Date().toISOString()),
    }))
    .filter((x) => x.id);
}

export default function SupportTicketsPage() {
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [statusFilter, setStatusFilter] = useState("");
  const [priorityFilter, setPriorityFilter] = useState("");

  const { data: liveTickets, isLoading } = useQuery({
    queryKey: ["support", "tickets"],
    queryFn: async () => {
      const res = await ticketsApi.list();
      return normalizeTickets(res);
    },
    retry: 1,
  });

  const rawTickets: TicketRow[] = liveTickets?.length
    ? liveTickets
    : mockTickets.map((t) => ({
        id: t.id,
        user: t.user,
        subject: t.subject,
        priority: t.priority,
        category: "general",
        status: t.status,
        created: t.created,
      }));

  // Priority-sorted tickets (urgent → high → normal → low)
  const tickets = useMemo(() => {
    let filtered = rawTickets;
    if (statusFilter) filtered = filtered.filter((t) => t.status === statusFilter);
    if (priorityFilter) filtered = filtered.filter((t) => t.priority === priorityFilter);
    return [...filtered].sort((a, b) => (PRIORITY_ORDER[a.priority] ?? 9) - (PRIORITY_ORDER[b.priority] ?? 9));
  }, [rawTickets, statusFilter, priorityFilter]);

  const statusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      ticketsApi.updateStatus(id, status),
    onSuccess: (_, variables) => {
      qc.invalidateQueries({ queryKey: ["support", "tickets"] });
      showToast({ type: "success", title: "Ticket updated", message: `Status changed to ${variables.status}.` });
    },
    onError: (error: { message?: string }) => {
      showToast({ type: "error", title: "Update failed", message: error?.message ?? "Could not update ticket status." });
    },
  });

  // Bulk actions
  const handleBulkClose = async () => {
    if (selected.size === 0) return;
    try {
      await Promise.all(Array.from(selected).map((id) => ticketsApi.updateStatus(id, "closed")));
      qc.invalidateQueries({ queryKey: ["support", "tickets"] });
      setSelected(new Set());
      showToast({ type: "success", title: "Bulk close", message: `${selected.size} ticket(s) closed.` });
    } catch {
      showToast({ type: "error", title: "Bulk close failed", message: "Could not close all selected tickets." });
    }
  };

  const handleBulkPriority = async (priority: string) => {
    if (selected.size === 0) return;
    try {
      await Promise.all(Array.from(selected).map((id) => ticketsApi.updateStatus(id, priority)));
      qc.invalidateQueries({ queryKey: ["support", "tickets"] });
      setSelected(new Set());
      showToast({ type: "success", title: "Priority updated", message: `${selected.size} ticket(s) set to ${priority}.` });
    } catch {
      showToast({ type: "error", title: "Bulk update failed", message: "Could not update all selected tickets." });
    }
  };

  const toggleSelect = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  };

  const toggleAll = () => {
    if (selected.size === tickets.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(tickets.map((t) => t.id)));
    }
  };

  // SLA metrics
  const openCount = rawTickets.filter((t) => t.status === "open").length;
  const avgAge = rawTickets.length > 0
    ? Math.round(rawTickets.reduce((sum, t) => sum + (Date.now() - new Date(t.created).getTime()) / 3600000, 0) / rawTickets.length)
    : 0;
  const resolvedToday = rawTickets.filter((t) => {
    const d = new Date(t.created);
    const today = new Date();
    return t.status === "resolved" && d.toDateString() === today.toDateString();
  }).length;

  const mutationBusy = statusMutation.isPending;

  return (
    <div className="space-y-4">
      <PageHeader title="Support Tickets" description="Agent queue — sorted by priority with bulk actions and SLA tracking" />

      {/* SLA Metrics */}
      <div className="grid grid-cols-3 gap-3">
        <div className="surface p-3 rounded-lg">
          <div className="flex items-center gap-2 mb-1"><Clock className="w-4 h-4" style={{ color: "var(--color-warning)" }} /><span className="text-xs" style={{ color: "var(--text-tertiary)" }}>Open Tickets</span></div>
          <p className="text-lg font-bold" style={{ color: "var(--text-primary)" }}>{openCount}</p>
        </div>
        <div className="surface p-3 rounded-lg">
          <div className="flex items-center gap-2 mb-1"><Clock className="w-4 h-4" style={{ color: "var(--color-brand)" }} /><span className="text-xs" style={{ color: "var(--text-tertiary)" }}>Avg Response Time</span></div>
          <p className="text-lg font-bold" style={{ color: "var(--text-primary)" }}>{avgAge}h</p>
        </div>
        <div className="surface p-3 rounded-lg">
          <div className="flex items-center gap-2 mb-1"><CheckSquare className="w-4 h-4" style={{ color: "var(--color-success)" }} /><span className="text-xs" style={{ color: "var(--text-tertiary)" }}>Resolved Today</span></div>
          <p className="text-lg font-bold" style={{ color: "var(--text-primary)" }}>{resolvedToday}</p>
        </div>
      </div>

      {/* Filters + Bulk Actions */}
      <div className="flex flex-wrap items-center gap-3">
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="px-3 py-1.5 border rounded-lg text-sm"
          style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-secondary)" }}
        >
          <option value="">All Statuses</option>
          <option value="open">Open</option>
          <option value="in_progress">In Progress</option>
          <option value="waiting">Waiting</option>
          <option value="resolved">Resolved</option>
          <option value="closed">Closed</option>
        </select>
        <select
          value={priorityFilter}
          onChange={(e) => setPriorityFilter(e.target.value)}
          className="px-3 py-1.5 border rounded-lg text-sm"
          style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-secondary)" }}
        >
          <option value="">All Priorities</option>
          <option value="urgent">Urgent</option>
          <option value="high">High</option>
          <option value="normal">Normal</option>
          <option value="low">Low</option>
        </select>

        {selected.size > 0 && (
          <div className="flex items-center gap-2 ml-auto">
            <span className="text-xs font-medium" style={{ color: "var(--text-tertiary)" }}>{selected.size} selected</span>
            <button onClick={handleBulkClose} className="px-3 py-1.5 rounded-lg text-xs font-medium text-white" style={{ background: "var(--color-danger)" }}>
              <XSquare className="w-3 h-3 inline mr-1" />Close
            </button>
            <select
              onChange={(e) => { if (e.target.value) handleBulkPriority(e.target.value); e.target.value = ""; }}
              className="px-2 py-1.5 border rounded-lg text-xs"
              style={{ background: "var(--bg-surface)", borderColor: "var(--border-default)", color: "var(--text-secondary)" }}
              defaultValue=""
            >
              <option value="" disabled>Set Priority</option>
              <option value="urgent">Urgent</option>
              <option value="high">High</option>
              <option value="normal">Normal</option>
              <option value="low">Low</option>
            </select>
          </div>
        )}
      </div>

      {/* Table */}
      <div className="surface rounded-lg overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr style={{ borderBottom: "1px solid var(--border-default)" }}>
              <th className="px-3 py-3 w-8"><input type="checkbox" checked={selected.size === tickets.length && tickets.length > 0} onChange={toggleAll} /></th>
              <th className="text-left px-3 py-3 text-xs font-semibold uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>ID</th>
              <th className="text-left px-3 py-3 text-xs font-semibold uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>User</th>
              <th className="text-left px-3 py-3 text-xs font-semibold uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>Subject</th>
              <th className="text-left px-3 py-3 text-xs font-semibold uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>Category</th>
              <th className="text-left px-3 py-3 text-xs font-semibold uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>Priority</th>
              <th className="text-left px-3 py-3 text-xs font-semibold uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>Status</th>
              <th className="text-left px-3 py-3 text-xs font-semibold uppercase tracking-wider" style={{ color: "var(--text-tertiary)" }}>Created</th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <tr><td colSpan={8} className="px-4 py-8 text-center" style={{ color: "var(--text-tertiary)" }}>Loading tickets...</td></tr>
            ) : tickets.length === 0 ? (
              <tr><td colSpan={8} className="px-4 py-8 text-center" style={{ color: "var(--text-tertiary)" }}>No tickets in the queue.</td></tr>
            ) : (
              tickets.map((t) => (
                <tr key={t.id} className="hover:bg-black/[0.02] transition-colors" style={{ borderBottom: "1px solid var(--border-subtle)" }}>
                  <td className="px-3 py-3"><input type="checkbox" checked={selected.has(t.id)} onChange={() => toggleSelect(t.id)} /></td>
                  <td className="px-3 py-3"><Link href={`/tickets/${t.id}`} className="font-mono text-xs hover:underline" style={{ color: "var(--color-brand)" }}>{t.id}</Link></td>
                  <td className="px-3 py-3" style={{ color: "var(--text-secondary)" }}>{t.user}</td>
                  <td className="px-3 py-3 font-medium" style={{ color: "var(--text-primary)" }}>{t.subject}</td>
                  <td className="px-3 py-3">
                    <span className="inline-flex items-center gap-1 text-xs font-medium px-2 py-0.5 rounded-full" style={{ background: `${CATEGORY_COLORS[t.category] ?? CATEGORY_COLORS.general}18`, color: CATEGORY_COLORS[t.category] ?? CATEGORY_COLORS.general }}>
                      <span className="w-1.5 h-1.5 rounded-full" style={{ background: CATEGORY_COLORS[t.category] ?? CATEGORY_COLORS.general }} />
                      {t.category}
                    </span>
                  </td>
                  <td className="px-3 py-3"><StatusBadge status={t.priority} dot /></td>
                  <td className="px-3 py-3">
                    <select
                      className="text-xs px-2 py-1 rounded-md"
                      style={{ border: "1px solid var(--border-default)", background: "var(--bg-surface)" }}
                      defaultValue={t.status}
                      disabled={mutationBusy}
                      onChange={(e) => statusMutation.mutate({ id: t.id, status: e.target.value })}
                    >
                      <option value="open">open</option>
                      <option value="in_progress">in_progress</option>
                      <option value="waiting">waiting</option>
                      <option value="resolved">resolved</option>
                      <option value="closed">closed</option>
                    </select>
                  </td>
                  <td className="px-3 py-3 text-xs" style={{ color: "var(--text-tertiary)" }}>{new Date(t.created).toLocaleString()}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
