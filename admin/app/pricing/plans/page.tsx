"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { plansApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import { Plus, Pencil, Trash2, X, Check, DollarSign } from "lucide-react";

interface Plan {
  id: string;
  name: string;
  display_name: string;
  price_monthly: number;
  currency: string;
  listing_limit: number;
  features: string[];
  is_active: boolean;
  sort_order: number;
}

const EMPTY: Plan = { id: "", name: "", display_name: "", price_monthly: 0, currency: "AED", listing_limit: 5, features: [], is_active: true, sort_order: 0 };

export default function PricePlansPage() {
  const qc = useQueryClient();
  const [editing, setEditing] = useState<Plan | null>(null);
  const [creating, setCreating] = useState(false);
  const [featuresText, setFeaturesText] = useState("");

  const { data = [], isLoading } = useQuery({ queryKey: ["plans"], queryFn: plansApi.list });

  const createMut = useMutation({
    mutationFn: (d: Record<string, unknown>) => plansApi.create(d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["plans"] }); setCreating(false); setEditing(null); },
  });
  const updateMut = useMutation({
    mutationFn: ({ id, ...d }: Record<string, unknown>) => plansApi.update(id as string, d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["plans"] }); setEditing(null); },
  });
  const deleteMut = useMutation({
    mutationFn: (id: string) => plansApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["plans"] }),
  });

  const startEdit = (p: Plan) => {
    setEditing(p);
    setFeaturesText(Array.isArray(p.features) ? p.features.join("\n") : "");
    setCreating(false);
  };

  const save = () => {
    if (!editing) return;
    const payload = { ...editing, features: featuresText.split("\n").map((s) => s.trim()).filter(Boolean) };
    if (creating) createMut.mutate(payload as unknown as Record<string, unknown>);
    else updateMut.mutate(payload as unknown as Record<string, unknown>);
  };

  const plans: Plan[] = Array.isArray(data) ? data : [];

  return (
    <div>
      <PageHeader
        title="Price Plans"
        description="Manage subscription and listing plans"
        actions={<button onClick={() => { setEditing({ ...EMPTY }); setFeaturesText(""); setCreating(true); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Plus className="w-4 h-4" /> New Plan</button>}
      />

      {editing && (
        <div className="p-4 mb-4 rounded-xl border border-slate-200 bg-white">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold text-sm">{creating ? "New Plan" : "Edit Plan"}</h3>
            <button onClick={() => { setEditing(null); setCreating(false); }}><X className="w-4 h-4 text-slate-400" /></button>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <input placeholder="Name (key)" value={editing.name} onChange={(e) => setEditing({ ...editing, name: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <input placeholder="Display Name" value={editing.display_name} onChange={(e) => setEditing({ ...editing, display_name: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <input type="number" placeholder="Price/month" value={editing.price_monthly} onChange={(e) => setEditing({ ...editing, price_monthly: +e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <input placeholder="Currency" value={editing.currency} onChange={(e) => setEditing({ ...editing, currency: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <input type="number" placeholder="Listing Limit" value={editing.listing_limit} onChange={(e) => setEditing({ ...editing, listing_limit: +e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <input type="number" placeholder="Sort Order" value={editing.sort_order} onChange={(e) => setEditing({ ...editing, sort_order: +e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
          </div>
          <textarea rows={4} placeholder="Features (one per line)" value={featuresText} onChange={(e) => setFeaturesText(e.target.value)} className="w-full border rounded-lg px-3 py-2 text-sm mt-3" />
          <label className="flex items-center gap-2 text-sm mt-2"><input type="checkbox" checked={editing.is_active} onChange={(e) => setEditing({ ...editing, is_active: e.target.checked })} /> Active</label>
          <div className="mt-3 flex gap-2">
            <button onClick={save} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Check className="w-4 h-4" /> Save</button>
            <button onClick={() => { setEditing(null); setCreating(false); }} className="px-3 py-1.5 rounded-lg text-sm border">Cancel</button>
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="text-center py-12 text-sm text-slate-400">Loading plans...</div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {plans.map((p) => (
            <div key={p.id} className="rounded-xl border border-slate-200 bg-white p-5 hover:shadow-sm transition-shadow">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="font-bold text-base text-slate-800">{p.display_name}</h3>
                  <p className="text-xs text-slate-400 mt-0.5">{p.name}</p>
                </div>
                <StatusBadge status={p.is_active ? "active" : "inactive"} />
              </div>
              <div className="flex items-baseline gap-1 mt-3">
                <DollarSign className="w-4 h-4 text-slate-400" />
                <span className="text-2xl font-bold text-slate-800">{p.price_monthly}</span>
                <span className="text-xs text-slate-400">/ month</span>
              </div>
              <p className="text-xs text-slate-500 mt-2">Up to {p.listing_limit} listings</p>
              {Array.isArray(p.features) && p.features.length > 0 && (
                <ul className="mt-3 space-y-1">
                  {p.features.map((f, i) => <li key={i} className="text-xs text-slate-600 flex items-center gap-1.5"><Check className="w-3 h-3 text-green-500" />{f}</li>)}
                </ul>
              )}
              <div className="flex gap-2 mt-4 pt-3 border-t border-slate-100">
                <button onClick={() => startEdit(p)} className="text-xs font-medium px-2 py-1 rounded hover:bg-slate-100 flex items-center gap-1"><Pencil className="w-3 h-3" /> Edit</button>
                <button onClick={() => { if (confirm("Delete this plan?")) deleteMut.mutate(p.id); }} className="text-xs font-medium px-2 py-1 rounded hover:bg-red-50 text-red-500 flex items-center gap-1"><Trash2 className="w-3 h-3" /> Delete</button>
              </div>
            </div>
          ))}
          {plans.length === 0 && <p className="text-sm text-slate-400 col-span-full text-center py-8">No plans configured.</p>}
        </div>
      )}
    </div>
  );
}
