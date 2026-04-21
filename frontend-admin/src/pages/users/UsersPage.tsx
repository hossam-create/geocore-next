import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Search, Ban, CheckCircle, Trash2, UserCog } from "lucide-react"
import { format } from "date-fns"
import { api } from "@/api/client"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { useAuthStore } from "@/store/auth"
import { PERMISSIONS, hasPermission } from "@/lib/permissions"

export function UsersPage() {
  const queryClient = useQueryClient()
  const [search, setSearch] = useState("")
  const [page, setPage] = useState(1)
  const role = useAuthStore((state) => state.user?.role)
  const canBanUsers = hasPermission(role, PERMISSIONS.USERS_BAN)
  const canEditUsers = hasPermission(role, PERMISSIONS.USERS_WRITE)
  const canDeleteUsers = hasPermission(role, PERMISSIONS.USERS_DELETE)

  const { data, isLoading } = useQuery({
    queryKey: ["admin", "users", search, page],
    queryFn: () =>
      api.get(`/admin/users?q=${search}&page=${page}&per_page=25`).then((r) => r.data),
  })

  const banMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      api.post(`/admin/users/${id}/ban`, { reason }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["admin", "users"] }),
  })

  const unbanMutation = useMutation({
    mutationFn: (id: string) => api.post(`/admin/users/${id}/unban`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["admin", "users"] }),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Record<string, unknown> }) =>
      api.put(`/admin/users/${id}`, data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["admin", "users"] }),
  })

  const users: {
    id: string
    name: string
    email: string
    phone?: string
    role: string
    is_banned: boolean
    email_verified: boolean
    is_verified: boolean
    rating: number
    sold_count: number
    created_at: string
  }[] = data?.data ?? []

  const meta = data?.meta

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Users</h1>
          <p className="text-gray-500 text-sm mt-0.5">{meta?.total?.toLocaleString() ?? "—"} total users</p>
        </div>
      </div>

      {/* Search */}
      <Card className="p-4">
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <Input
            placeholder="Search by name or email..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1) }}
            className="pl-9"
          />
        </div>
      </Card>

      <Card className="overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-100">
              <tr>
                {["User", "Email", "Role", "Status", "Rating", "Sold", "Joined", "Actions"].map((h) => (
                  <th key={h} className="p-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
              {isLoading && Array.from({ length: 5 }).map((_, i) => (
                <tr key={i}>
                  {Array.from({ length: 8 }).map((_, j) => (
                    <td key={j} className="p-3"><Skeleton className="h-4 w-full" /></td>
                  ))}
                </tr>
              ))}
              {!isLoading && users.length === 0 && (
                <tr>
                  <td colSpan={8} className="p-12 text-center text-gray-400 text-sm">No users found</td>
                </tr>
              )}
              {users.map((user) => (
                <tr key={user.id} className="hover:bg-gray-50 transition-colors">
                  <td className="p-3">
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 rounded-full bg-[#0071CE]/20 flex items-center justify-center text-[#0071CE] text-sm font-bold shrink-0">
                        {user.name?.[0]?.toUpperCase()}
                      </div>
                      <span className="text-sm font-medium text-gray-900">{user.name}</span>
                    </div>
                  </td>
                  <td className="p-3 text-sm text-gray-600">{user.email}</td>
                  <td className="p-3">
                    <Badge variant={user.role === "admin" ? "default" : "secondary"}>
                      {user.role}
                    </Badge>
                  </td>
                  <td className="p-3">
                    {user.is_banned ? (
                      <Badge variant="destructive">Banned</Badge>
                    ) : user.email_verified ? (
                      <Badge variant="success">Active</Badge>
                    ) : (
                      <Badge variant="warning">Unverified</Badge>
                    )}
                  </td>
                  <td className="p-3 text-sm text-gray-600">⭐ {user.rating?.toFixed(1)}</td>
                  <td className="p-3 text-sm text-gray-600">{user.sold_count}</td>
                  <td className="p-3 text-xs text-gray-400">
                    {format(new Date(user.created_at), "MMM d, yyyy")}
                  </td>
                  <td className="p-3">
                    <div className="flex items-center gap-1">
                      {canBanUsers && user.is_banned ? (
                        <Button size="sm" variant="outline" className="h-7 text-green-600"
                          onClick={() => unbanMutation.mutate(user.id)}>
                          <CheckCircle className="w-3 h-3" /> Unban
                        </Button>
                      ) : canBanUsers ? (
                        <Button size="sm" variant="ghost" className="h-7 text-red-500"
                          onClick={() => {
                            const reason = prompt("Ban reason:")
                            if (reason) banMutation.mutate({ id: user.id, reason })
                          }}>
                          <Ban className="w-3 h-3" /> Ban
                        </Button>
                      ) : null}
                      {canEditUsers && (
                        <Button size="sm" variant="ghost" className="h-7"
                          onClick={() => {
                            const role = prompt("New role (user/admin/super_admin):", user.role)
                            if (role) updateMutation.mutate({ id: user.id, data: { role } })
                          }}>
                          <UserCog className="w-3 h-3" />
                        </Button>
                      )}
                      {canDeleteUsers && (
                        <Button size="sm" variant="ghost" className="h-7 text-red-400"
                          onClick={() => {
                            if (confirm(`Delete user ${user.name}?`))
                              api.delete(`/admin/users/${user.id}`).then(() =>
                                queryClient.invalidateQueries({ queryKey: ["admin", "users"] })
                              )
                          }}>
                          <Trash2 className="w-3 h-3" />
                        </Button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {meta && meta.pages > 1 && (
          <div className="p-4 border-t border-gray-100 flex items-center justify-between">
            <p className="text-sm text-gray-500">Page {meta.page} of {meta.pages}</p>
            <div className="flex gap-2">
              <Button size="sm" variant="outline" disabled={page === 1} onClick={() => setPage(p => p - 1)}>Previous</Button>
              <Button size="sm" variant="outline" disabled={page >= meta.pages} onClick={() => setPage(p => p + 1)}>Next</Button>
            </div>
          </div>
        )}
      </Card>
    </div>
  )
}
