"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { userGroupsApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import DataTable from "@/components/shared/DataTable";
import StatusBadge from "@/components/shared/StatusBadge";
import { Plus, Pencil, Trash2, X, Check } from "lucide-react";

interface UserGroup {
  id: number;
  name: string;
  slug: string;
  description: string;
  max_active_listings: number;
  can_place_auctions: boolean;
  requires_approval: boolean;
  is_default: boolean;
  sort_order: number;
}

const EMPTY: UserGroup = { id: 0, name: "", slug: "", description: "", max_active_listings: 10, can_place_auctions: true, requires_approval: false, is_default: false, sort_order: 0 };

export default function UserGroupsPage() {
  const qc = useQueryClient();
  const [editing, setEditing] = useState<UserGroup | null>(null);
  const [creating, setCreating] = useState(false);

  const { data = [], isLoading } = useQuery({ queryKey: ["user-groups"], queryFn: userGroupsApi.list });

  const createMut = useMutation({
    mutationFn: (d: Record<string, unknown>) => userGroupsApi.create(d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["user-groups"] }); setCreating(false); setEditing(null); },
  });
  const updateMut = useMutation({
    mutationFn: ({ id, ...d }: Record<string, unknown>) => userGroupsApi.update(id as number, d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["user-groups"] }); setEditing(null); },
  });
  const deleteMut = useMutation({
    mutationFn: (id: number) => userGroupsApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["user-groups"] }),
  });

  const save = () => {
    if (!editing) return;
    if (creating) createMut.mutate(editing as unknown as Record<string, unknown>);
    else updateMut.mutate(editing as unknown as Record<string, unknown>);
  };

  const columns = [
    { key: "name", label: "Name" },
    { key: "slug", label: "Slug" },
    { key: "max_active_listings", label: "Max Listings" },
    { key: "can_place_auctions", label: "Auctions", render: (r: UserGroup) => r.can_place_auctions ? <StatusBadge status="active" /> : <StatusBadge status="disabled" variant="neutral" /> },
    { key: "is_default", label: "Default", render: (r: UserGroup) => r.is_default ? <StatusBadge status="default" variant="brand" /> : "—" },
    { key: "actions", label: "", render: (r: UserGroup) => (
      <div className="flex gap-1">
        <button onClick={() => { setEditing(r); setCreating(false); }} className="p-1 hover:bg-slate-100 rounded"><Pencil className="w-3.5 h-3.5 text-slate-500" /></button>
        <button onClick={() => { if (confirm("Delete?")) deleteMut.mutate(r.id); }} className="p-1 hover:bg-red-50 rounded"><Trash2 className="w-3.5 h-3.5 text-red-400" /></button>
      </div>
    )},
  ];

  return (
    <div>
      <PageHeader
        title="User Groups"
        description="Manage user group tiers and permissions"
        actions={<button onClick={() => { setEditing({ ...EMPTY }); setCreating(true); }} className="btn-primary flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Plus className="w-4 h-4" /> Add Group</button>}
      />

      {editing && (
        <div className="surface p-4 mb-4 rounded-xl border border-slate-200 bg-white">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold text-sm" style={{ color: "var(--text-primary)" }}>{creating ? "New Group" : "Edit Group"}</h3>
            <button onClick={() => { setEditing(null); setCreating(false); }}><X className="w-4 h-4 text-slate-400" /></button>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <input placeholder="Name" value={editing.name} onChange={(e) => setEditing({ ...editing, name: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <input placeholder="Slug" value={editing.slug} onChange={(e) => setEditing({ ...editing, slug: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <input placeholder="Description" value={editing.description} onChange={(e) => setEditing({ ...editing, description: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <input type="number" placeholder="Max Listings" value={editing.max_active_listings} onChange={(e) => setEditing({ ...editing, max_active_listings: +e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={editing.can_place_auctions} onChange={(e) => setEditing({ ...editing, can_place_auctions: e.target.checked })} /> Can Place Auctions</label>
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={editing.requires_approval} onChange={(e) => setEditing({ ...editing, requires_approval: e.target.checked })} /> Requires Approval</label>
          </div>
          <div className="mt-3 flex gap-2">
            <button onClick={save} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Check className="w-4 h-4" /> Save</button>
            <button onClick={() => { setEditing(null); setCreating(false); }} className="px-3 py-1.5 rounded-lg text-sm border">Cancel</button>
          </div>
        </div>
      )}

      <DataTable columns={columns} data={Array.isArray(data) ? data : []} isLoading={isLoading} emptyMessage="No user groups yet." rowKey={(r: UserGroup) => String(r.id)} />
    </div>
  );
}
