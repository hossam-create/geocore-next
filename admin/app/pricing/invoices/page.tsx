"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { invoicesApi } from "@/lib/api";
import PageHeader from "@/components/shared/PageHeader";
import DataTable from "@/components/shared/DataTable";
import StatusBadge from "@/components/shared/StatusBadge";
import FiltersBar from "@/components/shared/FiltersBar";
import { FileText, DollarSign } from "lucide-react";

interface Invoice {
  id: number;
  invoice_number: string;
  user_id: string;
  subtotal: number;
  discount: number;
  tax: number;
  total: number;
  status: string;
  gateway_reference?: string;
  created_at: string;
  paid_at?: string;
  [key: string]: unknown;
}

export default function InvoicesPage() {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("");

  const { data = [], isLoading } = useQuery({
    queryKey: ["invoices"],
    queryFn: () => invoicesApi.list(),
  });

  const invoices: Invoice[] = Array.isArray(data) ? data : [];
  const filtered = invoices.filter((inv) => {
    if (search && !inv.invoice_number?.toLowerCase().includes(search.toLowerCase())) return false;
    if (statusFilter && inv.status !== statusFilter) return false;
    return true;
  });

  const totalRevenue = filtered.reduce((sum, inv) => sum + (inv.status === "paid" ? inv.total : 0), 0);

  const columns = [
    { key: "invoice_number", label: "Invoice #", render: (r: Invoice) => (
      <span className="font-mono text-xs font-semibold text-indigo-600">{r.invoice_number ?? `INV-${r.id}`}</span>
    )},
    { key: "total", label: "Total", render: (r: Invoice) => <span className="font-bold">${r.total?.toFixed(2)}</span> },
    { key: "discount", label: "Discount", render: (r: Invoice) => r.discount > 0 ? <span className="text-green-600">-${r.discount.toFixed(2)}</span> : "—" },
    { key: "status", label: "Status", render: (r: Invoice) => <StatusBadge status={r.status} dot /> },
    { key: "created_at", label: "Date", render: (r: Invoice) => new Date(r.created_at).toLocaleDateString() },
    { key: "paid_at", label: "Paid", render: (r: Invoice) => r.paid_at ? new Date(r.paid_at).toLocaleDateString() : "—" },
  ];

  return (
    <div>
      <PageHeader title="Invoices" description="View and manage all invoices" />

      {/* Summary */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-4">
        <div className="rounded-xl border border-slate-200 bg-white p-4 flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-indigo-50 flex items-center justify-center"><FileText className="w-5 h-5 text-indigo-500" /></div>
          <div><p className="text-2xl font-bold text-slate-800">{filtered.length}</p><p className="text-xs text-slate-400">Total Invoices</p></div>
        </div>
        <div className="rounded-xl border border-slate-200 bg-white p-4 flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-green-50 flex items-center justify-center"><DollarSign className="w-5 h-5 text-green-500" /></div>
          <div><p className="text-2xl font-bold text-slate-800">${totalRevenue.toFixed(2)}</p><p className="text-xs text-slate-400">Paid Revenue</p></div>
        </div>
        <div className="rounded-xl border border-slate-200 bg-white p-4 flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-amber-50 flex items-center justify-center"><FileText className="w-5 h-5 text-amber-500" /></div>
          <div><p className="text-2xl font-bold text-slate-800">{filtered.filter((i) => i.status === "pending").length}</p><p className="text-xs text-slate-400">Pending</p></div>
        </div>
      </div>

      <FiltersBar
        search={search}
        onSearchChange={setSearch}
        searchPlaceholder="Search by invoice number..."
        filters={[{
          key: "status", label: "All Status", value: statusFilter, onChange: setStatusFilter,
          options: [
            { label: "Pending", value: "pending" },
            { label: "Paid", value: "paid" },
            { label: "Refunded", value: "refunded" },
            { label: "Cancelled", value: "cancelled" },
          ],
        }]}
      />

      <DataTable columns={columns} data={filtered} isLoading={isLoading} emptyMessage="No invoices found." rowKey={(r: Invoice) => String(r.id)} />
    </div>
  );
}
