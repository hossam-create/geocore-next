"use client";

import { useParams } from "next/navigation";
import Link from "next/link";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import axios from "axios";
import { ArrowLeft, Shield, Lock, CheckCircle, XCircle, AlertTriangle } from "lucide-react";

interface EscrowContract {
  id: string;
  order_id: string;
  buyer_id: string;
  seller_id: string;
  amount: number;
  currency: string;
  chain: string;
  contract_address: string;
  tx_hash_fund: string;
  tx_hash_release: string;
  status: string;
  funded_at: string | null;
  released_at: string | null;
  expires_at: string | null;
  created_at: string;
}

export default function EscrowDetailPage() {
  const { id } = useParams<{ id: string }>();
  const qc = useQueryClient();

  const { data: ec, isLoading } = useQuery<EscrowContract>({
    queryKey: ["escrow", id],
    queryFn: async () => {
      const { data } = await axios.get(`/api/v1/escrow/${id}`);
      return data.data;
    },
  });

  const fundMut = useMutation({
    mutationFn: () => axios.post(`/api/v1/escrow/${id}/fund`, { tx_hash: "0x" + Math.random().toString(16).slice(2, 18), contract_address: "0x" + Math.random().toString(16).slice(2, 42) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["escrow", id] }),
  });

  const releaseMut = useMutation({
    mutationFn: () => axios.post(`/api/v1/escrow/${id}/release`, { tx_hash: "0x" + Math.random().toString(16).slice(2, 18) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["escrow", id] }),
  });

  const refundMut = useMutation({
    mutationFn: () => axios.post(`/api/v1/escrow/${id}/refund`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["escrow", id] }),
  });

  const disputeMut = useMutation({
    mutationFn: () => axios.post(`/api/v1/escrow/${id}/dispute`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["escrow", id] }),
  });

  if (isLoading) return <div className="text-center py-16 text-gray-400">Loading...</div>;
  if (!ec) return <div className="text-center py-16 text-gray-400">Not found</div>;

  const statusColor: Record<string, string> = {
    pending: "bg-orange-100 text-orange-700",
    funded: "bg-blue-100 text-blue-700",
    released: "bg-green-100 text-green-700",
    refunded: "bg-gray-100 text-gray-600",
    disputed: "bg-red-100 text-red-700",
  };

  return (
    <div className="max-w-2xl mx-auto px-4 py-8">
      <Link href="/escrow" className="flex items-center gap-2 text-gray-500 hover:text-gray-700 mb-6 text-sm">
        <ArrowLeft className="w-4 h-4" /> Back
      </Link>

      <div className="bg-white rounded-xl border p-6 mb-6">
        <div className="flex items-center justify-between mb-5">
          <div className="flex items-center gap-3">
            <Shield className="w-6 h-6 text-indigo-600" />
            <h1 className="text-xl font-bold">Escrow Contract</h1>
          </div>
          <span className={`text-sm px-3 py-1 rounded-full font-medium ${statusColor[ec.status] ?? "bg-gray-100"}`}>{ec.status}</span>
        </div>

        <div className="grid grid-cols-2 gap-4 text-sm mb-5">
          <div><span className="text-gray-500">Amount</span><p className="text-lg font-bold">{ec.currency} {ec.amount.toLocaleString()}</p></div>
          <div><span className="text-gray-500">Chain</span><p className="font-medium">{ec.chain}</p></div>
          <div><span className="text-gray-500">Order</span><p className="font-mono text-xs">{ec.order_id}</p></div>
          <div><span className="text-gray-500">Created</span><p>{new Date(ec.created_at).toLocaleString()}</p></div>
          {ec.contract_address && <div className="col-span-2"><span className="text-gray-500">Contract</span><p className="font-mono text-xs break-all">{ec.contract_address}</p></div>}
          {ec.tx_hash_fund && <div className="col-span-2"><span className="text-gray-500">Fund TX</span><p className="font-mono text-xs break-all">{ec.tx_hash_fund}</p></div>}
          {ec.tx_hash_release && <div className="col-span-2"><span className="text-gray-500">Release TX</span><p className="font-mono text-xs break-all">{ec.tx_hash_release}</p></div>}
          {ec.expires_at && <div><span className="text-gray-500">Expires</span><p>{new Date(ec.expires_at).toLocaleString()}</p></div>}
        </div>

        {/* Actions */}
        <div className="flex flex-wrap gap-3">
          {ec.status === "pending" && (
            <button onClick={() => fundMut.mutate()} disabled={fundMut.isPending} className="flex items-center gap-2 bg-blue-600 text-white px-5 py-2 rounded-lg hover:bg-blue-700 disabled:opacity-50 transition text-sm">
              <Lock className="w-4 h-4" /> Fund Escrow
            </button>
          )}
          {ec.status === "funded" && (
            <>
              <button onClick={() => releaseMut.mutate()} disabled={releaseMut.isPending} className="flex items-center gap-2 bg-green-600 text-white px-5 py-2 rounded-lg hover:bg-green-700 disabled:opacity-50 transition text-sm">
                <CheckCircle className="w-4 h-4" /> Release Funds
              </button>
              <button onClick={() => refundMut.mutate()} disabled={refundMut.isPending} className="flex items-center gap-2 bg-gray-100 text-gray-700 px-5 py-2 rounded-lg hover:bg-gray-200 disabled:opacity-50 transition text-sm">
                <XCircle className="w-4 h-4" /> Refund
              </button>
              <button onClick={() => disputeMut.mutate()} disabled={disputeMut.isPending} className="flex items-center gap-2 bg-red-50 text-red-600 border border-red-200 px-5 py-2 rounded-lg hover:bg-red-100 disabled:opacity-50 transition text-sm">
                <AlertTriangle className="w-4 h-4" /> Dispute
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
