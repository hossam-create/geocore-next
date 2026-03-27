import { useDashboardStats } from "@/hooks/use-dashboard";
import { KPICard } from "@/components/KPICard";
import { DollarSign, Tag, Users, Hammer, CheckCircle, XCircle } from "lucide-react";
import { Card } from "@/components/ui/card";
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell, Legend } from "recharts";
import { Button } from "@/components/ui/button";
import { Link } from "wouter";
import { PageLayout } from "@/components/layout";
import { format } from "date-fns";
import { Skeleton } from "@/components/ui/skeleton";

const PIE_COLORS = ['#0071CE', '#FFC220', '#10B981', '#8B5CF6', '#F97316'];

export default function Dashboard() {
  const { data: stats, isLoading } = useDashboardStats();

  return (
    <PageLayout title="Dashboard" subtitle="Overview of your marketplace performance">
      {/* KPI Row */}
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-6">
        {isLoading ? (
          Array.from({length: 4}).map((_, i) => <Skeleton key={i} className="h-32 rounded-xl" />)
        ) : (
          <>
            <KPICard
              title="Revenue (30d)"
              value={`AED ${stats?.total_revenue.toLocaleString()}`}
              change="+12.5%"
              trend="up"
              icon={DollarSign}
              colorClass="text-blue-600 bg-blue-100 dark:bg-blue-900/30"
            />
            <KPICard
              title="Active Listings"
              value={stats?.active_listings.toLocaleString() || 0}
              change="+8.2%"
              trend="up"
              icon={Tag}
              colorClass="text-emerald-600 bg-emerald-100 dark:bg-emerald-900/30"
            />
            <KPICard
              title="New Users (7d)"
              value={stats?.new_users_today || 0}
              change="+3.1%"
              trend="up"
              icon={Users}
              colorClass="text-purple-600 bg-purple-100 dark:bg-purple-900/30"
            />
            <KPICard
              title="Live Auctions"
              value={stats?.active_auctions.toLocaleString() || 0}
              change="-2.4%"
              trend="down"
              icon={Hammer}
              colorClass="text-orange-600 bg-orange-100 dark:bg-orange-900/30"
            />
          </>
        )}
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <Card className="p-6 col-span-2 border-border shadow-sm">
          <h3 className="font-semibold text-lg mb-6">Revenue (Last 30 Days)</h3>
          {isLoading ? (
            <Skeleton className="h-[300px] w-full" />
          ) : (
            <div className="h-[300px] w-full">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={stats?.revenue_chart}>
                  <defs>
                    <linearGradient id="colorRev" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="hsl(var(--primary))" stopOpacity={0.3}/>
                      <stop offset="95%" stopColor="hsl(var(--primary))" stopOpacity={0}/>
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="hsl(var(--border))" />
                  <XAxis dataKey="date" tickLine={false} axisLine={false} tick={{fill: 'hsl(var(--muted-foreground))', fontSize: 12}} tickFormatter={(v) => new Date(v).getDate().toString()} />
                  <YAxis tickLine={false} axisLine={false} tick={{fill: 'hsl(var(--muted-foreground))', fontSize: 12}} tickFormatter={(v) => `AED ${v/1000}k`} />
                  <Tooltip 
                    contentStyle={{ backgroundColor: 'hsl(var(--card))', borderRadius: '8px', border: '1px solid hsl(var(--border))', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)' }}
                    itemStyle={{ color: 'hsl(var(--foreground))', fontWeight: 'bold' }}
                  />
                  <Area type="monotone" dataKey="revenue" stroke="hsl(var(--primary))" strokeWidth={3} fillOpacity={1} fill="url(#colorRev)" />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          )}
        </Card>

        <Card className="p-6 border-border shadow-sm flex flex-col">
          <h3 className="font-semibold text-lg mb-2">Listings by Category</h3>
          {isLoading ? (
            <Skeleton className="flex-1 min-h-[300px]" />
          ) : (
            <div className="flex-1 min-h-[300px]">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={stats?.listings_by_category}
                    cx="50%"
                    cy="50%"
                    innerRadius={60}
                    outerRadius={90}
                    paddingAngle={2}
                    dataKey="value"
                    stroke="none"
                  >
                    {stats?.listings_by_category.map((_, index) => (
                      <Cell key={`cell-${index}`} fill={PIE_COLORS[index % PIE_COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip contentStyle={{ borderRadius: '8px', border: '1px solid hsl(var(--border))' }} />
                  <Legend iconType="circle" />
                </PieChart>
              </ResponsiveContainer>
            </div>
          )}
        </Card>
      </div>

      {/* Activity Row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <Card className="col-span-2 border-border shadow-sm overflow-hidden flex flex-col">
          <div className="p-6 border-b border-border flex items-center justify-between">
            <h3 className="font-semibold text-lg">Pending Approvals</h3>
            <Button variant="outline" size="sm" asChild>
              <Link href="/listings">View All</Link>
            </Button>
          </div>
          <div className="flex-1 overflow-x-auto">
            <table className="w-full text-sm text-left">
              <thead className="bg-muted/50 text-muted-foreground font-medium">
                <tr>
                  <th className="px-6 py-3">Listing</th>
                  <th className="px-6 py-3">Seller</th>
                  <th className="px-6 py-3">Date</th>
                  <th className="px-6 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {isLoading ? (
                  <tr><td colSpan={4} className="p-6 text-center text-muted-foreground">Loading...</td></tr>
                ) : (
                  [1,2,3,4].map((i) => (
                    <tr key={i} className="hover:bg-muted/30 transition-colors group">
                      <td className="px-6 py-4">
                        <div className="flex items-center gap-3">
                          <img src={`https://picsum.photos/seed/${i + 10}/100`} alt="" className="w-10 h-10 rounded object-cover bg-muted" />
                          <div>
                            <div className="font-medium text-foreground">Premium Item #{i}</div>
                            <div className="text-xs text-muted-foreground">Electronics</div>
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-4 text-muted-foreground">Ahmed M.</td>
                      <td className="px-6 py-4 text-muted-foreground">{format(new Date(), "MMM d, yyyy")}</td>
                      <td className="px-6 py-4 text-right">
                        <div className="flex items-center justify-end gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                          <Button size="sm" variant="ghost" className="h-8 text-emerald-600 hover:text-emerald-700 hover:bg-emerald-50">
                            <CheckCircle className="w-4 h-4 mr-1" /> Approve
                          </Button>
                          <Button size="sm" variant="ghost" className="h-8 text-destructive hover:text-destructive hover:bg-destructive/10">
                            <XCircle className="w-4 h-4 mr-1" /> Reject
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </Card>

        <Card className="p-6 border-border shadow-sm">
          <h3 className="font-semibold text-lg mb-6">Today's Activity</h3>
          {isLoading ? (
            <div className="space-y-4">
              {Array.from({length: 5}).map((_, i) => <Skeleton key={i} className="h-10 w-full" />)}
            </div>
          ) : (
            <div className="space-y-6">
              {[
                { label: "Revenue", value: `AED ${stats?.revenue_today?.toLocaleString()}`, color: "text-blue-600" },
                { label: "New Listings", value: stats?.new_listings_today, color: "text-emerald-600" },
                { label: "Bids Placed", value: stats?.bids_today, color: "text-orange-600" },
                { label: "New Users", value: stats?.new_users_today, color: "text-purple-600" },
                { label: "Open Reports", value: stats?.open_reports, color: "text-destructive" },
              ].map(item => (
                <div key={item.label} className="flex items-center justify-between border-b border-border pb-4 last:border-0 last:pb-0">
                  <span className="text-muted-foreground font-medium">{item.label}</span>
                  <span className={`font-semibold ${item.color}`}>{item.value}</span>
                </div>
              ))}
            </div>
          )}
        </Card>
      </div>
    </PageLayout>
  );
}
