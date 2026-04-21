"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listingsApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import ConfirmDialog from "@/components/shared/ConfirmDialog";
import { useToastStore } from "@/lib/toast";
import {
  ArrowLeft, Check, X, Star, Clock, Trash2, Save,
  Image as ImageIcon, Tag, User, DollarSign, Calendar,
} from "lucide-react";

interface ListingDetail {
  id: string;
  title: string;
  description: string;
  price: number;
  status: string;
  category_id?: number;
  category_name?: string;
  seller_id?: string;
  seller_name?: string;
  is_featured?: boolean;
  images?: { url: string }[];
  created_at: string;
  expires_at?: string;
  [key: string]: unknown;
}

export default function ListingDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [rejectDialog, setRejectDialog] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState(false);
  const [editing, setEditing] = useState(false);
  const [form, setForm] = useState<Partial<ListingDetail>>({});
  const [extendDays, setExtendDays] = useState(7);

  const { data: listing, isLoading } = useQuery<ListingDetail>({
    queryKey: ["admin", "listing", id],
    queryFn: () => listingsApi.get(id),
    enabled: !!id,
  });

  const approveMut = useMutation({
    mutationFn: () => listingsApi.approve(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["admin", "listing", id] }); showToast({ type: "success", title: "Listing approved" }); },
  });
  const rejectMut = useMutation({
    mutationFn: (reason: string) => listingsApi.reject(id, reason),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["admin", "listing", id] }); setRejectDialog(false); showToast({ type: "success", title: "Listing rejected" }); },
  });
  const featureMut = useMutation({
    mutationFn: () => listingsApi.feature(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["admin", "listing", id] }); showToast({ type: "success", title: "Featured status toggled" }); },
  });
  const extendMut = useMutation({
    mutationFn: () => listingsApi.extend(id, extendDays),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["admin", "listing", id] }); showToast({ type: "success", title: `Extended by ${extendDays} days` }); },
  });
  const updateMut = useMutation({
    mutationFn: (data: Record<string, unknown>) => listingsApi.update(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["admin", "listing", id] }); setEditing(false); showToast({ type: "success", title: "Listing updated" }); },
  });
  const deleteMut = useMutation({
    mutationFn: () => listingsApi.delete(id),
    onSuccess: () => { router.push("/operations/listings"); showToast({ type: "success", title: "Listing deleted" }); },
  });

  if (isLoading) return <div className="text-center py-20 text-slate-400">Loading listing...</div>;
  if (!listing) return <div className="text-center py-20 text-slate-400">Listing not found</div>;

  const startEdit = () => { setForm({ title: listing.title, description: listing.description, price: listing.price }); setEditing(true); };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <button onClick={() => router.push("/operations/listings")} className="p-2 hover:bg-slate-100 rounded-lg"><ArrowLeft className="w-4 h-4 text-slate-500" /></button>
        <PageHeader title={listing.title} description={`ID: ${listing.id}`} />
      </div>

      {/* Status + Quick Actions */}
      <div className="rounded-xl border border-slate-200 bg-white p-5">
        <div className="flex flex-wrap items-center gap-3 mb-4">
          <StatusBadge status={listing.status} dot />
          {listing.is_featured && <StatusBadge status="featured" variant="brand" />}
        </div>
        <div className="flex flex-wrap gap-2">
          {listing.status === "pending" && (
            <>
              <button onClick={() => approveMut.mutate()} disabled={approveMut.isPending} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-green-50 text-green-600 hover:bg-green-100"><Check className="w-3.5 h-3.5" /> Approve</button>
              <button onClick={() => setRejectDialog(true)} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-red-50 text-red-500 hover:bg-red-100"><X className="w-3.5 h-3.5" /> Reject</button>
            </>
          )}
          <button onClick={() => featureMut.mutate()} disabled={featureMut.isPending} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-amber-50 text-amber-600 hover:bg-amber-100"><Star className="w-3.5 h-3.5" /> {listing.is_featured ? "Unfeature" : "Feature"}</button>
          <button onClick={startEdit} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-slate-100 text-slate-600 hover:bg-slate-200"><Save className="w-3.5 h-3.5" /> Edit</button>
          <button onClick={() => setDeleteDialog(true)} className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-red-50 text-red-400 hover:bg-red-100"><Trash2 className="w-3.5 h-3.5" /> Delete</button>
        </div>
      </div>

      {/* Edit Form */}
      {editing && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="text-sm font-semibold text-slate-700 mb-3">Edit Listing</h3>
          <div className="space-y-3">
            <input value={form.title ?? ""} onChange={(e) => setForm({ ...form, title: e.target.value })} placeholder="Title" className="w-full border rounded-lg px-3 py-2 text-sm" />
            <textarea rows={4} value={form.description ?? ""} onChange={(e) => setForm({ ...form, description: e.target.value })} placeholder="Description" className="w-full border rounded-lg px-3 py-2 text-sm" />
            <input type="number" value={form.price ?? 0} onChange={(e) => setForm({ ...form, price: +e.target.value })} placeholder="Price" className="w-full border rounded-lg px-3 py-2 text-sm" />
            <div className="flex gap-2">
              <button onClick={() => updateMut.mutate(form as Record<string, unknown>)} disabled={updateMut.isPending} className="px-4 py-2 text-sm rounded-lg text-white bg-indigo-600 disabled:opacity-50">Save Changes</button>
              <button onClick={() => setEditing(false)} className="px-4 py-2 text-sm rounded-lg border">Cancel</button>
            </div>
          </div>
        </div>
      )}

      {/* Details Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="text-sm font-semibold text-slate-700 mb-3">Details</h3>
          <dl className="space-y-3">
            <div className="flex items-center gap-2"><DollarSign className="w-4 h-4 text-slate-400" /><dt className="text-xs text-slate-400 w-20">Price</dt><dd className="text-sm font-bold text-slate-800">${listing.price?.toLocaleString()}</dd></div>
            <div className="flex items-center gap-2"><Tag className="w-4 h-4 text-slate-400" /><dt className="text-xs text-slate-400 w-20">Category</dt><dd className="text-sm text-slate-700">{listing.category_name ?? listing.category_id ?? "—"}</dd></div>
            <div className="flex items-center gap-2"><User className="w-4 h-4 text-slate-400" /><dt className="text-xs text-slate-400 w-20">Seller</dt><dd className="text-sm text-slate-700">{listing.seller_name ?? listing.seller_id ?? "—"}</dd></div>
            <div className="flex items-center gap-2"><Calendar className="w-4 h-4 text-slate-400" /><dt className="text-xs text-slate-400 w-20">Created</dt><dd className="text-sm text-slate-700">{new Date(listing.created_at).toLocaleString()}</dd></div>
            {listing.expires_at && <div className="flex items-center gap-2"><Clock className="w-4 h-4 text-slate-400" /><dt className="text-xs text-slate-400 w-20">Expires</dt><dd className="text-sm text-slate-700">{new Date(listing.expires_at).toLocaleString()}</dd></div>}
          </dl>
        </div>

        {/* Extend Duration */}
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="text-sm font-semibold text-slate-700 mb-3">Extend Duration</h3>
          <div className="flex items-center gap-3">
            <select value={extendDays} onChange={(e) => setExtendDays(+e.target.value)} className="border rounded-lg px-3 py-2 text-sm">
              <option value={7}>7 days</option>
              <option value={14}>14 days</option>
              <option value={30}>30 days</option>
              <option value={60}>60 days</option>
              <option value={90}>90 days</option>
            </select>
            <button onClick={() => extendMut.mutate()} disabled={extendMut.isPending} className="px-4 py-2 text-sm rounded-lg text-white bg-indigo-600 disabled:opacity-50">
              <Clock className="w-3.5 h-3.5 inline mr-1" /> Extend
            </button>
          </div>
          {listing.description && (
            <div className="mt-4 pt-3 border-t border-slate-100">
              <h4 className="text-xs font-medium text-slate-400 mb-1">Description</h4>
              <p className="text-sm text-slate-600 whitespace-pre-wrap">{listing.description}</p>
            </div>
          )}
        </div>
      </div>

      {/* Images */}
      {listing.images && listing.images.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="text-sm font-semibold text-slate-700 mb-3">Images</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            {listing.images.map((img, i) => (
              <div key={i} className="aspect-square rounded-lg bg-slate-100 overflow-hidden">
                <img src={img.url} alt={`Image ${i + 1}`} className="w-full h-full object-cover" />
              </div>
            ))}
          </div>
        </div>
      )}

      <ConfirmDialog open={rejectDialog} title="Reject Listing" message="This listing will be marked as rejected. The seller will be notified." confirmLabel="Reject" variant="danger" requireReason reasonLabel="Rejection Reason" onConfirm={(reason) => rejectMut.mutate(reason ?? "Rejected")} onCancel={() => setRejectDialog(false)} isLoading={rejectMut.isPending} />
      <ConfirmDialog open={deleteDialog} title="Delete Listing" message="This will permanently delete the listing. This action cannot be undone." confirmLabel="Delete" variant="danger" onConfirm={() => deleteMut.mutate()} onCancel={() => setDeleteDialog(false)} isLoading={deleteMut.isPending} />
    </div>
  );
}
