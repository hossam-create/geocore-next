import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Search, Download, Check, X, Eye, Star, CheckCircle, XCircle } from "lucide-react"
import { format } from "date-fns"
import { api } from "@/api/client"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { useAuthStore } from "@/store/auth"
import { PERMISSIONS, hasPermission } from "@/lib/permissions"

const STATUS_TABS = ["pending", "active", "sold", "expired", "rejected"] as const

export function ListingsPage() {
  const queryClient = useQueryClient()
  const [status, setStatus] = useState<string>("pending")
  const [search, setSearch] = useState("")
  const [page, setPage] = useState(1)
  const [selected, setSelected] = useState<string[]>([])
  const role = useAuthStore((state) => state.user?.role)
  const canModerateListings = hasPermission(role, PERMISSIONS.LISTINGS_MODERATE)

  const { data, isLoading } = useQuery({
    queryKey: ["admin", "listings", status, search, page],
    queryFn: () =>
      api
        .get(`/admin/listings?status=${status}&q=${search}&page=${page}&per_page=25`)
        .then((r) => r.data),
  })

  const approveMutation = useMutation({
    mutationFn: (id: string) => api.put(`/admin/listings/${id}/approve`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["admin", "listings"] }),
  })

  const rejectMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      api.put(`/admin/listings/${id}/reject`, { reason }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["admin", "listings"] }),
  })

  const bulkApproveMutation = useMutation({
    mutationFn: (ids: string[]) =>
      Promise.all(ids.map((id) => api.put(`/admin/listings/${id}/approve`))),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin", "listings"] })
      setSelected([])
    },
  })

  const listings: {
    id: string
    title: string
    images?: { url: string }[]
    user?: { name: string }
    category?: { name_en: string }
    price?: number
    currency?: string
    type?: string
    city?: string
    country?: string
    status?: string
    created_at: string
  }[] = data?.data ?? []

  const meta = data?.meta

  const toggleSelect = (id: string) => {
    setSelected((prev) =>
      prev.includes(id) ? prev.filter((i) => i !== id) : [...prev, id]
    )
  }

  const toggleAll = (checked: boolean) => {
    setSelected(checked ? listings.map((l) => l.id) : [])
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Listings</h1>
          <p className="text-gray-500 text-sm mt-0.5">
            {meta?.total?.toLocaleString() ?? "—"} total listings
          </p>
        </div>
        <Button
          variant="outline"
          onClick={() => window.open("/api/v1/admin/transactions?export=csv")}
        >
          <Download className="w-4 h-4" />
          Export CSV
        </Button>
      </div>

      {/* Filters */}
      <Card className="p-4">
        <div className="flex flex-wrap items-center gap-3">
          {/* Status tabs */}
          <div className="flex gap-1 bg-gray-100 p-1 rounded-lg">
            {STATUS_TABS.map((s) => (
              <button
                key={s}
                onClick={() => { setStatus(s); setPage(1) }}
                className={`px-3 py-1.5 rounded-md text-sm font-medium capitalize transition-colors ${
                  status === s
                    ? "bg-white shadow text-gray-900"
                    : "text-gray-500 hover:text-gray-700"
                }`}
              >
                {s}
              </button>
            ))}
          </div>

          {/* Search */}
          <div className="flex-1 min-w-48 relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
            <Input
              placeholder="Search listings..."
              value={search}
              onChange={(e) => { setSearch(e.target.value); setPage(1) }}
              className="pl-9"
            />
          </div>
        </div>
      </Card>

      {/* Bulk actions */}
      {canModerateListings && selected.length > 0 && (
        <div className="bg-blue-50 border border-blue-100 rounded-xl p-3 flex items-center gap-3">
          <span className="text-[#0071CE] text-sm font-medium">{selected.length} selected</span>
          <Button size="sm" onClick={() => bulkApproveMutation.mutate(selected)}>
            <CheckCircle className="w-3 h-3" /> Approve All
          </Button>
          <Button size="sm" variant="destructive" onClick={() => {
            const reason = prompt("Rejection reason:")
            if (reason) selected.forEach((id) => rejectMutation.mutate({ id, reason }))
            setSelected([])
          }}>
            <XCircle className="w-3 h-3" /> Reject All
          </Button>
          <button onClick={() => setSelected([])} className="text-gray-500 text-sm ml-auto hover:text-gray-700">
            Clear selection
          </button>
        </div>
      )}

      {/* Table */}
      <Card className="overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-100">
              <tr>
                <th className="p-3 w-10">
                  {canModerateListings && (
                    <input
                      type="checkbox"
                      checked={selected.length === listings.length && listings.length > 0}
                      onChange={(e) => toggleAll(e.target.checked)}
                      className="rounded"
                    />
                  )}
                </th>
                {["Listing", "Seller", "Category", "Price", "Type", "Location", "Date", "Actions"].map((h) => (
                  <th key={h} className="p-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
              {isLoading &&
                Array.from({ length: 5 }).map((_, i) => (
                  <tr key={i}>
                    {Array.from({ length: 9 }).map((_, j) => (
                      <td key={j} className="p-3">
                        <Skeleton className="h-4 w-full" />
                      </td>
                    ))}
                  </tr>
                ))}
              {!isLoading && listings.length === 0 && (
                <tr>
                  <td colSpan={9} className="p-12 text-center text-gray-400 text-sm">
                    No listings found
                  </td>
                </tr>
              )}
              {listings.map((listing) => (
                <tr key={listing.id} className="hover:bg-gray-50 transition-colors">
                  <td className="p-3">
                    {canModerateListings && (
                      <input
                        type="checkbox"
                        checked={selected.includes(listing.id)}
                        onChange={() => toggleSelect(listing.id)}
                        className="rounded"
                      />
                    )}
                  </td>
                  <td className="p-3">
                    <div className="flex items-center gap-3">
                      <img
                        src={listing.images?.[0]?.url || `https://picsum.photos/40?random=${listing.id}`}
                        className="w-10 h-10 rounded-lg object-cover shrink-0"
                        alt=""
                      />
                      <div>
                        <p className="text-sm font-medium text-gray-900 line-clamp-1 max-w-[200px]">
                          {listing.title}
                        </p>
                        <p className="text-xs text-gray-400">#{listing.id.slice(0, 8)}</p>
                      </div>
                    </div>
                  </td>
                  <td className="p-3 text-sm text-gray-600">{listing.user?.name ?? "—"}</td>
                  <td className="p-3 text-sm text-gray-600">{listing.category?.name_en ?? "—"}</td>
                  <td className="p-3 text-sm font-medium text-gray-900">
                    {listing.price
                      ? `${listing.currency} ${Number(listing.price).toLocaleString()}`
                      : "—"}
                  </td>
                  <td className="p-3">
                    <Badge variant={listing.type === "auction" ? "destructive" : "secondary"}>
                      {listing.type ?? "sale"}
                    </Badge>
                  </td>
                  <td className="p-3 text-sm text-gray-500">
                    {listing.city}, {listing.country}
                  </td>
                  <td className="p-3 text-xs text-gray-400">
                    {format(new Date(listing.created_at), "MMM d, yyyy")}
                  </td>
                  <td className="p-3">
                    <div className="flex items-center gap-1">
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => window.open(`/listings/${listing.id}`, "_blank")}
                        title="View"
                      >
                        <Eye className="w-3 h-3" />
                      </Button>
                      {canModerateListings && listing.status === "pending" && (
                        <>
                          <Button
                            size="sm"
                            className="bg-green-500 hover:bg-green-600 h-7 w-7 p-0"
                            onClick={() => approveMutation.mutate(listing.id)}
                            title="Approve"
                          >
                            <Check className="w-3 h-3" />
                          </Button>
                          <Button
                            size="sm"
                            variant="destructive"
                            className="h-7 w-7 p-0"
                            onClick={() => {
                              const reason = prompt("Rejection reason:")
                              if (reason) rejectMutation.mutate({ id: listing.id, reason })
                            }}
                            title="Reject"
                          >
                            <X className="w-3 h-3" />
                          </Button>
                        </>
                      )}
                      <Button
                        size="sm"
                        variant="ghost"
                        className="text-yellow-500 h-7 w-7 p-0"
                        title="Feature"
                      >
                        <Star className="w-3 h-3" />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {meta && meta.pages > 1 && (
          <div className="p-4 border-t border-gray-100 flex items-center justify-between">
            <p className="text-sm text-gray-500">
              Page {meta.page} of {meta.pages} ({meta.total.toLocaleString()} total)
            </p>
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="outline"
                disabled={page === 1}
                onClick={() => setPage((p) => p - 1)}
              >
                Previous
              </Button>
              <Button
                size="sm"
                variant="outline"
                disabled={page >= meta.pages}
                onClick={() => setPage((p) => p + 1)}
              >
                Next
              </Button>
            </div>
          </div>
        )}
      </Card>
    </div>
  )
}
