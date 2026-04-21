"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import axios from "axios";
import { Shield, Lock, CheckCircle, AlertTriangle, XCircle } from "lucide-react";

interface EscrowContract {
  id: string;
  order_id: string;
  buyer_id: string;
  seller_id: string;
  amount: number;
  currency: string;
  chain: string;
  contract_address: string;
  status: string;
  funded_at: string | null;
  released_at: string | null;
  created_at: string;
}

const statusConfig: Record<string, { color: string; icon: typeof Shield }> = {
  pending: { color: "bg-orange-100 text-orange-700", icon: AlertTriangle },
  funded: { color: "bg-blue-100 text-blue-700", icon: Lock },
  released: { color: "bg-green-100 text-green-700", icon: CheckCircle },
  refunded: { color: "bg-gray-100 text-gray-600", icon: XCircle },
  disputed: { color: "bg-red-100 text-red-700", icon: AlertTriangle },
};

export default function EscrowListPage() {
  const { data: contracts = [], isLoading } = useQuery<EscrowContract[]>({
    queryKey: ["escrow-contracts"],
    queryFn: async () => {
      const { data } = await axios.get("/api/v1/escrow");
      return data.data ?? [];
    },
  });

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <div className="flex items-center gap-3 mb-6">
        <div className="bg-indigo-100 p-2 rounded-lg"><Shield className="w-6 h-6 text-indigo-600" /></div>
        <div>
          <h1 className="text-2xl font-bold">Escrow Contracts</h1>
          <p className="text-gray-500 text-sm">Secure blockchain-backed escrow for your transactions</p>
        </div>
      </div>

      {isLoading && <p className="text-center py-12 text-gray-400">Loading...</p>}

      {!isLoading && contracts.length === 0 && (
        <div className="text-center py-16 text-gray-400">
          <Lock className="w-12 h-12 mx-auto mb-3 opacity-40" />
          <p>No escrow contracts yet.</p>
          <p className="text-sm mt-1">Escrow is created automatically during checkout for eligible orders.</p>
        </div>
      )}

      <div className="space-y-3">
        {contracts.map((ec) => {
          const cfg = statusConfig[ec.status] || statusConfig.pending;
          const Icon = cfg.icon;
          return (
            <Link key={ec.id} href={`/escrow/${ec.id}`} className="block bg-white rounded-xl border p-5 hover:shadow-md transition">
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-3">
                  <Icon className="w-5 h-5 text-indigo-500" />
                  <span className="font-medium text-sm">Order: {ec.order_id.slice(0, 8)}...</span>
                  <span className="text-xs text-gray-400">{ec.chain}</span>
                </div>
                <span className={`text-xs px-2 py-1 rounded-full font-medium ${cfg.color}`}>{ec.status}</span>
              </div>
              <div className="flex items-center gap-6 text-sm text-gray-500">
                <span className="font-semibold text-gray-800">{ec.currency} {ec.amount.toLocaleString()}</span>
                {ec.contract_address && <span className="text-xs font-mono">{ec.contract_address.slice(0, 10)}...</span>}
                <span>{new Date(ec.created_at).toLocaleDateString()}</span>
              </div>
            </Link>
          );
        })}
      </div>
    </div>
  );
}
