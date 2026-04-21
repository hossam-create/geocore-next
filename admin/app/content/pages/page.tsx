"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { staticPagesApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import DataTable from "@/components/shared/DataTable";
import StatusBadge from "@/components/shared/StatusBadge";
import { Plus, Pencil, Trash2, X, Check } from "lucide-react";

interface StaticPage {
  id: number;
  title: string;
  slug: string;
  content: string;
  meta_title: string;
  meta_description: string;
  is_published: boolean;
  show_in_footer: boolean;
}

const EMPTY: StaticPage = { id: 0, title: "", slug: "", content: "", meta_title: "", meta_description: "", is_published: false, show_in_footer: false };

export default function StaticPagesPage() {
  const qc = useQueryClient();
  const [editing, setEditing] = useState<StaticPage | null>(null);
  const [creating, setCreating] = useState(false);

  const { data = [], isLoading } = useQuery({ queryKey: ["static-pages"], queryFn: staticPagesApi.list });

  const createMut = useMutation({
    mutationFn: (d: Record<string, unknown>) => staticPagesApi.create(d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["static-pages"] }); setCreating(false); setEditing(null); },
  });
  const updateMut = useMutation({
    mutationFn: ({ id, ...d }: Record<string, unknown>) => staticPagesApi.update(id as number, d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["static-pages"] }); setEditing(null); },
  });
  const deleteMut = useMutation({
    mutationFn: (id: number) => staticPagesApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["static-pages"] }),
  });

  const save = () => {
    if (!editing) return;
    if (creating) createMut.mutate(editing as unknown as Record<string, unknown>);
    else updateMut.mutate(editing as unknown as Record<string, unknown>);
  };

  const columns = [
    { key: "title", label: "Title" },
    { key: "slug", label: "Slug" },
    { key: "is_published", label: "Status", render: (r: StaticPage) => <StatusBadge status={r.is_published ? "published" : "draft"} variant={r.is_published ? "success" : "neutral"} /> },
    { key: "show_in_footer", label: "Footer", render: (r: StaticPage) => r.show_in_footer ? "Yes" : "—" },
    { key: "actions", label: "", render: (r: StaticPage) => (
      <div className="flex gap-1">
        <button onClick={() => { setEditing(r); setCreating(false); }} className="p-1 hover:bg-slate-100 rounded"><Pencil className="w-3.5 h-3.5 text-slate-500" /></button>
        <button onClick={() => { if (confirm("Delete?")) deleteMut.mutate(r.id); }} className="p-1 hover:bg-red-50 rounded"><Trash2 className="w-3.5 h-3.5 text-red-400" /></button>
      </div>
    )},
  ];

  return (
    <div>
      <PageHeader
        title="Static Pages"
        description="Manage CMS pages (about, privacy, terms, etc.)"
        actions={<button onClick={() => { setEditing({ ...EMPTY }); setCreating(true); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Plus className="w-4 h-4" /> New Page</button>}
      />

      {editing && (
        <div className="p-4 mb-4 rounded-xl border border-slate-200 bg-white">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold text-sm">{creating ? "New Page" : "Edit Page"}</h3>
            <button onClick={() => { setEditing(null); setCreating(false); }}><X className="w-4 h-4 text-slate-400" /></button>
          </div>
          <div className="space-y-3">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              <input placeholder="Title" value={editing.title} onChange={(e) => setEditing({ ...editing, title: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
              <input placeholder="Slug" value={editing.slug} onChange={(e) => setEditing({ ...editing, slug: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            </div>
            <textarea rows={8} placeholder="Page content (HTML)" value={editing.content} onChange={(e) => setEditing({ ...editing, content: e.target.value })} className="w-full border rounded-lg px-3 py-2 text-sm" />
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              <input placeholder="Meta Title" value={editing.meta_title} onChange={(e) => setEditing({ ...editing, meta_title: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
              <input placeholder="Meta Description" value={editing.meta_description} onChange={(e) => setEditing({ ...editing, meta_description: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            </div>
            <div className="flex gap-4">
              <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={editing.is_published} onChange={(e) => setEditing({ ...editing, is_published: e.target.checked })} /> Published</label>
              <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={editing.show_in_footer} onChange={(e) => setEditing({ ...editing, show_in_footer: e.target.checked })} /> Show in Footer</label>
            </div>
          </div>
          <div className="mt-3 flex gap-2">
            <button onClick={save} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Check className="w-4 h-4" /> Save</button>
            <button onClick={() => { setEditing(null); setCreating(false); }} className="px-3 py-1.5 rounded-lg text-sm border">Cancel</button>
          </div>
        </div>
      )}

      <DataTable columns={columns} data={Array.isArray(data) ? data : []} isLoading={isLoading} emptyMessage="No static pages." rowKey={(r: StaticPage) => String(r.id)} />
    </div>
  );
}
