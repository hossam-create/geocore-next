"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { geographyApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import { Plus, Pencil, Trash2, X, Check, ChevronRight, ChevronDown, MapPin } from "lucide-react";

interface GeoRegion {
  id: number;
  name: string;
  name_ar: string;
  code: string;
  type: string;
  parent_id: number | null;
  is_active: boolean;
  sort_order: number;
  children?: GeoRegion[];
}

const EMPTY: GeoRegion = { id: 0, name: "", name_ar: "", code: "", type: "country", parent_id: null, is_active: true, sort_order: 0 };
const REGION_TYPES = ["country", "state", "city"];

function RegionRow({ region, depth, onEdit, onDelete, expanded, onToggle }: { region: GeoRegion; depth: number; onEdit: (r: GeoRegion) => void; onDelete: (id: number) => void; expanded: Set<number>; onToggle: (id: number) => void }) {
  const hasChildren = region.children && region.children.length > 0;
  const isExpanded = expanded.has(region.id);

  return (
    <>
      <div className="flex items-center justify-between py-2 px-3 hover:bg-slate-50 rounded-lg" style={{ paddingLeft: `${depth * 24 + 12}px` }}>
        <div className="flex items-center gap-2">
          {hasChildren ? (
            <button onClick={() => onToggle(region.id)} className="p-0.5">{isExpanded ? <ChevronDown className="w-3.5 h-3.5 text-slate-400" /> : <ChevronRight className="w-3.5 h-3.5 text-slate-400" />}</button>
          ) : <span className="w-4.5" />}
          <MapPin className="w-3.5 h-3.5 text-slate-400" />
          <span className="text-sm font-medium text-slate-700">{region.name}</span>
          {region.name_ar && <span className="text-xs text-slate-400">({region.name_ar})</span>}
          {region.code && <span className="text-[10px] px-1.5 py-0.5 bg-slate-100 rounded font-mono text-slate-500">{region.code}</span>}
          <StatusBadge status={region.type} variant={region.type === "country" ? "brand" : region.type === "state" ? "info" : "neutral"} />
        </div>
        <div className="flex gap-1">
          <button onClick={() => onEdit(region)} className="p-1 hover:bg-slate-100 rounded"><Pencil className="w-3.5 h-3.5 text-slate-500" /></button>
          <button onClick={() => { if (confirm("Delete?")) onDelete(region.id); }} className="p-1 hover:bg-red-50 rounded"><Trash2 className="w-3.5 h-3.5 text-red-400" /></button>
        </div>
      </div>
      {isExpanded && hasChildren && region.children!.map((child) => (
        <RegionRow key={child.id} region={child} depth={depth + 1} onEdit={onEdit} onDelete={onDelete} expanded={expanded} onToggle={onToggle} />
      ))}
    </>
  );
}

export default function GeographyPage() {
  const qc = useQueryClient();
  const [editing, setEditing] = useState<GeoRegion | null>(null);
  const [creating, setCreating] = useState(false);
  const [expanded, setExpanded] = useState<Set<number>>(new Set());

  const { data = [], isLoading } = useQuery({ queryKey: ["geography"], queryFn: geographyApi.list });

  const createMut = useMutation({
    mutationFn: (d: Record<string, unknown>) => geographyApi.create(d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["geography"] }); setCreating(false); setEditing(null); },
  });
  const updateMut = useMutation({
    mutationFn: ({ id, ...d }: Record<string, unknown>) => geographyApi.update(id as number, d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["geography"] }); setEditing(null); },
  });
  const deleteMut = useMutation({
    mutationFn: (id: number) => geographyApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["geography"] }),
  });

  const save = () => {
    if (!editing) return;
    if (creating) createMut.mutate(editing as unknown as Record<string, unknown>);
    else updateMut.mutate(editing as unknown as Record<string, unknown>);
  };

  const toggle = (id: number) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  const regions: GeoRegion[] = Array.isArray(data) ? data : [];

  return (
    <div>
      <PageHeader
        title="Geography"
        description="Countries, states, and cities"
        actions={<button onClick={() => { setEditing({ ...EMPTY }); setCreating(true); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Plus className="w-4 h-4" /> Add Region</button>}
      />

      {editing && (
        <div className="p-4 mb-4 rounded-xl border border-slate-200 bg-white">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold text-sm">{creating ? "New Region" : "Edit Region"}</h3>
            <button onClick={() => { setEditing(null); setCreating(false); }}><X className="w-4 h-4 text-slate-400" /></button>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <input placeholder="Name (EN)" value={editing.name} onChange={(e) => setEditing({ ...editing, name: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <input placeholder="Name (AR)" value={editing.name_ar} onChange={(e) => setEditing({ ...editing, name_ar: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <input placeholder="Code (e.g. EG)" value={editing.code} onChange={(e) => setEditing({ ...editing, code: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <select value={editing.type} onChange={(e) => setEditing({ ...editing, type: e.target.value })} className="border rounded-lg px-3 py-2 text-sm">
              {REGION_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
            </select>
            <input type="number" placeholder="Parent ID" value={editing.parent_id ?? ""} onChange={(e) => setEditing({ ...editing, parent_id: e.target.value ? +e.target.value : null })} className="border rounded-lg px-3 py-2 text-sm" />
            <input type="number" placeholder="Sort Order" value={editing.sort_order} onChange={(e) => setEditing({ ...editing, sort_order: +e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
          </div>
          <div className="mt-3 flex gap-2">
            <button onClick={save} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Check className="w-4 h-4" /> Save</button>
            <button onClick={() => { setEditing(null); setCreating(false); }} className="px-3 py-1.5 rounded-lg text-sm border">Cancel</button>
          </div>
        </div>
      )}

      <div className="rounded-xl border border-slate-200 bg-white p-2">
        {isLoading ? (
          <div className="text-center py-12 text-sm text-slate-400">Loading regions...</div>
        ) : regions.length === 0 ? (
          <div className="text-center py-12 text-sm text-slate-400">No regions. Run migration 026 to seed defaults.</div>
        ) : (
          regions.map((r) => <RegionRow key={r.id} region={r} depth={0} onEdit={(r) => { setEditing(r); setCreating(false); }} onDelete={(id) => deleteMut.mutate(id)} expanded={expanded} onToggle={toggle} />)
        )}
      </div>
    </div>
  );
}
