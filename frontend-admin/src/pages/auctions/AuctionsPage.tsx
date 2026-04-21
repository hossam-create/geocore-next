import { useQuery } from "@tanstack/react-query"
import { api } from "@/api/client"
import { Card } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { formatCurrency } from "@/lib/utils"
import { format } from "date-fns"

export function AuctionsPage() {
  const { data, isLoading } = useQuery({
    queryKey: ["admin", "auctions", "active"],
    queryFn: () =>
      api.get("/admin/listings?type=auction&per_page=50").then((r) => r.data),
    refetchInterval: 15000,
  })

  const auctions: {
    id: string
    title: string
    user?: { name: string }
    price?: number
    currency?: string
    status?: string
    created_at: string
    images?: { url: string }[]
  }[] = data?.data ?? []

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Auctions</h1>
        <p className="text-gray-500 text-sm mt-0.5">
          {data?.meta?.total?.toLocaleString() ?? "—"} auction listings · refreshes every 15s
        </p>
      </div>

      <Card className="overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-100">
              <tr>
                {["Listing", "Seller", "Starting Price", "Status", "Date"].map((h) => (
                  <th key={h} className="p-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
              {isLoading && Array.from({ length: 5 }).map((_, i) => (
                <tr key={i}>
                  {Array.from({ length: 5 }).map((_, j) => (
                    <td key={j} className="p-3"><Skeleton className="h-4 w-full" /></td>
                  ))}
                </tr>
              ))}
              {!isLoading && auctions.length === 0 && (
                <tr>
                  <td colSpan={5} className="p-12 text-center text-gray-400 text-sm">No auction listings found</td>
                </tr>
              )}
              {auctions.map((auction) => (
                <tr key={auction.id} className="hover:bg-gray-50 transition-colors">
                  <td className="p-3">
                    <div className="flex items-center gap-3">
                      <img
                        src={auction.images?.[0]?.url || `https://picsum.photos/40?random=${auction.id}`}
                        className="w-10 h-10 rounded-lg object-cover shrink-0"
                        alt=""
                      />
                      <div>
                        <p className="text-sm font-medium text-gray-900 line-clamp-1 max-w-[200px]">{auction.title}</p>
                        <p className="text-xs text-gray-400">#{auction.id.slice(0, 8)}</p>
                      </div>
                    </div>
                  </td>
                  <td className="p-3 text-sm text-gray-600">{auction.user?.name ?? "—"}</td>
                  <td className="p-3 text-sm font-semibold text-gray-900">
                    {auction.price ? formatCurrency(auction.price, auction.currency) : "—"}
                  </td>
                  <td className="p-3">
                    <Badge variant={auction.status === "active" ? "success" : "secondary"}>
                      {auction.status ?? "unknown"}
                    </Badge>
                  </td>
                  <td className="p-3 text-xs text-gray-400">
                    {format(new Date(auction.created_at), "MMM d, yyyy")}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  )
}
