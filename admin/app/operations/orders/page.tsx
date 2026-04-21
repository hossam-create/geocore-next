"use client";

import { useQuery } from "@tanstack/react-query";
import PageHeader from "@/components/shared/PageHeader";
import StatusBadge from "@/components/shared/StatusBadge";
import DataTable from "@/components/shared/DataTable";
import { mockOrders } from "@/lib/mockData";
import { ordersApi } from "@/lib/api";

type OrderRow = {
  id: string;
  buyer: string;
  seller: string;
  amount: number;
  status: string;
  date: string;
};

function normalizeOrders(payload: unknown): OrderRow[] {
  const box = payload as
    | { data?: Array<Record<string, unknown>>; meta?: unknown; pagination?: unknown }
    | Array<Record<string, unknown>>
    | null
    | undefined;

  const rows = Array.isArray(box) ? box : Array.isArray(box?.data) ? box.data : [];
  return rows
    .map((item) => {
      const amount = Number(item.total ?? item.amount ?? item.subtotal ?? 0);
      const buyerName = (item.buyer as { name?: unknown } | undefined)?.name;
      const sellerName = (item.seller as { name?: unknown } | undefined)?.name;
      return {
        id: String(item.id ?? ""),
        buyer: String(item.buyer_name ?? buyerName ?? item.buyer_id ?? "Unknown"),
        seller: String(item.seller_name ?? sellerName ?? item.seller_id ?? "Unknown"),
        amount: Number.isFinite(amount) ? amount : 0,
        status: String(item.status ?? "pending"),
        date: String(item.created_at ?? item.date ?? new Date().toISOString()),
      };
    })
    .filter((x) => x.id);
}

export default function OrdersPage() {
  const { data: liveOrders, isLoading } = useQuery({
    queryKey: ["operations", "orders"],
    queryFn: async () => {
      const res = await ordersApi.list();
      return normalizeOrders(res);
    },
    retry: 1,
  });

  const orders = liveOrders?.length
    ? liveOrders
    : mockOrders.map((o) => ({
        id: o.id,
        buyer: o.buyer,
        seller: o.seller,
        amount: o.amount,
        status: o.status,
        date: o.date,
      }));

  return (
    <div>
      <PageHeader title="Orders" description="Track and manage all marketplace orders" />
      <DataTable
        columns={[
          { key: "id", label: "Order ID", render: (o: OrderRow) => <span className="font-mono text-xs">{o.id}</span> },
          { key: "buyer", label: "Buyer" },
          { key: "seller", label: "Seller" },
          { key: "amount", label: "Amount", render: (o: OrderRow) => `$${o.amount.toLocaleString()}` },
          { key: "status", label: "Status", render: (o: OrderRow) => <StatusBadge status={o.status} dot /> },
          { key: "date", label: "Date", render: (o: OrderRow) => new Date(o.date).toLocaleDateString() },
        ]}
        data={orders}
        isLoading={isLoading}
        loadingMessage="Loading orders..."
        rowKey={(o) => o.id}
        emptyMessage="No orders found."
      />
    </div>
  );
}
