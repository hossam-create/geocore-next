"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listingExtrasApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import DataTable from "@/components/shared/DataTable";
import StatusBadge from "@/components/shared/StatusBadge";
import { Plus, Pencil, Trash2, X, Check } from "lucide-react";

interface ListingExtra {
  id: number;
  name: string;
  description: string;
  type: string;
  price: number;
  duration_days: number | null;
  is_active: boolean;
}

const EMPTY: ListingExtra = { id: 0, name: "", description: "", type: "featured", price: 0, duration_days: 7, is_active: true };
const TYPES = ["featured", "bold", "highlight", "gallery", "video"];

export default function ListingExtrasPage() {
  const qc = useQueryClient();
  const [editing, setEditing] = useState<ListingExtra | null>(null);
  const [creating, setCreating] = useState(false);

  const { data = [], isLoading } = useQuery({ queryKey: ["listing-extras"], queryFn: listingExtrasApi.list });

  const createMut = useMutation({
    mutationFn: (d: Record<string, unknown>) => listingExtrasApi.create(d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["listing-extras"] }); setCreating(false); setEditing(null); },
  });
  const updateMut = useMutation({
    mutationFn: ({ id, ...d }: Record<string, unknown>) => listingExtrasApi.update(id as number, d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["listing-extras"] }); setEditing(null); },
  });
  const deleteMut = useMutation({
    mutationFn: (id: number) => listingExtrasApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["listing-extras"] }),
  });

  const save = () => {
    if (!editing) return;
    if (creating) createMut.mutate(editing as unknown as Record<string, unknown>);
    else updateMut.mutate(editing as unknown as Record<string, unknown>);
  };

  const columns = [
    { key: "name", label: "Name" },
    { key: "type", label: "Type", render: (r: ListingExtra) => <StatusBadge status={r.type} variant="brand" /> },
    { key: "price", label: "Price", render: (r: ListingExtra) => <span className="font-semibold">${r.price.toFixed(2)}</span> },
    { key: "duration_days", label: "Duration", render: (r: ListingExtra) => r.duration_days ? `${r.duration_days}d` : "—" },
    { key: "is_active", label: "Status", render: (r: ListingExtra) => <StatusBadge status={r.is_active ? "active" : "inactive"} /> },
    { key: "actions", label: "", render: (r: ListingExtra) => (
      <div className="flex gap-1">
        <button onClick={() => { setEditing(r); setCreating(false); }} className="p-1 hover:bg-slate-100 rounded"><Pencil className="w-3.5 h-3.5 text-slate-500" /></button>
        <button onClick={() => { if (confirm("Delete?")) deleteMut.mutate(r.id); }} className="p-1 hover:bg-red-50 rounded"><Trash2 className="w-3.5 h-3.5 text-red-400" /></button>
      </div>
    )},
  ];

  return (
    <div>
      <PageHeader
        title="Listing Extras"
        description="Paid add-ons for listings (featured, bold, highlight, etc.)"
        actions={<button onClick={() => { setEditing({ ...EMPTY }); setCreating(true); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Plus className="w-4 h-4" /> New Extra</button>}
      />

      {editing && (
        <div className="p-4 mb-4 rounded-xl border border-slate-200 bg-white">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold text-sm">{creating ? "New Extra" : "Edit Extra"}</h3>
            <button onClick={() => { setEditing(null); setCreating(false); }}><X className="w-4 h-4 text-slate-400" /></button>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <input placeholder="Name" value={editing.name} onChange={(e) => setEditing({ ...editing, name: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <select value={editing.type} onChange={(e) => setEditing({ ...editing, type: e.target.value })} className="border rounded-lg px-3 py-2 text-sm">
              {TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
            </select>
            <input type="number" step="0.01" placeholder="Price" value={editing.price} onChange={(e) => setEditing({ ...editing, price: +e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <input type="number" placeholder="Duration (days)" value={editing.duration_days ?? ""} onChange={(e) => setEditing({ ...editing, duration_days: e.target.value ? +e.target.value : null })} className="border rounded-lg px-3 py-2 text-sm" />
            <input placeholder="Description" value={editing.description} onChange={(e) => setEditing({ ...editing, description: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={editing.is_active} onChange={(e) => setEditing({ ...editing, is_active: e.target.checked })} /> Active</label>
          </div>
          <div className="mt-3 flex gap-2">
            <button onClick={save} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Check className="w-4 h-4" /> Save</button>
            <button onClick={() => { setEditing(null); setCreating(false); }} className="px-3 py-1.5 rounded-lg text-sm border">Cancel</button>
          </div>
        </div>
      )}

      <DataTable columns={columns} data={Array.isArray(data) ? data : []} isLoading={isLoading} emptyMessage="No listing extras." rowKey={(r: ListingExtra) => String(r.id)} />
    </div>
  );
}
