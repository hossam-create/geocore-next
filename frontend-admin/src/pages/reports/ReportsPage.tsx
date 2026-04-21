import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/api/client"
import { ReadOnlyNotice } from "@/components/authz/ReadOnlyNotice"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Flag, CheckCircle, XCircle, Zap, ChevronDown, ChevronUp, Loader2 } from "lucide-react"
import { format } from "date-fns"
import { useAuthStore } from "@/store/auth"
import { PERMISSIONS, hasAnyPermission, hasPermission } from "@/lib/permissions"

type ReportStatus = "pending" | "reviewed" | "dismissed" | "actioned"

interface Report {
  id: string
  reporter_id: string
  reporter_name?: string
  target_type: "listing" | "user"
  target_id: string
  reason: string
  description?: string
  status: ReportStatus
  admin_note?: string
  reviewed_at?: string
  created_at: string
}

const STATUS_COLORS: Record<ReportStatus, string> = {
  pending:   "bg-yellow-100 text-yellow-700 border-yellow-200",
  reviewed:  "bg-blue-100 text-blue-700 border-blue-200",
  dismissed: "bg-gray-100 text-gray-600 border-gray-200",
  actioned:  "bg-red-100 text-red-700 border-red-200",
}

const STATUSES: ReportStatus[] = ["pending", "reviewed", "dismissed", "actioned"]

export function ReportsPage() {
  const qc = useQueryClient()
  const [filterStatus, setFilterStatus] = useState<ReportStatus | "">("")
  const [expanded, setExpanded] = useState<string | null>(null)
  const [noteMap, setNoteMap] = useState<Record<string, string>>({})
  const role = useAuthStore((state) => state.user?.role)
  const canReviewReports = hasPermission(role, PERMISSIONS.REPORTS_REVIEW)

  const canTakeAction = (report: Report): boolean => {
    if (report.target_type === "listing") {
      return hasPermission(role, PERMISSIONS.LISTINGS_MODERATE)
    }
    if (report.target_type === "user") {
      return hasAnyPermission(role, [PERMISSIONS.USERS_BAN, PERMISSIONS.USERS_WRITE])
    }
    return false
  }

  const { data, isLoading } = useQuery<{ data: Report[]; meta: { total: number } }>({
    queryKey: ["admin-reports", filterStatus],
    queryFn: () =>
      api.get("/api/v1/admin/reports", { params: { status: filterStatus || undefined } })
         .then((r: { data: unknown }) => r.data as { data: Report[]; meta: { total: number } }),
  })

  const review = useMutation({
    mutationFn: ({ id, status, admin_note }: { id: string; status: string; admin_note?: string }) =>
      api.patch(`/api/v1/admin/reports/${id}`, { status, admin_note }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin-reports"] })
      setExpanded(null)
    },
  })

  const reports = data?.data ?? []
  const total = data?.meta?.total ?? 0

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Reports Queue</h1>
          <p className="text-gray-500 text-sm mt-0.5">
            {total} report{total !== 1 ? "s" : ""} total
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setFilterStatus("")}
            className={`px-3 py-1.5 rounded-lg text-xs font-medium border transition-colors ${filterStatus === "" ? "bg-gray-900 text-white border-gray-900" : "bg-white text-gray-600 border-gray-200 hover:bg-gray-50"}`}
          >
            All
          </button>
          {STATUSES.map(s => (
            <button
              key={s}
              onClick={() => setFilterStatus(s)}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium border transition-colors capitalize ${filterStatus === s ? "bg-gray-900 text-white border-gray-900" : "bg-white text-gray-600 border-gray-200 hover:bg-gray-50"}`}
            >
              {s}
            </button>
          ))}
        </div>
      </div>

      <Card className="divide-y divide-gray-100 overflow-hidden p-0">
        {isLoading ? (
          <div className="p-4 space-y-3">
            {[1, 2, 3].map(i => <Skeleton key={i} className="h-14 w-full rounded-lg" />)}
          </div>
        ) : reports.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-gray-400 gap-2">
            <Flag className="w-8 h-8" />
            <p className="text-sm">No reports found</p>
          </div>
        ) : (
          reports.map(r => {
            const isOpen = expanded === r.id
            return (
              <div key={r.id} className="p-4">
                <div
                  className="flex items-start justify-between cursor-pointer gap-3"
                  onClick={() => setExpanded(isOpen ? null : r.id)}
                >
                  <div className="flex items-start gap-3 min-w-0">
                    <div className={`mt-0.5 px-2 py-0.5 rounded-full border text-xs font-medium capitalize flex-shrink-0 ${STATUS_COLORS[r.status]}`}>
                      {r.status}
                    </div>
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-gray-900 truncate">
                        <span className="text-gray-500 capitalize">{r.target_type}</span> — {r.reason}
                      </p>
                      <p className="text-xs text-gray-400 mt-0.5">
                        by {r.reporter_name ?? r.reporter_id.slice(0, 8)} · {format(new Date(r.created_at), "MMM d, yyyy HH:mm")}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 flex-shrink-0">
                    <Badge variant="outline" className="text-xs capitalize">{r.target_type}</Badge>
                    {isOpen ? <ChevronUp className="w-4 h-4 text-gray-400" /> : <ChevronDown className="w-4 h-4 text-gray-400" />}
                  </div>
                </div>

                {isOpen && (
                  <div className="mt-3 pl-16 space-y-3">
                    {r.description && (
                      <p className="text-sm text-gray-600 bg-gray-50 rounded-lg p-3">{r.description}</p>
                    )}
                    <p className="text-xs text-gray-400 font-mono">Target ID: {r.target_id}</p>
                    {r.admin_note && (
                      <p className="text-xs text-blue-700 bg-blue-50 rounded p-2">Admin note: {r.admin_note}</p>
                    )}
                    {r.status === "pending" && canReviewReports && (
                      <div className="space-y-2">
                        <textarea
                          placeholder="Admin note (optional)"
                          rows={2}
                          value={noteMap[r.id] ?? ""}
                          onChange={e => setNoteMap(m => ({ ...m, [r.id]: e.target.value }))}
                          className="w-full px-3 py-2 border border-input rounded-md text-xs focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring resize-none"
                        />
                        <div className="flex gap-2">
                          {canTakeAction(r) && (
                            <Button
                              size="sm"
                              variant="outline"
                              className="text-red-600 border-red-200 hover:bg-red-50 flex items-center gap-1.5"
                              disabled={review.isPending}
                              onClick={() => review.mutate({ id: r.id, status: "actioned", admin_note: noteMap[r.id] })}
                            >
                              {review.isPending ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Zap className="w-3.5 h-3.5" />}
                              Take Action
                            </Button>
                          )}
                          <Button
                            size="sm"
                            variant="outline"
                            className="flex items-center gap-1.5"
                            disabled={review.isPending}
                            onClick={() => review.mutate({ id: r.id, status: "reviewed", admin_note: noteMap[r.id] })}
                          >
                            <CheckCircle className="w-3.5 h-3.5" /> Mark Reviewed
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            className="text-gray-500 flex items-center gap-1.5"
                            disabled={review.isPending}
                            onClick={() => review.mutate({ id: r.id, status: "dismissed", admin_note: noteMap[r.id] })}
                          >
                            <XCircle className="w-3.5 h-3.5" /> Dismiss
                          </Button>
                        </div>
                      </div>
                    )}
                    {r.status === "pending" && !canReviewReports && (
                      <ReadOnlyNotice />
                    )}
                  </div>
                )}
              </div>
            )
          })
        )}
      </Card>
    </div>
  )
}
