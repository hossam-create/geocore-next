import { Card } from "@/components/ui/card";
import { LucideIcon, TrendingUp, TrendingDown } from "lucide-react";

interface KPICardProps {
  title: string;
  value: string | number;
  change?: string;
  trend?: "up" | "down" | "neutral";
  icon: LucideIcon;
  colorClass?: string;
}

export function KPICard({ title, value, change, trend, icon: Icon, colorClass = "text-primary bg-primary/10" }: KPICardProps) {
  return (
    <Card className="p-6 border-none shadow-sm hover:shadow-md transition-shadow duration-300 group">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-muted-foreground mb-1">{title}</p>
          <h3 className="text-3xl font-bold text-foreground font-display tracking-tight">{value}</h3>
        </div>
        <div className={`w-12 h-12 rounded-2xl flex items-center justify-center ${colorClass} group-hover:scale-110 transition-transform duration-300`}>
          <Icon className="w-6 h-6" />
        </div>
      </div>
      
      {change && (
        <div className="mt-4 flex items-center gap-1.5 text-sm">
          {trend === "up" && <TrendingUp className="w-4 h-4 text-emerald-500" />}
          {trend === "down" && <TrendingDown className="w-4 h-4 text-destructive" />}
          <span className={trend === "up" ? "text-emerald-500 font-medium" : trend === "down" ? "text-destructive font-medium" : "text-muted-foreground"}>
            {change}
          </span>
          <span className="text-muted-foreground">vs last month</span>
        </div>
      )}
    </Card>
  );
}
