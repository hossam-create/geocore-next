import { useState } from "react";
import { usePayments } from "@/hooks/use-payments";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";
import { format } from "date-fns";

export default function PaymentsPage() {
  const [page, setPage] = useState(1);
  const { data, isLoading } = usePayments(page);

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold font-display tracking-tight text-foreground">Payments & Revenue</h1>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card className="p-6 border-none shadow-sm bg-primary text-primary-foreground">
          <p className="text-primary-foreground/80 font-medium text-sm mb-1">Total Revenue</p>
          <p className="text-3xl font-bold font-display tracking-tight">AED {data?.summary?.total_revenue?.toLocaleString() || 0}</p>
        </Card>
        <Card className="p-6 border-none shadow-sm">
          <p className="text-muted-foreground font-medium text-sm mb-1">This Month</p>
          <p className="text-3xl font-bold font-display tracking-tight text-foreground">AED {data?.summary?.this_month?.toLocaleString() || 0}</p>
        </Card>
        <Card className="p-6 border-none shadow-sm">
          <p className="text-muted-foreground font-medium text-sm mb-1">This Week</p>
          <p className="text-3xl font-bold font-display tracking-tight text-foreground">AED {data?.summary?.this_week?.toLocaleString() || 0}</p>
        </Card>
        <Card className="p-6 border-none shadow-sm">
          <p className="text-muted-foreground font-medium text-sm mb-1">Avg Transaction</p>
          <p className="text-3xl font-bold font-display tracking-tight text-foreground">AED {data?.summary?.avg_transaction?.toLocaleString() || 0}</p>
        </Card>
      </div>

      <Card className="p-6 border-none shadow-sm">
        <h3 className="font-bold text-lg mb-6 font-display">Revenue by Month</h3>
        <div className="h-[300px] w-full">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={data?.monthly_chart}>
              <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="hsl(var(--border))" />
              <XAxis dataKey="month" tickLine={false} axisLine={false} tick={{fill: 'hsl(var(--muted-foreground))', fontSize: 12}} />
              <YAxis tickLine={false} axisLine={false} tick={{fill: 'hsl(var(--muted-foreground))', fontSize: 12}} tickFormatter={(v) => `AED ${v/1000}k`} />
              <Tooltip 
                cursor={{fill: 'hsl(var(--muted)/0.5)'}}
                contentStyle={{ backgroundColor: 'hsl(var(--card))', borderRadius: '8px', border: 'none', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)' }}
              />
              <Bar dataKey="revenue" fill="hsl(var(--primary))" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </Card>

      <Card className="border-none shadow-sm overflow-hidden">
        <table className="w-full text-sm text-left">
          <thead className="bg-muted/50 text-muted-foreground uppercase text-xs font-semibold">
            <tr>
              <th className="p-4">Transaction ID</th>
              <th className="p-4">User</th>
              <th className="p-4">Type</th>
              <th className="p-4">Amount</th>
              <th className="p-4">Status</th>
              <th className="p-4 text-right">Date</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {isLoading ? <tr><td colSpan={6} className="p-8 text-center">Loading...</td></tr> : data?.data?.map((payment: any) => (
              <tr key={payment.id} className="hover:bg-muted/20">
                <td className="p-4 font-mono text-xs text-muted-foreground">{payment.id}</td>
                <td className="p-4">
                  <p className="font-semibold text-foreground">{payment.user.name}</p>
                  <p className="text-xs text-muted-foreground">{payment.user.email}</p>
                </td>
                <td className="p-4">
                  <Badge variant="outline" className="capitalize bg-background">{payment.type.replace('_', ' ')}</Badge>
                </td>
                <td className="p-4 font-bold text-foreground">{payment.currency} {payment.amount.toLocaleString()}</td>
                <td className="p-4">
                  {payment.status === 'completed' && <Badge className="bg-emerald-500">Completed</Badge>}
                  {payment.status === 'pending' && <Badge className="bg-amber-500">Pending</Badge>}
                  {payment.status === 'failed' && <Badge variant="destructive">Failed</Badge>}
                </td>
                <td className="p-4 text-right text-muted-foreground">
                  {format(new Date(payment.created_at), "MMM d, yyyy HH:mm")}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>
    </div>
  );
}
