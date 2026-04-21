"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import DataTable from "@/components/shared/DataTable";
import { mockUsers } from "@/lib/mockData";
import { usersApi } from "@/lib/api";
import { useToastStore } from "@/lib/toast";
import { maskEmail, timeAgo } from "@/lib/format";

type UserRow = {
  id: string;
  name: string;
  email: string;
  role: string;
  status: string;
  joined: string;
};

function normalizeUsers(payload: unknown): UserRow[] {
  const box = payload as
    | { data?: Array<Record<string, unknown>>; meta?: unknown }
    | Array<Record<string, unknown>>
    | null
    | undefined;

  const rows = Array.isArray(box) ? box : Array.isArray(box?.data) ? box.data : [];
  return rows
    .map((item) => ({
      id: String(item.id ?? ""),
      name: String(item.name ?? "Unknown"),
      email: String(item.email ?? "—"),
      role: String(item.role ?? "user"),
      status: item.is_banned ? "banned" : item.is_active === false ? "inactive" : "active",
      joined: String(item.created_at ?? new Date().toISOString()),
    }))
    .filter((x) => x.id);
}

export default function AdminUsersPage() {
  const qc = useQueryClient();
  const showToast = useToastStore((s) => s.showToast);

  const { data: liveUsers, isLoading } = useQuery({
    queryKey: ["admin", "users"],
    queryFn: async () => {
      const res = await usersApi.list();
      return normalizeUsers(res);
    },
    retry: 1,
  });

  const users = liveUsers?.length
    ? liveUsers
    : mockUsers.map((u) => ({
        id: u.id,
        name: u.name,
        email: u.email,
        role: u.role,
        status: u.status,
        joined: u.joined,
      }));

  const banMutation = useMutation({
    mutationFn: (id: string) => usersApi.ban(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "users"] });
      showToast({ type: "success", title: "User banned", message: "The account was blocked successfully." });
    },
    onError: (error: { message?: string }) => {
      showToast({ type: "error", title: "Ban failed", message: error?.message ?? "Could not ban this user." });
    },
  });

  const unbanMutation = useMutation({
    mutationFn: (id: string) => usersApi.unban(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "users"] });
      showToast({ type: "success", title: "User unbanned", message: "The account is active again." });
    },
    onError: (error: { message?: string }) => {
      showToast({ type: "error", title: "Unban failed", message: error?.message ?? "Could not unban this user." });
    },
  });

  const mutationBusy = banMutation.isPending || unbanMutation.isPending;

  return (
    <div>
      <PageHeader title="Users" description="Manage platform users and account states" />
      <DataTable
        columns={[
          { key: "id", label: "ID", render: (u: UserRow) => <span className="font-mono text-xs">{u.id}</span> },
          { key: "name", label: "Name" },
          { key: "email", label: "Email", render: (u: UserRow) => <span className="font-mono text-xs" style={{ color: "var(--text-secondary)" }}>{maskEmail(u.email)}</span> },
          { key: "role", label: "Role", render: (u: UserRow) => <StatusBadge status={u.role} variant="brand" /> },
          { key: "status", label: "Status", render: (u: UserRow) => <StatusBadge status={u.status} dot /> },
          { key: "joined", label: "Joined", render: (u: UserRow) => <span className="text-xs" style={{ color: "var(--text-tertiary)" }}>{timeAgo(u.joined)}</span> },
          {
            key: "actions",
            label: "Actions",
            render: (u: UserRow) =>
              u.status === "banned" ? (
                <button
                  className="text-xs px-2.5 py-1 rounded-md"
                  style={{ background: "var(--color-success-light)", color: "var(--color-success)" }}
                  disabled={mutationBusy}
                  onClick={() => unbanMutation.mutate(u.id)}
                >
                  {unbanMutation.isPending ? "Unbanning..." : "Unban"}
                </button>
              ) : (
                <button
                  className="text-xs px-2.5 py-1 rounded-md"
                  style={{ background: "var(--color-danger-light)", color: "var(--color-danger)" }}
                  disabled={mutationBusy}
                  onClick={() => banMutation.mutate(u.id)}
                >
                  {banMutation.isPending ? "Banning..." : "Ban"}
                </button>
              ),
          },
        ]}
        data={users}
        isLoading={isLoading}
        loadingMessage="Loading users..."
        emptyMessage="No users found."
        rowKey={(u) => u.id}
      />
    </div>
  );
}
