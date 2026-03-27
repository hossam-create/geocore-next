import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useLocation } from "wouter";
import { loadStripe } from "@stripe/stripe-js";
import { Elements, CardElement, useStripe, useElements } from "@stripe/react-stripe-js";
import api from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { formatPrice, formatRelativeTime } from "@/lib/utils";
import { ArrowDownLeft, ArrowUpRight, Wallet, Plus, TrendingUp, CreditCard, X } from "lucide-react";
import type { WalletBalance, WalletTransaction, ApiError } from "@/lib/types";

const stripePromise = import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY
  ? loadStripe(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY)
  : null;

const TYPE_META: Record<string, { label: string; color: string; icon: "in" | "out" }> = {
  wallet_topup: { label: "Top-up", color: "text-green-600", icon: "in" },
  refund: { label: "Refund", color: "text-green-600", icon: "in" },
  purchase: { label: "Purchase", color: "text-red-500", icon: "out" },
  auction_payment: { label: "Auction Payment", color: "text-red-500", icon: "out" },
  failed: { label: "Failed Payment", color: "text-gray-400", icon: "out" },
  cancelled: { label: "Cancelled", color: "text-gray-400", icon: "out" },
};

const ADD_FUNDS_AMOUNTS = [100, 250, 500, 1000, 2500];

const CARD_ELEMENT_OPTIONS = {
  style: {
    base: {
      fontSize: "14px",
      color: "#111827",
      "::placeholder": { color: "#9ca3af" },
    },
    invalid: { color: "#ef4444" },
  },
};

interface TopUpFormProps {
  onClose: () => void;
  onSuccess: () => void;
}

function TopUpForm({ onClose, onSuccess }: TopUpFormProps) {
  const stripe = useStripe();
  const elements = useElements();
  const [addAmount, setAddAmount] = useState("500");
  const [msg, setMsg] = useState("");
  const [loading, setLoading] = useState(false);

  const handlePay = async () => {
    const amount = Number(addAmount);
    if (!amount || amount <= 0) {
      setMsg("Please enter a valid amount.");
      return;
    }

    setLoading(true);
    setMsg("");

    try {
      const { data } = await api.post("/wallet/top-up", { amount, currency: "AED" });
      const clientSecret: string = data?.data?.client_secret;
      const paymentIntentId: string = data?.data?.payment_intent_id;

      if (!clientSecret || !paymentIntentId) {
        setMsg("Top-up recorded (Stripe not configured — balance will not be credited in this environment).");
        setTimeout(onSuccess, 2000);
        return;
      }

      if (!stripe || !elements) {
        setMsg("Stripe is not available. Please try again.");
        setLoading(false);
        return;
      }

      const cardEl = elements.getElement(CardElement);
      if (!cardEl) {
        setMsg("Card element not loaded. Please try again.");
        setLoading(false);
        return;
      }

      const result = await stripe.confirmCardPayment(clientSecret, {
        payment_method: { card: cardEl },
      });

      if (result.error) {
        setMsg(result.error.message ?? "Payment failed.");
        setLoading(false);
        return;
      }

      await api.post("/payments/confirm", { payment_intent_id: paymentIntentId });

      if (result.paymentIntent?.status === "succeeded") {
        setMsg("Top-up successful!");
      } else {
        setMsg("Payment is processing. Your balance will update shortly.");
      }
      onSuccess();
    } catch (err) {
      const apiErr = err as ApiError;
      setMsg(apiErr?.response?.data?.message ?? apiErr?.message ?? "Failed to initiate top-up.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-white rounded-2xl shadow-2xl p-6 w-full max-w-sm">
      <div className="flex items-center justify-between mb-4">
        <h2 className="font-bold text-gray-900 flex items-center gap-2">
          <CreditCard size={18} className="text-[#0071CE]" /> Add Funds
        </h2>
        <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
          <X size={20} />
        </button>
      </div>
      <p className="text-sm text-gray-500 mb-4">Top up your wallet using a debit or credit card.</p>

      <div className="flex flex-wrap gap-2 mb-4">
        {ADD_FUNDS_AMOUNTS.map((a) => (
          <button
            key={a}
            onClick={() => setAddAmount(String(a))}
            className={`px-3 py-1.5 text-sm font-semibold rounded-full border transition-colors ${
              addAmount === String(a)
                ? "bg-[#0071CE] text-white border-[#0071CE]"
                : "border-gray-300 text-gray-700 hover:border-[#0071CE]"
            }`}
          >
            {a} AED
          </button>
        ))}
      </div>

      <input
        type="number"
        value={addAmount}
        onChange={(e) => setAddAmount(e.target.value)}
        placeholder="Enter amount (AED)"
        className="w-full border border-gray-200 rounded-xl px-4 py-3 text-sm outline-none focus:ring-2 focus:ring-[#0071CE] mb-4"
      />

      {stripePromise && (
        <div className="border border-gray-200 rounded-xl px-4 py-3 mb-4 bg-gray-50">
          <CardElement options={CARD_ELEMENT_OPTIONS} />
        </div>
      )}

      {!stripePromise && (
        <p className="text-xs text-amber-600 bg-amber-50 border border-amber-200 rounded-xl px-3 py-2 mb-4">
          Stripe is not configured. Top-up will be recorded without card charge.
        </p>
      )}

      {msg && (
        <p className={`text-sm mb-3 ${msg.includes("success") || msg.includes("received") || msg.includes("processing") ? "text-green-600" : "text-red-500"}`}>
          {msg}
        </p>
      )}

      <button
        onClick={handlePay}
        disabled={loading || !Number(addAmount) || Number(addAmount) <= 0}
        className="w-full bg-[#0071CE] hover:bg-[#005BA1] text-white font-bold py-3 rounded-xl transition-colors disabled:opacity-60 flex items-center justify-center gap-2"
      >
        <CreditCard size={16} />
        {loading ? "Processing…" : `Pay ${addAmount ? `${addAmount} AED` : "amount"}`}
      </button>
    </div>
  );
}

export default function WalletPage() {
  const { isAuthenticated } = useAuthStore();
  const [, navigate] = useLocation();
  const qc = useQueryClient();
  const [showAddFunds, setShowAddFunds] = useState(false);

  if (!isAuthenticated) {
    navigate("/login?next=/wallet");
    return null;
  }

  const { data: balanceData } = useQuery<WalletBalance>({
    queryKey: ["wallet", "balance"],
    queryFn: () => api.get("/wallet/balance").then((r) => r.data.data as WalletBalance),
    retry: false,
  });

  const { data: txData, isLoading: txLoading } = useQuery<WalletTransaction[]>({
    queryKey: ["wallet", "transactions"],
    queryFn: () => api.get("/wallet/transactions").then((r) => r.data.data as WalletTransaction[]),
    retry: false,
  });

  const balance = balanceData?.balance ?? 0;
  const currency = balanceData?.currency ?? "AED";
  const transactions: WalletTransaction[] = txData ?? [];

  const totalIn = transactions.filter((t) => t.amount > 0).reduce((s, t) => s + t.amount, 0);
  const totalOut = transactions.filter((t) => t.amount < 0).reduce((s, t) => s + Math.abs(t.amount), 0);

  const stripeConfigured = Boolean(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY);

  const handleTopUpSuccess = () => {
    qc.invalidateQueries({ queryKey: ["wallet"] });
    setTimeout(() => setShowAddFunds(false), 2500);
  };

  return (
    <div className="max-w-3xl mx-auto px-4 py-10">
      <h1 className="text-2xl font-bold text-gray-900 mb-6 flex items-center gap-2">
        <Wallet size={24} className="text-[#0071CE]" /> My Wallet
      </h1>

      {showAddFunds && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 px-4">
          <Elements stripe={stripePromise ?? Promise.resolve(null)}>
            <TopUpForm
              onClose={() => setShowAddFunds(false)}
              onSuccess={handleTopUpSuccess}
            />
          </Elements>
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
        <div className="md:col-span-1 bg-gradient-to-br from-[#0071CE] to-[#003f75] rounded-2xl p-6 text-white shadow-lg">
          <p className="text-blue-200 text-sm">Available Balance</p>
          <p className="text-4xl font-extrabold mt-2">{formatPrice(balance, currency)}</p>
          <p className="text-blue-200 text-xs mt-1">{currency} · Wallet credits only — purchases charged to card</p>
          {stripeConfigured ? (
            <button
              onClick={() => setShowAddFunds(true)}
              className="mt-5 bg-[#FFC220] text-gray-900 font-bold text-sm px-4 py-2 rounded-xl hover:bg-yellow-400 transition-colors flex items-center gap-1.5 w-full justify-center"
            >
              <Plus size={14} /> Add Funds
            </button>
          ) : (
            <p className="mt-4 text-xs text-blue-200 text-center">
              Payment processing is not configured in this environment.
            </p>
          )}
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

        {txLoading ? (
          <div className="text-center py-16 text-gray-400">
            <p className="text-sm">Loading transactions…</p>
          </div>
        ) : transactions.length === 0 ? (
          <div className="text-center py-16 text-gray-400">
            <p className="text-4xl mb-3">💳</p>
            <p className="font-semibold">No transactions yet</p>
            <p className="text-sm mt-1">Add funds to get started</p>
          </div>
        ) : (
          <ul className="divide-y divide-gray-50">
            {transactions.map((tx: WalletTransaction) => {
              const meta = TYPE_META[tx.kind] ?? { label: tx.kind, color: "text-gray-600", icon: "in" as const };
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
                    <p className="text-xs text-gray-400 truncate">{tx.description}</p>
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
