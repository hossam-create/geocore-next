"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { userFieldsApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import DataTable from "@/components/shared/DataTable";
import StatusBadge from "@/components/shared/StatusBadge";
import { Plus, Pencil, Trash2, X, Check } from "lucide-react";

interface UserField {
  id: number;
  name: string;
  label: string;
  label_en: string;
  field_type: string;
  options: string;
  is_required: boolean;
  placeholder: string;
  sort_order: number;
  is_active: boolean;
}

const EMPTY: UserField = { id: 0, name: "", label: "", label_en: "", field_type: "text", options: "[]", is_required: false, placeholder: "", sort_order: 0, is_active: true };
const FIELD_TYPES = ["text", "number", "select", "boolean", "date", "url"];

export default function UserFieldsPage() {
  const qc = useQueryClient();
  const [editing, setEditing] = useState<UserField | null>(null);
  const [creating, setCreating] = useState(false);

  const { data = [], isLoading } = useQuery({ queryKey: ["user-fields"], queryFn: userFieldsApi.list });

  const createMut = useMutation({
    mutationFn: (d: Record<string, unknown>) => userFieldsApi.create(d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["user-fields"] }); setCreating(false); setEditing(null); },
  });
  const updateMut = useMutation({
    mutationFn: ({ id, ...d }: Record<string, unknown>) => userFieldsApi.update(id as number, d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["user-fields"] }); setEditing(null); },
  });
  const deleteMut = useMutation({
    mutationFn: (id: number) => userFieldsApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["user-fields"] }),
  });

  const save = () => {
    if (!editing) return;
    if (creating) createMut.mutate(editing as unknown as Record<string, unknown>);
    else updateMut.mutate(editing as unknown as Record<string, unknown>);
  };

  const columns = [
    { key: "name", label: "Name" },
    { key: "label", label: "Label" },
    { key: "field_type", label: "Type", render: (r: UserField) => <StatusBadge status={r.field_type} variant="info" /> },
    { key: "is_required", label: "Required", render: (r: UserField) => r.is_required ? <StatusBadge status="required" variant="warning" /> : "Optional" },
    { key: "is_active", label: "Active", render: (r: UserField) => r.is_active ? <StatusBadge status="active" /> : <StatusBadge status="inactive" variant="neutral" /> },
    { key: "sort_order", label: "Order" },
    { key: "actions", label: "", render: (r: UserField) => (
      <div className="flex gap-1">
        <button onClick={() => { setEditing(r); setCreating(false); }} className="p-1 hover:bg-slate-100 rounded"><Pencil className="w-3.5 h-3.5 text-slate-500" /></button>
        <button onClick={() => { if (confirm("Delete?")) deleteMut.mutate(r.id); }} className="p-1 hover:bg-red-50 rounded"><Trash2 className="w-3.5 h-3.5 text-red-400" /></button>
      </div>
    )},
  ];

  return (
    <div>
      <PageHeader
        title="User Custom Fields"
        description="Add custom profile fields for users"
        actions={<button onClick={() => { setEditing({ ...EMPTY }); setCreating(true); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Plus className="w-4 h-4" /> Add Field</button>}
      />

      {editing && (
        <div className="p-4 mb-4 rounded-xl border border-slate-200 bg-white">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold text-sm">{creating ? "New Field" : "Edit Field"}</h3>
            <button onClick={() => { setEditing(null); setCreating(false); }}><X className="w-4 h-4 text-slate-400" /></button>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <input placeholder="Name (key)" value={editing.name} onChange={(e) => setEditing({ ...editing, name: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <input placeholder="Label (AR)" value={editing.label} onChange={(e) => setEditing({ ...editing, label: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <input placeholder="Label (EN)" value={editing.label_en} onChange={(e) => setEditing({ ...editing, label_en: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <select value={editing.field_type} onChange={(e) => setEditing({ ...editing, field_type: e.target.value })} className="border rounded-lg px-3 py-2 text-sm">
              {FIELD_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
            </select>
            <input placeholder="Placeholder" value={editing.placeholder} onChange={(e) => setEditing({ ...editing, placeholder: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <input type="number" placeholder="Sort Order" value={editing.sort_order} onChange={(e) => setEditing({ ...editing, sort_order: +e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={editing.is_required} onChange={(e) => setEditing({ ...editing, is_required: e.target.checked })} /> Required</label>
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={editing.is_active} onChange={(e) => setEditing({ ...editing, is_active: e.target.checked })} /> Active</label>
          </div>
          <div className="mt-3 flex gap-2">
            <button onClick={save} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Check className="w-4 h-4" /> Save</button>
            <button onClick={() => { setEditing(null); setCreating(false); }} className="px-3 py-1.5 rounded-lg text-sm border">Cancel</button>
          </div>
        </div>
      )}

      <DataTable columns={columns} data={Array.isArray(data) ? data : []} isLoading={isLoading} emptyMessage="No custom fields yet." rowKey={(r: UserField) => String(r.id)} />
    </div>
  );
}
