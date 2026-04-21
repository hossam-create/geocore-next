import { Suspense, lazy, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { Download } from "lucide-react"
import { format } from "date-fns"
import { api } from "@/api/client"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { formatCurrency } from "@/lib/utils"

const RevenueChart = lazy(() =>
  import("@/components/charts/RevenueChart").then((mod) => ({ default: mod.RevenueChart }))
)

export function PaymentsPage() {
  const [page, setPage] = useState(1)

  const { data: revenue } = useQuery({
    queryKey: ["admin", "revenue"],
    queryFn: () => api.get("/admin/revenue").then((r) => r.data.data),
  })

  const { data, isLoading } = useQuery({
    queryKey: ["admin", "transactions", page],
    queryFn: () =>
      api.get(`/admin/transactions?page=${page}&per_page=25`).then((r) => r.data),
  })

  const txList: {
    id: string
    user_id: string
    amount: number
    currency: string
    status: string
    kind: string
    description?: string
    stripe_payment_intent_id?: string
    created_at: string
  }[] = data?.data ?? []
  const meta = data?.meta

  const statusColor: Record<string, "success" | "destructive" | "warning" | "secondary"> = {
    succeeded: "success",
    failed: "destructive",
    pending: "warning",
    refunded: "secondary",
    cancelled: "secondary",
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Payments & Revenue</h1>
        <Button
          variant="outline"
          onClick={() => window.open("/api/v1/admin/transactions?export=csv")}
        >
          <Download className="w-4 h-4" /> Export CSV
        </Button>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card className="p-5">
          <p className="text-gray-500 text-sm">Total Revenue</p>
          <p className="text-3xl font-bold text-[#0071CE] mt-1">
            {formatCurrency(revenue?.total)}
          </p>
        </Card>
        <Card className="p-5">
          <p className="text-gray-500 text-sm">Last 30 Days</p>
          <p className="text-3xl font-bold mt-1">
            {formatCurrency(
              revenue?.daily_30days?.reduce(
                (sum: number, d: { revenue: number }) => sum + d.revenue,
                0
              )
            )}
          </p>
        </Card>
        <Card className="p-5">
          <p className="text-gray-500 text-sm">Transactions</p>
          <p className="text-3xl font-bold mt-1">{meta?.total?.toLocaleString() ?? "—"}</p>
        </Card>
      </div>

      {/* Revenue Chart */}
      <Card className="p-6">
        <h3 className="font-semibold text-gray-900 mb-4">Revenue Over Time</h3>
        <Suspense fallback={<Skeleton className="h-[220px] w-full" />}>
          <RevenueChart data={revenue?.daily_30days} />
        </Suspense>
      </Card>

      {/* Transactions */}
      <Card className="overflow-hidden">
        <div className="p-4 border-b border-gray-100">
          <h3 className="font-semibold text-gray-900">Recent Transactions</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-100">
              <tr>
                {["User ID", "Kind", "Amount", "Status", "Date", "Reference"].map((h) => (
                  <th key={h} className="p-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
              {isLoading && Array.from({ length: 5 }).map((_, i) => (
                <tr key={i}>
                  {Array.from({ length: 6 }).map((_, j) => (
                    <td key={j} className="p-3"><Skeleton className="h-4 w-full" /></td>
                  ))}
                </tr>
              ))}
              {txList.map((tx) => (
                <tr key={tx.id} className="hover:bg-gray-50 transition-colors">
                  <td className="p-3 text-sm text-gray-600 font-mono">{tx.user_id.slice(0, 12)}...</td>
                  <td className="p-3">
                    <Badge variant="secondary">{tx.kind}</Badge>
                  </td>
                  <td className="p-3 text-sm font-semibold text-gray-900">
                    {tx.currency} {Number(tx.amount).toLocaleString()}
                  </td>
                  <td className="p-3">
                    <Badge variant={statusColor[tx.status] ?? "secondary"}>
                      {tx.status}
                    </Badge>
                  </td>
                  <td className="p-3 text-xs text-gray-400">
                    {format(new Date(tx.created_at), "MMM d, HH:mm")}
                  </td>
                  <td className="p-3 text-xs text-gray-400 font-mono">
                    {tx.stripe_payment_intent_id?.slice(0, 16)}...
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
