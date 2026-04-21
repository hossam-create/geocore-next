"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import axios from "axios";
import { ArrowLeft, ArrowLeftRight, Inbox } from "lucide-react";

interface ExchangeRequest {
  id: string;
  from_currency: string;
  to_currency: string;
  from_amount: number;
  to_amount: number;
  desired_rate: number;
  use_escrow: boolean;
  status: string;
  created_at: string;
}

export default function MyExchangeRequestsPage() {
  const { data: requests = [], isLoading } = useQuery<ExchangeRequest[]>({
    queryKey: ["my-p2p-requests"],
    queryFn: async () => {
      const { data } = await axios.get("/api/v1/p2p/requests?mine=true&status=");
      return data.data ?? [];
    },
  });

  const statusColor: Record<string, string> = {
    open: "bg-green-100 text-green-700",
    matched: "bg-blue-100 text-blue-700",
    completed: "bg-emerald-100 text-emerald-700",
    cancelled: "bg-red-100 text-red-600",
    disputed: "bg-orange-100 text-orange-700",
  };

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <Link href="/p2p" className="flex items-center gap-2 text-gray-500 hover:text-gray-700 mb-6 text-sm">
        <ArrowLeft className="w-4 h-4" /> Back to Marketplace
      </Link>

      <h1 className="text-2xl font-bold mb-6">My Exchange Requests</h1>

      {isLoading && <p className="text-center py-12 text-gray-400">Loading...</p>}

      {!isLoading && requests.length === 0 && (
        <div className="text-center py-16 text-gray-400">
          <Inbox className="w-12 h-12 mx-auto mb-3 opacity-40" />
          <p>You haven&apos;t created any exchange requests yet.</p>
          <Link href="/p2p/new" className="text-blue-600 hover:underline text-sm mt-2 inline-block">Create one now</Link>
        </div>
      )}

      <div className="space-y-3">
        {requests.map((req) => (
          <Link
            key={req.id}
            href={`/p2p/${req.id}`}
            className="block bg-white rounded-xl border p-5 hover:shadow-md transition"
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <span className="bg-blue-100 text-blue-700 text-xs font-bold px-2 py-1 rounded">{req.from_currency}</span>
                <ArrowLeftRight className="w-4 h-4 text-gray-400" />
                <span className="bg-green-100 text-green-700 text-xs font-bold px-2 py-1 rounded">{req.to_currency}</span>
                <span className="text-sm text-gray-600">{req.from_amount.toLocaleString()} &rarr; {req.to_amount.toLocaleString()}</span>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-xs text-gray-400">{new Date(req.created_at).toLocaleDateString()}</span>
                <span className={`text-xs px-2 py-1 rounded-full font-medium ${statusColor[req.status] ?? "bg-gray-100"}`}>{req.status}</span>
              </div>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
