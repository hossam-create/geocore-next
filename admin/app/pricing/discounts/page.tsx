"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { discountCodesApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import DataTable from "@/components/shared/DataTable";
import StatusBadge from "@/components/shared/StatusBadge";
import { Plus, Pencil, Trash2, X, Check } from "lucide-react";

interface DiscountCode {
  id: number;
  code: string;
  description: string;
  discount_type: string;
  discount_value: number;
  applies_to: string;
  min_order_amount: number;
  max_uses: number | null;
  uses_per_user: number;
  current_uses: number;
  is_active: boolean;
  valid_from: string;
  valid_until: string;
}

const EMPTY: DiscountCode = { id: 0, code: "", description: "", discount_type: "percent", discount_value: 0, applies_to: "all", min_order_amount: 0, max_uses: null, uses_per_user: 1, current_uses: 0, is_active: true, valid_from: "", valid_until: "" };

export default function DiscountCodesPage() {
  const qc = useQueryClient();
  const [editing, setEditing] = useState<DiscountCode | null>(null);
  const [creating, setCreating] = useState(false);

  const { data = [], isLoading } = useQuery({ queryKey: ["discount-codes"], queryFn: discountCodesApi.list });

  const createMut = useMutation({
    mutationFn: (d: Record<string, unknown>) => discountCodesApi.create(d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["discount-codes"] }); setCreating(false); setEditing(null); },
  });
  const updateMut = useMutation({
    mutationFn: ({ id, ...d }: Record<string, unknown>) => discountCodesApi.update(id as number, d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["discount-codes"] }); setEditing(null); },
  });
  const deleteMut = useMutation({
    mutationFn: (id: number) => discountCodesApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["discount-codes"] }),
  });

  const save = () => {
    if (!editing) return;
    if (creating) createMut.mutate(editing as unknown as Record<string, unknown>);
    else updateMut.mutate(editing as unknown as Record<string, unknown>);
  };

  const columns = [
    { key: "code", label: "Code", render: (r: DiscountCode) => <span className="font-mono font-bold text-slate-800">{r.code}</span> },
    { key: "discount_type", label: "Type", render: (r: DiscountCode) => <span>{r.discount_value}{r.discount_type === "percent" ? "%" : " fixed"}</span> },
    { key: "applies_to", label: "Applies To", render: (r: DiscountCode) => <StatusBadge status={r.applies_to} variant="info" /> },
    { key: "usage", label: "Usage", render: (r: DiscountCode) => <span className="text-xs text-slate-500">{r.current_uses}/{r.max_uses ?? "∞"}</span> },
    { key: "is_active", label: "Status", render: (r: DiscountCode) => <StatusBadge status={r.is_active ? "active" : "inactive"} /> },
    { key: "actions", label: "", render: (r: DiscountCode) => (
      <div className="flex gap-1">
        <button onClick={() => { setEditing(r); setCreating(false); }} className="p-1 hover:bg-slate-100 rounded"><Pencil className="w-3.5 h-3.5 text-slate-500" /></button>
        <button onClick={() => { if (confirm("Delete?")) deleteMut.mutate(r.id); }} className="p-1 hover:bg-red-50 rounded"><Trash2 className="w-3.5 h-3.5 text-red-400" /></button>
      </div>
    )},
  ];

  return (
    <div>
      <PageHeader
        title="Discount Codes"
        description="Manage promotional codes and coupons"
        actions={<button onClick={() => { setEditing({ ...EMPTY }); setCreating(true); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Plus className="w-4 h-4" /> New Code</button>}
      />

      {editing && (
        <div className="p-4 mb-4 rounded-xl border border-slate-200 bg-white">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold text-sm">{creating ? "New Discount Code" : "Edit Code"}</h3>
            <button onClick={() => { setEditing(null); setCreating(false); }}><X className="w-4 h-4 text-slate-400" /></button>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <input placeholder="Code (e.g. SAVE20)" value={editing.code} onChange={(e) => setEditing({ ...editing, code: e.target.value.toUpperCase() })} className="border rounded-lg px-3 py-2 text-sm font-mono" />
            <input placeholder="Description" value={editing.description} onChange={(e) => setEditing({ ...editing, description: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <select value={editing.discount_type} onChange={(e) => setEditing({ ...editing, discount_type: e.target.value })} className="border rounded-lg px-3 py-2 text-sm">
              <option value="percent">Percent (%)</option>
              <option value="fixed">Fixed Amount</option>
            </select>
            <input type="number" step="0.01" placeholder="Value" value={editing.discount_value} onChange={(e) => setEditing({ ...editing, discount_value: +e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <select value={editing.applies_to} onChange={(e) => setEditing({ ...editing, applies_to: e.target.value })} className="border rounded-lg px-3 py-2 text-sm">
              <option value="all">All</option>
              <option value="classifieds">Classifieds</option>
              <option value="auctions">Auctions</option>
              <option value="subscriptions">Subscriptions</option>
            </select>
            <input type="number" placeholder="Max Uses (0=unlimited)" value={editing.max_uses ?? ""} onChange={(e) => setEditing({ ...editing, max_uses: e.target.value ? +e.target.value : null })} className="border rounded-lg px-3 py-2 text-sm" />
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={editing.is_active} onChange={(e) => setEditing({ ...editing, is_active: e.target.checked })} /> Active</label>
          </div>
          <div className="mt-3 flex gap-2">
            <button onClick={save} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Check className="w-4 h-4" /> Save</button>
            <button onClick={() => { setEditing(null); setCreating(false); }} className="px-3 py-1.5 rounded-lg text-sm border">Cancel</button>
          </div>
        </div>
      )}

      <DataTable columns={columns} data={Array.isArray(data) ? data : []} isLoading={isLoading} emptyMessage="No discount codes." rowKey={(r: DiscountCode) => String(r.id)} />
    </div>
  );
}
