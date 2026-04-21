import {
  ResponsiveContainer,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
} from "recharts"
import { format, isValid, parseISO } from "date-fns"

interface DataPoint {
  date: string
  revenue: number
}

interface Props {
  data?: DataPoint[] | Record<string, number | { revenue?: number }>
}

function normalizeData(data: Props["data"]): DataPoint[] {
  if (Array.isArray(data)) return data
  if (!data || typeof data !== "object") return []

  return Object.entries(data)
    .map(([date, value]) => {
      if (typeof value === "number") return { date, revenue: value }
      if (value && typeof value === "object" && typeof value.revenue === "number") {
        return { date, revenue: value.revenue }
      }
      return null
    })
    .filter((point): point is DataPoint => Boolean(point))
}

function formatLabel(dateValue: string): string {
  const parsed = parseISO(dateValue)
  if (!isValid(parsed)) return dateValue
  return format(parsed, "MMM d")
}

export function RevenueChart({ data }: Props) {
  const formatted = normalizeData(data).reverse().map((d) => ({
    ...d,
    label: formatLabel(d.date),
  }))

  if (!formatted.length) {
    return (
      <div className="h-48 flex items-center justify-center text-gray-400 text-sm">
        No revenue data available
      </div>
    )
  }

  return (
    <ResponsiveContainer width="100%" height={220}>
      <AreaChart data={formatted} margin={{ top: 5, right: 10, left: 0, bottom: 0 }}>
        <defs>
          <linearGradient id="colorRevenue" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#0071CE" stopOpacity={0.15} />
            <stop offset="95%" stopColor="#0071CE" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="#E2E8F0" />
        <XAxis
          dataKey="label"
          tick={{ fontSize: 11, fill: "#718096" }}
          tickLine={false}
          axisLine={false}
          interval="preserveStartEnd"
        />
        <YAxis
          tick={{ fontSize: 11, fill: "#718096" }}
          tickLine={false}
          axisLine={false}
          tickFormatter={(v) => `$${(v / 1000).toFixed(0)}k`}
        />
        <Tooltip
          formatter={(value: number) => [`$${value.toLocaleString()}`, "Revenue"]}
          labelStyle={{ color: "#1A202C" }}
          contentStyle={{
            border: "1px solid #E2E8F0",
            borderRadius: "8px",
            fontSize: "12px",
          }}
        />
        <Area
          type="monotone"
          dataKey="revenue"
          stroke="#0071CE"
          strokeWidth={2}
          fill="url(#colorRevenue)"
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}
