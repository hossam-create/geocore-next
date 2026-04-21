"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { usersApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import DataTable from "@/components/shared/DataTable";
import ConfirmDialog from "@/components/shared/ConfirmDialog";
import { useToastStore } from "@/lib/toast";
import {
  ArrowLeft, Ban, ShieldCheck, UserCog, Key, Eye,
  Mail, Calendar, MapPin, Globe, Package, ShoppingCart,
} from "lucide-react";

interface UserDetail {
  id: string;
  name: string;
  email: string;
  role: string;
  is_active: boolean;
  is_banned: boolean;
  is_verified: boolean;
  ban_reason?: string;
  group_id?: number;
  created_at: string;
  updated_at?: string;
  phone?: string;
  avatar_url?: string;
  location?: string;
}

const TABS = ["Overview", "Listings", "Orders", "Activity"] as const;
type Tab = typeof TABS[number];

export default function UserDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);
  const [tab, setTab] = useState<Tab>("Overview");
  const [banDialog, setBanDialog] = useState(false);
  const [roleDialog, setRoleDialog] = useState(false);
  const [newRole, setNewRole] = useState("user");

  const { data: user, isLoading } = useQuery<UserDetail>({
    queryKey: ["admin", "user", id],
    queryFn: async () => {
      const res = await usersApi.get(id);
      const d = (res as Record<string, unknown>)?.data ?? res;
      return d as UserDetail;
    },
    enabled: !!id,
  });

  const { data: listings = [] } = useQuery({
    queryKey: ["admin", "user", id, "listings"],
    queryFn: async () => {
      try {
        const res = await usersApi.listings(id);
        const arr = (res as Record<string, unknown>)?.data ?? res;
        return Array.isArray(arr) ? arr : [];
      } catch { return []; }
    },
    enabled: tab === "Listings",
  });

  const { data: orders = [] } = useQuery({
    queryKey: ["admin", "user", id, "orders"],
    queryFn: async () => {
      try {
        const res = await usersApi.orders(id);
        const arr = (res as Record<string, unknown>)?.data ?? res;
        return Array.isArray(arr) ? arr : [];
      } catch { return []; }
    },
    enabled: tab === "Orders",
  });

  const banMut = useMutation({
    mutationFn: (reason?: string) => usersApi.ban(id, reason),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["admin", "user", id] }); setBanDialog(false); showToast({ type: "success", title: "User banned" }); },
  });
  const unbanMut = useMutation({
    mutationFn: () => usersApi.unban(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["admin", "user", id] }); showToast({ type: "success", title: "User unbanned" }); },
  });
  const verifyMut = useMutation({
    mutationFn: () => usersApi.verify(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["admin", "user", id] }); showToast({ type: "success", title: "User verified" }); },
  });
  const roleMut = useMutation({
    mutationFn: (role: string) => usersApi.changeRole(id, role),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["admin", "user", id] }); setRoleDialog(false); showToast({ type: "success", title: "Role changed" }); },
  });

  if (isLoading) return <div className="text-center py-20 text-slate-400">Loading user...</div>;
  if (!user) return <div className="text-center py-20 text-slate-400">User not found</div>;

  const status = user.is_banned ? "banned" : user.is_active ? "active" : "inactive";

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <button onClick={() => router.push("/admin/users")} className="p-2 hover:bg-slate-100 rounded-lg"><ArrowLeft className="w-4 h-4 text-slate-500" /></button>
        <PageHeader title={user.name || "User"} description={user.email} />
      </div>

      {/* User Info Card */}
      <div className="rounded-xl border border-slate-200 bg-white p-6">
        <div className="flex flex-col md:flex-row md:items-start gap-6">
          <div className="w-16 h-16 rounded-full bg-slate-200 flex items-center justify-center text-xl font-bold text-slate-500 flex-shrink-0">
            {(user.name || "?")[0].toUpperCase()}
          </div>
          <div className="flex-1 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            <div className="flex items-center gap-2"><Mail className="w-4 h-4 text-slate-400" /><span className="text-sm">{user.email}</span></div>
            <div className="flex items-center gap-2"><UserCog className="w-4 h-4 text-slate-400" /><StatusBadge status={user.role} variant="brand" /></div>
            <div className="flex items-center gap-2"><ShieldCheck className="w-4 h-4 text-slate-400" /><StatusBadge status={status} dot /></div>
            <div className="flex items-center gap-2"><Calendar className="w-4 h-4 text-slate-400" /><span className="text-sm text-slate-600">Joined {new Date(user.created_at).toLocaleDateString()}</span></div>
            {user.is_verified && <div className="flex items-center gap-2"><ShieldCheck className="w-4 h-4 text-green-500" /><span className="text-sm text-green-600 font-medium">Verified</span></div>}
            {user.ban_reason && <div className="flex items-center gap-2 col-span-full"><Ban className="w-4 h-4 text-red-400" /><span className="text-sm text-red-500">Ban reason: {user.ban_reason}</span></div>}
          </div>
        </div>

        {/* Quick Actions */}
        <div className="flex flex-wrap gap-2 mt-5 pt-4 border-t border-slate-100">
          {user.is_banned ? (
            <button onClick={() => unbanMut.mutate()} disabled={unbanMut.isPending} className="px-3 py-1.5 text-xs font-medium rounded-lg bg-green-50 text-green-600 hover:bg-green-100">Unban</button>
          ) : (
            <button onClick={() => setBanDialog(true)} className="px-3 py-1.5 text-xs font-medium rounded-lg bg-red-50 text-red-500 hover:bg-red-100">Ban User</button>
          )}
          {!user.is_verified && (
            <button onClick={() => verifyMut.mutate()} disabled={verifyMut.isPending} className="px-3 py-1.5 text-xs font-medium rounded-lg bg-blue-50 text-blue-600 hover:bg-blue-100">Verify</button>
          )}
          <button onClick={() => { setNewRole(user.role); setRoleDialog(true); }} className="px-3 py-1.5 text-xs font-medium rounded-lg bg-slate-100 text-slate-600 hover:bg-slate-200">Change Role</button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-slate-200">
        {TABS.map((t) => (
          <button key={t} onClick={() => setTab(t)} className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${tab === t ? "border-indigo-500 text-indigo-600" : "border-transparent text-slate-400 hover:text-slate-600"}`}>{t}</button>
        ))}
      </div>

      {/* Tab Content */}
      {tab === "Overview" && (
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <h3 className="text-sm font-semibold text-slate-700 mb-3">Account Details</h3>
          <dl className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div><dt className="text-xs text-slate-400">User ID</dt><dd className="text-sm font-mono text-slate-700">{user.id}</dd></div>
            <div><dt className="text-xs text-slate-400">Role</dt><dd className="text-sm text-slate-700">{user.role}</dd></div>
            <div><dt className="text-xs text-slate-400">Status</dt><dd><StatusBadge status={status} dot /></dd></div>
            <div><dt className="text-xs text-slate-400">Verified</dt><dd className="text-sm text-slate-700">{user.is_verified ? "Yes" : "No"}</dd></div>
            <div><dt className="text-xs text-slate-400">Created</dt><dd className="text-sm text-slate-700">{new Date(user.created_at).toLocaleString()}</dd></div>
            {user.updated_at && <div><dt className="text-xs text-slate-400">Last Updated</dt><dd className="text-sm text-slate-700">{new Date(user.updated_at).toLocaleString()}</dd></div>}
          </dl>
        </div>
      )}

      {tab === "Listings" && (
        <DataTable
          columns={[
            { key: "id", label: "ID", render: (r: Record<string, unknown>) => <span className="font-mono text-xs">{String(r.id)}</span> },
            { key: "title", label: "Title" },
            { key: "status", label: "Status", render: (r: Record<string, unknown>) => <StatusBadge status={String(r.status ?? "pending")} dot /> },
            { key: "price", label: "Price", render: (r: Record<string, unknown>) => `$${Number(r.price ?? 0).toLocaleString()}` },
          ]}
          data={listings as Record<string, unknown>[]}
          emptyMessage="No listings for this user."
          rowKey={(r: Record<string, unknown>) => String(r.id)}
        />
      )}

      {tab === "Orders" && (
        <DataTable
          columns={[
            { key: "id", label: "ID", render: (r: Record<string, unknown>) => <span className="font-mono text-xs">{String(r.id)}</span> },
            { key: "amount", label: "Amount", render: (r: Record<string, unknown>) => `$${Number(r.amount ?? r.total ?? 0).toLocaleString()}` },
            { key: "status", label: "Status", render: (r: Record<string, unknown>) => <StatusBadge status={String(r.status ?? "pending")} dot /> },
          ]}
          data={orders as Record<string, unknown>[]}
          emptyMessage="No orders for this user."
          rowKey={(r: Record<string, unknown>) => String(r.id)}
        />
      )}

      {tab === "Activity" && (
        <div className="rounded-xl border border-slate-200 bg-white p-5 text-sm text-slate-400 text-center py-12">
          Activity log coming soon. Check the audit logs page for admin actions.
        </div>
      )}

      {/* Ban Dialog */}
      <ConfirmDialog
        open={banDialog}
        title="Ban User"
        message={`Are you sure you want to ban ${user.name}? They will lose access immediately.`}
        confirmLabel="Ban User"
        variant="danger"
        requireReason
        reasonLabel="Ban Reason"
        onConfirm={(reason) => banMut.mutate(reason)}
        onCancel={() => setBanDialog(false)}
        isLoading={banMut.isPending}
      />

      {/* Role Dialog */}
      {roleDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4" style={{ background: "rgba(0,0,0,0.4)" }} onClick={() => setRoleDialog(false)}>
          <div className="bg-white rounded-xl shadow-xl max-w-sm w-full p-6" onClick={(e) => e.stopPropagation()}>
            <h3 className="text-base font-semibold text-slate-800 mb-3">Change Role</h3>
            <select value={newRole} onChange={(e) => setNewRole(e.target.value)} className="w-full border rounded-lg px-3 py-2 text-sm mb-4">
              <option value="user">User</option>
              <option value="seller">Seller</option>
              <option value="moderator">Moderator</option>
              <option value="admin">Admin</option>
              <option value="super_admin">Super Admin</option>
            </select>
            <div className="flex justify-end gap-2">
              <button onClick={() => setRoleDialog(false)} className="px-4 py-2 text-sm rounded-lg border">Cancel</button>
              <button onClick={() => roleMut.mutate(newRole)} disabled={roleMut.isPending} className="px-4 py-2 text-sm rounded-lg text-white bg-indigo-600 disabled:opacity-50">Save</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
