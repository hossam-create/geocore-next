"use client";

import { useQuery } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import DataTable from "@/components/shared/DataTable";
import { mockDisputes } from "@/lib/mockData";
import { disputesApi } from "@/lib/api";

type DisputeRow = {
  id: string;
  buyer: string;
  seller: string;
  reason: string;
  amount: number;
  status: string;
};

function normalizeDisputes(payload: unknown): DisputeRow[] {
  const box = payload as
    | { data?: Array<Record<string, unknown>>; meta?: unknown }
    | Array<Record<string, unknown>>
    | null
    | undefined;
  const rows = Array.isArray(box) ? box : Array.isArray(box?.data) ? box.data : [];

  return rows
    .map((item) => ({
      id: String(item.id ?? ""),
      buyer: String(item.opener_name ?? item.buyer_name ?? item.opener_id ?? "Unknown"),
      seller: String(item.seller_name ?? item.seller_id ?? "Unknown"),
      reason: String(item.reason ?? "No reason"),
      amount: Number(item.amount ?? item.total ?? 0),
      status: String(item.status ?? "open"),
    }))
    .filter((x) => x.id);
}

export default function SupportDisputesPage() {
  const { data: liveDisputes, isLoading } = useQuery({
    queryKey: ["support", "disputes"],
    queryFn: async () => {
      const res = await disputesApi.list();
      return normalizeDisputes(res);
    },
    retry: 1,
  });

  const disputes = liveDisputes?.length
    ? liveDisputes
    : mockDisputes.map((d) => ({
        id: d.id,
        buyer: d.buyer,
        seller: d.seller,
        reason: d.reason,
        amount: d.amount,
        status: d.status,
      }));

  return (
    <div>
      <PageHeader title="Disputes" description="Buyer-seller conflicts and resolution status" />
      <DataTable
        columns={[
          { key: "id", label: "ID", render: (r: DisputeRow) => <span className="font-mono text-xs">{r.id}</span> },
          { key: "buyer", label: "Buyer" },
          { key: "seller", label: "Seller" },
          { key: "reason", label: "Reason" },
          { key: "amount", label: "Amount", render: (r: DisputeRow) => `$${r.amount.toLocaleString()}` },
          { key: "status", label: "Status", render: (r: DisputeRow) => <StatusBadge status={r.status} dot /> },
        ]}
        data={disputes}
        isLoading={isLoading}
        loadingMessage="Loading disputes..."
        emptyMessage="No disputes found."
        rowKey={(r) => r.id}
      />
    </div>
  );
}
