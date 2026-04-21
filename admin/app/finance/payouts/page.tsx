"use client";

import { useQuery } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import DataTable from "@/components/shared/DataTable";
import { escrowApi } from "@/lib/api";
import { DollarSign, Clock, CheckCircle, AlertCircle } from "lucide-react";
import StatCard from "@/components/shared/StatCard";

export default function PayoutsPage() {
  const { data: payouts, isLoading } = useQuery({
    queryKey: ["payouts"],
    queryFn: () => escrowApi.list(),
  });

  const columns = [
    { key: "id", label: "ID", render: (row: Record<string, unknown>) => String(row.id ?? "").slice(0, 8) },
    { key: "seller_name", label: "Seller" },
    { key: "amount", label: "Amount", render: (row: Record<string, unknown>) => `$${Number(row.amount || 0).toFixed(2)}` },
    { key: "method", label: "Method" },
    { key: "status", label: "Status", render: (row: Record<string, unknown>) => {
      const v = String(row.status || "");
      const colors: Record<string, string> = { pending: "text-amber-600", completed: "text-green-600", failed: "text-red-600", on_hold: "text-gray-500" };
      return <span className={colors[v] || "text-gray-600"}>{v.replace(/_/g, " ")}</span>;
    }},
    { key: "created_at", label: "Requested", render: (row: Record<string, unknown>) => row.created_at ? new Date(String(row.created_at)).toLocaleDateString() : "—" },
    { key: "processed_at", label: "Processed", render: (row: Record<string, unknown>) => row.processed_at ? new Date(String(row.processed_at)).toLocaleDateString() : "—" },
  ];

  return (
    <div className="space-y-6">
      <PageHeader title="Payouts" />

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatCard label="Pending" value="—" icon={<Clock className="w-5 h-5 text-amber-500" />} />
        <StatCard label="Completed" value="—" icon={<CheckCircle className="w-5 h-5 text-green-500" />} />
        <StatCard label="Failed" value="—" icon={<AlertCircle className="w-5 h-5 text-red-500" />} />
        <StatCard label="On Hold" value="—" icon={<DollarSign className="w-5 h-5 text-gray-500" />} />
      </div>

      <DataTable
        data={(payouts || []) as Record<string, unknown>[]}
        columns={columns}
        isLoading={isLoading}
      />
    </div>
  );
}
