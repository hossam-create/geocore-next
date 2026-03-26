import { useQuery } from "@tanstack/react-query";
import { useLocation } from "wouter";
import api from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { formatPrice, formatRelativeTime } from "@/lib/utils";
import { ArrowDownLeft, ArrowUpRight, Wallet, Plus, TrendingUp } from "lucide-react";

const MOCK_TRANSACTIONS = [
  { id: "t1", amount: 500, type: "deposit", note: "Top-up via Stripe", balance_after: 500, currency: "AED", created_at: new Date(Date.now() - 86400000 * 2).toISOString() },
  { id: "t2", amount: -25, type: "listing_fee", note: "Listing: iPhone 15 Pro Max", balance_after: 475, currency: "AED", created_at: new Date(Date.now() - 86400000).toISOString() },
  { id: "t3", amount: -15, type: "featured_fee", note: "Featured: 7 days", balance_after: 460, currency: "AED", created_at: new Date(Date.now() - 3600000 * 12).toISOString() },
  { id: "t4", amount: 1250, type: "auction_win", note: "Sold: Rolex Submariner", balance_after: 1710, currency: "AED", created_at: new Date(Date.now() - 3600000 * 6).toISOString() },
  { id: "t5", amount: -85, type: "final_value_fee", note: "Commission (5%) on sale", balance_after: 1625, currency: "AED", created_at: new Date(Date.now() - 3600000 * 5).toISOString() },
  { id: "t6", amount: 200, type: "refund", note: "Refund: Cancelled listing", balance_after: 1825, currency: "AED", created_at: new Date(Date.now() - 1800000).toISOString() },
];

const TYPE_META: Record<string, { label: string; color: string; icon: "in" | "out" }> = {
  deposit: { label: "Top-up", color: "text-green-600", icon: "in" },
  refund: { label: "Refund", color: "text-green-600", icon: "in" },
  auction_win: { label: "Sale Proceeds", color: "text-green-600", icon: "in" },
  admin_credit: { label: "Admin Credit", color: "text-green-600", icon: "in" },
  listing_fee: { label: "Listing Fee", color: "text-red-500", icon: "out" },
  featured_fee: { label: "Featured Fee", color: "text-red-500", icon: "out" },
  final_value_fee: { label: "Commission", color: "text-red-500", icon: "out" },
  withdrawal: { label: "Withdrawal", color: "text-red-500", icon: "out" },
};

export default function WalletPage() {
  const { isAuthenticated } = useAuthStore();
  const [, navigate] = useLocation();

  if (!isAuthenticated) {
    navigate("/login?next=/wallet");
    return null;
  }

  const { data: balanceData } = useQuery({
    queryKey: ["wallet", "balance"],
    queryFn: () => api.get("/wallet/balance").then((r) => r.data.data),
    retry: false,
  });

  const { data: txData } = useQuery({
    queryKey: ["wallet", "transactions"],
    queryFn: () => api.get("/wallet/transactions").then((r) => r.data.data),
    retry: false,
  });

  const balance = balanceData?.balance ?? 1825;
  const currency = balanceData?.currency ?? "AED";
  const transactions: any[] = txData?.length ? txData : MOCK_TRANSACTIONS;

  const totalIn = transactions.filter((t) => t.amount > 0).reduce((s, t) => s + t.amount, 0);
  const totalOut = transactions.filter((t) => t.amount < 0).reduce((s, t) => s + Math.abs(t.amount), 0);

  return (
    <div className="max-w-3xl mx-auto px-4 py-10">
      <h1 className="text-2xl font-bold text-gray-900 mb-6 flex items-center gap-2">
        <Wallet size={24} className="text-[#0071CE]" /> My Wallet
      </h1>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
        <div className="md:col-span-1 bg-gradient-to-br from-[#0071CE] to-[#003f75] rounded-2xl p-6 text-white shadow-lg">
          <p className="text-blue-200 text-sm">Available Balance</p>
          <p className="text-4xl font-extrabold mt-2">{formatPrice(balance, currency)}</p>
          <p className="text-blue-200 text-xs mt-1">{currency}</p>
          <button className="mt-5 bg-[#FFC220] text-gray-900 font-bold text-sm px-4 py-2 rounded-xl hover:bg-yellow-400 transition-colors flex items-center gap-1.5 w-full justify-center">
            <Plus size={14} /> Add Funds
          </button>
        </div>

        <div className="bg-white rounded-2xl p-5 shadow-sm flex flex-col justify-center">
          <div className="flex items-center gap-2 text-green-600 mb-1">
            <ArrowDownLeft size={16} />
            <p className="text-sm font-medium">Total In</p>
          </div>
          <p className="text-2xl font-bold text-gray-900">{formatPrice(totalIn, currency)}</p>
        </div>

        <div className="bg-white rounded-2xl p-5 shadow-sm flex flex-col justify-center">
          <div className="flex items-center gap-2 text-red-500 mb-1">
            <ArrowUpRight size={16} />
            <p className="text-sm font-medium">Total Out</p>
          </div>
          <p className="text-2xl font-bold text-gray-900">{formatPrice(totalOut, currency)}</p>
        </div>
      </div>

      <div className="bg-white rounded-2xl shadow-sm overflow-hidden">
        <div className="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
          <h2 className="font-bold text-gray-800 flex items-center gap-2">
            <TrendingUp size={16} /> Transaction History
          </h2>
          <span className="text-xs text-gray-400">{transactions.length} transactions</span>
        </div>

        {transactions.length === 0 ? (
          <div className="text-center py-16 text-gray-400">
            <p className="text-4xl mb-3">💳</p>
            <p className="font-semibold">No transactions yet</p>
            <p className="text-sm mt-1">Add funds to get started</p>
          </div>
        ) : (
          <ul className="divide-y divide-gray-50">
            {transactions.map((tx) => {
              const meta = TYPE_META[tx.type] ?? { label: tx.type, color: "text-gray-600", icon: "in" as const };
              const isCredit = tx.amount > 0;
              return (
                <li key={tx.id} className="flex items-center gap-4 px-5 py-4 hover:bg-gray-50 transition-colors">
                  <div className={`w-9 h-9 rounded-full flex items-center justify-center shrink-0 ${isCredit ? "bg-green-50" : "bg-red-50"}`}>
                    {isCredit
                      ? <ArrowDownLeft size={16} className="text-green-600" />
                      : <ArrowUpRight size={16} className="text-red-500" />
                    }
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-gray-800">{meta.label}</p>
                    <p className="text-xs text-gray-400 truncate">{tx.note}</p>
                  </div>
                  <div className="text-right shrink-0">
                    <p className={`text-sm font-bold ${meta.color}`}>
                      {isCredit ? "+" : ""}{formatPrice(tx.amount, tx.currency || "AED")}
                    </p>
                    <p className="text-xs text-gray-400">{formatRelativeTime(tx.created_at)}</p>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </div>
  );
}
