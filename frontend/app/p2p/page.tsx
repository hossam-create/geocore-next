"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import axios from "axios";
import { ArrowLeftRight, Plus, Search, Filter } from "lucide-react";
import { useTranslations } from "next-intl";

interface ExchangeRequest {
  id: string;
  user_id: string;
  from_currency: string;
  to_currency: string;
  from_amount: number;
  to_amount: number;
  desired_rate: number;
  use_escrow: boolean;
  status: string;
  created_at: string;
}

const CURRENCIES = ["USD", "AED", "SAR", "EGP", "EUR", "GBP"];

export default function P2PMarketplacePage() {
  const t = useTranslations("p2p");
  const [fromCurrency, setFromCurrency] = useState("");
  const [toCurrency, setToCurrency] = useState("");

  const { data: requests = [], isLoading } = useQuery<ExchangeRequest[]>({
    queryKey: ["p2p-requests", fromCurrency, toCurrency],
    queryFn: async () => {
      const params = new URLSearchParams({ status: "open" });
      if (fromCurrency) params.set("from_currency", fromCurrency);
      if (toCurrency) params.set("to_currency", toCurrency);
      const { data } = await axios.get(`/api/v1/p2p/requests?${params}`);
      return data.data ?? [];
    },
  });

  return (
    <div className="max-w-5xl mx-auto px-4 py-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold">{t("title")}</h1>
          <p className="text-gray-500 mt-1">{t("subtitle")}</p>
        </div>
        <Link
          href="/p2p/new"
          className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition"
        >
          <Plus className="w-4 h-4" /> New Request
        </Link>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-xl border p-4 mb-6">
        <div className="flex items-center gap-3 flex-wrap">
          <Filter className="w-4 h-4 text-gray-400" />
          <select
            className="border rounded-lg px-3 py-2 text-sm"
            value={fromCurrency}
            onChange={(e) => setFromCurrency(e.target.value)}
          >
            <option value="">From (any)</option>
            {CURRENCIES.map((c) => (
              <option key={c} value={c}>{c}</option>
            ))}
          </select>
          <ArrowLeftRight className="w-4 h-4 text-gray-400" />
          <select
            className="border rounded-lg px-3 py-2 text-sm"
            value={toCurrency}
            onChange={(e) => setToCurrency(e.target.value)}
          >
            <option value="">To (any)</option>
            {CURRENCIES.map((c) => (
              <option key={c} value={c}>{c}</option>
            ))}
          </select>
          <Link href="/p2p/my" className="ml-auto text-sm text-blue-600 hover:underline">
            My Requests
          </Link>
        </div>
      </div>

      {isLoading && <p className="text-center py-12 text-gray-400">Loading...</p>}

      {!isLoading && requests.length === 0 && (
        <div className="text-center py-16 text-gray-400">
          <Search className="w-12 h-12 mx-auto mb-3 opacity-40" />
          <p>No open exchange requests. Be the first!</p>
        </div>
      )}

      <div className="grid gap-4 sm:grid-cols-2">
        {requests.map((req) => (
          <Link
            key={req.id}
            href={`/p2p/${req.id}`}
            className="bg-white rounded-xl border p-5 hover:shadow-md transition"
          >
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <span className="bg-blue-100 text-blue-700 text-xs font-bold px-2 py-1 rounded">{req.from_currency}</span>
                <ArrowLeftRight className="w-4 h-4 text-gray-400" />
                <span className="bg-green-100 text-green-700 text-xs font-bold px-2 py-1 rounded">{req.to_currency}</span>
              </div>
              {req.use_escrow && (
                <span className="text-xs bg-yellow-100 text-yellow-700 px-2 py-1 rounded-full">Escrow</span>
              )}
            </div>
            <div className="grid grid-cols-3 gap-2 text-sm">
              <div>
                <span className="text-gray-500">Selling</span>
                <p className="font-semibold">{req.from_amount.toLocaleString()} {req.from_currency}</p>
              </div>
              <div>
                <span className="text-gray-500">Buying</span>
                <p className="font-semibold">{req.to_amount.toLocaleString()} {req.to_currency}</p>
              </div>
              <div>
                <span className="text-gray-500">Rate</span>
                <p className="font-semibold">{req.desired_rate.toFixed(4)}</p>
              </div>
            </div>
            <p className="text-xs text-gray-400 mt-3">{new Date(req.created_at).toLocaleDateString()}</p>
          </Link>
        ))}
      </div>
    </div>
  );
}
