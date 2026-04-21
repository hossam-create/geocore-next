"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { gatewaysApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import { Pencil, X, Check, CreditCard } from "lucide-react";

interface Gateway {
  id: number;
  name: string;
  slug: string;
  display_name: string;
  is_active: boolean;
  is_sandbox: boolean;
  fee_percent: number;
  fee_fixed: number;
  sort_order: number;
}

export default function GatewaysPage() {
  const qc = useQueryClient();
  const [editing, setEditing] = useState<Gateway | null>(null);

  const { data = [], isLoading } = useQuery({ queryKey: ["gateways"], queryFn: gatewaysApi.list });

  const updateMut = useMutation({
    mutationFn: ({ id, ...d }: Record<string, unknown>) => gatewaysApi.update(id as number, d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["gateways"] }); setEditing(null); },
  });

  const gateways: Gateway[] = Array.isArray(data) ? data : [];

  return (
    <div>
      <PageHeader title="Payment Gateways" description="Configure payment providers" />

      {editing && (
        <div className="p-4 mb-4 rounded-xl border border-slate-200 bg-white">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold text-sm">Edit: {editing.display_name}</h3>
            <button onClick={() => setEditing(null)}><X className="w-4 h-4 text-slate-400" /></button>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <input placeholder="Display Name" value={editing.display_name} onChange={(e) => setEditing({ ...editing, display_name: e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <input type="number" step="0.01" placeholder="Fee %" value={editing.fee_percent} onChange={(e) => setEditing({ ...editing, fee_percent: +e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <input type="number" step="0.01" placeholder="Fixed Fee" value={editing.fee_fixed} onChange={(e) => setEditing({ ...editing, fee_fixed: +e.target.value })} className="border rounded-lg px-3 py-2 text-sm" />
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={editing.is_active} onChange={(e) => setEditing({ ...editing, is_active: e.target.checked })} /> Active</label>
            <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={editing.is_sandbox} onChange={(e) => setEditing({ ...editing, is_sandbox: e.target.checked })} /> Sandbox Mode</label>
          </div>
          <div className="mt-3 flex gap-2">
            <button onClick={() => updateMut.mutate(editing as unknown as Record<string, unknown>)} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium text-white" style={{ background: "var(--color-brand)" }}><Check className="w-4 h-4" /> Save</button>
            <button onClick={() => setEditing(null)} className="px-3 py-1.5 rounded-lg text-sm border">Cancel</button>
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="text-center py-12 text-sm text-slate-400">Loading gateways...</div>
      ) : (
        <div className="grid gap-3">
          {gateways.map((g) => (
            <div key={g.id} className="flex items-center justify-between p-4 rounded-xl border border-slate-200 bg-white">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-lg flex items-center justify-center" style={{ background: g.is_active ? "var(--color-success-light)" : "var(--bg-inset)" }}>
                  <CreditCard className="w-5 h-5" style={{ color: g.is_active ? "var(--color-success)" : "var(--text-tertiary)" }} />
                </div>
                <div>
                  <p className="text-sm font-semibold text-slate-800">{g.display_name}</p>
                  <p className="text-xs text-slate-400">{g.slug} — Fee: {g.fee_percent}% + {g.fee_fixed}</p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                {g.is_sandbox && <StatusBadge status="sandbox" variant="warning" />}
                <StatusBadge status={g.is_active ? "active" : "inactive"} />
                <button onClick={() => setEditing(g)} className="p-1.5 hover:bg-slate-100 rounded"><Pencil className="w-3.5 h-3.5 text-slate-500" /></button>
              </div>
            </div>
          ))}
          {gateways.length === 0 && <p className="text-sm text-slate-400 text-center py-8">No payment gateways. Run migration 026 to seed defaults.</p>}
        </div>
      )}
    </div>
  );
}
