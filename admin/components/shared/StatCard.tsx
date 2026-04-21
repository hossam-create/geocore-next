import { clsx } from "clsx";

interface StatCardProps {
  label: string;
  value: string | number;
  icon: React.ReactNode;
  change?: string;
  trend?: "up" | "down" | "neutral";
  className?: string;
}

export default function StatCard({ label, value, icon, change, trend, className }: StatCardProps) {
  return (
    <div className={clsx("bg-white rounded-xl border border-slate-200 p-5", className)}>
      <div className="flex items-center justify-between mb-3">
        <span className="text-sm text-slate-500 font-medium">{label}</span>
        <div className="text-slate-400">{icon}</div>
      </div>
      <p className="text-2xl font-bold text-slate-900">{typeof value === "number" ? value.toLocaleString() : value}</p>
      {change && (
        <p className={clsx("text-xs mt-1 font-medium", {
          "text-green-600": trend === "up",
          "text-red-500": trend === "down",
          "text-slate-400": trend === "neutral",
        })}>
          {change}
        </p>
      )}
    </div>
  );
}
