"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { storefrontsApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import DataTable from "@/components/shared/DataTable";
import StatusBadge from "@/components/shared/StatusBadge";
import ConfirmDialog from "@/components/shared/ConfirmDialog";
import { useToastStore } from "@/lib/toast";
import { Store, Check, Pause, Star, Trash2, ExternalLink } from "lucide-react";

interface Storefront {
  id: string;
  name: string;
  slug: string;
  owner_name?: string;
  owner_id?: string;
  status: string;
  is_featured?: boolean;
  listings_count?: number;
  created_at: string;
  [key: string]: unknown;
}

export default function StorefrontsPage() {
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [deleteTarget, setDeleteTarget] = useState<Storefront | null>(null);

  const { data = [], isLoading } = useQuery({
    queryKey: ["storefronts"],
    queryFn: () => storefrontsApi.list(),
    retry: 1,
  });

  const approveMut = useMutation({
    mutationFn: (id: string) => storefrontsApi.approve(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["storefronts"] }); showToast({ type: "success", title: "Storefront approved" }); },
  });
  const suspendMut = useMutation({
    mutationFn: (id: string) => storefrontsApi.suspend(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["storefronts"] }); showToast({ type: "success", title: "Storefront suspended" }); },
  });
  const featureMut = useMutation({
    mutationFn: (id: string) => storefrontsApi.feature(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["storefronts"] }); showToast({ type: "success", title: "Featured status toggled" }); },
  });
  const deleteMut = useMutation({
    mutationFn: (id: string) => storefrontsApi.delete(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["storefronts"] }); setDeleteTarget(null); showToast({ type: "success", title: "Storefront deleted" }); },
  });

  const stores: Storefront[] = Array.isArray(data) ? data : [];

  const columns = [
    { key: "name", label: "Store Name", render: (r: Storefront) => (
      <div className="flex items-center gap-2">
        <Store className="w-4 h-4 text-slate-400" />
        <span className="font-medium text-slate-800">{r.name}</span>
        {r.is_featured && <Star className="w-3 h-3 text-amber-400 fill-amber-400" />}
      </div>
    )},
    { key: "slug", label: "Slug", render: (r: Storefront) => <span className="font-mono text-xs text-slate-400">/{r.slug}</span> },
    { key: "owner_name", label: "Owner", render: (r: Storefront) => r.owner_name ?? r.owner_id ?? "—" },
    { key: "listings_count", label: "Listings", render: (r: Storefront) => r.listings_count ?? 0 },
    { key: "status", label: "Status", render: (r: Storefront) => <StatusBadge status={r.status} dot /> },
    { key: "created_at", label: "Created", render: (r: Storefront) => new Date(r.created_at).toLocaleDateString() },
    { key: "actions", label: "", render: (r: Storefront) => (
      <div className="flex gap-1">
        {r.status === "pending" && (
          <button onClick={() => approveMut.mutate(r.id)} title="Approve" className="p-1 hover:bg-green-50 rounded"><Check className="w-3.5 h-3.5 text-green-500" /></button>
        )}
        {r.status === "active" && (
          <button onClick={() => suspendMut.mutate(r.id)} title="Suspend" className="p-1 hover:bg-amber-50 rounded"><Pause className="w-3.5 h-3.5 text-amber-500" /></button>
        )}
        <button onClick={() => featureMut.mutate(r.id)} title="Toggle Featured" className="p-1 hover:bg-amber-50 rounded"><Star className="w-3.5 h-3.5 text-amber-400" /></button>
        <button onClick={() => setDeleteTarget(r)} title="Delete" className="p-1 hover:bg-red-50 rounded"><Trash2 className="w-3.5 h-3.5 text-red-400" /></button>
      </div>
    )},
  ];

  return (
    <div>
      <PageHeader title="Storefronts" description="Manage seller stores" />
      <DataTable columns={columns} data={stores} isLoading={isLoading} emptyMessage="No storefronts yet." rowKey={(r: Storefront) => r.id} />

      <ConfirmDialog
        open={!!deleteTarget}
        title="Delete Storefront"
        message={`Delete "${deleteTarget?.name}"? This cannot be undone.`}
        confirmLabel="Delete"
        variant="danger"
        onConfirm={() => deleteTarget && deleteMut.mutate(deleteTarget.id)}
        onCancel={() => setDeleteTarget(null)}
        isLoading={deleteMut.isPending}
      />
    </div>
  );
}
