"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { announcementsApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import DataTable from "@/components/shared/DataTable";
import StatusBadge from "@/components/shared/StatusBadge";
import { Plus, Pencil, Trash2, X, Check } from "lucide-react";

interface Announcement {
  id: number;
  title: string;
  content: string;
  type: string;
  display_location: string;
  is_active: boolean;
  starts_at: string;
  ends_at: string;
}

const EMPTY: Announcement = { id: 0, title: "", content: "", type: "info", display_location: "homepage", is_active: true, starts_at: "", ends_at: "" };
const TYPES = ["info", "warning", "success", "error"];
const LOCATIONS = ["homepage", "all", "listing_form", "auction_form"];

export default function AnnouncementsPage() {
  const qc = useQueryClient();
  const [editing, setEditing] = useState<Announcement | null>(null);
  const [creating, setCreating] = useState(false);

  const { data = [], isLoading } = useQuery({ queryKey: ["announcements"], queryFn: announcementsApi.list });

  const createMut = useMutation({
    mutationFn: (d: Record<string, unknown>) => announcementsApi.create(d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["announcements"] }); setCreating(false); setEditing(null); },
  });
  const updateMut = useMutation({
    mutationFn: ({ id, ...d }: Record<string, unknown>) => announcementsApi.update(id as number, d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["announcements"] }); setEditing(null); },
  });
  const deleteMut = useMutation({
    mutationFn: (id: number) => announcementsApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["announcements"] }),
  });

  const save = () => {
    if (!editing) return;
    if (creating) createMut.mutate(editing as unknown as Record<string, unknown>);
    else updateMut.mutate(editing as unknown as Record<string, unknown>);
  };

  const columns = [
    { key: "title", label: "Title" },
    { key: "type", label: "Type", render: (r: Announcement) => <StatusBadge status={r.type} variant={r.type === "error" ? "danger" : r.type === "warning" ? "warning" : r.type === "success" ? "success" : "info"} /> },
    { key: "display_location", label: "Location", render: (r: Announcement) => <span className="text-xs text-slate-500">{r.display_location}</span> },
    { key: "is_active", label: "Active", render: (r: Announcement) => <StatusBadge status={r.is_active ? "active" : "inactive"} /> },
    { key: "actions", label: "", render: (r: Announcement) => (
      <div className="flex gap-1">
        <button onClick={() => { setEditing(r); setCreating(false); }} className="p-1 hover:bg-slate-100 rounded"><Pencil className="w-3.5 h-3.5 text-slate-500" /></button>
        <button onClick={() => { if (confirm("Delete?")) deleteMut.mutate(r.id); }} className="p-1 hover:bg-red-50 rounded"><Trash2 className="w-3.5 h-3.5 text-red-400" /></button>
      </div>
    )},
  ];

  return (
    <div>
      <PageHeader
        title="Announcements"
        description="Site-wide banners and notifications"
        actions={<button onClick={() => { setEditing({ ...EMPTY }); setCreating(true); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Plus className="w-4 h-4" /> New Announcement</button>}
      />

      {editing && (
        <div className="p-4 mb-4 rounded-xl border border-slate-200 bg-white">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold text-sm">{creating ? "New Announcement" : "Edit Announcement"}</h3>
            <button onClick={() => { setEditing(null); setCreating(false); }}><X className="w-4 h-4 text-slate-400" /></button>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <input placeholder="Title" value={editing.title} onChange={(e) => setEditing({ ...editing, title: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <select value={editing.type} onChange={(e) => setEditing({ ...editing, type: e.target.value })} className="border rounded-lg px-3 py-2 text-sm">
              {TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
            </select>
            <select value={editing.display_location} onChange={(e) => setEditing({ ...editing, display_location: e.target.value })} className="border rounded-lg px-3 py-2 text-sm">
              {LOCATIONS.map((l) => <option key={l} value={l}>{l}</option>)}
            </select>
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={editing.is_active} onChange={(e) => setEditing({ ...editing, is_active: e.target.checked })} /> Active</label>
          </div>
          <textarea rows={4} placeholder="Content" value={editing.content} onChange={(e) => setEditing({ ...editing, content: e.target.value })} className="w-full border rounded-lg px-3 py-2 text-sm mt-3" />
          <div className="mt-3 flex gap-2">
            <button onClick={save} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Check className="w-4 h-4" /> Save</button>
            <button onClick={() => { setEditing(null); setCreating(false); }} className="px-3 py-1.5 rounded-lg text-sm border">Cancel</button>
          </div>
        </div>
      )}

      <DataTable columns={columns} data={Array.isArray(data) ? data : []} isLoading={isLoading} emptyMessage="No announcements." rowKey={(r: Announcement) => String(r.id)} />
    </div>
  );
}
